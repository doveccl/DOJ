package judger

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

const containerWorkDir = "/work"

type ContainerTask struct {
	Runner     string
	Work       string
	CgroupRoot string
	ProcRoot   string
	Task       Task
	Logf       func(format string, args ...any)
}

func RunContainerTask(ctx context.Context, req ContainerTask) (TaskResult, error) {
	if req.Runner == "" {
		return TaskResult{}, fmt.Errorf("runner path is required")
	}
	if req.Work == "" {
		return TaskResult{}, fmt.Errorf("work directory is required")
	}
	if req.ProcRoot == "" {
		req.ProcRoot = "/proc"
	}
	work, err := filepath.Abs(req.Work)
	if err != nil {
		return TaskResult{}, err
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		return TaskResult{}, err
	}

	task := req.Task
	langStartedAt := time.Now()
	lang, err := prepareLanguageImage(ctx, work, task.Lang, task.Source, task.Limits, task.SubmissionID, task.Attempt, req.Logf)
	logStep(req.Logf, task.SubmissionID, task.Attempt, "prepare_language_image", langStartedAt)
	if err != nil {
		if result, ok := taskResultForLanguageBuildError(task, err); ok {
			return result, nil
		}
		return TaskResult{}, err
	}
	defer lang.Cleanup()

	socket := filepath.Join(work, "runner.sock")
	_ = os.Remove(socket)
	startContainerStartedAt := time.Now()
	containerID, err := startRunnerContainer(ctx, lang.Image, req.Runner, work, socket)
	logStep(req.Logf, task.SubmissionID, task.Attempt, "start_runner_container", startContainerStartedAt)
	if err != nil {
		return TaskResult{}, err
	}
	defer dockerRemoveContainer(context.Background(), containerID)

	pidStartedAt := time.Now()
	initPID, err := dockerContainerPID(ctx, containerID)
	logStep(req.Logf, task.SubmissionID, task.Attempt, "inspect_runner_pid", pidStartedAt)
	if err != nil {
		return TaskResult{}, containerError(ctx, containerID, err)
	}
	socketStartedAt := time.Now()
	if err := waitUnixSocket(ctx, socket); err != nil {
		logStep(req.Logf, task.SubmissionID, task.Attempt, "wait_runner_socket_error", socketStartedAt)
		return TaskResult{}, containerError(ctx, containerID, fmt.Errorf("wait runner socket: %w", err))
	}
	logStep(req.Logf, task.SubmissionID, task.Attempt, "wait_runner_socket", socketStartedAt)
	connectStartedAt := time.Now()
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", socket)
	logStep(req.Logf, task.SubmissionID, task.Attempt, "connect_runner_socket", connectStartedAt)
	if err != nil {
		return TaskResult{}, err
	}
	defer conn.Close()
	codec := NewCodec(conn)
	client := runnerClient{
		codec:      codec,
		cgroupRoot: req.CgroupRoot,
		procRoot:   req.ProcRoot,
		initPID:    initPID,
		taskID:     safeCaseID(fmt.Sprintf("%d-%d", req.Task.SubmissionID, req.Task.Attempt)),
		limits:     req.Task.Limits,
		work:       work,
		logf:       req.Logf,
	}
	return client.runTask(ctx, task, lang.Command)
}

func taskResultForLanguageBuildError(task Task, err error) (TaskResult, bool) {
	var buildErr languageBuildError
	if !errors.As(err, &buildErr) {
		return TaskResult{}, false
	}
	return TaskResult{
		SubmissionID: task.SubmissionID,
		Attempt:      task.Attempt,
		Verdict:      VerdictCompileError,
		Message:      buildErr.Message,
	}, true
}

type runnerClient struct {
	codec      *Codec
	cgroupRoot string
	procRoot   string
	initPID    int
	taskID     string
	limits     Limits
	work       string
	logf       func(format string, args ...any)
}

