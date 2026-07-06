package web

import (
	"time"
)

type Home struct {
	Notice      string           `json:"notice"`
	Heatmap     []HeatCell       `json:"heatmap"`
	Problems    []HomeProblem    `json:"problems"`
	Assignments []HomeAssignment `json:"assignments"`
	Contests    []HomeContest    `json:"contests"`
}

type CreatedID struct {
	ID uint `json:"id"`
}

type CountResult struct {
	Count int64 `json:"count"`
}

type NoticeUpdate struct {
	Content string `json:"content"`
}

type HeatCell struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type Item struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
}

type HomeProblem struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
}

type HomeAssignment struct {
	ID     uint   `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Total  int    `json:"total"`
	Done   int    `json:"done"`
}

type HomeContest struct {
	ID     uint   `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type PageResult[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
}

type MeDTO struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Mail   string `json:"mail"`
	Bio    string `json:"bio"`
	Avatar string `json:"avatar"`
	Admin  bool   `json:"admin"`
}

type LanguageDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Source string `json:"source"`
}

type MeUpdate struct {
	Mail   *string `json:"mail,omitempty"`
	Bio    *string `json:"bio,omitempty"`
	Avatar *string `json:"avatar,omitempty"`
}

type PasswordUpdate struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Mail     string `json:"mail"`
	Password string `json:"password"`
}

type AssignmentDTO struct {
	ID     uint      `json:"id"`
	Title  string    `json:"title"`
	EndAt  time.Time `json:"endAt"`
	Status string    `json:"status"`
	Total  int       `json:"total"`
	Done   int       `json:"done"`
	Users  []uint    `json:"users,omitempty"`
	Groups []uint    `json:"groups,omitempty"`
}

type AssignmentCreate struct {
	Title    string       `json:"title"`
	EndAt    string       `json:"endAt"`
	Problems []ProblemRef `json:"problems"`
	Users    []uint       `json:"users"`
	Groups   []uint       `json:"groups"`
}

type AssignmentUpdate struct {
	Title    string       `json:"title"`
	EndAt    string       `json:"endAt"`
	Problems []ProblemRef `json:"problems"`
	Users    []uint       `json:"users"`
	Groups   []uint       `json:"groups"`
}

type ProblemRef struct {
	ID   uint   `json:"id"`
	Sort string `json:"sort"`
}

type AssignmentDetail struct {
	Assignment AssignmentDTO           `json:"assignment"`
	Problems   []ProblemDTO            `json:"problems"`
	Progress   []AssignmentProgressDTO `json:"progress"`
}

type AssignmentProgressDTO struct {
	User     string                         `json:"user"`
	AC       int                            `json:"ac"`
	Submit   int                            `json:"submit"`
	Problems []AssignmentProblemProgressDTO `json:"problems"`
}

type AssignmentProblemProgressDTO struct {
	ProblemID uint   `json:"problemId"`
	Status    string `json:"status"`
	Score     *int   `json:"score,omitempty"`
}

type ContestDTO struct {
	ID       uint       `json:"id"`
	Title    string     `json:"title"`
	Kind     string     `json:"kind"`
	StartAt  time.Time  `json:"startAt"`
	EndAt    time.Time  `json:"endAt"`
	FreezeAt *time.Time `json:"freezeAt"`
	Status   string     `json:"status"`
	Total    int        `json:"total"`
}

type ContestCreate struct {
	Title    string       `json:"title"`
	Kind     string       `json:"kind"`
	StartAt  string       `json:"startAt"`
	EndAt    string       `json:"endAt"`
	FreezeAt string       `json:"freezeAt"`
	Problems []ProblemRef `json:"problems"`
}

type ContestUpdate struct {
	Title    string       `json:"title"`
	Kind     string       `json:"kind"`
	StartAt  string       `json:"startAt"`
	EndAt    string       `json:"endAt"`
	FreezeAt string       `json:"freezeAt"`
	Problems []ProblemRef `json:"problems"`
}

type ContestDetail struct {
	Contest  ContestDTO    `json:"contest"`
	Problems []ProblemDTO  `json:"problems"`
	Rank     []RankUserDTO `json:"rank"`
}

type SubmissionListItem struct {
	ID           uint      `json:"id"`
	ProblemID    uint      `json:"problemId"`
	ProblemTitle string    `json:"problemTitle"`
	User         string    `json:"user"`
	Language     string    `json:"language"`
	Status       string    `json:"status"`
	TimeMS       *int      `json:"timeMs,omitempty"`
	MemoryKB     *int      `json:"memoryKb,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type SubmissionDTO struct {
	ID           uint      `json:"id"`
	ProblemID    uint      `json:"problemId"`
	ProblemTitle string    `json:"problemTitle"`
	User         string    `json:"user"`
	Language     string    `json:"language"`
	Status       string    `json:"status"`
	Score        int       `json:"score"`
	Message      string    `json:"message"`
	TimeMS       *int      `json:"timeMs,omitempty"`
	MemoryKB     *int      `json:"memoryKb,omitempty"`
	Public       bool      `json:"public"`
	CreatedAt    time.Time `json:"createdAt"`
}

type SubmissionDetail struct {
	Submission SubmissionDTO `json:"submission"`
	Code       string        `json:"code"`
	Cases      []CaseDTO     `json:"cases"`
	Progress   *ProgressDTO  `json:"progress,omitempty"`
}

type ProgressDTO struct {
	Stage     string    `json:"stage"`
	Done      int64     `json:"done"`
	Total     *int64    `json:"total,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SubmitRequest struct {
	ProblemID uint   `json:"problemId"`
	Language  string `json:"language"`
	Code      string `json:"code"`
	Public    bool   `json:"public"`
}

