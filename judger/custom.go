package judger

import (
	"context"
	"fmt"
	"os"
	slashpath "path"
	"path/filepath"
	"strings"
	"time"
)

const maxCustomJudgeBinaryBytes = 64 << 20
const minCustomJudgeCompileTimeout = 30 * time.Second

func prepareContainerCustomJudge(ctx context.Context, work string, limits Limits, cachePath string) (string, CompileResult, error) {
	path, result, err := prepareCustomJudgePath(ctx, work, limits, cachePath)
	if path == "" || err != nil || !result.OK {
		return "", result, err
	}
	rel, err := filepath.Rel(work, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", CompileResult{OK: false, Message: "custom judge path is outside job directory"}, nil
	}
	return slashpath.Join(containerWorkDir, filepath.ToSlash(rel)), result, nil
}

func prepareCustomJudgePath(ctx context.Context, work string, limits Limits, cachePath string) (string, CompileResult, error) {
	startedAt := time.Now()
	dir := filepath.Join(work, "judge")
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", CompileResult{OK: false, Message: "custom judge resources are missing"}, nil
		}
		return "", CompileResult{}, err
	}
	if !info.IsDir() {
		return "", CompileResult{OK: false, Message: "custom judge resource path is not a directory"}, nil
	}
	output := filepath.Join(work, "judge-program")
	if cachePath != "" {
		if err := copyCachedCustomJudge(cachePath, output); err == nil {
			return output, CompileResult{OK: true, TimeMS: int(time.Since(startedAt).Milliseconds())}, nil
		} else if !os.IsNotExist(err) {
			return "", CompileResult{}, err
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err != nil {
		if os.IsNotExist(err) {
			return "", CompileResult{OK: false, Message: "custom judge requires Dockerfile"}, nil
		}
		return "", CompileResult{}, err
	}
	got, err := compileDockerfileJudge(ctx, dir, output, limits)
	if err != nil || !got.OK {
		return "", got, err
	}
	if cachePath != "" {
		if err := storeCachedCustomJudge(output, cachePath); err != nil {
			return "", CompileResult{}, err
		}
	}
	return output, got, nil
}

func copyCachedCustomJudge(cachePath string, output string) error {
	if err := validateCustomJudgeProgram(cachePath, cachePath); err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return os.Remove(cachePath)
	}
	return copyFile(cachePath, output, 0o700)
}

func storeCachedCustomJudge(output string, cachePath string) error {
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(cachePath), ".judge-program-")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	if err := copyFile(output, tmpPath, 0o700); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, cachePath)
}

func compileDockerfileJudge(ctx context.Context, dir string, output string, limits Limits) (CompileResult, error) {
	timeout := minCustomJudgeCompileTimeout
	if limits.TimeMS > int(minCustomJudgeCompileTimeout/time.Millisecond) {
		timeout = time.Duration(limits.TimeMS) * time.Millisecond
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	outputLimit := int64(defaultCompileOutputLimit)
	if limits.OutputKB > 0 {
		outputLimit = int64(limits.OutputKB) * 1024
	}
	startedAt := time.Now()

	imageID, out, err := dockerBuildImage(runCtx, dir, "Dockerfile", outputLimit)
	if err != nil {
		return dockerBuildResult(out, err, startedAt), nil
	}
	defer dockerRemoveImage(context.Background(), imageID)

	cmd, err := dockerImageCmd(runCtx, imageID)
	if err != nil {
		return dockerBuildResult("", err, startedAt), nil
	}
	program, ok := customJudgeProgramFromCmd(cmd)
	if !ok {
		return CompileResult{OK: false, Message: "custom judge Dockerfile CMD must point to a single absolute program path", TimeMS: int(time.Since(startedAt).Milliseconds())}, nil
	}
	containerID, err := dockerCreateContainer(runCtx, dockerCreateRequest{Image: imageID})
	if err != nil {
		return dockerBuildResult("", err, startedAt), nil
	}
	defer dockerRemoveContainer(context.Background(), containerID)

	if err := dockerCopyFile(runCtx, containerID, program, output); err != nil {
		return dockerBuildResult("", err, startedAt), nil
	}
	if err := validateCustomJudgeProgram(output, program); err != nil {
		return CompileResult{OK: false, Message: err.Error(), TimeMS: int(time.Since(startedAt).Milliseconds())}, nil
	}
	return CompileResult{OK: true, TimeMS: int(time.Since(startedAt).Milliseconds())}, nil
}

func customJudgeProgramFromCmd(cmd []string) (string, bool) {
	if len(cmd) == 1 && slashpath.IsAbs(cmd[0]) {
		return cmd[0], true
	}
	if len(cmd) == 3 && cmd[0] == "/bin/sh" && cmd[1] == "-c" {
		program := strings.TrimSpace(cmd[2])
		if slashpath.IsAbs(program) && len(strings.Fields(program)) == 1 {
			return program, true
		}
	}
	return "", false
}

func dockerBuildResult(out string, err error, startedAt time.Time) CompileResult {
	message := strings.TrimSpace(out)
	if message == "" && err != nil {
		message = err.Error()
	}
	return CompileResult{OK: false, Message: message, TimeMS: int(time.Since(startedAt).Milliseconds())}
}

func validateCustomJudgeProgram(path string, source string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("custom judge CMD %s is missing: %w", source, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("custom judge CMD %s is not a regular file", source)
	}
	if info.Size() <= 0 {
		return fmt.Errorf("custom judge CMD %s is empty", source)
	}
	if info.Size() > maxCustomJudgeBinaryBytes {
		return fmt.Errorf("custom judge CMD %s exceeds %d bytes", source, maxCustomJudgeBinaryBytes)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return err
	}
	return nil
}
