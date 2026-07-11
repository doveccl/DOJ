package runner

import (
	"context"

	contractlimits "github.com/doveccl/doj/contract/limits"
)

type LocalRun struct {
	Runner       string
	Work         string
	UserWork     string
	TaskID       string
	CgroupRoot   string
	UserCommand  string
	JudgeCommand string
	UserIdentity ProcessIdentity
	UserGate     UserGate
	Mode         JudgeMode
	Case         Case
	Limits       Limits
}

type ProcessIdentity struct {
	UID     uint32
	GID     uint32
	Enabled bool
}

type UserGate interface {
	WaitUserRelease(context.Context, UserPID) error
}

type waitResult struct {
	name string
	err  error
}

func RunLocalCase(ctx context.Context, req LocalRun) (CaseResult, error) {
	var result CaseResult
	var err error
	if req.JudgeCommand == "" {
		result, err = runBuiltinLocalCase(ctx, req)
	} else {
		result, err = runCustomLocalCase(ctx, req)
	}
	if err == nil {
		result.Message = truncateRunes(result.Message, contractlimits.MaxCaseMessageRunes)
	}
	return result, err
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
