package memories

import (
	"fmt"
	"squad/commons/sqlite"
	"squad/commons/utils"
	"time"
)

type Memory struct {
	MemoryID  int       `json:"memoryId"`
	BotID     int       `json:"botId"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

func GetMemories(botID int) ([]Memory, error) {
	memories := []Memory{}

	rows, err := sqlite.DB.Query("SELECT memory_id, bot_id, content, created_at FROM memories WHERE bot_id = ? ORDER BY memory_id", botID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving memories: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var memory Memory
		if err := rows.Scan(&memory.MemoryID, &memory.BotID, &memory.Content, &memory.CreatedAt); err != nil {
			return nil, fmt.Errorf("error scanning memory: %w", err)
		}
		memories = append(memories, memory)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating memories: %w", err)
	}

	return memories, nil
}

func CreateMemory(botID int, content string) (Memory, error) {
	var memory Memory

	query := "INSERT INTO memories (bot_id, content) VALUES (?, ?) RETURNING memory_id, bot_id, content, created_at"

	err := sqlite.DB.QueryRow(query, botID, content).Scan(&memory.MemoryID, &memory.BotID, &memory.Content, &memory.CreatedAt)
	if err != nil {
		return memory, fmt.Errorf("error creating memory: %w", err)
	}

	return memory, nil
}

// DeleteMemory is scoped to the bot so one bot's tool call can't delete another bot's memory.
func DeleteMemory(botID, memoryID int) error {
	result, err := sqlite.DB.Exec("DELETE FROM memories WHERE memory_id = ? AND bot_id = ?", memoryID, botID)
	if err != nil {
		return fmt.Errorf("error deleting memory %d: %w", memoryID, err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking deleted memory %d: %w", memoryID, err)
	}
	if count == 0 {
		return fmt.Errorf("error deleting memory %d: %w", memoryID, utils.ErrNotFound)
	}

	return nil
}
