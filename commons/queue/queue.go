package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"squad/commons/sqlite"
	"sync"
	"time"
)

const (
	STATUS_QUEUED  = "queued"
	STATUS_RUNNING = "running"
	STATUS_FAILED  = "failed"
)

type Job struct {
	JobID       int
	Kind        string
	Key         string
	Payload     string
	Attempts    int
	MaxAttempts int
}

type Handler struct {
	Run         func(ctx context.Context, payload string) error
	OnRetry     func(payload string, err error, delay time.Duration)
	OnFailed    func(payload string, err error)
	MaxAttempts int
}

type permanentError struct {
	err error
}

func (e permanentError) Error() string {
	return e.err.Error()
}

func (e permanentError) Unwrap() error {
	return e.err
}

var handlers = map[string]Handler{}

var wake = make(chan struct{}, 1)

// runningMutex is held across claim and registration so Cancel never sees a job that is claimed but not yet cancellable.
var runningMutex sync.Mutex
var runningJobs = map[string]context.CancelFunc{}

func RegisterHandler(kind string, handler Handler) {
	handlers[kind] = handler
}

// Permanent marks an error that retrying can't fix, such as a bad API key or a deleted thread.
func Permanent(err error) error {
	return permanentError{err: err}
}

func IsPermanent(err error) bool {
	var target permanentError
	return errors.As(err, &target)
}

// EnqueueTx adds a job inside the caller's transaction so the job exists only if the domain write commits.
// Callers call Wake after committing.
func EnqueueTx(tx *sql.Tx, kind, key string, payload any) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error encoding job payload: %w", err)
	}

	maxAttempts := 5
	if handler, ok := handlers[kind]; ok && handler.MaxAttempts > 0 {
		maxAttempts = handler.MaxAttempts
	}

	// A failed job is kept for debugging until the same key is enqueued again, which is how retries work.
	if key != "" {
		_, err = tx.Exec("DELETE FROM jobs WHERE job_key = ? AND status = ?", key, STATUS_FAILED)
		if err != nil {
			return fmt.Errorf("error clearing failed jobs: %w", err)
		}
	}

	query := "INSERT INTO jobs (kind, job_key, payload, max_attempts) VALUES (?, ?, ?, ?)"

	_, err = tx.Exec(query, kind, key, string(payloadBytes), maxAttempts)
	if err != nil {
		return fmt.Errorf("error enqueuing job: %w", err)
	}

	return nil
}

func Enqueue(kind, key string, payload any) error {
	tx, err := sqlite.DB.Begin()
	if err != nil {
		return fmt.Errorf("error beginning enqueue tx: %w", err)
	}
	defer tx.Rollback()

	if err := EnqueueTx(tx, kind, key, payload); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing enqueue tx: %w", err)
	}

	Wake()
	return nil
}

func Wake() {
	select {
	case wake <- struct{}{}:
	default:
	}
}

// Cancel reports whether a running job was cancelled, in which case that job's handler finishes the cleanup.
func Cancel(key string) (bool, error) {
	runningMutex.Lock()
	defer runningMutex.Unlock()

	if cancel, ok := runningJobs[key]; ok {
		cancel()
		return true, nil
	}

	_, err := sqlite.DB.Exec("DELETE FROM jobs WHERE job_key = ? AND status = ?", key, STATUS_QUEUED)
	if err != nil {
		return false, fmt.Errorf("error deleting queued job: %w", err)
	}

	return false, nil
}

func Start(workerCount int) {
	recoverRunningJobs()

	for range workerCount {
		go runWorker()
	}
}

// Only this process runs jobs, so anything still marked running at boot was cut off by a crash or restart.
func recoverRunningJobs() {
	_, err := sqlite.DB.Exec("UPDATE jobs SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE status = ?", STATUS_QUEUED, STATUS_RUNNING)
	if err != nil {
		slog.Error("error recovering running jobs", "error", err)
	}
}

