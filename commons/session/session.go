package session

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"squad/commons/sqlite"
	"time"
)

var sessionLength = time.Hour * 24 * 365
var expiresAtFormat = "2006-01-02T15:04:05-07:00"

type SessionRecord struct {
	SessionID string
	UserID    int
	ExpiresAt time.Time
}

func NewSession(userID int) (SessionRecord, error) {
	expiresAt := time.Now().Add(sessionLength)

	var session SessionRecord
	var expiresAtString string

	query := "INSERT INTO sessions (session_id, user_id, expires_at) VALUES (LOWER(HEX(RANDOMBLOB(16))), ?, ?) RETURNING session_id, user_id, expires_at"

	row := sqlite.DB.QueryRow(query, userID, expiresAt.Format(expiresAtFormat))
	err := row.Scan(&session.SessionID, &session.UserID, &expiresAtString)
	if err != nil {
		return session, fmt.Errorf("error creating session: %w", err)
	}

	session.ExpiresAt, err = time.Parse(expiresAtFormat, expiresAtString)
	if err != nil {
		return session, fmt.Errorf("error parsing session expiry: %w", err)
	}

	return session, nil
}

func SetSessionCookie(w http.ResponseWriter, session SessionRecord) {
	cookie := http.Cookie{Name: "session_token", Value: session.SessionID, Expires: session.ExpiresAt, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode}
	http.SetCookie(w, &cookie)
}

func ClearSessionCookie(w http.ResponseWriter) {
	cookie := http.Cookie{Name: "session_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode}
	http.SetCookie(w, &cookie)
}

// GetValidUserID returns false for unknown and expired sessions alike, since callers treat both as logged out.
func GetValidUserID(sessionID string) (int, bool) {
	query := "SELECT user_id, expires_at FROM sessions WHERE session_id = ?"
	var userID int
	var expiresAtString string

	err := sqlite.DB.QueryRow(query, sessionID).Scan(&userID, &expiresAtString)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false
	}
	if err != nil {
		slog.Error("error retrieving session", "error", err)
		return 0, false
	}

	expiresAt, err := time.Parse(expiresAtFormat, expiresAtString)
	if err != nil {
		slog.Error("error parsing session expiry", "error", err)
		return 0, false
	}

	if expiresAt.Before(time.Now()) {
		return 0, false
	}

	return userID, true
}

func DeleteSession(sessionID string) error {
	query := "DELETE FROM sessions WHERE session_id = ?"

	_, err := sqlite.DB.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("error deleting session: %w", err)
	}

	return nil
}

func DeleteExpiredSessions() {
	query := "DELETE FROM sessions WHERE expires_at < ?"

	_, err := sqlite.DB.Exec(query, time.Now().Format(expiresAtFormat))
	if err != nil {
		slog.Error("error deleting expired sessions", "error", err)
	}
}
