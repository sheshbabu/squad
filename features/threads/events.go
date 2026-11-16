package threads

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"squad/commons/stream"
	"squad/commons/utils"
	"time"
)

// Browsers refetch the thread whenever the connection opens, so anything sent while they were away is not lost.
func HandleThreadEvents(w http.ResponseWriter, r *http.Request) {
	threadID, ok := utils.ParsePathID(w, r, "threadId", "INVALID_THREAD_ID", "Invalid chat")
	if !ok {
		return
	}

	controller := http.NewResponseController(w)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	// Reverse proxies buffer responses by default, which would hold back every token until the reply ends.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	channel, partial := stream.Subscribe(threadID)
	defer stream.Unsubscribe(threadID, channel)

	fmt.Fprint(w, ": connected\n\n")
	if partial != "" {
		writeEvent(w, stream.Event{Type: "delta", Data: map[string]string{"text": partial}})
	}
	if err := controller.Flush(); err != nil {
		return
	}

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, isOpen := <-channel:
			if !isOpen {
				return
			}
			writeEvent(w, event)
		case <-heartbeat.C:
			fmt.Fprint(w, ": ping\n\n")
		}

		if err := controller.Flush(); err != nil {
			return
		}
	}
}

func writeEvent(w http.ResponseWriter, event stream.Event) {
	data, err := json.Marshal(event.Data)
	if err != nil {
		slog.Error("error encoding stream event", "type", event.Type, "error", err)
		return
	}

	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
}

// PublishStatus reads the thread back so every browser gets the same status and error the database holds.
func PublishStatus(threadID int) {
	thread, err := GetThread(threadID)
	if err != nil {
		slog.Error("error publishing thread status", "threadID", threadID, "error", err)
		return
	}

	stream.Publish(threadID, stream.Event{Type: "status", Data: map[string]string{"status": thread.Status, "errorMessage": thread.ErrorMessage}})
}

func PublishThread(thread Thread) {
	stream.Publish(thread.ThreadID, stream.Event{Type: "thread", Data: thread})
}
