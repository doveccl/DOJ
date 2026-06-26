package judger

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type WorkerConfig struct {
	Server     string
	Token      string
	Runner     string
	Work       string
	CgroupRoot string
	ProcRoot   string
	HTTPClient *http.Client
	Logf       func(format string, args ...any)
}

type LoopConfig struct {
	Worker      WorkerConfig
	Concurrency int
	Logf        func(format string, args ...any)
}

type leaseRequest struct {
	Version string `json:"version"`
	Host    string `json:"host"`
	Arch    string `json:"arch"`
}

type leaseResponse struct {
	Task *leaseTask `json:"task"`
}

type leaseTask struct {
	ID           uint          `json:"id"`
	SubmissionID uint          `json:"submissionId"`
	Attempt      int           `json:"attempt"`
	Source       string        `json:"source"`
	Lang         Lang          `json:"lang"`
	Mode         JudgeMode     `json:"mode"`
	Limits       Limits        `json:"limits"`
	Cases        []casePayload `json:"cases"`
	Problem      taskProblem   `json:"problem"`
}

type taskProblem struct {
	ID             uint   `json:"id"`
	PackageVersion string `json:"packageVersion"`
}

type casePayload struct {
	ID     string `json:"id"`
	Input  string `json:"input"`
	Answer string `json:"answer"`
	Score  int    `json:"score"`
}

type resultRequest struct {
	SubmissionID uint         `json:"submissionId"`
	Attempt      int          `json:"attempt"`
	Status       string       `json:"status"`
	Score        int          `json:"score"`
	Message      string       `json:"message"`
	TimeMS       *int         `json:"timeMs,omitempty"`
	MemoryKB     *int         `json:"memoryKb,omitempty"`
	Cases        []caseResult `json:"cases"`
}

type heartbeatRequest struct {
	SubmissionID uint `json:"submissionId"`
	Attempt      int  `json:"attempt"`
}

type caseResult struct {
	No       int    `json:"no"`
	Status   string `json:"status"`
	Score    int    `json:"score"`
	TimeMS   *int   `json:"timeMs,omitempty"`
	MemoryKB *int   `json:"memoryKb,omitempty"`
	Message  string `json:"message"`
}

func RunOne(ctx context.Context, cfg WorkerConfig) (bool, error) {
	client := workerClient(cfg)
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
	go heartbeatLoop(heartbeatCtx, client, cfg, task.ID, task.SubmissionID, task.Attempt)

	workStartedAt := time.Now()
	work := filepath.Join(cfg.Work, strconv.FormatUint(uint64(task.SubmissionID), 10), strconv.Itoa(task.Attempt))
	if err := os.RemoveAll(work); err != nil {
		return true, err
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		return true, err
	}
	logStep(cfg.Logf, task.SubmissionID, task.Attempt, "prepare_work", workStartedAt)
	if task.needsAssets() {
		assetsStartedAt := time.Now()
		if err := downloadTaskAssets(ctx, client, cfg, task.Problem.ID, task.Problem.PackageVersion, work, task.SubmissionID, task.Attempt); err != nil {
			logStep(cfg.Logf, task.SubmissionID, task.Attempt, "download_assets_error", assetsStartedAt)
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
		logStep(cfg.Logf, task.SubmissionID, task.Attempt, "download_assets", assetsStartedAt)
	}
	runStartedAt := time.Now()
	result, err := RunContainerTask(ctx, ContainerTask{
		Runner:     cfg.Runner,
		Work:       work,
		CgroupRoot: cfg.CgroupRoot,
		ProcRoot:   cfg.ProcRoot,
		Task:       task.toTask(),
		Logf:       cfg.Logf,
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
	if err := postResult(ctx, client, cfg, task.ID, result); err != nil {
		return true, err
	}
	logTask(cfg.Logf, task.SubmissionID, task.Attempt, "post_result=%s verdict=%s score=%d", formatDuration(time.Since(postStartedAt)), result.Verdict, result.Score)
	return true, nil
}

func (task leaseTask) needsAssets() bool {
	for _, item := range task.Cases {
		if !filepath.IsAbs(item.Input) || !filepath.IsAbs(item.Answer) {
			return true
		}
	}
	return false
}

func heartbeatLoop(ctx context.Context, client *http.Client, cfg WorkerConfig, taskID uint, submissionID uint, attempt int) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = postHeartbeat(ctx, client, cfg, taskID, submissionID, attempt)
		}
	}
}

