package models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:32;uniqueIndex:idx_users_name_ci,expression:LOWER(name);not null" json:"name"`
	Mail      string         `gorm:"size:256;uniqueIndex:idx_users_mail_ci,expression:LOWER(mail);not null" json:"mail"`
	Auth      string         `gorm:"size:128;not null" json:"-"`
	Bio       string         `gorm:"size:256;not null;default:''" json:"bio"`
	Avatar    string         `gorm:"size:512;not null;default:''" json:"avatar"`
	Admin     bool           `gorm:"not null;default:false" json:"admin"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Group struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64;uniqueIndex;not null" json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type GroupUser struct {
	GroupID uint `gorm:"primaryKey;index" json:"groupId"`
	UserID  uint `gorm:"primaryKey;index" json:"userId"`
}

type Problem struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Title     string         `gorm:"size:256;not null" json:"title"`
	Tags      datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"tags"`
	Visible   bool           `gorm:"not null;default:false" json:"visible"`
	Mode      string         `gorm:"size:16;not null;default:'default'" json:"mode"`
	TimeMS    int            `gorm:"not null;default:1000" json:"timeMs"`
	MemoryMB  int            `gorm:"not null;default:256" json:"memoryMb"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Language struct {
	ID        string    `gorm:"size:32;primaryKey" json:"id"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	Source    string    `gorm:"size:128;not null" json:"source"`
	Image     string    `gorm:"size:256;not null;default:''" json:"image"`
	Compile   string    `gorm:"type:text;not null;default:''" json:"compile"`
	Run       string    `gorm:"type:text;not null;default:''" json:"run"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Submission struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       uint       `gorm:"index;not null" json:"userId"`
	ProblemID    uint       `gorm:"index;not null" json:"problemId"`
	AssignmentID *uint      `gorm:"index" json:"assignmentId"`
	ContestID    *uint      `gorm:"index" json:"contestId"`
	Language     string     `gorm:"size:32;index;not null" json:"language"`
	Code         string     `gorm:"type:text;not null" json:"code"`
	Status       string     `gorm:"size:32;not null;default:'queued';index" json:"status"`
	Score        int        `gorm:"not null;default:0" json:"score"`
	Message      string     `gorm:"type:text;not null;default:''" json:"message"`
	Attempt      int        `gorm:"not null;default:0" json:"attempt"`
	JudgerID     *uint      `gorm:"index" json:"judgerId"`
	LeaseUntil   *time.Time `gorm:"index" json:"leaseUntil"`
	TimeMS       *int       `json:"timeMs"`
	MemoryKB     *int       `json:"memoryKb"`
	Public       bool       `gorm:"not null;default:false" json:"public"`
	CreatedAt    time.Time  `gorm:"index" json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type Case struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	SubmissionID uint   `gorm:"uniqueIndex:idx_case_submission_no;not null" json:"submissionId"`
	No           int    `gorm:"uniqueIndex:idx_case_submission_no;not null" json:"no"`
	Status       string `gorm:"size:32;not null" json:"status"`
	TimeMS       *int   `json:"timeMs"`
	MemoryKB     *int   `json:"memoryKb"`
	Message      string `gorm:"size:1024;not null;default:''" json:"message"`
}

type Judger struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64;uniqueIndex;not null" json:"name"`
	Auth      string    `gorm:"size:128;not null" json:"-"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Assignment struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Title       string         `gorm:"size:256;not null" json:"title"`
	Description string         `gorm:"type:text;not null;default:''" json:"description"`
	EndAt       time.Time      `gorm:"index;not null" json:"endAt"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type AssignmentProblem struct {
	AssignmentID uint   `gorm:"primaryKey;index" json:"assignmentId"`
	ProblemID    uint   `gorm:"primaryKey;index" json:"problemId"`
	Sort         string `gorm:"size:16;not null;default:''" json:"sort"`
}

type AssignmentUser struct {
	AssignmentID uint `gorm:"primaryKey;index" json:"assignmentId"`
	UserID       uint `gorm:"primaryKey;index" json:"userId"`
}

type AssignmentGroup struct {
	AssignmentID uint `gorm:"primaryKey;index" json:"assignmentId"`
	GroupID      uint `gorm:"primaryKey;index" json:"groupId"`
}

type Contest struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Title       string         `gorm:"size:256;not null" json:"title"`
	Description string         `gorm:"type:text;not null;default:''" json:"description"`
	Kind        string         `gorm:"size:8;not null;default:'OI'" json:"kind"`
	StartAt     time.Time      `gorm:"index;not null" json:"startAt"`
	EndAt       time.Time      `gorm:"index;not null" json:"endAt"`
	FreezeAt    *time.Time     `json:"freezeAt"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type ContestProblem struct {
	ContestID uint   `gorm:"primaryKey;index" json:"contestId"`
	ProblemID uint   `gorm:"primaryKey;index" json:"problemId"`
	Sort      string `gorm:"size:16;not null;default:''" json:"sort"`
}

type Discussion struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Title     string         `gorm:"size:256;not null" json:"title"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	UserID    uint           `gorm:"index;not null" json:"userId"`
	Tags      datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"tags"`
	Pinned    bool           `gorm:"not null;default:false;index" json:"pinned"`
	Locked    bool           `gorm:"not null;default:false" json:"locked"`
	CreatedAt time.Time      `gorm:"index" json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Comment struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	DiscussionID uint           `gorm:"index;not null" json:"discussionId"`
	UserID       uint           `gorm:"index;not null" json:"userId"`
	Content      string         `gorm:"type:text;not null" json:"content"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type Setting struct {
	Key   string         `gorm:"size:64;primaryKey" json:"key"`
	Value datatypes.JSON `gorm:"type:jsonb;not null" json:"value"`
}

func All() []any {
	return []any{
		&User{},
		&Group{},
		&GroupUser{},
		&Problem{},
		&Language{},
		&Submission{},
		&Case{},
		&Judger{},
		&Assignment{},
		&AssignmentProblem{},
		&AssignmentUser{},
		&AssignmentGroup{},
		&Contest{},
		&ContestProblem{},
		&Discussion{},
		&Comment{},
		&Setting{},
	}
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(All()...)
}
