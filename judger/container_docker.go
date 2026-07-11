package judger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/doveccl/doj/judger/runner"
)

const (
	runnerMemoryFloor = int64(1 << 30)
	runnerNanoCPUs    = runner.CgroupCPUQuotaUS * 1_000_000_000 / runner.CgroupCPUPeriodUS
	runnerPidsFloor   = int64(256)
)

func startRunnerContainer(ctx context.Context, image string, runner string, work string, socket string, runtimeRoot string, skipRuntime string, limits Limits) (string, error) {
	_ = os.Remove(socket)
	containerID, err := dockerCreateContainer(ctx, dockerCreateRequest{
		Image:      image,
		Entrypoint: []string{"/usr/local/bin/doj"},
		Env:        []string{"HOME=/tmp/home", "TMPDIR=/tmp", "XDG_CACHE_HOME=/tmp/cache", "GOCACHE=/tmp/go-cache"},
		Cmd: []string{
			"runner", "serve",
			"--socket", "/runner/runner.sock",
			"--work", containerWorkDir,
			"--runner", "/usr/local/bin/doj",
			"--runtime-root", runtimeRoot,
			"--skip-runtime", skipRuntime,
		},
		WorkingDir: containerWorkDir,
		HostConfig: runnerHostConfig(limits, []string{
			work + ":" + containerWorkDir,
			filepath.Dir(socket) + ":/runner",
			runner + ":/usr/local/bin/doj:ro",
		}),
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

func runnerHostConfig(limits Limits, binds []string) dockerHostConfig {
	memory := int64(limits.MemoryKB)*1024 + (256 << 20)
	if memory < runnerMemoryFloor {
		memory = runnerMemoryFloor
	}
	pids := int64(limits.Pids) + 128
	if pids < runnerPidsFloor {
		pids = runnerPidsFloor
	}
	fileSize := int64(limits.FileKB) * 1024
	if fileSize <= 0 {
		fileSize = 64 << 20
	}
	tmpSize := fileSize * 2
	if tmpSize < 128<<20 {
		tmpSize = 128 << 20
	}
	varTmpSize := fileSize
	if varTmpSize < 64<<20 {
		varTmpSize = 64 << 20
	}
	return dockerHostConfig{
		Binds:          binds,
		NetworkMode:    "none",
		SecurityOpt:    []string{"no-new-privileges"},
		CapDrop:        []string{"ALL"},
		CapAdd:         []string{"CHOWN", "SETUID", "SETGID", "KILL"},
		Memory:         memory,
		NanoCPUs:       runnerNanoCPUs,
		PidsLimit:      pids,
		FileSize:       fileSize,
		ReadonlyRootfs: true,
		Tmpfs: map[string]string{
			"/tmp":     fmt.Sprintf("rw,exec,nosuid,nodev,mode=1777,size=%d", tmpSize),
			"/var/tmp": fmt.Sprintf("rw,exec,nosuid,nodev,mode=1777,size=%d", varTmpSize),
		},
		ShmSize: 16 << 20,
		Init:    true,
	}
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
