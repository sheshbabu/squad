package tools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"squad/features/bots"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

const calendarDateFormat = "2006-01-02"

type calendarEvent struct {
	UID          string
	Summary      string
	Location     string
	Description  string
	Start        time.Time
	End          time.Time
	Duration     string
	IsAllDay     bool
	IsCancelled  bool
	RecurrenceID time.Time
	RRule        string
	ExDates      []time.Time
}

type calendarProperty struct {
	Name   string
	Params map[string]string
	Value  string
}

func registerCalendarTools() {
	register("calendar", Tool{
		Name:        "calendar_events",
		Description: "List events from the user's calendar between two dates, with repeating events expanded. Times are in the user's time zone.",
		Parameters: objectSchema(map[string]any{
			"from": stringProperty("First day to include, as YYYY-MM-DD."),
			"to":   stringProperty("Last day to include, as YYYY-MM-DD."),
		}, "from", "to"),
		Run: runCalendarEvents,
	})
}

func runCalendarEvents(ctx context.Context, bot bots.Bot, arguments string) (string, error) {
	var args struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := parseArguments(arguments, &args); err != nil {
		return "", err
	}

	fromDate, err := time.ParseInLocation(calendarDateFormat, args.From, time.Local)
	if err != nil {
		return "", fmt.Errorf("from must be YYYY-MM-DD")
	}
	toDate, err := time.ParseInLocation(calendarDateFormat, args.To, time.Local)
	if err != nil {
		return "", fmt.Errorf("to must be YYYY-MM-DD")
	}
	rangeEnd := toDate.AddDate(0, 0, 1)

	connection, err := pickConnection(bot.BotID, "calendar", arguments)
	if err != nil {
		return "", err
	}

	feed, err := fetchCalendar(ctx, connection.Secret)
	if err != nil {
		return "", err
	}

	events := expandCalendarEvents(parseCalendar(feed), fromDate, rangeEnd)
	if len(events) == 0 {
		return fmt.Sprintf("No events between %s and %s.", args.From, args.To), nil
	}

	lines := []string{}
	for _, event := range events {
		lines = append(lines, formatCalendarEvent(event))
	}

	return strings.Join(lines, "\n"), nil
}

func fetchCalendar(ctx context.Context, calendarURL string) (string, error) {
	calendarURL = strings.Replace(calendarURL, "webcal://", "https://", 1)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, calendarURL, nil)
	if err != nil {
		return "", fmt.Errorf("invalid calendar link")
	}

	// The request error repeats the whole URL, and the link's path is its secret, so only the cause underneath is kept.
	response, err := httpClient.Do(request)
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return "", fmt.Errorf("error fetching calendar: %w", urlErr.Err)
	}
	if err != nil {
		return "", fmt.Errorf("error fetching calendar")
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		return "", fmt.Errorf("calendar link returned status %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("error reading calendar: %w", err)
	}

	return string(body), nil
}

func parseCalendar(feed string) []calendarEvent {
	events := []calendarEvent{}
	var current *calendarEvent
	nestedDepth := 0

	for _, line := range unfoldCalendarLines(feed) {
		property := parseCalendarProperty(line)

		switch {
		case property.Name == "BEGIN" && property.Value == "VEVENT":
			current = &calendarEvent{}
		case current == nil:
			continue
		case property.Name == "BEGIN":
			nestedDepth++
		case property.Name == "END" && nestedDepth > 0:
			nestedDepth--
		case property.Name == "END" && property.Value == "VEVENT":
			if current.End.IsZero() {
				current.End = getCalendarEventEnd(*current)
			}
			events = append(events, *current)
			current = nil
		case nestedDepth > 0:
			continue
		default:
			applyCalendarProperty(current, property)
		}
	}

	return events
}

