package judger

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type taskProgress struct {
	stage string
	done  int64
	total *int64
}

type progressReporter struct {
	mu           sync.Mutex
	client       *http.Client
	cfg          WorkerConfig
	taskID       uint
	submissionID uint
	attempt      int
	current      taskProgress
}

func heartbeatLoop(ctx context.Context, progress *progressReporter) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			progress.Flush(ctx)
		}
	}
}

func (reporter *progressReporter) Update(stage string, done int64, total *int64) {
	reporter.update(stage, done, total)
}

func (reporter *progressReporter) Set(ctx context.Context, stage string, done int64, total *int64) {
	reporter.update(stage, done, total)
	reporter.Flush(ctx)
}

func (reporter *progressReporter) Flush(ctx context.Context) {
	reporter.mu.Lock()
	progress := reporter.current
	reporter.mu.Unlock()
	reporter.send(ctx, progress)
}

func (reporter *progressReporter) update(stage string, done int64, total *int64) {
	if stage == "" {
		return
	}
	var totalCopy *int64
	if total != nil {
		got := *total
		totalCopy = &got
	}
	reporter.mu.Lock()
	reporter.current = taskProgress{stage: stage, done: done, total: totalCopy}
	reporter.mu.Unlock()
}

func (reporter *progressReporter) send(ctx context.Context, progress taskProgress) {
	if progress.stage == "" {
		return
	}
	_ = postProgressHeartbeat(ctx, reporter.client, reporter.cfg, reporter.taskID, reporter.submissionID, reporter.attempt, progress)
}
