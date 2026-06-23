package judger

import (
	"io"
	"os"
	"path/filepath"
)

func prepareCaseWork(work string, caseWork string, item Case) error {
	if err := os.RemoveAll(caseWork); err != nil {
		return err
	}
	if err := os.MkdirAll(caseWork, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(work)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src := filepath.Join(work, entry.Name())
		if shouldSkipRuntimeCopy(src, item) {
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
	return len(name) >= len("judge-result-") && name[:len("judge-result-")] == "judge-result-"
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
	return out.Close()
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
