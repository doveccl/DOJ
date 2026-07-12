package judger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/doveccl/doj/judger/runner"
)

type runnerClient struct {
	codec      *runner.Codec
	cgroupRoot string
	procRoot   string
	initPID    int
	taskID     string
	limits     runner.Limits
	work       string
	judgeCache string
	logf       func(format string, args ...any)
	progress   func(stage string, done int64, total *int64)
}

func (client runnerClient) runTask(ctx context.Context, task runner.Task, compileCommand string, compileMS int, userCommand string, beforeCases func() error) (runner.TaskResult, error) {
	result := runner.TaskResult{SubmissionID: task.SubmissionID, Attempt: task.Attempt}
	if len(task.Cases) == 0 {
		result.Verdict = runner.VerdictSystemError
		result.Message = "task has no cases"
		return result, nil
	}
	helloStartedAt := time.Now()
	if err := client.hello(); err != nil {
		return runner.TaskResult{}, err
	}
	logStep(client.logf, task.SubmissionID, task.Attempt, "runner_hello", helloStartedAt)
	compileStartedAt := time.Now()
	client.reportProgress("compile", 0, nil)
	compile, err := client.compile(task, compileCommand, compileMS, userCommand)
	logTask(client.logf, task.SubmissionID, task.Attempt, "compile=%s ok=%t reported=%dms", formatDuration(time.Since(compileStartedAt)), compile.OK, compile.TimeMS)
	if err != nil {
		return runner.TaskResult{}, err
	}
	if !compile.OK {
		result.Verdict = runner.VerdictCompileError
		result.Message = compile.Message
		return result, nil
	}
	if beforeCases != nil {
		if err := beforeCases(); err != nil {
			return runner.TaskResult{}, err
		}
		if err := protectCaseFiles(client.work, task.Cases); err != nil {
			return runner.TaskResult{}, err
		}
	}
	judgeCommand := ""
	if task.Mode == runner.ModeCustom {
		var judgeBuild runner.CompileResult
		customStartedAt := time.Now()
		judgeCommand, judgeBuild, err = prepareContainerCustomJudge(ctx, client.work, task.Limits, client.judgeCache)
		logTask(client.logf, task.SubmissionID, task.Attempt, "custom_judge_build=%s ok=%t reported=%dms", formatDuration(time.Since(customStartedAt)), judgeBuild.OK, judgeBuild.TimeMS)
		if err != nil {
			return runner.TaskResult{}, err
		}
		if !judgeBuild.OK {
			result.Verdict = runner.VerdictSystemError
			result.Message = "custom judge compile failed: " + judgeBuild.Message
			result.TimeMS = judgeBuild.TimeMS
			return result, nil
		}
	}

	totalScore := 0
	verdict := runner.VerdictAccepted
	maxTime := 0
	maxMemory := 0
	caseResults := make([]runner.CaseResult, 0, len(task.Cases))
	totalCases := int64(len(task.Cases))
	client.reportProgress("judge", 0, &totalCases)
	for index, item := range task.Cases {
		caseStartedAt := time.Now()
		got, err := client.runCase(ctx, runner.RunCaseRequest{
			TaskID:       client.taskID,
			JudgeCommand: judgeCommand,
			Case:         item,
			Mode:         task.Mode,
			Limits:       task.Limits,
		})
		logTask(client.logf, task.SubmissionID, task.Attempt, "case=%d id=%s elapsed=%s verdict=%s reported=%dms", index+1, item.ID, formatDuration(time.Since(caseStartedAt)), got.Verdict, got.TimeMS)
		if err != nil {
			return runner.TaskResult{}, err
		}
		caseResults = append(caseResults, got)
		totalScore += got.Score
		if got.TimeMS > maxTime {
			maxTime = got.TimeMS
		}
		if got.MemoryKB > maxMemory {
			maxMemory = got.MemoryKB
		}
		if verdict == runner.VerdictAccepted && got.Verdict != runner.VerdictAccepted {
			verdict = got.Verdict
		}
		done := int64(index + 1)
		client.reportProgress("judge", done, &totalCases)
	}
	result.Verdict = verdict
	result.Score = totalScore
	result.TimeMS = maxTime
	result.MemoryKB = maxMemory
	result.Cases = caseResults
	return result, nil
}

