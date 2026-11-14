package threads

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"squad/commons/stream"
	"testing"
	"time"
)

func TestThreadEventsSendsPartialThenLiveEvents(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/threads/{threadId}/events/", HandleThreadEvents)
	server := httptest.NewServer(mux)
	defer server.Close()

	stream.AppendPartial(42, "Hello so far")

	response, err := http.Get(server.URL + "/api/threads/42/events/")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("unexpected content type %s", response.Header.Get("Content-Type"))
	}

	lines := make(chan string, 20)
	go func() {
		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()

	waitForLine := func(expected string) {
		t.Helper()
		deadline := time.After(2 * time.Second)
		for {
			select {
			case line := <-lines:
				if line == expected {
					return
				}
			case <-deadline:
				t.Fatalf("never received %q", expected)
			}
		}
	}

	waitForLine(`data: {"text":"Hello so far"}`)

	stream.Publish(42, stream.Event{Type: "status", Data: map[string]string{"status": "idle"}})
	waitForLine("event: status")
	waitForLine(`data: {"status":"idle"}`)

	stream.ClearPartial(42)
}
