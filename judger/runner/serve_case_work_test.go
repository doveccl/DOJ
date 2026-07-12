package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakeCaseWorkUsesWorkParent(t *testing.T) {
	work := filepath.Join(t.TempDir(), "work")
	got, err := makeCaseWork(work, "1")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(got) != filepath.Join(work, ".cases") || !strings.HasPrefix(filepath.Base(got), "doj-case-1-") {
		t.Fatalf("case work = %q, want under %q", got, filepath.Join(work, ".cases"))
	}
	info, err := os.Stat(filepath.Join(work, ".cases"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf(".cases mode = %v, want 0700", info.Mode().Perm())
	}
}
