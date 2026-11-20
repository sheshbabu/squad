package connections

import (
	"encoding/json"
	"errors"
	"net/http"
	"squad/commons/utils"
	"strings"
)

func HandleGetConnections(w http.ResponseWriter, r *http.Request) {
	botID, ok := utils.ParsePathID(w, r, "botId", "INVALID_BOT_ID", "Invalid bot")
	if !ok {
		return
	}

	connections, err := GetConnections(botID)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "CONNECTIONS_READ_FAILED", "Failed to load connections", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, connections)
}

func HandleCreateConnection(w http.ResponseWriter, r *http.Request) {
	connection, ok := decodeConnection(w, r)
	if !ok {
		return
	}

	if connection.GroupID == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_CONNECTION", "Tool group is required", nil)
		return
	}

	created, err := CreateConnection(connection)
	if errors.Is(err, ErrNameTaken) {
		utils.SendErrorResponse(w, http.StatusConflict, "CONNECTION_NAME_TAKEN", "A connection with this name already exists", nil)
		return
	}
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "CONNECTION_CREATE_FAILED", "Failed to add connection", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, created)
}

func HandleUpdateConnection(w http.ResponseWriter, r *http.Request) {
	connectionID, ok := utils.ParsePathID(w, r, "connectionId", "INVALID_CONNECTION_ID", "Invalid connection")
	if !ok {
		return
	}

	connection, ok := decodeConnection(w, r)
	if !ok {
		return
	}
	connection.ConnectionID = connectionID

	updated, err := UpdateConnection(connection)
	if errors.Is(err, ErrNameTaken) {
		utils.SendErrorResponse(w, http.StatusConflict, "CONNECTION_NAME_TAKEN", "A connection with this name already exists", nil)
		return
	}
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "CONNECTION_UPDATE_FAILED", "Failed to update connection", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, updated)
}

func HandleDeleteConnection(w http.ResponseWriter, r *http.Request) {
	botID, ok := utils.ParsePathID(w, r, "botId", "INVALID_BOT_ID", "Invalid bot")
	if !ok {
		return
	}

	connectionID, ok := utils.ParsePathID(w, r, "connectionId", "INVALID_CONNECTION_ID", "Invalid connection")
	if !ok {
		return
	}

	if err := DeleteConnection(botID, connectionID); err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "CONNECTION_DELETE_FAILED", "Failed to delete connection", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func decodeConnection(w http.ResponseWriter, r *http.Request) (Connection, bool) {
	var connection Connection

	botID, ok := utils.ParsePathID(w, r, "botId", "INVALID_BOT_ID", "Invalid bot")
	if !ok {
		return connection, false
	}

	if err := json.NewDecoder(r.Body).Decode(&connection); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request data", nil)
		return connection, false
	}
	connection.BotID = botID

	connection.Name = strings.TrimSpace(connection.Name)
	connection.Secret = strings.TrimSpace(connection.Secret)

	if connection.Name == "" || connection.Secret == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_CONNECTION", "Name and secret are required", nil)
		return connection, false
	}

	return connection, true
}