func RunLoop(ctx context.Context, cfg LoopConfig) error {
	if cfg.Worker.Logf == nil {
		cfg.Worker.Logf = cfg.Logf
	}
	workers := cfg.Concurrency
	if workers <= 1 {
		return runLoopWorker(ctx, cfg)
	}
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func() {
			errCh <- runLoopWorker(ctx, cfg)
		}()
	}
	return <-errCh
}

func runLoopWorker(ctx context.Context, cfg LoopConfig) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		worked, err := RunOne(ctx, cfg.Worker)
		if err != nil {
			if cfg.Logf != nil {
				cfg.Logf("judger task failed: %v", err)
			}
			if err := sleepContext(ctx, time.Second); err != nil {
				return err
			}
			continue
		}
		if !worked {
			continue
		}
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func logStep(logf func(format string, args ...any), submissionID uint, attempt int, step string, startedAt time.Time) {
	logTask(logf, submissionID, attempt, "%s=%s", step, formatDuration(time.Since(startedAt)))
}

func logTask(logf func(format string, args ...any), submissionID uint, attempt int, format string, args ...any) {
	if logf == nil {
		return
	}
	prefix := fmt.Sprintf("judger timing submission=%d attempt=%d ", submissionID, attempt)
	logf(prefix+format, args...)
}

func formatDuration(d time.Duration) string {
	return d.Round(time.Millisecond).String()
}

func (task leaseTask) toTask() Task {
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
		Lang:         task.Lang,
		Mode:         task.Mode,
		Limits:       task.Limits,
		Cases:        cases,
	}
}

func lease(ctx context.Context, client *http.Client, cfg WorkerConfig) (*leaseTask, error) {
	host, _ := os.Hostname()
	req := leaseRequest{
		Version: Version,
		Host:    host,
		Arch:    runtime.GOOS + "/" + runtime.GOARCH,
	}
	var resp leaseResponse
	if err := doJSON(ctx, client, cfg, http.MethodPost, "/api/judger/lease", req, &resp); err != nil {
		return nil, err
	}
	return resp.Task, nil
}

func postResult(ctx context.Context, client *http.Client, cfg WorkerConfig, taskID uint, result TaskResult) error {
	req := resultRequest{
		SubmissionID: result.SubmissionID,
		Attempt:      result.Attempt,
		Status:       string(result.Verdict),
		Score:        result.Score,
		Message:      result.Message,
		Cases:        make([]caseResult, 0, len(result.Cases)),
	}
	if result.TimeMS > 0 {
		req.TimeMS = &result.TimeMS
	}
	if result.MemoryKB > 0 {
		req.MemoryKB = &result.MemoryKB
	}
	for index, item := range result.Cases {
		got := caseResult{
			No:      index + 1,
			Status:  string(item.Verdict),
			Score:   item.Score,
			Message: item.Message,
		}
		if item.TimeMS > 0 {
			got.TimeMS = &item.TimeMS
		}
		if item.MemoryKB > 0 {
			got.MemoryKB = &item.MemoryKB
		}
		req.Cases = append(req.Cases, got)
	}
	return doJSON(ctx, client, cfg, http.MethodPost, fmt.Sprintf("/api/judger/tasks/%d/result", taskID), req, nil)
}

func postHeartbeat(ctx context.Context, client *http.Client, cfg WorkerConfig, taskID uint, submissionID uint, attempt int) error {
	req := heartbeatRequest{SubmissionID: submissionID, Attempt: attempt}
	return doJSON(ctx, client, cfg, http.MethodPost, fmt.Sprintf("/api/judger/tasks/%d/heartbeat", taskID), req, nil)
}

