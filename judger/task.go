package judger

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

func prepareCaseWork(work string, caseWork string, item Case, runtimeRoot string, skipName string) error {
	if err := os.RemoveAll(caseWork); err != nil {
		return err
	}
	if err := os.MkdirAll(caseWork, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(runtimeRoot)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if entry.Name() == skipName {
			continue
		}
		src := filepath.Join(runtimeRoot, entry.Name())
		if runtimeRoot == work && shouldSkipRuntimeCopy(src, item) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if err := copyFile(src, filepath.Join(caseWork, entry.Name()), info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}

func shouldSkipRuntimeCopy(path string, item Case) bool {
	if path == item.Input || path == item.Answer {
		return true
	}
	name := filepath.Base(path)
	if name == "judge-program" {
		return true
	}
	for _, prefix := range []string{"judge-result-", "judge-transcript-", "user-output-"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func copyFile(src string, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, mode)
}

func absolutizeCase(work string, item Case) Case {
	item.Input = absolutizeWorkFile(work, item.Input)
	item.Answer = absolutizeWorkFile(work, item.Answer)
	return item
}

func absolutizeWorkFile(work string, name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(work, name)
}
