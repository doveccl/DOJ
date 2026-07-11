//go:build linux

package runner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
)

func TestDropIdentityClearsSupplementaryGroups(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("root is required to change identity")
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestDropIdentityHelper$")
	cmd.Env = append(os.Environ(), "DOJ_TEST_DROP_IDENTITY=1")
	output, err := cmd.CombinedOutput()
	if strings.HasPrefix(string(output), "SKIP:") {
		t.Skip(strings.TrimSpace(string(output)))
	}
	if err != nil {
		t.Fatalf("drop identity helper: %v: %s", err, output)
	}
}

func TestDropIdentityHelper(t *testing.T) {
	if os.Getenv("DOJ_TEST_DROP_IDENTITY") != "1" {
		return
	}
	if err := syscall.Setgroups([]int{0}); err != nil {
		fmt.Printf("SKIP: cannot set supplementary groups: %v\n", err)
		os.Exit(0)
	}
	if err := dropIdentity(ProcessIdentity{UID: 65534, GID: 65534, Enabled: true}); err != nil {
		fmt.Printf("drop identity: %v\n", err)
		os.Exit(1)
	}
	groups, err := syscall.Getgroups()
	if err != nil || len(groups) != 0 || os.Geteuid() != 65534 || os.Getegid() != 65534 {
		fmt.Printf("identity uid=%d gid=%d groups=%v err=%v\n", os.Geteuid(), os.Getegid(), groups, err)
		os.Exit(1)
	}
	os.Exit(0)
}
