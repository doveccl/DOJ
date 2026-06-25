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

type customBuild struct {
	source  string
	command string
}

func prepareCustomJudge(ctx context.Context, work string, limits Limits) (string, CompileResult, error) {
	path, result, err := prepareCustomJudgePath(ctx, work, limits)
	if path == "" || err != nil || !result.OK {
		return "", result, err
	}
	return shellQuote(path), result, nil
}

func prepareContainerCustomJudge(ctx context.Context, work string, limits Limits) (string, CompileResult, error) {
	path, result, err := prepareCustomJudgePath(ctx, work, limits)
	if path == "" || err != nil || !result.OK {
		return "", result, err
	}
	rel, err := filepath.Rel(work, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", CompileResult{OK: false, Message: "custom judge path is outside job directory"}, nil
	}
	return shellQuote(slashpath.Join(containerWorkDir, filepath.ToSlash(rel))), result, nil
}

func prepareCustomJudgePath(ctx context.Context, work string, limits Limits) (string, CompileResult, error) {
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
	if path, ok, err := executableCustomJudgePath(dir); err != nil || ok {
		return path, CompileResult{OK: ok, Message: errorString(err)}, err
	}

	output := filepath.Join(work, "judge-program")
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err == nil {
		got, err := compileDockerfileJudge(ctx, dir, output, limits)
		if err != nil || !got.OK {
			return "", got, err
		}
		return output, got, nil
	}
	for _, item := range customBuilds(output) {
		source := filepath.Join(dir, item.source)
		if _, err := os.Stat(source); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", CompileResult{}, err
		}
		got, err := compileCustomJudge(ctx, dir, item.command, limits)
		if err != nil || !got.OK {
			return "", got, err
		}
		if err := os.Chmod(output, 0o700); err != nil {
			return "", CompileResult{}, err
		}
		return output, got, nil
	}
	return "", CompileResult{OK: false, Message: "custom judge requires an executable judge file or supported source file"}, nil
}

func executableCustomJudge(dir string) (string, bool, error) {
	path, ok, err := executableCustomJudgePath(dir)
	if path == "" || err != nil || !ok {
		return "", ok, err
	}
	return shellQuote(path), true, nil
}

func executableCustomJudgePath(dir string) (string, bool, error) {
	for _, name := range []string{"judge", "checker", "interactor", "main"} {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", false, err
		}
		if info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
			return path, true, nil
		}
	}
	return "", false, nil
}

func customBuilds(output string) []customBuild {
	quoted := shellQuote(output)
	return []customBuild{
		{source: "main.cc", command: "g++ -O2 -pipe main.cc -o " + quoted},
		{source: "main.cpp", command: "g++ -O2 -pipe main.cpp -o " + quoted},
		{source: "checker.cc", command: "g++ -O2 -pipe checker.cc -o " + quoted},
		{source: "checker.cpp", command: "g++ -O2 -pipe checker.cpp -o " + quoted},
		{source: "main.go", command: "go build -o " + quoted + " main.go"},
		{source: "main.rs", command: "rustc -O -o " + quoted + " main.rs"},
	}
}

func compileCustomJudge(ctx context.Context, dir string, command string, limits Limits) (CompileResult, error) {
	timeout := minCustomJudgeCompileTimeout
	if limits.TimeMS > int(minCustomJudgeCompileTimeout/time.Millisecond) {
		timeout = time.Duration(limits.TimeMS) * time.Millisecond
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := shellCommand(runCtx, command)
	cmd.Dir = dir
	configureProcess(cmd)
	outputLimit := int64(defaultCompileOutputLimit)
	if limits.OutputKB > 0 {
		outputLimit = int64(limits.OutputKB) * 1024
	}
	output := &limitBuffer{limit: outputLimit + 1}
	cmd.Stdout = output
	cmd.Stderr = output

	startedAt := time.Now()
	if err := cmd.Start(); err != nil {
		return CompileResult{}, err
	}
	err := cmd.Wait()
	elapsed := int(time.Since(startedAt).Milliseconds())
	if runCtx.Err() != nil {
		killProcessGroup(cmd)
		return CompileResult{OK: false, Message: runCtx.Err().Error(), TimeMS: elapsed}, nil
	}
	if output.overflow || int64(output.Len()) > outputLimit {
		return CompileResult{OK: false, Message: "custom judge compile output limit exceeded", TimeMS: elapsed}, nil
	}
	if err != nil {
		return CompileResult{OK: false, Message: output.String(), TimeMS: elapsed}, nil
	}
	return CompileResult{OK: true, Message: output.String(), TimeMS: elapsed}, nil
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

	containerID, err := dockerCreateContainer(runCtx, dockerCreateRequest{Image: imageID})
	if err != nil {
		return dockerBuildResult("", err, startedAt), nil
	}
	defer dockerRemoveContainer(context.Background(), containerID)

	if err := dockerCopyFile(runCtx, containerID, "/out/judge", output); err != nil {
		return dockerBuildResult("", err, startedAt), nil
	}
	if err := validateCustomJudgeBinary(output); err != nil {
		return CompileResult{OK: false, Message: err.Error(), TimeMS: int(time.Since(startedAt).Milliseconds())}, nil
	}
	return CompileResult{OK: true, TimeMS: int(time.Since(startedAt).Milliseconds())}, nil
}

func dockerBuildResult(out string, err error, startedAt time.Time) CompileResult {
	message := strings.TrimSpace(out)
	if message == "" && err != nil {
		message = err.Error()
	}
	return CompileResult{OK: false, Message: message, TimeMS: int(time.Since(startedAt).Milliseconds())}
}

func validateCustomJudgeBinary(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("custom judge output /out/judge is missing: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("custom judge output /out/judge is not a regular file")
	}
	if info.Size() <= 0 {
		return fmt.Errorf("custom judge output /out/judge is empty")
	}
	if info.Size() > maxCustomJudgeBinaryBytes {
		return fmt.Errorf("custom judge output /out/judge exceeds %d bytes", maxCustomJudgeBinaryBytes)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return err
	}
	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprint(err)
}
