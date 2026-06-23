package events

import (
	"encoding/json"
	"sync"
)

type Event struct {
	Type string
	Data []byte
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

func (broker *Broker) Publish(kind string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		payload = []byte("{}")
	}
	event := Event{Type: kind, Data: payload}

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
	Default.Publish("submission", map[string]string{"changed": "submission"})
}
