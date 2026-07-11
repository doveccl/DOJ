package runner

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	common "github.com/doveccl/doj/contract/judger"
)

const Version = common.Version

const (
	runnerUID = 20001
	runnerGID = 20001
)

func CLI(ctx context.Context, args []string) int {
	if len(args) == 0 {
		usage(os.Stderr)
		return 2
	}
	switch args[0] {
	case "judge":
		return BuiltinJudgeMain(args[1:])
	case "serve":
		return serveCLI(ctx, args[1:])
	case "wait-exec":
		return waitExecCLI(ctx, args[1:])
	case "version":
		fmt.Fprintln(os.Stdout, Version)
		return 0
	default:
		usage(os.Stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage: doj runner judge [--mode=default|strict] input output answer [result]")
	fmt.Fprintln(w, "       doj runner serve --socket /path/runner.sock --work /path/task")
	fmt.Fprintln(w, "       doj runner wait-exec /path/release uid gid command [args...]")
}

func serveCLI(ctx context.Context, args []string) int {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	socket := flags.String("socket", "", "unix socket path")
	work := flags.String("work", "", "task work directory")
	runner := flags.String("runner", "", "runner binary path")
	runtimeRoot := flags.String("runtime-root", "", "runtime file root")
	skipRuntime := flags.String("skip-runtime", "", "comma-separated runtime files to skip")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	identity := ProcessIdentity{UID: runnerUID, GID: runnerGID, Enabled: true}
	err := ServeRunner(ctx, RunnerServe{
		Socket:       *socket,
		Work:         *work,
		Runner:       *runner,
		RuntimeRoot:  *runtimeRoot,
		SkipRuntime:  *skipRuntime,
		UserIdentity: identity,
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func waitExecCLI(ctx context.Context, args []string) int {
	if len(args) < 4 {
		usage(os.Stderr)
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
