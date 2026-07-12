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

func TestServeRunnerCompileAndRunCase(t *testing.T) {
	skipShortRunnerIntegration(t)
	runner := buildRunner(t)
	work := t.TempDir()
	socket := filepath.Join(os.TempDir(), fmt.Sprintf("doj-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socket)
	if err := os.WriteFile(filepath.Join(work, "1.in"), []byte("42\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "1.out"), []byte("42\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- ServeRunner(ctx, RunnerServe{Socket: socket, Work: work, Runner: runner})
	}()
	waitSocket(t, socket)
	info, err := os.Stat(socket)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("runner socket mode = %v, want 0600", info.Mode().Perm())
	}

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
	if err := codec.Send(Message{Kind: MsgCompile, Compile: &CompileRequest{
		TaskID:      "task-1",
		UserCommand: "cat",
		Limits:      Limits{TimeMS: 1000, OutputKB: 64},
	}}); err != nil {
		t.Fatal(err)
	}
	compiled, err := codec.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if compiled.Kind != MsgCompileResult || compiled.CompileResult == nil || !compiled.CompileResult.OK {
		t.Fatalf("compiled = %#v", compiled)
	}

	if err := codec.Send(Message{Kind: MsgRunCase, RunCase: &RunCaseRequest{
		TaskID: "task-1",
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

	resultCh := make(chan Message, 1)
	errRecvCh := make(chan error, 1)
	go func() {
		got, err := codec.Recv()
		if err != nil {
			errRecvCh <- err
			return
		}
		resultCh <- got
	}()
	select {
	case got := <-resultCh:
		t.Fatalf("case result arrived before release: %#v", got)
	case err := <-errRecvCh:
		t.Fatalf("recv before release failed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if err := codec.Send(Message{Kind: MsgReleaseUser, ReleaseUser: &ReleaseUser{
		TaskID: pid.UserPID.TaskID,
		CaseID: pid.UserPID.CaseID,
	}}); err != nil {
		t.Fatal(err)
	}
	var ran Message
	select {
	case ran = <-resultCh:
	case err := <-errRecvCh:
		t.Fatal(err)
	case <-time.After(time.Second):
		t.Fatal("case result did not arrive after release")
	}
	if ran.Kind != MsgCaseResult || ran.CaseResult == nil || ran.CaseResult.Verdict != VerdictAccepted {
		t.Fatalf("ran = %#v", ran)
	}
	if err := codec.Send(Message{Kind: MsgBye}); err != nil {
		t.Fatal(err)
	}
	if got, err := codec.Recv(); err != nil || got.Kind != MsgBye {
		t.Fatalf("bye = %#v, %v", got, err)
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

func TestServeRunnerRunCaseTimesOutWithoutRelease(t *testing.T) {
	skipShortRunnerIntegration(t)
	runner := buildRunner(t)
	work := t.TempDir()
	socket := filepath.Join(os.TempDir(), fmt.Sprintf("doj-timeout-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socket)
	if err := os.WriteFile(filepath.Join(work, "1.in"), []byte("42\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "1.out"), []byte("42\n"), 0o600); err != nil {
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
	if err := codec.Send(Message{Kind: MsgCompile, Compile: &CompileRequest{
		TaskID:      "task-1",
		UserCommand: "cat",
		Limits:      Limits{TimeMS: 200, OutputKB: 64},
	}}); err != nil {
		t.Fatal(err)
	}
	if got, err := codec.Recv(); err != nil || got.Kind != MsgCompileResult {
		t.Fatalf("compiled = %#v, %v", got, err)
	}
	if err := codec.Send(Message{Kind: MsgRunCase, RunCase: &RunCaseRequest{
		TaskID: "task-1",
		Case:   Case{ID: "1", Input: "1.in", Answer: "1.out", Score: 100},
		Mode:   ModeDefault,
		Limits: Limits{TimeMS: 200, OutputKB: 64},
	}}); err != nil {
		t.Fatal(err)
	}
	if got, err := codec.Recv(); err != nil || got.Kind != MsgUserPID {
		t.Fatalf("pid = %#v, %v", got, err)
	}
	ran, err := codec.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if ran.Kind != MsgCaseResult || ran.CaseResult == nil || ran.CaseResult.Verdict != VerdictTimeLimit {
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

func waitSocket(t *testing.T, socket string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socket); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("socket %s was not created", socket)
}

func skipShortRunnerIntegration(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("runner service integration test")
	}
}

func TestRunnerRejectsCommandsBeforeHello(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()
	go func() {
		defer server.Close()
		handleRunnerConn(context.Background(), server, RunnerServe{})
	}()
	codec := NewCodec(client)
	if err := codec.Send(Message{Kind: MsgCompile, Compile: &CompileRequest{UserCommand: "true"}}); err != nil {
		t.Fatal(err)
	}
	got, err := codec.Recv()
	if err != nil || got.Kind != MsgError {
		t.Fatalf("pre-hello response = %#v, %v", got, err)
	}
}
