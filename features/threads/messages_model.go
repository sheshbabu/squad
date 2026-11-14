package threads

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"squad/commons/llm"
	"squad/commons/sqlite"
	"strings"
	"time"
)

const (
	ROLE_USER      = "user"
	ROLE_ASSISTANT = "assistant"
	ROLE_TOOL      = "tool"

	TURNS_PER_PAGE = 10
)

type Message struct {
	MessageID        int              `json:"messageId"`
	ThreadID         int              `json:"threadId"`
	Role             string           `json:"role"`
	Content          string           `json:"content"`
	ToolCalls        []llm.ToolCall   `json:"toolCalls"`
	ReasoningDetails []map[string]any `json:"-"`
	ToolCallID       string           `json:"toolCallId"`
	ToolName         string           `json:"toolName"`
	Model            string           `json:"model"`
	Provider         string           `json:"provider"`
	PromptTokens     int              `json:"promptTokens"`
	CompletionTokens int              `json:"completionTokens"`
	Cost             float64          `json:"cost"`
	IsCancelled      bool             `json:"isCancelled"`
	Attachments      []Attachment     `json:"attachments"`
	CreatedAt        time.Time        `json:"createdAt"`
}

const messageColumns = `
	m.message_id, m.thread_id, m.role, m.content, m.tool_calls, m.reasoning_details, m.tool_call_id, m.tool_name,
	m.model, m.provider, m.prompt_tokens, m.completion_tokens, m.cost, m.is_cancelled, m.created_at
`

func scanMessage(row interface{ Scan(...any) error }) (Message, error) {
	var message Message
	var toolCalls, reasoningDetails string

	err := row.Scan(&message.MessageID, &message.ThreadID, &message.Role, &message.Content, &toolCalls, &reasoningDetails, &message.ToolCallID, &message.ToolName,
		&message.Model, &message.Provider, &message.PromptTokens, &message.CompletionTokens, &message.Cost, &message.IsCancelled, &message.CreatedAt)
	if err != nil {
		return message, err
	}

	message.ToolCalls = []llm.ToolCall{}
	if toolCalls != "" {
		if err := json.Unmarshal([]byte(toolCalls), &message.ToolCalls); err != nil {
			return message, fmt.Errorf("error decoding tool calls: %w", err)
		}
	}

	if reasoningDetails != "" {
		if err := json.Unmarshal([]byte(reasoningDetails), &message.ReasoningDetails); err != nil {
			return message, fmt.Errorf("error decoding reasoning details: %w", err)
		}
	}

	message.Attachments = []Attachment{}
	return message, nil
}

func InsertMessage(message Message) (Message, error) {
	tx, err := sqlite.DB.Begin()
	if err != nil {
		return message, fmt.Errorf("error beginning message tx: %w", err)
	}
	defer tx.Rollback()

	inserted, err := InsertMessageTx(tx, message)
	if err != nil {
		return inserted, err
	}

	if err := tx.Commit(); err != nil {
		return inserted, fmt.Errorf("error committing message: %w", err)
	}

	return inserted, nil
}

func InsertMessageTx(tx *sql.Tx, message Message) (Message, error) {
	toolCalls := ""
	if len(message.ToolCalls) > 0 {
		toolCallsBytes, err := json.Marshal(message.ToolCalls)
		if err != nil {
			return message, fmt.Errorf("error encoding tool calls: %w", err)
		}
		toolCalls = string(toolCallsBytes)
	}

	reasoningDetails := ""
	if len(message.ReasoningDetails) > 0 {
		reasoningBytes, err := json.Marshal(message.ReasoningDetails)
		if err != nil {
			return message, fmt.Errorf("error encoding reasoning details: %w", err)
		}
		reasoningDetails = string(reasoningBytes)
	}

	query := `
		INSERT INTO messages (thread_id, role, content, tool_calls, reasoning_details, tool_call_id, tool_name, model, provider, prompt_tokens, completion_tokens, cost, is_cancelled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING message_id
	`

	var messageID int
	err := tx.QueryRow(query, message.ThreadID, message.Role, message.Content, toolCalls, reasoningDetails, message.ToolCallID, message.ToolName,
		message.Model, message.Provider, message.PromptTokens, message.CompletionTokens, message.Cost, message.IsCancelled).Scan(&messageID)
	if err != nil {
		return message, fmt.Errorf("error inserting message: %w", err)
	}

	_, err = tx.Exec("UPDATE threads SET updated_at = CURRENT_TIMESTAMP WHERE thread_id = ?", message.ThreadID)
	if err != nil {
		return message, fmt.Errorf("error touching thread: %w", err)
	}

	row := tx.QueryRow("SELECT "+messageColumns+" FROM messages m WHERE m.message_id = ?", messageID)
	inserted, err := scanMessage(row)
	if err != nil {
		return inserted, fmt.Errorf("error reading inserted message: %w", err)
	}

	return inserted, nil
}

func insertUserMessageTx(tx *sql.Tx, botID, threadID int, content string, attachmentIDs []int) (Message, error) {
	message, err := InsertMessageTx(tx, Message{ThreadID: threadID, Role: ROLE_USER, Content: content})
	if err != nil {
		return message, err
	}

	if err := LinkAttachmentsTx(tx, botID, message.MessageID, attachmentIDs); err != nil {
		return message, err
	}

	message.Attachments, err = getMessageAttachmentsTx(tx, message.MessageID)
	if err != nil {
		return message, err
	}

	return message, nil
}

