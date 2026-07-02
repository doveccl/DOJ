package events

import (
	"strings"
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

func TestSubmissionEventsAreLightweight(t *testing.T) {
	old := Default
	broker := NewBroker()
	Default = broker
	t.Cleanup(func() { Default = old })

	ch, unsubscribe := broker.Subscribe()
	defer unsubscribe()

	SubmissionChanged()
	got := readEvent(t, ch)
	if got.Type != "submission" || string(got.Data) != `{"changed":"submission"}` {
		t.Fatalf("submission event should only invalidate queries: %+v data=%s", got, got.Data)
	}
	for _, field := range []string{"status", "score", "timeMs", "memoryKb", "message", "case", "progress", "code"} {
		if strings.Contains(string(got.Data), field) {
			t.Fatalf("submission event leaked result field %q: %s", field, got.Data)
		}
	}

	SubmissionProgressChanged()
	got = readEvent(t, ch)
	if got.Type != "submission-progress" || string(got.Data) != `{"changed":"submission-progress"}` {
		t.Fatalf("submission progress event should only invalidate queries: %+v data=%s", got, got.Data)
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
