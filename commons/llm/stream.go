package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

const streamIdleTimeout = 5 * time.Minute

var errStreamEnded = errors.New("stream ended before completion")

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content          string           `json:"content"`
			ToolCalls        []toolCallDelta  `json:"tool_calls"`
			ReasoningDetails []map[string]any `json:"reasoning_details"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *usage `json:"usage"`
	Error *struct {
		Code    any    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type toolCallDelta struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

func (g Gateway) Stream(ctx context.Context, request Request, onDelta func(string)) (Response, error) {
	// A stream can stall without closing, so a watchdog cancels it after a long silence.
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	idleTimer := time.AfterFunc(streamIdleTimeout, cancel)
	defer idleTimer.Stop()

	httpResponse, err := g.post(streamCtx, "/chat/completions", g.buildBody(request, true))
	if err != nil {
		return Response{}, wrapStreamError(ctx, streamCtx, err)
	}
	defer httpResponse.Body.Close()

	onLine := func() {
		idleTimer.Reset(streamIdleTimeout)
	}

	response, err := parseStream(httpResponse.Body, onDelta, onLine)
	if err != nil {
		return response, wrapStreamError(ctx, streamCtx, err)
	}

	return response, nil
}

func wrapStreamError(ctx, streamCtx context.Context, err error) error {
	if ctx.Err() == nil && streamCtx.Err() != nil {
		return fmt.Errorf("stream stalled for %s", streamIdleTimeout)
	}
	return err
}

func parseStream(reader io.Reader, onDelta func(string), onLine func()) (Response, error) {
	var response Response
	var content strings.Builder
	toolCallsByIndex := map[int]*ToolCall{}
	reasoningByIndex := map[int]map[string]any{}
	isDone := false

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		onLine()
		line := scanner.Text()

		// Lines starting with ":" are keepalive comments.
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			isDone = true
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return response, fmt.Errorf("error decoding stream chunk: %w", err)
		}

		if chunk.Error != nil {
			return response, APIError{StatusCode: parseErrorCode(chunk.Error.Code), Message: chunk.Error.Message}
		}

		if chunk.Usage != nil {
			response.PromptTokens = chunk.Usage.PromptTokens
			response.CompletionTokens = chunk.Usage.CompletionTokens
			response.Cost = chunk.Usage.Cost
		}

		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				content.WriteString(choice.Delta.Content)
				onDelta(choice.Delta.Content)
			}

			for _, delta := range choice.Delta.ToolCalls {
				mergeToolCallDelta(toolCallsByIndex, delta)
			}

			for _, detail := range choice.Delta.ReasoningDetails {
				mergeReasoningDetail(reasoningByIndex, detail)
			}

			if choice.FinishReason == "error" {
				return response, errStreamEnded
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return response, fmt.Errorf("error reading stream: %w", err)
	}

	if !isDone {
		return response, errStreamEnded
	}

	response.Content = content.String()
	response.ToolCalls = sortToolCalls(toolCallsByIndex)
	response.ReasoningDetails = sortReasoningDetails(reasoningByIndex)

	return response, nil
}

func mergeToolCallDelta(toolCallsByIndex map[int]*ToolCall, delta toolCallDelta) {
	toolCall, ok := toolCallsByIndex[delta.Index]
	if !ok {
		toolCall = &ToolCall{Type: "function"}
		toolCallsByIndex[delta.Index] = toolCall
	}

	if delta.ID != "" {
		toolCall.ID = delta.ID
	}
	if delta.Type != "" {
		toolCall.Type = delta.Type
	}
	if delta.Function.Name != "" {
		toolCall.Function.Name += delta.Function.Name
	}
	toolCall.Function.Arguments += delta.Function.Arguments
}

// Reasoning arrives in pieces keyed by index, and must be sent back whole with the assistant message so reasoning models keep continuity across tool calls.
func mergeReasoningDetail(reasoningByIndex map[int]map[string]any, detail map[string]any) {
	index := 0
	if value, ok := detail["index"].(float64); ok {
		index = int(value)
	}

	existing, ok := reasoningByIndex[index]
	if !ok {
		reasoningByIndex[index] = detail
		return
	}

	for key, value := range detail {
		text, isText := value.(string)
		previousText, wasText := existing[key].(string)
		if isText && wasText && (key == "text" || key == "summary" || key == "data") {
			existing[key] = previousText + text
			continue
		}
		if value != nil {
			existing[key] = value
		}
	}
}

func sortToolCalls(toolCallsByIndex map[int]*ToolCall) []ToolCall {
	indexes := []int{}
	for index := range toolCallsByIndex {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	toolCalls := []ToolCall{}
	for _, index := range indexes {
		toolCalls = append(toolCalls, *toolCallsByIndex[index])
	}

	return toolCalls
}

func sortReasoningDetails(reasoningByIndex map[int]map[string]any) []map[string]any {
	indexes := []int{}
	for index := range reasoningByIndex {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	details := []map[string]any{}
	for _, index := range indexes {
		details = append(details, reasoningByIndex[index])
	}

	return details
}

func parseErrorCode(code any) int {
	if number, ok := code.(float64); ok {
		return int(number)
	}

	// An error without a numeric code arrived mid-stream, which is most likely the upstream provider failing.
	return 502
}
