package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"squad/commons/llm"
	"squad/commons/queue"
	"squad/commons/stream"
	"squad/commons/utils"
	"squad/features/bots"
	"squad/features/threads"
	"squad/features/tools"
	"strings"
	"time"
)

const maxModelCallsPerRun = 25

func Init() {
	queue.RegisterHandler(threads.JOB_AGENT_RUN, queue.Handler{Run: runAgent, OnRetry: handleAgentRetry, OnFailed: handleAgentFailed, MaxAttempts: 5})
	queue.RegisterHandler(threads.JOB_THREAD_TITLE, queue.Handler{Run: runTitle, MaxAttempts: 2})
}

func runAgent(ctx context.Context, payload string) error {
	var runPayload threads.RunPayload
	if err := json.Unmarshal([]byte(payload), &runPayload); err != nil {
		return queue.Permanent(fmt.Errorf("error decoding run payload: %w", err))
	}
	threadID := runPayload.ThreadID

	err := runThread(ctx, threadID)

	// The only thing that cancels a run's context is the Stop button.
	if ctx.Err() != nil {
		if err := threads.FinishCancelledRun(threadID, stream.GetPartial(threadID)); err != nil {
			slog.Error("error finishing cancelled run", "threadID", threadID, "error", err)
		}
		return nil
	}

	if err != nil {
		stream.ResetPartial(threadID)
	}

	return err
}

func handleAgentRetry(payload string, err error, delay time.Duration) {
	var runPayload threads.RunPayload
	if json.Unmarshal([]byte(payload), &runPayload) != nil {
		return
	}
	threadID := runPayload.ThreadID

	message := fmt.Sprintf("%s Retrying in %s.", getFailureMessage(err), delay)
	if err := threads.SetThreadStatus(threadID, threads.STATUS_WORKING, message); err != nil {
		slog.Error("error recording thread retry", "threadID", threadID, "error", err)
		return
	}
	threads.PublishStatus(threadID)
}

func handleAgentFailed(payload string, err error) {
	var runPayload threads.RunPayload
	if json.Unmarshal([]byte(payload), &runPayload) != nil {
		return
	}
	threadID := runPayload.ThreadID

	stream.ResetPartial(threadID)

	message := getFailureMessage(err)
	var apiErr llm.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == 429 {
		message += " Try again later, or pick another model or provider."
	}

	if err := threads.SetThreadStatus(threadID, threads.STATUS_FAILED, message); err != nil {
		slog.Error("error marking thread failed", "threadID", threadID, "error", err)
		return
	}
	threads.PublishStatus(threadID)
}

// runThread decides each step from what is saved, so a retry after a crash or an error picks up where the last run stopped.
func runThread(ctx context.Context, threadID int) error {
	thread, err := threads.GetThread(threadID)
	if err != nil {
		return wrapLookupError(err)
	}

	// A retry is starting, so the waiting message from the last attempt no longer applies.
	if thread.ErrorMessage != "" {
		if err := threads.SetThreadStatus(threadID, threads.STATUS_WORKING, ""); err != nil {
			return err
		}
		threads.PublishStatus(threadID)
	}

	bot, err := bots.GetBot(thread.BotID)
	if err != nil {
		return wrapLookupError(err)
	}

	gateway, ok := llm.GetGateway(bot.Gateway)
	if !ok {
		return queue.Permanent(fmt.Errorf("gateway %q is not configured", bot.Gateway))
	}

	modelCalls := 0
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		messages, err := threads.GetAllMessages(threadID)
		if err != nil {
			return err
		}
		if len(messages) == 0 {
			return queue.Permanent(fmt.Errorf("thread %d has no messages", threadID))
		}

		pendingToolCalls := threads.GetPendingToolCalls(messages)
		if len(pendingToolCalls) > 0 {
			if err := runToolCalls(ctx, bot, threadID, pendingToolCalls); err != nil {
				return err
			}
			continue
		}

		if messages[len(messages)-1].Role == threads.ROLE_ASSISTANT {
			return finishRun(thread)
		}

		if modelCalls >= maxModelCallsPerRun {
			return queue.Permanent(fmt.Errorf("stopped after %d model calls in one reply", maxModelCallsPerRun))
		}
		modelCalls++

		if err := callModel(ctx, gateway, bot, thread, messages); err != nil {
			return err
		}
	}
}

