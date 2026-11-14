package queue

import (
	"context"
	"errors"
	"squad/commons/sqlite"
	"squad/commons/testutil"
	"sync"
	"testing"
	"time"
)

func getJobState(t *testing.T, key string) (string, int, bool) {
	t.Helper()

	var status string
	var attempts int
	var isDue bool
	err := sqlite.DB.QueryRow("SELECT status, attempts, run_after <= CURRENT_TIMESTAMP FROM jobs WHERE job_key = ?", key).Scan(&status, &attempts, &isDue)
	if err != nil {
		t.Fatalf("reading job %s: %v", key, err)
	}

	return status, attempts, isDue
}

func countJobs(t *testing.T) int {
	t.Helper()

	var count int
	if err := sqlite.DB.QueryRow("SELECT COUNT(*) FROM jobs").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestRetryableErrorRequeuesWithBackoff(t *testing.T) {
	testutil.SetupDB(t)

	calls := 0
	var retryDelay time.Duration
	RegisterHandler("test_flaky", Handler{
		Run: func(ctx context.Context, payload string) error {
			calls++
			if calls == 1 {
				return errors.New("temporary")
			}
			return nil
		},
		OnRetry: func(payload string, err error, delay time.Duration) { retryDelay = delay },
	})

	if err := Enqueue("test_flaky", "flaky", map[string]int{"id": 1}); err != nil {
		t.Fatal(err)
	}

	processNextJob()
	status, attempts, isDue := getJobState(t, "flaky")
	if status != STATUS_QUEUED || attempts != 1 || isDue {
		t.Fatalf("expected a delayed retry, got status=%s attempts=%d isDue=%v", status, attempts, isDue)
	}

	if processNextJob() {
		t.Fatal("a backed-off job must not be claimed early")
	}

	if retryDelay != 5*time.Second {
		t.Fatalf("expected OnRetry with the first backoff, got %s", retryDelay)
	}

	sqlite.DB.Exec("UPDATE jobs SET run_after = datetime('now', '-1 second')")
	processNextJob()

	if countJobs(t) != 0 {
		t.Fatal("a successful job should be deleted")
	}
}

func TestPermanentErrorFailsImmediately(t *testing.T) {
	testutil.SetupDB(t)

	var failedPayload string
	RegisterHandler("test_permanent", Handler{
		Run:      func(ctx context.Context, payload string) error { return Permanent(errors.New("bad key")) },
		OnFailed: func(payload string, err error) { failedPayload = payload },
	})

	Enqueue("test_permanent", "permanent", map[string]int{"id": 7})
	processNextJob()

	status, attempts, _ := getJobState(t, "permanent")
	if status != STATUS_FAILED || attempts != 1 || failedPayload != `{"id":7}` {
		t.Fatalf("expected failed after one attempt, got status=%s attempts=%d payload=%s", status, attempts, failedPayload)
	}

	// Enqueueing the same key again is how a retry works, and it clears the old failure.
	Enqueue("test_permanent", "permanent", map[string]int{"id": 7})
	status, attempts, _ = getJobState(t, "permanent")
	if status != STATUS_QUEUED || attempts != 0 || countJobs(t) != 1 {
		t.Fatalf("expected a fresh queued job, got status=%s attempts=%d count=%d", status, attempts, countJobs(t))
	}
}

func TestRecoverRunningJobs(t *testing.T) {
	testutil.SetupDB(t)

	sqlite.DB.Exec("INSERT INTO jobs (kind, job_key, payload, status) VALUES ('x', 'orphan', '{}', 'running')")
	recoverRunningJobs()

	status, _, _ := getJobState(t, "orphan")
	if status != STATUS_QUEUED {
		t.Fatalf("expected orphaned job to be requeued, got %s", status)
	}
}

func TestClaimIsExclusive(t *testing.T) {
	testutil.SetupDB(t)

	var mutex sync.Mutex
	runs := map[string]int{}
	RegisterHandler("test_count", Handler{Run: func(ctx context.Context, payload string) error {
		mutex.Lock()
		runs[payload]++
		mutex.Unlock()
		return nil
	}})

	for i := range 30 {
		Enqueue("test_count", "", map[string]int{"id": i})
	}

	var waitGroup sync.WaitGroup
	for range 3 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for processNextJob() {
			}
		}()
	}
	waitGroup.Wait()

	if len(runs) != 30 {
		t.Fatalf("expected 30 distinct jobs, got %d", len(runs))
	}
	for payload, count := range runs {
		if count != 1 {
			t.Fatalf("job %s ran %d times", payload, count)
		}
	}
}

func TestCancelStopsRunningJobAndDeletesQueuedOne(t *testing.T) {
	testutil.SetupDB(t)

	started := make(chan struct{})
	RegisterHandler("test_block", Handler{Run: func(ctx context.Context, payload string) error {
		close(started)
		<-ctx.Done()
		return nil
	}})

	Enqueue("test_block", "thread:1", map[string]int{"threadId": 1})

	done := make(chan struct{})
	go func() {
		processNextJob()
		close(done)
	}()
	<-started

	wasRunning, err := Cancel("thread:1")
	if err != nil || !wasRunning {
		t.Fatalf("expected to cancel a running job, got %v %v", wasRunning, err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled job did not stop")
	}

	Enqueue("test_block", "thread:2", map[string]int{"threadId": 2})
	wasRunning, _ = Cancel("thread:2")
	if wasRunning || countJobs(t) != 0 {
		t.Fatalf("expected the queued job to be deleted, wasRunning=%v count=%d", wasRunning, countJobs(t))
	}
}
