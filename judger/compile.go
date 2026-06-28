package judger

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
)

const (
	minUserCompileTimeout     = 30 * time.Second
	defaultUserCompileTimeout = 2 * time.Minute
)

func compileUserProgram(ctx context.Context, work string, req CompileRequest) (CompileResult, error) {
	if req.CompileCommand == "" {
		return CompileResult{OK: true}, nil
	}
	timeout := minUserCompileTimeout
	if req.Limits.TimeMS > int(timeout/time.Millisecond) {
		timeout = time.Duration(req.Limits.TimeMS) * time.Millisecond
	}
	if timeout > defaultUserCompileTimeout {
		timeout = defaultUserCompileTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startedAt := time.Now()
	output := &limitBuffer{limit: defaultCompileOutputLimit + 1}
	bin, args, err := parseCommand(req.CompileCommand)
	if err != nil {
		return CompileResult{OK: false, Message: err.Error()}, nil
	}
	cmd := commandContext(runCtx, bin, args...)
	cmd.Dir = filepath.Join(work, languageSourceDir)
	cmd.Stdout = output
	cmd.Stderr = output
	err = cmd.Run()
	elapsed := int(time.Since(startedAt).Milliseconds())
	message := cleanCompileMessage(output.String())
	if runCtx.Err() != nil {
		return CompileResult{OK: false, Message: runCtx.Err().Error(), TimeMS: elapsed}, nil
	}
	if output.overflow {
		return CompileResult{OK: false, Message: "compile output limit exceeded", TimeMS: elapsed}, nil
	}
	if err != nil {
		if message == "" {
			message = fmt.Sprintf("compile failed: %v", err)
		}
		return CompileResult{OK: false, Message: message, TimeMS: elapsed}, nil
	}
	return CompileResult{OK: true, Message: message, TimeMS: elapsed}, nil
}

func cleanCompileMessage(message string) string {
	return cleanBuildMessage(message)
}
