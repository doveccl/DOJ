package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompileUserProgramEnforcesWorkspaceLimit(t *testing.T) {
	work := t.TempDir()
	source := filepath.Join(work, languageSourceDir)
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatal(err)
	}
	script := "dd if=/dev/zero of=blob bs=1048576 count=1 status=none\nwhile :; do :; done\n"
	if err := os.WriteFile(filepath.Join(source, "fill.sh"), []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := compileUserProgram(ctx, work, "", ProcessIdentity{}, CompileRequest{
		CompileCommand: "sh fill.sh",
		UserCommand:    "./main",
		Limits:         Limits{TimeMS: 1000, FileKB: 4},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.OK || result.Message != "compile file limit exceeded" {
		t.Fatalf("compile result = %+v", result)
	}
}

func TestCompileTimeoutUsesLanguageLimit(t *testing.T) {
	if got := compileTimeout(0); got != 10*time.Second {
		t.Fatalf("default timeout = %v", got)
	}
	if got := compileTimeout(12_345); got != 12345*time.Millisecond {
		t.Fatalf("language timeout = %v", got)
	}
	if got := compileTimeout(60_000); got != 30*time.Second {
		t.Fatalf("capped timeout = %v", got)
	}
}
