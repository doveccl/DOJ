package admin

import (
	"time"

	"github.com/doveccl/doj/server/settings"
)

type Members struct {
	Users  []User  `json:"users"`
	Groups []Group `json:"groups"`
}

type PageResult[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
}

type Judgers struct {
	Judgers []Judger   `json:"judgers"`
	Queue   JudgeQueue `json:"queue"`
}

type AdminSettings = settings.Settings

type AdminSettingsPatch = settings.Patch

type User struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Mail   string `json:"mail"`
	Role   string `json:"role"`
	Groups []uint `json:"groups"`
}

type Group struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Users []uint `json:"users"`
}

type Language struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Source    string `json:"source"`
	Image     string `json:"image"`
	Compile   string `json:"compile"`
	CompileMS int    `json:"compileMs"`
	Run       string `json:"run"`
}

type Judger struct {
	ID            uint       `json:"id"`
	Name          string     `json:"name"`
	Token         string     `json:"token,omitempty"`
	Online        bool       `json:"online"`
	ConnectedAt   *time.Time `json:"connectedAt"`
	ActiveAt      *time.Time `json:"activeAt"`
	UptimeSeconds int        `json:"uptimeSeconds"`
}

type JudgeQueue struct {
	Queued  int `json:"queued"`
	Running int `json:"running"`
	Done    int `json:"done"`
}

type UserUpdate struct {
	Role   string `json:"role"`
	Groups []uint `json:"groups"`
}

type UserCreate struct {
	Name     string `json:"name"`
	Mail     string `json:"mail"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Groups   []uint `json:"groups"`
}

type PasswordReset struct {
	Password string `json:"password"`
}

type GroupUpdate struct {
	Name  string `json:"name"`
	Users []uint `json:"users"`
}

type LanguageUpdate struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Source    string `json:"source"`
	Image     string `json:"image"`
	Compile   string `json:"compile"`
	CompileMS int    `json:"compileMs"`
	Run       string `json:"run"`
}

type LanguageCreate struct {
	LanguageUpdate
}

type JudgerUpdate struct {
	Name string `json:"name"`
	Auth string `json:"auth"`
}

type JudgerCreate struct {
	Name string `json:"name"`
}