// The chat view only holds the latest pages, so the privacy badge gets every reply's endpoint from here.
func GetReplyEndpoints(threadID int) ([]llm.Endpoint, error) {
	endpoints := []llm.Endpoint{}

	rows, err := sqlite.DB.Query("SELECT DISTINCT model, provider FROM messages WHERE thread_id = ? AND role = ? AND model != ''", threadID, ROLE_ASSISTANT)
	if err != nil {
		return nil, fmt.Errorf("error reading thread %d reply endpoints: %w", threadID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var endpoint llm.Endpoint
		if err := rows.Scan(&endpoint.ModelID, &endpoint.ProviderID); err != nil {
			return nil, fmt.Errorf("error scanning thread %d reply endpoint: %w", threadID, err)
		}
		endpoints = append(endpoints, endpoint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating thread %d reply endpoints: %w", threadID, err)
	}

	return endpoints, nil
}

func GetAllMessages(threadID int) ([]Message, error) {
	return queryMessages("m.thread_id = ?", threadID)
}

// Pages break at user messages, so a page never splits a reply from the question it answers.
func GetTurns(threadID, beforeMessageID int) ([]Message, bool, error) {
	query := `
		SELECT message_id FROM messages
		WHERE thread_id = ? AND role = ? AND (? = 0 OR message_id < ?)
		ORDER BY message_id DESC
		LIMIT ?
	`

	rows, err := sqlite.DB.Query(query, threadID, ROLE_USER, beforeMessageID, beforeMessageID, TURNS_PER_PAGE+1)
	if err != nil {
		return nil, false, fmt.Errorf("error retrieving turns: %w", err)
	}
	defer rows.Close()

	turnStartIDs := []int{}
	for rows.Next() {
		var messageID int
		if err := rows.Scan(&messageID); err != nil {
			return nil, false, fmt.Errorf("error scanning turn: %w", err)
		}
		turnStartIDs = append(turnStartIDs, messageID)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("error iterating turns: %w", err)
	}

	if len(turnStartIDs) == 0 {
		return []Message{}, false, nil
	}

	hasMore := len(turnStartIDs) > TURNS_PER_PAGE
	if hasMore {
		turnStartIDs = turnStartIDs[:TURNS_PER_PAGE]
	}
	firstMessageID := turnStartIDs[len(turnStartIDs)-1]

	messages, err := queryMessages("m.thread_id = ? AND m.message_id >= ? AND (? = 0 OR m.message_id < ?)", threadID, firstMessageID, beforeMessageID, beforeMessageID)
	if err != nil {
		return nil, false, err
	}

	return messages, hasMore, nil
}

func queryMessages(where string, args ...any) ([]Message, error) {
	messages := []Message{}

	query := "SELECT " + messageColumns + " FROM messages m WHERE " + where + " ORDER BY m.message_id"

	rows, err := sqlite.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error retrieving messages: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		message, err := scanMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning message: %w", err)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	if err := addAttachments(messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func GetPendingToolCalls(messages []Message) []llm.ToolCall {
	lastAssistantIndex := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == ROLE_ASSISTANT {
			lastAssistantIndex = i
			break
		}
	}

	if lastAssistantIndex == -1 {
		return nil
	}

	answeredIDs := map[string]bool{}
	for _, message := range messages[lastAssistantIndex+1:] {
		if message.Role == ROLE_TOOL {
			answeredIDs[message.ToolCallID] = true
		}
	}

	pending := []llm.ToolCall{}
	for _, toolCall := range messages[lastAssistantIndex].ToolCalls {
		if !answeredIDs[toolCall.ID] {
			pending = append(pending, toolCall)
		}
	}

	return pending
}

// FillPendingToolCalls answers unfinished tool calls with a fixed result, because the next model request is rejected if any tool call lacks one.
func FillPendingToolCalls(threadID int, result string) ([]Message, error) {
	messages, err := GetAllMessages(threadID)
	if err != nil {
		return nil, err
	}

	inserted := []Message{}
	for _, toolCall := range GetPendingToolCalls(messages) {
		message, err := InsertMessage(Message{ThreadID: threadID, Role: ROLE_TOOL, Content: result, ToolCallID: toolCall.ID, ToolName: toolCall.Function.Name})
		if err != nil {
			return inserted, err
		}
		inserted = append(inserted, message)
	}

	return inserted, nil
}

func CountUserMessages(threadID int) (int, error) {
	var count int

	err := sqlite.DB.QueryRow("SELECT COUNT(*) FROM messages WHERE thread_id = ? AND role = ?", threadID, ROLE_USER).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error counting user messages: %w", err)
	}

	return count, nil
}

func GetTitleFromContent(content string) string {
	title := strings.Join(strings.Fields(content), " ")

	runes := []rune(title)
	if len(runes) > 60 {
		title = string(runes[:60]) + "…"
	}

	return title
}

type ThreadUsage struct {
	MessageCount     int     `json:"messageCount"`
	PromptTokens     int     `json:"promptTokens"`
	CompletionTokens int     `json:"completionTokens"`
	Cost             float64 `json:"cost"`
}

// GetThreadUsage counts only user and assistant messages, since tool results are part of a reply rather than messages anyone reads.
func GetThreadUsage(threadID int) (ThreadUsage, error) {
	var usage ThreadUsage

	query := `
		SELECT
			COUNT(*) FILTER (WHERE role IN ('user', 'assistant') AND content != ''),
			COALESCE(SUM(prompt_tokens), 0),
			COALESCE(SUM(completion_tokens), 0),
			COALESCE(SUM(cost), 0)
		FROM messages
		WHERE thread_id = ?
	`

	err := sqlite.DB.QueryRow(query, threadID).Scan(&usage.MessageCount, &usage.PromptTokens, &usage.CompletionTokens, &usage.Cost)
	if err != nil {
		return usage, fmt.Errorf("error reading usage for thread %d: %w", threadID, err)
	}

	return usage, nil
}