func (client runnerClient) runTask(ctx context.Context, task Task, userCommand string) (TaskResult, error) {
	result := TaskResult{SubmissionID: task.SubmissionID, Attempt: task.Attempt}
	if len(task.Cases) == 0 {
		result.Verdict = VerdictSystemError
		result.Message = "task has no cases"
		return result, nil
	}
	helloStartedAt := time.Now()
	if err := client.hello(); err != nil {
		return TaskResult{}, err
	}
	logStep(client.logf, task.SubmissionID, task.Attempt, "runner_hello", helloStartedAt)
	compileStartedAt := time.Now()
	compile, err := client.compile(task, userCommand)
	logTask(client.logf, task.SubmissionID, task.Attempt, "compile=%s ok=%t reported=%dms", formatDuration(time.Since(compileStartedAt)), compile.OK, compile.TimeMS)
	if err != nil {
		return TaskResult{}, err
	}
	if !compile.OK {
		result.Verdict = VerdictCompileError
		result.Message = compile.Message
		result.TimeMS = compile.TimeMS
		return result, nil
	}
	judgeCommand := ""
	if task.Mode == ModeCustom {
		var judgeBuild CompileResult
		customStartedAt := time.Now()
		judgeCommand, judgeBuild, err = prepareContainerCustomJudge(ctx, client.work, task.Limits)
		logTask(client.logf, task.SubmissionID, task.Attempt, "custom_judge_build=%s ok=%t reported=%dms", formatDuration(time.Since(customStartedAt)), judgeBuild.OK, judgeBuild.TimeMS)
		if err != nil {
			return TaskResult{}, err
		}
		if !judgeBuild.OK {
			result.Verdict = VerdictSystemError
			result.Message = "custom judge compile failed: " + judgeBuild.Message
			result.TimeMS = judgeBuild.TimeMS
			return result, nil
		}
	}

	startedAt := time.Now()
	totalScore := 0
	verdict := VerdictAccepted
	maxMemory := 0
	caseResults := make([]CaseResult, 0, len(task.Cases))
	for index, item := range task.Cases {
		caseStartedAt := time.Now()
		got, err := client.runCase(ctx, RunCaseRequest{
			TaskID:       client.taskID,
			JudgeCommand: judgeCommand,
			Case:         item,
			Mode:         task.Mode,
			Limits:       task.Limits,
		})
		logTask(client.logf, task.SubmissionID, task.Attempt, "case=%d id=%s elapsed=%s verdict=%s reported=%dms", index+1, item.ID, formatDuration(time.Since(caseStartedAt)), got.Verdict, got.TimeMS)
		if err != nil {
			return TaskResult{}, err
		}
		caseResults = append(caseResults, got)
		totalScore += got.Score
		if got.MemoryKB > maxMemory {
			maxMemory = got.MemoryKB
		}
		if verdict == VerdictAccepted && got.Verdict != VerdictAccepted {
			verdict = got.Verdict
		}
	}
	result.Verdict = verdict
	result.Score = totalScore
	result.TimeMS = int(time.Since(startedAt).Milliseconds())
	result.MemoryKB = maxMemory
	result.Cases = caseResults
	return result, nil
}

func (client runnerClient) hello() error {
	if err := client.codec.Send(Message{Kind: MsgHello, Hello: &Hello{Role: "judger", Version: Version}}); err != nil {
		return err
	}
	msg, err := client.codec.Recv()
	if err != nil {
		return err
	}
	if msg.Kind != MsgHello {
		return fmt.Errorf("runner hello got %s", msg.Kind)
	}
	return nil
}

func (client runnerClient) compile(task Task, userCommand string) (CompileResult, error) {
	if err := client.codec.Send(Message{Kind: MsgCompile, Compile: &CompileRequest{
		TaskID:      client.taskID,
		UserCommand: userCommand,
		Limits:      task.Limits,
	}}); err != nil {
		return CompileResult{}, err
	}
	msg, err := client.codec.Recv()
	if err != nil {
		return CompileResult{}, err
	}
	if msg.Kind == MsgError {
		return CompileResult{}, errors.New(msg.Error)
	}
	if msg.Kind != MsgCompileResult || msg.CompileResult == nil {
		return CompileResult{}, fmt.Errorf("runner compile got %s", msg.Kind)
	}
	return *msg.CompileResult, nil
}

