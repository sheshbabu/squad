package usage

import (
	"fmt"
	"squad/commons/sqlite"
	"time"
)

type CostEntry struct {
	ThreadID    int
	Title       string
	IsScheduled bool
	Model       string
	Cost        float64
	CreatedAt   time.Time
}

func GetBotCostEntries(botID int, since time.Time) ([]CostEntry, error) {
	entries := []CostEntry{}

	query := `
		SELECT m.thread_id, t.title, t.schedule_id IS NOT NULL, m.model, m.cost, m.created_at
		FROM messages m
		JOIN threads t ON t.thread_id = m.thread_id
		WHERE t.bot_id = ? AND m.cost > 0 AND m.created_at >= ?
		ORDER BY m.created_at
	`

	rows, err := sqlite.DB.Query(query, botID, since.UTC().Format(time.DateTime))
	if err != nil {
		return nil, fmt.Errorf("error retrieving costs for bot %d: %w", botID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var entry CostEntry
		if err := rows.Scan(&entry.ThreadID, &entry.Title, &entry.IsScheduled, &entry.Model, &entry.Cost, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("error scanning cost: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating costs: %w", err)
	}

	return entries, nil
}
