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

const (
	progressKeepaliveInterval = 10 * time.Second
	progressUpdateInterval    = time.Second
)

type progressReporter struct {
	mu           sync.Mutex
	client       *http.Client
	cfg          WorkerConfig
	taskID       uint
	submissionID uint
	attempt      int
	current      taskProgress
	sent         taskProgress
	sentAt       time.Time
}

func heartbeatLoop(ctx context.Context, progress *progressReporter) {
	ticker := time.NewTicker(progressUpdateInterval)
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
	reporter.FlushNow(ctx)
}

func (reporter *progressReporter) Flush(ctx context.Context) {
	reporter.mu.Lock()
	progress := reporter.current
	empty := progress.stage == ""
	changed := !sameProgress(progress, reporter.sent)
	elapsed := time.Since(reporter.sentAt)
	if empty || (!changed && elapsed < progressKeepaliveInterval) || (changed && progress.stage == reporter.sent.stage && elapsed < progressUpdateInterval) {
		reporter.mu.Unlock()
		return
	}
	reporter.sent = progress
	reporter.sentAt = time.Now()
	reporter.mu.Unlock()
	reporter.send(ctx, progress)
}

func (reporter *progressReporter) FlushNow(ctx context.Context) {
	reporter.mu.Lock()
	progress := reporter.current
	if progress.stage == "" {
		reporter.mu.Unlock()
		return
	}
	reporter.sent = progress
	reporter.sentAt = time.Now()
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
	_ = postProgressHeartbeat(ctx, reporter.client, reporter.cfg, reporter.taskID, reporter.submissionID, reporter.attempt, progress)
}

func sameProgress(a taskProgress, b taskProgress) bool {
	return a.stage == b.stage && a.done == b.done && sameTotal(a.total, b.total)
}

func sameTotal(a *int64, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
