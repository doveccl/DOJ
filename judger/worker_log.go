package judger

import (
	"fmt"
	"time"
)

func logStep(logf func(format string, args ...any), submissionID uint, attempt int, step string, startedAt time.Time) {
	logTask(logf, submissionID, attempt, "%s=%s", step, formatDuration(time.Since(startedAt)))
}

func logTask(logf func(format string, args ...any), submissionID uint, attempt int, format string, args ...any) {
	if logf == nil {
		return
	}
	prefix := fmt.Sprintf("judger timing submission=%d attempt=%d ", submissionID, attempt)
	logf(prefix+format, args...)
}

func formatDuration(d time.Duration) string {
	return d.Round(time.Millisecond).String()
}
