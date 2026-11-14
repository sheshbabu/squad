package search

import (
	"squad/commons/sqlite"
	"squad/commons/testutil"
	"testing"
)

func TestBuildMatchQuery(t *testing.T) {
	cases := map[string]string{
		"":                  "",
		"passport":          `"passport"*`,
		`  where "is" it? `: `"where" """is""" "it?"*`,
	}

	for input, expected := range cases {
		if actual := buildMatchQuery(input); actual != expected {
			t.Errorf("%q: expected %s, got %s", input, expected, actual)
		}
	}
}

func TestSearchBotMessagesStaysWithinBot(t *testing.T) {
	testutil.SetupDB(t)

	sqlite.DB.Exec("INSERT INTO bots (name, model) VALUES ('Finance', 'm'), ('Health', 'm')")
	sqlite.DB.Exec("INSERT INTO threads (bot_id, title) VALUES (1, 'Insurance'), (2, 'Checkup')")
	sqlite.DB.Exec("INSERT INTO messages (thread_id, role, content) VALUES (1, 'user', 'We decided to keep the AIA premium'), (2, 'user', 'The AIA clinic was busy')")

	matches, err := SearchBotMessages(1, BuildAnyWordQuery("AIA premium decision"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Title != "Insurance" {
		t.Fatalf("expected only the Finance bot's chat, got %+v", matches)
	}
}

func TestSearchBotMessagesRanksScheduledChatsLast(t *testing.T) {
	testutil.SetupDB(t)

	sqlite.DB.Exec("INSERT INTO bots (name, model) VALUES ('News', 'm')")
	sqlite.DB.Exec("INSERT INTO schedules (bot_id, name, prompt, frequency, time_of_day, next_run_at) VALUES (1, 'Daily', 'news', 'daily', '08:00', CURRENT_TIMESTAMP)")
	sqlite.DB.Exec("INSERT INTO threads (bot_id, title, schedule_id) VALUES (1, 'Daily news', 1)")
	sqlite.DB.Exec("INSERT INTO threads (bot_id, title) VALUES (1, 'Our chat')")
	sqlite.DB.Exec("INSERT INTO messages (thread_id, role, content) VALUES (1, 'assistant', 'Singapore news Singapore news Singapore'), (2, 'user', 'I moved to Singapore')")

	matches, err := SearchBotMessages(1, BuildAnyWordQuery("Singapore"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 || matches[0].Title != "Our chat" {
		t.Fatalf("expected the chat with the user first, got %+v", matches)
	}
}
