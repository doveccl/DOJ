package judger

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestJudgerCLIRejectsRuntimeFlags(t *testing.T) {
	if code := JudgerCLI(context.Background(), []string{"serve"}); code != 2 {
		t.Fatalf("JudgerCLI serve code = %d, want 2", code)
	}
	if code := JudgerCLI(context.Background(), []string{"--server", "http://example.test"}); code != 2 {
		t.Fatalf("JudgerCLI flag code = %d, want 2", code)
	}
}

func TestInstallRunnerFileReplacesAtomically(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "new-doj")
	target := filepath.Join(root, "bin", "doj")
	if err := os.WriteFile(src, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := installRunnerFile(src, target); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "new" {
		t.Fatalf("installed runner = %q, %v", data, err)
	}
	info, err := os.Stat(target)
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("installed runner mode = %v, %v", info, err)
	}
}

func TestJudgerCLIVersion(t *testing.T) {
	if code := JudgerCLI(context.Background(), []string{"version"}); code != 0 {
		t.Fatalf("JudgerCLI version code = %d, want 0", code)
	}
}
