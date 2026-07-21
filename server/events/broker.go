package events

import (
	"sync"
)

type Event struct {
	Type string
}

type Broker struct {
	mu          sync.RWMutex
	subscribers map[chan Event]struct{}
}

var Default = NewBroker()

func NewBroker() *Broker {
	return &Broker{subscribers: map[chan Event]struct{}{}}
}

func (broker *Broker) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, 16)
	broker.mu.Lock()
	broker.subscribers[ch] = struct{}{}
	broker.mu.Unlock()

	return ch, func() {
		broker.mu.Lock()
		if _, ok := broker.subscribers[ch]; ok {
			delete(broker.subscribers, ch)
			close(ch)
		}
		broker.mu.Unlock()
	}
}

func (broker *Broker) Publish(kind string) {
	event := Event{Type: kind}
	broker.mu.RLock()
	defer broker.mu.RUnlock()
	for ch := range broker.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func SubmissionChanged() {
	Default.Publish("submission")
}

func SubmissionProgressChanged() {
	Default.Publish("submission-progress")
}

func NotificationChanged() {
	Default.Publish("notification")
}
