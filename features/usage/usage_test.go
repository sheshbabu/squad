package usage

import (
	"squad/commons/sqlite"
	"squad/commons/testutil"
	"testing"
	"time"
)

func TestSummarizeCostsByDay(t *testing.T) {
	testutil.SetupDB(t)
	location, _ := time.LoadLocation("Asia/Singapore")
	time.Local = location

	sqlite.DB.Exec("INSERT INTO bots (name, model) VALUES ('Finance', 'm'), ('Health', 'm')")
	sqlite.DB.Exec("INSERT INTO schedules (bot_id, name, prompt, frequency, time_of_day, next_run_at) VALUES (1, 'Briefing', 'p', 'daily', '07:00', '2026-11-16 00:00:00')")
	sqlite.DB.Exec("INSERT INTO threads (bot_id, schedule_id, title) VALUES (1, NULL, 'Budget'), (1, 1, 'Briefing'), (2, NULL, 'Checkup')")

	// 23:30 UTC on the 13th is the 14th in Singapore.
	sqlite.DB.Exec(`INSERT INTO messages (thread_id, role, model, cost, created_at) VALUES
		(1, 'assistant', 'model-a', 0.10, '2026-11-13 23:30:00'),
		(1, 'assistant', 'model-b', 0.05, '2026-11-15 01:00:00'),
		(2, 'assistant', 'model-a', 0.20, '2026-11-15 00:30:00'),
		(3, 'assistant', 'model-a', 9.00, '2026-11-15 01:00:00'),
		(1, 'assistant', 'model-a', 5.00, '2026-11-01 01:00:00')`)

	today := time.Date(2026, 11, 15, 0, 0, 0, 0, time.Local)
	since := today.AddDate(0, 0, -6)
	entries, err := GetBotCostEntries(1, since)
	if err != nil {
		t.Fatal(err)
	}

	summary := summarizeCosts(entries, since, today, false)

	if len(summary.Buckets) != 7 {
		t.Fatalf("expected 7 days, got %d", len(summary.Buckets))
	}
	if summary.Buckets[5].ChatCost != 0.10 {
		t.Errorf("expected the late UTC reply on the 14th, got %+v", summary.Buckets[5])
	}
	if summary.Buckets[6].ChatCost != 0.05 || summary.Buckets[6].ScheduledCost != 0.20 {
		t.Errorf("expected chat and scheduled split on the 15th, got %+v", summary.Buckets[6])
	}
	if summary.ChatCount != 2 || summary.TopChats[0].Title != "Briefing" {
		t.Errorf("expected Briefing as the top of 2 chats, got %+v", summary.TopChats)
	}
	if len(summary.Models) != 2 || summary.Models[0].Model != "model-a" {
		t.Errorf("expected model-a first, got %+v", summary.Models)
	}
}
