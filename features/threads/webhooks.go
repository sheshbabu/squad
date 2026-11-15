package threads

import (
	"io"
	"net/http"
	"squad/commons/queue"
	"squad/commons/utils"
	"squad/features/bots"
	"strings"
)

const maxWebhookBodySize = 64 * 1024

func HandleReceiveWebhook(w http.ResponseWriter, r *http.Request) {
	bot, err := bots.GetBotByWebhookToken(r.PathValue("token"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "WEBHOOK_NOT_FOUND", "Webhook not found", err)
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBodySize))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusRequestEntityTooLarge, "MESSAGE_TOO_LARGE", "Message is too large", nil)
		return
	}

	content := strings.TrimSpace(string(body))
	if content == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "MESSAGE_REQUIRED", "Send the message as the request body", nil)
		return
	}

	thread, err := CreateThreadWithRun(bot.BotID, GetTitleFromContent(content), content, nil)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "THREAD_CREATE_FAILED", "Failed to start chat", err)
		return
	}
	queue.Wake()

	utils.SendJSONResponse(w, http.StatusAccepted, thread)
}
