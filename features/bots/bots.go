package bots

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"squad/commons/llm"
	"squad/commons/utils"
	"strconv"
	"strings"
)

type UpdateBotToolsRequest struct {
	DisabledToolGroups []string `json:"disabledToolGroups"`
	ZenAPIToken        string   `json:"zenApiToken"`
}

var colors = []string{"gray", "red", "orange", "yellow", "green", "teal", "blue", "purple", "pink"}

func HandleGetBots(w http.ResponseWriter, r *http.Request) {
	bots, err := GetBots()
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "BOTS_READ_FAILED", "Failed to load bots", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, bots)
}

func HandleCreateBot(w http.ResponseWriter, r *http.Request) {
	bot, ok := decodeBot(w, r)
	if !ok {
		return
	}

	created, err := CreateBot(bot)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "BOT_CREATE_FAILED", "Failed to create bot", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, created)
}

func HandleUpdateBot(w http.ResponseWriter, r *http.Request) {
	botID, err := strconv.Atoi(r.PathValue("botId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_BOT_ID", "Invalid bot", nil)
		return
	}

	bot, ok := decodeBot(w, r)
	if !ok {
		return
	}
	bot.BotID = botID

	updated, err := UpdateBot(bot)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "BOT_UPDATE_FAILED", "Failed to update bot", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, updated)
}

// HandleUpdateBotTools is separate from HandleUpdateBot so saving the profile or model never resets a bot's tools.
func HandleUpdateBotTools(w http.ResponseWriter, r *http.Request) {
	botID, err := strconv.Atoi(r.PathValue("botId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_BOT_ID", "Invalid bot", nil)
		return
	}

	var payload UpdateBotToolsRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request data", nil)
		return
	}

	for _, group := range payload.DisabledToolGroups {
		if group == "" || strings.Contains(group, ",") {
			utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_TOOL_GROUP", "Invalid tool group", nil)
			return
		}
	}

	updated, err := UpdateBotTools(botID, payload.DisabledToolGroups, strings.TrimSpace(payload.ZenAPIToken))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "BOT_UPDATE_FAILED", "Failed to update tools", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, updated)
}

func HandleDeleteBot(w http.ResponseWriter, r *http.Request) {
	botID, err := strconv.Atoi(r.PathValue("botId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_BOT_ID", "Invalid bot", nil)
		return
	}

	if err := DeleteBot(botID); err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "BOT_DELETE_FAILED", "Failed to delete bot", err)
		return
	}

	// The rows are gone, so a leftover folder only wastes disk and is not worth failing the request over.
	if err := os.RemoveAll(GetBotFolderPath(botID)); err != nil {
		slog.Error("error removing bot folder", "botID", botID, "error", err)
	}

	w.WriteHeader(http.StatusOK)
}

// Files and attachments share one folder per bot, so deleting a bot removes everything with one RemoveAll.
func GetBotFolderPath(botID int) string {
	return utils.GetDataFolderPath("bots", strconv.Itoa(botID))
}

func decodeBot(w http.ResponseWriter, r *http.Request) (Bot, bool) {
	var bot Bot
	if err := json.NewDecoder(r.Body).Decode(&bot); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request data", nil)
		return bot, false
	}

	bot.Name = strings.TrimSpace(bot.Name)
	bot.Model = strings.TrimSpace(bot.Model)
	bot.Provider = strings.TrimSpace(bot.Provider)

	if bot.Name == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_NAME", "Name is required", nil)
		return bot, false
	}

	if bot.Model == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_MODEL", "Model is required", nil)
		return bot, false
	}

	gateway, ok := llm.GetGateway(bot.Gateway)
	if !ok {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_GATEWAY", fmt.Sprintf("Gateway %q is not configured", bot.Gateway), nil)
		return bot, false
	}

	if bot.ReasoningEffort != "" {
		models, err := gateway.ListModels(r.Context())
		if err != nil {
			utils.SendErrorResponse(w, http.StatusInternalServerError, "MODELS_READ_FAILED", "Failed to load models", err)
			return bot, false
		}
		isSupported := slices.ContainsFunc(models, func(model llm.Model) bool {
			return model.ModelID == bot.Model && slices.Contains(model.ReasoningEfforts, bot.ReasoningEffort)
		})
		if !isSupported {
			utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_REASONING_EFFORT", "This model doesn't support that thinking level", nil)
			return bot, false
		}
	}

	if bot.Color == "" {
		bot.Color = "gray"
	}
	if !slices.Contains(colors, bot.Color) {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_COLOR", "Invalid color", nil)
		return bot, false
	}

	return bot, true
}

func HandleRegenerateBotWebhook(w http.ResponseWriter, r *http.Request) {
	botID, err := strconv.Atoi(r.PathValue("botId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_BOT_ID", "Invalid bot", nil)
		return
	}

	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "WEBHOOK_UPDATE_FAILED", "Failed to create webhook", err)
		return
	}

	updated, err := UpdateBotWebhookToken(botID, hex.EncodeToString(tokenBytes))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "WEBHOOK_UPDATE_FAILED", "Failed to create webhook", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, updated)
}

func HandleDeleteBotWebhook(w http.ResponseWriter, r *http.Request) {
	botID, err := strconv.Atoi(r.PathValue("botId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_BOT_ID", "Invalid bot", nil)
		return
	}

	updated, err := UpdateBotWebhookToken(botID, "")
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "WEBHOOK_UPDATE_FAILED", "Failed to turn off webhook", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, updated)
}
