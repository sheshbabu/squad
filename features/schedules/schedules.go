package schedules

import (
	"encoding/json"
	"net/http"
	"squad/commons/utils"
	"strconv"
	"strings"
	"time"
)

func HandleGetSchedules(w http.ResponseWriter, r *http.Request) {
	botID, err := strconv.Atoi(r.PathValue("botId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_BOT_ID", "Invalid bot", nil)
		return
	}

	schedules, err := GetSchedules(botID)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "SCHEDULES_READ_FAILED", "Failed to load schedules", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, schedules)
}

func HandleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	schedule, ok := decodeSchedule(w, r)
	if !ok {
		return
	}

	created, err := CreateSchedule(schedule)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "SCHEDULE_CREATE_FAILED", "Failed to create schedule", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, created)
}

func HandleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	scheduleID, err := strconv.Atoi(r.PathValue("scheduleId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_SCHEDULE_ID", "Invalid schedule", nil)
		return
	}

	schedule, ok := decodeSchedule(w, r)
	if !ok {
		return
	}
	schedule.ScheduleID = scheduleID

	updated, err := UpdateSchedule(schedule)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "SCHEDULE_UPDATE_FAILED", "Failed to update schedule", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, updated)
}

func HandleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	botID, err := strconv.Atoi(r.PathValue("botId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_BOT_ID", "Invalid bot", nil)
		return
	}

	scheduleID, err := strconv.Atoi(r.PathValue("scheduleId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_SCHEDULE_ID", "Invalid schedule", nil)
		return
	}

	if err := DeleteSchedule(botID, scheduleID); err != nil {
		utils.SendErrorResponse(w, http.StatusInternalServerError, "SCHEDULE_DELETE_FAILED", "Failed to delete schedule", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func decodeSchedule(w http.ResponseWriter, r *http.Request) (Schedule, bool) {
	var schedule Schedule

	botID, err := strconv.Atoi(r.PathValue("botId"))
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_BOT_ID", "Invalid bot", nil)
		return schedule, false
	}

	if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST_BODY", "Invalid request data", nil)
		return schedule, false
	}
	schedule.BotID = botID

	schedule.Name = strings.TrimSpace(schedule.Name)
	schedule.Prompt = strings.TrimSpace(schedule.Prompt)

	if schedule.Name == "" || schedule.Prompt == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_SCHEDULE", "Name and task are required", nil)
		return schedule, false
	}

	if schedule.Frequency != FREQUENCY_DAILY && schedule.Frequency != FREQUENCY_WEEKLY {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_FREQUENCY", "Frequency must be daily or weekly", nil)
		return schedule, false
	}

	if schedule.Frequency == FREQUENCY_DAILY {
		schedule.Weekdays = []int{}
	}

	for _, weekday := range schedule.Weekdays {
		if weekday < 0 || weekday > 6 {
			utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_WEEKDAYS", "Invalid weekday", nil)
			return schedule, false
		}
	}

	if schedule.Frequency == FREQUENCY_WEEKLY && len(schedule.Weekdays) == 0 {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_WEEKDAYS", "Pick at least one day", nil)
		return schedule, false
	}

	nextRunAt, err := GetNextRunAt(schedule, time.Now())
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_TIME", err.Error(), nil)
		return schedule, false
	}
	schedule.NextRunAt = nextRunAt

	return schedule, true
}
