//go:build linux

package judger

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCgroupLinuxSmoke(t *testing.T) {
	root := testCgroupRoot(t)
	cg, err := PrepareCgroup(CgroupConfig{
		Root:         root,
		SubmissionID: "smoke",
		CaseID:       "case-1",
		MemoryMax:    64 << 20,
		PidsMax:      16,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := cg.Cleanup(); err != nil && !os.IsNotExist(err) {
			t.Fatalf("cleanup cgroup: %v", err)
		}
		_ = os.Remove(filepath.Join(root, "smoke"))
		_ = os.Remove(root)
	}()

	cmd := exec.Command("sleep", "10")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	if err := cg.Add(cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()

	if _, err := cg.Stats(); err != nil {
		t.Fatal(err)
	}
}

func TestCgroupLinuxMemoryBomb(t *testing.T) {
	root := testCgroupRoot(t)
	cg := prepareTestCgroup(t, root, "memory", "case-1", 32<<20, 32)
	cmd, release := startCgroupHelper(t, cg, "memory")
	release()
	waitOrKill(t, cmd, 5*time.Second)

	stats, err := cg.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if !stats.MemoryOOM && stats.MemoryPeak < 32<<20 {
		t.Fatalf("memory limit did not appear to apply: %+v", stats)
	}
}

func TestCgroupLinuxPidsBomb(t *testing.T) {
	root := testCgroupRoot(t)
	cg := prepareTestCgroup(t, root, "pids", "case-1", 128<<20, 8)
	cmd, release := startCgroupHelper(t, cg, "pids")
	release()
	time.Sleep(500 * time.Millisecond)
	killProcessGroup(cmd)
	_ = cmd.Wait()

	stats, err := cg.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.PidsCurrent > 8 {
		t.Fatalf("pids.current exceeded limit: %+v", stats)
	}
	if !stats.PidsMaxed {
		t.Fatalf("pids.max was not exercised: %+v", stats)
	}
}

func TestCgroupLinuxHelperProcess(t *testing.T) {
	mode := helperMode()
	if mode == "" {
		return
	}
	_, _ = io.CopyN(io.Discard, os.Stdin, 1)
	switch mode {
	case "memory":
		hold := make([][]byte, 0, 512)
		for index := 0; index < 512; index++ {
			block := make([]byte, 1<<20)
			for offset := 0; offset < len(block); offset += 4096 {
				block[offset] = byte(index)
			}
			hold = append(hold, block)
			time.Sleep(5 * time.Millisecond)
		}
	case "pids":
		children := make([]*exec.Cmd, 0, 64)
		defer func() {
			for _, child := range children {
				if child.Process != nil {
					_ = child.Process.Kill()
					_ = child.Wait()
				}
			}
		}()
		for index := 0; index < 64; index++ {
			child := exec.Command("sleep", "30")
			if err := child.Start(); err != nil {
				time.Sleep(30 * time.Second)
				return
			}
			children = append(children, child)
		}
		time.Sleep(30 * time.Second)
	default:
		t.Fatalf("unknown cgroup helper mode %q", mode)
	}
}

func entryNames(entries []os.DirEntry) string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return strings.Join(names, ",")
}

func testCgroupRoot(t *testing.T) string {
	t.Helper()
	root := "/sys/fs/cgroup/doj-test"
	_ = os.RemoveAll(root)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Skipf("writable cgroup v2 test root is required: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(root)
	})
	return root
}

func helperMode() string {
	for index, arg := range os.Args {
		if arg == "--" && index+1 < len(os.Args) {
			return os.Args[index+1]
		}
	}
	return ""
}

func prepareTestCgroup(t *testing.T, root string, submission string, caseID string, memoryMax int64, pidsMax int) *CgroupCase {
	t.Helper()
	cg, err := PrepareCgroup(CgroupConfig{
		Root:         root,
		SubmissionID: submission,
		CaseID:       caseID,
		MemoryMax:    memoryMax,
		PidsMax:      pidsMax,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := cg.Cleanup(); err != nil && !os.IsNotExist(err) {
			t.Fatalf("cleanup cgroup: %v", err)
		}
		_ = os.Remove(filepath.Join(root, submission))
		_ = os.Remove(root)
	})
	return cg
}

func startCgroupHelper(t *testing.T, cg *CgroupCase, mode string) (*exec.Cmd, func()) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestCgroupLinuxHelperProcess", "--", mode)
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if err := cg.Add(cmd.Process.Pid); err != nil {
		killProcessGroup(cmd)
		_ = cmd.Wait()
		t.Fatal(err)
	}
	return cmd, func() {
		_, _ = stdin.Write([]byte{'\n'})
		_ = stdin.Close()
	}
}

func waitOrKill(t *testing.T, cmd *exec.Cmd, timeout time.Duration) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-time.After(timeout):
		killProcessGroup(cmd)
		<-done
	case <-done:
	}
}
