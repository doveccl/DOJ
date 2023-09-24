package judger

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

func tgzPick(r io.Reader, f string) (b *bytes.Buffer, s string) {
	b = &bytes.Buffer{}
	r = io.TeeReader(r, b)
	if gr, err := gzip.NewReader(r); err == nil {
		tr := tar.NewReader(gr)
		for {
			h, e := tr.Next()
			if e == io.EOF || e != nil {
				break
			} else if h.Name == f {
				if b, e := io.ReadAll(tr); e == nil {
					s = string(b)
					break
				}
			}
		}
	}
	return
}

func envList(m map[string]any) (l []string) {
	for k, v := range m {
		l = append(l, fmt.Sprintf("%v=%v", k, v))
	}
	return
}