func unfoldCalendarLines(feed string) []string {
	lines := []string{}
	for _, line := range strings.Split(strings.ReplaceAll(feed, "\r\n", "\n"), "\n") {
		if len(lines) > 0 && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) {
			lines[len(lines)-1] += line[1:]
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

// The value starts at the first colon outside quotes, since quoted parameters like CN="Dr: Tan" can contain colons.
func parseCalendarProperty(line string) calendarProperty {
	isQuoted := false
	separator := -1
	for index, character := range line {
		if character == '"' {
			isQuoted = !isQuoted
		}
		if character == ':' && !isQuoted {
			separator = index
			break
		}
	}
	if separator == -1 {
		return calendarProperty{}
	}

	parts := strings.Split(line[:separator], ";")
	property := calendarProperty{Name: strings.ToUpper(parts[0]), Params: map[string]string{}, Value: line[separator+1:]}
	for _, param := range parts[1:] {
		key, value, _ := strings.Cut(param, "=")
		property.Params[strings.ToUpper(key)] = strings.Trim(value, `"`)
	}

	return property
}

func applyCalendarProperty(event *calendarEvent, property calendarProperty) {
	switch property.Name {
	case "UID":
		event.UID = property.Value
	case "SUMMARY":
		event.Summary = unescapeCalendarText(property.Value)
	case "LOCATION":
		event.Location = unescapeCalendarText(property.Value)
	case "DESCRIPTION":
		event.Description = unescapeCalendarText(property.Value)
	case "STATUS":
		event.IsCancelled = strings.EqualFold(property.Value, "CANCELLED")
	case "RRULE":
		event.RRule = property.Value
	case "DTSTART":
		event.Start, event.IsAllDay = parseCalendarTime(property)
	case "DTEND":
		event.End, _ = parseCalendarTime(property)
	case "DURATION":
		event.Duration = property.Value
	case "RECURRENCE-ID":
		event.RecurrenceID, _ = parseCalendarTime(property)
	case "EXDATE":
		for _, value := range strings.Split(property.Value, ",") {
			exDate, _ := parseCalendarTime(calendarProperty{Params: property.Params, Value: value})
			event.ExDates = append(event.ExDates, exDate)
		}
	}
}

// All-day dates and times without a time zone are floating, so they follow the user's time zone.
func parseCalendarTime(property calendarProperty) (time.Time, bool) {
	value := property.Value

	if property.Params["VALUE"] == "DATE" || len(value) == 8 {
		date, _ := time.ParseInLocation("20060102", value, time.Local)
		return date, true
	}

	if strings.HasSuffix(value, "Z") {
		parsed, _ := time.Parse("20060102T150405Z", value)
		return parsed, false
	}

	location := time.Local
	if property.Params["TZID"] != "" {
		if loaded, err := time.LoadLocation(property.Params["TZID"]); err == nil {
			location = loaded
		}
	}

	parsed, _ := time.ParseInLocation("20060102T150405", value, location)
	return parsed, false
}

func getCalendarEventEnd(event calendarEvent) time.Time {
	if event.Duration != "" {
		return addCalendarDuration(event.Start, event.Duration)
	}
	if event.IsAllDay {
		return event.Start.AddDate(0, 0, 1)
	}
	return event.Start
}

// Durations look like P1W, P3D or PT1H30M, and days are added as calendar days so all-day events keep their dates across DST changes.
func addCalendarDuration(start time.Time, value string) time.Time {
	value = strings.TrimPrefix(value, "+")
	if !strings.HasPrefix(value, "P") {
		return start
	}

	end := start
	number := 0
	for _, character := range value[1:] {
		switch character {
		case 'W':
			end = end.AddDate(0, 0, 7*number)
		case 'D':
			end = end.AddDate(0, 0, number)
		case 'H':
			end = end.Add(time.Duration(number) * time.Hour)
		case 'M':
			end = end.Add(time.Duration(number) * time.Minute)
		case 'S':
			end = end.Add(time.Duration(number) * time.Second)
		case 'T':
		default:
			number = number*10 + int(character-'0')
			continue
		}
		number = 0
	}

	return end
}

func unescapeCalendarText(value string) string {
	replacer := strings.NewReplacer(`\n`, "\n", `\N`, "\n", `\,`, ",", `\;`, ";", `\\`, `\`)
	return replacer.Replace(value)
}

// A moved or cancelled occurrence of a repeating event is its own event with a RECURRENCE-ID, which replaces that occurrence.
func expandCalendarEvents(events []calendarEvent, rangeStart, rangeEnd time.Time) []calendarEvent {
	movedOccurrences := map[string][]time.Time{}
	for _, event := range events {
		if !event.RecurrenceID.IsZero() {
			movedOccurrences[event.UID] = append(movedOccurrences[event.UID], event.RecurrenceID)
		}
	}

	expanded := []calendarEvent{}
	for _, event := range events {
		if event.IsCancelled || event.Start.IsZero() {
			continue
		}

		if event.RRule == "" {
			if event.Start.Before(rangeEnd) && event.End.After(rangeStart) {
				expanded = append(expanded, event)
			}
			continue
		}

		for _, start := range getOccurrences(event, movedOccurrences[event.UID], rangeStart, rangeEnd) {
			occurrence := event
			occurrence.Start = start
			occurrence.End = start.Add(event.End.Sub(event.Start))
			expanded = append(expanded, occurrence)
		}
	}

	slices.SortFunc(expanded, func(a, b calendarEvent) int {
		return a.Start.Compare(b.Start)
	})

	return expanded
}

func getOccurrences(event calendarEvent, movedOccurrences []time.Time, rangeStart, rangeEnd time.Time) []time.Time {
	option, err := rrule.StrToROptionInLocation(event.RRule, event.Start.Location())
	if err != nil {
		return []time.Time{}
	}
	option.Dtstart = event.Start

	rule, err := rrule.NewRRule(*option)
	if err != nil {
		return []time.Time{}
	}

	set := rrule.Set{}
	set.RRule(rule)
	for _, exDate := range append(slices.Clone(event.ExDates), movedOccurrences...) {
		set.ExDate(exDate)
	}

	duration := event.End.Sub(event.Start)
	occurrences := []time.Time{}
	for _, start := range set.Between(rangeStart.Add(-duration), rangeEnd, false) {
		if start.Add(duration).After(rangeStart) {
			occurrences = append(occurrences, start)
		}
	}

	return occurrences
}

func formatCalendarEvent(event calendarEvent) string {
	start := event.Start.In(time.Local)
	end := event.End.In(time.Local)

	var when string
	if event.IsAllDay {
		lastDay := end.AddDate(0, 0, -1)
		when = start.Format("Mon 2 Jan 2006") + ", all day"
		if lastDay.After(start) {
			when = fmt.Sprintf("%s to %s, all day", start.Format("Mon 2 Jan 2006"), lastDay.Format("Mon 2 Jan 2006"))
		}
	} else if start.Format(calendarDateFormat) == end.Format(calendarDateFormat) {
		when = fmt.Sprintf("%s, %s to %s", start.Format("Mon 2 Jan 2006"), start.Format("15:04"), end.Format("15:04"))
	} else {
		when = fmt.Sprintf("%s to %s", start.Format("Mon 2 Jan 2006 15:04"), end.Format("Mon 2 Jan 2006 15:04"))
	}

	line := fmt.Sprintf("- %s: %s", when, event.Summary)
	if event.Location != "" {
		line += " (" + event.Location + ")"
	}
	if event.Description != "" {
		line += "\n  " + strings.ReplaceAll(event.Description, "\n", "\n  ")
	}

	return line
}
