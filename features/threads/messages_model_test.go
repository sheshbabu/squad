package threads

import (
	"fmt"
	"squad/commons/sqlite"
	"squad/commons/testutil"
	"testing"
)

func TestGetTurnsPagesByUserMessage(t *testing.T) {
	testutil.SetupDB(t)

	sqlite.DB.Exec("INSERT INTO bots (name, model) VALUES ('Helper', 'm')")
	sqlite.DB.Exec("INSERT INTO threads (bot_id, title) VALUES (1, 'Test')")

	for i := range 25 {
		InsertMessage(Message{ThreadID: 1, Role: ROLE_USER, Content: fmt.Sprintf("question %d", i)})
		InsertMessage(Message{ThreadID: 1, Role: ROLE_ASSISTANT, Content: fmt.Sprintf("answer %d", i)})
	}

	latest, hasMore, err := GetTurns(1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(latest) != 20 || !hasMore || latest[0].Content != "question 15" || latest[19].Content != "answer 24" {
		t.Fatalf("unexpected latest page: %d messages, hasMore=%v, first=%q", len(latest), hasMore, latest[0].Content)
	}

	middle, hasMore, _ := GetTurns(1, latest[0].MessageID)
	if len(middle) != 20 || !hasMore || middle[0].Content != "question 5" || middle[19].Content != "answer 14" {
		t.Fatalf("unexpected middle page: %d messages, hasMore=%v, first=%q", len(middle), hasMore, middle[0].Content)
	}

	oldest, hasMore, _ := GetTurns(1, middle[0].MessageID)
	if len(oldest) != 10 || hasMore || oldest[0].Content != "question 0" {
		t.Fatalf("unexpected oldest page: %d messages, hasMore=%v", len(oldest), hasMore)
	}
}

func TestSearchIndexFollowsInsertsAndDeletes(t *testing.T) {
	testutil.SetupDB(t)

	sqlite.DB.Exec("INSERT INTO bots (name, model) VALUES ('Helper', 'm')")
	sqlite.DB.Exec("INSERT INTO threads (bot_id, title) VALUES (1, 'Test')")
	InsertMessage(Message{ThreadID: 1, Role: ROLE_USER, Content: "where is the passport"})
	InsertMessage(Message{ThreadID: 1, Role: ROLE_TOOL, Content: "passport tool output"})

	var count int
	sqlite.DB.QueryRow("SELECT COUNT(*) FROM messages_fts WHERE messages_fts MATCH 'passport'").Scan(&count)
	if count != 1 {
		t.Fatalf("expected only the user message to be indexed, got %d", count)
	}

	DeleteThread(1)
	sqlite.DB.QueryRow("SELECT COUNT(*) FROM messages_fts WHERE messages_fts MATCH 'passport'").Scan(&count)
	if count != 0 {
		t.Fatalf("expected the index to drop deleted messages, got %d", count)
	}
}
