package judger

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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

func buildRunner(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "doj-runner")
	cmd := exec.Command("go", "build", "-tags", "runner", "-o", path, "../cmd")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build runner: %v\n%s", err, out)
	}
	return path
}
