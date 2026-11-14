package agent

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"squad/commons/llm"
	"squad/commons/queue"
	"squad/commons/sqlite"
	"squad/commons/stream"
	"squad/commons/testutil"
	"squad/features/bots"
	"squad/features/threads"
	"squad/features/tools"
	"strings"
	"sync"
	"testing"
	"time"
)

const toolCallStream = `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"list_files","arguments":"{}"}}]}}]}

data: [DONE]

`

const finalStream = `data: {"choices":[{"delta":{"content":"All "}}]}

data: {"choices":[{"delta":{"content":"done"}}],"usage":{"prompt_tokens":3,"completion_tokens":2,"cost":0.01}}

data: [DONE]

`

const cutStream = `data: {"choices":[{"delta":{"content":"Partial"}}]}

`

type fakeGateway struct {
	mutex     sync.Mutex
	responses []func(w http.ResponseWriter, r *http.Request)
	requests  int
}

func (f *fakeGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mutex.Lock()
	index := f.requests
	f.requests++
	f.mutex.Unlock()

	if index >= len(f.responses) {
		http.Error(w, "unexpected request", http.StatusInternalServerError)
		return
	}
	f.responses[index](w, r)
}

func sendStream(body string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, body)
	}
}

func setupThread(t *testing.T, responses ...func(w http.ResponseWriter, r *http.Request)) (*fakeGateway, int) {
	t.Helper()
	testutil.SetupDB(t)
	tools.Init()

	fake := &fakeGateway{responses: responses}
	server := httptest.NewServer(fake)
	t.Cleanup(server.Close)
	llm.RegisterGateway(llm.Gateway{Name: "fake", BaseURL: server.URL})

	bot, err := bots.CreateBot(bots.Bot{Name: "Helper", Color: "gray", Gateway: "fake", Model: "fake-model"})
	if err != nil {
		t.Fatal(err)
	}

	tx, _ := sqlite.DB.Begin()
	thread, err := threads.CreateThreadTx(tx, bot.BotID, nil, "Test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := threads.InsertMessageTx(tx, threads.Message{ThreadID: thread.ThreadID, Role: threads.ROLE_USER, Content: "List my files"}); err != nil {
		t.Fatal(err)
	}
	if err := threads.StartRunTx(tx, thread.ThreadID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	return fake, thread.ThreadID
}

func getRoles(t *testing.T, threadID int) string {
	t.Helper()
	messages, err := threads.GetAllMessages(threadID)
	if err != nil {
		t.Fatal(err)
	}

	roles := []string{}
	for _, message := range messages {
		roles = append(roles, message.Role)
	}
	return strings.Join(roles, ",")
}

func getPayload(threadID int) string {
	return fmt.Sprintf(`{"threadId":%d}`, threadID)
}

func TestRunCallsToolThenAnswers(t *testing.T) {
	fake, threadID := setupThread(t, sendStream(toolCallStream), sendStream(finalStream))

	if err := runAgent(context.Background(), getPayload(threadID)); err != nil {
		t.Fatal(err)
	}

	if roles := getRoles(t, threadID); roles != "user,assistant,tool,assistant" {
		t.Fatalf("unexpected messages %s", roles)
	}

	messages, _ := threads.GetAllMessages(threadID)
	final := messages[3]
	if final.Content != "All done" || final.Cost != 0.01 || final.Model != "fake-model" {
		t.Fatalf("unexpected final message %+v", final)
	}

	thread, _ := threads.GetThread(threadID)
	if thread.Status != threads.STATUS_IDLE || fake.requests != 2 {
		t.Fatalf("expected idle after 2 requests, got %s after %d", thread.Status, fake.requests)
	}

	var titleJobs int
	sqlite.DB.QueryRow("SELECT COUNT(*) FROM jobs WHERE kind = ?", threads.JOB_THREAD_TITLE).Scan(&titleJobs)
	if titleJobs != 1 {
		t.Fatal("expected a title job after the first reply")
	}
}

func TestRetryResumesFromLastSavedStep(t *testing.T) {
	fake, threadID := setupThread(t, sendStream(toolCallStream), sendStream(cutStream), sendStream(finalStream))

	err := runAgent(context.Background(), getPayload(threadID))
	if err == nil || queue.IsPermanent(err) {
		t.Fatalf("expected a retryable error, got %v", err)
	}
	if roles := getRoles(t, threadID); roles != "user,assistant,tool" {
		t.Fatalf("unexpected messages after cut %s", roles)
	}
	if stream.GetPartial(threadID) != "" {
		t.Fatal("partial text from a failed attempt should be reset")
	}

	if err := runAgent(context.Background(), getPayload(threadID)); err != nil {
		t.Fatal(err)
	}
	if roles := getRoles(t, threadID); roles != "user,assistant,tool,assistant" {
		t.Fatalf("retry duplicated or lost messages: %s", roles)
	}
	if fake.requests != 3 {
		t.Fatalf("expected the tool step to be skipped on retry, got %d requests", fake.requests)
	}
}

func TestClientErrorFailsPermanently(t *testing.T) {
	_, threadID := setupThread(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"No auth credentials found"}}`)
	})

	err := runAgent(context.Background(), getPayload(threadID))
	if !queue.IsPermanent(err) {
		t.Fatalf("expected a permanent error, got %v", err)
	}

	handleAgentFailed(getPayload(threadID), err)
	thread, _ := threads.GetThread(threadID)
	if thread.Status != threads.STATUS_FAILED || thread.ErrorMessage != "No auth credentials found" {
		t.Fatalf("unexpected thread state %s %q", thread.Status, thread.ErrorMessage)
	}
}

func TestCancelKeepsPartialText(t *testing.T) {
	_, threadID := setupThread(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hel\"}}]}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error)
	go func() {
		done <- runAgent(ctx, getPayload(threadID))
	}()

	deadline := time.Now().Add(2 * time.Second)
	for stream.GetPartial(threadID) != "Hel" {
		if time.Now().After(deadline) {
			t.Fatal("partial text never arrived")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()

	if err := <-done; err != nil {
		t.Fatalf("a cancelled run should finish cleanly, got %v", err)
	}

	messages, _ := threads.GetAllMessages(threadID)
	last := messages[len(messages)-1]
	if last.Role != threads.ROLE_ASSISTANT || last.Content != "Hel" || !last.IsCancelled {
		t.Fatalf("expected the partial reply to be kept, got %+v", last)
	}

	thread, _ := threads.GetThread(threadID)
	if thread.Status != threads.STATUS_IDLE {
		t.Fatalf("expected idle after cancel, got %s", thread.Status)
	}
}
