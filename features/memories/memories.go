package memories

import (
	"net/http"
	"squad/commons/utils"
	"strconv"
)

func HandleGetMemories(w http.ResponseWriter, r *http.Request) {
	botID, err := strconv.Atoi(r.PathValue("botId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_BOT_ID", "Invalid bot", nil)
		return
	}

	memories, err := GetMemories(botID)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "MEMORIES_READ_FAILED", "Failed to load memories", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, memories)
}

func HandleDeleteMemory(w http.ResponseWriter, r *http.Request) {
	botID, err := strconv.Atoi(r.PathValue("botId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_BOT_ID", "Invalid bot", nil)
		return
	}

	memoryID, err := strconv.Atoi(r.PathValue("memoryId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_MEMORY_ID", "Invalid memory", nil)
		return
	}

	if err := DeleteMemory(botID, memoryID); err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "MEMORY_DELETE_FAILED", "Failed to delete memory", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
