//go:build linux

package judger

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	common "github.com/doveccl/doj/contract/judger"
	jr "github.com/doveccl/doj/judger/runner"
)

func TestRunOneLeasesExecutesAndPostsResult(t *testing.T) {
	requireDocker(t)
	runner := buildRunner(t)
	root := t.TempDir()

	gotResult := make(chan common.ResultRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/judger/lease":
			_ = json.NewEncoder(w).Encode(common.LeaseResponse{Task: &common.TaskPayload{
				ID:           7,
				SubmissionID: 11,
				Attempt:      2,
				Source:       "cat\n",
				Lang:         testLeaseLang(),
				Mode:         string(jr.ModeDefault),
				Limits:       common.LimitsPayload{TimeMS: 1000, MemoryKB: 64 << 10, OutputKB: 64, Pids: 32, FileKB: 64 << 10},
				Problem:      common.ProblemPayload{ID: 1000, PackageHash: "fixture-v1"},
				Cases: []common.CasePayload{{
					ID:     "1",
					Input:  "data/1.in",
					Answer: "data/1.out",
					Score:  100,
				}},
			}})
		case "/api/judger/tasks/7/package":
			if r.URL.Query().Get("attempt") != "2" || r.URL.Query().Get("hash") != "fixture-v1" {
				http.Error(w, "bad package lease", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(testPackageZip(t))
		case "/api/judger/tasks/7/result":
			var req common.ResultRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			gotResult <- req
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	worked, err := RunOne(ctx, WorkerConfig{
		Server: server.URL,
		Token:  "secret",
		Runner: runner,
		Tasks:  filepath.Join(root, "tasks"),
		Cache:  filepath.Join(root, "cache"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !worked {
		t.Fatal("expected task")
	}
	if _, err := os.Stat(filepath.Join(root, "tasks", "11")); !os.IsNotExist(err) {
		t.Fatalf("task directory should be cleaned after task, err=%v", err)
	}

	select {
	case result := <-gotResult:
		if result.SubmissionID != 11 || result.Attempt != 2 || result.Status != string(jr.VerdictAccepted) || result.Score != 100 {
			t.Fatalf("result = %#v", result)
		}
		if len(result.Cases) != 1 || result.Cases[0].Status != string(jr.VerdictAccepted) {
			t.Fatalf("cases = %#v", result.Cases)
		}
	case <-ctx.Done():
		t.Fatal("missing result callback")
	}
}

func TestRunOneDownloadsPackageForRelativeCases(t *testing.T) {
	requireDocker(t)
	runner := buildRunner(t)
	root := t.TempDir()

	gotResult := make(chan common.ResultRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/judger/lease":
			_ = json.NewEncoder(w).Encode(common.LeaseResponse{Task: &common.TaskPayload{
				ID:           8,
				SubmissionID: 12,
				Attempt:      1,
				Source:       "cat\n",
				Lang:         testLeaseLang(),
				Mode:         string(jr.ModeDefault),
				Limits:       common.LimitsPayload{TimeMS: 1000, MemoryKB: 64 << 10, OutputKB: 64, Pids: 32, FileKB: 64 << 10},
				Problem:      common.ProblemPayload{ID: 1000, PackageHash: "fixture-v1"},
				Cases: []common.CasePayload{{
					ID:     "1",
					Input:  "data/1.in",
					Answer: "data/1.out",
					Score:  100,
				}},
			}})
		case "/api/judger/tasks/8/package":
			if r.URL.Query().Get("attempt") != "1" || r.URL.Query().Get("hash") != "fixture-v1" {
				http.Error(w, "bad package lease", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(testPackageZip(t))
		case "/api/judger/tasks/8/result":
			var req common.ResultRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			gotResult <- req
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	worked, err := RunOne(ctx, WorkerConfig{
		Server: server.URL,
		Token:  "secret",
		Runner: runner,
		Tasks:  filepath.Join(root, "tasks"),
		Cache:  filepath.Join(root, "cache"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !worked {
		t.Fatal("expected task")
	}
	if _, err := os.Stat(filepath.Join(root, "tasks", "12")); !os.IsNotExist(err) {
		t.Fatalf("task directory should be cleaned after task, err=%v", err)
	}

	select {
	case result := <-gotResult:
		if result.SubmissionID != 12 || result.Attempt != 1 || result.Status != string(jr.VerdictAccepted) || result.Score != 100 {
			t.Fatalf("result = %#v", result)
		}
	case <-ctx.Done():
		t.Fatal("missing result callback")
	}
}

func TestRunOneCleansWorkAfterPackageError(t *testing.T) {
	root := t.TempDir()
	gotResult := make(chan common.ResultRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/judger/lease":
			_ = json.NewEncoder(w).Encode(common.LeaseResponse{Task: &common.TaskPayload{
				ID:           9,
				SubmissionID: 13,
				Attempt:      1,
				Source:       "cat\n",
				Lang:         testLeaseLang(),
				Mode:         string(jr.ModeDefault),
				Limits:       common.LimitsPayload{TimeMS: 1000, MemoryKB: 64 << 10, OutputKB: 64, Pids: 32, FileKB: 64 << 10},
				Problem:      common.ProblemPayload{ID: 1000, PackageHash: "fixture-v1"},
				Cases: []common.CasePayload{{
					ID:     "1",
					Input:  "data/1.in",
					Answer: "data/1.out",
					Score:  100,
				}},
			}})
		case "/api/judger/tasks/9/package":
			http.Error(w, "missing", http.StatusNotFound)
		case "/api/judger/tasks/9/result":
			var req common.ResultRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error(err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			gotResult <- req
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	worked, err := RunOne(t.Context(), WorkerConfig{
		Server: server.URL,
		Tasks:  filepath.Join(root, "tasks"),
		Cache:  filepath.Join(root, "cache"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !worked {
		t.Fatal("expected task")
	}
	if _, err := os.Stat(filepath.Join(root, "tasks", "13")); !os.IsNotExist(err) {
		t.Fatalf("task directory should be cleaned after package error, err=%v", err)
	}
	result := <-gotResult
	if result.Status != string(jr.VerdictSystemError) {
		t.Fatalf("result = %#v", result)
	}
}

func testLeaseLang() common.LangPayload {
	lang := testShellLang()
	return common.LangPayload{
		ID:        lang.ID,
		Source:    lang.Source,
		Image:     lang.Image,
		Compile:   lang.Compile,
		CompileMS: 10000,
		Run:       lang.Run,
	}
}
