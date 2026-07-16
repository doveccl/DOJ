package judger

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/maphash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/doveccl/doj/judger/runner"
)

var problemPackageLockSeed = maphash.MakeSeed()
var problemPackageLocks [256]sync.Mutex
var problemCacheTrimLock sync.Mutex

const (
	maxProblemPackageBytes         = 512 << 20
	maxProblemPackageEntries       = 10_000
	maxProblemPackageFileBytes     = 256 << 20
	maxProblemPackageExpandedBytes = 2 << 30
)

func downloadProblemPackage(ctx context.Context, client *http.Client, cfg WorkerConfig, problemID uint, zipHash string, zipSize int64, files []string, taskDir string, taskID uint, submissionID uint, attempt int) error {
	zipHash = strings.TrimSpace(zipHash)
	if zipHash == "" {
		return fmt.Errorf("problem package hash is required")
	}
	cacheRoot := strings.TrimSpace(cfg.Cache)
	if cacheRoot == "" {
		return fmt.Errorf("cache directory is required")
	}
	if err := privateDir(cacheRoot); err != nil {
		return err
	}
	cacheDir := problemCacheDir(cacheRoot, problemID, zipHash)
	lock := problemPackageLock(cacheDir)
	lock.Lock()
	defer lock.Unlock()

	cacheReady := fileExists(filepath.Join(cacheDir, ".ready"))
	logTask(cfg.Logf, submissionID, attempt, "cache ready=%t hash=%s dir=%s", cacheReady, shortHash(zipHash), cacheDir)

	path := fmt.Sprintf("/api/judger/tasks/%d/package?attempt=%d&hash=%s", taskID, attempt, zipHash)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.Server+path, nil)
	if err != nil {
		return err
	}
	if cfg.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	if cacheReady {
		httpReq.Header.Set("If-None-Match", `"`+zipHash+`"`)
	}
	downloadClient := client
	if cfg.HTTPClient == nil {
		downloadClient = &http.Client{Transport: client.Transport, Timeout: defaultPackageHTTPTimeout}
	}
	requestStartedAt := time.Now()
	httpResp, err := downloadClient.Do(httpReq)
	logTask(cfg.Logf, submissionID, attempt, "problem_package_request=%s status=%s hash=%s", formatDuration(time.Since(requestStartedAt)), responseStatus(httpResp), shortHash(zipHash))
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode == http.StatusNotModified {
		if !cacheReady {
			return fmt.Errorf("GET problem package returned unexpected %s", httpResp.Status)
		}
		copyStartedAt := time.Now()
		if err := copyProblemCache(cacheDir, taskDir, files); err != nil {
			return err
		}
		touchProblemCache(cacheDir)
		logTask(cfg.Logf, submissionID, attempt, "cache_copy=%s source=hit", formatDuration(time.Since(copyStartedAt)))
		return nil
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("GET %s returned %s", path, httpResp.Status)
	}
	readStartedAt := time.Now()
	var total *int64
	if zipSize > 0 {
		total = &zipSize
	}
	if cfg.Progress != nil {
		cfg.Progress("download", 0, total)
	}
	reader := &progressReader{reader: httpResp.Body, report: func(done int64) {
		if cfg.Progress != nil {
			cfg.Progress("download", done, total)
		}
	}}
	packageFile, err := os.CreateTemp(cacheRoot, ".package-")
	if err != nil {
		return err
	}
	packagePath := packageFile.Name()
	defer os.Remove(packagePath)
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(packageFile, hash), io.LimitReader(reader, maxProblemPackageBytes+1))
	closeErr := packageFile.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written > maxProblemPackageBytes {
		return fmt.Errorf("problem package exceed %d bytes", maxProblemPackageBytes)
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != zipHash {
		return fmt.Errorf("problem package checksum mismatch")
	}
	logTask(cfg.Logf, submissionID, attempt, "problem_package_read=%s bytes=%d hash=%s", formatDuration(time.Since(readStartedAt)), written, shortHash(zipHash))
	cacheStartedAt := time.Now()
	if err := replaceProblemCache(cacheDir, packagePath); err != nil {
		return err
	}
	logTask(cfg.Logf, submissionID, attempt, "cache_write=%s", formatDuration(time.Since(cacheStartedAt)))
	copyStartedAt := time.Now()
	if err := copyProblemCache(cacheDir, taskDir, files); err != nil {
		return err
	}
	touchProblemCache(cacheDir)
	trimProblemCache(cacheRoot, cfg.CacheBytes, cacheDir)
	logTask(cfg.Logf, submissionID, attempt, "cache_copy=%s source=download", formatDuration(time.Since(copyStartedAt)))
	return nil
}

func touchProblemCache(cacheDir string) {
	now := time.Now()
	_ = os.Chtimes(cacheDir, now, now)
}

