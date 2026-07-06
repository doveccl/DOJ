package judger

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

type runnerClient struct {
	codec      *Codec
	cgroupRoot string
	procRoot   string
	initPID    int
	taskID     string
	limits     Limits
	work       string
	judgeCache string
	logf       func(format string, args ...any)
	progress   func(stage string, done int64, total *int64)
}

func (client runnerClient) runTask(ctx context.Context, task Task, compileCommand string, userCommand string, beforeCases func() error) (TaskResult, error) {
	result := TaskResult{SubmissionID: task.SubmissionID, Attempt: task.Attempt}
	if len(task.Cases) == 0 {
		result.Verdict = VerdictSystemError
		result.Message = "task has no cases"
		return result, nil
	}
	helloStartedAt := time.Now()
	if err := client.hello(); err != nil {
		return TaskResult{}, err
	}
	logStep(client.logf, task.SubmissionID, task.Attempt, "runner_hello", helloStartedAt)
	compileStartedAt := time.Now()
	client.reportProgress("compile", 0, nil)
	compile, err := client.compile(task, compileCommand, userCommand)
	logTask(client.logf, task.SubmissionID, task.Attempt, "compile=%s ok=%t reported=%dms", formatDuration(time.Since(compileStartedAt)), compile.OK, compile.TimeMS)
	if err != nil {
		return TaskResult{}, err
	}
	if !compile.OK {
		result.Verdict = VerdictCompileError
		result.Message = compile.Message
		return result, nil
	}
	if beforeCases != nil {
		if err := beforeCases(); err != nil {
			return TaskResult{}, err
		}
		if err := protectCaseFiles(client.work, task.Cases); err != nil {
			return TaskResult{}, err
		}
	}
	judgeCommand := ""
	if task.Mode == ModeCustom {
		var judgeBuild CompileResult
		customStartedAt := time.Now()
		judgeCommand, judgeBuild, err = prepareContainerCustomJudge(ctx, client.work, task.Limits, client.judgeCache)
		logTask(client.logf, task.SubmissionID, task.Attempt, "custom_judge_build=%s ok=%t reported=%dms", formatDuration(time.Since(customStartedAt)), judgeBuild.OK, judgeBuild.TimeMS)
		if err != nil {
			return TaskResult{}, err
		}
		if !judgeBuild.OK {
			result.Verdict = VerdictSystemError
			result.Message = "custom judge compile failed: " + judgeBuild.Message
			result.TimeMS = judgeBuild.TimeMS
			return result, nil
		}
	}

	totalScore := 0
	verdict := VerdictAccepted
	maxTime := 0
	maxMemory := 0
	caseResults := make([]CaseResult, 0, len(task.Cases))
	totalCases := int64(len(task.Cases))
	client.reportProgress("judge", 0, &totalCases)
	for index, item := range task.Cases {
		caseStartedAt := time.Now()
		got, err := client.runCase(ctx, RunCaseRequest{
			TaskID:       client.taskID,
			JudgeCommand: judgeCommand,
			Case:         item,
			Mode:         task.Mode,
			Limits:       task.Limits,
		})
		logTask(client.logf, task.SubmissionID, task.Attempt, "case=%d id=%s elapsed=%s verdict=%s reported=%dms", index+1, item.ID, formatDuration(time.Since(caseStartedAt)), got.Verdict, got.TimeMS)
		if err != nil {
			return TaskResult{}, err
		}
		caseResults = append(caseResults, got)
		totalScore += got.Score
		if got.TimeMS > maxTime {
			maxTime = got.TimeMS
		}
		if got.MemoryKB > maxMemory {
			maxMemory = got.MemoryKB
		}
		if verdict == VerdictAccepted && got.Verdict != VerdictAccepted {
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
	if err := client.codec.Send(Message{Kind: MsgHello, Hello: &Hello{Role: "judger", Version: Version}}); err != nil {
		return err
	}
	msg, err := client.codec.Recv()
	if err != nil {
		return err
	}
	if msg.Kind != MsgHello {
		return fmt.Errorf("runner hello got %s", msg.Kind)
	}
	return nil
}

func (client runnerClient) compile(task Task, compileCommand string, userCommand string) (CompileResult, error) {
	if err := client.codec.Send(Message{Kind: MsgCompile, Compile: &CompileRequest{
		TaskID:         client.taskID,
		CompileCommand: compileCommand,
		UserCommand:    userCommand,
		Limits:         task.Limits,
	}}); err != nil {
		return CompileResult{}, err
	}
	msg, err := client.codec.Recv()
	if err != nil {
		return CompileResult{}, err
	}
	if msg.Kind == MsgError {
		return CompileResult{}, errors.New(msg.Error)
	}
	if msg.Kind != MsgCompileResult || msg.CompileResult == nil {
		return CompileResult{}, fmt.Errorf("runner compile got %s", msg.Kind)
	}
	return *msg.CompileResult, nil
}

func (client runnerClient) runCase(ctx context.Context, req RunCaseRequest) (CaseResult, error) {
	startedAt := time.Now()
	if err := client.codec.Send(Message{Kind: MsgRunCase, RunCase: &req}); err != nil {
		return CaseResult{}, err
	}
	var cgroup *CgroupCase
	stopCgroupWatch := func() {}
	defer func() {
		stopCgroupWatch()
		if cgroup != nil {
			_ = cgroup.Cleanup()
		}
	}()
	for {
		msg, err := client.codec.Recv()
		if err != nil {
			return CaseResult{}, err
		}
		switch msg.Kind {
		case MsgUserPID:
			if msg.UserPID == nil {
				return CaseResult{}, fmt.Errorf("runner sent empty user pid")
			}
			if cgroup == nil && client.cgroupRoot != "" {
				cg, err := client.prepareUserCgroup(msg.UserPID)
				if err != nil {
					if !errors.Is(err, os.ErrNotExist) {
						return CaseResult{}, err
					}
				} else {
					cgroup = cg
					stopCgroupWatch = watchCgroupMemoryLimit(cgroup)
				}
			}
			if err := client.codec.Send(Message{Kind: MsgReleaseUser, ReleaseUser: &ReleaseUser{
				TaskID: msg.UserPID.TaskID,
				CaseID: msg.UserPID.CaseID,
			}}); err != nil {
				return CaseResult{}, err
			}
		case MsgCaseResult:
			if msg.CaseResult == nil {
				return CaseResult{}, fmt.Errorf("runner sent empty case result")
			}
			result := *msg.CaseResult
			if cgroup != nil {
				applyCgroupStats(&result, cgroup)
			}
			if result.TimeMS == 0 && (result.Verdict == VerdictTimeLimit || result.Verdict == VerdictMemoryLimit) {
				result.TimeMS = elapsedMS(startedAt)
			}
			return result, nil
		case MsgError:
			return CaseResult{}, errors.New(msg.Error)
		default:
			if err := ctx.Err(); err != nil {
				return CaseResult{CaseID: req.Case.ID, Verdict: VerdictTimeLimit, TimeMS: elapsedMS(startedAt), Message: err.Error()}, nil
			}
			return CaseResult{}, fmt.Errorf("runner run_case got %s", msg.Kind)
		}
	}
}
