package judger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTokenPrefersFlag(t *testing.T) {
	file := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(file, []byte("file-token\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}
	token, err := resolveToken(" flag-token ", file)
	if err != nil {
		t.Fatalf("resolveToken: %v", err)
	}
	if token != "flag-token" {
		t.Fatalf("token = %q, want flag token", token)
	}
}

func TestResolveTokenReadsFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(file, []byte(" file-token \n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}
	token, err := resolveToken("", file)
	if err != nil {
		t.Fatalf("resolveToken: %v", err)
	}
	if token != "file-token" {
		t.Fatalf("token = %q, want file token", token)
	}
}
