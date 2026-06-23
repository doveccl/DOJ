package utils

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestCleanObjectKey(t *testing.T) {
	valid, err := CleanObjectKey("users/uploads/a.png")
	if err != nil || valid != "users/uploads/a.png" {
		t.Fatalf("valid key = %q, %v", valid, err)
	}
	if _, err := CleanObjectKey("../users/uploads/a.png"); err == nil {
		t.Fatal("path traversal should be invalid")
	}
	if _, err := CleanObjectKey(""); err == nil {
		t.Fatal("empty key should be invalid")
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
