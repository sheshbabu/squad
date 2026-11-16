package threads

import (
	"database/sql"
	"errors"
	"fmt"
	"squad/commons/queue"
	"squad/commons/sqlite"
	"squad/commons/utils"
	"time"
)

const (
	STATUS_IDLE    = "idle"
	STATUS_WORKING = "working"
	STATUS_FAILED  = "failed"

	JOB_AGENT_RUN    = "agent_run"
	JOB_THREAD_TITLE = "thread_title"
)

type Thread struct {
	ThreadID     int       `json:"threadId"`
	BotID        int       `json:"botId"`
	ScheduleID   *int      `json:"scheduleId"`
	Title        string    `json:"title"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"errorMessage"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type RunPayload struct {
	ThreadID int `json:"threadId"`
}

const threadColumns = "thread_id, bot_id, schedule_id, title, status, error_message, created_at, updated_at"

func scanThread(row interface{ Scan(...any) error }) (Thread, error) {
	var thread Thread
	err := row.Scan(&thread.ThreadID, &thread.BotID, &thread.ScheduleID, &thread.Title, &thread.Status, &thread.ErrorMessage, &thread.CreatedAt, &thread.UpdatedAt)
	return thread, err
}

// GetRunKey identifies a thread's agent run in the queue, which is how Stop finds it.
func GetRunKey(threadID int) string {
	return fmt.Sprintf("thread:%d", threadID)
}

func GetThreads(botID int) ([]Thread, error) {
	threads := []Thread{}

	query := "SELECT " + threadColumns + " FROM threads WHERE bot_id = ? ORDER BY updated_at DESC, thread_id DESC LIMIT 50"

	rows, err := sqlite.DB.Query(query, botID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving threads: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		thread, err := scanThread(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning thread: %w", err)
		}
		threads = append(threads, thread)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating threads: %w", err)
	}

	return threads, nil
}

func GetScheduleThreads(scheduleID int) ([]Thread, error) {
	threads := []Thread{}

	query := "SELECT " + threadColumns + " FROM threads WHERE schedule_id = ? ORDER BY created_at DESC, thread_id DESC LIMIT 20"

	rows, err := sqlite.DB.Query(query, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving schedule threads: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		thread, err := scanThread(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning schedule thread: %w", err)
		}
		threads = append(threads, thread)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schedule threads: %w", err)
	}

	return threads, nil
}

func GetThread(threadID int) (Thread, error) {
	return getThread(sqlite.DB, threadID)
}

func GetThreadTx(tx *sql.Tx, threadID int) (Thread, error) {
	return getThread(tx, threadID)
}

func getThread(queryer interface {
	QueryRow(string, ...any) *sql.Row
}, threadID int) (Thread, error) {
	thread, err := scanThread(queryer.QueryRow("SELECT "+threadColumns+" FROM threads WHERE thread_id = ?", threadID))
	if errors.Is(err, sql.ErrNoRows) {
		return thread, fmt.Errorf("error retrieving thread %d: %w", threadID, utils.ErrNotFound)
	}
	if err != nil {
		return thread, fmt.Errorf("error retrieving thread %d: %w", threadID, err)
	}

	return thread, nil
}

func CreateThreadTx(tx *sql.Tx, botID int, scheduleID *int, title string) (Thread, error) {
	query := "INSERT INTO threads (bot_id, schedule_id, title) VALUES (?, ?, ?) RETURNING " + threadColumns

	thread, err := scanThread(tx.QueryRow(query, botID, scheduleID, title))
	if err != nil {
		return thread, fmt.Errorf("error creating thread: %w", err)
	}

	return thread, nil
}

// The run is queued in the caller's transaction so it exists only if the message that triggered it is saved.
// Callers commit, then call queue.Wake and PublishStatus.
func StartRunTx(tx *sql.Tx, threadID int) error {
	if err := setThreadStatus(tx, threadID, STATUS_WORKING, ""); err != nil {
		return err
	}

	return queue.EnqueueTx(tx, JOB_AGENT_RUN, GetRunKey(threadID), RunPayload{ThreadID: threadID})
}

var ErrThreadBusy = errors.New("thread is busy")
var ErrThreadNotFailed = errors.New("thread has not failed")

func CreateThreadWithRun(botID int, title, content string, attachmentIDs []int) (Thread, error) {
	tx, err := sqlite.DB.Begin()
	if err != nil {
		return Thread{}, fmt.Errorf("error beginning thread tx: %w", err)
	}
	defer tx.Rollback()

	thread, err := CreateThreadTx(tx, botID, nil, title)
	if err != nil {
		return thread, err
	}

	if _, err := insertUserMessageTx(tx, botID, thread.ThreadID, content, attachmentIDs); err != nil {
		return thread, err
	}

	if err := StartRunTx(tx, thread.ThreadID); err != nil {
		return thread, err
	}

	if err := tx.Commit(); err != nil {
		return thread, fmt.Errorf("error committing thread: %w", err)
	}

	thread.Status = STATUS_WORKING
	return thread, nil
}

// AddMessageWithRun reads the status inside the write transaction so two tabs sending at once can't both start a run.
func AddMessageWithRun(threadID int, content string, attachmentIDs []int) (Message, error) {
	tx, err := sqlite.DB.Begin()
	if err != nil {
		return Message{}, fmt.Errorf("error beginning message tx: %w", err)
	}
	defer tx.Rollback()

	thread, err := GetThreadTx(tx, threadID)
	if err != nil {
		return Message{}, err
	}

	if thread.Status == STATUS_WORKING {
		return Message{}, fmt.Errorf("error sending to thread %d: %w", threadID, ErrThreadBusy)
	}

	message, err := insertUserMessageTx(tx, thread.BotID, threadID, content, attachmentIDs)
	if err != nil {
		return message, err
	}

	if err := StartRunTx(tx, threadID); err != nil {
		return message, err
	}

	if err := tx.Commit(); err != nil {
		return message, fmt.Errorf("error committing message: %w", err)
	}

	return message, nil
}

func RetryRun(threadID int) error {
	tx, err := sqlite.DB.Begin()
	if err != nil {
		return fmt.Errorf("error beginning retry tx: %w", err)
	}
	defer tx.Rollback()

	thread, err := GetThreadTx(tx, threadID)
	if err != nil {
		return err
	}

	if thread.Status != STATUS_FAILED {
		return fmt.Errorf("error retrying thread %d: %w", threadID, ErrThreadNotFailed)
	}

	if err := StartRunTx(tx, threadID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing retry: %w", err)
	}

	return nil
}

func SetThreadStatus(threadID int, status, errorMessage string) error {
	return setThreadStatus(sqlite.DB, threadID, status, errorMessage)
}

func setThreadStatus(execer interface {
	Exec(string, ...any) (sql.Result, error)
}, threadID int, status, errorMessage string) error {
	query := "UPDATE threads SET status = ?, error_message = ?, updated_at = CURRENT_TIMESTAMP WHERE thread_id = ?"

	_, err := execer.Exec(query, status, errorMessage, threadID)
	if err != nil {
		return fmt.Errorf("error updating thread %d status: %w", threadID, err)
	}

	return nil
}

func UpdateThreadTitle(threadID int, title string) (Thread, error) {
	query := "UPDATE threads SET title = ? WHERE thread_id = ? RETURNING " + threadColumns

	thread, err := scanThread(sqlite.DB.QueryRow(query, title, threadID))
	if errors.Is(err, sql.ErrNoRows) {
		return thread, fmt.Errorf("error updating thread %d title: %w", threadID, utils.ErrNotFound)
	}
	if err != nil {
		return thread, fmt.Errorf("error updating thread %d title: %w", threadID, err)
	}

	return thread, nil
}

func DeleteThread(threadID int) error {
	result, err := sqlite.DB.Exec("DELETE FROM threads WHERE thread_id = ?", threadID)
	if err != nil {
		return fmt.Errorf("error deleting thread %d: %w", threadID, err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking deleted thread %d: %w", threadID, err)
	}
	if count == 0 {
		return fmt.Errorf("error deleting thread %d: %w", threadID, utils.ErrNotFound)
	}

	return nil
}
