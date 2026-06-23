//go:build judger && linux

package main

import (
	"context"
	"os"

	"github.com/doveccl/doj/judger"
)

func main() {
	os.Exit(judger.JudgerCLI(context.Background(), os.Args[1:]))
}
