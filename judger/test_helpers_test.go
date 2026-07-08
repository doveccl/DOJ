package judger

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func buildRunner(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "doj")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", path, "..")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build doj: %v\n%s", err, out)
	}
	return path
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

func testCgroupRoot(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("cgroup integration test")
	}
	name := strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
	root := filepath.Join("/sys/fs/cgroup", "doj-test-"+name+"-"+strconv.Itoa(os.Getpid()))
	_ = os.RemoveAll(root)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Skipf("writable cgroup v2 test root is required: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(root)
	})
	return root
}
