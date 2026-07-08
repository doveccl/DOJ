package runner

import "context"

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
	if req.JudgeCommand == "" {
		return runBuiltinLocalCase(ctx, req)
	}
	return runCustomLocalCase(ctx, req)
}
