package judger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	minUserCompileTimeout     = 30 * time.Second
	defaultUserCompileTimeout = 2 * time.Minute
)

func compileLanguageRuntime(ctx context.Context, lang preparedLang, work string, limits Limits, submissionID uint, attempt int, logf func(format string, args ...any)) (CompileResult, error) {
	if lang.CompileCommand == "" {
		source := filepath.Join(work, languageSourceDir, lang.SourceName)
		if err := copyFile(source, filepath.Join(work, lang.SourceName), 0o644); err != nil {
			return CompileResult{}, err
		}
		return CompileResult{OK: true}, nil
	}
	sourceDir := filepath.Join(work, languageSourceDir)
	outputDir, err := os.MkdirTemp(work, "lang-out-")
	if err != nil {
		return CompileResult{}, err
	}
	defer os.RemoveAll(outputDir)

	timeout := minUserCompileTimeout
	if limits.TimeMS > int(timeout/time.Millisecond) {
		timeout = time.Duration(limits.TimeMS) * time.Millisecond
	}
	if timeout > defaultUserCompileTimeout {
		timeout = defaultUserCompileTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startedAt := time.Now()
	containerID, err := dockerCreateContainer(runCtx, dockerCreateRequest{
		Image:      lang.Image,
		Cmd:        []string{"sh", "-lc", lang.CompileCommand},
		WorkingDir: "/src",
		HostConfig: dockerHostConfig{
			Binds:       []string{sourceDir + ":/src:ro", outputDir + ":/work"},
			NetworkMode: "none",
			SecurityOpt: []string{"no-new-privileges"},
			CapDrop:     []string{"ALL"},
		},
	})
	if err != nil {
		return CompileResult{}, err
	}
	defer dockerRemoveContainer(context.Background(), containerID)
	if err := dockerStartContainer(runCtx, containerID); err != nil {
		return CompileResult{}, err
	}
	exitCode, err := dockerWaitContainer(runCtx, containerID)
	elapsed := int(time.Since(startedAt).Milliseconds())
	logTask(logf, submissionID, attempt, "compile_container=%s exit=%d", formatDuration(time.Since(startedAt)), exitCode)
	logCtx, logCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer logCancel()
	message := cleanCompileMessage(dockerLogs(logCtx, containerID, defaultCompileOutputLimit))
	if runCtx.Err() != nil {
		return CompileResult{OK: false, Message: runCtx.Err().Error(), TimeMS: elapsed}, nil
	}
	if err != nil {
		return CompileResult{}, err
	}
	if exitCode != 0 {
		if message == "" {
			message = fmt.Sprintf("compile exited with status %d", exitCode)
		}
		return CompileResult{OK: false, Message: message, TimeMS: elapsed}, nil
	}
	if err := copyDir(outputDir, work); err != nil {
		return CompileResult{}, err
	}
	return CompileResult{OK: true, Message: message, TimeMS: elapsed}, nil
}

func cleanCompileMessage(message string) string {
	return cleanBuildMessage(message)
}
