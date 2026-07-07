package judger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	common "github.com/doveccl/doj/common/judger"
)

const (
	defaultHTTPTimeout        = 30 * time.Second
	defaultPackageHTTPTimeout = 5 * time.Minute
)

type leaseRequest = common.LeaseRequest
type leaseResponse = common.LeaseResponse
type leaseTask = common.TaskPayload
type taskProblem = common.ProblemPayload
type casePayload = common.CasePayload
type resultRequest = common.ResultRequest
type heartbeatRequest = common.HeartbeatRequest
type caseResult = common.CaseResult

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

func postProgressHeartbeat(ctx context.Context, client *http.Client, cfg WorkerConfig, taskID uint, submissionID uint, attempt int, progress taskProgress) error {
	req := heartbeatRequest{SubmissionID: submissionID, Attempt: attempt, Stage: progress.stage, Done: progress.done, Total: progress.total}
	return doJSON(ctx, client, cfg, http.MethodPost, fmt.Sprintf("/api/judger/tasks/%d/heartbeat", taskID), req, nil)
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
