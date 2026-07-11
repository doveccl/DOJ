package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func runCustomLocalCase(ctx context.Context, req LocalRun) (CaseResult, error) {
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
	transcriptPath := filepath.Join(req.Work, "judge-transcript-"+safeCaseID(req.Case.ID)+".txt")
	if err := touchPrivateFile(resultPath); err != nil {
		return CaseResult{}, err
	}
	if err := touchPrivateFile(transcriptPath); err != nil {
		return CaseResult{}, err
	}
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

	judgeArgs := []string{req.Case.Input, transcriptPath, req.Case.Answer, resultPath}
	judge := commandContext(ctx, req.JudgeCommand, judgeArgs...)
	bin, args, err := parseCommand(req.UserCommand)
	if err != nil {
		return CaseResult{}, err
	}
	user := commandContext(ctx, bin, args...)
	if needsReleaseGate {
		wrapperArgs := []string{"runner", "wait-exec", releasePath, strconv.FormatUint(uint64(req.UserIdentity.UID), 10), strconv.FormatUint(uint64(req.UserIdentity.GID), 10), bin}
		user = commandContext(ctx, req.Runner, append(wrapperArgs, args...)...)
	}
	judge.Dir = req.Work
	user.Dir = req.Work
	if req.UserWork != "" {
		user.Dir = req.UserWork
	}
	if err := prepareUserWorkIdentity(req.UserWork, req.UserIdentity); err != nil {
		return CaseResult{}, err
	}
	if req.UserIdentity.Enabled {
		identity := ProcessIdentity{UID: req.UserIdentity.UID + 1, GID: req.UserIdentity.GID + 1, Enabled: true}
		if err := prepareJudgeIdentity(req.Work, identity, map[string]os.FileMode{
			req.JudgeCommand: 0o550,
			req.Case.Input:   0o440,
			req.Case.Answer:  0o440,
			resultPath:       0o660,
			transcriptPath:   0o660,
		}); err != nil {
			return CaseResult{}, err
		}
		wrapperArgs := []string{"runner", "wait-exec", "/dev/null", strconv.FormatUint(uint64(identity.UID), 10), strconv.FormatUint(uint64(identity.GID), 10), req.JudgeCommand}
		judge = commandContext(ctx, req.Runner, append(wrapperArgs, judgeArgs...)...)
		judge.Dir = req.Work
	}
	configureProcess(judge)
	configureProcess(user)
	if !needsReleaseGate {
		applyProcessIdentity(user, req.UserIdentity)
	}

	judgeToUserR, judgeToUserW, err := os.Pipe()
	if err != nil {
		return CaseResult{}, err
	}
	userToJudgeR, userToJudgeW, err := os.Pipe()
	if err != nil {
		_ = judgeToUserR.Close()
		_ = judgeToUserW.Close()
		return CaseResult{}, err
	}
	defer closePipeFiles(judgeToUserR, judgeToUserW, userToJudgeR, userToJudgeW)

	judgeErr := &limitBuffer{limit: defaultCompileOutputLimit}
	userErr := &limitBuffer{limit: defaultCompileOutputLimit}
	userOutput := &limitFileWriter{file: userToJudgeW, limit: outputLimit}
	judge.Stdin = userToJudgeR
	judge.Stdout = judgeToUserW
	judge.Stderr = judgeErr
	user.Stdin = judgeToUserR
	user.Stdout = userOutput
	user.Stderr = userErr

	startedAt := time.Now()
	if err := judge.Start(); err != nil {
		return CaseResult{}, err
	}
	if err := user.Start(); err != nil {
		killProcessGroup(judge)
		_ = judge.Wait()
		return CaseResult{}, err
	}
	if req.UserGate != nil {
		if err := req.UserGate.WaitUserRelease(ctx, UserPID{TaskID: req.TaskID, CaseID: req.Case.ID, PID: user.Process.Pid}); err != nil {
			killProcessGroup(judge)
			killProcessGroup(user)
			_ = judge.Wait()
			_ = user.Wait()
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
			killProcessGroup(judge)
			killProcessGroup(user)
			_ = judge.Wait()
			_ = user.Wait()
			return CaseResult{CaseID: req.Case.ID, Verdict: VerdictSystemError, Message: err.Error()}, nil
		}
		cgroup = cg
		if err := cgroup.Add(user.Process.Pid); err != nil {
			killProcessGroup(judge)
			killProcessGroup(user)
			_ = judge.Wait()
			_ = user.Wait()
			return CaseResult{CaseID: req.Case.ID, Verdict: VerdictSystemError, Message: err.Error()}, nil
		}
	}
	if needsReleaseGate {
		if err := os.WriteFile(releasePath, []byte("1"), 0o600); err != nil {
			killProcessGroup(judge)
			killProcessGroup(user)
			_ = judge.Wait()
			_ = user.Wait()
			return CaseResult{CaseID: req.Case.ID, Verdict: VerdictSystemError, Message: err.Error()}, nil
		}
	}
	closePipeFiles(judgeToUserR, judgeToUserW, userToJudgeR)

	waitCh := make(chan waitResult, 2)
	go func() { waitCh <- waitResult{name: "judge", err: judge.Wait()} }()
	go func() { waitCh <- waitResult{name: "user", err: user.Wait()} }()

	var judgeWait error
	var userWait error
	judgeDone := false
	for waits := 0; waits < 2; waits++ {
		select {
		case <-ctx.Done():
			killProcessGroup(judge)
			killProcessGroup(user)
			judgeWait, userWait = collectWaitResults(waitCh, judgeWait, userWait)
			if judgeDone {
				result := caseResultFromTestlib(req, testlibExitCode(judgeWait), resultPath, elapsedMS(startedAt))
				if result.Verdict != VerdictAccepted {
					return applyStats(result), nil
				}
			}
			return applyStats(CaseResult{CaseID: req.Case.ID, Verdict: VerdictTimeLimit, TimeMS: elapsedMS(startedAt)}), nil
		case got := <-waitCh:
			if got.name == "judge" {
				judgeWait = got.err
				judgeDone = true
			} else {
				userWait = got.err
				_ = userToJudgeW.Close()
			}
		}
	}

	result := caseResultFromTestlib(req, testlibExitCode(judgeWait), resultPath, elapsedMS(startedAt))
	if userOutput.overflow || errors.Is(userWait, errOutputLimit) {
		result.Verdict = VerdictOutputLimit
		result.Score = 0
		result.Message = ""
		return applyStats(result), nil
	}
	if userWait != nil && result.Verdict != VerdictOutputLimit && !isBrokenPipeExit(userWait) {
		result.Verdict = VerdictRuntimeError
		result.Score = 0
		result.Message = userFailureMessage(userWait, userErr.String())
	}
	if result.Verdict == VerdictSystemError && result.Message == "" {
		result.Message = compactErrors("judge program failed", judgeWait, nil, judgeErr.String(), "")
	}
	return applyStats(result), nil
}

func prepareJudgeIdentity(work string, identity ProcessIdentity, files map[string]os.FileMode) error {
	if err := os.Chmod(work, 0o701); err != nil {
		return err
	}
	for path, mode := range files {
		rel, err := filepath.Rel(work, path)
		if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("judge file is outside work directory: %s", path)
		}
		for dir := filepath.Dir(path); dir != work; dir = filepath.Dir(dir) {
			if err := chmodIfNeeded(dir, 0o550); err != nil {
				return err
			}
			if err := os.Chown(dir, int(identity.UID), 0); err != nil {
				return err
			}
		}
		if err := chmodIfNeeded(path, mode); err != nil {
			return err
		}
		if err := os.Chown(path, int(identity.UID), 0); err != nil {
			return err
		}
	}
	return nil
}

func chmodIfNeeded(path string, mode os.FileMode) error {
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() == mode {
		return err
	}
	return os.Chmod(path, mode)
}
