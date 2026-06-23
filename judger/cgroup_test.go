package judger

import (
	"runtime"
	"testing"
)

func TestPrepareCgroupUnsupportedOutsideLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("linux cgroup test needs a writable cgroup v2 root")
	}
	_, err := PrepareCgroup(CgroupConfig{})
	if err != ErrCgroupUnsupported {
		t.Fatalf("err = %v", err)
	}
}
