package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"squad/commons/llm"
	"squad/commons/queue"
	"squad/features/bots"
	"squad/features/threads"
	"strings"
)

func runTitle(ctx context.Context, payload string) error {
	var runPayload threads.RunPayload
	if err := json.Unmarshal([]byte(payload), &runPayload); err != nil {
		return queue.Permanent(fmt.Errorf("error decoding title payload: %w", err))
	}

	thread, err := threads.GetThread(runPayload.ThreadID)
	if err != nil {
		return wrapLookupError(err)
	}

	bot, err := bots.GetBot(thread.BotID)
	if err != nil {
		return wrapLookupError(err)
	}

	gateway, ok := llm.GetGateway(bot.Gateway)
	if !ok {
		return queue.Permanent(fmt.Errorf("gateway %q is not configured", bot.Gateway))
	}

	messages, err := threads.GetAllMessages(thread.ThreadID)
	if err != nil {
		return err
	}

	conversation := buildTitleConversation(messages)
	if conversation == "" {
		return nil
	}

	request := llm.Request{
		Model:    bot.Model,
		Provider: bot.Provider,
		User:     GetGatewayUser(bot.BotID),
		Messages: []llm.Message{
			{Role: "system", Content: "Write a title of 6 words or fewer for this conversation. Reply with the title only, without quotes or a trailing full stop."},
			{Role: "user", Content: conversation},
		},
	}

	response, err := gateway.Complete(ctx, request)
	if err != nil {
		if !llm.IsRetryable(err) {
			return queue.Permanent(err)
		}
		return err
	}

	title := cleanTitle(response.Content)
	if title == "" {
		return nil
	}

	updated, err := threads.UpdateThreadTitle(thread.ThreadID, title)
	if err != nil {
		return err
	}
	threads.PublishThread(updated)

	return nil
}

func buildTitleConversation(messages []threads.Message) string {
	lines := []string{}
	for _, message := range messages {
		if message.Content == "" || (message.Role != threads.ROLE_USER && message.Role != threads.ROLE_ASSISTANT) {
			continue
		}

		content := []rune(message.Content)
		if len(content) > 2000 {
			content = content[:2000]
		}
		lines = append(lines, message.Role+": "+string(content))
	}

	return strings.Join(lines, "\n\n")
}

func cleanTitle(content string) string {
	title := strings.TrimSpace(strings.Split(strings.TrimSpace(content), "\n")[0])
	title = strings.Trim(title, "\"'`*# ")
	title = strings.TrimSuffix(title, ".")

	runes := []rune(title)
	if len(runes) > 80 {
		title = string(runes[:80])
	}

	return title
}
