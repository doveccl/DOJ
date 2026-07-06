package judger

import (
	"os"
	"path/filepath"
	"strings"
)

func stashCaseFiles(work string, cases []Case) (func() error, error) {
	tmp, err := os.MkdirTemp(filepath.Dir(work), ".case-assets-")
	if err != nil {
		return nil, err
	}
	moved := map[string]string{}
	for _, item := range cases {
		for _, name := range []string{item.Input, item.Answer} {
			path := absolutizeWorkFile(work, name)
			if moved[path] != "" {
				continue
			}
			info, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				_ = os.RemoveAll(tmp)
				return nil, err
			}
			if !info.Mode().IsRegular() {
				continue
			}
			rel, err := filepath.Rel(work, path)
			if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
				continue
			}
			target := filepath.Join(tmp, rel)
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				_ = os.RemoveAll(tmp)
				return nil, err
			}
			if err := os.Rename(path, target); err != nil {
				_ = os.RemoveAll(tmp)
				return nil, err
			}
			moved[path] = target
		}
	}
	restored := false
	return func() error {
		if restored {
			return nil
		}
		restored = true
		defer os.RemoveAll(tmp)
		for path, target := range moved {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			if err := os.Rename(target, path); err != nil {
				return err
			}
		}
		return nil
	}, nil
}

func protectCaseFiles(work string, cases []Case) error {
	for _, item := range cases {
		for _, name := range []string{item.Input, item.Answer} {
			path := absolutizeWorkFile(work, name)
			rel, err := filepath.Rel(work, path)
			if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
				continue
			}
			if err := os.Chmod(path, 0o600); err != nil && !os.IsNotExist(err) {
				return err
			}
			for dir := filepath.Dir(path); dir != work && strings.HasPrefix(dir, work+string(filepath.Separator)); dir = filepath.Dir(dir) {
				if err := os.Chmod(dir, 0o700); err != nil && !os.IsNotExist(err) {
					return err
				}
			}
		}
	}
	return nil
}
