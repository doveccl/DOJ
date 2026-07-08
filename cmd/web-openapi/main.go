package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/doveccl/doj/server/api"
)

func main() {
	target := "contract/web/openapi.yaml"
	if len(os.Args) > 1 {
		target = os.Args[1]
	}
	data, err := api.NewOpenAPI().OpenAPI().YAML()
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(target, data, 0o644); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
