package judger

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
)

func tgzPick(r io.Reader, f string) (b *bytes.Buffer, s string) {
	b = &bytes.Buffer{}
	r = io.TeeReader(r, b)
	if gr, err := gzip.NewReader(r); err == nil {
		tr := tar.NewReader(gr)
		for {
			if h, e := tr.Next(); e != nil {
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
