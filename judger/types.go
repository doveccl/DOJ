package judger

type Verdict string

const (
	VerdictAccepted          Verdict = "AC"
	VerdictCompileError      Verdict = "CE"
	VerdictWrongAnswer       Verdict = "WA"
	VerdictPresentationError Verdict = "PE"
	VerdictTimeLimit         Verdict = "TLE"
	VerdictMemoryLimit       Verdict = "MLE"
	VerdictOutputLimit       Verdict = "OLE"
	VerdictRuntimeError      Verdict = "RE"
	VerdictSystemError       Verdict = "SE"
)

type JudgeMode string

const (
	ModeDefault JudgeMode = "default"
	ModeStrict  JudgeMode = "strict"
	ModeCustom  JudgeMode = "custom"
)

type Lang struct {
	ID      string `json:"id"`
	Source  string `json:"source"`
	Image   string `json:"image"`
	Compile string `json:"compile"`
	Run     string `json:"run"`
}

type Limits struct {
	TimeMS   int
	MemoryKB int
	OutputKB int
	Pids     int
	FileKB   int
}

type Case struct {
	ID     string
	Input  string
	Answer string
	Score  int
}

type Task struct {
	SubmissionID uint
	Attempt      int
	Source       string
	Lang         Lang
	Mode         JudgeMode
	Limits       Limits
	Cases        []Case
}

type CompileResult struct {
	OK      bool
	Message string
	TimeMS  int
}

type CaseResult struct {
	CaseID   string  `json:"caseId"`
	Verdict  Verdict `json:"verdict"`
	Score    int     `json:"score"`
	TimeMS   int     `json:"timeMs"`
	MemoryKB int     `json:"memoryKb"`
	Message  string  `json:"message"`
}

type TaskResult struct {
	SubmissionID uint
	Attempt      int
	Verdict      Verdict
	Score        int
	TimeMS       int
	MemoryKB     int
	Message      string
	Cases        []CaseResult
}
