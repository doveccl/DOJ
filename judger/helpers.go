package judger

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/doveccl/doj/judger/runner"
)

const defaultCompileOutputLimit = 256 << 10

func safeCaseID(id string) string {
	return runner.SafeCaseID(id)
}

func elapsedMS(startedAt time.Time) int {
	return int(time.Since(startedAt).Milliseconds())
}

func cgroupMemoryLimitReached(stats CgroupStats) bool {
	return runner.CgroupMemoryLimitReached(stats)
}

type limitBuffer struct {
	data     []byte
	limit    int64
	overflow bool
}

func (b *limitBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 || int64(len(b.data)+len(p)) <= b.limit {
		b.data = append(b.data, p...)
		return len(p), nil
	}
	remaining := int(b.limit) - len(b.data)
	if remaining > 0 {
		b.data = append(b.data, p[:remaining]...)
	}
	b.overflow = true
	return len(p), nil
}

func (b *limitBuffer) String() string {
	return string(b.data)
}

func (b *limitBuffer) Len() int {
	return len(b.data)
}

func absolutizeWorkFile(work string, name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(work, name)
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
