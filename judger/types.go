package judger

import jr "github.com/doveccl/doj/judger/runner"

type Verdict = jr.Verdict

const (
	VerdictAccepted          = jr.VerdictAccepted
	VerdictCompileError      = jr.VerdictCompileError
	VerdictWrongAnswer       = jr.VerdictWrongAnswer
	VerdictPresentationError = jr.VerdictPresentationError
	VerdictTimeLimit         = jr.VerdictTimeLimit
	VerdictMemoryLimit       = jr.VerdictMemoryLimit
	VerdictOutputLimit       = jr.VerdictOutputLimit
	VerdictRuntimeError      = jr.VerdictRuntimeError
	VerdictSystemError       = jr.VerdictSystemError
)

type JudgeMode = jr.JudgeMode

const (
	ModeDefault = jr.ModeDefault
	ModeStrict  = jr.ModeStrict
	ModeCustom  = jr.ModeCustom
)

type Lang = jr.Lang
type Limits = jr.Limits
type Case = jr.Case
type Task = jr.Task
type CompileResult = jr.CompileResult
type CaseResult = jr.CaseResult
type TaskResult = jr.TaskResult

type MessageKind = jr.MessageKind

const (
	MsgHello         = jr.MsgHello
	MsgCompile       = jr.MsgCompile
	MsgCompileResult = jr.MsgCompileResult
	MsgRunCase       = jr.MsgRunCase
	MsgUserPID       = jr.MsgUserPID
	MsgReleaseUser   = jr.MsgReleaseUser
	MsgCaseResult    = jr.MsgCaseResult
	MsgError         = jr.MsgError
	MsgBye           = jr.MsgBye
)

type Message = jr.Message
type Hello = jr.Hello
type CompileRequest = jr.CompileRequest
type RunCaseRequest = jr.RunCaseRequest
type UserPID = jr.UserPID
type ReleaseUser = jr.ReleaseUser
type Codec = jr.Codec
type CgroupConfig = jr.CgroupConfig
type CgroupStats = jr.CgroupStats
type CgroupCase = jr.CgroupCase

var NewCodec = jr.NewCodec
var PrepareCgroup = jr.PrepareCgroup

func DefaultCgroupRoot() string {
	return jr.DefaultCgroupRoot()
}
