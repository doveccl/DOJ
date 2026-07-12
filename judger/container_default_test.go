//go:build linux

package judger

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	jr "github.com/doveccl/doj/judger/runner"
)

func TestRunContainerTaskNormalMultiCase(t *testing.T) {
	requireDocker(t)
	runner := buildRunner(t)
	work := t.TempDir()
	writeCase(t, work, "1", "1 2\n", "3\n")
	writeCase(t, work, "2", "5 7\n", "12\n")

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	result, err := RunContainerTask(ctx, ContainerTask{
		Runner: runner,
		Work:   work,
		Task: jr.Task{
			SubmissionID: 80,
			Attempt:      1,
			Source:       "#!/bin/sh\nread a b\necho $((a + b))\n",
			Lang:         testShellLang(),
			Mode:         jr.ModeDefault,
			Limits:       jr.Limits{TimeMS: 3000, OutputKB: 64, MemoryKB: 64 << 10, Pids: 32},
			Cases: []jr.Case{
				{ID: "1", Input: "1.in", Answer: "1.out", Score: 40},
				{ID: "2", Input: "2.in", Answer: "2.out", Score: 60},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != jr.VerdictAccepted || result.Score != 100 || len(result.Cases) != 2 {
		t.Fatalf("result = %#v", result)
	}
	maxCaseTime := 0
	for _, item := range result.Cases {
		if item.TimeMS > maxCaseTime {
			maxCaseTime = item.TimeMS
		}
	}
	if result.TimeMS != maxCaseTime {
		t.Fatalf("submission time = %d, want max case time %d: %#v", result.TimeMS, maxCaseTime, result.Cases)
	}
	matches, err := filepath.Glob(filepath.Join(work, "lang-build-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("language build context was not cleaned: %v", matches)
	}
}

func TestRunContainerTaskCppPartialInputReaderIsWrongAnswer(t *testing.T) {
	requireDocker(t)
	runner := buildRunner(t)
	work := t.TempDir()
	writeCase(t, work, "1", "1 2\n"+strings.Repeat("3\n", 1<<15), "not 3\n")

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	result, err := RunContainerTask(ctx, ContainerTask{
		Runner: runner,
		Work:   work,
		Task: jr.Task{
			SubmissionID: 85,
			Attempt:      1,
			Source:       "#include <bits/stdc++.h>\nusing namespace std;\nint main(){ long long a,b; cin>>a>>b; cout<<a+b<<'\\n'; }\n",
			Lang: jr.Lang{
				ID:      "cpp",
				Source:  "main.cc",
				Image:   "gcc:14",
				Compile: "g++ -std=c++20 -O2 -pipe -static -s main.cc -o main",
				Run:     "./main",
			},
			Mode:   jr.ModeDefault,
			Limits: jr.Limits{TimeMS: 1000, MemoryKB: 256 * 1024, OutputKB: 64, Pids: 64},
			Cases:  []jr.Case{{ID: "1", Input: "1.in", Answer: "1.out", Score: 100}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != jr.VerdictWrongAnswer || len(result.Cases) != 1 || result.Cases[0].Verdict != jr.VerdictWrongAnswer {
		t.Fatalf("result = %#v", result)
	}
}

func TestRunContainerTaskCgroupAttach(t *testing.T) {
	requireDocker(t)
	root := testCgroupRoot(t)
	runner := buildRunner(t)
	work := t.TempDir()
	writeCase(t, work, "1", "ready\n", "ok\n")

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	result, err := RunContainerTask(ctx, ContainerTask{
		Runner:     runner,
		Work:       work,
		CgroupRoot: root,
		ProcRoot:   "/proc",
		Task: jr.Task{
			SubmissionID: 81,
			Attempt:      2,
			Source:       "#!/bin/sh\nread ignored\necho ok\n",
			Lang:         testShellLang(),
			Mode:         jr.ModeDefault,
			Limits:       jr.Limits{TimeMS: 3000, OutputKB: 64, MemoryKB: 64 << 10, Pids: 32},
			Cases: []jr.Case{
				{ID: "1", Input: "1.in", Answer: "1.out", Score: 100},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != jr.VerdictAccepted || result.Score != 100 || len(result.Cases) != 1 {
		t.Fatalf("result = %#v", result)
	}
	if result.Cases[0].MemoryKB <= 0 {
		t.Fatalf("missing cgroup memory stats: %#v", result.Cases[0])
	}
	if entries, _ := os.ReadDir(filepath.Join(root, "81-2")); len(entries) != 0 {
		t.Fatalf("cgroup submission directory was not cleaned: %s", containerEntryNames(entries))
	}
}

func TestRunContainerTaskMemoryLimit(t *testing.T) {
	requireDocker(t)
	root := testCgroupRoot(t)
	runner := buildRunner(t)
	work := t.TempDir()
	writeCase(t, work, "1", "go\n", "done\n")
	source := `#include <cstring>
#include <vector>
int main() {
  std::vector<char*> blocks;
  while (true) {
    char* block = new char[1 << 20];
    std::memset(block, 1, 1 << 20);
    blocks.push_back(block);
  }
}`

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	result, err := RunContainerTask(ctx, ContainerTask{
		Runner:     runner,
		Work:       work,
		CgroupRoot: root,
		ProcRoot:   "/proc",
		Task: jr.Task{
			SubmissionID: 88,
			Attempt:      1,
			Source:       source,
			Lang: jr.Lang{
				ID:      "cpp",
				Source:  "main.cc",
				Image:   "gcc",
				Compile: "g++ main.cc -o main",
				Run:     "./main",
			},
			Mode:   jr.ModeDefault,
			Limits: jr.Limits{TimeMS: 3000, MemoryKB: 16 * 1024, OutputKB: 64, Pids: 64},
			Cases:  []jr.Case{{ID: "1", Input: "1.in", Answer: "1.out", Score: 100}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != jr.VerdictMemoryLimit || len(result.Cases) != 1 || result.Cases[0].Verdict != jr.VerdictMemoryLimit {
		t.Fatalf("result = %#v", result)
	}
	if result.MemoryKB <= 0 {
		t.Fatalf("missing memory usage: %#v", result)
	}
}
