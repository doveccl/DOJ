package storage

import (
	"bytes"
	"context"
	"errors"
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
	prefix, err := cleanListPrefix("users/1/")
	if err != nil || prefix != "users/1/" {
		t.Fatalf("list prefix = %q, %v", prefix, err)
	}
}

func TestIsNotFound(t *testing.T) {
	if !IsNotFound(os.ErrNotExist) || !IsNotFound(minio.ErrorResponse{Code: "NoSuchKey"}) {
		t.Fatal("missing storage objects were not recognized")
	}
	if IsNotFound(errors.New("storage offline")) {
		t.Fatal("storage failure was treated as a missing object")
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

func TestNewFromEnvCachesS3StoresByURL(t *testing.T) {
	t.Setenv("STORAGE", "https://ak:sk@s3.example.com/doj?lookup=dns")
	first, err := NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if first.(s3Store).client != second.(s3Store).client {
		t.Fatal("same S3 STORAGE should reuse client")
	}
	t.Setenv("STORAGE", "https://ak:sk@s3.example.com/other?lookup=dns")
	third, err := NewFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if first.(s3Store).client == third.(s3Store).client {
		t.Fatal("different S3 STORAGE should not reuse client")
	}
}

func TestLocalStore(t *testing.T) {
	root := t.TempDir()
	store := localStore{root: root}
	ctx := context.Background()
	if err := os.MkdirAll(root+"/users/uploads", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(ctx, "users/uploads/dot.txt", bytes.NewBufferString("hello"), 5, "text/plain"); err != nil {
		t.Fatalf("put failed: %v", err)
	}
	dirInfo, err := os.Stat(root + "/users/uploads")
	if err != nil || dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("object directory mode = %v, %v", dirInfo, err)
	}
	fileInfo, err := os.Stat(root + "/users/uploads/dot.txt")
	if err != nil || fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("object file mode = %v, %v", fileInfo, err)
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
	rangeReader, err := store.OpenRange(ctx, "users/uploads/dot.txt", 1, 3)
	if err != nil {
		t.Fatalf("open range failed: %v", err)
	}
	rangeData, err := io.ReadAll(rangeReader)
	_ = rangeReader.Close()
	if err != nil || string(rangeData) != "ell" {
		t.Fatalf("range data = %q, %v", rangeData, err)
	}
	if _, err := store.OpenRange(ctx, "users/uploads/dot.txt", 4, 2); err == nil {
		t.Fatal("out-of-bounds range should fail")
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

func TestS3StoreRangeIntegration(t *testing.T) {
	raw := os.Getenv("DOJ_TEST_S3")
	if raw == "" {
		t.Skip("DOJ_TEST_S3 is not set")
	}
	t.Setenv("STORAGE", raw)
	store, err := NewFromEnv()
	if err != nil {
		t.Fatalf("new S3 store: %v", err)
	}
	s3 := store.(s3Store)
	ctx := t.Context()
	if err := s3.client.MakeBucket(ctx, s3.bucket, minio.MakeBucketOptions{}); err != nil {
		exists, existsErr := s3.client.BucketExists(ctx, s3.bucket)
		if existsErr != nil || !exists {
			t.Fatalf("make bucket: %v, exists: %v", err, existsErr)
		}
	}
	if err := store.Put(ctx, "range/test.txt", bytes.NewBufferString("0123456789"), 10, "text/plain"); err != nil {
		t.Fatalf("put S3 object: %v", err)
	}
	reader, err := store.OpenRange(ctx, "range/test.txt", 3, 4)
	if err != nil {
		t.Fatalf("open S3 range: %v", err)
	}
	got, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if readErr != nil || string(got) != "3456" {
		t.Fatalf("S3 range data = %q, %v", got, readErr)
	}
}
