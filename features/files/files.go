package files

import (
	"encoding/json"
	"errors"
	"net/http"
	"squad/commons/utils"
)

type WriteFileRequest struct {
	Content string `json:"content"`
}

func HandleGetFiles(w http.ResponseWriter, r *http.Request) {
	botID, ok := utils.ParsePathID(w, r, "botId", "INVALID_BOT_ID", "Invalid bot")
	if !ok {
		return
	}

	files, err := ListFiles(botID)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "FILES_READ_FAILED", "Failed to load files", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, files)
}

func HandleGetFile(w http.ResponseWriter, r *http.Request) {
	botID, ok := utils.ParsePathID(w, r, "botId", "INVALID_BOT_ID", "Invalid bot")
	if !ok {
		return
	}

	fileContent, err := ReadFile(botID, r.PathValue("name"))
	if errors.Is(err, ErrInvalidName) {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_FILE_NAME", err.Error(), nil)
		return
	}
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "FILE_READ_FAILED", "Failed to load file", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, fileContent)
}

func HandleWriteFile(w http.ResponseWriter, r *http.Request) {
	botID, ok := utils.ParsePathID(w, r, "botId", "INVALID_BOT_ID", "Invalid bot")
	if !ok {
		return
	}

	var payload WriteFileRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request data", nil)
		return
	}

	name := r.PathValue("name")
	err := WriteFile(botID, name, payload.Content)
	if errors.Is(err, ErrInvalidName) {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_FILE_NAME", err.Error(), nil)
		return
	}
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "FILE_WRITE_FAILED", "Failed to save file", err)
		return
	}

	fileContent, err := ReadFile(botID, name)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "FILE_READ_FAILED", "Failed to load file", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, fileContent)
}

func HandleDeleteFile(w http.ResponseWriter, r *http.Request) {
	botID, ok := utils.ParsePathID(w, r, "botId", "INVALID_BOT_ID", "Invalid bot")
	if !ok {
		return
	}

	err := DeleteFile(botID, r.PathValue("name"))
	if errors.Is(err, ErrInvalidName) {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_FILE_NAME", err.Error(), nil)
		return
	}
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "FILE_DELETE_FAILED", "Failed to delete file", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
