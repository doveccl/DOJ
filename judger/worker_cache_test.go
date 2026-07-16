//go:build linux

package judger

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	common "github.com/doveccl/doj/contract/judger"
)

var (
	testPackageBody = buildTestPackageZip()
	testPackageHash = sha256Hex(testPackageBody)
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

func TestProblemPackageRejectsOversizedExpansion(t *testing.T) {
	file := &zip.File{FileHeader: zip.FileHeader{Name: "data/huge.in", UncompressedSize64: maxProblemPackageFileBytes + 1}}
	if err := validateProblemPackageFiles([]*zip.File{file}); err == nil {
		t.Fatal("oversized expanded file was accepted")
	}
}

func TestDownloadProblemPackageUsesZipHashCache(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !validTestPackageRequest(r) {
			http.NotFound(w, r)
			return
		}
		requests++
		if r.Header.Get("If-None-Match") != "" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if requests != 1 {
			t.Errorf("unexpected package download %d", requests)
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(testPackageZip(t))
	}))
	defer server.Close()

	root := t.TempDir()
	cfg := WorkerConfig{Server: server.URL, Cache: filepath.Join(root, "cache")}
	work1 := filepath.Join(root, "one")
	if err := os.MkdirAll(work1, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := downloadProblemPackage(t.Context(), server.Client(), cfg, 1000, testPackageHash, 0, testPackageFiles(), work1, 11, 11, 1); err != nil {
		t.Fatalf("first download: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(work1, "data", "1.in")); err != nil || string(got) != "42\n" {
		t.Fatalf("first work package file = %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(root, "cache", "P1000", testPackageHash, "data", "1.in")); err != nil {
		t.Fatalf("problem cache data file missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "cache", "P1000", testPackageHash, "package")); !os.IsNotExist(err) {
		t.Fatalf("problem cache should not contain package directory, err=%v", err)
	}

	work2 := filepath.Join(root, "two")
	if err := os.MkdirAll(work2, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := downloadProblemPackage(t.Context(), server.Client(), cfg, 1000, testPackageHash, 0, testPackageFiles(), work2, 12, 12, 1); err != nil {
		t.Fatalf("cached download: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(work2, "data", "1.out")); err != nil || string(got) != "42\n" {
		t.Fatalf("cached work package file = %q, %v", got, err)
	}
	if requests != 2 {
		t.Fatalf("package authorization requests = %d", requests)
	}
}

func TestPackageFileListChangeReusesSameZip(t *testing.T) {
	downloads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-None-Match") != "" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		downloads++
		_, _ = w.Write(testPackageZip(t))
	}))
	defer server.Close()
	root := t.TempDir()
	cfg := WorkerConfig{Server: server.URL, Cache: filepath.Join(root, "cache")}
	first := filepath.Join(root, "first")
	_ = os.MkdirAll(first, 0o755)
	if err := downloadProblemPackage(t.Context(), server.Client(), cfg, 1000, testPackageHash, 0, []string{"data/1.in", "data/1.out"}, first, 1, 1, 1); err != nil {
		t.Fatal(err)
	}
	second := filepath.Join(root, "second")
	_ = os.MkdirAll(second, 0o755)
	if err := downloadProblemPackage(t.Context(), server.Client(), cfg, 1000, testPackageHash, 0, []string{"data/1.in"}, second, 2, 2, 1); err != nil {
		t.Fatal(err)
	}
	if downloads != 1 {
		t.Fatalf("package body downloads = %d, want 1", downloads)
	}
	if _, err := os.Stat(filepath.Join(second, "data", "1.out")); !os.IsNotExist(err) {
		t.Fatalf("tombstoned file should be pruned, err=%v", err)
	}
}

func TestProblemCacheKeepsHashesUntilLRU(t *testing.T) {
	root := t.TempDir()
	archive := filepath.Join(root, "package.zip")
	if err := os.WriteFile(archive, testPackageZip(t), 0o600); err != nil {
		t.Fatal(err)
	}
	old := problemCacheDir(root, 1000, "old")
	current := problemCacheDir(root, 1000, "current")
	if err := replaceProblemCache(old, archive); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	if err := replaceProblemCache(current, archive); err != nil {
		t.Fatal(err)
	}
	if !fileExists(filepath.Join(current, ".size")) {
		t.Fatal("cache size metadata was not written")
	}
	trimProblemCache(root, 1, current)
	if dirExists(old) || !dirExists(current) {
		t.Fatalf("LRU should remove old hash only: old=%t current=%t", dirExists(old), dirExists(current))
	}
}

func TestCustomJudgeCacheUpdatesSize(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "data"), 0o700); err != nil {
		t.Fatal(err)
	}
	data := []byte("case data")
	if err := os.WriteFile(filepath.Join(dir, "data", ".size"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".ready"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeProblemCacheSize(dir); err != nil {
		t.Fatal(err)
	}
	program := filepath.Join(t.TempDir(), "judge")
	body := []byte("custom judge")
	if err := os.WriteFile(program, body, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := storeCachedCustomJudge(program, filepath.Join(dir, "judge.bin")); err != nil {
		t.Fatal(err)
	}
	want := int64(len(data) + len(body))
	if got := readProblemCacheSize(dir); got != want {
		t.Fatalf("cache size = %d, want %d", got, want)
	}
}

func TestInvalidCustomJudgeCacheBecomesMiss(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "judge.bin")
	if err := os.WriteFile(cachePath, nil, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := copyCachedCustomJudge(cachePath, filepath.Join(t.TempDir(), "judge")); !os.IsNotExist(err) {
		t.Fatalf("invalid cache error = %v, want not exist", err)
	}
	if fileExists(cachePath) || readProblemCacheSize(dir) != 0 {
		t.Fatal("invalid custom judge remained cached")
	}
}

func TestDownloadProblemPackageSharesConcurrentDownload(t *testing.T) {
	var mu sync.Mutex
	requests := 0
	downloads := 0
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !validTestPackageRequest(r) {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		requests++
		if r.Header.Get("If-None-Match") != "" {
			mu.Unlock()
			w.WriteHeader(http.StatusNotModified)
			return
		}
		downloads++
		got := downloads
		mu.Unlock()
		entered <- struct{}{}
		if got > 1 {
			t.Errorf("unexpected concurrent package download %d", got)
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
			errCh <- downloadProblemPackage(t.Context(), server.Client(), cfg, 1000, testPackageHash, 0, testPackageFiles(), work, uint(index+1), uint(index+1), 1)
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
	gotDownloads := downloads
	mu.Unlock()
	if gotRequests != 2 || gotDownloads != 1 {
		t.Fatalf("package requests = %d downloads = %d", gotRequests, gotDownloads)
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
		if !validTestPackageRequest(r) {
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
	if err := downloadProblemPackage(t.Context(), server.Client(), cfg, 1000, testPackageHash, int64(len(body)), testPackageFiles(), work, 11, 11, 1); err != nil {
		t.Fatalf("download: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("missing download progress")
	}
	last := got[len(got)-1]
	if last.Stage != "download" || last.Done != int64(len(body)) || last.Total == nil || *last.Total != int64(len(body)) {
		t.Fatalf("download progress = %+v, want done=%d total=%d", last, len(body), len(body))
	}
}

func TestDownloadProblemPackageRejectsWrongHash(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(testPackageZip(t))
	}))
	defer server.Close()

	root := t.TempDir()
	work := filepath.Join(root, "work")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := downloadProblemPackage(t.Context(), server.Client(), WorkerConfig{Server: server.URL, Cache: filepath.Join(root, "cache")}, 1000, strings.Repeat("0", 64), 0, testPackageFiles(), work, 11, 11, 1); err == nil {
		t.Fatal("wrong package hash should fail")
	}
}

func TestDownloadProblemPackageUsesLongerTimeoutThanAPIClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !validTestPackageRequest(r) {
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
	if err := downloadProblemPackage(t.Context(), client, cfg, 1000, testPackageHash, 0, testPackageFiles(), work, 11, 11, 1); err != nil {
		t.Fatalf("download should not use short API timeout: %v", err)
	}
}

func validTestPackageRequest(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/api/judger/tasks/") && strings.HasSuffix(r.URL.Path, "/package") && r.URL.Query().Get("attempt") == "1" && r.URL.Query().Get("hash") == testPackageHash
}

func testPackageFiles() []string {
	return []string{"data/1.in", "data/1.out"}
}

func testPackageZip(t *testing.T) []byte {
	t.Helper()
	return bytes.Clone(testPackageBody)
}

func buildTestPackageZip() []byte {
	var body bytes.Buffer
	writer := zip.NewWriter(&body)
	for name, content := range map[string]string{
		"data/1.in":  "42\n",
		"data/1.out": "42\n",
	} {
		header := &zip.FileHeader{Name: name, Method: zip.Store}
		header.SetMode(0o600)
		file, err := writer.CreateHeader(header)
		if err != nil {
			panic(err)
		}
		if _, err := file.Write([]byte(content)); err != nil {
			panic(err)
		}
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	return body.Bytes()
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
