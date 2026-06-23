//go:build linux

package judger

import (
	"context"
	"os"
	"os/exec"
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
	matches, err := filepath.Glob(filepath.Join(work, "lang-build-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("language build context was not cleaned: %v", matches)
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
	if entries, _ := os.ReadDir(filepath.Join(root, "sub-81-attempt-2")); len(entries) != 0 {
		t.Fatalf("cgroup submission directory was not cleaned: %s", containerEntryNames(entries))
	}
}

func TestRunContainerTaskCustomInteractorFromAsset(t *testing.T) {
	requireDocker(t)
	runner := buildRunner(t)
	work := t.TempDir()
	writeCase(t, work, "interactive", "", "")
	judgeDir := filepath.Join(work, "judge")
	if err := os.MkdirAll(judgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	interactor := `#!/bin/sh
printf '3\n'
read a
printf '4\n'
read b
if [ "$a" = 9 ] && [ "$b" = 16 ]; then
  printf '{"verdict":"AC","score":100}\n' > "$RESULT"
else
  printf '{"verdict":"WA","score":0,"message":"bad interaction"}\n' > "$RESULT"
fi
`
	if err := os.WriteFile(filepath.Join(judgeDir, "interactor"), []byte(interactor), 0o700); err != nil {
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
	dockerfile := "FROM alpine:3.20\nCOPY judge.sh /out/judge\nRUN chmod +x /out/judge\n"
	if err := os.WriteFile(filepath.Join(judgeDir, "Dockerfile"), []byte(dockerfile), 0o600); err != nil {
		t.Fatal(err)
	}
	checker := `#!/bin/sh
cat "$INPUT"
exec 1>&-
got="$(cat)"
ans="$(cat "$ANSWER")"
if [ "$got" = "$ans" ]; then
  printf '{"verdict":"AC","score":100}\n' > "$RESULT"
else
  printf '{"verdict":"WA","score":0,"message":"docker judge rejected"}\n' > "$RESULT"
fi
`
	if err := os.WriteFile(filepath.Join(judgeDir, "judge.sh"), []byte(checker), 0o600); err != nil {
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
		"judge-program":                  "secret judge binary",
		"judge-result-secret-probe.json": `{"verdict":"AC","score":100}`,
	} {
		if err := os.WriteFile(filepath.Join(work, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	source := `#!/bin/sh
read _
bad=""
for p in /jobs/secret.out /jobs/judge-program /jobs/judge-result-secret-probe.json /jobs/runner.sock; do
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

func requireDocker(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker CLI is not available")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "info")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("docker daemon is not available: %v\n%s", err, out)
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
		ID:     "sh",
		Source: "main.sh",
		Dockerfile: `FROM alpine:3.20
WORKDIR /src
COPY main.sh main.sh
CMD ["sh", "/src/main.sh"]
`,
	}
}
