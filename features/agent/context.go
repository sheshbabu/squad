package agent

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"squad/commons/llm"
	"squad/features/bots"
	"squad/features/files"
	"squad/features/memories"
	"squad/features/threads"
	"squad/features/tools"
	"squad/features/users"
	"strings"
	"time"
)

const clearedToolResult = "[Old tool result cleared. Run the tool again if you need it.]"

func buildRequest(bot bots.Bot, thread threads.Thread, messages []threads.Message) (llm.Request, error) {
	memoryList, err := memories.GetMemories(bot.BotID)
	if err != nil {
		return llm.Request{}, err
	}

	fileList, err := files.ListFiles(bot.BotID)
	if err != nil {
		return llm.Request{}, err
	}

	aboutMe := ""
	if bot.UsesAboutMe {
		aboutMe, err = users.GetAboutMe()
		if err != nil {
			return llm.Request{}, err
		}
	}

	llmMessages := []llm.Message{{Role: "system", Content: buildSystemPrompt(bot, thread, aboutMe, memoryList, fileList)}}

	currentTurnStart := 0
	for i, message := range messages {
		if message.Role == threads.ROLE_USER {
			currentTurnStart = i
		}
	}

	for i, message := range messages {
		switch message.Role {
		case threads.ROLE_USER:
			llmMessages = append(llmMessages, llm.Message{Role: "user", Content: buildUserContent(message)})
		case threads.ROLE_ASSISTANT:
			if message.Content == "" && len(message.ToolCalls) == 0 {
				continue
			}
			llmMessages = append(llmMessages, llm.Message{Role: "assistant", Content: message.Content, ToolCalls: message.ToolCalls, ReasoningDetails: message.ReasoningDetails})
		case threads.ROLE_TOOL:
			content := message.Content
			if i < currentTurnStart {
				content = clearedToolResult
			}
			llmMessages = append(llmMessages, llm.Message{Role: "tool", Content: content, ToolCallID: message.ToolCallID})
		}
	}

	return llm.Request{Model: bot.Model, Provider: bot.Provider, ReasoningEffort: bot.ReasoningEffort, User: GetGatewayUser(bot.BotID), Messages: llmMessages, Tools: tools.GetDefinitions(bot)}, nil
}

func buildSystemPrompt(bot bots.Bot, thread threads.Thread, aboutMe string, memoryList []memories.Memory, fileList []files.File) string {
	sections := []string{}

	if strings.TrimSpace(bot.SystemPrompt) != "" {
		sections = append(sections, strings.TrimSpace(bot.SystemPrompt))
	}

	sections = append(sections, fmt.Sprintf("You are %s. The current date and time is %s.", bot.Name, time.Now().Format("Monday, 2 January 2006, 15:04 MST")))

	if thread.ScheduleID != nil {
		sections = append(sections, "This chat was started automatically by a schedule the user set up, so carry out the first message's task without asking follow-up questions.")
	}

	if aboutMe != "" {
		sections = append(sections, "About the user, written by them and shared with all their bots:\n"+aboutMe)
	}

	memoryLines := []string{"Memories you saved earlier (manage them with save_memory and delete_memory):"}
	if len(memoryList) == 0 {
		memoryLines = append(memoryLines, "None yet.")
	}
	for _, memory := range memoryList {
		memoryLines = append(memoryLines, fmt.Sprintf("- [%d] %s", memory.MemoryID, memory.Content))
	}
	sections = append(sections, strings.Join(memoryLines, "\n"))

	fileLines := []string{"Markdown files in your folder (use the file tools to read or change them):"}
	if len(fileList) == 0 {
		fileLines = append(fileLines, "None yet.")
	}
	for _, file := range fileList {
		fileLines = append(fileLines, "- "+file.Name)
	}
	sections = append(sections, strings.Join(fileLines, "\n"))

	return strings.Join(sections, "\n\n")
}

func buildUserContent(message threads.Message) any {
	if len(message.Attachments) == 0 {
		return message.Content
	}

	parts := []llm.ContentPart{}
	if message.Content != "" {
		parts = append(parts, llm.ContentPart{Type: "text", Text: message.Content})
	}
	for _, attachment := range message.Attachments {
		parts = append(parts, buildAttachmentPart(attachment))
	}

	return parts
}

func buildAttachmentPart(attachment threads.Attachment) llm.ContentPart {
	fileBytes, err := os.ReadFile(threads.GetAttachmentPath(attachment))
	if err != nil {
		slog.Error("error reading attachment", "attachmentID", attachment.AttachmentID, "error", err)
		return llm.ContentPart{Type: "text", Text: fmt.Sprintf("[The attached file %s could not be read]", attachment.Filename)}
	}

	dataURL := "data:" + attachment.MimeType + ";base64," + base64.StdEncoding.EncodeToString(fileBytes)

	if attachment.MimeType == "application/pdf" {
		return llm.ContentPart{Type: "file", File: &llm.File{Filename: attachment.Filename, FileData: dataURL}}
	}

	return llm.ContentPart{Type: "image_url", ImageURL: &llm.ImageURL{URL: dataURL}}
}

// OpenRouter's Activity and Logs pages can filter and group spend by this value, which gives a cost breakdown per bot.
func GetGatewayUser(botID int) string {
	return fmt.Sprintf("bot-%d", botID)
}
