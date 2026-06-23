//go:build !(linux || darwin || freebsd || openbsd || netbsd)

package judger

import (
	"context"
	"os/exec"
)

func shellCommand(ctx context.Context, command string) *exec.Cmd {
	return exec.CommandContext(ctx, command)
}

func configureProcess(*exec.Cmd) {}

func applyProcessIdentity(*exec.Cmd, ProcessIdentity) {}

func killProcessGroup(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
