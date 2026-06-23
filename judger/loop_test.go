package judger

import (
	"context"
	"errors"
	"testing"
)

func TestRunLoopStopsOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := RunLoop(ctx, LoopConfig{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("RunLoop error = %v, want context.Canceled", err)
	}
}