func downloadTaskAssets(ctx context.Context, client *http.Client, cfg WorkerConfig, problemID uint, packageVersion string, work string, submissionID uint, attempt int) error {
	cacheDir := taskAssetCacheDir(cfg, problemID)
	cacheFilesDir := filepath.Join(cacheDir, "files")
	cacheVersion := readTaskAssetCacheVersion(cacheDir)
	cacheReady := cacheVersion != "" && cacheVersion == strings.TrimSpace(packageVersion) && dirExists(cacheFilesDir)
	logTask(cfg.Logf, submissionID, attempt, "asset_cache ready=%t version=%s dir=%s", cacheReady, shortVersion(cacheVersion), cacheDir)
	if cacheReady {
		copyStartedAt := time.Now()
		if err := copyDir(cacheFilesDir, work); err != nil {
			return err
		}
		logTask(cfg.Logf, submissionID, attempt, "asset_cache_copy=%s source=hit", formatDuration(time.Since(copyStartedAt)))
		return nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.Server+fmt.Sprintf("/api/judger/P%d.zip", problemID), nil)
	if err != nil {
		return err
	}
	if cfg.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	requestStartedAt := time.Now()
	httpResp, err := client.Do(httpReq)
	logTask(cfg.Logf, submissionID, attempt, "asset_package_request=%s status=%s cache_version=%s lease_version=%s", formatDuration(time.Since(requestStartedAt)), responseStatus(httpResp), shortVersion(cacheVersion), shortVersion(packageVersion))
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("GET problem package returned %s", httpResp.Status)
	}
	const maxTaskAssetBytes = 512 << 20
	readStartedAt := time.Now()
	data, err := io.ReadAll(io.LimitReader(httpResp.Body, maxTaskAssetBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxTaskAssetBytes {
		return fmt.Errorf("task assets exceed %d bytes", maxTaskAssetBytes)
	}
	logTask(cfg.Logf, submissionID, attempt, "asset_package_read=%s bytes=%d version=%s", formatDuration(time.Since(readStartedAt)), len(data), shortVersion(packageVersion))
	cacheStartedAt := time.Now()
	if err := replaceTaskAssetCache(cacheDir, data, packageVersion); err != nil {
		return err
	}
	logTask(cfg.Logf, submissionID, attempt, "asset_cache_write=%s", formatDuration(time.Since(cacheStartedAt)))
	copyStartedAt := time.Now()
	if err := copyDir(cacheFilesDir, work); err != nil {
		return err
	}
	logTask(cfg.Logf, submissionID, attempt, "asset_cache_copy=%s source=download", formatDuration(time.Since(copyStartedAt)))
	return nil
}

func taskAssetCacheDir(cfg WorkerConfig, problemID uint) string {
	root := strings.TrimSpace(cfg.Work)
	if root == "" {
		root = os.TempDir()
	}
	return filepath.Join(root, "asset-cache", fmt.Sprintf("P%d", problemID))
}

func readTaskAssetCacheVersion(cacheDir string) string {
	raw, err := os.ReadFile(filepath.Join(cacheDir, "version"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func replaceTaskAssetCache(cacheDir string, data []byte, version string) error {
	version = strings.TrimSpace(version)
	if version == "" {
		return fmt.Errorf("problem package version is required")
	}
	parent := filepath.Dir(cacheDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, ".asset-cache-")
	if err != nil {
		return err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(tmp)
		}
	}()
	filesDir := filepath.Join(tmp, "files")
	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		return err
	}
	if err := extractTaskAssets(data, filesDir); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmp, "version"), []byte(version+"\n"), 0o644); err != nil {
		return err
	}
	if err := os.RemoveAll(cacheDir); err != nil {
		return err
	}
	if err := os.Rename(tmp, cacheDir); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func copyDir(src string, dst string) error {
	return filepath.WalkDir(src, func(file string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, file)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(file)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
		if err != nil {
			_ = in.Close()
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			_ = in.Close()
			_ = out.Close()
			return err
		}
		if err := in.Close(); err != nil {
			_ = out.Close()
			return err
		}
		return out.Close()
	})
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func responseStatus(resp *http.Response) string {
	if resp == nil {
		return "-"
	}
	return resp.Status
}

func shortVersion(version string) string {
	clean := strings.Trim(strings.TrimSpace(version), `"`)
	if len(clean) <= 12 {
		return clean
	}
	return clean[:12]
}

func extractTaskAssets(data []byte, work string) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, file := range reader.File {
		name := filepath.Clean(strings.ReplaceAll(file.Name, "\\", "/"))
		if name == "." || name == ".." || filepath.IsAbs(name) || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			return fmt.Errorf("invalid asset path %q", file.Name)
		}
		target := filepath.Join(work, filepath.FromSlash(name))
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode().Perm())
		if err != nil {
			_ = src.Close()
			return err
		}
		if _, err := io.Copy(dst, src); err != nil {
			_ = src.Close()
			_ = dst.Close()
			return err
		}
		if err := src.Close(); err != nil {
			_ = dst.Close()
			return err
		}
		if err := dst.Close(); err != nil {
			return err
		}
	}
	return nil
}

func doJSON(ctx context.Context, client *http.Client, cfg WorkerConfig, method string, path string, in any, out any) error {
	var body bytes.Buffer
	if in != nil {
		if err := json.NewEncoder(&body).Encode(in); err != nil {
			return err
		}
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, cfg.Server+path, &body)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if cfg.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.Token)
	}

	httpResp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("%s %s returned %s", method, path, httpResp.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(httpResp.Body).Decode(out)
}

func workerClient(cfg WorkerConfig) *http.Client {
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}
