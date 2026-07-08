package runner

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

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

func SafeCaseID(id string) string {
	return safeCaseID(id)
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
	return strings.Join(parts, ": ")
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
