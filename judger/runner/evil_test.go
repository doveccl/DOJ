//go:build linux

package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	contractlimits "github.com/doveccl/doj/contract/limits"
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
		message string
	}{
		{
			name:   "output-flood",
			limits: Limits{TimeMS: 1000, OutputKB: 1},
			want:   VerdictOutputLimit,
		},
		{
			name:    "runtime-error",
			command: "sh runtime-error.sh",
			limits:  Limits{TimeMS: 1000, OutputKB: 64},
			want:    VerdictRuntimeError,
			message: "exit status 7",
		},
		{
			name:    "stderr-flood",
			command: "sh stderr-flood.sh",
			limits:  Limits{TimeMS: 3000, OutputKB: 64},
			want:    VerdictRuntimeError,
		},
		{
			name:    "strict-presentation-error",
			answer:  "hello\n",
			command: "sh strict-presentation-error.sh",
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
			switch tt.name {
			case "output-flood":
				writeScript(t, work, "flood.sh", "cat >/dev/null; yes x | head -c 4096\n")
				tt.command = "sh flood.sh"
			case "runtime-error":
				writeScript(t, work, "runtime-error.sh", "cat >/dev/null; exit 7\n")
			case "stderr-flood":
				writeScript(t, work, "stderr-flood.sh", "cat >/dev/null; yes x | head -c 1048576 >&2; exit 7\n")
			case "strict-presentation-error":
				writeScript(t, work, "strict-presentation-error.sh", "cat >/dev/null; printf 'hello  \\n\\n'\n")
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
			if tt.name == "stderr-flood" {
				if len([]rune(result.Message)) != contractlimits.MaxCaseMessageRunes {
					t.Fatalf("stderr message runes = %d, want %d", len([]rune(result.Message)), contractlimits.MaxCaseMessageRunes)
				}
			} else if result.Message != tt.message {
				t.Fatalf("message = %q, want %q; result = %#v", result.Message, tt.message, result)
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
	if result.Message != "" {
		t.Fatalf("timeout message = %q", result.Message)
	}
}
