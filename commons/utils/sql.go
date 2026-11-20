package utils

import (
	"database/sql"
	"errors"

	"github.com/mattn/go-sqlite3"
)

func NotFoundIfNoRows(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// RequireAffected takes Exec's results directly, so a write that matched no rows reports ErrNotFound.
func RequireAffected(result sql.Result, err error) error {
	if err != nil {
		return err
	}

	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}

	return nil
}

func IsUniqueViolation(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
}
