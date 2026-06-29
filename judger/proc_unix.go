//go:build linux

package judger

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func commandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, arg...)
}

func execProgram(name string, args []string) error {
	if !strings.ContainsRune(name, '/') {
		path, err := exec.LookPath(name)
		if err != nil {
			return err
		}
		name = path
	}
	return syscall.Exec(name, append([]string{name}, args...), os.Environ())
}

func dropIdentity(identity ProcessIdentity) error {
	if !identity.Enabled {
		return nil
	}
	if err := syscall.Setgid(int(identity.GID)); err != nil {
		return err
	}
	return syscall.Setuid(int(identity.UID))
}

func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func applyProcessIdentity(cmd *exec.Cmd, identity ProcessIdentity) {
	if !identity.Enabled {
		return
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	cmd.SysProcAttr.Credential = &syscall.Credential{
		Uid:         identity.UID,
		Gid:         identity.GID,
		NoSetGroups: true,
	}
}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
