package judger

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestMapInnerPIDUsesContainerNamespace(t *testing.T) {
	root := t.TempDir()
	writeProcEntry(t, root, 100, "pid:[init-a]", []int{100, 1})
	writeProcEntry(t, root, 201, "pid:[init-a]", []int{201, 42})
	writeProcEntry(t, root, 300, "pid:[init-b]", []int{300, 1})
	writeProcEntry(t, root, 301, "pid:[init-b]", []int{301, 42})

	hostPID, err := MapInnerPID(root, 100, 42)
	if err != nil {
		t.Fatal(err)
	}
	if hostPID != 201 {
		t.Fatalf("host pid = %d, want 201", hostPID)
	}
}

func TestMapInnerPIDNotFound(t *testing.T) {
	root := t.TempDir()
	writeProcEntry(t, root, 100, "pid:[init-a]", []int{100, 1})
	writeProcEntry(t, root, 201, "pid:[init-a]", []int{201, 42})

	_, err := MapInnerPID(root, 100, 43)
	if err == nil {
		t.Fatal("expected missing inner pid error")
	}
}

func TestMapInnerPIDRequiresNamespaceLink(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "100")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "status"), []byte("Name:\tinit\nNSpid:\t100\t1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := MapInnerPID(root, 100, 1)
	if err == nil {
		t.Fatal("expected missing namespace link error")
	}
}

func TestMapInnerPIDRejectsInvalidArgs(t *testing.T) {
	if _, err := MapInnerPID(t.TempDir(), 0, 1); err == nil {
		t.Fatal("expected missing init pid error")
	}
	if _, err := MapInnerPID(t.TempDir(), 1, 0); err == nil {
		t.Fatal("expected missing inner pid error")
	}
}

func writeProcEntry(t *testing.T, root string, hostPID int, ns string, nspids []int) {
	t.Helper()
	dir := filepath.Join(root, intString(hostPID))
	if err := os.MkdirAll(filepath.Join(dir, "ns"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(ns, filepath.Join(dir, "ns", "pid")); err != nil && !errors.Is(err, os.ErrExist) {
		t.Fatal(err)
	}
	content := "Name:\tfixture\nNSpid:"
	for _, id := range nspids {
		content += "\t" + intString(id)
	}
	content += "\n"
	if err := os.WriteFile(filepath.Join(dir, "status"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func intString(value int) string {
	return strconv.Itoa(value)
}
