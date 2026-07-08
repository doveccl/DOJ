package judge

import (
	"context"
	"github.com/doveccl/doj/server/cache"
	"strconv"
	"time"
)

const progressTTL = 2 * time.Minute

type Progress struct {
	Attempt   int       `json:"attempt"`
	Stage     string    `json:"stage"`
	Done      int64     `json:"done"`
	Total     *int64    `json:"total,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func SaveProgress(ctx context.Context, submissionID uint, progress Progress) error {
	if submissionID == 0 || progress.Attempt <= 0 || progress.Stage == "" {
		return nil
	}
	return cache.Set(ctx, progressKey(submissionID), progress, progressTTL)
}

func ReadProgress(ctx context.Context, submissionID uint, attempt int) (*Progress, error) {
	if submissionID == 0 || attempt <= 0 {
		return nil, nil
	}
	var progress Progress
	found, err := cache.Get(ctx, progressKey(submissionID), &progress)
	if err != nil || !found || progress.Attempt != attempt {
		return nil, err
	}
	return &progress, nil
}

func DeleteProgress(ctx context.Context, submissionID uint) {
	if submissionID == 0 {
		return
	}
	_ = cache.Delete(ctx, progressKey(submissionID))
}

func ValidProgressStage(stage string) bool {
	switch stage {
	case "leased", "download", "prepare", "compile", "judge", "upload":
		return true
	default:
		return false
	}
}

func progressKey(id uint) string {
	return "doj:submission:" + strconv.FormatUint(uint64(id), 10) + ":progress"
}
