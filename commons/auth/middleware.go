package auth

import (
	"net/http"
	"squad/commons/session"
	"squad/commons/utils"
)

func EnsureSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, isValid := GetSessionUserID(r); !isValid {
			utils.SendErrorResponse(w, http.StatusUnauthorized, "NO_SESSION", "Please log in again", nil)
			return
		}

		next(w, r)
	}
}

func GetSessionUserID(r *http.Request) (int, bool) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return 0, false
	}

	return session.GetValidUserID(cookie.Value)
}
