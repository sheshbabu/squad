package memories

import (
	"net/http"
	"squad/commons/utils"
)

func HandleGetMemories(w http.ResponseWriter, r *http.Request) {
	botID, ok := utils.ParsePathID(w, r, "botId", "INVALID_BOT_ID", "Invalid bot")
	if !ok {
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
	botID, ok := utils.ParsePathID(w, r, "botId", "INVALID_BOT_ID", "Invalid bot")
	if !ok {
		return
	}

	memoryID, ok := utils.ParsePathID(w, r, "memoryId", "INVALID_MEMORY_ID", "Invalid memory")
	if !ok {
		return
	}

	if err := DeleteMemory(botID, memoryID); err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "MEMORY_DELETE_FAILED", "Failed to delete memory", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
