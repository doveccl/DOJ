//go:build linux

package runner

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
	judge := filepath.Join(work, "judge.sh")
	writeScript(t, work, "judge.sh", `#!/bin/sh
cat "$1"
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
		JudgeCommand: judge,
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

func TestRunLocalCaseCustomDefaultTemplateEOF(t *testing.T) {
	runner := buildRunner(t)
	work := t.TempDir()
	input, answer := writeCase(t, work, "custom", "1\n2\n", "1\n4\n")
	writeScript(t, work, "user.sh", "while read n; do echo $((n * n)); done\n")
	judge := filepath.Join(work, "judge.cc")
	if err := os.WriteFile(judge, []byte(`#include <bits/stdc++.h>
using namespace std;

string read_all(istream& in) { return string(istreambuf_iterator<char>(in), {}); }
string read_file(const char* p) { ifstream f(p, ios::binary); return read_all(f); }
void trim_right(string& s) { while (!s.empty() && isspace((unsigned char)s.back())) s.pop_back(); }

int main(int argc, char** argv) {
  if (argc != 5) return 3;
  thread feeder([&] {
    cout << ifstream(argv[1], ios::binary).rdbuf() << flush;
    fclose(stdout);
  });
  string got = read_all(cin);
  feeder.join();

  string ans = read_file(argv[3]);
  trim_right(got);
  trim_right(ans);
  if (got != ans) {
    ofstream(argv[4]) << "expected output differs";
    return 1;
  }
  return 0;
}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(work, "judge")
	if output, err := commandContext(context.Background(), "g++", judge, "-o", out).CombinedOutput(); err != nil {
		t.Fatalf("compile judge: %v\n%s", err, output)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := RunLocalCase(ctx, LocalRun{
		Runner:       runner,
		Work:         work,
		UserCommand:  "sh user.sh",
		JudgeCommand: out,
		Case:         Case{ID: "custom", Input: input, Answer: answer, Score: 100},
		Limits:       Limits{TimeMS: 1000, OutputKB: 64},
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
	writeScript(t, work, "interactive.sh", "while read n; do echo $((n * n)); done\n")
	judge := filepath.Join(work, "judge.sh")
	writeScript(t, work, "judge.sh", `#!/bin/sh
printf '3\n'
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
		JudgeCommand: judge,
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
	judge := filepath.Join(work, "judge.sh")
	writeScript(t, work, "judge.sh", `#!/bin/sh
cat "$1"
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
		JudgeCommand: judge,
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
