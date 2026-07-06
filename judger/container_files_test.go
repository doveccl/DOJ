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

func TestRunContainerTaskUserCannotReadJobArtifacts(t *testing.T) {
	requireDocker(t)
	runner := buildRunner(t)
	work := t.TempDir()
	writeCase(t, work, "secret", "probe\n", "safe\n")
	for name, content := range map[string]string{
		"judge-program":                 "secret judge binary",
		"judge-result-secret-probe.txt": "accepted",
	} {
		if err := os.WriteFile(filepath.Join(work, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	source := `#!/bin/sh
read _
bad=""
for p in /work/secret.out /work/judge-program /work/judge-result-secret-probe.txt /work/runner.sock; do
  if [ -r "$p" ]; then
    if [ -S "$p" ] || cat "$p" >/dev/null 2>&1; then
      bad="$p"
      break
    fi
  fi
done
if [ -n "$bad" ]; then
  echo leak
else
  echo safe
fi
`

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	result, err := RunContainerTask(ctx, ContainerTask{
		Runner: runner,
		Work:   work,
		Task: Task{
			SubmissionID: 84,
			Attempt:      1,
			Source:       source,
			Lang:         testShellLang(),
			Mode:         ModeDefault,
			Limits:       Limits{TimeMS: 3000, OutputKB: 64, MemoryKB: 64 << 10, Pids: 32},
			Cases: []Case{
				{ID: "secret", Input: "secret.in", Answer: "secret.out", Score: 100},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictAccepted || result.Score != 100 || len(result.Cases) != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunContainerTaskCompileCannotReadAnswers(t *testing.T) {
	requireDocker(t)
	runner := buildRunner(t)
	work := t.TempDir()
	writeCase(t, work, "secret", "probe\n", "safe\n")
	source := `#include "/work/secret.out"
int main(){ return 0; }
`

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	result, err := RunContainerTask(ctx, ContainerTask{
		Runner: runner,
		Work:   work,
		Task: Task{
			SubmissionID: 86,
			Attempt:      1,
			Source:       source,
			Lang: Lang{
				ID:      "cpp",
				Source:  "main.cc",
				Image:   "gcc:14",
				Compile: "g++ -std=c++20 -O2 -pipe -static -s main.cc -o main",
				Run:     "./main",
			},
			Mode:   ModeDefault,
			Limits: Limits{TimeMS: 1000, MemoryKB: 256 * 1024, OutputKB: 64, Pids: 64},
			Cases:  []Case{{ID: "secret", Input: "secret.in", Answer: "secret.out", Score: 100}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictCompileError {
		t.Fatalf("result = %#v", result)
	}
	if result.TimeMS != 0 {
		t.Fatalf("compile error should not expose submission time, got %dms", result.TimeMS)
	}
	if !strings.Contains(result.Message, "/work/secret.out") {
		t.Fatalf("compile error did not mention missing answer include: %q", result.Message)
	}
}

func TestRunContainerTaskUserCannotReadNestedAnswers(t *testing.T) {
	requireDocker(t)
	runner := buildRunner(t)
	work := t.TempDir()
	if err := os.MkdirAll(filepath.Join(work, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "data", "0.in"), []byte("probe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "data", "0.out"), []byte("safe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := `#include <fstream>
#include <iostream>
int main() {
  std::ifstream ans("/work/data/0.out");
  std::cout << (ans.good() ? "leak\n" : "safe\n");
}`

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	result, err := RunContainerTask(ctx, ContainerTask{
		Runner: runner,
		Work:   work,
		Task: Task{
			SubmissionID: 87,
			Attempt:      1,
			Source:       source,
			Lang: Lang{
				ID:      "cpp",
				Source:  "main.cc",
				Image:   "gcc",
				Compile: "g++ main.cc -o main",
				Run:     "./main",
			},
			Mode:   ModeDefault,
			Limits: Limits{TimeMS: 1000, MemoryKB: 256 * 1024, OutputKB: 64, Pids: 64},
			Cases:  []Case{{ID: "0", Input: "data/0.in", Answer: "data/0.out", Score: 100}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictAccepted {
		t.Fatalf("result = %#v", result)
	}
}
