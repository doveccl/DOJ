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
