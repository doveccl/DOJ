package worker

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/doveccl/doj/contract/judger"
	"github.com/doveccl/doj/contract/limits"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/auth"
	"github.com/doveccl/doj/server/judge"
	"github.com/labstack/echo/v4"
)

func TestValidateResult(t *testing.T) {
	valid := judger.ResultRequest{
		SubmissionID: 1,
		Attempt:      1,
		Status:       "AC",
		Score:        100,
		Cases:        []judger.CaseResult{{No: 1, Status: "AC", Score: 100}},
	}
	if err := validateResult(valid); err != nil {
		t.Fatalf("valid result rejected: %v", err)
	}

	tests := []struct {
		name string
		req  judger.ResultRequest
		want int
	}{
		{name: "missing target", req: judger.ResultRequest{Status: "AC", Score: 100}, want: http.StatusBadRequest},
		{name: "missing attempt", req: judger.ResultRequest{SubmissionID: 1, Status: "AC", Score: 100}, want: http.StatusBadRequest},
		{name: "bad status", req: judger.ResultRequest{SubmissionID: 1, Attempt: 1, Status: "queued", Score: 0}, want: http.StatusBadRequest},
		{name: "negative score", req: judger.ResultRequest{SubmissionID: 1, Attempt: 1, Status: "WA", Score: -1}, want: http.StatusBadRequest},
		{name: "large score", req: judger.ResultRequest{SubmissionID: 1, Attempt: 1, Status: "WA", Score: 101}, want: http.StatusBadRequest},
		{name: "large message", req: judger.ResultRequest{SubmissionID: 1, Attempt: 1, Status: "WA", Score: 0, Message: strings.Repeat("x", limits.MaxJudgerMessageBytes+1)}, want: http.StatusRequestEntityTooLarge},
		{name: "bad case no", req: judger.ResultRequest{SubmissionID: 1, Attempt: 1, Status: "WA", Score: 0, Cases: []judger.CaseResult{{No: 0, Status: "WA"}}}, want: http.StatusBadRequest},
		{name: "bad case status", req: judger.ResultRequest{SubmissionID: 1, Attempt: 1, Status: "WA", Score: 0, Cases: []judger.CaseResult{{No: 1, Status: "queued"}}}, want: http.StatusBadRequest},
		{name: "bad case score", req: judger.ResultRequest{SubmissionID: 1, Attempt: 1, Status: "WA", Score: 0, Cases: []judger.CaseResult{{No: 1, Status: "WA", Score: 101}}}, want: http.StatusBadRequest},
		{name: "duplicate case", req: judger.ResultRequest{SubmissionID: 1, Attempt: 1, Status: "WA", Score: 0, Cases: []judger.CaseResult{{No: 1, Status: "WA"}, {No: 1, Status: "WA"}}}, want: http.StatusBadRequest},
		{name: "negative usage", req: func() judger.ResultRequest {
			value := -1
			return judger.ResultRequest{SubmissionID: 1, Attempt: 1, Status: "WA", TimeMS: &value}
		}(), want: http.StatusBadRequest},
		{name: "large case message", req: judger.ResultRequest{SubmissionID: 1, Attempt: 1, Status: "WA", Score: 0, Cases: []judger.CaseResult{{No: 1, Status: "WA", Message: strings.Repeat("x", models.CaseMessageMax+1)}}}, want: http.StatusRequestEntityTooLarge},
	}
	for _, item := range tests {
		t.Run(item.name, func(t *testing.T) {
			err := validateResult(item.req)
			if err == nil {
				t.Fatal("expected validation error")
			}
			expectHTTPStatus(t, err, item.want)
		})
	}
}

