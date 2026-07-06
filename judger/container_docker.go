package judger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func startRunnerContainer(ctx context.Context, image string, runner string, work string, socket string, runtimeRoot string, skipRuntime string) (string, error) {
	_ = os.Remove(socket)
	containerID, err := dockerCreateContainer(ctx, dockerCreateRequest{
		Image: image,
		Cmd: []string{
			"/usr/local/bin/doj", "runner", "serve",
			"--socket", "/runner/runner.sock",
			"--work", containerWorkDir,
			"--runner", "/usr/local/bin/doj",
			"--runtime-root", runtimeRoot,
			"--skip-runtime", skipRuntime,
		},
		WorkingDir: containerWorkDir,
		HostConfig: dockerHostConfig{
			Binds: []string{
				work + ":" + containerWorkDir,
				filepath.Dir(socket) + ":/runner",
				runner + ":/usr/local/bin/doj:ro",
			},
			NetworkMode: "none",
			SecurityOpt: []string{"no-new-privileges"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("create runner container: %w", err)
	}
	if err := dockerStartContainer(ctx, containerID); err != nil {
		dockerRemoveContainer(context.Background(), containerID)
		return "", fmt.Errorf("start runner container: %w", err)
	}
	return containerID, nil
}

func dockerContainerPID(ctx context.Context, containerID string) (int, error) {
	pid, err := dockerInspectPID(ctx, containerID)
	if err != nil {
		return 0, fmt.Errorf("inspect runner container pid: %w", err)
	}
	return pid, nil
}

func containerError(ctx context.Context, containerID string, err error) error {
	logs := dockerContainerLogs(ctx, containerID)
	if logs == "" {
		return err
	}
	return fmt.Errorf("%w\nrunner container logs:\n%s", err, logs)
}

func dockerContainerLogs(ctx context.Context, containerID string) string {
	logCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return dockerLogs(logCtx, containerID, defaultCompileOutputLimit)
}

func waitUnixSocket(ctx context.Context, socket string) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		if info, err := os.Stat(socket); err == nil && info.Mode()&os.ModeSocket != 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
