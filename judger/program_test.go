//go:build linux

package judger

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunLocalCaseCustomInteractorCanEmulateOutputCheck(t *testing.T) {
	runner := buildRunner(t)
	work := t.TempDir()
	input, answer := writeCase(t, work, "custom", "5\n", "25")
	writeScript(t, work, "square.sh", "read n; echo $((n * n))\n")
	writeScript(t, work, "judge.sh", `cat "$1"
printf 'input=%s transcript=%s answer=%s result=%s\n' "$1" "$2" "$3" "$4" > "$2"
exec 1>&-
got="$(cat)"
ans="$(cat "$3")"
if [ "$got" = "$ans" ]; then
  exit 0
else
  printf 'custom judge rejected' > "$4"
  exit 1
fi
`)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := RunLocalCase(ctx, LocalRun{
		Runner:       runner,
		Work:         work,
		UserCommand:  "sh square.sh",
		JudgeCommand: "sh judge.sh",
		Case:         Case{ID: "custom", Input: input, Answer: answer, Score: 100},
		Limits:       Limits{TimeMS: 3000, OutputKB: 64},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictAccepted || result.Score != 100 {
		t.Fatalf("result = %#v", result)
	}
	transcript, err := os.ReadFile(filepath.Join(work, "judge-transcript-custom.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(transcript), "transcript="+filepath.Join(work, "judge-transcript-custom.txt")) ||
		!strings.Contains(string(transcript), "result="+filepath.Join(work, "judge-result-custom.txt")) {
		t.Fatalf("transcript did not record testlib interactor args: %q", transcript)
	}
}

func TestRunLocalCaseInteractor(t *testing.T) {
	runner := buildRunner(t)
	work := t.TempDir()
	input, answer := writeCase(t, work, "interactive", "", "")
	writeScript(t, work, "interactive.sh", "while read n; do echo $((n * n)); done\n")
	writeScript(t, work, "judge.sh", `printf '3\n'
read a
printf '4\n'
read b
if [ "$a" = 9 ] && [ "$b" = 16 ]; then
  exit 0
else
  printf 'bad interaction' > "$4"
  exit 1
fi
`)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := RunLocalCase(ctx, LocalRun{
		Runner:       runner,
		Work:         work,
		UserCommand:  "sh interactive.sh",
		JudgeCommand: "sh judge.sh",
		Case:         Case{ID: "interactive", Input: input, Answer: answer, Score: 100},
		Limits:       Limits{TimeMS: 3000, OutputKB: 64},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictAccepted || result.Score != 100 {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunLocalCaseQuineRejectedByInteractor(t *testing.T) {
	runner := buildRunner(t)
	work := t.TempDir()
	input, answer := writeCase(t, work, "quine", "quine payload\n", "expected semantic output")
	writeScript(t, work, "judge.sh", `cat "$1"
exec 1>&-
got="$(cat)"
ans="$(cat "$3")"
if [ "$got" = "$ans" ]; then
  exit 0
else
  printf 'quine echo is not accepted' > "$4"
  exit 1
fi
`)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := RunLocalCase(ctx, LocalRun{
		Runner:       runner,
		Work:         work,
		UserCommand:  "cat",
		JudgeCommand: "sh judge.sh",
		Case:         Case{ID: "quine", Input: input, Answer: answer, Score: 100},
		Limits:       Limits{TimeMS: 3000, OutputKB: 64},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictWrongAnswer || result.Score != 0 {
		t.Fatalf("result = %#v", result)
	}
}

func writeCase(t *testing.T, work string, id string, input string, answer string) (string, string) {
	t.Helper()
	inputPath := filepath.Join(work, id+".in")
	answerPath := filepath.Join(work, id+".out")
	if err := os.WriteFile(inputPath, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(answerPath, []byte(answer), 0o600); err != nil {
		t.Fatal(err)
	}
	return inputPath, answerPath
}
