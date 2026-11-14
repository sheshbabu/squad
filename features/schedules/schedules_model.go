package schedules

import (
	"database/sql"
	"errors"
	"fmt"
	"squad/commons/sqlite"
	"squad/commons/utils"
	"squad/features/threads"
	"strconv"
	"strings"
	"time"
)

const (
	FREQUENCY_DAILY  = "daily"
	FREQUENCY_WEEKLY = "weekly"

	// Times are stored in UTC in SQLite's own format so they compare correctly against CURRENT_TIMESTAMP.
	timestampFormat = "2006-01-02 15:04:05"
)

type Schedule struct {
	ScheduleID int        `json:"scheduleId"`
	BotID      int        `json:"botId"`
	Name       string     `json:"name"`
	Prompt     string     `json:"prompt"`
	Frequency  string     `json:"frequency"`
	TimeOfDay  string     `json:"timeOfDay"`
	Weekdays   []int      `json:"weekdays"`
	IsEnabled  bool       `json:"isEnabled"`
	NextRunAt  time.Time  `json:"nextRunAt"`
	LastRunAt  *time.Time `json:"lastRunAt"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

const scheduleColumns = "schedule_id, bot_id, name, prompt, frequency, time_of_day, weekdays, is_enabled, next_run_at, last_run_at, created_at, updated_at"

func scanSchedule(row interface{ Scan(...any) error }) (Schedule, error) {
	var schedule Schedule
	var weekdays string

	err := row.Scan(&schedule.ScheduleID, &schedule.BotID, &schedule.Name, &schedule.Prompt, &schedule.Frequency, &schedule.TimeOfDay, &weekdays, &schedule.IsEnabled, &schedule.NextRunAt, &schedule.LastRunAt, &schedule.CreatedAt, &schedule.UpdatedAt)
	if err != nil {
		return schedule, err
	}

	schedule.Weekdays = parseWeekdays(weekdays)
	return schedule, nil
}

func GetSchedules(botID int) ([]Schedule, error) {
	return querySchedules("SELECT "+scheduleColumns+" FROM schedules WHERE bot_id = ? ORDER BY name COLLATE NOCASE", botID)
}

func GetDueSchedules() ([]Schedule, error) {
	return querySchedules("SELECT " + scheduleColumns + " FROM schedules WHERE is_enabled = 1 AND next_run_at <= CURRENT_TIMESTAMP ORDER BY next_run_at")
}

func querySchedules(query string, args ...any) ([]Schedule, error) {
	schedules := []Schedule{}

	rows, err := sqlite.DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error retrieving schedules: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		schedule, err := scanSchedule(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning schedule: %w", err)
		}
		schedules = append(schedules, schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schedules: %w", err)
	}

	return schedules, nil
}

func CreateSchedule(schedule Schedule) (Schedule, error) {
	query := `
		INSERT INTO schedules (bot_id, name, prompt, frequency, time_of_day, weekdays, is_enabled, next_run_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING ` + scheduleColumns

	created, err := scanSchedule(sqlite.DB.QueryRow(query, schedule.BotID, schedule.Name, schedule.Prompt, schedule.Frequency, schedule.TimeOfDay, formatWeekdays(schedule.Weekdays), schedule.IsEnabled, formatTimestamp(schedule.NextRunAt)))
	if err != nil {
		return created, fmt.Errorf("error creating schedule: %w", err)
	}

	return created, nil
}

func UpdateSchedule(schedule Schedule) (Schedule, error) {
	query := `
		UPDATE schedules
		SET name = ?, prompt = ?, frequency = ?, time_of_day = ?, weekdays = ?, is_enabled = ?, next_run_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE schedule_id = ? AND bot_id = ?
		RETURNING ` + scheduleColumns

	updated, err := scanSchedule(sqlite.DB.QueryRow(query, schedule.Name, schedule.Prompt, schedule.Frequency, schedule.TimeOfDay, formatWeekdays(schedule.Weekdays), schedule.IsEnabled, formatTimestamp(schedule.NextRunAt), schedule.ScheduleID, schedule.BotID))
	if errors.Is(err, sql.ErrNoRows) {
		return updated, fmt.Errorf("error updating schedule %d: %w", schedule.ScheduleID, utils.ErrNotFound)
	}
	if err != nil {
		return updated, fmt.Errorf("error updating schedule %d: %w", schedule.ScheduleID, err)
	}

	return updated, nil
}

// StartScheduleRun creates the thread, queues its run and moves the schedule forward in one transaction, so a crash can't fire it twice.
func StartScheduleRun(schedule Schedule, title string, nextRunAt time.Time) (threads.Thread, error) {
	tx, err := sqlite.DB.Begin()
	if err != nil {
		return threads.Thread{}, fmt.Errorf("error beginning schedule tx: %w", err)
	}
	defer tx.Rollback()

	thread, err := threads.CreateThreadTx(tx, schedule.BotID, &schedule.ScheduleID, title)
	if err != nil {
		return thread, err
	}

	if _, err := threads.InsertMessageTx(tx, threads.Message{ThreadID: thread.ThreadID, Role: threads.ROLE_USER, Content: schedule.Prompt}); err != nil {
		return thread, err
	}

	if err := threads.StartRunTx(tx, thread.ThreadID); err != nil {
		return thread, err
	}

	if err := MarkScheduleRunTx(tx, schedule.ScheduleID, nextRunAt); err != nil {
		return thread, err
	}

	if err := tx.Commit(); err != nil {
		return thread, fmt.Errorf("error committing schedule run: %w", err)
	}

	return thread, nil
}

func MarkScheduleRunTx(tx *sql.Tx, scheduleID int, nextRunAt time.Time) error {
	query := "UPDATE schedules SET last_run_at = CURRENT_TIMESTAMP, next_run_at = ? WHERE schedule_id = ?"

	_, err := tx.Exec(query, formatTimestamp(nextRunAt), scheduleID)
	if err != nil {
		return fmt.Errorf("error recording schedule %d run: %w", scheduleID, err)
	}

	return nil
}

func DeleteSchedule(botID, scheduleID int) error {
	result, err := sqlite.DB.Exec("DELETE FROM schedules WHERE schedule_id = ? AND bot_id = ?", scheduleID, botID)
	if err != nil {
		return fmt.Errorf("error deleting schedule %d: %w", scheduleID, err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking deleted schedule %d: %w", scheduleID, err)
	}
	if count == 0 {
		return fmt.Errorf("error deleting schedule %d: %w", scheduleID, utils.ErrNotFound)
	}

	return nil
}

func formatTimestamp(value time.Time) string {
	return value.UTC().Format(timestampFormat)
}

func formatWeekdays(weekdays []int) string {
	values := []string{}
	for _, weekday := range weekdays {
		values = append(values, strconv.Itoa(weekday))
	}
	return strings.Join(values, ",")
}

func parseWeekdays(value string) []int {
	weekdays := []int{}
	for _, part := range strings.Split(value, ",") {
		weekday, err := strconv.Atoi(strings.TrimSpace(part))
		if err == nil {
			weekdays = append(weekdays, weekday)
		}
	}
	return weekdays
}
