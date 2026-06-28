package judger

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type RunnerServe struct {
	Socket       string
	Runner       string
	Work         string
	RuntimeRoot  string
	SkipRuntime  string
	UserIdentity ProcessIdentity
}

func ServeRunner(ctx context.Context, req RunnerServe) error {
	if req.Socket == "" {
		return fmt.Errorf("socket path is required")
	}
	if req.Work == "" {
		return fmt.Errorf("work directory is required")
	}
	if req.Runner == "" {
		runner, err := os.Executable()
		if err != nil {
			return err
		}
		req.Runner = runner
	}
	if err := os.MkdirAll(filepath.Dir(req.Socket), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(req.Work, 0o755); err != nil {
		return err
	}
	if req.RuntimeRoot == "" {
		req.RuntimeRoot = req.Work
	}
	_ = os.Remove(req.Socket)

	listener, err := net.Listen("unix", req.Socket)
	if err != nil {
		return err
	}
	if err := os.Chmod(req.Socket, 0o666); err != nil {
		_ = listener.Close()
		return err
	}
	defer listener.Close()
	defer os.Remove(req.Socket)

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = listener.Close()
		case <-done:
		}
	}()
	defer close(done)

	var wg sync.WaitGroup
	defer wg.Wait()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer conn.Close()
			handleRunnerConn(ctx, conn, req)
		}()
	}
}

func handleRunnerConn(ctx context.Context, conn net.Conn, req RunnerServe) {
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()
	defer close(done)

	codec := NewCodec(conn)
	commands := map[string]string{}
	pending := map[string]chan struct{}{}
	var pendingMu sync.Mutex
	for {
		msg, err := codec.Recv()
		if err != nil {
			return
		}
		switch msg.Kind {
		case MsgHello:
			_ = codec.Send(Message{Kind: MsgHello, Hello: &Hello{Role: "runner", Version: Version}})
		case MsgCompile:
			if msg.Compile == nil {
				_ = codec.Send(Message{Kind: MsgError, Error: "compile request is empty"})
				continue
			}
			if msg.Compile.UserCommand == "" {
				_ = codec.Send(Message{Kind: MsgError, Error: "user command is required"})
				continue
			}
			result, err := compileUserProgram(ctx, req.Work, *msg.Compile)
			if err != nil {
				_ = codec.Send(Message{Kind: MsgError, Error: err.Error()})
				continue
			}
			commands[msg.Compile.TaskID] = msg.Compile.UserCommand
			_ = codec.Send(Message{Kind: MsgCompileResult, CompileResult: &result})
		case MsgRunCase:
			if msg.RunCase == nil {
				_ = codec.Send(Message{Kind: MsgError, Error: "run_case request is empty"})
				continue
			}
			userCommand, ok := commands[msg.RunCase.TaskID]
			if !ok {
				_ = codec.Send(Message{Kind: MsgError, Error: "task is not compiled"})
				continue
			}
			release := make(chan struct{})
			key := releaseKey(msg.RunCase.TaskID, msg.RunCase.Case.ID)
			pendingMu.Lock()
			pending[key] = release
			pendingMu.Unlock()
			go func(run RunCaseRequest, command string, runKey string, runRelease chan struct{}) {
				defer func() {
					pendingMu.Lock()
					delete(pending, runKey)
					pendingMu.Unlock()
				}()
				runCtx := ctx
				cancel := func() {}
				if run.Limits.TimeMS > 0 {
					runCtx, cancel = context.WithTimeout(ctx, time.Duration(run.Limits.TimeMS)*time.Millisecond)
				}
				defer cancel()
				absCase := absolutizeCase(req.Work, run.Case)
				caseWork, err := os.MkdirTemp("", "doj-case-"+safeCaseID(run.Case.ID)+"-")
				if err != nil {
					_ = codec.Send(Message{Kind: MsgError, Error: err.Error()})
					return
				}
				defer os.RemoveAll(caseWork)
				if err := prepareCaseWork(req.Work, caseWork, absCase, req.RuntimeRoot, req.SkipRuntime); err != nil {
					_ = codec.Send(Message{Kind: MsgError, Error: err.Error()})
					return
				}
				result, err := RunLocalCase(runCtx, LocalRun{
					Runner:       req.Runner,
					Work:         req.Work,
					UserWork:     caseWork,
					TaskID:       run.TaskID,
					UserCommand:  command,
					JudgeCommand: run.JudgeCommand,
					UserIdentity: req.UserIdentity,
					UserGate:     protocolUserGate{codec: codec, release: runRelease},
					Mode:         run.Mode,
					Case:         absCase,
					Limits:       run.Limits,
				})
				if err != nil {
					_ = codec.Send(Message{Kind: MsgError, Error: err.Error()})
					return
				}
				_ = codec.Send(Message{Kind: MsgCaseResult, CaseResult: &result})
			}(*msg.RunCase, userCommand, key, release)
		case MsgReleaseUser:
			if msg.ReleaseUser == nil {
				_ = codec.Send(Message{Kind: MsgError, Error: "release_user request is empty"})
				continue
			}
			key := releaseKey(msg.ReleaseUser.TaskID, msg.ReleaseUser.CaseID)
			pendingMu.Lock()
			release, ok := pending[key]
			if ok {
				delete(pending, key)
				close(release)
			}
			pendingMu.Unlock()
			if !ok {
				_ = codec.Send(Message{Kind: MsgError, Error: "release target is not pending"})
			}
		case MsgBye:
			_ = codec.Send(Message{Kind: MsgBye})
			return
		default:
			_ = codec.Send(Message{Kind: MsgError, Error: "unsupported message kind"})
		}
	}
}

type protocolUserGate struct {
	codec   *Codec
	release <-chan struct{}
}

func (gate protocolUserGate) WaitUserRelease(ctx context.Context, pid UserPID) error {
	if err := gate.codec.Send(Message{Kind: MsgUserPID, UserPID: &pid}); err != nil {
		return err
	}
	select {
	case <-gate.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseKey(taskID string, caseID string) string {
	return taskID + "\x00" + caseID
}
