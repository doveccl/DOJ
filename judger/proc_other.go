//go:build !linux

package judger

import (
	"context"
	"os/exec"
)

func shellCommand(ctx context.Context, command string) *exec.Cmd {
	return exec.CommandContext(ctx, command)
}

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
