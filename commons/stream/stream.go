package stream

import "sync"

type Event struct {
	Type string
	Data any
}

const subscriberBufferSize = 256

var mutex sync.Mutex
var subscribers = map[int]map[chan Event]bool{}
var partials = map[int]string{}

// Subscribe also returns the reply text streamed so far, so a browser that opens a thread mid-reply sees it immediately.
func Subscribe(threadID int) (chan Event, string) {
	mutex.Lock()
	defer mutex.Unlock()

	channel := make(chan Event, subscriberBufferSize)
	if subscribers[threadID] == nil {
		subscribers[threadID] = map[chan Event]bool{}
	}
	subscribers[threadID][channel] = true

	return channel, partials[threadID]
}

func Unsubscribe(threadID int, channel chan Event) {
	mutex.Lock()
	defer mutex.Unlock()

	if subscribers[threadID][channel] {
		removeSubscriber(threadID, channel)
	}
}

func Publish(threadID int, event Event) {
	mutex.Lock()
	defer mutex.Unlock()
	publish(threadID, event)
}

func AppendPartial(threadID int, text string) {
	mutex.Lock()
	defer mutex.Unlock()

	partials[threadID] += text
	publish(threadID, Event{Type: "delta", Data: map[string]string{"text": text}})
}

func GetPartial(threadID int) string {
	mutex.Lock()
	defer mutex.Unlock()
	return partials[threadID]
}

// ResetPartial tells browsers to drop text from a reply attempt that failed and will be retried.
func ResetPartial(threadID int) {
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := partials[threadID]; !ok {
		return
	}

	delete(partials, threadID)
	publish(threadID, Event{Type: "reset", Data: map[string]string{}})
}

func ClearPartial(threadID int) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(partials, threadID)
}

// A subscriber that falls this far behind is dropped rather than slowing the agent, and its browser reconnects and refetches.
func publish(threadID int, event Event) {
	for channel := range subscribers[threadID] {
		select {
		case channel <- event:
		default:
			removeSubscriber(threadID, channel)
		}
	}
}

func removeSubscriber(threadID int, channel chan Event) {
	delete(subscribers[threadID], channel)
	close(channel)
	if len(subscribers[threadID]) == 0 {
		delete(subscribers, threadID)
	}
}
