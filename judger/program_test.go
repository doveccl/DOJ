package judger

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunLocalCaseCustomChecker(t *testing.T) {
	runner := buildRunner(t)
	work := t.TempDir()
	input, answer := writeCase(t, work, "checker", "5\n", "25")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := RunLocalCase(ctx, LocalRun{
		Runner:       runner,
		Work:         work,
		UserCommand:  "read n; echo $((n * n))",
		JudgeCommand: "cat \"$DOJ_INPUT\"; exec 1>&-; got=$(cat); ans=$(cat \"$DOJ_ANSWER\"); if [ \"$got\" = \"$ans\" ]; then printf '{\"verdict\":\"AC\",\"score\":100}\\n' > \"$DOJ_RESULT\"; else printf '{\"verdict\":\"WA\",\"score\":0,\"message\":\"checker rejected\"}\\n' > \"$DOJ_RESULT\"; fi",
		Case:         Case{ID: "checker", Input: input, Answer: answer, Score: 100},
		Limits:       Limits{TimeMS: 3000, OutputKB: 64},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictAccepted || result.Score != 100 {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunLocalCaseInteractor(t *testing.T) {
	runner := buildRunner(t)
	work := t.TempDir()
	input, answer := writeCase(t, work, "interactive", "", "")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := RunLocalCase(ctx, LocalRun{
		Runner:       runner,
		Work:         work,
		UserCommand:  "while read n; do echo $((n * n)); done",
		JudgeCommand: "printf '3\\n'; read a; printf '4\\n'; read b; if [ \"$a\" = 9 ] && [ \"$b\" = 16 ]; then printf '{\"verdict\":\"AC\",\"score\":100}\\n' > \"$DOJ_RESULT\"; else printf '{\"verdict\":\"WA\",\"score\":0,\"message\":\"bad interaction\"}\\n' > \"$DOJ_RESULT\"; fi",
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

func TestRunLocalCaseQuineRejectedByChecker(t *testing.T) {
	runner := buildRunner(t)
	work := t.TempDir()
	input, answer := writeCase(t, work, "quine", "quine payload\n", "expected semantic output")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := RunLocalCase(ctx, LocalRun{
		Runner:       runner,
		Work:         work,
		UserCommand:  "cat",
		JudgeCommand: "cat \"$DOJ_INPUT\"; exec 1>&-; got=$(cat); ans=$(cat \"$DOJ_ANSWER\"); if [ \"$got\" = \"$ans\" ]; then printf '{\"verdict\":\"AC\",\"score\":100}\\n' > \"$DOJ_RESULT\"; else printf '{\"verdict\":\"WA\",\"score\":0,\"message\":\"quine echo is not accepted\"}\\n' > \"$DOJ_RESULT\"; fi",
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
