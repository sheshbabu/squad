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