func (client runnerClient) runCase(ctx context.Context, req RunCaseRequest) (CaseResult, error) {
	if err := client.codec.Send(Message{Kind: MsgRunCase, RunCase: &req}); err != nil {
		return CaseResult{}, err
	}
	var cgroup *CgroupCase
	defer func() {
		if cgroup != nil {
			_ = cgroup.Cleanup()
		}
	}()
	for {
		msg, err := client.codec.Recv()
		if err != nil {
			return CaseResult{}, err
		}
		switch msg.Kind {
		case MsgUserPID:
			if msg.UserPID == nil {
				return CaseResult{}, fmt.Errorf("runner sent empty user pid")
			}
			if cgroup == nil && client.cgroupRoot != "" {
				cg, err := client.prepareUserCgroup(msg.UserPID)
				if err != nil {
					if !errors.Is(err, os.ErrNotExist) {
						return CaseResult{}, err
					}
				} else {
					cgroup = cg
				}
			}
			if err := client.codec.Send(Message{Kind: MsgReleaseUser, ReleaseUser: &ReleaseUser{
				TaskID: msg.UserPID.TaskID,
				CaseID: msg.UserPID.CaseID,
			}}); err != nil {
				return CaseResult{}, err
			}
		case MsgCaseResult:
			if msg.CaseResult == nil {
				return CaseResult{}, fmt.Errorf("runner sent empty case result")
			}
			result := *msg.CaseResult
			if cgroup != nil {
				applyCgroupStats(&result, cgroup)
			}
			return result, nil
		case MsgError:
			return CaseResult{}, errors.New(msg.Error)
		default:
			if err := ctx.Err(); err != nil {
				return CaseResult{CaseID: req.Case.ID, Verdict: VerdictTimeLimit, Message: err.Error()}, nil
			}
			return CaseResult{}, fmt.Errorf("runner run_case got %s", msg.Kind)
		}
	}
}

func (client runnerClient) prepareUserCgroup(pid *UserPID) (*CgroupCase, error) {
	hostPID, err := MapInnerPID(client.procRoot, client.initPID, pid.PID)
	if err != nil {
		return nil, err
	}
	cg, err := PrepareCgroup(CgroupConfig{
		Root:         client.cgroupRoot,
		SubmissionID: client.taskID,
		CaseID:       safeCaseID(pid.CaseID),
		MemoryMax:    int64(client.limits.MemoryKB) * 1024,
		PidsMax:      client.limits.Pids,
	})
	if err != nil {
		return nil, err
	}
	if err := cg.Add(hostPID); err != nil {
		_ = cg.Cleanup()
		return nil, err
	}
	return cg, nil
}

func applyCgroupStats(result *CaseResult, cgroup *CgroupCase) {
	stats, err := cgroup.Stats()
	if err != nil {
		return
	}
	applyCgroupStatsSnapshot(result, stats)
}

func applyCgroupStatsSnapshot(result *CaseResult, stats CgroupStats) {
	if stats.MemoryPeak > 0 {
		result.MemoryKB = int((stats.MemoryPeak + 1023) / 1024)
	}
	if stats.MemoryOOM {
		result.Verdict = VerdictMemoryLimit
		result.Score = 0
		result.Message = "memory limit exceeded"
		return
	}
	if stats.PidsMaxed && result.Verdict == VerdictAccepted {
		result.Verdict = VerdictRuntimeError
		result.Score = 0
		result.Message = "process limit exceeded"
	}
}

func startRunnerContainer(ctx context.Context, image string, runner string, work string, socket string) (string, error) {
	_ = os.Remove(socket)
	containerID, err := dockerCreateContainer(ctx, dockerCreateRequest{
		Image: image,
		Cmd: []string{
			"/usr/local/bin/doj-runner", "serve",
			"--socket", filepath.ToSlash(filepath.Join(containerWorkDir, "runner.sock")),
			"--work", containerWorkDir,
			"--runner", "/usr/local/bin/doj-runner",
		},
		WorkingDir: containerWorkDir,
		HostConfig: dockerHostConfig{
			Binds: []string{
				work + ":" + containerWorkDir,
				runner + ":/usr/local/bin/doj-runner:ro",
			},
			NetworkMode: "none",
			SecurityOpt: []string{"no-new-privileges"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("create runner container: %w", err)
	}
	if err := dockerStartContainer(ctx, containerID); err != nil {
		dockerRemoveContainer(context.Background(), containerID)
		return "", fmt.Errorf("start runner container: %w", err)
	}
	return containerID, nil
}

func dockerContainerPID(ctx context.Context, containerID string) (int, error) {
	pid, err := dockerInspectPID(ctx, containerID)
	if err != nil {
		return 0, fmt.Errorf("inspect runner container pid: %w", err)
	}
	return pid, nil
}

func containerError(ctx context.Context, containerID string, err error) error {
	logs := dockerContainerLogs(ctx, containerID)
	if logs == "" {
		return err
	}
	return fmt.Errorf("%w\nrunner container logs:\n%s", err, logs)
}

func dockerContainerLogs(ctx context.Context, containerID string) string {
	logCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return dockerLogs(logCtx, containerID, defaultCompileOutputLimit)
}

func waitUnixSocket(ctx context.Context, socket string) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		if info, err := os.Stat(socket); err == nil && info.Mode()&os.ModeSocket != 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