func (client runnerClient) reportProgress(stage string, done int64, total *int64) {
	if client.progress != nil {
		client.progress(stage, done, total)
	}
}

func (client runnerClient) hello() error {
	if err := client.codec.Send(runner.Message{Kind: runner.MsgHello, Hello: &runner.Hello{Role: "judger", Version: runner.Version}}); err != nil {
		return err
	}
	msg, err := client.codec.Recv()
	if err != nil {
		return err
	}
	if msg.Kind != runner.MsgHello || msg.Hello == nil || msg.Hello.Role != "runner" || msg.Hello.Version != runner.Version {
		return fmt.Errorf("runner hello got %s", msg.Kind)
	}
	return nil
}

func (client runnerClient) compile(task runner.Task, compileCommand string, compileMS int, userCommand string) (runner.CompileResult, error) {
	if err := client.codec.Send(runner.Message{Kind: runner.MsgCompile, Compile: &runner.CompileRequest{
		TaskID:         client.taskID,
		CompileCommand: compileCommand,
		CompileMS:      compileMS,
		UserCommand:    userCommand,
		Limits:         task.Limits,
	}}); err != nil {
		return runner.CompileResult{}, err
	}
	msg, err := client.codec.Recv()
	if err != nil {
		return runner.CompileResult{}, err
	}
	if msg.Kind == runner.MsgError {
		return runner.CompileResult{}, errors.New(msg.Error)
	}
	if msg.Kind != runner.MsgCompileResult || msg.CompileResult == nil {
		return runner.CompileResult{}, fmt.Errorf("runner compile got %s", msg.Kind)
	}
	return *msg.CompileResult, nil
}

func (client runnerClient) runCase(ctx context.Context, req runner.RunCaseRequest) (result runner.CaseResult, err error) {
	startedAt := time.Now()
	if err := client.codec.Send(runner.Message{Kind: runner.MsgRunCase, RunCase: &req}); err != nil {
		return runner.CaseResult{}, err
	}
	var cgroup *runner.CgroupCase
	stopCgroupWatch := func() {}
	defer func() {
		stopCgroupWatch()
		if cgroup != nil {
			if cleanupErr := cgroup.Cleanup(); cleanupErr != nil && err == nil {
				err = fmt.Errorf("cleanup user cgroup: %w", cleanupErr)
			}
		}
	}()
	for {
		msg, err := client.codec.Recv()
		if err != nil {
			return runner.CaseResult{}, err
		}
		switch msg.Kind {
		case runner.MsgUserPID:
			if msg.UserPID == nil {
				return runner.CaseResult{}, fmt.Errorf("runner sent empty user pid")
			}
			if cgroup == nil && client.cgroupRoot != "" {
				cg, err := client.prepareUserCgroup(msg.UserPID)
				if err != nil {
					return runner.CaseResult{}, err
				}
				cgroup = cg
				stopCgroupWatch = watchCgroupMemoryLimit(cgroup)
			}
			if err := client.codec.Send(runner.Message{Kind: runner.MsgReleaseUser, ReleaseUser: &runner.ReleaseUser{
				TaskID: msg.UserPID.TaskID,
				CaseID: msg.UserPID.CaseID,
			}}); err != nil {
				return runner.CaseResult{}, err
			}
		case runner.MsgCaseResult:
			if msg.CaseResult == nil {
				return runner.CaseResult{}, fmt.Errorf("runner sent empty case result")
			}
			result := *msg.CaseResult
			if cgroup != nil {
				applyCgroupStats(&result, cgroup)
			}
			if result.TimeMS == 0 && (result.Verdict == runner.VerdictTimeLimit || result.Verdict == runner.VerdictMemoryLimit) {
				result.TimeMS = elapsedMS(startedAt)
			}
			return result, nil
		case runner.MsgError:
			return runner.CaseResult{}, errors.New(msg.Error)
		default:
			if err := ctx.Err(); err != nil {
				return runner.CaseResult{CaseID: req.Case.ID, Verdict: runner.VerdictTimeLimit, TimeMS: elapsedMS(startedAt), Message: err.Error()}, nil
			}
			return runner.CaseResult{}, fmt.Errorf("runner run_case got %s", msg.Kind)
		}
	}
}
