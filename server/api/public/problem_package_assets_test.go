package public

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	problemdata "github.com/doveccl/doj/server/problem"
	"github.com/doveccl/doj/server/storage"
)

func TestPackageArchiveStreamsOnlyManifestData(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"1.in": "input", "1.out": "answer"} {
		if err := os.WriteFile(filepath.Join(root, "data", name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	packagePath := filepath.Join(t.TempDir(), "package.zip")
	item, err := problemdata.Build(root, packagePath, nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatal(err)
	}
	item.Files = item.Files[:1]
	store := &measuredPackageStore{data: data}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	if err := writeProblemPackageArchive(t.Context(), writer, store, 1000, item); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if store.opens != 1 || store.read >= int64(len(data)) {
		t.Fatalf("package reads = %d/%d opens=%d", store.read, len(data), store.opens)
	}
	archive, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil || len(archive.File) != 1 {
		t.Fatalf("streamed archive files=%d err=%v", len(archive.File), err)
	}
	content, err := readZipFile(archive.File[0])
	if err != nil || string(content) != "input" {
		t.Fatalf("streamed file = %q err=%v", content, err)
	}
}

type measuredPackageStore struct {
	storage.Store
	data  []byte
	opens int
	read  int64
}

func (store *measuredPackageStore) Open(context.Context, string) (io.ReadCloser, string, error) {
	store.opens++
	return &measuredPackageReader{Reader: bytes.NewReader(store.data), read: &store.read}, "application/zip", nil
}

type measuredPackageReader struct {
	*bytes.Reader
	read *int64
}

func (reader *measuredPackageReader) Read(buffer []byte) (int, error) {
	n, err := reader.Reader.Read(buffer)
	*reader.read += int64(n)
	return n, err
}

func (*measuredPackageReader) Close() error { return nil }
