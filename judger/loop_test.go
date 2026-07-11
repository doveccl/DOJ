package judger

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestRunLoopStopsOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := RunLoop(ctx, LoopConfig{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("RunLoop error = %v, want context.Canceled", err)
	}
}

func TestRunLoopWaitsForAllWorkers(t *testing.T) {
	const workers = 3
	started := make(chan struct{}, workers)
	finished := make(chan struct{}, workers)
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		started <- struct{}{}
		<-req.Context().Done()
		finished <- struct{}{}
		return nil, req.Context().Err()
	})}
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunLoop(ctx, LoopConfig{Concurrency: workers, Worker: WorkerConfig{Server: "https://example.test", HTTPClient: client}})
	}()
	for i := 0; i < workers; i++ {
		<-started
	}
	cancel()
	if err := <-errCh; !errors.Is(err, context.Canceled) {
		t.Fatalf("RunLoop error = %v, want context.Canceled", err)
	}
	if len(finished) != workers {
		t.Fatalf("finished workers = %d, want %d", len(finished), workers)
	}
}
