package users

import (
	"database/sql"
	"errors"
	"fmt"
	"squad/commons/sqlite"
	"squad/commons/utils"
)

type User struct {
	UserID  int    `json:"userId"`
	Email   string `json:"email"`
	AboutMe string `json:"aboutMe"`
}

type UserRecord struct {
	User
	PasswordHash string
}

func HasUsers() (bool, error) {
	var count int

	err := sqlite.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error counting users: %w", err)
	}

	return count > 0, nil
}

func GetUserByEmail(email string) (UserRecord, error) {
	var user UserRecord

	query := "SELECT user_id, email, password_hash FROM users WHERE email = ?"

	err := sqlite.DB.QueryRow(query, email).Scan(&user.UserID, &user.Email, &user.PasswordHash)
	if err != nil {
		return user, fmt.Errorf("error retrieving user: %w", utils.NotFoundIfNoRows(err))
	}

	return user, nil
}

func GetUserByID(userID int) (User, error) {
	var user User

	query := "SELECT user_id, email, about_me FROM users WHERE user_id = ?"

	err := sqlite.DB.QueryRow(query, userID).Scan(&user.UserID, &user.Email, &user.AboutMe)
	if err != nil {
		return user, fmt.Errorf("error retrieving user: %w", utils.NotFoundIfNoRows(err))
	}

	return user, nil
}

func InsertUser(email, passwordHash string) (User, error) {
	var user User

	query := "INSERT INTO users (email, password_hash) VALUES (?, ?) RETURNING user_id, email"

	err := sqlite.DB.QueryRow(query, email, passwordHash).Scan(&user.UserID, &user.Email)
	if err != nil {
		return user, fmt.Errorf("error inserting user: %w", err)
	}

	return user, nil
}

func GetUserRecordByID(userID int) (UserRecord, error) {
	var user UserRecord

	query := "SELECT user_id, email, password_hash FROM users WHERE user_id = ?"

	err := sqlite.DB.QueryRow(query, userID).Scan(&user.UserID, &user.Email, &user.PasswordHash)
	if err != nil {
		return user, fmt.Errorf("error retrieving user: %w", utils.NotFoundIfNoRows(err))
	}

	return user, nil
}

func UpdatePassword(userID int, passwordHash string) error {
	err := utils.RequireAffected(sqlite.DB.Exec("UPDATE users SET password_hash = ? WHERE user_id = ?", passwordHash, userID))
	if err != nil {
		return fmt.Errorf("error updating password: %w", err)
	}

	return nil
}

func UpdateAboutMe(userID int, aboutMe string) error {
	err := utils.RequireAffected(sqlite.DB.Exec("UPDATE users SET about_me = ? WHERE user_id = ?", aboutMe, userID))
	if err != nil {
		return fmt.Errorf("error updating about me: %w", err)
	}

	return nil
}

// GetAboutMe reads the only user's profile, since scheduled runs and webhooks have no session to identify them.
func GetAboutMe() (string, error) {
	var aboutMe string

	err := sqlite.DB.QueryRow("SELECT about_me FROM users ORDER BY user_id LIMIT 1").Scan(&aboutMe)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("error retrieving about me: %w", err)
	}

	return aboutMe, nil
}
