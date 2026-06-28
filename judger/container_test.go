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
		Task: Task{
			SubmissionID: 80,
			Attempt:      1,
			Source:       "#!/bin/sh\nread a b\necho $((a + b))\n",
			Lang:         testShellLang(),
			Mode:         ModeDefault,
			Limits:       Limits{TimeMS: 3000, OutputKB: 64, MemoryKB: 64 << 10, Pids: 32},
			Cases: []Case{
				{ID: "1", Input: "1.in", Answer: "1.out", Score: 40},
				{ID: "2", Input: "2.in", Answer: "2.out", Score: 60},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictAccepted || result.Score != 100 || len(result.Cases) != 2 {
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
		Task: Task{
			SubmissionID: 85,
			Attempt:      1,
			Source:       "#include <bits/stdc++.h>\nusing namespace std;\nint main(){ long long a,b; cin>>a>>b; cout<<a+b<<'\\n'; }\n",
			Lang: Lang{
				ID:      "cpp",
				Source:  "main.cc",
				Image:   "gcc:14",
				Compile: "g++ -std=c++20 -O2 -pipe -static -s main.cc -o main",
				Run:     "./main",
			},
			Mode:   ModeDefault,
			Limits: Limits{TimeMS: 1000, MemoryKB: 256 * 1024, OutputKB: 64, Pids: 64},
			Cases:  []Case{{ID: "1", Input: "1.in", Answer: "1.out", Score: 100}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictWrongAnswer || len(result.Cases) != 1 || result.Cases[0].Verdict != VerdictWrongAnswer {
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
		Task: Task{
			SubmissionID: 81,
			Attempt:      2,
			Source:       "#!/bin/sh\nread ignored\necho ok\n",
			Lang:         testShellLang(),
			Mode:         ModeDefault,
			Limits:       Limits{TimeMS: 3000, OutputKB: 64, MemoryKB: 64 << 10, Pids: 32},
			Cases: []Case{
				{ID: "1", Input: "1.in", Answer: "1.out", Score: 100},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictAccepted || result.Score != 100 || len(result.Cases) != 1 {
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
		Task: Task{
			SubmissionID: 88,
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
			Limits: Limits{TimeMS: 3000, MemoryKB: 16 * 1024, OutputKB: 64, Pids: 64},
			Cases:  []Case{{ID: "1", Input: "1.in", Answer: "1.out", Score: 100}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictMemoryLimit || len(result.Cases) != 1 || result.Cases[0].Verdict != VerdictMemoryLimit {
		t.Fatalf("result = %#v", result)
	}
	if result.MemoryKB <= 0 {
		t.Fatalf("missing memory usage: %#v", result)
	}
}

func TestRunContainerTaskCustomInteractorFromDockerfileCMD(t *testing.T) {
	requireDocker(t)
	runner := buildRunner(t)
	work := t.TempDir()
	writeCase(t, work, "interactive", "", "")
	judgeDir := filepath.Join(work, "judge")
	if err := os.MkdirAll(judgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dockerfile := "FROM alpine:3.20\nCOPY my-tool /usr/local/bin/my-tool\nRUN chmod +x /usr/local/bin/my-tool\nCMD [\"/usr/local/bin/my-tool\"]\n"
	if err := os.WriteFile(filepath.Join(judgeDir, "Dockerfile"), []byte(dockerfile), 0o600); err != nil {
		t.Fatal(err)
	}
	interactor := `#!/bin/sh
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
`
	if err := os.WriteFile(filepath.Join(judgeDir, "my-tool"), []byte(interactor), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	result, err := RunContainerTask(ctx, ContainerTask{
		Runner: runner,
		Work:   work,
		Task: Task{
			SubmissionID: 82,
			Attempt:      1,
			Source:       "#!/bin/sh\nwhile read n; do echo $((n * n)); done\n",
			Lang:         testShellLang(),
			Mode:         ModeCustom,
			Limits:       Limits{TimeMS: 3000, OutputKB: 64, MemoryKB: 64 << 10, Pids: 32},
			Cases: []Case{
				{ID: "interactive", Input: "interactive.in", Answer: "interactive.out", Score: 100},
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

func TestRunContainerTaskCustomJudgeFromDockerfile(t *testing.T) {
	requireDocker(t)
	runner := buildRunner(t)
	work := t.TempDir()
	writeCase(t, work, "docker", "5\n", "25")
	judgeDir := filepath.Join(work, "judge")
	if err := os.MkdirAll(judgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dockerfile := "FROM alpine:3.20\nCOPY judge.sh /opt/judge\nRUN chmod +x /opt/judge\nCMD /opt/judge\n"
	if err := os.WriteFile(filepath.Join(judgeDir, "Dockerfile"), []byte(dockerfile), 0o600); err != nil {
		t.Fatal(err)
	}
	interactor := `#!/bin/sh
cat "$1"
exec 1>&-
got="$(cat)"
ans="$(cat "$3")"
if [ "$got" = "$ans" ]; then
  exit 0
else
  printf 'docker judge rejected' > "$4"
  exit 1
fi
`
	if err := os.WriteFile(filepath.Join(judgeDir, "judge.sh"), []byte(interactor), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	result, err := RunContainerTask(ctx, ContainerTask{
		Runner: runner,
		Work:   work,
		Task: Task{
			SubmissionID: 83,
			Attempt:      1,
			Source:       "#!/bin/sh\nread n\necho $((n * n))\n",
			Lang:         testShellLang(),
			Mode:         ModeCustom,
			Limits:       Limits{TimeMS: 60000, OutputKB: 128, MemoryKB: 64 << 10, Pids: 32},
			Cases: []Case{
				{ID: "docker", Input: "docker.in", Answer: "docker.out", Score: 100},
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

func requireDocker(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := dockerPing(ctx); err != nil {
		t.Skipf("docker engine is not available: %v", err)
	}
}

func containerEntryNames(entries []os.DirEntry) string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return strings.Join(names, ",")
}

func testShellLang() Lang {
	return Lang{
		ID:      "sh",
		Source:  "main.sh",
		Image:   "alpine:3.20",
		Compile: "",
		Run:     "sh main.sh",
	}
}
