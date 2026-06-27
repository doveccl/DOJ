//go:build linux

package judger

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunLocalCaseAccepted(t *testing.T) {
	runner := buildRunner(t)
	work := t.TempDir()
	input := filepath.Join(work, "1.in")
	answer := filepath.Join(work, "1.out")
	if err := os.WriteFile(input, []byte("42\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(answer, []byte("42\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := RunLocalCase(ctx, LocalRun{
		Runner:      runner,
		Work:        work,
		UserCommand: "cat",
		Mode:        ModeDefault,
		Case:        Case{ID: "1", Input: input, Answer: answer, Score: 100},
		Limits:      Limits{OutputKB: 64},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictAccepted || result.Score != 100 {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunLocalCaseWrongAnswer(t *testing.T) {
	runner := buildRunner(t)
	work := t.TempDir()
	input := filepath.Join(work, "1.in")
	answer := filepath.Join(work, "1.out")
	if err := os.WriteFile(input, []byte("42\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(answer, []byte("42\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := RunLocalCase(ctx, LocalRun{
		Runner:      runner,
		Work:        work,
		UserCommand: "cat >/dev/null; printf '41\\n'",
		Mode:        ModeDefault,
		Case:        Case{ID: "1", Input: input, Answer: answer, Score: 100},
		Limits:      Limits{OutputKB: 64},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictWrongAnswer || result.Score != 0 {
		t.Fatalf("result = %#v", result)
	}
	if result.Message != "" {
		t.Fatalf("wrong answer message = %q", result.Message)
	}
}

func TestRunLocalCaseUserMayExitBeforeReadingAllInput(t *testing.T) {
	runner := buildRunner(t)
	work := t.TempDir()
	input := filepath.Join(work, "1.in")
	answer := filepath.Join(work, "1.out")
	if err := os.WriteFile(input, []byte(strings.Repeat("ignored input\n", 1<<15)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(answer, []byte("42\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := RunLocalCase(ctx, LocalRun{
		Runner:      runner,
		Work:        work,
		UserCommand: "printf '42\\n'",
		Mode:        ModeDefault,
		Case:        Case{ID: "1", Input: input, Answer: answer, Score: 100},
		Limits:      Limits{OutputKB: 64},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictAccepted || result.Score != 100 {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunLocalCasePartialInputReaderIsWrongAnswer(t *testing.T) {
	runner := buildRunner(t)
	work := t.TempDir()
	input := filepath.Join(work, "1.in")
	answer := filepath.Join(work, "1.out")
	if err := os.WriteFile(input, []byte("1 2\n"+strings.Repeat("3\n", 1<<15)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(answer, []byte("not 3\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := RunLocalCase(ctx, LocalRun{
		Runner:      runner,
		Work:        work,
		UserCommand: "read a b; echo $((a + b))",
		Mode:        ModeDefault,
		Case:        Case{ID: "1", Input: input, Answer: answer, Score: 100},
		Limits:      Limits{OutputKB: 64},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictWrongAnswer || result.Score != 0 {
		t.Fatalf("result = %#v", result)
	}
}

func TestApplyCgroupStatsReportsMemoryLimit(t *testing.T) {
	result := CaseResult{CaseID: "1", Verdict: VerdictRuntimeError, Score: 100, Message: "boom"}
	applyCgroupStatsSnapshot(&result, CgroupStats{MemoryPeak: 130*1024*1024 + 1, MemoryOOM: true})
	if result.Verdict != VerdictMemoryLimit || result.Score != 0 || result.Message != "" {
		t.Fatalf("result = %#v", result)
	}
	if result.MemoryKB != 130*1024+1 {
		t.Fatalf("memory KB = %d", result.MemoryKB)
	}
}

func TestApplyCgroupStatsReportsMemoryMax(t *testing.T) {
	result := CaseResult{CaseID: "1", Verdict: VerdictTimeLimit, Score: 0, TimeMS: 1001}
	applyCgroupStatsSnapshot(&result, CgroupStats{MemoryPeak: 64 * 1024 * 1024, MemoryMaxed: true})
	if result.Verdict != VerdictMemoryLimit || result.Score != 0 || result.Message != "" {
		t.Fatalf("result = %#v", result)
	}
	if result.TimeMS != 1001 {
		t.Fatalf("time ms = %d", result.TimeMS)
	}
}

func TestCgroupMemoryLimitReached(t *testing.T) {
	if !cgroupMemoryLimitReached(CgroupStats{MemoryMaxed: true}) {
		t.Fatal("memory.max event was not treated as memory limit")
	}
	if !cgroupMemoryLimitReached(CgroupStats{MemoryOOM: true}) {
		t.Fatal("memory oom event was not treated as memory limit")
	}
	if cgroupMemoryLimitReached(CgroupStats{PidsMaxed: true}) {
		t.Fatal("pids max should not be treated as memory limit")
	}
}

func TestApplyCgroupStatsReportsProcessLimitOnlyForAccepted(t *testing.T) {
	ac := CaseResult{Verdict: VerdictAccepted, Score: 100}
	applyCgroupStatsSnapshot(&ac, CgroupStats{PidsMaxed: true})
	if ac.Verdict != VerdictRuntimeError || ac.Score != 0 || ac.Message != "process limit exceeded" {
		t.Fatalf("accepted result = %#v", ac)
	}
	wa := CaseResult{Verdict: VerdictWrongAnswer, Score: 0}
	applyCgroupStatsSnapshot(&wa, CgroupStats{PidsMaxed: true})
	if wa.Verdict != VerdictWrongAnswer || wa.Message != "" {
		t.Fatalf("wrong answer result = %#v", wa)
	}
}

func TestRunLocalCaseRuntimeErrorKeepsOnlyUsefulMessage(t *testing.T) {
	runner := buildRunner(t)
	work := t.TempDir()
	input := filepath.Join(work, "1.in")
	answer := filepath.Join(work, "1.out")
	if err := os.WriteFile(input, []byte("1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(answer, []byte("1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := RunLocalCase(ctx, LocalRun{
		Runner:      runner,
		Work:        work,
		UserCommand: "echo boom >&2; exit 7",
		Mode:        ModeDefault,
		Case:        Case{ID: "1", Input: input, Answer: answer, Score: 100},
		Limits:      Limits{OutputKB: 64},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictRuntimeError || result.Message != "boom" {
		t.Fatalf("result = %#v", result)
	}
}

func buildRunner(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "doj-runner")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-tags", "runner", "-o", path, "../cmd/runner.go")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build runner: %v\n%s", err, out)
	}
	return path
}
