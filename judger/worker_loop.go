package judger

import (
	"context"
	"time"
)

type LoopConfig struct {
	Worker      WorkerConfig
	Concurrency int
	Logf        func(format string, args ...any)
}

func RunLoop(ctx context.Context, cfg LoopConfig) error {
	if cfg.Worker.Logf == nil {
		cfg.Worker.Logf = cfg.Logf
	}
	workers := cfg.Concurrency
	if workers <= 1 {
		return runLoopWorker(ctx, cfg)
	}
	errCh := make(chan error, workers)
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	for i := 0; i < workers; i++ {
		go func() {
			errCh <- runLoopWorker(workerCtx, cfg)
		}()
	}
	first := <-errCh
	cancel()
	for i := 1; i < workers; i++ {
		if err := <-errCh; first == nil {
			first = err
		}
	}
	return first
}

func runLoopWorker(ctx context.Context, cfg LoopConfig) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		worked, err := RunOne(ctx, cfg.Worker)
		if err != nil {
			if cfg.Logf != nil {
				cfg.Logf("judger task failed: %v", err)
			}
			if err := sleepContext(ctx, time.Second); err != nil {
				return err
			}
			continue
		}
		if !worked {
			continue
		}
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
