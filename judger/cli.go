package judger

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

const Version = "v4"
const JudgerRoot = "/var/lib/doj"

const (
	defaultServer = "http://localhost:7974"
	runnerUID     = 20001
	runnerGID     = 20001
)

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
	work := filepath.Join(JudgerRoot, "jobs")
	if err := os.MkdirAll(work, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	err = RunLoop(ctx, LoopConfig{
		Worker: WorkerConfig{
			Server:     getenv("SERVER", defaultServer),
			Token:      os.Getenv("TOKEN"),
			Runner:     runner,
			Work:       work,
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
	_ = ctx
	if len(args) == 0 {
		runnerUsage(os.Stderr)
		return 2
	}
	switch args[0] {
	case "judge":
		return BuiltinJudgeMain(args[1:])
	case "serve":
		return runnerServe(ctx, args[1:])
	case "wait-exec":
		return runnerWaitExec(ctx, args[1:])
	case "version":
		fmt.Fprintln(os.Stdout, Version)
		return 0
	default:
		runnerUsage(os.Stderr)
		return 2
	}
}

func runnerUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: doj-runner judge [--mode=default|strict] input output answer [result]")
	fmt.Fprintln(w, "       doj-runner serve --socket /path/runner.sock --work /path/job")
	fmt.Fprintln(w, "       doj-runner wait-exec /path/release uid gid command [args...]")
}

func runnerServe(ctx context.Context, args []string) int {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	socket := flags.String("socket", "", "unix socket path")
	work := flags.String("work", "", "job work directory")
	runner := flags.String("runner", "", "runner binary path")
	runtimeRoot := flags.String("runtime-root", "", "runtime file root")
	skipRuntime := flags.String("skip-runtime", "", "comma-separated runtime files to skip")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	identity := ProcessIdentity{UID: runnerUID, GID: runnerGID, Enabled: true}
	if err := ServeRunner(ctx, RunnerServe{Socket: *socket, Work: *work, Runner: *runner, RuntimeRoot: *runtimeRoot, SkipRuntime: *skipRuntime, UserIdentity: identity}); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func runnerWaitExec(ctx context.Context, args []string) int {
	if len(args) < 4 {
		runnerUsage(os.Stderr)
		return 2
	}
	release := args[0]
	uid, err := strconv.ParseUint(args[1], 10, 32)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	gid, err := strconv.ParseUint(args[2], 10, 32)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	for {
		if _, err := os.Stat(release); err == nil {
			break
		} else if !os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		select {
		case <-ctx.Done():
			return 1
		case <-time.After(10 * time.Millisecond):
		}
	}
	if err := dropIdentity(ProcessIdentity{UID: uint32(uid), GID: uint32(gid), Enabled: true}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := execProgram(args[3], args[4:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 127
}

func judgerUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: doj-judger")
	fmt.Fprintln(w, "       doj-judger version")
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
	target := filepath.Join(JudgerRoot, "bin", "doj-runner")
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
		candidate := filepath.Join(filepath.Dir(exe), "doj-runner")
		if info, statErr := os.Stat(candidate); statErr == nil && info.Mode().IsRegular() {
			return candidate, nil
		}
	}
	path, err := exec.LookPath("doj-runner")
	if err != nil {
		return "", fmt.Errorf("doj-runner binary is required beside doj-judger or in PATH")
	}
	return path, nil
}

func sameFile(a string, b string) bool {
	aInfo, aErr := os.Stat(a)
	bInfo, bErr := os.Stat(b)
	return aErr == nil && bErr == nil && os.SameFile(aInfo, bInfo)
}
