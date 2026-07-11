//go:build linux

package judger

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

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
if [ "$(id -u)" = 0 ]; then
  printf 'judge ran as root' > "$4"
  exit 3
fi
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

func TestRunContainerTaskCustomJudgeDefaultTemplateEOF(t *testing.T) {
	requireDocker(t)
	runner := buildRunner(t)
	work := t.TempDir()
	writeCase(t, work, "0", "1\n2\n", "1\n4\n")
	judgeDir := filepath.Join(work, "judge")
	if err := os.MkdirAll(judgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	dockerfile := "FROM gcc\nWORKDIR /src\nCOPY main.cc .\nRUN g++ main.cc -o main\nCMD [\"/src/main\"]\n"
	if err := os.WriteFile(filepath.Join(judgeDir, "Dockerfile"), []byte(dockerfile), 0o600); err != nil {
		t.Fatal(err)
	}
	judge := `#include <bits/stdc++.h>
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
`
	if err := os.WriteFile(filepath.Join(judgeDir, "main.cc"), []byte(judge), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	result, err := RunContainerTask(ctx, ContainerTask{
		Runner: runner,
		Work:   work,
		Task: Task{
			SubmissionID: 204,
			Attempt:      1,
			Source: `#include <bits/stdc++.h>
using namespace std;
int main() {
  int x;
  while (cin >> x) cout << x * x << endl;
  return 0;
}
`,
			Lang: Lang{
				ID:      "cpp",
				Source:  "main.cc",
				Image:   "gcc",
				Compile: "g++ main.cc -o main",
				Run:     "./main",
			},
			Mode:   ModeCustom,
			Limits: Limits{TimeMS: 1000, MemoryKB: 64 << 10, OutputKB: 64, Pids: 32},
			Cases:  []Case{{ID: "0", Input: "0.in", Answer: "0.out", Score: 100}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictAccepted || result.Score != 100 || len(result.Cases) != 1 {
		t.Fatalf("result = %#v", result)
	}
}
