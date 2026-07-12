package judger

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/doveccl/doj/judger/runner"
)

func TestRunCaseDoesNotReleaseUserWhenCgroupPrepareFails(t *testing.T) {
	clientConn, runnerConn := net.Pipe()
	defer clientConn.Close()
	defer runnerConn.Close()

	client := runnerClient{
		codec:      runner.NewCodec(clientConn),
		cgroupRoot: t.TempDir(),
		procRoot:   t.TempDir(),
		initPID:    123,
		taskID:     "submission-1",
		limits:     runner.Limits{MemoryKB: 1024, Pids: 2},
	}
	errCh := make(chan error, 1)
	go func() {
		_, err := client.runCase(context.Background(), runner.RunCaseRequest{
			TaskID: "submission-1",
			Case:   runner.Case{ID: "case-1"},
		})
		errCh <- err
	}()

	codec := runner.NewCodec(runnerConn)
	if msg, err := codec.Recv(); err != nil || msg.Kind != runner.MsgRunCase {
		t.Fatalf("run request = %+v, %v", msg, err)
	}
	if err := codec.Send(runner.Message{Kind: runner.MsgUserPID, UserPID: &runner.UserPID{
		TaskID: "submission-1",
		CaseID: "case-1",
		PID:    42,
	}}); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-errCh:
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("prepare error = %v, want not exist", err)
		}
	case <-time.After(time.Second):
		t.Fatal("runCase did not fail after cgroup preparation error")
	}
	if err := runnerConn.SetReadDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if msg, err := codec.Recv(); err == nil {
		t.Fatalf("unexpected message after cgroup failure: %s", msg.Kind)
	}
}
