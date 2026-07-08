//go:build linux

package judger

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	common "github.com/doveccl/doj/contract/judger"
)

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
	var got []common.HeartbeatRequest
	cfg := WorkerConfig{
		Server: server.URL,
		Cache:  filepath.Join(root, "cache"),
		Progress: func(stage string, done int64, total *int64) {
			got = append(got, common.HeartbeatRequest{Stage: stage, Done: done, Total: total})
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
