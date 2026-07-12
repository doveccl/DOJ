package judger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	common "github.com/doveccl/doj/contract/judger"
	"github.com/doveccl/doj/judger/runner"
)

func validWorkerTask() *common.TaskPayload {
	return &common.TaskPayload{
		ID: 7, SubmissionID: 11, Attempt: 1, Source: "package main",
		Lang:    common.LangPayload{ID: "go", Source: "main.go", Image: "go:latest", Run: "go run main.go"},
		Mode:    string(runner.ModeDefault),
		Limits:  common.LimitsPayload{TimeMS: 1000, MemoryKB: 64 << 10, OutputKB: 64, Pids: 32, FileKB: 64 << 10},
		Cases:   []common.CasePayload{{ID: "1", Input: "data/1.in", Answer: "data/1.out", Score: 100}},
		Problem: common.ProblemPayload{ID: 1000, PackageHash: "hash"},
	}
}

func TestValidateTaskTrustBoundary(t *testing.T) {
	if err := validateTask(validWorkerTask()); err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*common.TaskPayload){
		"source":       func(task *common.TaskPayload) { task.Source = strings.Repeat("x", (1<<20)+1) },
		"mode":         func(task *common.TaskPayload) { task.Mode = "root" },
		"time":         func(task *common.TaskPayload) { task.Limits.TimeMS = 60_001 },
		"memory":       func(task *common.TaskPayload) { task.Limits.MemoryKB = (4096 << 10) + 1 },
		"output":       func(task *common.TaskPayload) { task.Limits.OutputKB = (64 << 10) + 1 },
		"pids":         func(task *common.TaskPayload) { task.Limits.Pids = 257 },
		"file":         func(task *common.TaskPayload) { task.Limits.FileKB = (1 << 20) + 1 },
		"score":        func(task *common.TaskPayload) { task.Cases[0].Score = 99 },
		"package hash": func(task *common.TaskPayload) { task.Problem.PackageHash = "" },
	}
	for name, breakTask := range tests {
		t.Run(name, func(t *testing.T) {
			task := validWorkerTask()
			breakTask(task)
			if err := validateTask(task); err == nil {
				t.Fatal("invalid task was accepted")
			}
		})
	}
}

func TestValidateTaskCasePaths(t *testing.T) {
	valid := &common.TaskPayload{Cases: []common.CasePayload{{Input: "data/1.in", Answer: "data/1.out"}}}
	if err := validateTaskCasePaths(valid); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"", ".", "/tmp/1.in", "../1.in", "data/../1.in", "data//1.in", `data\..\1.in`} {
		t.Run(path, func(t *testing.T) {
			task := &common.TaskPayload{Cases: []common.CasePayload{{Input: path, Answer: "data/1.out"}}}
			if err := validateTaskCasePaths(task); err == nil {
				t.Fatalf("path %q was accepted", path)
			}
		})
	}
	if err := validateTaskCasePaths(&common.TaskPayload{Cases: []common.CasePayload{{Input: "data/1.in", Answer: "../1.out"}}}); err == nil {
		t.Fatal("unsafe answer path was accepted")
	}
}

func TestValidateWorkerServerTransport(t *testing.T) {
	for _, value := range []string{"http://127.0.0.1:7974", "http://[::1]:7974", "https://judge.example.com"} {
		if err := validateWorkerServer(value); err != nil {
			t.Fatalf("server %q rejected: %v", value, err)
		}
	}
	for _, value := range []string{"http://judge.example.com", "ftp://judge.example.com", "https://user:pass@judge.example.com"} {
		if err := validateWorkerServer(value); err == nil {
			t.Fatalf("unsafe server %q accepted", value)
		}
	}
}

func TestRunOneRejectsUnsafeCasePathBeforeDownload(t *testing.T) {
	packageRequested := false
	var result common.ResultRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/judger/lease":
			_ = json.NewEncoder(w).Encode(common.LeaseResponse{Task: &common.TaskPayload{
				ID: 7, SubmissionID: 11, Attempt: 1,
				Problem: common.ProblemPayload{ID: 1000, PackageHash: "hash"},
				Cases:   []common.CasePayload{{Input: "../secret", Answer: "data/1.out"}},
			}})
		case "/api/judger/tasks/7/package":
			packageRequested = true
			http.Error(w, "unexpected", http.StatusInternalServerError)
		case "/api/judger/tasks/7/result":
			if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
				t.Error(err)
			}
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	worked, err := RunOne(t.Context(), WorkerConfig{Server: server.URL})
	if err != nil || !worked {
		t.Fatalf("RunOne = %t, %v", worked, err)
	}
	if packageRequested || result.Status != string(runner.VerdictSystemError) {
		t.Fatalf("package requested = %t, result = %#v", packageRequested, result)
	}
}

func TestPrivateDirTightensExistingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := privateDir(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("mode = %o, want 700", got)
	}
}
