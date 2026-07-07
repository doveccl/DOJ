package judger

type LeaseRequest struct {
	Version string `json:"version"`
	Host    string `json:"host"`
	Arch    string `json:"arch"`
}

type LeaseResponse struct {
	Task *TaskPayload `json:"task"`
}

type TaskPayload struct {
	ID           uint           `json:"id"`
	SubmissionID uint           `json:"submissionId"`
	Attempt      int            `json:"attempt"`
	Source       string         `json:"source"`
	Lang         LangPayload    `json:"lang"`
	Mode         string         `json:"mode"`
	Limits       LimitsPayload  `json:"limits"`
	Cases        []CasePayload  `json:"cases"`
	Problem      ProblemPayload `json:"problem"`
}

type LangPayload struct {
	ID      string `json:"id"`
	Source  string `json:"source"`
	Image   string `json:"image"`
	Compile string `json:"compile"`
	Run     string `json:"run"`
}

type ProblemPayload struct {
	ID          uint     `json:"id"`
	Mode        string   `json:"mode"`
	TimeMS      int      `json:"timeMs"`
	MemoryMB    int      `json:"memoryMb"`
	Tags        []string `json:"tags"`
	PackageHash string   `json:"packageHash"`
}

type LimitsPayload struct {
	TimeMS   int `json:"timeMs"`
	MemoryKB int `json:"memoryKb"`
	OutputKB int `json:"outputKb"`
	Pids     int `json:"pids"`
	FileKB   int `json:"fileKb"`
}

type CasePayload struct {
	ID     string `json:"id"`
	Input  string `json:"input"`
	Answer string `json:"answer"`
	Score  int    `json:"score"`
}

type ResultRequest struct {
	SubmissionID uint         `json:"submissionId"`
	Attempt      int          `json:"attempt"`
	Status       string       `json:"status"`
	Score        int          `json:"score"`
	Message      string       `json:"message"`
	TimeMS       *int         `json:"timeMs,omitempty"`
	MemoryKB     *int         `json:"memoryKb,omitempty"`
	Cases        []CaseResult `json:"cases"`
}

type HeartbeatRequest struct {
	SubmissionID uint   `json:"submissionId"`
	Attempt      int    `json:"attempt"`
	Stage        string `json:"stage,omitempty"`
	Done         int64  `json:"done,omitempty"`
	Total        *int64 `json:"total,omitempty"`
}

type CaseResult struct {
	No       int    `json:"no"`
	Status   string `json:"status"`
	Score    int    `json:"score"`
	TimeMS   *int   `json:"timeMs,omitempty"`
	MemoryKB *int   `json:"memoryKb,omitempty"`
	Message  string `json:"message"`
}
