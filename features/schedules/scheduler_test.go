package schedules

import (
	"testing"
	"time"
)

func TestGetNextRunAt(t *testing.T) {
	location, _ := time.LoadLocation("Asia/Singapore")
	time.Local = location

	// Saturday 14 November 2026, 08:00
	saturdayMorning := time.Date(2026, 11, 14, 8, 0, 0, 0, location)

	cases := []struct {
		name     string
		schedule Schedule
		expected time.Time
	}{
		{"daily later today", Schedule{Frequency: FREQUENCY_DAILY, TimeOfDay: "09:15"}, time.Date(2026, 11, 14, 9, 15, 0, 0, location)},
		{"daily already passed", Schedule{Frequency: FREQUENCY_DAILY, TimeOfDay: "07:30"}, time.Date(2026, 11, 15, 7, 30, 0, 0, location)},
		{"weekdays skip the weekend", Schedule{Frequency: FREQUENCY_WEEKLY, TimeOfDay: "07:30", Weekdays: []int{1, 2, 3, 4, 5}}, time.Date(2026, 11, 16, 7, 30, 0, 0, location)},
		{"same weekday next week", Schedule{Frequency: FREQUENCY_WEEKLY, TimeOfDay: "07:00", Weekdays: []int{6}}, time.Date(2026, 11, 21, 7, 0, 0, 0, location)},
	}

	for _, testCase := range cases {
		next, err := GetNextRunAt(testCase.schedule, saturdayMorning)
		if err != nil || !next.Equal(testCase.expected) {
			t.Errorf("%s: expected %v, got %v (%v)", testCase.name, testCase.expected, next, err)
		}
	}

	if _, err := GetNextRunAt(Schedule{Frequency: FREQUENCY_DAILY, TimeOfDay: "25:00"}, saturdayMorning); err == nil {
		t.Error("expected an invalid time to be rejected")
	}
}
