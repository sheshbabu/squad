package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"squad/commons/llm"
	"squad/commons/queue"
	"squad/commons/testutil"
	"squad/features/agent"
	"squad/features/tools"
	"strings"
	"testing"
	"time"
)

func TestChatEndToEnd(t *testing.T) {
	testutil.SetupDB(t)

	fakeGateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hi \"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"there\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer fakeGateway.Close()
	llm.RegisterGateway(llm.Gateway{Name: "fake", BaseURL: fakeGateway.URL})

	tools.Init()
	agent.Init()
	queue.Start(1)

	server := httptest.NewServer(newRouter())
	defer server.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	post := func(path, body string) map[string]any {
		t.Helper()
		response, err := client.Post(server.URL+path, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s returned %d", path, response.StatusCode)
		}
		var result map[string]any
		json.NewDecoder(response.Body).Decode(&result)
		return result
	}

	post("/api/users/", `{"email":"alex@example.com","password":"pw"}`)
	bot := post("/api/bots/", `{"name":"Helper","gateway":"fake","model":"fake-model"}`)
	thread := post(fmt.Sprintf("/api/bots/%v/threads/", bot["botId"]), `{"content":"Hello"}`)
	threadID := thread["threadId"]

	// The reply may already be done by the time the stream opens, so the final state is checked through the API too.
	events, err := client.Get(fmt.Sprintf("%s/api/threads/%v/events/", server.URL, threadID))
	if err != nil {
		t.Fatal(err)
	}
	defer events.Body.Close()
	go bufio.NewScanner(events.Body).Scan()

	deadline := time.Now().Add(5 * time.Second)
	for {
		response, err := client.Get(fmt.Sprintf("%s/api/threads/%v/", server.URL, threadID))
		if err != nil {
			t.Fatal(err)
		}
		var body struct {
			Thread struct {
				Status string `json:"status"`
			} `json:"thread"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		json.NewDecoder(response.Body).Decode(&body)
		response.Body.Close()

		if body.Thread.Status == "idle" && len(body.Messages) == 2 {
			if body.Messages[1].Content != "Hi there" {
				t.Fatalf("unexpected reply %q", body.Messages[1].Content)
			}
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("thread never finished, last state %+v", body)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
