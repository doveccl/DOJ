package judger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
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
	resultPath := filepath.Join(req.Work, "judge-result-"+safeCaseID(req.Case.ID)+".json")
	userCommand := req.UserCommand
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
		userCommand = waitForReleaseCommand(releasePath, req.UserCommand)
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

	judge := judgeCommand(ctx, req, resultPath, outputLimit)
	user := shellCommand(ctx, userCommand)
	judge.Dir = req.Work
	user.Dir = req.Work
	if req.UserWork != "" {
		user.Dir = req.UserWork
	}
	if err := prepareUserWorkIdentity(req.UserWork, req.UserIdentity); err != nil {
		return CaseResult{}, err
	}
	configureProcess(judge)
	configureProcess(user)
	applyProcessIdentity(user, req.UserIdentity)

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

	var judgeErr bytes.Buffer
	var userErr bytes.Buffer
	judge.Stdin = userToJudgeR
	judge.Stdout = judgeToUserW
	judge.Stderr = &judgeErr
	user.Stdin = judgeToUserR
	user.Stdout = userToJudgeW
	user.Stderr = &userErr

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
				return CaseResult{CaseID: req.Case.ID, Verdict: VerdictTimeLimit, TimeMS: elapsedMS(startedAt), Message: ctx.Err().Error()}, nil
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
	closePipeFiles(judgeToUserR, judgeToUserW, userToJudgeR, userToJudgeW)

	waitCh := make(chan waitResult, 2)
	go func() { waitCh <- waitResult{name: "judge", err: judge.Wait()} }()
	go func() { waitCh <- waitResult{name: "user", err: user.Wait()} }()

	var judgeWait error
	var userWait error
	for waits := 0; waits < 2; waits++ {
		select {
		case <-ctx.Done():
			killProcessGroup(judge)
			killProcessGroup(user)
			judgeWait, userWait = collectWaitResults(waitCh, judgeWait, userWait)
			if report, err := readReport(resultPath); err == nil && report.Verdict != VerdictAccepted {
				return applyStats(caseResultFromReport(req, report, startedAt)), nil
			}
			return applyStats(CaseResult{CaseID: req.Case.ID, Verdict: VerdictTimeLimit, TimeMS: elapsedMS(startedAt), Message: ctx.Err().Error()}), nil
		case got := <-waitCh:
			if got.name == "judge" {
				judgeWait = got.err
			} else {
				userWait = got.err
			}
		}
	}

	report, err := readReport(resultPath)
	if err != nil {
		if userWait != nil {
			return applyStats(CaseResult{
				CaseID:  req.Case.ID,
				Verdict: VerdictRuntimeError,
				Message: compactErrors("user program failed", userWait, nil, "", userErr.String()),
			}), nil
		}
		return applyStats(CaseResult{
			CaseID:  req.Case.ID,
			Verdict: VerdictSystemError,
			Message: compactErrors("missing judge report", judgeWait, userWait, judgeErr.String(), userErr.String()),
		}), nil
	}
	result := caseResultFromReport(req, report, startedAt)
	if result.Score == 100 && req.Case.Score > 0 {
		result.Score = req.Case.Score
	}
	if userWait != nil && result.Verdict != VerdictOutputLimit && !isBrokenPipeExit(userWait) {
		result.Verdict = VerdictRuntimeError
		result.Score = 0
		result.Message = compactErrors("user program failed", userWait, nil, "", userErr.String())
	}
	if judgeWait != nil && result.Verdict == VerdictAccepted {
		result.Verdict = VerdictSystemError
		result.Score = 0
		result.Message = compactErrors("judge program failed", judgeWait, nil, judgeErr.String(), "")
	}
	return applyStats(result), nil
}

func collectWaitResults(waitCh <-chan waitResult, judgeWait error, userWait error) (error, error) {
	for index := 0; index < 2; index++ {
		select {
		case got := <-waitCh:
			if got.name == "judge" {
				judgeWait = got.err
			} else {
				userWait = got.err
			}
		case <-time.After(100 * time.Millisecond):
			return judgeWait, userWait
		}
	}
	return judgeWait, userWait
}

func isBrokenPipeExit(err error) bool {
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return errors.Is(err, syscall.EPIPE)
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	return ok && status.Signaled() && status.Signal() == syscall.SIGPIPE
}

func caseResultFromReport(req LocalRun, report JudgeReport, startedAt time.Time) CaseResult {
	result := CaseResult{
		CaseID:  req.Case.ID,
		Verdict: report.Verdict,
		Score:   report.Score,
		TimeMS:  elapsedMS(startedAt),
		Message: report.Message,
	}
	if result.Score == 100 && req.Case.Score > 0 {
		result.Score = req.Case.Score
	}
	return result
}

func elapsedMS(startedAt time.Time) int {
	elapsed := int(time.Since(startedAt).Milliseconds())
	if elapsed <= 0 {
		return 1
	}
	return elapsed
}

func prepareUserWorkIdentity(root string, identity ProcessIdentity) error {
	if root == "" || !identity.Enabled {
		return nil
	}
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		mode := info.Mode().Perm()
		if entry.IsDir() {
			mode |= 0o700
		} else if info.Mode().IsRegular() {
			mode |= 0o400
			if mode&0o111 != 0 {
				mode |= 0o100
			}
		}
		if err := os.Chmod(path, mode); err != nil {
			return err
		}
		return os.Chown(path, int(identity.UID), int(identity.GID))
	})
}

func waitForReleaseCommand(path string, command string) string {
	return "while [ ! -e " + shellQuote(path) + " ]; do sleep 0.01; done\nexec " + command
}

func judgeCommand(ctx context.Context, req LocalRun, resultPath string, outputLimit int64) *exec.Cmd {
	if req.JudgeCommand != "" {
		cmd := shellCommand(ctx, req.JudgeCommand)
		cmd.Env = append(os.Environ(),
			"INPUT="+req.Case.Input,
			"ANSWER="+req.Case.Answer,
			"RESULT="+resultPath,
			"OUTPUT_LIMIT="+strconv.FormatInt(outputLimit, 10),
		)
		return cmd
	}
	return exec.CommandContext(
		ctx,
		req.Runner,
		"judge",
		"--mode", string(req.Mode),
		"--input", req.Case.Input,
		"--answer", req.Case.Answer,
		"--result", resultPath,
		"--output-limit", strconv.FormatInt(outputLimit, 10),
	)
}

func readReport(path string) (JudgeReport, error) {
	file, err := os.Open(path)
	if err != nil {
		return JudgeReport{}, err
	}
	defer file.Close()
	var report JudgeReport
	return report, json.NewDecoder(file).Decode(&report)
}

func closePipeFiles(files ...*os.File) {
	for _, file := range files {
		if file != nil {
			_ = file.Close()
		}
	}
}

func safeCaseID(id string) string {
	if id == "" {
		return "case"
	}
	var out []rune
	for _, r := range id {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "case"
	}
	return string(out)
}

func compactErrors(prefix string, errA error, errB error, stderrA string, stderrB string) string {
	var parts []string
	if prefix != "" {
		parts = append(parts, prefix)
	}
	if errA != nil {
		parts = append(parts, errA.Error())
	}
	if errB != nil {
		parts = append(parts, errB.Error())
	}
	if stderrA != "" {
		parts = append(parts, stderrA)
	}
	if stderrB != "" {
		parts = append(parts, stderrB)
	}
	return stringsJoin(parts, ": ")
}

func stringsJoin(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, part := range parts[1:] {
		out += sep + part
	}
	return out
}