func trimProblemCache(root string, limit int64, keep string) {
	if limit <= 0 || !problemCacheTrimLock.TryLock() {
		return
	}
	defer problemCacheTrimLock.Unlock()
	type entry struct {
		path string
		size int64
		used time.Time
	}
	var entries []entry
	var total int64
	dirs, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, problem := range dirs {
		if !problem.IsDir() || !strings.HasPrefix(problem.Name(), "P") {
			continue
		}
		problemPath := filepath.Join(root, problem.Name())
		versions, err := os.ReadDir(problemPath)
		if err != nil {
			continue
		}
		for _, version := range versions {
			if !version.IsDir() {
				continue
			}
			item := entry{path: filepath.Join(problemPath, version.Name())}
			info, err := version.Info()
			if err != nil {
				continue
			}
			item.used = info.ModTime()
			item.size = readProblemCacheSize(item.path)
			total += item.size
			entries = append(entries, item)
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].used.Before(entries[j].used) })
	for _, item := range entries {
		if total <= limit {
			break
		}
		if item.path == keep {
			continue
		}
		lock := problemPackageLock(item.path)
		if !lock.TryLock() {
			continue
		}
		if os.RemoveAll(item.path) == nil {
			total -= item.size
			_ = os.Remove(filepath.Dir(item.path))
		}
		lock.Unlock()
	}
}

func problemPackageLock(path string) *sync.Mutex {
	return &problemPackageLocks[maphash.String(problemPackageLockSeed, path)%uint64(len(problemPackageLocks))]
}

type progressReader struct {
	reader io.Reader
	done   int64
	report func(done int64)
}

func (reader *progressReader) Read(p []byte) (int, error) {
	n, err := reader.reader.Read(p)
	if n > 0 {
		reader.done += int64(n)
		reader.report(reader.done)
	}
	return n, err
}

func problemCacheDir(root string, problemID uint, hash string) string {
	root = strings.TrimSpace(root)
	return filepath.Join(root, fmt.Sprintf("P%d", problemID), hash)
}

func customJudgeCachePath(root string, problemID uint, hash string, mode runner.JudgeMode) string {
	root = strings.TrimSpace(root)
	if mode != runner.ModeCustom || root == "" {
		return ""
	}
	return filepath.Join(problemCacheDir(root, problemID, hash), "judge.bin")
}

func replaceProblemCache(cacheDir string, packagePath string) error {
	parent := filepath.Dir(cacheDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, ".cache-")
	if err != nil {
		return err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(tmp)
		}
	}()
	if err := extractProblemPackageFile(packagePath, tmp); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmp, ".ready"), nil, 0o600); err != nil {
		return err
	}
	if err := writeProblemCacheSize(tmp); err != nil {
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

func writeProblemCacheSize(cacheDir string) error {
	size, err := problemCacheSize(cacheDir)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cacheDir, ".size"), []byte(strconv.FormatInt(size, 10)), 0o600)
}

func readProblemCacheSize(cacheDir string) int64 {
	data, err := os.ReadFile(filepath.Join(cacheDir, ".size"))
	if err == nil {
		if size, parseErr := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); parseErr == nil && size >= 0 {
			return size
		}
	}
	size, err := problemCacheSize(cacheDir)
	if err == nil {
		_ = os.WriteFile(filepath.Join(cacheDir, ".size"), []byte(strconv.FormatInt(size, 10)), 0o600)
	}
	return size
}

func problemCacheSize(cacheDir string) (int64, error) {
	var size int64
	sizePath := filepath.Join(cacheDir, ".size")
	err := filepath.WalkDir(cacheDir, func(path string, file os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if file.IsDir() || path == sizePath {
			return nil
		}
		info, err := file.Info()
		if err != nil {
			return err
		}
		size += info.Size()
		return nil
	})
	return size, err
}

func copyProblemCache(cacheDir string, taskDir string, files []string) error {
	for _, name := range files {
		clean := filepath.Clean(filepath.FromSlash(name))
		if clean == "." || !filepath.IsLocal(clean) {
			return fmt.Errorf("invalid package file %q", name)
		}
		target := filepath.Join(taskDir, clean)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := copyFile(filepath.Join(cacheDir, clean), target, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func responseStatus(resp *http.Response) string {
	if resp == nil {
		return "-"
	}
	return resp.Status
}

func shortHash(hash string) string {
	clean := strings.Trim(strings.TrimSpace(hash), `"`)
	if len(clean) <= 12 {
		return clean
	}
	return clean[:12]
}

func extractProblemPackage(data []byte, work string) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	return extractProblemPackageFiles(reader.File, work)
}

func extractProblemPackageFile(path string, work string) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()
	return extractProblemPackageFiles(reader.File, work)
}

func extractProblemPackageFiles(files []*zip.File, work string) error {
	if err := validateProblemPackageFiles(files); err != nil {
		return err
	}
	var expanded int64
	for _, file := range files {
		name := filepath.Clean(strings.ReplaceAll(file.Name, "\\", "/"))
		if name == "." || name == ".." || filepath.IsAbs(name) || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			return fmt.Errorf("invalid package path %q", file.Name)
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
		written, copyErr := io.Copy(dst, io.LimitReader(src, maxProblemPackageFileBytes+1))
		expanded += written
		if copyErr != nil || written > maxProblemPackageFileBytes || expanded > maxProblemPackageExpandedBytes {
			_ = src.Close()
			_ = dst.Close()
			if copyErr != nil {
				return copyErr
			}
			return fmt.Errorf("problem package expands beyond limit")
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

func validateProblemPackageFiles(files []*zip.File) error {
	if len(files) > maxProblemPackageEntries {
		return fmt.Errorf("problem package has too many entries")
	}
	var total uint64
	for _, file := range files {
		if file.FileInfo().IsDir() {
			continue
		}
		if file.UncompressedSize64 > maxProblemPackageFileBytes || total > maxProblemPackageExpandedBytes-file.UncompressedSize64 {
			return fmt.Errorf("problem package expands beyond limit")
		}
		total += file.UncompressedSize64
	}
	return nil
}
