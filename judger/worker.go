package judger

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	common "github.com/doveccl/doj/contract/judger"
)

type WorkerConfig struct {
	Server     string
	Token      string
	Runner     string
	Tasks      string
	Cache      string
	CgroupRoot string
	ProcRoot   string
	HTTPClient *http.Client
	Logf       func(format string, args ...any)
	Progress   func(stage string, done int64, total *int64)
}

func RunOne(ctx context.Context, cfg WorkerConfig) (bool, error) {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	leaseStartedAt := time.Now()
	task, err := lease(ctx, client, cfg)
	logStep(cfg.Logf, 0, 0, "lease", leaseStartedAt)
	if err != nil {
		return false, err
	}
	if task == nil {
		return false, nil
	}
	totalStartedAt := time.Now()
	logTask(cfg.Logf, task.SubmissionID, task.Attempt, "start problem=P%d cases=%d lang=%s", task.Problem.ID, len(task.Cases), task.Lang.ID)
	defer func() {
		logTask(cfg.Logf, task.SubmissionID, task.Attempt, "total=%s", formatDuration(time.Since(totalStartedAt)))
	}()

	heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
	defer stopHeartbeat()
	progress := &progressReporter{client: client, cfg: cfg, taskID: task.ID, submissionID: task.SubmissionID, attempt: task.Attempt}
	cfg.Progress = progress.Update
	progress.Set(ctx, "prepare", 0, nil)
	go heartbeatLoop(heartbeatCtx, progress)

	taskStartedAt := time.Now()
	tasksRoot := strings.TrimSpace(cfg.Tasks)
	if tasksRoot == "" {
		return true, fmt.Errorf("task directory is required")
	}
	taskDir := filepath.Join(tasksRoot, strconv.FormatUint(uint64(task.SubmissionID), 10), strconv.Itoa(task.Attempt))
	if err := os.RemoveAll(taskDir); err != nil {
		return true, err
	}
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		return true, err
	}
	defer func() {
		_ = os.RemoveAll(taskDir)
		_ = os.Remove(filepath.Dir(taskDir))
	}()
	logStep(cfg.Logf, task.SubmissionID, task.Attempt, "prepare_task", taskStartedAt)
	if taskNeedsProblemPackage(task) {
		packageStartedAt := time.Now()
		if err := downloadProblemPackage(ctx, client, cfg, task.Problem.ID, task.Problem.PackageHash, taskDir, task.SubmissionID, task.Attempt); err != nil {
			logStep(cfg.Logf, task.SubmissionID, task.Attempt, "download_problem_package_error", packageStartedAt)
			result := TaskResult{
				SubmissionID: task.SubmissionID,
				Attempt:      task.Attempt,
				Verdict:      VerdictSystemError,
				Message:      err.Error(),
			}
			if postErr := postResult(ctx, client, cfg, task.ID, result); postErr != nil {
				return true, postErr
			}
			return true, nil
		}
		logStep(cfg.Logf, task.SubmissionID, task.Attempt, "download_problem_package", packageStartedAt)
	}
	runStartedAt := time.Now()
	result, err := RunContainerTask(ctx, ContainerTask{
		Runner:           cfg.Runner,
		Work:             taskDir,
		CgroupRoot:       cfg.CgroupRoot,
		ProcRoot:         cfg.ProcRoot,
		CustomJudgeCache: customJudgeCachePath(cfg.Cache, task.Problem.ID, JudgeMode(task.Mode)),
		Task:             taskToTask(task),
		Logf:             cfg.Logf,
		Progress:         cfg.Progress,
	})
	logStep(cfg.Logf, task.SubmissionID, task.Attempt, "run_container", runStartedAt)
	if err != nil {
		result = TaskResult{
			SubmissionID: task.SubmissionID,
			Attempt:      task.Attempt,
			Verdict:      VerdictSystemError,
			Message:      err.Error(),
		}
	}
	postStartedAt := time.Now()
	progress.Set(ctx, "upload", 0, nil)
	if err := postResult(ctx, client, cfg, task.ID, result); err != nil {
		return true, err
	}
	logTask(cfg.Logf, task.SubmissionID, task.Attempt, "post_result=%s verdict=%s score=%d", formatDuration(time.Since(postStartedAt)), result.Verdict, result.Score)
	return true, nil
}

func taskNeedsProblemPackage(task *common.TaskPayload) bool {
	if JudgeMode(task.Mode) == ModeCustom {
		return true
	}
	for _, item := range task.Cases {
		if !filepath.IsAbs(item.Input) || !filepath.IsAbs(item.Answer) {
			return true
		}
	}
	return false
}

func taskToTask(task *common.TaskPayload) Task {
	cases := make([]Case, 0, len(task.Cases))
	for _, item := range task.Cases {
		cases = append(cases, Case{
			ID:     item.ID,
			Input:  item.Input,
			Answer: item.Answer,
			Score:  item.Score,
		})
	}
	return Task{
		SubmissionID: task.SubmissionID,
		Attempt:      task.Attempt,
		Source:       task.Source,
		Lang: Lang{
			ID:      task.Lang.ID,
			Source:  task.Lang.Source,
			Image:   task.Lang.Image,
			Compile: task.Lang.Compile,
			Run:     task.Lang.Run,
		},
		Mode: JudgeMode(task.Mode),
		Limits: Limits{
			TimeMS:   task.Limits.TimeMS,
			MemoryKB: task.Limits.MemoryKB,
			OutputKB: task.Limits.OutputKB,
			Pids:     task.Limits.Pids,
			FileKB:   task.Limits.FileKB,
		},
		Cases: cases,
	}
}
