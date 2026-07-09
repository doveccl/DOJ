package main

import (
	"context"
	"fmt"
	"os"

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
		os.Exit(judger.JudgerCLI(context.Background(), os.Args[2:]))
	case "runner":
		os.Exit(runner.CLI(context.Background(), os.Args[2:]))
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: doj server|judger|runner [args...]")
}
