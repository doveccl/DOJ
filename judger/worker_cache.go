package judger

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var problemPackageLocks sync.Map

func downloadProblemPackage(ctx context.Context, client *http.Client, cfg WorkerConfig, problemID uint, packageHash string, taskDir string, submissionID uint, attempt int) error {
	packageHash = strings.TrimSpace(packageHash)
	if packageHash == "" {
		return fmt.Errorf("problem package hash is required")
	}
	cacheRoot := strings.TrimSpace(cfg.Cache)
	if cacheRoot == "" {
		return fmt.Errorf("cache directory is required")
	}
	cacheDir := problemCacheDir(cacheRoot, problemID)
	lockValue, _ := problemPackageLocks.LoadOrStore(cacheDir, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	cacheHash := readProblemCacheHash(cacheDir)
	cacheReady := cacheHash != "" && cacheHash == packageHash && dirExists(cacheDir)
	logTask(cfg.Logf, submissionID, attempt, "cache ready=%t hash=%s dir=%s", cacheReady, shortHash(cacheHash), cacheDir)
	if cacheReady {
		copyStartedAt := time.Now()
		if err := copyProblemCache(cacheDir, taskDir); err != nil {
			return err
		}
		logTask(cfg.Logf, submissionID, attempt, "cache_copy=%s source=hit", formatDuration(time.Since(copyStartedAt)))
		return nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.Server+fmt.Sprintf("/api/judger/P%d.zip", problemID), nil)
	if err != nil {
		return err
	}
	if cfg.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	downloadClient := client
	if cfg.HTTPClient == nil {
		downloadClient = &http.Client{Transport: client.Transport, Timeout: defaultPackageHTTPTimeout}
	}
	requestStartedAt := time.Now()
	httpResp, err := downloadClient.Do(httpReq)
	logTask(cfg.Logf, submissionID, attempt, "problem_package_request=%s status=%s cache_hash=%s lease_hash=%s", formatDuration(time.Since(requestStartedAt)), responseStatus(httpResp), shortHash(cacheHash), shortHash(packageHash))
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("GET problem package returned %s", httpResp.Status)
	}
	const maxProblemPackageBytes = 512 << 20
	readStartedAt := time.Now()
	if cfg.Progress != nil {
		cfg.Progress("download", 0, nil)
	}
	reader := &progressReader{reader: httpResp.Body, report: func(done int64) {
		if cfg.Progress != nil {
			cfg.Progress("download", done, nil)
		}
	}}
	data, err := io.ReadAll(io.LimitReader(reader, maxProblemPackageBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxProblemPackageBytes {
		return fmt.Errorf("problem package exceed %d bytes", maxProblemPackageBytes)
	}
	logTask(cfg.Logf, submissionID, attempt, "problem_package_read=%s bytes=%d hash=%s", formatDuration(time.Since(readStartedAt)), len(data), shortHash(packageHash))
	cacheStartedAt := time.Now()
	if err := replaceProblemCache(cacheDir, data, packageHash); err != nil {
		return err
	}
	logTask(cfg.Logf, submissionID, attempt, "cache_write=%s", formatDuration(time.Since(cacheStartedAt)))
	copyStartedAt := time.Now()
	if err := copyProblemCache(cacheDir, taskDir); err != nil {
		return err
	}
	logTask(cfg.Logf, submissionID, attempt, "cache_copy=%s source=download", formatDuration(time.Since(copyStartedAt)))
	return nil
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

func problemCacheDir(root string, problemID uint) string {
	root = strings.TrimSpace(root)
	return filepath.Join(root, fmt.Sprintf("P%d", problemID))
}

func customJudgeCachePath(root string, problemID uint, mode JudgeMode) string {
	root = strings.TrimSpace(root)
	if mode != ModeCustom || root == "" {
		return ""
	}
	return filepath.Join(root, fmt.Sprintf("P%d", problemID), "judge.bin")
}

func readProblemCacheHash(cacheDir string) string {
	raw, err := os.ReadFile(filepath.Join(cacheDir, "hash"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func replaceProblemCache(cacheDir string, data []byte, hash string) error {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return fmt.Errorf("problem package hash is required")
	}
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
	if err := extractProblemPackage(data, tmp); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmp, "hash"), []byte(hash+"\n"), 0o644); err != nil {
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

func copyProblemCache(cacheDir string, taskDir string) error {
	for _, name := range []string{"data", "judge"} {
		src := filepath.Join(cacheDir, name)
		if !dirExists(src) {
			continue
		}
		if err := copyDir(src, filepath.Join(taskDir, name)); err != nil {
			return err
		}
	}
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
	for _, file := range reader.File {
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
