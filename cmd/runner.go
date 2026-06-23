//go:build runner && linux

package main

import (
	"context"
	"os"

	"github.com/doveccl/doj/judger"
)

func main() {
	os.Exit(judger.RunnerCLI(context.Background(), os.Args[1:]))
}
