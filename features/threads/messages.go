package threads

import (
	"squad/commons/llm"
	"strings"
)

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

func GetTitleFromContent(content string) string {
	title := strings.Join(strings.Fields(content), " ")

	runes := []rune(title)
	if len(runes) > 60 {
		title = string(runes[:60]) + "…"
	}

	return title
}
