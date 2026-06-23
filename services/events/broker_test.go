package events

import (
	"testing"
	"time"
)

func TestBrokerDeliversAndUnsubscribes(t *testing.T) {
	broker := NewBroker()
	ch, unsubscribe := broker.Subscribe()
	broker.Publish("submission", map[string]string{"changed": "submission"})

	select {
	case event := <-ch:
		if event.Type != "submission" || string(event.Data) == "" {
			t.Fatalf("bad event: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("event was not delivered")
	}

	unsubscribe()
	broker.Publish("submission", map[string]string{"changed": "submission"})

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("unsubscribed channel stayed open")
		}
	case <-time.After(time.Second):
		t.Fatal("unsubscribe did not close channel")
	}
}
