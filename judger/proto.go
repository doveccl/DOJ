package judger

import (
	"encoding/gob"
	"errors"
	"io"
	"sync"
)

type MessageKind string

const (
	MsgHello         MessageKind = "hello"
	MsgCompile       MessageKind = "compile"
	MsgCompileResult MessageKind = "compile_result"
	MsgRunCase       MessageKind = "run_case"
	MsgUserPID       MessageKind = "user_pid"
	MsgReleaseUser   MessageKind = "release_user"
	MsgCaseResult    MessageKind = "case_result"
	MsgError         MessageKind = "error"
	MsgBye           MessageKind = "bye"
)

type Message struct {
	Kind          MessageKind
	Hello         *Hello
	Compile       *CompileRequest
	CompileResult *CompileResult
	RunCase       *RunCaseRequest
	UserPID       *UserPID
	ReleaseUser   *ReleaseUser
	CaseResult    *CaseResult
	Error         string
}

type Hello struct {
	Role    string
	Version string
}

type CompileRequest struct {
	TaskID         string
	CompileCommand string
	UserCommand    string
	Limits         Limits
}

type RunCaseRequest struct {
	TaskID       string
	JudgeCommand string
	Case         Case
	Mode         JudgeMode
	Limits       Limits
}

type UserPID struct {
	TaskID string
	CaseID string
	PID    int
}

type ReleaseUser struct {
	TaskID string
	CaseID string
}

type Codec struct {
	enc *gob.Encoder
	dec *gob.Decoder
	mu  sync.Mutex
}

func NewCodec(rw io.ReadWriter) *Codec {
	return &Codec{
		enc: gob.NewEncoder(rw),
		dec: gob.NewDecoder(rw),
	}
}

func (c *Codec) Send(msg Message) error {
	if msg.Kind == "" {
		return errors.New("judger proto: empty message kind")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enc.Encode(msg)
}

func (c *Codec) Recv() (Message, error) {
	var msg Message
	if err := c.dec.Decode(&msg); err != nil {
		return Message{}, err
	}
	if msg.Kind == "" {
		return Message{}, errors.New("judger proto: empty message kind")
	}
	return msg, nil
}

func init() {
	gob.Register(Message{})
	gob.Register(Hello{})
	gob.Register(CompileRequest{})
	gob.Register(CompileResult{})
	gob.Register(RunCaseRequest{})
	gob.Register(UserPID{})
	gob.Register(ReleaseUser{})
	gob.Register(CaseResult{})
}