func TestResultIgnoresStaleAttempt(t *testing.T) {
	db := newJudgerTestDB(t)
	remote := models.Judger{Name: "linux-a", Auth: auth.TokenHash("token-a")}
	if err := db.Create(&remote).Error; err != nil {
		t.Fatalf("create judger: %v", err)
	}
	until := time.Now().Add(time.Minute)
	submission := models.Submission{UserID: 1, ProblemID: 1000, Language: "cpp", Code: "int main(){}", Status: "judging", Attempt: 2, JudgerID: &remote.ID, LeaseUntil: &until}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	e := echo.New()
	Register(e, db)
	target := "/api/judger/tasks/" + strconv.FormatUint(uint64(submission.ID), 10) + "/result"

	stale := `{"submissionId":` + strconv.FormatUint(uint64(submission.ID), 10) + `,"attempt":1,"status":"AC","score":100,"message":"old","cases":[]}`
	res := judgerJSON(e, target, "token-a", stale)
	if res.Code != http.StatusAccepted {
		t.Fatalf("stale result got %d body=%s", res.Code, res.Body.String())
	}
	var got models.Submission
	if err := db.First(&got, submission.ID).Error; err != nil {
		t.Fatalf("read submission: %v", err)
	}
	if got.Status != "judging" || got.Score != 0 {
		t.Fatalf("stale attempt changed submission: %+v", got)
	}

	current := `{"submissionId":` + strconv.FormatUint(uint64(submission.ID), 10) + `,"attempt":2,"status":"AC","score":100,"message":"ok","cases":[{"no":1,"status":"AC","score":100}]}`
	res = judgerJSON(e, target, "token-a", current)
	if res.Code != http.StatusAccepted {
		t.Fatalf("current result got %d body=%s", res.Code, res.Body.String())
	}
	if err := db.First(&got, submission.ID).Error; err != nil {
		t.Fatalf("read updated submission: %v", err)
	}
	if got.Status != "AC" || got.Score != 100 || got.Message != "ok" {
		t.Fatalf("current attempt did not update submission: %+v", got)
	}
	var cases int64
	if err := db.Model(&models.Case{}).Where("submission_id = ?", submission.ID).Count(&cases).Error; err != nil {
		t.Fatalf("count cases: %v", err)
	}
	if cases != 1 {
		t.Fatalf("case results = %d, want 1", cases)
	}
}

func TestHeartbeatStoresProgressForCurrentAttempt(t *testing.T) {
	db := newJudgerTestDB(t)
	remote := models.Judger{Name: "linux-a", Auth: auth.TokenHash("token-a")}
	if err := db.Create(&remote).Error; err != nil {
		t.Fatalf("create judger: %v", err)
	}
	until := time.Now().Add(time.Minute)
	submission := models.Submission{UserID: 1, ProblemID: 1000, Language: "cpp", Code: "int main(){}", Status: "judging", Attempt: 3, JudgerID: &remote.ID, LeaseUntil: &until}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	e := echo.New()
	Register(e, db)
	target := "/api/judger/tasks/" + strconv.FormatUint(uint64(submission.ID), 10) + "/heartbeat"
	body := `{"submissionId":` + strconv.FormatUint(uint64(submission.ID), 10) + `,"attempt":3,"stage":"download","done":4096}`
	res := judgerJSON(e, target, "token-a", body)
	if res.Code != http.StatusAccepted {
		t.Fatalf("heartbeat got %d body=%s", res.Code, res.Body.String())
	}
	progress, err := judge.ReadProgress(t.Context(), submission.ID, 3)
	if err != nil {
		t.Fatalf("read progress: %v", err)
	}
	if progress == nil || progress.Stage != "download" || progress.Done != 4096 || progress.Total != nil {
		t.Fatalf("progress = %+v", progress)
	}

	stale := `{"submissionId":` + strconv.FormatUint(uint64(submission.ID), 10) + `,"attempt":2,"stage":"judge","done":1,"total":2}`
	res = judgerJSON(e, target, "token-a", stale)
	if res.Code != http.StatusAccepted {
		t.Fatalf("stale heartbeat got %d body=%s", res.Code, res.Body.String())
	}
	progress, err = judge.ReadProgress(t.Context(), submission.ID, 3)
	if err != nil {
		t.Fatalf("read progress after stale: %v", err)
	}
	if progress == nil || progress.Stage != "download" || progress.Done != 4096 {
		t.Fatalf("stale heartbeat changed progress: %+v", progress)
	}
}

func TestResultClearsProgress(t *testing.T) {
	db := newJudgerTestDB(t)
	remote := models.Judger{Name: "linux-a", Auth: auth.TokenHash("token-a")}
	if err := db.Create(&remote).Error; err != nil {
		t.Fatalf("create judger: %v", err)
	}
	until := time.Now().Add(time.Minute)
	submission := models.Submission{UserID: 1, ProblemID: 1000, Language: "cpp", Code: "int main(){}", Status: "judging", Attempt: 2, JudgerID: &remote.ID, LeaseUntil: &until}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	if err := judge.SaveProgress(t.Context(), submission.ID, judge.Progress{Attempt: 2, Stage: "judge", Done: 1, UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("save progress: %v", err)
	}
	e := echo.New()
	Register(e, db)
	target := "/api/judger/tasks/" + strconv.FormatUint(uint64(submission.ID), 10) + "/result"
	body := `{"submissionId":` + strconv.FormatUint(uint64(submission.ID), 10) + `,"attempt":2,"status":"AC","score":100,"message":"","cases":[]}`
	res := judgerJSON(e, target, "token-a", body)
	if res.Code != http.StatusAccepted {
		t.Fatalf("result got %d body=%s", res.Code, res.Body.String())
	}
	progress, err := judge.ReadProgress(t.Context(), submission.ID, 2)
	if err != nil {
		t.Fatalf("read progress: %v", err)
	}
	if progress != nil {
		t.Fatalf("progress after result = %+v, want nil", progress)
	}
}
