package search

import (
	"fmt"
	"squad/commons/sqlite"
	"time"
)

type Result struct {
	ThreadID  int       `json:"threadId"`
	BotID     int       `json:"botId"`
	Title     string    `json:"title"`
	MessageID int       `json:"messageId"`
	Snippet   string    `json:"snippet"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// The frontend splits snippets on these markers and renders every part as text, so message markup is never parsed as HTML.
const highlightStart = "<mark>"
const highlightEnd = "</mark>"

func SearchMessages(matchQuery string) ([]Result, error) {
	results := []Result{}

	query := `
		SELECT t.thread_id, t.bot_id, t.title, m.message_id, snippet(messages_fts, 0, ?, ?, '…', 16), t.updated_at
		FROM messages_fts
		JOIN messages m ON m.message_id = messages_fts.rowid
		JOIN threads t ON t.thread_id = m.thread_id
		WHERE messages_fts MATCH ?
		ORDER BY rank
		LIMIT 100
	`

	rows, err := sqlite.DB.Query(query, highlightStart, highlightEnd, matchQuery)
	if err != nil {
		return nil, fmt.Errorf("error searching messages: %w", err)
	}
	defer rows.Close()

	seenThreadIDs := map[int]bool{}
	for rows.Next() {
		var result Result
		if err := rows.Scan(&result.ThreadID, &result.BotID, &result.Title, &result.MessageID, &result.Snippet, &result.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error scanning search result: %w", err)
		}

		// Results are ranked best first, so the first hit per thread is the one to show.
		if seenThreadIDs[result.ThreadID] || len(results) >= 30 {
			continue
		}
		seenThreadIDs[result.ThreadID] = true

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating search results: %w", err)
	}

	return results, nil
}

type ChatMatch struct {
	ThreadID  int       `json:"threadId"`
	Title     string    `json:"title"`
	Role      string    `json:"role"`
	Snippet   string    `json:"snippet"`
	CreatedAt time.Time `json:"createdAt"`
}

func SearchBotMessages(botID int, matchQuery string) ([]ChatMatch, error) {
	matches := []ChatMatch{}

	query := `
		SELECT t.thread_id, t.title, m.role, snippet(messages_fts, 0, '', '', '…', 64), m.created_at
		FROM messages_fts
		JOIN messages m ON m.message_id = messages_fts.rowid
		JOIN threads t ON t.thread_id = m.thread_id
		WHERE messages_fts MATCH ? AND t.bot_id = ?
		ORDER BY t.schedule_id IS NOT NULL, rank
		LIMIT 10
	`

	rows, err := sqlite.DB.Query(query, matchQuery, botID)
	if err != nil {
		return nil, fmt.Errorf("error searching bot messages: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var match ChatMatch
		if err := rows.Scan(&match.ThreadID, &match.Title, &match.Role, &match.Snippet, &match.CreatedAt); err != nil {
			return nil, fmt.Errorf("error scanning chat match: %w", err)
		}
		matches = append(matches, match)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating chat matches: %w", err)
	}

	return matches, nil
}
