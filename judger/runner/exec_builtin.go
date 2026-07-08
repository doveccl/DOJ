package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func runBuiltinLocalCase(ctx context.Context, req LocalRun) (CaseResult, error) {
	if req.Runner == "" {
		return CaseResult{}, fmt.Errorf("runner path is required")
	}
	if req.Work == "" {
		return CaseResult{}, fmt.Errorf("work directory is required")
	}
	if req.UserCommand == "" {
		return CaseResult{}, fmt.Errorf("user command is required")
	}
	outputLimit := int64(req.Limits.OutputKB) * 1024
	if outputLimit <= 0 {
		outputLimit = defaultOutputLimit
	}
	resultPath := filepath.Join(req.Work, "judge-result-"+safeCaseID(req.Case.ID)+".txt")
	outputPath := filepath.Join(req.Work, "user-output-"+safeCaseID(req.Case.ID)+".txt")
	defer os.Remove(outputPath)

	cleanupGate := func() {}
	releasePath := ""
	needsReleaseGate := req.UserGate != nil || req.CgroupRoot != ""
	if needsReleaseGate {
		gateDir, err := os.MkdirTemp("", "doj-user-release-"+safeCaseID(req.Case.ID)+"-")
		if err != nil {
			return CaseResult{}, err
		}
		if req.UserIdentity.Enabled {
			_ = os.Chmod(gateDir, 0o755)
		}
		cleanupGate = func() { _ = os.RemoveAll(gateDir) }
		defer cleanupGate()
		releasePath = filepath.Join(gateDir, "release")
	}
	var cgroup *CgroupCase
	defer func() {
		if cgroup != nil {
			_ = cgroup.Cleanup()
		}
	}()
	applyStats := func(result CaseResult) CaseResult {
		if cgroup == nil {
			return result
		}
		stats, err := cgroup.Stats()
		if err != nil {
			return result
		}
		applyCgroupStatsSnapshot(&result, stats)
		return result
	}

	bin, args, err := parseCommand(req.UserCommand)
	if err != nil {
		return CaseResult{}, err
	}
	user := commandContext(ctx, bin, args...)
	if needsReleaseGate {
		wrapperArgs := []string{"runner", "wait-exec", releasePath, strconv.FormatUint(uint64(req.UserIdentity.UID), 10), strconv.FormatUint(uint64(req.UserIdentity.GID), 10), bin}
		user = commandContext(ctx, req.Runner, append(wrapperArgs, args...)...)
	}
	user.Dir = req.Work
	if req.UserWork != "" {
		user.Dir = req.UserWork
	}
	if err := prepareUserWorkIdentity(req.UserWork, req.UserIdentity); err != nil {
		return CaseResult{}, err
	}
	configureProcess(user)
	if !needsReleaseGate {
		applyProcessIdentity(user, req.UserIdentity)
	}

	input, err := os.Open(req.Case.Input)
	if err != nil {
		return CaseResult{}, err
	}
	defer input.Close()
	output, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return CaseResult{}, err
	}
	limitedOutput := &limitFileWriter{file: output, limit: outputLimit}
	var userErr bytes.Buffer
	user.Stdin = input
	user.Stdout = limitedOutput
	user.Stderr = &userErr

	startedAt := time.Now()
	if err := user.Start(); err != nil {
		_ = output.Close()
		return CaseResult{}, err
	}
	if req.UserGate != nil {
		if err := req.UserGate.WaitUserRelease(ctx, UserPID{TaskID: req.TaskID, CaseID: req.Case.ID, PID: user.Process.Pid}); err != nil {
			killProcessGroup(user)
			_ = user.Wait()
			_ = output.Close()
			if ctx.Err() != nil {
				return CaseResult{CaseID: req.Case.ID, Verdict: VerdictTimeLimit, TimeMS: elapsedMS(startedAt)}, nil
			}
			return CaseResult{CaseID: req.Case.ID, Verdict: VerdictSystemError, Message: err.Error()}, nil
		}
	}
	if req.CgroupRoot != "" {
		cg, err := PrepareCgroup(CgroupConfig{
			Root:         req.CgroupRoot,
			SubmissionID: safeCaseID(req.TaskID),
			CaseID:       safeCaseID(req.Case.ID),
			MemoryMax:    int64(req.Limits.MemoryKB) * 1024,
			PidsMax:      req.Limits.Pids,
		})
		if err != nil {
			killProcessGroup(user)
			_ = user.Wait()
			_ = output.Close()
			return CaseResult{CaseID: req.Case.ID, Verdict: VerdictSystemError, Message: err.Error()}, nil
		}
		cgroup = cg
		if err := cgroup.Add(user.Process.Pid); err != nil {
			killProcessGroup(user)
			_ = user.Wait()
			_ = output.Close()
			return CaseResult{CaseID: req.Case.ID, Verdict: VerdictSystemError, Message: err.Error()}, nil
		}
	}
	if needsReleaseGate {
		if err := os.WriteFile(releasePath, []byte("1"), 0o600); err != nil {
			killProcessGroup(user)
			_ = user.Wait()
			_ = output.Close()
			return CaseResult{CaseID: req.Case.ID, Verdict: VerdictSystemError, Message: err.Error()}, nil
		}
	}

	waitCh := make(chan error, 1)
	go func() { waitCh <- user.Wait() }()

	var userWait error
	select {
	case <-ctx.Done():
		killProcessGroup(user)
		select {
		case userWait = <-waitCh:
		case <-time.After(100 * time.Millisecond):
		}
		_ = output.Close()
		return applyStats(CaseResult{CaseID: req.Case.ID, Verdict: VerdictTimeLimit, TimeMS: elapsedMS(startedAt)}), nil
	case userWait = <-waitCh:
	}
	userTimeMS := elapsedMS(startedAt)
	if err := output.Close(); err != nil {
		return applyStats(CaseResult{CaseID: req.Case.ID, Verdict: VerdictSystemError, Message: err.Error()}), nil
	}
	if limitedOutput.overflow || errors.Is(userWait, errOutputLimit) {
		return applyStats(CaseResult{CaseID: req.Case.ID, Verdict: VerdictOutputLimit, TimeMS: userTimeMS}), nil
	}
	if userWait != nil {
		return applyStats(CaseResult{
			CaseID:  req.Case.ID,
			Verdict: VerdictRuntimeError,
			TimeMS:  userTimeMS,
			Message: userFailureMessage(userWait, userErr.String()),
		}), nil
	}
	exitCode, err := RunBuiltinChecker(req.Mode, req.Case.Input, outputPath, req.Case.Answer, resultPath)
	if err != nil {
		return applyStats(CaseResult{CaseID: req.Case.ID, Verdict: VerdictSystemError, TimeMS: userTimeMS, Message: err.Error()}), nil
	}
	return applyStats(caseResultFromTestlib(req, exitCode, resultPath, userTimeMS)), nil
}
