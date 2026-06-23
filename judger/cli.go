package judger

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const Version = "v4-dev"

func JudgerCLI(ctx context.Context, args []string) int {
	if len(args) > 0 && args[0] == "version" {
		fmt.Fprintln(os.Stdout, Version)
		return 0
	}
	if len(args) > 0 && args[0] == "once" {
		return judgerOnce(ctx, args[1:])
	}
	if len(args) > 0 && args[0] == "serve" {
		return judgerServe(ctx, args[1:])
	}
	judgerUsage(os.Stderr)
	return 2
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
	case "version":
		fmt.Fprintln(os.Stdout, Version)
		return 0
	default:
		runnerUsage(os.Stderr)
		return 2
	}
}

func runnerUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: doj-runner judge --mode default --input in --answer out --result result.json")
	fmt.Fprintln(w, "       doj-runner serve --socket /path/runner.sock --work /path/job")
}

func runnerServe(ctx context.Context, args []string) int {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	socket := flags.String("socket", "", "unix socket path")
	work := flags.String("work", "", "job work directory")
	runner := flags.String("runner", "", "runner binary path")
	userUID := flags.Int("user-uid", getenvInt("DOJ_RUNNER_USER_UID", -1), "uid for user program; negative disables setuid")
	userGID := flags.Int("user-gid", getenvInt("DOJ_RUNNER_USER_GID", -1), "gid for user program; negative disables setgid")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	identity := ProcessIdentity{}
	if *userUID >= 0 && *userGID >= 0 {
		identity = ProcessIdentity{UID: uint32(*userUID), GID: uint32(*userGID), Enabled: true}
	}
	if err := ServeRunner(ctx, RunnerServe{Socket: *socket, Work: *work, Runner: *runner, UserIdentity: identity}); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func judgerOnce(ctx context.Context, args []string) int {
	flags := flag.NewFlagSet("once", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	server := flags.String("server", getenv("DOJ_SERVER", "http://localhost:7974"), "server base URL")
	token := flags.String("token", "", "judger API token")
	tokenFile := flags.String("token-file", "", "file containing judger API token")
	name := flags.String("name", getenv("DOJ_JUDGER_NAME", "local-judger"), "judger name")
	runner := flags.String("runner", getenv("DOJ_RUNNER", "doj-runner"), "runner binary path")
	work := flags.String("work", getenv("DOJ_WORK", filepath.Join(os.TempDir(), "doj-judger")), "work directory")
	cgroupRoot := flags.String("cgroup-root", getenv("DOJ_CGROUP_ROOT", ""), "cgroup v2 root for user programs")
	procRoot := flags.String("proc-root", getenv("DOJ_PROC_ROOT", "/proc"), "proc root for container pid mapping")
	lease := flags.Int("lease", 60, "lease seconds")
	timeout := flags.Duration("timeout", 5*time.Minute, "single task timeout")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	resolvedToken, err := resolveToken(*token, *tokenFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	runCtx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	worked, err := RunOne(runCtx, WorkerConfig{
		Server:       *server,
		Token:        resolvedToken,
		Name:         *name,
		Runner:       *runner,
		Work:         *work,
		CgroupRoot:   *cgroupRoot,
		ProcRoot:     *procRoot,
		LeaseSeconds: *lease,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !worked {
		fmt.Fprintln(os.Stdout, "no task")
		return 0
	}
	fmt.Fprintln(os.Stdout, "task finished")
	return 0
}

func judgerServe(ctx context.Context, args []string) int {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	server := flags.String("server", getenv("DOJ_SERVER", "http://localhost:7974"), "server base URL")
	token := flags.String("token", "", "judger API token")
	tokenFile := flags.String("token-file", "", "file containing judger API token")
	name := flags.String("name", getenv("DOJ_JUDGER_NAME", "local-judger"), "judger name")
	runner := flags.String("runner", getenv("DOJ_RUNNER", "doj-runner"), "runner binary path")
	work := flags.String("work", getenv("DOJ_WORK", filepath.Join(os.TempDir(), "doj-judger")), "work directory")
	cgroupRoot := flags.String("cgroup-root", getenv("DOJ_CGROUP_ROOT", ""), "cgroup v2 root for user programs")
	procRoot := flags.String("proc-root", getenv("DOJ_PROC_ROOT", "/proc"), "proc root for container pid mapping")
	lease := flags.Int("lease", 60, "lease seconds")
	timeout := flags.Duration("timeout", 5*time.Minute, "single task timeout")
	interval := flags.Duration("interval", time.Second, "poll interval when idle or after errors")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	resolvedToken, err := resolveToken(*token, *tokenFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	err = RunLoop(ctx, LoopConfig{
		Worker: WorkerConfig{
			Server:       *server,
			Token:        resolvedToken,
			Name:         *name,
			Runner:       *runner,
			Work:         *work,
			CgroupRoot:   *cgroupRoot,
			ProcRoot:     *procRoot,
			LeaseSeconds: *lease,
		},
		TaskTimeout:  *timeout,
		PollInterval: *interval,
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

func judgerUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: doj-judger once --server http://localhost:7974 --runner ./doj-runner")
	fmt.Fprintln(w, "       doj-judger serve --server http://localhost:7974 --runner ./doj-runner")
}

func resolveToken(token string, file string) (string, error) {
	token = strings.TrimSpace(token)
	file = strings.TrimSpace(file)
	if token != "" || file == "" {
		return token, nil
	}
	content, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("read judger token file: %w", err)
	}
	return strings.TrimSpace(string(content)), nil
}

func getenv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil {
		return fallback
	}
	return parsed
}
