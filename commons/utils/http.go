package utils

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
)

var ErrNotFound = errors.New("not found")

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func SendErrorResponse(w http.ResponseWriter, statusCode int, code string, message string, err error) {
	if err != nil {
		slog.Error("error handling request", "code", code, "error", err)
	}

	if errors.Is(err, ErrNotFound) {
		statusCode = http.StatusNotFound
	}

	SendJSONResponse(w, statusCode, ErrorResponse{
		Code:    code,
		Message: message,
	})
}

func ParsePathID(w http.ResponseWriter, r *http.Request, name string, code string, message string) (int, bool) {
	id, err := strconv.Atoi(r.PathValue(name))
	if err != nil {
		SendErrorResponse(w, http.StatusBadRequest, code, message, nil)
		return 0, false
	}

	return id, true
}

func SendJSONResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}
