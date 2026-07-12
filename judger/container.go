package judger

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/doveccl/doj/judger/runner"
)

const containerWorkDir = "/work"

type ContainerTask struct {
	Runner           string
	Work             string
	CgroupRoot       string
	ProcRoot         string
	CustomJudgeCache string
	Task             runner.Task
	Logf             func(format string, args ...any)
	Progress         func(stage string, done int64, total *int64)
}

func RunContainerTask(ctx context.Context, req ContainerTask) (runner.TaskResult, error) {
	if req.Runner == "" {
		return runner.TaskResult{}, fmt.Errorf("runner path is required")
	}
	if req.Work == "" {
		return runner.TaskResult{}, fmt.Errorf("work directory is required")
	}
	if req.ProcRoot == "" {
		req.ProcRoot = "/proc"
	}
	work, err := filepath.Abs(req.Work)
	if err != nil {
		return runner.TaskResult{}, err
	}
	if err := privateDir(work); err != nil {
		return runner.TaskResult{}, err
	}

	task := req.Task
	langStartedAt := time.Now()
	lang, err := prepareLanguageRuntime(work, task.Lang, task.Source)
	logStep(req.Logf, task.SubmissionID, task.Attempt, "prepare_language_runtime", langStartedAt)
	if err != nil {
		return runner.TaskResult{}, err
	}
	imageStartedAt := time.Now()
	pulled, err := dockerEnsureImage(ctx, lang.Image)
	logStep(req.Logf, task.SubmissionID, task.Attempt, "ensure_language_image", imageStartedAt)
	if err != nil {
		return runner.TaskResult{}, fmt.Errorf("ensure language image %q: %w", lang.Image, err)
	}
	if pulled {
		logTask(req.Logf, task.SubmissionID, task.Attempt, "language_image_pulled=%s", lang.Image)
	}
	if lang.CompileCommand == "" {
		source := filepath.Join(work, languageSourceDir, lang.SourceName)
		if err := copyFile(source, filepath.Join(work, lang.SourceName), 0o600); err != nil {
			return runner.TaskResult{}, err
		}
	}
	runtimeRoot := containerWorkDir
	skipRuntime := ""
	if lang.CompileCommand != "" {
		runtimeRoot = filepath.ToSlash(filepath.Join(containerWorkDir, languageSourceDir))
		skipRuntime = lang.SourceName
	}
	restoreAssets, err := stashCaseFiles(work, task.Cases)
	if err != nil {
		return runner.TaskResult{}, err
	}
	defer restoreAssets()

	socketDir, err := os.MkdirTemp(filepath.Dir(work), ".doj-runner-")
	if err != nil {
		return runner.TaskResult{}, err
	}
	defer os.RemoveAll(socketDir)
	socket := filepath.Join(socketDir, "runner.sock")
	_ = os.Remove(socket)
	startContainerStartedAt := time.Now()
	containerID, err := startRunnerContainer(ctx, lang.Image, req.Runner, work, socket, runtimeRoot, skipRuntime, task.Limits)
	logStep(req.Logf, task.SubmissionID, task.Attempt, "start_runner_container", startContainerStartedAt)
	if err != nil {
		return runner.TaskResult{}, err
	}
	defer dockerRemoveContainer(context.Background(), containerID)

	pidStartedAt := time.Now()
	initPID, err := dockerContainerPID(ctx, containerID)
	logStep(req.Logf, task.SubmissionID, task.Attempt, "inspect_runner_pid", pidStartedAt)
	if err != nil {
		return runner.TaskResult{}, containerError(ctx, containerID, err)
	}
	socketStartedAt := time.Now()
	socketCtx, cancelSocketWait := context.WithTimeout(ctx, 10*time.Second)
	defer cancelSocketWait()
	if err := waitUnixSocket(socketCtx, socket); err != nil {
		logStep(req.Logf, task.SubmissionID, task.Attempt, "wait_runner_socket_error", socketStartedAt)
		return runner.TaskResult{}, containerError(ctx, containerID, fmt.Errorf("wait runner socket: %w", err))
	}
	logStep(req.Logf, task.SubmissionID, task.Attempt, "wait_runner_socket", socketStartedAt)
	connectStartedAt := time.Now()
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", socket)
	logStep(req.Logf, task.SubmissionID, task.Attempt, "connect_runner_socket", connectStartedAt)
	if err != nil {
		return runner.TaskResult{}, err
	}
	defer conn.Close()
	stopClose := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stopClose()
	codec := runner.NewCodec(conn)
	client := runnerClient{
		codec:      codec,
		cgroupRoot: req.CgroupRoot,
		procRoot:   req.ProcRoot,
		initPID:    initPID,
		taskID:     runner.SafeCaseID(fmt.Sprintf("%d-%d", req.Task.SubmissionID, req.Task.Attempt)),
		limits:     req.Task.Limits,
		work:       work,
		judgeCache: req.CustomJudgeCache,
		logf:       req.Logf,
		progress:   req.Progress,
	}
	return client.runTask(ctx, task, lang.CompileCommand, lang.CompileMS, lang.RunCommand, restoreAssets)
}
