//go:build linux

package judger

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func requireDocker(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("docker integration test")
	}
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
