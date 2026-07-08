package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"
)

func TestCleanKey(t *testing.T) {
	valid, err := CleanKey("users/uploads/a.png")
	if err != nil || valid != "users/uploads/a.png" {
		t.Fatalf("valid key = %q, %v", valid, err)
	}
	if _, err := CleanKey("../users/uploads/a.png"); err == nil {
		t.Fatal("path traversal should be invalid")
	}
	if _, err := CleanKey(""); err == nil {
		t.Fatal("empty key should be invalid")
	}
}

func TestRootUsesStorage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("STORAGE", "")
	if got := Root(); got != home {
		t.Fatalf("default upload root = %q, want home %q", got, home)
	}
	t.Setenv("STORAGE", t.TempDir())
	if got := Root(); got != os.Getenv("STORAGE") {
		t.Fatalf("upload root = %q, want STORAGE", got)
	}
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("STORAGE", "")
	if got := Root(); got != "storage" {
		t.Fatalf("fallback upload root = %q, want storage", got)
	}
}

func TestParseS3Storage(t *testing.T) {
	got, err := parseS3Storage("https://ak:sk@s3.example.com/doj?lookup=dns")
	if err != nil {
		t.Fatalf("parse storage: %v", err)
	}
	if got.endpoint != "s3.example.com" || got.access != "ak" || got.secret != "sk" || got.bucket != "doj" || !got.secure || got.lookup != minio.BucketLookupDNS {
		t.Fatalf("unexpected storage config: %+v", got)
	}
	got, err = parseS3Storage("http://ak:sk@s3.example.com/doj")
	if err != nil {
		t.Fatalf("parse default storage: %v", err)
	}
	if got.lookup != minio.BucketLookupAuto || got.secure {
		t.Fatalf("unexpected default storage config: %+v", got)
	}
	if _, err := parseS3Storage("https://ak:sk@s3.example.com/doj/prefix"); err == nil {
		t.Fatalf("bucket path with prefix should be rejected")
	}
	if _, err := parseS3Storage("https://ak:sk@s3.example.com/doj?lookup=bad"); err == nil {
		t.Fatalf("bad lookup should be rejected")
	}
}

func TestLocalStore(t *testing.T) {
	store := localStore{root: t.TempDir()}
	ctx := context.Background()
	if err := store.Put(ctx, "users/uploads/dot.txt", bytes.NewBufferString("hello"), 5, "text/plain"); err != nil {
		t.Fatalf("put failed: %v", err)
	}
	reader, contentType, err := store.Open(ctx, "users/uploads/dot.txt")
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("data = %q", data)
	}
	if contentType != "text/plain; charset=utf-8" {
		t.Fatalf("content type = %q", contentType)
	}
	items, err := store.List(ctx, "users")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 || items[0].Key != "users/uploads/dot.txt" || items[0].Size != 5 {
		t.Fatalf("unexpected list: %+v", items)
	}
	if err := store.Delete(ctx, "users/uploads/dot.txt"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, _, err := store.Open(ctx, "users/uploads/dot.txt"); err == nil {
		t.Fatal("deleted object should not open")
	}
}
