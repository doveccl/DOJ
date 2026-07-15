package worker

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/doveccl/doj/contract/judger"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/auth"
	"github.com/labstack/echo/v4"
)

func TestLeaseLongPollsWhenNoTask(t *testing.T) {
	db := newJudgerTestDB(t)
	api := &API{db: db, leaseWait: 30 * time.Millisecond}
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/judger/lease", strings.NewReader(`{"version":"`+judger.Version+`"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	started := time.Now()
	if err := api.auth(api.lease)(c); err != nil {
		t.Fatalf("lease: %v", err)
	}
	if elapsed := time.Since(started); elapsed < 20*time.Millisecond {
		t.Fatalf("lease returned too early after %s", elapsed)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"task":null}` {
		t.Fatalf("response = %s", got)
	}
}

func TestLeaseRejectsVersionMismatch(t *testing.T) {
	api := &API{db: newJudgerTestDB(t)}
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/judger/lease", strings.NewReader(`{"version":"old"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.RemoteAddr = "127.0.0.1:1234"
	err := api.auth(api.lease)(e.NewContext(req, httptest.NewRecorder()))
	expectHTTPStatus(t, err, http.StatusUpgradeRequired)
}

func TestTryLeaseStoresDatabaseLease(t *testing.T) {
	db := newJudgerTestDB(t)
	seedTaskData(t, db)
	remote := models.Judger{Name: "linux-a", Auth: auth.TokenHash("token-a")}
	if err := db.Create(&remote).Error; err != nil {
		t.Fatalf("create judger: %v", err)
	}
	submission := models.Submission{UserID: 1, ProblemID: 1000, Language: "cpp", Code: "int main(){}", Status: "queued"}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}

	api := &API{db: db}
	payload, err := api.tryLease(t.Context(), remote.ID)
	if err != nil {
		t.Fatalf("try lease: %v", err)
	}
	if payload == nil || payload.SubmissionID != submission.ID || payload.Attempt != 1 {
		t.Fatalf("payload = %+v", payload)
	}
	if payload.Problem.Hash == "" {
		t.Fatalf("lease payload should include problem package hash: %+v", payload.Problem)
	}
	var got models.Submission
	if err := db.First(&got, submission.ID).Error; err != nil {
		t.Fatalf("read submission: %v", err)
	}
	if got.Status != "judging" || got.Attempt != 1 || got.JudgerID == nil || *got.JudgerID != remote.ID || got.LeaseUntil == nil {
		t.Fatalf("submission lease not stored in db: %+v", got)
	}
}

func TestTryLeaseRequeuesWhenPayloadBuildFails(t *testing.T) {
	db := newJudgerTestDB(t)
	seedTaskData(t, db)
	judgerRow := models.Judger{Name: "linux-a", Auth: auth.TokenHash("token-a")}
	if err := db.Create(&judgerRow).Error; err != nil {
		t.Fatal(err)
	}
	submission := models.Submission{UserID: 1, ProblemID: 1000, Language: "cpp", Code: "int main(){}", Status: "queued"}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.Problem{}).Where("id = ?", 1000).Update("package", []byte(`{"broken"`)).Error; err != nil {
		t.Fatal(err)
	}
	if payload, err := (&API{db: db}).tryLease(t.Context(), judgerRow.ID); err == nil || payload != nil {
		t.Fatalf("payload = %+v, err = %v", payload, err)
	}
	var got models.Submission
	if err := db.First(&got, submission.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Status != "queued" || got.JudgerID != nil || got.LeaseUntil != nil {
		t.Fatalf("failed payload left active lease: %+v", got)
	}
}

func TestHeartbeatExtendsDatabaseLease(t *testing.T) {
	db := newJudgerTestDB(t)
	remote := models.Judger{Name: "linux-a", Auth: auth.TokenHash("token-a")}
	if err := db.Create(&remote).Error; err != nil {
		t.Fatalf("create judger: %v", err)
	}
	oldLease := time.Now().Add(5 * time.Second)
	submission := models.Submission{UserID: 1, ProblemID: 1000, Language: "cpp", Code: "int main(){}", Status: "judging", Attempt: 3, JudgerID: &remote.ID, LeaseUntil: &oldLease}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	e := echo.New()
	Register(e, db)
	target := "/api/judger/tasks/" + strconv.FormatUint(uint64(submission.ID), 10) + "/heartbeat"
	body := `{"submissionId":` + strconv.FormatUint(uint64(submission.ID), 10) + `,"attempt":3}`
	res := judgerJSON(e, target, "token-a", body)
	if res.Code != http.StatusAccepted {
		t.Fatalf("heartbeat got %d body=%s", res.Code, res.Body.String())
	}
	var got models.Submission
	if err := db.First(&got, submission.ID).Error; err != nil {
		t.Fatalf("read submission: %v", err)
	}
	if got.LeaseUntil == nil || !got.LeaseUntil.After(oldLease) {
		t.Fatalf("lease not extended: old=%s got=%v", oldLease, got.LeaseUntil)
	}
	if got.JudgerID == nil || *got.JudgerID != remote.ID || got.Attempt != 3 {
		t.Fatalf("heartbeat changed owner/attempt: %+v", got)
	}
}
