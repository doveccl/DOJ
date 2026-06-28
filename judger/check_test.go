package judger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompareDefaultIgnoresTrailingSpaceAndBlankLines(t *testing.T) {
	verdict, message := Compare(ModeDefault, []byte("1 2  \n3\n\n"), []byte("1 2\n3   \n"))
	if verdict != VerdictAccepted {
		t.Fatalf("verdict = %s, message = %q", verdict, message)
	}
}

func TestCompareStrictReportsPresentationError(t *testing.T) {
	verdict, message := Compare(ModeStrict, []byte("1 2\n"), []byte("1 2  \n\n"))
	if verdict != VerdictPresentationError || message != "" {
		t.Fatalf("verdict = %s, message = %q", verdict, message)
	}
}

func TestCompareWrongAnswer(t *testing.T) {
	verdict, message := Compare(ModeDefault, []byte("42\n"), []byte("43\n"))
	if verdict != VerdictWrongAnswer || message != "" {
		t.Fatalf("verdict = %s, message = %q", verdict, message)
	}
}

func TestBuiltinJudgeMainUsesTestlibCheckerArgs(t *testing.T) {
	work := t.TempDir()
	input := filepath.Join(work, "1.in")
	output := filepath.Join(work, "1.out")
	answer := filepath.Join(work, "1.ans")
	result := filepath.Join(work, "result.txt")
	for path, body := range map[string]string{
		input:  "ignored\n",
		output: "hello  \n\n",
		answer: "hello\n",
	} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if code := BuiltinJudgeMain([]string{"--mode=strict", input, output, answer, result}); code != 2 {
		t.Fatalf("exit code = %d", code)
	}
	if message, err := os.ReadFile(result); err != nil || string(message) != "" {
		t.Fatalf("result file = %q, %v", message, err)
	}
}
