package schedules

import (
	"fmt"
	"log/slog"
	"slices"
	"squad/commons/queue"
	"strconv"
	"strings"
	"time"
)

func StartScheduler() {
	go func() {
		runDueSchedules()
		for range time.Tick(30 * time.Second) {
			runDueSchedules()
		}
	}()
}

// GetNextRunAt works in local time (set TZ) so 07:30 stays 07:30 across daylight saving changes.
func GetNextRunAt(schedule Schedule, after time.Time) (time.Time, error) {
	hour, minute, err := parseTimeOfDay(schedule.TimeOfDay)
	if err != nil {
		return time.Time{}, err
	}

	localAfter := after.In(time.Local)
	for dayOffset := 0; dayOffset <= 7; dayOffset++ {
		candidate := time.Date(localAfter.Year(), localAfter.Month(), localAfter.Day()+dayOffset, hour, minute, 0, 0, time.Local)
		if !candidate.After(after) {
			continue
		}

		if schedule.Frequency == FREQUENCY_DAILY || slices.Contains(schedule.Weekdays, int(candidate.Weekday())) {
			return candidate, nil
		}
	}

	return time.Time{}, fmt.Errorf("schedule %q never runs", schedule.Name)
}

func runDueSchedules() {
	dueSchedules, err := GetDueSchedules()
	if err != nil {
		slog.Error("error loading due schedules", "error", err)
		return
	}

	for _, schedule := range dueSchedules {
		if err := fireSchedule(schedule); err != nil {
			slog.Error("error running schedule", "scheduleID", schedule.ScheduleID, "error", err)
		}
	}
}

// fireSchedule runs a schedule once even if several runs were missed while the server was down, then moves it forward from now.
func fireSchedule(schedule Schedule) error {
	now := time.Now()

	nextRunAt, err := GetNextRunAt(schedule, now)
	if err != nil {
		return err
	}

	title := schedule.Name + " · " + now.Format("2 Jan")
	thread, err := StartScheduleRun(schedule, title, nextRunAt)
	if err != nil {
		return err
	}
	queue.Wake()

	slog.Info("schedule started a thread", "scheduleID", schedule.ScheduleID, "threadID", thread.ThreadID)
	return nil
}

func parseTimeOfDay(timeOfDay string) (int, int, error) {
	parts := strings.Split(timeOfDay, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("time must look like 07:30")
	}

	hour, hourErr := strconv.Atoi(parts[0])
	minute, minuteErr := strconv.Atoi(parts[1])
	if hourErr != nil || minuteErr != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("time must look like 07:30")
	}

	return hour, minute, nil
}
