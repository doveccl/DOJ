package judger

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
	if req.JudgeCommand == "" {
		return runBuiltinLocalCase(ctx, req)
	}
	return runCustomLocalCase(ctx, req)
}

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

	judge := customJudgeCommand(ctx, req, transcriptPath, resultPath)
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

	var judgeErr bytes.Buffer
	var userErr bytes.Buffer
	userOutput := &limitFileWriter{file: userToJudgeW, limit: outputLimit}
	judge.Stdin = userToJudgeR
	judge.Stdout = judgeToUserW
	judge.Stderr = &judgeErr
	user.Stdin = judgeToUserR
	user.Stdout = userOutput
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

func customJudgeCommand(ctx context.Context, req LocalRun, transcriptPath string, resultPath string) *exec.Cmd {
	return commandContext(ctx, req.JudgeCommand, req.Case.Input, transcriptPath, req.Case.Answer, resultPath)
}

func touchPrivateFile(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	return file.Close()
}

func caseResultFromTestlib(req LocalRun, exitCode int, resultPath string, timeMS int) CaseResult {
	message := readResultMessage(resultPath)
	result := CaseResult{CaseID: req.Case.ID, TimeMS: timeMS, Message: message}
	switch exitCode {
	case 0:
		result.Verdict = VerdictAccepted
		result.Score = fullCaseScore(req)
	case 1:
		result.Verdict = VerdictWrongAnswer
	case 2, 8:
		result.Verdict = VerdictPresentationError
	case 7:
		result.Verdict = VerdictWrongAnswer
		if score, ok := parseTestlibPoints(message); ok {
			result.Score = clamp(score, 0, fullCaseScore(req))
			if result.Score == fullCaseScore(req) {
				result.Verdict = VerdictAccepted
			}
		}
	default:
		if score, ok := parseTestlibPoints(message); ok {
			result.Verdict = VerdictWrongAnswer
			result.Score = clamp(score, 0, fullCaseScore(req))
			if result.Score == fullCaseScore(req) {
				result.Verdict = VerdictAccepted
			}
			return result
		}
		result.Verdict = VerdictSystemError
		if result.Message == "" {
			result.Message = fmt.Sprintf("judge exited with code %d", exitCode)
		}
	}
	return result
}

func testlibExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return 3
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || status.Signaled() {
		return 3
	}
	return status.ExitStatus()
}

func readResultMessage(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, defaultCompileOutputLimit))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func parseTestlibPoints(message string) (int, bool) {
	var points float64
	if _, err := fmt.Sscanf(message, "%f", &points); err != nil {
		return 0, false
	}
	return int(math.Round(points)), true
}

func fullCaseScore(req LocalRun) int {
	if req.Case.Score > 0 {
		return req.Case.Score
	}
	return 100
}

func clamp(value int, low int, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func closePipeFiles(files ...*os.File) {
	for _, file := range files {
		if file != nil {
			_ = file.Close()
		}
	}
}

type limitFileWriter struct {
	file     *os.File
	limit    int64
	written  int64
	overflow bool
}

func (w *limitFileWriter) Write(p []byte) (int, error) {
	if w.limit <= 0 {
		n, err := w.file.Write(p)
		w.written += int64(n)
		return n, err
	}
	remaining := w.limit - w.written
	if remaining <= 0 {
		w.overflow = true
		return 0, errOutputLimit
	}
	if int64(len(p)) > remaining {
		n, err := w.file.Write(p[:remaining])
		w.written += int64(n)
		w.overflow = true
		if err != nil {
			return n, err
		}
		return n, errOutputLimit
	}
	n, err := w.file.Write(p)
	w.written += int64(n)
	return n, err
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

func userFailureMessage(err error, stderr string) string {
	if message := strings.TrimSpace(stderr); message != "" {
		return message
	}
	if err != nil {
		return err.Error()
	}
	return ""
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
