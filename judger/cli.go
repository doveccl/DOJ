package judger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/doveccl/doj/judger/runner"
)

const Version = runner.Version
const JudgerRoot = "/var/lib/doj"

const defaultServer = "http://localhost:7974"

func JudgerCLI(ctx context.Context, args []string) int {
	if len(args) > 0 && args[0] == "version" {
		fmt.Fprintln(os.Stdout, Version)
		return 0
	}
	if len(args) > 0 {
		judgerUsage(os.Stderr)
		return 2
	}
	runner, err := installRunner()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	root := JudgerRoot
	tasks := filepath.Join(root, "tasks")
	cache := filepath.Join(root, "cache")
	if err := os.MkdirAll(tasks, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := os.MkdirAll(cache, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	err = RunLoop(ctx, LoopConfig{
		Worker: WorkerConfig{
			Server:     getenv("SERVER", defaultServer),
			Token:      os.Getenv("TOKEN"),
			Runner:     runner,
			Tasks:      tasks,
			Cache:      cache,
			CgroupRoot: DefaultCgroupRoot(),
			ProcRoot:   "/proc",
		},
		Concurrency: getenvInt("CONCURRENCY", 1),
		Logf: func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		},
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func RunnerCLI(ctx context.Context, args []string) int {
	return runner.CLI(ctx, args)
}

func judgerUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: doj judger")
	fmt.Fprintln(w, "       doj judger version")
}

func getenv(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getenvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var got int
	if _, err := fmt.Sscanf(value, "%d", &got); err != nil || got <= 0 {
		return defaultValue
	}
	return got
}

func installRunner() (string, error) {
	src, err := findRunner()
	if err != nil {
		return "", err
	}
	target := filepath.Join(JudgerRoot, "bin", "doj")
	if sameFile(src, target) {
		return target, nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", err
	}
	input, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return "", err
	}
	if err := output.Close(); err != nil {
		return "", err
	}
	return target, nil
}

func findRunner() (string, error) {
	exe, err := os.Executable()
	if err == nil {
		return exe, nil
	}
	path, err := exec.LookPath("doj")
	if err != nil {
		return "", fmt.Errorf("doj binary is required in PATH")
	}
	return path, nil
}

func sameFile(a string, b string) bool {
	aInfo, aErr := os.Stat(a)
	bInfo, bErr := os.Stat(b)
	return aErr == nil && bErr == nil && os.SameFile(aInfo, bInfo)
}
