package problem

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/doveccl/doj/server/storage"
)

func TestBuildAndRangeRead(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{
		"data/1.in": "input", "data/1.out": "answer", "data/2.in": "two", "data/2.out": "second", "judge/main.cc": "main",
	} {
		file := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	score := 0
	zipPath := filepath.Join(t.TempDir(), "package.zip")
	item, err := Build(root, zipPath, []Case{{ID: "1", Score: &score}})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(item.Files) != 5 || len(item.Cases) != 2 || item.Cases[0].Points() != 0 || item.Cases[1].Points() != 10 {
		t.Fatalf("package = %+v", item)
	}
	storageRoot := t.TempDir()
	t.Setenv("STORAGE", storageRoot)
	store, err := storage.NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.Open(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), ObjectKey(1000, item.Hash), body, item.Size, "application/zip"); err != nil {
		t.Fatal(err)
	}
	_ = body.Close()
	file, ok := FindFile(item, "data/1.out")
	if !ok {
		t.Fatal("file not indexed")
	}
	reader, err := OpenFile(context.Background(), store, ObjectKey(1000, item.Hash), file)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil || string(got) != "answer" {
		t.Fatalf("range read = %q, %v", got, err)
	}
	file.CRC32++
	reader, err = OpenFile(context.Background(), store, ObjectKey(1000, item.Hash), file)
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.ReadAll(reader)
	_ = reader.Close()
	if err == nil {
		t.Fatal("corrupt package file should fail checksum validation")
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "data"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "data", "1.in"), []byte(strings.Repeat("x", 100)), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := Build(root, filepath.Join(t.TempDir(), "a.zip"), nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(root, filepath.Join(t.TempDir(), "b.zip"), nil)
	if err != nil || first.Hash != second.Hash {
		t.Fatalf("hashes = %s, %s, %v", first.Hash, second.Hash, err)
	}
}

func TestBuildRejectsDuplicateCaseSide(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"data/1.in", "data/input1.txt", "data/1.out"} {
		file := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	_, err := Build(root, filepath.Join(t.TempDir(), "package.zip"), nil)
	if err == nil || !strings.Contains(err.Error(), "multiple in files") {
		t.Fatalf("duplicate case input error = %v", err)
	}
}
