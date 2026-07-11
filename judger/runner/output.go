package runner

import "bytes"

const defaultCompileOutputLimit = 256 << 10

type limitBuffer struct {
	buffer   bytes.Buffer
	limit    int64
	overflow bool
}

func (b *limitBuffer) String() string {
	return b.buffer.String()
}

func (b *limitBuffer) Write(p []byte) (int, error) {
	original := len(p)
	if b.limit <= 0 {
		return original, nil
	}
	remaining := b.limit - int64(b.buffer.Len())
	if remaining <= 0 {
		b.overflow = true
		return original, nil
	}
	if int64(len(p)) > remaining {
		b.overflow = true
		p = p[:remaining]
	}
	_, _ = b.buffer.Write(p)
	return original, nil
}