func runWorker() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		for processNextJob() {
		}

		select {
		case <-wake:
		case <-ticker.C:
		}
	}
}

func processNextJob() bool {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	job, err := claimNextJob(cancel)
	if err != nil {
		slog.Error("error claiming job", "error", err)
		return false
	}
	if job == nil {
		return false
	}
	defer unregisterRunningJob(job.Key)

	handler, ok := handlers[job.Kind]
	if !ok {
		finishJob(*job, Permanent(fmt.Errorf("no handler for job kind %s", job.Kind)), Handler{})
		return true
	}

	err = runHandler(ctx, handler, job.Payload)
	finishJob(*job, err, handler)
	return true
}

func claimNextJob(cancel context.CancelFunc) (*Job, error) {
	runningMutex.Lock()
	defer runningMutex.Unlock()

	query := `
		UPDATE jobs
		SET status = ?, attempts = attempts + 1, updated_at = CURRENT_TIMESTAMP
		WHERE job_id = (
			SELECT job_id FROM jobs
			WHERE status = ? AND run_after <= CURRENT_TIMESTAMP
			ORDER BY job_id
			LIMIT 1
		)
		RETURNING job_id, kind, job_key, payload, attempts, max_attempts
	`

	var job Job
	err := sqlite.DB.QueryRow(query, STATUS_RUNNING, STATUS_QUEUED).Scan(&job.JobID, &job.Kind, &job.Key, &job.Payload, &job.Attempts, &job.MaxAttempts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error claiming job: %w", err)
	}

	if job.Key != "" {
		runningJobs[job.Key] = cancel
	}

	return &job, nil
}

func unregisterRunningJob(key string) {
	runningMutex.Lock()
	defer runningMutex.Unlock()
	delete(runningJobs, key)
}

func runHandler(ctx context.Context, handler Handler, payload string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = Permanent(fmt.Errorf("job panicked: %v", r))
		}
	}()

	return handler.Run(ctx, payload)
}

func finishJob(job Job, err error, handler Handler) {
	if err == nil {
		_, deleteErr := sqlite.DB.Exec("DELETE FROM jobs WHERE job_id = ?", job.JobID)
		if deleteErr != nil {
			slog.Error("error deleting finished job", "jobID", job.JobID, "error", deleteErr)
		}
		return
	}

	if !IsPermanent(err) && job.Attempts < job.MaxAttempts {
		delay := getRetryDelay(job.Attempts)
		slog.Warn("job failed, will retry", "jobID", job.JobID, "kind", job.Kind, "attempts", job.Attempts, "delay", delay, "error", err)

		query := "UPDATE jobs SET status = ?, last_error = ?, run_after = datetime('now', ?), updated_at = CURRENT_TIMESTAMP WHERE job_id = ?"
		_, updateErr := sqlite.DB.Exec(query, STATUS_QUEUED, err.Error(), fmt.Sprintf("+%d seconds", int(delay.Seconds())), job.JobID)
		if updateErr != nil {
			slog.Error("error requeueing job", "jobID", job.JobID, "error", updateErr)
		}

		if handler.OnRetry != nil {
			handler.OnRetry(job.Payload, err, delay)
		}
		return
	}

	slog.Error("job failed", "jobID", job.JobID, "kind", job.Kind, "attempts", job.Attempts, "error", err)

	query := "UPDATE jobs SET status = ?, last_error = ?, updated_at = CURRENT_TIMESTAMP WHERE job_id = ?"
	_, updateErr := sqlite.DB.Exec(query, STATUS_FAILED, err.Error(), job.JobID)
	if updateErr != nil {
		slog.Error("error marking job failed", "jobID", job.JobID, "error", updateErr)
	}

	if handler.OnFailed != nil {
		handler.OnFailed(job.Payload, err)
	}
}

func getRetryDelay(attempts int) time.Duration {
	delay := 5 * time.Second
	for range attempts - 1 {
		delay *= 3
	}

	if delay > 5*time.Minute {
		return 5 * time.Minute
	}

	return delay
}
