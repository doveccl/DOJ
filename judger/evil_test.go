package judger

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunLocalCaseEvilPrograms(t *testing.T) {
	runner := buildRunner(t)
	tests := []struct {
		name    string
		input   string
		answer  string
		command string
		mode    JudgeMode
		limits  Limits
		want    Verdict
	}{
		{
			name:    "output-flood",
			command: "cat >/dev/null; yes x | head -c 4096",
			limits:  Limits{TimeMS: 1000, OutputKB: 1},
			want:    VerdictOutputLimit,
		},
		{
			name:    "runtime-error",
			command: "cat >/dev/null; exit 7",
			limits:  Limits{TimeMS: 1000, OutputKB: 64},
			want:    VerdictRuntimeError,
		},
		{
			name:    "strict-presentation-error",
			answer:  "hello\n",
			command: "cat >/dev/null; printf 'hello  \\n\\n'",
			mode:    ModeStrict,
			limits:  Limits{TimeMS: 1000, OutputKB: 64},
			want:    VerdictPresentationError,
		},
		{
			name:    "quine-like-echo",
			input:   "print(open(__file__).read())\n",
			answer:  "expected output\n",
			command: "cat",
			limits:  Limits{TimeMS: 1000, OutputKB: 64},
			want:    VerdictWrongAnswer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			work := t.TempDir()
			input := filepath.Join(work, "1.in")
			answer := filepath.Join(work, "1.out")
			if err := os.WriteFile(input, []byte(tt.input), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(answer, []byte(tt.answer), 0o600); err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			result, err := RunLocalCase(ctx, LocalRun{
				Runner:      runner,
				Work:        work,
				UserCommand: tt.command,
				Mode:        tt.mode,
				Case:        Case{ID: "1", Input: input, Answer: answer, Score: 100},
				Limits:      tt.limits,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Verdict != tt.want {
				t.Fatalf("verdict = %s, want %s; result = %#v", result.Verdict, tt.want, result)
			}
		})
	}
}

func TestRunLocalCaseTimeout(t *testing.T) {
	runner := buildRunner(t)
	work := t.TempDir()
	input := filepath.Join(work, "1.in")
	answer := filepath.Join(work, "1.out")
	if err := os.WriteFile(input, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(answer, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
	defer cancel()
	result, err := RunLocalCase(ctx, LocalRun{
		Runner:      runner,
		Work:        work,
		UserCommand: "sleep 2",
		Mode:        ModeDefault,
		Case:        Case{ID: "1", Input: input, Answer: answer, Score: 100},
		Limits:      Limits{OutputKB: 64},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictTimeLimit {
		t.Fatalf("result = %#v", result)
	}
}
