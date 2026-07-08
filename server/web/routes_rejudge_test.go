package web

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/doveccl/doj/models"
	judgersvc "github.com/doveccl/doj/server/judger"
	"github.com/doveccl/doj/server/web/contract"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

func TestSubmissionCanBeRejudgedByAdmin(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	owner := models.User{Name: "owner", Mail: "owner@example.com", Auth: "hash"}
	other := models.User{Name: "other", Mail: "other@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	for _, user := range []*models.User{&owner, &other, &admin} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	timeMS := 123
	memoryKB := 456
	leaseUntil := time.Now().Add(time.Minute)
	submission := models.Submission{
		UserID:     owner.ID,
		ProblemID:  problem.ID,
		Language:   "cpp",
		Code:       "code",
		Status:     "WA",
		Score:      20,
		Message:    "wrong",
		Attempt:    3,
		JudgerID:   &admin.ID,
		LeaseUntil: &leaseUntil,
		TimeMS:     &timeMS,
		MemoryKB:   &memoryKB,
		Public:     true,
	}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	if err := db.Create(&models.Case{SubmissionID: submission.ID, No: 1, Status: "WA", TimeMS: &timeMS, MemoryKB: &memoryKB, Message: "bad"}).Error; err != nil {
		t.Fatalf("create case: %v", err)
	}
	if err := judgersvc.SaveProgress(t.Context(), submission.ID, judgersvc.Progress{Attempt: 3, Stage: "judge", Done: 1, UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("save progress: %v", err)
	}

	e := echo.New()
	Register(e, db)
	target := "/api/submissions/" + strconv.FormatUint(uint64(submission.ID), 10) + "/rejudge"
	if res := requestWithCookies(e, http.MethodPost, target, nil, nil); res.Code != http.StatusForbidden {
		t.Fatalf("guest should not rejudge submission, got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodPost, target, databaseSession(t, db, other.ID), nil); res.Code != http.StatusForbidden {
		t.Fatalf("non-admin should not rejudge submission, got %d body=%s", res.Code, res.Body.String())
	}
	adminRes := requestWithCookies(e, http.MethodPost, target, databaseSession(t, db, admin.ID), nil)
	if adminRes.Code != http.StatusOK {
		t.Fatalf("admin should rejudge submission, got %d body=%s", adminRes.Code, adminRes.Body.String())
	}
	gotID := decodeJSON[contract.CreatedID](t, adminRes)
	if gotID.ID != submission.ID {
		t.Fatalf("rejudge should return submission id: %+v", gotID)
	}
	var got models.Submission
	if err := db.First(&got, submission.ID).Error; err != nil {
		t.Fatalf("reload submission: %v", err)
	}
	if got.Status != "queued" || got.Score != 0 || got.Message != "" || got.Attempt != 4 || got.JudgerID != nil || got.LeaseUntil != nil || got.TimeMS != nil || got.MemoryKB != nil {
		t.Fatalf("rejudged submission = %+v", got)
	}
	var cases int64
	if err := db.Model(&models.Case{}).Where("submission_id = ?", submission.ID).Count(&cases).Error; err != nil {
		t.Fatalf("count cases: %v", err)
	}
	if cases != 0 {
		t.Fatalf("cases after rejudge = %d, want 0", cases)
	}
	progress, err := judgersvc.ReadProgress(t.Context(), submission.ID, 3)
	if err != nil {
		t.Fatalf("read progress after rejudge: %v", err)
	}
	if progress != nil {
		t.Fatalf("progress after rejudge = %+v, want nil", progress)
	}
}

func TestProblemSubmissionsCanBeRejudgedByAdmin(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	owner := models.User{Name: "owner", Mail: "owner@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	for _, user := range []*models.User{&owner, &admin} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	problem := models.Problem{ID: 1000, Title: "Target", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	otherProblem := models.Problem{ID: 1001, Title: "Other", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&[]models.Problem{problem, otherProblem}).Error; err != nil {
		t.Fatalf("create problems: %v", err)
	}
	timeMS := 123
	memoryKB := 456
	leaseUntil := time.Now().Add(time.Minute)
	submissions := []models.Submission{
		{UserID: owner.ID, ProblemID: problem.ID, Language: "cpp", Code: "a", Status: "WA", Score: 20, Message: "wrong", Attempt: 3, JudgerID: &admin.ID, LeaseUntil: &leaseUntil, TimeMS: &timeMS, MemoryKB: &memoryKB},
		{UserID: owner.ID, ProblemID: problem.ID, Language: "cpp", Code: "b", Status: "AC", Score: 100, Message: "ok", Attempt: 1, TimeMS: &timeMS, MemoryKB: &memoryKB},
		{UserID: owner.ID, ProblemID: otherProblem.ID, Language: "cpp", Code: "c", Status: "WA", Score: 10, Message: "other", Attempt: 2, TimeMS: &timeMS, MemoryKB: &memoryKB},
	}
	if err := db.Create(&submissions).Error; err != nil {
		t.Fatalf("create submissions: %v", err)
	}
	for _, submission := range submissions {
		if err := db.Create(&models.Case{SubmissionID: submission.ID, No: 1, Status: submission.Status}).Error; err != nil {
			t.Fatalf("create case: %v", err)
		}
	}

	e := echo.New()
	Register(e, db)
	target := "/api/problems/1000/rejudge"
	if res := requestWithCookies(e, http.MethodPost, target, nil, nil); res.Code != http.StatusForbidden {
		t.Fatalf("guest should not rejudge problem, got %d body=%s", res.Code, res.Body.String())
	}
	adminRes := requestWithCookies(e, http.MethodPost, target, databaseSession(t, db, admin.ID), nil)
	if adminRes.Code != http.StatusOK {
		t.Fatalf("admin should rejudge problem, got %d body=%s", adminRes.Code, adminRes.Body.String())
	}
	if got := decodeJSON[contract.CountResult](t, adminRes); got.Count != 2 {
		t.Fatalf("rejudge count = %+v", got)
	}
	var got []models.Submission
	if err := db.Order("id").Find(&got).Error; err != nil {
		t.Fatalf("reload submissions: %v", err)
	}
	for _, row := range got[:2] {
		if row.Status != "queued" || row.Score != 0 || row.Message != "" || row.JudgerID != nil || row.LeaseUntil != nil || row.TimeMS != nil || row.MemoryKB != nil {
			t.Fatalf("rejudged submission = %+v", row)
		}
	}
	if got[0].Attempt != 4 || got[1].Attempt != 2 {
		t.Fatalf("attempts after rejudge = %d, %d", got[0].Attempt, got[1].Attempt)
	}
	if got[2].Status != "WA" || got[2].Attempt != 2 {
		t.Fatalf("other problem submission changed: %+v", got[2])
	}
	var cases int64
	if err := db.Model(&models.Case{}).Where("submission_id IN ?", []uint{submissions[0].ID, submissions[1].ID}).Count(&cases).Error; err != nil {
		t.Fatalf("count target cases: %v", err)
	}
	if cases != 0 {
		t.Fatalf("target cases after rejudge = %d, want 0", cases)
	}
}
