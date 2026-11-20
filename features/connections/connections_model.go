package connections

import (
	"errors"
	"fmt"
	"squad/commons/sqlite"
	"squad/commons/utils"
	"time"
)

type Connection struct {
	ConnectionID int       `json:"connectionId"`
	BotID        int       `json:"botId"`
	GroupID      string    `json:"groupId"`
	Name         string    `json:"name"`
	Secret       string    `json:"secret"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

var ErrNameTaken = errors.New("connection name is taken")

const connectionColumns = "connection_id, bot_id, group_id, name, secret, created_at, updated_at"

func scanConnection(row interface{ Scan(...any) error }) (Connection, error) {
	var connection Connection
	err := row.Scan(&connection.ConnectionID, &connection.BotID, &connection.GroupID, &connection.Name, &connection.Secret, &connection.CreatedAt, &connection.UpdatedAt)
	return connection, err
}

func GetConnections(botID int) ([]Connection, error) {
	connections := []Connection{}

	rows, err := sqlite.DB.Query("SELECT "+connectionColumns+" FROM connections WHERE bot_id = ? ORDER BY connection_id", botID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving connections: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		connection, err := scanConnection(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning connection: %w", err)
		}
		connections = append(connections, connection)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating connections: %w", err)
	}

	return connections, nil
}

func CreateConnection(connection Connection) (Connection, error) {
	query := "INSERT INTO connections (bot_id, group_id, name, secret) VALUES (?, ?, ?, ?) RETURNING " + connectionColumns

	created, err := scanConnection(sqlite.DB.QueryRow(query, connection.BotID, connection.GroupID, connection.Name, connection.Secret))
	if err != nil {
		return created, fmt.Errorf("error creating connection: %w", wrapNameTaken(err))
	}

	return created, nil
}

func UpdateConnection(connection Connection) (Connection, error) {
	query := `
		UPDATE connections
		SET name = ?, secret = ?, updated_at = CURRENT_TIMESTAMP
		WHERE connection_id = ? AND bot_id = ?
		RETURNING ` + connectionColumns

	updated, err := scanConnection(sqlite.DB.QueryRow(query, connection.Name, connection.Secret, connection.ConnectionID, connection.BotID))
	if err != nil {
		return updated, fmt.Errorf("error updating connection %d: %w", connection.ConnectionID, wrapNameTaken(utils.NotFoundIfNoRows(err)))
	}

	return updated, nil
}

func DeleteConnection(botID, connectionID int) error {
	err := utils.RequireAffected(sqlite.DB.Exec("DELETE FROM connections WHERE connection_id = ? AND bot_id = ?", connectionID, botID))
	if err != nil {
		return fmt.Errorf("error deleting connection %d: %w", connectionID, err)
	}

	return nil
}

func wrapNameTaken(err error) error {
	if utils.IsUniqueViolation(err) {
		return fmt.Errorf("%w: %w", ErrNameTaken, err)
	}
	return err
}
