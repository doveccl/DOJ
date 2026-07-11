//go:build linux

package runner

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestServeRunnerUserProgramDoesNotInheritInternalFDs(t *testing.T) {
	runner := buildRunner(t)
	work := t.TempDir()
	socket := filepath.Join(os.TempDir(), fmt.Sprintf("doj-fd-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socket)
	if err := os.WriteFile(filepath.Join(work, "1.in"), []byte("go\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "1.out"), []byte("clean\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- ServeRunner(ctx, RunnerServe{Socket: socket, Work: work, Runner: runner})
	}()
	waitSocket(t, socket)

	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	codec := NewCodec(conn)
	if err := codec.Send(Message{Kind: MsgHello, Hello: &Hello{Role: "judger", Version: Version}}); err != nil {
		t.Fatal(err)
	}
	if got, err := codec.Recv(); err != nil || got.Kind != MsgHello {
		t.Fatalf("hello = %#v, %v", got, err)
	}
	source := `#!/bin/sh
read ignored
bad=""
for fd in /proc/self/fd/*; do
  target="$(readlink "$fd" 2>/dev/null || true)"
  case "$target" in
    socket:*|*judge-result-*|*/1.out|*/judge-program)
      bad="$target"
      break
      ;;
  esac
done
if [ -n "$bad" ]; then
  printf 'leak %s\n' "$bad"
else
  printf 'clean\n'
fi
`
	if err := os.WriteFile(filepath.Join(work, "main.sh"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := codec.Send(Message{Kind: MsgCompile, Compile: &CompileRequest{
		TaskID:      "fd-task",
		UserCommand: "sh main.sh",
		Limits:      Limits{TimeMS: 1000, OutputKB: 64},
	}}); err != nil {
		t.Fatal(err)
	}
	if got, err := codec.Recv(); err != nil || got.Kind != MsgCompileResult || got.CompileResult == nil || !got.CompileResult.OK {
		t.Fatalf("compiled = %#v, %v", got, err)
	}
	if err := codec.Send(Message{Kind: MsgRunCase, RunCase: &RunCaseRequest{
		TaskID: "fd-task",
		Case:   Case{ID: "1", Input: "1.in", Answer: "1.out", Score: 100},
		Mode:   ModeDefault,
		Limits: Limits{TimeMS: 1000, OutputKB: 64},
	}}); err != nil {
		t.Fatal(err)
	}
	pid, err := codec.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if pid.Kind != MsgUserPID || pid.UserPID == nil || pid.UserPID.PID <= 0 {
		t.Fatalf("pid = %#v", pid)
	}
	if err := codec.Send(Message{Kind: MsgReleaseUser, ReleaseUser: &ReleaseUser{
		TaskID: pid.UserPID.TaskID,
		CaseID: pid.UserPID.CaseID,
	}}); err != nil {
		t.Fatal(err)
	}
	ran, err := codec.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if ran.Kind != MsgCaseResult || ran.CaseResult == nil || ran.CaseResult.Verdict != VerdictAccepted {
		t.Fatalf("ran = %#v", ran)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("runner server did not exit")
	}
}
