package events

import (
	"testing"
	"time"
)

func TestBrokerDeliversAndUnsubscribes(t *testing.T) {
	broker := NewBroker()
	ch, unsubscribe := broker.Subscribe()
	broker.Publish("submission")

	select {
	case event := <-ch:
		if event.Type != "submission" {
			t.Fatalf("bad event: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("event was not delivered")
	}

	unsubscribe()
	broker.Publish("submission")

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("unsubscribed channel stayed open")
		}
	case <-time.After(time.Second):
		t.Fatal("unsubscribe did not close channel")
	}
}

func TestSubmissionEventsAreLightweight(t *testing.T) {
	old := Default
	broker := NewBroker()
	Default = broker
	t.Cleanup(func() { Default = old })

	ch, unsubscribe := broker.Subscribe()
	defer unsubscribe()

	SubmissionChanged()
	got := readEvent(t, ch)
	if got.Type != "submission" {
		t.Fatalf("submission event should only invalidate queries: %+v", got)
	}

	SubmissionProgressChanged()
	got = readEvent(t, ch)
	if got.Type != "submission-progress" {
		t.Fatalf("submission progress event should only invalidate queries: %+v", got)
	}

	NotificationChanged()
	got = readEvent(t, ch)
	if got.Type != "notification" {
		t.Fatalf("notification event should only invalidate queries: %+v", got)
	}
}

func readEvent(t *testing.T, ch <-chan Event) Event {
	t.Helper()
	select {
	case event := <-ch:
		return event
	case <-time.After(time.Second):
		t.Fatal("event was not delivered")
	}
	return Event{}
}
