//go:build !linux

package runner

import (
	"context"
	"os"
	"os/exec"
)

func commandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, arg...)
}

func execProgram(name string, args []string) error {
	return exec.Command(name, args...).Run()
}

func dropIdentity(ProcessIdentity) error {
	return nil
}

func configureProcess(*exec.Cmd) {}

func applyProcessIdentity(*exec.Cmd, ProcessIdentity) {}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func cancelProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return os.ErrProcessDone
	}
	return cmd.Process.Kill()
}
