package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/doveccl/doj/judger"
	"github.com/doveccl/doj/judger/runner"
	"github.com/doveccl/doj/server"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "server":
		server.Main()
	case "judger":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		code := judger.JudgerCLI(ctx, os.Args[2:])
		stop()
		os.Exit(code)
	case "runner":
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		code := runner.CLI(ctx, os.Args[2:])
		stop()
		os.Exit(code)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: doj server|judger|runner [args...]")
}
