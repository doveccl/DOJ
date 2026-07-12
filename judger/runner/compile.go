package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

const (
	defaultUserCompileTimeout = 10 * time.Second
	maxUserCompileTimeout     = 30 * time.Second
	compileDiskPollInterval   = 10 * time.Millisecond
	maxCompileFiles           = 10_000
)

func compileUserProgram(ctx context.Context, work string, runner string, userIdentity ProcessIdentity, req CompileRequest) (CompileResult, error) {
	if req.CompileCommand == "" {
		return CompileResult{OK: true}, nil
	}
	timeout := compileTimeout(req.CompileMS)
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startedAt := time.Now()
	output := &limitBuffer{limit: defaultCompileOutputLimit + 1}
	bin, args, err := parseCommand(req.CompileCommand)
	if err != nil {
		return CompileResult{OK: false, Message: err.Error()}, nil
	}
	cmd := commandContext(runCtx, bin, args...)
	if userIdentity.Enabled {
		identity := ProcessIdentity{UID: userIdentity.UID + 2, GID: 0, Enabled: true}
		if err := prepareCompilerIdentity(work); err != nil {
			return CompileResult{}, err
		}
		wrapperArgs := []string{"runner", "wait-exec", "/dev/null", strconv.FormatUint(uint64(identity.UID), 10), strconv.FormatUint(uint64(identity.GID), 10), bin}
		cmd = commandContext(runCtx, runner, append(wrapperArgs, args...)...)
	}
	cmd.Dir = filepath.Join(work, languageSourceDir)
	cmd.Stdout = output
	cmd.Stderr = output
	configureProcess(cmd)
	cmd.Cancel = func() error { return cancelProcessGroup(cmd) }
	cmd.WaitDelay = 100 * time.Millisecond
	err, fileLimitExceeded := runCompiler(cmd, filepath.Join(work, languageSourceDir), int64(req.Limits.FileKB)*1024)
	elapsed := int(time.Since(startedAt).Milliseconds())
	message := cleanCompileMessage(output.String())
	if fileLimitExceeded {
		return CompileResult{OK: false, Message: "compile file limit exceeded", TimeMS: elapsed}, nil
	}
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

func compileTimeout(compileMS int) time.Duration {
	if compileMS <= 0 {
		return defaultUserCompileTimeout
	}
	timeout := time.Duration(compileMS) * time.Millisecond
	if timeout > maxUserCompileTimeout {
		return maxUserCompileTimeout
	}
	return timeout
}

func runCompiler(cmd *exec.Cmd, root string, maxBytes int64) (error, bool) {
	if err := cmd.Start(); err != nil {
		return err, false
	}
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	ticker := time.NewTicker(compileDiskPollInterval)
	defer ticker.Stop()
	for {
		select {
		case err := <-wait:
			exceeded, scanErr := compileWorkspaceExceeded(root, maxBytes)
			if scanErr != nil {
				return scanErr, false
			}
			return err, exceeded
		case <-ticker.C:
			exceeded, err := compileWorkspaceExceeded(root, maxBytes)
			if err != nil {
				_ = cancelProcessGroup(cmd)
				<-wait
				return err, false
			}
			if exceeded {
				_ = cancelProcessGroup(cmd)
				<-wait
				return nil, true
			}
		}
	}
}

func compileWorkspaceExceeded(root string, maxBytes int64) (bool, error) {
	var bytes int64
	files := 0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		files++
		if files > maxCompileFiles {
			return filepath.SkipAll
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			bytes += info.Size()
		}
		if maxBytes > 0 && bytes > maxBytes {
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return files > maxCompileFiles || (maxBytes > 0 && bytes > maxBytes), nil
}

func prepareCompilerIdentity(work string) error {
	if err := os.Chmod(work, 0o710); err != nil {
		return err
	}
	root := filepath.Join(work, languageSourceDir)
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		mode := os.FileMode(0o660)
		if entry.IsDir() {
			mode = 0o770
		}
		if err := os.Chown(path, 0, 0); err != nil {
			return err
		}
		if err := chmodIfNeeded(path, mode); err != nil {
			return err
		}
		return nil
	})
}

func cleanCompileMessage(message string) string {
	return CleanBuildMessage(message)
}