func runToolCalls(ctx context.Context, bot bots.Bot, threadID int, toolCalls []llm.ToolCall) error {
	for _, toolCall := range toolCalls {
		result := tools.Run(ctx, bot, toolCall)

		// A tool interrupted by Stop has no real result, and the cancel path records one instead.
		if ctx.Err() != nil {
			return ctx.Err()
		}

		message, err := threads.InsertMessage(threads.Message{ThreadID: threadID, Role: threads.ROLE_TOOL, Content: result, ToolCallID: toolCall.ID, ToolName: toolCall.Function.Name})
		if err != nil {
			return err
		}
		stream.Publish(threadID, stream.Event{Type: "message", Data: message})
	}

	return nil
}

func callModel(ctx context.Context, gateway llm.Gateway, bot bots.Bot, thread threads.Thread, messages []threads.Message) error {
	request, err := buildRequest(bot, thread, messages)
	if err != nil {
		return err
	}

	response, err := gateway.Stream(ctx, request, func(delta string) {
		stream.AppendPartial(thread.ThreadID, delta)
	})
	if err != nil {
		if ctx.Err() == nil && !llm.IsRetryable(err) {
			return queue.Permanent(err)
		}
		return err
	}

	message, err := threads.InsertMessage(threads.Message{
		ThreadID:         thread.ThreadID,
		Role:             threads.ROLE_ASSISTANT,
		Content:          response.Content,
		ToolCalls:        response.ToolCalls,
		ReasoningDetails: response.ReasoningDetails,
		Model:            bot.Model,
		Provider:         bot.Provider,
		PromptTokens:     response.PromptTokens,
		CompletionTokens: response.CompletionTokens,
		Cost:             response.Cost,
	})
	if err != nil {
		return err
	}

	stream.ClearPartial(thread.ThreadID)
	stream.Publish(thread.ThreadID, stream.Event{Type: "message", Data: message})

	return nil
}

func finishRun(thread threads.Thread) error {
	if err := threads.SetThreadStatus(thread.ThreadID, threads.STATUS_IDLE, ""); err != nil {
		return err
	}
	threads.PublishStatus(thread.ThreadID)

	// Scheduled threads keep their schedule name as the title.
	if thread.ScheduleID != nil {
		return nil
	}

	userMessageCount, err := threads.CountUserMessages(thread.ThreadID)
	if err != nil {
		slog.Error("error checking whether thread needs a title", "threadID", thread.ThreadID, "error", err)
		return nil
	}

	if userMessageCount == 1 {
		if err := queue.Enqueue(threads.JOB_THREAD_TITLE, "", threads.RunPayload{ThreadID: thread.ThreadID}); err != nil {
			slog.Error("error queueing thread title", "threadID", thread.ThreadID, "error", err)
		}
	}

	return nil
}

func wrapLookupError(err error) error {
	if errors.Is(err, utils.ErrNotFound) {
		return queue.Permanent(err)
	}
	return err
}

func getFailureMessage(err error) string {
	var apiErr llm.APIError
	if errors.As(err, &apiErr) {
		// OpenRouter reports a provider's rate limit as a bare "Provider returned error", which doesn't tell anyone what happened.
		if apiErr.StatusCode == 429 {
			return "The provider is rate limiting this model."
		}
		// OpenRouter answers "No endpoints found that support image input" when the bot's model is text only.
		if strings.Contains(apiErr.Message, "image input") {
			return "This model can't read images. Send the message without the image, or switch to a model that accepts images."
		}
		return apiErr.Message
	}
	return err.Error()
}
