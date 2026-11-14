package users

import (
	"database/sql"
	"errors"
	"fmt"
	"squad/commons/sqlite"
	"squad/commons/utils"
)

type User struct {
	UserID int    `json:"userId"`
	Email  string `json:"email"`
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
	if errors.Is(err, sql.ErrNoRows) {
		return user, fmt.Errorf("error retrieving user: %w", utils.ErrNotFound)
	}
	if err != nil {
		return user, fmt.Errorf("error retrieving user: %w", err)
	}

	return user, nil
}

func GetUserByID(userID int) (User, error) {
	var user User

	query := "SELECT user_id, email FROM users WHERE user_id = ?"

	err := sqlite.DB.QueryRow(query, userID).Scan(&user.UserID, &user.Email)
	if errors.Is(err, sql.ErrNoRows) {
		return user, fmt.Errorf("error retrieving user: %w", utils.ErrNotFound)
	}
	if err != nil {
		return user, fmt.Errorf("error retrieving user: %w", err)
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
	if errors.Is(err, sql.ErrNoRows) {
		return user, fmt.Errorf("error retrieving user: %w", utils.ErrNotFound)
	}
	if err != nil {
		return user, fmt.Errorf("error retrieving user: %w", err)
	}

	return user, nil
}

func UpdatePassword(userID int, passwordHash string) error {
	result, err := sqlite.DB.Exec("UPDATE users SET password_hash = ? WHERE user_id = ?", passwordHash, userID)
	if err != nil {
		return fmt.Errorf("error updating password: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking updated password: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("error updating password: %w", utils.ErrNotFound)
	}

	return nil
}
