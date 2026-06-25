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
	ID uint `json:"id"`
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
	task, err := lease(ctx, client, cfg)
	if err != nil {
		return false, err
	}
	if task == nil {
		return false, nil
	}

	heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
	defer stopHeartbeat()
	go heartbeatLoop(heartbeatCtx, client, cfg, task.ID, task.SubmissionID, task.Attempt)

	work := filepath.Join(cfg.Work, strconv.FormatUint(uint64(task.SubmissionID), 10), strconv.Itoa(task.Attempt))
	if err := os.RemoveAll(work); err != nil {
		return true, err
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		return true, err
	}
	if task.needsAssets() {
		if err := downloadTaskAssets(ctx, client, cfg, task.Problem.ID, work); err != nil {
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
	}
	result, err := RunContainerTask(ctx, ContainerTask{
		Runner:     cfg.Runner,
		Work:       work,
		CgroupRoot: cfg.CgroupRoot,
		ProcRoot:   cfg.ProcRoot,
		Task:       task.toTask(),
	})
	if err != nil {
		result = TaskResult{
			SubmissionID: task.SubmissionID,
			Attempt:      task.Attempt,
			Verdict:      VerdictSystemError,
			Message:      err.Error(),
		}
	}
	if err := postResult(ctx, client, cfg, task.ID, result); err != nil {
		return true, err
	}
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

func downloadTaskAssets(ctx context.Context, client *http.Client, cfg WorkerConfig, problemID uint, work string) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.Server+fmt.Sprintf("/api/judger/P%d.zip", problemID), nil)
	if err != nil {
		return err
	}
	if cfg.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("GET problem package returned %s", httpResp.Status)
	}
	const maxTaskAssetBytes = 512 << 20
	data, err := io.ReadAll(io.LimitReader(httpResp.Body, maxTaskAssetBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxTaskAssetBytes {
		return fmt.Errorf("task assets exceed %d bytes", maxTaskAssetBytes)
	}
	return extractTaskAssets(data, work)
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
