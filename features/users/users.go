package users

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"squad/commons/auth"
	"squad/commons/session"
	"squad/commons/utils"
	"strings"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdatePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

func HandleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	hasUsers, err := HasUsers()
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "USER_READ_FAILED", "Failed to read users", err)
		return
	}

	if !hasUsers {
		utils.SendErrorResponse(w, http.StatusUnauthorized, "NO_USERS", "No users found", nil)
		return
	}

	userID, isValid := auth.GetSessionUserID(r)
	if !isValid {
		utils.SendErrorResponse(w, http.StatusUnauthorized, "NO_SESSION", "Session not found", nil)
		return
	}

	user, err := GetUserByID(userID)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "USER_READ_FAILED", "Failed to read user", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, user)
}

// HandleCreateUser only serves onboarding, since the app has a single user.
func HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	var payload CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request data", nil)
		return
	}

	hasUsers, err := HasUsers()
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "USER_READ_FAILED", "Failed to read users", err)
		return
	}

	if hasUsers {
		utils.SendErrorResponse(w, http.StatusConflict, "USER_EXISTS", "An account already exists", nil)
		return
	}

	if _, err := mail.ParseAddress(payload.Email); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_EMAIL", "Invalid email address", nil)
		return
	}

	if strings.TrimSpace(payload.Password) == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_PASSWORD", "Invalid password", nil)
		return
	}

	passwordHash, err := HashPassword(payload.Password)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "PASSWORD_HASH_FAILED", "Failed to process password", err)
		return
	}

	user, err := InsertUser(payload.Email, passwordHash)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "USER_CREATE_FAILED", "Failed to create user", err)
		return
	}

	newSession, err := session.NewSession(user.UserID)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "SESSION_CREATE_FAILED", "Failed to create session", err)
		return
	}
	session.SetSessionCookie(w, newSession)

	utils.SendJSONResponse(w, http.StatusOK, user)
}

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	var payload LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request data", nil)
		return
	}

	user, err := GetUserByEmail(payload.Email)
	if errors.Is(err, utils.ErrNotFound) {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INCORRECT_EMAIL", "Incorrect email address", nil)
		return
	}
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "USER_READ_FAILED", "Failed to read user", err)
		return
	}

	if !VerifyPassword(payload.Password, user.PasswordHash) {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INCORRECT_PASSWORD", "Incorrect password", nil)
		return
	}

	newSession, err := session.NewSession(user.UserID)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "SESSION_CREATE_FAILED", "Failed to create session", err)
		return
	}
	session.SetSessionCookie(w, newSession)

	utils.SendJSONResponse(w, http.StatusOK, user.User)
}

func HandleUpdatePassword(w http.ResponseWriter, r *http.Request) {
	var payload UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request data", nil)
		return
	}

	userID, isValid := auth.GetSessionUserID(r)
	if !isValid {
		utils.SendErrorResponse(w, http.StatusUnauthorized, "NO_SESSION", "Session not found", nil)
		return
	}

	user, err := GetUserRecordByID(userID)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "USER_READ_FAILED", "Failed to read user", err)
		return
	}

	if !VerifyPassword(payload.OldPassword, user.PasswordHash) {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INCORRECT_OLD_PASSWORD", "Incorrect password", nil)
		return
	}

	if strings.TrimSpace(payload.NewPassword) == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_NEW_PASSWORD", "New password is required", nil)
		return
	}

	if payload.OldPassword == payload.NewPassword {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_NEW_PASSWORD", "New password must be different", nil)
		return
	}

	passwordHash, err := HashPassword(payload.NewPassword)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "PASSWORD_HASH_FAILED", "Failed to process password", err)
		return
	}

	if err := UpdatePassword(userID, passwordHash); err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "PASSWORD_UPDATE_FAILED", "Failed to update password", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		if err := session.DeleteSession(cookie.Value); err != nil {
			utils.SendErrorResponse(w, http.StatusInternalServerError, "LOGOUT_FAILED", "Failed to log out", fmt.Errorf("error logging out: %w", err))
			return
		}
	}

	session.ClearSessionCookie(w)
	w.WriteHeader(http.StatusOK)
}
