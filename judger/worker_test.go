//go:build linux

package judger

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	common "github.com/doveccl/doj/common/judger"
)

func TestRunOneLeasesExecutesAndPostsResult(t *testing.T) {
	requireDocker(t)
	runner := buildRunner(t)
	root := t.TempDir()

	gotResult := make(chan resultRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/judger/lease":
			_ = json.NewEncoder(w).Encode(leaseResponse{Task: &leaseTask{
				ID:           7,
				SubmissionID: 11,
				Attempt:      2,
				Source:       "cat\n",
				Lang:         testLeaseLang(),
				Mode:         string(ModeDefault),
				Limits:       common.LimitsPayload{TimeMS: 1000, OutputKB: 64},
				Problem:      taskProblem{ID: 1000, PackageHash: "fixture-v1"},
				Cases: []casePayload{{
					ID:     "1",
					Input:  "data/1.in",
					Answer: "data/1.out",
					Score:  100,
				}},
			}})
		case "/api/judger/P1000.zip":
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(testPackageZip(t))
		case "/api/judger/tasks/7/result":
			var req resultRequest
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
		if result.SubmissionID != 11 || result.Attempt != 2 || result.Status != string(VerdictAccepted) || result.Score != 100 {
			t.Fatalf("result = %#v", result)
		}
		if len(result.Cases) != 1 || result.Cases[0].Status != string(VerdictAccepted) {
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

	gotResult := make(chan resultRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/judger/lease":
			_ = json.NewEncoder(w).Encode(leaseResponse{Task: &leaseTask{
				ID:           8,
				SubmissionID: 12,
				Attempt:      1,
				Source:       "cat\n",
				Lang:         testLeaseLang(),
				Mode:         string(ModeDefault),
				Limits:       common.LimitsPayload{TimeMS: 1000, OutputKB: 64},
				Problem:      taskProblem{ID: 1000, PackageHash: "fixture-v1"},
				Cases: []casePayload{{
					ID:     "1",
					Input:  "data/1.in",
					Answer: "data/1.out",
					Score:  100,
				}},
			}})
		case "/api/judger/P1000.zip":
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(testPackageZip(t))
		case "/api/judger/tasks/8/result":
			var req resultRequest
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
		if result.SubmissionID != 12 || result.Attempt != 1 || result.Status != string(VerdictAccepted) || result.Score != 100 {
			t.Fatalf("result = %#v", result)
		}
	case <-ctx.Done():
		t.Fatal("missing result callback")
	}
}

func TestRunOneCleansWorkAfterPackageError(t *testing.T) {
	root := t.TempDir()
	gotResult := make(chan resultRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/judger/lease":
			_ = json.NewEncoder(w).Encode(leaseResponse{Task: &leaseTask{
				ID:           9,
				SubmissionID: 13,
				Attempt:      1,
				Source:       "cat\n",
				Lang:         testLeaseLang(),
				Mode:         string(ModeDefault),
				Limits:       common.LimitsPayload{TimeMS: 1000, OutputKB: 64},
				Problem:      taskProblem{ID: 1000, PackageHash: "fixture-v1"},
				Cases: []casePayload{{
					ID:     "1",
					Input:  "data/1.in",
					Answer: "data/1.out",
					Score:  100,
				}},
			}})
		case "/api/judger/P1000.zip":
			http.Error(w, "missing", http.StatusNotFound)
		case "/api/judger/tasks/9/result":
			var req resultRequest
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
	if result.Status != string(VerdictSystemError) {
		t.Fatalf("result = %#v", result)
	}
}

func TestExtractProblemPackageRejectsZipSlip(t *testing.T) {
	for _, name := range []string{"../escape.txt", "/absolute.txt", "data/../../escape.txt"} {
		t.Run(name, func(t *testing.T) {
			var body bytes.Buffer
			writer := zip.NewWriter(&body)
			file, err := writer.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.Write([]byte("bad")); err != nil {
				t.Fatal(err)
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			if err := extractProblemPackage(body.Bytes(), t.TempDir()); err == nil {
				t.Fatal("expected zip-slip package to be rejected")
			}
		})
	}
}

func TestDownloadProblemPackageUsesLeasePackageHashCache(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/judger/P1000.zip" {
			http.NotFound(w, r)
			return
		}
		requests++
		switch requests {
		case 1:
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(testPackageZip(t))
		default:
			t.Errorf("unexpected package request %d", requests)
			http.Error(w, "too many requests", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	root := t.TempDir()
	cfg := WorkerConfig{Server: server.URL, Cache: filepath.Join(root, "cache")}
	work1 := filepath.Join(root, "one")
	if err := os.MkdirAll(work1, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := downloadProblemPackage(t.Context(), server.Client(), cfg, 1000, "v1", work1, 11, 1); err != nil {
		t.Fatalf("first download: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(work1, "data", "1.in")); err != nil || string(got) != "42\n" {
		t.Fatalf("first work package file = %q, %v", got, err)
	}
	if got := readProblemCacheHash(filepath.Join(root, "cache", "P1000")); got != "v1" {
		t.Fatalf("problem cache hash = %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "cache", "P1000", "data", "1.in")); err != nil {
		t.Fatalf("problem cache data file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "cache", "P1000", "package")); !os.IsNotExist(err) {
		t.Fatalf("problem cache should not contain package directory, err=%v", err)
	}

	work2 := filepath.Join(root, "two")
	if err := os.MkdirAll(work2, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := downloadProblemPackage(t.Context(), server.Client(), cfg, 1000, "v1", work2, 12, 1); err != nil {
		t.Fatalf("cached download: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(work2, "data", "1.out")); err != nil || string(got) != "42\n" {
		t.Fatalf("cached work package file = %q, %v", got, err)
	}
	if requests != 1 {
		t.Fatalf("package requests = %d", requests)
	}
}

func TestDownloadProblemPackageSharesConcurrentDownload(t *testing.T) {
	var mu sync.Mutex
	requests := 0
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/judger/P1000.zip" {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		requests++
		got := requests
		mu.Unlock()
		entered <- struct{}{}
		if got > 1 {
			t.Errorf("unexpected concurrent package request %d", got)
			http.Error(w, "too many requests", http.StatusInternalServerError)
			return
		}
		<-release
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(testPackageZip(t))
	}))
	defer server.Close()

	root := t.TempDir()
	cfg := WorkerConfig{Server: server.URL, Cache: filepath.Join(root, "cache")}
	errCh := make(chan error, 2)
	for index := 0; index < 2; index++ {
		go func(index int) {
			work := filepath.Join(root, "work", strconv.Itoa(index))
			if err := os.MkdirAll(work, 0o755); err != nil {
				errCh <- err
				return
			}
			errCh <- downloadProblemPackage(t.Context(), server.Client(), cfg, 1000, "v1", work, uint(index+1), 1)
		}(index)
	}
	<-entered
	close(release)
	for index := 0; index < 2; index++ {
		if err := <-errCh; err != nil {
			t.Fatalf("download %d: %v", index, err)
		}
	}
	mu.Lock()
	gotRequests := requests
	mu.Unlock()
	if gotRequests != 1 {
		t.Fatalf("package requests = %d", gotRequests)
	}
	for index := 0; index < 2; index++ {
		got, err := os.ReadFile(filepath.Join(root, "work", strconv.Itoa(index), "data", "1.in"))
		if err != nil || string(got) != "42\n" {
			t.Fatalf("work %d package file = %q, %v", index, got, err)
		}
	}
}

func TestDownloadProblemPackageReportsDownloadedBytes(t *testing.T) {
	body := testPackageZip(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/judger/P1000.zip" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	root := t.TempDir()
	work := filepath.Join(root, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	var got []heartbeatRequest
	cfg := WorkerConfig{
		Server: server.URL,
		Cache:  filepath.Join(root, "cache"),
		Progress: func(stage string, done int64, total *int64) {
			got = append(got, heartbeatRequest{Stage: stage, Done: done, Total: total})
		},
	}
	if err := downloadProblemPackage(t.Context(), server.Client(), cfg, 1000, "v1", work, 11, 1); err != nil {
		t.Fatalf("download: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("missing download progress")
	}
	last := got[len(got)-1]
	if last.Stage != "download" || last.Done != int64(len(body)) || last.Total != nil {
		t.Fatalf("download progress = %+v, want done=%d without total", last, len(body))
	}
}

func TestDownloadProblemPackageUsesLongerTimeoutThanAPIClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/judger/P1000.zip" {
			http.NotFound(w, r)
			return
		}
		time.Sleep(20 * time.Millisecond)
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(testPackageZip(t))
	}))
	defer server.Close()

	root := t.TempDir()
	work := filepath.Join(root, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: time.Nanosecond}
	cfg := WorkerConfig{Server: server.URL, Cache: filepath.Join(root, "cache")}
	if err := downloadProblemPackage(t.Context(), client, cfg, 1000, "v1", work, 11, 1); err != nil {
		t.Fatalf("download should not use short API timeout: %v", err)
	}
}

func testPackageZip(t *testing.T) []byte {
	t.Helper()
	var body bytes.Buffer
	writer := zip.NewWriter(&body)
	for name, content := range map[string]string{
		"data/1.in":  "42\n",
		"data/1.out": "42\n",
	} {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip file failed: %v", err)
		}
		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatalf("write zip file failed: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip failed: %v", err)
	}
	return body.Bytes()
}

func testLeaseLang() common.LangPayload {
	lang := testShellLang()
	return common.LangPayload{
		ID:      lang.ID,
		Source:  lang.Source,
		Image:   lang.Image,
		Compile: lang.Compile,
		Run:     lang.Run,
	}
}
