package judger

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	common "github.com/doveccl/doj/contract/judger"
	contractlimits "github.com/doveccl/doj/contract/limits"
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
	if err := validateWorkerServer(cfg.Server); err != nil {
		return false, err
	}
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
	if err := validateTask(task); err != nil {
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
	if err := privateDir(tasksRoot); err != nil {
		return true, err
	}
	taskDir := filepath.Join(tasksRoot, strconv.FormatUint(uint64(task.SubmissionID), 10), strconv.Itoa(task.Attempt))
	if err := os.RemoveAll(taskDir); err != nil {
		return true, err
	}
	if err := privateDir(taskDir); err != nil {
		return true, err
	}
	defer func() {
		_ = os.RemoveAll(taskDir)
		_ = os.Remove(filepath.Dir(taskDir))
	}()
	logStep(cfg.Logf, task.SubmissionID, task.Attempt, "prepare_task", taskStartedAt)
	if taskNeedsProblemPackage(task) {
		packageStartedAt := time.Now()
		if err := downloadProblemPackage(ctx, client, cfg, task.Problem.ID, task.Problem.PackageHash, taskDir, task.ID, task.SubmissionID, task.Attempt); err != nil {
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

func validateWorkerServer(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Hostname() == "" || parsed.User != nil {
		return fmt.Errorf("invalid judger server URL")
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme != "http" || (parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "::1") {
		return fmt.Errorf("remote judger server must use HTTPS")
	}
	return nil
}

func taskNeedsProblemPackage(task *common.TaskPayload) bool {
	return JudgeMode(task.Mode) == ModeCustom || len(task.Cases) > 0
}

const (
	maxTaskCaseIDBytes   = 128
	maxTaskCasePathBytes = 1024
	maxTaskImageBytes    = 512
)

func validateTask(task *common.TaskPayload) error {
	if task.ID == 0 || task.SubmissionID == 0 || task.Attempt <= 0 || task.Problem.ID == 0 {
		return fmt.Errorf("task identity is invalid")
	}
	if task.Mode != string(ModeDefault) && task.Mode != string(ModeStrict) && task.Mode != string(ModeCustom) {
		return fmt.Errorf("judge mode is invalid")
	}
	if len(task.Source) > contractlimits.MaxSourceBytes {
		return fmt.Errorf("source exceeds %d bytes", contractlimits.MaxSourceBytes)
	}
	if task.Lang.ID == "" || len(task.Lang.ID) > 32 || len(task.Lang.Source) > 128 || len(task.Lang.Image) > maxTaskImageBytes {
		return fmt.Errorf("language metadata is invalid")
	}
	if strings.TrimSpace(task.Lang.Image) == "" || strings.TrimSpace(task.Lang.Run) == "" {
		return fmt.Errorf("language image and run command are required")
	}
	if len(task.Lang.Compile) > contractlimits.MaxLanguageCommandBytes || len(task.Lang.Run) > contractlimits.MaxLanguageCommandBytes {
		return fmt.Errorf("language command is too large")
	}
	if _, err := cleanLanguageSource(task.Lang.Source); err != nil {
		return err
	}
	if task.Limits.TimeMS <= 0 || task.Limits.TimeMS > contractlimits.MaxProblemTimeMS ||
		task.Limits.MemoryKB <= 0 || task.Limits.MemoryKB > contractlimits.MaxProblemMemoryMB*1024 ||
		task.Limits.OutputKB <= 0 || task.Limits.OutputKB > contractlimits.MaxJudgerOutputKB ||
		task.Limits.Pids <= 0 || task.Limits.Pids > contractlimits.MaxJudgerPids ||
		task.Limits.FileKB <= 0 || task.Limits.FileKB > contractlimits.MaxJudgerFileKB {
		return fmt.Errorf("task resource limits are invalid")
	}
	if len(task.Cases) == 0 || len(task.Cases) > contractlimits.MaxJudgerCases {
		return fmt.Errorf("task case count is invalid")
	}
	if task.Problem.PackageHash == "" || len(task.Problem.PackageHash) > 128 {
		return fmt.Errorf("problem package hash is invalid")
	}
	seen := make(map[string]struct{}, len(task.Cases))
	score := 0
	for index, item := range task.Cases {
		if item.ID == "" || len(item.ID) > maxTaskCaseIDBytes {
			return fmt.Errorf("case %d id is invalid", index+1)
		}
		if _, ok := seen[item.ID]; ok {
			return fmt.Errorf("case %d id is duplicated", index+1)
		}
		seen[item.ID] = struct{}{}
		if item.Score < 0 || item.Score > 100 {
			return fmt.Errorf("case %d score is invalid", index+1)
		}
		score += item.Score
	}
	if score != 100 {
		return fmt.Errorf("case scores must total 100")
	}
	return validateTaskCasePaths(task)
}

func validateTaskCasePaths(task *common.TaskPayload) error {
	for index, item := range task.Cases {
		for _, file := range []struct {
			kind string
			path string
		}{{"input", item.Input}, {"answer", item.Answer}} {
			if file.path == "." || len(file.path) > maxTaskCasePathBytes || !filepath.IsLocal(file.path) || filepath.Clean(file.path) != file.path || strings.ContainsRune(file.path, '\\') {
				return fmt.Errorf("case %d %s must be a clean relative path", index+1, file.kind)
			}
		}
	}
	return nil
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
