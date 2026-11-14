package threads

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"squad/commons/llm"
	"squad/commons/queue"
	"squad/commons/stream"
	"squad/commons/utils"
	"squad/features/bots"
	"strconv"
	"strings"
)

type SendMessageRequest struct {
	Content       string `json:"content"`
	AttachmentIDs []int  `json:"attachmentIds"`
}

type RenameThreadRequest struct {
	Title string `json:"title"`
}

type ThreadResponse struct {
	Thread         Thread         `json:"thread"`
	Messages       []Message      `json:"messages"`
	HasMore        bool           `json:"hasMore"`
	ReplyEndpoints []llm.Endpoint `json:"replyEndpoints"`
}

type MessagesResponse struct {
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"hasMore"`
}

func HandleGetThreads(w http.ResponseWriter, r *http.Request) {
	botID, err := strconv.Atoi(r.PathValue("botId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_BOT_ID", "Invalid bot", nil)
		return
	}

	threads, err := GetThreads(botID)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "THREADS_READ_FAILED", "Failed to load chats", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, threads)
}

func HandleCreateThread(w http.ResponseWriter, r *http.Request) {
	botID, err := strconv.Atoi(r.PathValue("botId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_BOT_ID", "Invalid bot", nil)
		return
	}

	payload, ok := decodeSendMessageRequest(w, r)
	if !ok {
		return
	}

	if _, err := bots.GetBot(botID); err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "BOT_READ_FAILED", "Failed to load bot", err)
		return
	}

	title := GetTitleFromContent(payload.Content)
	if title == "" {
		title = "New chat"
	}

	thread, err := CreateThreadWithRun(botID, title, payload.Content, payload.AttachmentIDs)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "THREAD_CREATE_FAILED", "Failed to start chat", err)
		return
	}
	queue.Wake()

	utils.SendJSONResponse(w, http.StatusOK, thread)
}

func HandleGetThread(w http.ResponseWriter, r *http.Request) {
	threadID, ok := parseThreadID(w, r)
	if !ok {
		return
	}

	thread, err := GetThread(threadID)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "THREAD_READ_FAILED", "Failed to load chat", err)
		return
	}

	messages, hasMore, err := GetTurns(threadID, 0)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "THREAD_READ_FAILED", "Failed to load chat", err)
		return
	}

	replyEndpoints, err := GetReplyEndpoints(threadID)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "THREAD_READ_FAILED", "Failed to load chat", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, ThreadResponse{Thread: thread, Messages: messages, HasMore: hasMore, ReplyEndpoints: replyEndpoints})
}

func HandleGetMessages(w http.ResponseWriter, r *http.Request) {
	threadID, ok := parseThreadID(w, r)
	if !ok {
		return
	}

	beforeMessageID, err := strconv.Atoi(r.URL.Query().Get("beforeMessageId"))
	if err != nil || beforeMessageID <= 0 {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_CURSOR", "Invalid beforeMessageId", nil)
		return
	}

	messages, hasMore, err := GetTurns(threadID, beforeMessageID)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "MESSAGES_READ_FAILED", "Failed to load messages", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, MessagesResponse{Messages: messages, HasMore: hasMore})
}

func HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	threadID, ok := parseThreadID(w, r)
	if !ok {
		return
	}

	payload, ok := decodeSendMessageRequest(w, r)
	if !ok {
		return
	}

	message, err := AddMessageWithRun(threadID, payload.Content, payload.AttachmentIDs)
	if errors.Is(err, ErrThreadBusy) {
		utils.SendErrorResponse(w, http.StatusConflict, "THREAD_BUSY", "The bot is still replying", nil)
		return
	}
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "MESSAGE_SEND_FAILED", "Failed to send message", err)
		return
	}
	queue.Wake()

	stream.Publish(threadID, stream.Event{Type: "message", Data: message})
	PublishStatus(threadID)

	utils.SendJSONResponse(w, http.StatusOK, message)
}

func HandleRetryThread(w http.ResponseWriter, r *http.Request) {
	threadID, ok := parseThreadID(w, r)
	if !ok {
		return
	}

	err := RetryRun(threadID)
	if errors.Is(err, ErrThreadNotFailed) {
		utils.SendErrorResponse(w, http.StatusConflict, "THREAD_NOT_FAILED", "Only a failed reply can be retried", nil)
		return
	}
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "THREAD_RETRY_FAILED", "Failed to retry", err)
		return
	}
	queue.Wake()

	PublishStatus(threadID)
	w.WriteHeader(http.StatusOK)
}

func HandleCancelThread(w http.ResponseWriter, r *http.Request) {
	threadID, ok := parseThreadID(w, r)
	if !ok {
		return
	}

	wasRunning, err := queue.Cancel(GetRunKey(threadID))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "THREAD_CANCEL_FAILED", "Failed to stop", err)
		return
	}

	// A running job tidies up after itself, a queued one was just deleted so its thread is tidied here.
	if !wasRunning {
		if err := FinishCancelledRun(threadID, ""); err != nil {
			utils.SendErrorResponse(w, http.StatusInternalServerError, "THREAD_CANCEL_FAILED", "Failed to stop", err)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// Partial text is kept rather than discarded, so stopping a reply doesn't lose what the bot already wrote.
func FinishCancelledRun(threadID int, partialContent string) error {
	thread, err := GetThread(threadID)
	if err != nil {
		return err
	}

	filled, err := FillPendingToolCalls(threadID, "Cancelled by user")
	if err != nil {
		return err
	}
	for _, message := range filled {
		stream.Publish(threadID, stream.Event{Type: "message", Data: message})
	}

	if partialContent != "" || thread.Status == STATUS_WORKING {
		message, err := InsertMessage(Message{ThreadID: threadID, Role: ROLE_ASSISTANT, Content: partialContent, IsCancelled: true})
		if err != nil {
			return err
		}
		stream.Publish(threadID, stream.Event{Type: "message", Data: message})
	}

	stream.ClearPartial(threadID)

	if thread.Status == STATUS_WORKING {
		if err := SetThreadStatus(threadID, STATUS_IDLE, ""); err != nil {
			return err
		}
	}
	PublishStatus(threadID)

	return nil
}

func HandleRenameThread(w http.ResponseWriter, r *http.Request) {
	threadID, ok := parseThreadID(w, r)
	if !ok {
		return
	}

	var payload RenameThreadRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request data", nil)
		return
	}

	title := strings.TrimSpace(payload.Title)
	if title == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "EMPTY_TITLE", "Title is empty", nil)
		return
	}

	thread, err := UpdateThreadTitle(threadID, title)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "THREAD_RENAME_FAILED", "Failed to rename chat", err)
		return
	}

	PublishThread(thread)
	utils.SendJSONResponse(w, http.StatusOK, thread)
}

func HandleGetThreadUsage(w http.ResponseWriter, r *http.Request) {
	threadID, ok := parseThreadID(w, r)
	if !ok {
		return
	}

	usage, err := GetThreadUsage(threadID)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "THREAD_USAGE_READ_FAILED", "Failed to load chat details", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, usage)
}

func HandleDeleteThread(w http.ResponseWriter, r *http.Request) {
	threadID, ok := parseThreadID(w, r)
	if !ok {
		return
	}

	if _, err := queue.Cancel(GetRunKey(threadID)); err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "THREAD_DELETE_FAILED", "Failed to delete chat", err)
		return
	}

	attachments, err := GetThreadAttachments(threadID)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "THREAD_DELETE_FAILED", "Failed to delete chat", err)
		return
	}

	if err := DeleteThread(threadID); err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "THREAD_DELETE_FAILED", "Failed to delete chat", err)
		return
	}

	// The rows are gone, so a leftover file only wastes disk and is not worth failing the request over.
	for _, attachment := range attachments {
		if err := os.Remove(GetAttachmentPath(attachment)); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Error("error removing attachment file", "attachmentID", attachment.AttachmentID, "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func decodeSendMessageRequest(w http.ResponseWriter, r *http.Request) (SendMessageRequest, bool) {
	var payload SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request data", nil)
		return payload, false
	}

	payload.Content = strings.TrimSpace(payload.Content)
	if payload.Content == "" && len(payload.AttachmentIDs) == 0 {
		utils.SendErrorResponse(w, http.StatusBadRequest, "EMPTY_MESSAGE", "Message is empty", nil)
		return payload, false
	}

	return payload, true
}

func parseThreadID(w http.ResponseWriter, r *http.Request) (int, bool) {
	threadID, err := strconv.Atoi(r.PathValue("threadId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_THREAD_ID", "Invalid chat", nil)
		return 0, false
	}

	return threadID, true
}
