//go:build linux

package judger

import (
	"os"
	"testing"
)

func TestMapInnerPIDHostNamespaceSmoke(t *testing.T) {
	pid := os.Getpid()
	got, err := MapInnerPID("/proc", pid, pid)
	if err != nil {
		t.Fatal(err)
	}
	if got != pid {
		t.Fatalf("host pid = %d, want %d", got, pid)
	}
}
