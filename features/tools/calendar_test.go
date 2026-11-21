package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"squad/commons/llm"
	"squad/commons/testutil"
	"squad/features/bots"
	"squad/features/connections"
	"strings"
	"testing"
	"time"
)

const testCalendar = "BEGIN:VCALENDAR\r\n" +
	"VERSION:2.0\r\n" +
	"BEGIN:VTIMEZONE\r\n" +
	"TZID:America/New_York\r\n" +
	"BEGIN:STANDARD\r\n" +
	"DTSTART:19701101T020000\r\n" +
	"END:STANDARD\r\n" +
	"END:VTIMEZONE\r\n" +
	"BEGIN:VEVENT\r\n" +
	"UID:class\r\n" +
	"SUMMARY:Climbing class\r\n" +
	"DTSTART;TZID=America/New_York:20261027T183000\r\n" +
	"DTEND;TZID=America/New_York:20261027T193000\r\n" +
	"RRULE:FREQ=WEEKLY;BYDAY=TU\r\n" +
	"EXDATE;TZID=America/New_York:20261117T183000\r\n" +
	"BEGIN:VALARM\r\n" +
	"DESCRIPTION:Reminder\r\n" +
	"END:VALARM\r\n" +
	"END:VEVENT\r\n" +
	"BEGIN:VEVENT\r\n" +
	"UID:class\r\n" +
	"RECURRENCE-ID;TZID=America/New_York:20261124T183000\r\n" +
	"SUMMARY:Climbing class moved\r\n" +
	"DTSTART;TZID=America/New_York:20261125T183000\r\n" +
	"DTEND;TZID=America/New_York:20261125T193000\r\n" +
	"END:VEVENT\r\n" +
	"BEGIN:VEVENT\r\n" +
	"UID:birthday\r\n" +
	"SUMMARY:Amma's birthday\r\n" +
	"DTSTART;VALUE=DATE:19601123\r\n" +
	"DTEND;VALUE=DATE:19601124\r\n" +
	"RRULE:FREQ=YEARLY\r\n" +
	"END:VEVENT\r\n" +
	"BEGIN:VEVENT\r\n" +
	"UID:cancelled\r\n" +
	"SUMMARY:Cancelled dinner\r\n" +
	"STATUS:CANCELLED\r\n" +
	"DTSTART:20261110T110000Z\r\n" +
	"DTEND:20261110T120000Z\r\n" +
	"END:VEVENT\r\n" +
	"BEGIN:VEVENT\r\n" +
	"UID:doctor\r\n" +
	"SUMMARY:Dr Tan\\, follow-up\r\n" +
	"LOCATION:Clinic\r\n" +
	"DESCRIPTION:Bring reports\\nAsk about the\r\n" +
	"  dosage\r\n" +
	"DTSTART:20261112T020000Z\r\n" +
	"DTEND:20261112T030000Z\r\n" +
	"END:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

func useUTCAsLocal(t *testing.T) {
	t.Helper()
	previous := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = previous })
}

func TestCalendarExpandsRepeatsAndSkipsRemovedOccurrences(t *testing.T) {
	useUTCAsLocal(t)

	rangeStart := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)

	found := []string{}
	for _, event := range expandCalendarEvents(parseCalendar(testCalendar), rangeStart, rangeEnd) {
		found = append(found, fmt.Sprintf("%s %s", event.Start.UTC().Format("01-02 15:04"), event.Summary))
	}

	expected := []string{
		"11-03 23:30 Climbing class",
		"11-10 23:30 Climbing class",
		"11-12 02:00 Dr Tan, follow-up",
		"11-23 00:00 Amma's birthday",
		"11-25 23:30 Climbing class moved",
	}

	if strings.Join(found, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("unexpected events:\n%s", strings.Join(found, "\n"))
	}
}

func TestCalendarToolReadsTheConnectionLink(t *testing.T) {
	useUTCAsLocal(t)

	calendar := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testCalendar)
	}))
	defer calendar.Close()

	testutil.SetupDB(t)
	registry = map[string]Tool{}
	registryOrder = []string{}
	registerCalendarTools()

	bot, err := bots.CreateBot(bots.Bot{Name: "Health", Color: "gray", Gateway: "fake", Model: "fake-model"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := connections.CreateConnection(connections.Connection{BotID: bot.BotID, GroupID: "calendar", Name: "Shesh", Secret: calendar.URL}); err != nil {
		t.Fatal(err)
	}

	output := Run(context.Background(), bot, llm.ToolCall{Function: llm.FunctionCall{Name: "calendar_events", Arguments: `{"from":"2026-11-12","to":"2026-11-12"}`}})
	expected := "- Thu 12 Nov 2026, 02:00 to 03:00: Dr Tan, follow-up (Clinic)\n  Bring reports\n  Ask about the dosage"
	if output != expected {
		t.Fatalf("unexpected output:\n%s", output)
	}
}

func TestCalendarShowsDurationsAndMultiDayEvents(t *testing.T) {
	useUTCAsLocal(t)

	feed := "BEGIN:VCALENDAR\r\n" +
		"BEGIN:VEVENT\r\nSUMMARY:Call\r\nDTSTART:20261112T100000Z\r\nDURATION:PT1H30M\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nSUMMARY:Holiday\r\nDTSTART;VALUE=DATE:20261110\r\nDURATION:P3D\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nSUMMARY:Conference\r\nDTSTART:20261110T090000Z\r\nDTEND:20261113T170000Z\r\nEND:VEVENT\r\n" +
		"END:VCALENDAR\r\n"

	rangeStart := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)
	lines := []string{}
	for _, event := range expandCalendarEvents(parseCalendar(feed), rangeStart, rangeStart.AddDate(0, 0, 1)) {
		lines = append(lines, formatCalendarEvent(event))
	}

	expected := []string{
		"- Tue 10 Nov 2026 to Thu 12 Nov 2026, all day: Holiday",
		"- Tue 10 Nov 2026 09:00 to Fri 13 Nov 2026 17:00: Conference",
		"- Thu 12 Nov 2026, 10:00 to 11:30: Call",
	}
	if strings.Join(lines, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("unexpected events:\n%s", strings.Join(lines, "\n"))
	}
}

func TestCalendarFetchErrorHidesTheLink(t *testing.T) {
	_, err := fetchCalendar(context.Background(), "http://127.0.0.1:1/private-secret/basic.ics")
	if err == nil || strings.Contains(err.Error(), "private-secret") {
		t.Fatalf("expected an error without the link, got %v", err)
	}
}