type SubmissionUpdate struct {
	Public bool `json:"public"`
}

type CaseDTO struct {
	No       int    `json:"no"`
	Status   string `json:"status"`
	TimeMS   *int   `json:"timeMs,omitempty"`
	MemoryKB *int   `json:"memoryKb,omitempty"`
	Message  string `json:"message"`
}

type RankUserDTO struct {
	Rank     int              `json:"rank"`
	User     string           `json:"user"`
	Bio      string           `json:"bio"`
	Avatar   string           `json:"avatar"`
	AC       int              `json:"ac"`
	Submit   int              `json:"submit"`
	Score    int              `json:"score"`
	Penalty  int              `json:"penalty"`
	Problems []RankProblemDTO `json:"problems"`
}

type RankProblemDTO struct {
	ProblemID uint   `json:"problemId"`
	Status    string `json:"status"`
	Submit    int    `json:"submit"`
	Score     int    `json:"score"`
	Penalty   int    `json:"penalty"`
}

type PublicUserDTO struct {
	Name   string `json:"name"`
	Bio    string `json:"bio"`
	Avatar string `json:"avatar"`
	Admin  bool   `json:"admin"`
	AC     int    `json:"ac"`
	Submit int    `json:"submit"`
}

type UserOptionDTO struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type UserProfile struct {
	User       PublicUserDTO             `json:"user"`
	Heatmap    []HeatCell                `json:"heatmap"`
	Solved     PageResult[SolvedProblem] `json:"solved"`
	Activities []UserActivityDTO         `json:"activities"`
}

type SolvedProblem struct {
	ID    uint     `json:"id"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}

type UserActivityDTO struct {
	Type         string    `json:"type"`
	ID           uint      `json:"id"`
	Title        string    `json:"title"`
	Status       string    `json:"status,omitempty"`
	ProblemID    uint      `json:"problemId,omitempty"`
	ProblemTitle string    `json:"problemTitle,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type DiscussionDTO struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Tags      []string  `json:"tags"`
	Pinned    bool      `json:"pinned"`
	Locked    bool      `json:"locked"`
	Replies   int       `json:"replies"`
	CreatedAt time.Time `json:"createdAt"`
}

type DiscussionCreate struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

type DiscussionUpdate struct {
	Title   *string   `json:"title,omitempty"`
	Content *string   `json:"content,omitempty"`
	Tags    *[]string `json:"tags,omitempty"`
	Pinned  *bool     `json:"pinned,omitempty"`
	Locked  *bool     `json:"locked,omitempty"`
}

type DiscussionDetail struct {
	Discussion DiscussionDTO `json:"discussion"`
	Content    string        `json:"content"`
	Comments   []CommentDTO  `json:"comments"`
}

type CommentDTO struct {
	ID        uint      `json:"id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	Deleted   bool      `json:"deleted"`
	CreatedAt time.Time `json:"createdAt"`
}

type CommentCreate struct {
	Content string `json:"content"`
}

type ProblemDTO struct {
	ID        uint     `json:"id"`
	Sort      string   `json:"sort,omitempty"`
	Title     string   `json:"title"`
	Statement string   `json:"statement,omitempty"`
	Tags      []string `json:"tags"`
	Visible   bool     `json:"visible"`
	Mode      string   `json:"mode"`
	TimeMS    int      `json:"timeMs"`
	MemoryMB  int      `json:"memoryMb"`
	Cases     *int     `json:"cases,omitempty"`
	DataBytes *int64   `json:"dataBytes,omitempty"`
}

type ProblemStateDTO struct {
	ProblemID   uint       `json:"problemId"`
	AC          int        `json:"ac"`
	Submit      int        `json:"submit"`
	Discussions *int       `json:"discussions,omitempty"`
	Status      string     `json:"status"`
	Submission  *RecordDTO `json:"submission,omitempty"`
}

type RecordDTO struct {
	ID        uint      `json:"id"`
	Status    string    `json:"status"`
	Score     int       `json:"score"`
	CreatedAt time.Time `json:"createdAt"`
}

type ProblemCreate struct {
	Title    string   `json:"title"`
	Tags     []string `json:"tags"`
	Visible  *bool    `json:"visible"`
	Mode     string   `json:"mode"`
	TimeMS   int      `json:"timeMs"`
	MemoryMB int      `json:"memoryMb"`
}

type ProblemUpdate struct {
	Title     *string   `json:"title,omitempty"`
	Statement *string   `json:"statement,omitempty"`
	Tags      *[]string `json:"tags,omitempty"`
	Visible   *bool     `json:"visible,omitempty"`
	Mode      *string   `json:"mode,omitempty"`
	TimeMS    *int      `json:"timeMs,omitempty"`
	MemoryMB  *int      `json:"memoryMb,omitempty"`
}

type ProblemVisibilityUpdate struct {
	Visible bool `json:"visible"`
}

type ProblemAssets struct {
	Data      []AssetFile `json:"data"`
	Judge     []AssetFile `json:"judge"`
	Assets    []AssetFile `json:"assets"`
	Cases     int         `json:"cases"`
	DataBytes int64       `json:"dataBytes"`
}

type AssetFile struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Editable bool   `json:"editable"`
}

type AssetCaseCreate struct {
	Name   string `json:"name"`
	Input  string `json:"input"`
	Output string `json:"output"`
}

type AssetContent struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

type AssetContentUpdate struct {
	Key     string `json:"key"`
	Content string `json:"content"`
}

type UploadResult struct {
	URL string `json:"url"`
}
