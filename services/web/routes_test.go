package web

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/doveccl/doj/models"
	adminsvc "github.com/doveccl/doj/services/admin"
	judgersvc "github.com/doveccl/doj/services/judger"
	"github.com/doveccl/doj/utils"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestProblemDiscussionCountsUseTagsWithoutProblemVisibilityCoupling(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	visible := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	hidden := models.Problem{ID: 1001, Title: "Hidden", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&visible).Error; err != nil {
		t.Fatalf("create visible problem: %v", err)
	}
	if err := db.Create(&hidden).Error; err != nil {
		t.Fatalf("create hidden problem: %v", err)
	}
	discussions := []models.Discussion{
		{Title: "Visible only", Content: "body", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["P1000"]`))},
		{Title: "Mixed hidden", Content: "body", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["P1000","P1001"]`))},
	}
	if err := db.Create(&discussions).Error; err != nil {
		t.Fatalf("create discussions: %v", err)
	}
	e := echo.New()
	Register(e, db)

	guestProblem := decodeJSON[ProblemDTO](t, requestOK(t, e, http.MethodGet, "/api/problems/1000", ""))
	if guestProblem.Discussions != 2 {
		t.Fatalf("guest discussion count should use soft tags regardless of other problem tags: %+v", guestProblem)
	}
	adminProblem := decodeJSON[ProblemDTO](t, requestWithCookies(e, http.MethodGet, "/api/problems/1000", databaseSession(t, db, admin.ID), nil))
	if adminProblem.Discussions != 2 {
		t.Fatalf("admin discussion count should match soft tag count: %+v", adminProblem)
	}
}

func TestWriteSSE(t *testing.T) {
	var out bytes.Buffer
	if err := writeSSE(&out, "submission", []byte("{\"changed\":\"submission\"}")); err != nil {
		t.Fatalf("write sse: %v", err)
	}
	want := "event: submission\ndata: {\"changed\":\"submission\"}\n\n"
	if out.String() != want {
		t.Fatalf("sse = %q, want %q", out.String(), want)
	}
}

func TestUsernamePreservesCaseAndMatchesCaseInsensitively(t *testing.T) {
	db := testWebDB(t)
	if err := adminsvc.SaveSettings(db, adminsvc.AdminSettings{
		SiteName:                "DOJ",
		AllowRegistration:       true,
		AllowGuestAccess:        true,
		DefaultSubmissionPublic: false,
		Notice:                  "",
	}); err != nil {
		t.Fatalf("enable registration: %v", err)
	}
	e := echo.New()
	Register(e, db)

	registerRes := requestJSON(e, http.MethodPost, "/api/auth/register", "", `{"name":"Alice_One","mail":"Alice@example.com","password":"password123"}`)
	if registerRes.Code != http.StatusCreated {
		t.Fatalf("register mixed case user got %d body=%s", registerRes.Code, registerRes.Body.String())
	}
	registered := decodeJSON[MeDTO](t, registerRes)
	if registered.Name != "Alice_One" || registered.Mail != "alice@example.com" {
		t.Fatalf("registered user should preserve username case and lowercase mail: %+v", registered)
	}

	duplicate := requestJSON(e, http.MethodPost, "/api/auth/register", "", `{"name":"alice_one","mail":"other@example.com","password":"password123"}`)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("case folded duplicate register got %d body=%s", duplicate.Code, duplicate.Body.String())
	}

	loginRes := requestJSON(e, http.MethodPost, "/api/auth/login", "", `{"name":"alice_one","password":"password123"}`)
	if loginRes.Code != http.StatusOK {
		t.Fatalf("case folded login got %d body=%s", loginRes.Code, loginRes.Body.String())
	}
	loggedIn := decodeJSON[MeDTO](t, loginRes)
	if loggedIn.Name != "Alice_One" {
		t.Fatalf("login should return stored username case: %+v", loggedIn)
	}

	profile := decodeJSON[UserProfile](t, requestOK(t, e, http.MethodGet, "/api/users/alice_one", ""))
	if profile.User.Name != "Alice_One" {
		t.Fatalf("profile lookup should be case-insensitive and return stored name: %+v", profile.User)
	}

	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	var user models.User
	if err := db.Where("LOWER(name) = ?", "alice_one").First(&user).Error; err != nil {
		t.Fatalf("read user: %v", err)
	}
	if err := db.Create(&models.Submission{UserID: user.ID, ProblemID: problem.ID, Language: "cpp", Code: "code", Status: "AC", Score: 100, Public: true}).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	submissionRes := requestOK(t, e, http.MethodGet, "/api/submissions?user=ALICE_ONE", "")
	submissions := decodePageItems[SubmissionListItem](t, submissionRes)
	if len(submissions) != 1 || submissions[0].User != "Alice_One" {
		t.Fatalf("submission user filter should be case-insensitive: %+v", submissions)
	}
	rawSubmissions := decodeJSON[PageResult[map[string]any]](t, submissionRes).Items
	for _, key := range []string{"score", "message", "public"} {
		if _, ok := rawSubmissions[0][key]; ok {
			t.Fatalf("submission list should not include detail field %q: %+v", key, rawSubmissions[0])
		}
	}
}

func TestMePatchOnlyUpdatesProvidedFields(t *testing.T) {
	db := testWebDB(t)
	user := models.User{Name: "student", Mail: "student@example.com", Bio: "old bio", Avatar: "/old.png", Auth: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, user.ID)

	avatarRes := requestJSONWithCookies(e, http.MethodPatch, "/api/me", cookies, `{"avatar":"/new.png"}`)
	if avatarRes.Code != http.StatusOK {
		t.Fatalf("avatar patch got %d body=%s", avatarRes.Code, avatarRes.Body.String())
	}
	avatar := decodeJSON[MeDTO](t, avatarRes)
	if avatar.Mail != "student@example.com" || avatar.Bio != "old bio" || avatar.Avatar != "/new.png" {
		t.Fatalf("avatar patch should preserve mail and bio: %+v", avatar)
	}

	mailRes := requestJSONWithCookies(e, http.MethodPatch, "/api/me", cookies, `{"mail":"Next@Example.com"}`)
	if mailRes.Code != http.StatusOK {
		t.Fatalf("mail patch got %d body=%s", mailRes.Code, mailRes.Body.String())
	}
	mail := decodeJSON[MeDTO](t, mailRes)
	if mail.Mail != "next@example.com" || mail.Bio != "old bio" || mail.Avatar != "/new.png" {
		t.Fatalf("mail patch should preserve bio and avatar: %+v", mail)
	}
	clearRes := requestJSONWithCookies(e, http.MethodPatch, "/api/me", cookies, `{"bio":"","avatar":""}`)
	if clearRes.Code != http.StatusOK {
		t.Fatalf("clear profile fields got %d body=%s", clearRes.Code, clearRes.Body.String())
	}
	cleared := decodeJSON[MeDTO](t, clearRes)
	if cleared.Mail != "next@example.com" || cleared.Bio != "" || cleared.Avatar != "" {
		t.Fatalf("empty string patch should clear provided profile fields only: %+v", cleared)
	}
	emptyRes := requestJSONWithCookies(e, http.MethodPatch, "/api/me", cookies, `{}`)
	if emptyRes.Code != http.StatusBadRequest {
		t.Fatalf("empty profile patch got %d body=%s", emptyRes.Code, emptyRes.Body.String())
	}
}

func TestPrivateSubmissionSourceVisibilityWithDatabase(t *testing.T) {
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
	problem := models.Problem{
		ID:       1000,
		Title:    "Visible",
		Tags:     datatypes.JSON([]byte(`[]`)),
		Visible:  true,
		Mode:     "default",
		TimeMS:   1000,
		MemoryMB: 256,
	}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	private := models.Submission{UserID: owner.ID, ProblemID: problem.ID, Language: "cpp", Code: "secret", Status: "AC", Score: 100, Public: false}
	public := models.Submission{UserID: owner.ID, ProblemID: problem.ID, Language: "cpp", Code: "visible", Status: "AC", Score: 100, Public: true}
	if err := db.Create(&private).Error; err != nil {
		t.Fatalf("create private submission: %v", err)
	}
	if err := db.Create(&public).Error; err != nil {
		t.Fatalf("create public submission: %v", err)
	}

	e := echo.New()
	Register(e, db)

	privateTarget := "/api/submissions/" + strconv.FormatUint(uint64(private.ID), 10)
	publicTarget := "/api/submissions/" + strconv.FormatUint(uint64(public.ID), 10)
	if res := requestWithCookies(e, http.MethodGet, publicTarget, nil, nil); res.Code != http.StatusOK {
		t.Fatalf("guest should read public DB submission, got %d body=%s", res.Code, res.Body.String())
	}
	otherDetail := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, privateTarget, databaseSession(t, db, other.ID), nil))
	if otherDetail.Code != "" || otherDetail.Submission.ID != private.ID {
		t.Fatalf("other user should read private submission detail without source: %+v", otherDetail)
	}
	if got := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, privateTarget, databaseSession(t, db, owner.ID), nil)); got.Code != "secret" {
		t.Fatalf("owner should read private DB submission source: %+v", got)
	}
	if got := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, privateTarget, databaseSession(t, db, admin.ID), nil)); got.Code != "secret" {
		t.Fatalf("admin should read private DB submission source: %+v", got)
	}
}

func TestSubmissionDetailIncludesTopLevelMessage(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	owner := models.User{Name: "owner", Mail: "owner@example.com", Auth: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	submission := models.Submission{
		UserID:    owner.ID,
		ProblemID: problem.ID,
		Language:  "cpp",
		Code:      "int main(){",
		Status:    "CE",
		Message:   "compile failed\nmain.cpp:1: error: expected '}'",
		Public:    true,
	}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}

	e := echo.New()
	Register(e, db)

	target := "/api/submissions/" + strconv.FormatUint(uint64(submission.ID), 10)
	got := decodeJSON[SubmissionDetail](t, requestOK(t, e, http.MethodGet, target, ""))
	if got.Submission.Message != submission.Message {
		t.Fatalf("submission detail should expose top-level judge message: %+v", got.Submission)
	}
}

func TestSubmissionDetailIncludesProgressButListDoesNot(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	owner := models.User{Name: "owner", Mail: "owner@example.com", Auth: "hash"}
	if err := db.Create(&owner).Error; err != nil {
		t.Fatalf("create owner: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	total := int64(12)
	submission := models.Submission{
		UserID:    owner.ID,
		ProblemID: problem.ID,
		Language:  "cpp",
		Code:      "int main(){}",
		Status:    "judging",
		Attempt:   4,
		Public:    true,
	}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	if err := judgersvc.SaveProgress(t.Context(), submission.ID, judgersvc.Progress{Attempt: 4, Stage: "judge", Done: 3, Total: &total, UpdatedAt: time.Now()}); err != nil {
		t.Fatalf("save progress: %v", err)
	}

	e := echo.New()
	Register(e, db)

	target := "/api/submissions/" + strconv.FormatUint(uint64(submission.ID), 10)
	got := decodeJSON[SubmissionDetail](t, requestOK(t, e, http.MethodGet, target, ""))
	if got.Progress == nil || got.Progress.Stage != "judge" || got.Progress.Done != 3 || got.Progress.Total == nil || *got.Progress.Total != total {
		t.Fatalf("submission detail progress = %+v", got.Progress)
	}

	list := requestOK(t, e, http.MethodGet, "/api/submissions", "")
	if strings.Contains(list.Body.String(), "progress") {
		t.Fatalf("submission list should not expose progress: %s", list.Body.String())
	}
}

func TestSubmissionPublicCanBeUpdatedByOwnerOrAdmin(t *testing.T) {
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
	problem := models.Problem{
		ID:       1000,
		Title:    "Visible",
		Tags:     datatypes.JSON([]byte(`[]`)),
		Visible:  true,
		Mode:     "default",
		TimeMS:   1000,
		MemoryMB: 256,
	}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	submission := models.Submission{UserID: owner.ID, ProblemID: problem.ID, Language: "cpp", Code: "code", Status: "AC", Score: 100, Public: false}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}

	e := echo.New()
	Register(e, db)
	target := "/api/submissions/" + strconv.FormatUint(uint64(submission.ID), 10)
	if res := requestJSONWithCookies(e, http.MethodPatch, target, nil, `{"public":true}`); res.Code != http.StatusUnauthorized {
		t.Fatalf("guest should not update submission public flag, got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestJSONWithCookies(e, http.MethodPatch, target, databaseSession(t, db, other.ID), `{"public":true}`); res.Code != http.StatusForbidden {
		t.Fatalf("other user should not update submission public flag, got %d body=%s", res.Code, res.Body.String())
	}
	ownerRes := requestJSONWithCookies(e, http.MethodPatch, target, databaseSession(t, db, owner.ID), `{"public":true}`)
	if ownerRes.Code != http.StatusOK {
		t.Fatalf("owner should update submission public flag, got %d body=%s", ownerRes.Code, ownerRes.Body.String())
	}
	if got := decodeJSON[CreatedID](t, ownerRes); got.ID != submission.ID {
		t.Fatalf("owner update should return submission id: %+v", got)
	}
	var got models.Submission
	if err := db.First(&got, submission.ID).Error; err != nil {
		t.Fatalf("reload submission: %v", err)
	}
	if !got.Public {
		t.Fatalf("owner update should persist public=true")
	}
	adminRes := requestJSONWithCookies(e, http.MethodPatch, target, databaseSession(t, db, admin.ID), `{"public":false}`)
	if adminRes.Code != http.StatusOK {
		t.Fatalf("admin should update submission public flag, got %d body=%s", adminRes.Code, adminRes.Body.String())
	}
	if got := decodeJSON[CreatedID](t, adminRes); got.ID != submission.ID {
		t.Fatalf("admin update should return submission id: %+v", got)
	}
	if err := db.First(&got, submission.ID).Error; err != nil {
		t.Fatalf("reload submission: %v", err)
	}
	if got.Public {
		t.Fatalf("admin update should persist public=false")
	}
}

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
	gotID := decodeJSON[CreatedID](t, adminRes)
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
	if got := decodeJSON[CountResult](t, adminRes); got.Count != 2 {
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

func TestContextSubmissionSourceLockedUntilContextEnds(t *testing.T) {
	db := testWebDB(t)
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
	now := time.Now()
	liveContest := models.Contest{Title: "Live Contest", Kind: "ICPC", StartAt: now.Add(-time.Hour), EndAt: now.Add(time.Hour)}
	endedContest := models.Contest{Title: "Ended Contest", Kind: "ICPC", StartAt: now.Add(-2 * time.Hour), EndAt: now.Add(-time.Hour)}
	liveAssignment := models.Assignment{Title: "Live Assignment", EndAt: now.Add(time.Hour)}
	endedAssignment := models.Assignment{Title: "Ended Assignment", EndAt: now.Add(-time.Hour)}
	for _, row := range []any{&liveContest, &endedContest, &liveAssignment, &endedAssignment} {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create context: %v", err)
		}
	}
	liveContestID := liveContest.ID
	endedContestID := endedContest.ID
	liveAssignmentID := liveAssignment.ID
	endedAssignmentID := endedAssignment.ID
	submissions := []models.Submission{
		{UserID: owner.ID, ProblemID: problem.ID, ContestID: &liveContestID, Language: "cpp", Code: "live contest", Status: "AC", Score: 100, Public: true},
		{UserID: owner.ID, ProblemID: problem.ID, ContestID: &endedContestID, Language: "cpp", Code: "ended contest", Status: "AC", Score: 100, Public: true},
		{UserID: owner.ID, ProblemID: problem.ID, AssignmentID: &liveAssignmentID, Language: "cpp", Code: "live assignment", Status: "AC", Score: 100, Public: true},
		{UserID: owner.ID, ProblemID: problem.ID, AssignmentID: &endedAssignmentID, Language: "cpp", Code: "ended assignment", Status: "AC", Score: 100, Public: true},
		{UserID: owner.ID, ProblemID: problem.ID, Language: "cpp", Code: "private", Status: "AC", Score: 100, Public: false},
	}
	for index := range submissions {
		if err := db.Create(&submissions[index]).Error; err != nil {
			t.Fatalf("create submission: %v", err)
		}
	}

	e := echo.New()
	Register(e, db)
	ownerCookies := databaseSession(t, db, owner.ID)
	otherCookies := databaseSession(t, db, other.ID)
	adminCookies := databaseSession(t, db, admin.ID)
	target := func(row models.Submission) string {
		return "/api/submissions/" + strconv.FormatUint(uint64(row.ID), 10)
	}

	if got := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target(submissions[0]), otherCookies, nil)); got.Code != "" || got.Submission.Status != "AC" {
		t.Fatalf("other user should read live contest detail without source: %+v", got)
	}
	if res := requestWithCookies(e, http.MethodGet, target(submissions[0]), ownerCookies, nil); res.Code != http.StatusOK {
		t.Fatalf("owner should read live contest source, got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, target(submissions[0]), adminCookies, nil); res.Code != http.StatusOK {
		t.Fatalf("admin should read live contest source, got %d body=%s", res.Code, res.Body.String())
	}
	if got := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target(submissions[1]), otherCookies, nil)); got.Code != "ended contest" {
		t.Fatalf("other user should read ended contest public source: %+v", got)
	}
	if got := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target(submissions[2]), otherCookies, nil)); got.Code != "" || got.Submission.Status != "AC" {
		t.Fatalf("other user should read live assignment detail without source: %+v", got)
	}
	if res := requestWithCookies(e, http.MethodGet, target(submissions[2]), ownerCookies, nil); res.Code != http.StatusOK {
		t.Fatalf("owner should read live assignment source, got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, target(submissions[2]), adminCookies, nil); res.Code != http.StatusOK {
		t.Fatalf("admin should read live assignment source, got %d body=%s", res.Code, res.Body.String())
	}
	if got := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target(submissions[3]), otherCookies, nil)); got.Code != "ended assignment" {
		t.Fatalf("other user should read ended assignment public source: %+v", got)
	}
	if got := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target(submissions[4]), otherCookies, nil)); got.Code != "" {
		t.Fatalf("other user should read private detail without source: %+v", got)
	}
}

func TestHiddenProblemReferencesDoNotLeakFromDatabaseProfilesAndContexts(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	student := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	outsider := models.User{Name: "outsider", Mail: "outsider@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&student).Error; err != nil {
		t.Fatalf("create student: %v", err)
	}
	if err := db.Create(&outsider).Error; err != nil {
		t.Fatalf("create outsider: %v", err)
	}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	visible := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	hidden := models.Problem{ID: 1001, Title: "Hidden", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&visible).Error; err != nil {
		t.Fatalf("create visible problem: %v", err)
	}
	if err := db.Create(&hidden).Error; err != nil {
		t.Fatalf("create hidden problem: %v", err)
	}

	assignment := models.Assignment{Title: "HW", EndAt: time.Now().Add(time.Hour)}
	contest := models.Contest{Title: "Round", Kind: "OI", StartAt: time.Now().Add(-time.Hour), EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	links := []any{
		&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: visible.ID, Sort: "A"},
		&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: hidden.ID, Sort: "B"},
		&models.ContestProblem{ContestID: contest.ID, ProblemID: visible.ID, Sort: "A"},
		&models.ContestProblem{ContestID: contest.ID, ProblemID: hidden.ID, Sort: "B"},
	}
	for _, link := range links {
		if err := db.Create(link).Error; err != nil {
			t.Fatalf("create link: %v", err)
		}
	}
	if err := db.Create(&models.AssignmentUser{AssignmentID: assignment.ID, UserID: student.ID}).Error; err != nil {
		t.Fatalf("assign student: %v", err)
	}

	assignmentID := assignment.ID
	contestID := contest.ID
	submissions := []models.Submission{
		{UserID: student.ID, ProblemID: visible.ID, Language: "cpp", Code: "visible", Status: "AC", Score: 100, Public: true},
		{UserID: student.ID, ProblemID: hidden.ID, Language: "cpp", Code: "hidden", Status: "AC", Score: 100, Public: true},
		{UserID: student.ID, ProblemID: visible.ID, Language: "cpp", Code: "visible hw", AssignmentID: &assignmentID, Status: "AC", Score: 100, Public: true},
		{UserID: student.ID, ProblemID: hidden.ID, Language: "cpp", Code: "hidden hw", AssignmentID: &assignmentID, Status: "AC", Score: 100, Public: true},
		{UserID: student.ID, ProblemID: visible.ID, Language: "cpp", Code: "visible round", ContestID: &contestID, Status: "AC", Score: 100, Public: true},
		{UserID: student.ID, ProblemID: hidden.ID, Language: "cpp", Code: "hidden round", ContestID: &contestID, Status: "AC", Score: 100, Public: true},
	}
	for index := range submissions {
		if err := db.Create(&submissions[index]).Error; err != nil {
			t.Fatalf("create submission: %v", err)
		}
	}
	if err := db.Create(&models.Discussion{Title: "Student note", Content: "body", UserID: student.ID, Tags: datatypes.JSON([]byte(`["P1000"]`))}).Error; err != nil {
		t.Fatalf("create discussion: %v", err)
	}

	e := echo.New()
	Register(e, db)

	profileRes := requestOK(t, e, http.MethodGet, "/api/users/student", "")
	profile := decodeJSON[UserProfile](t, profileRes)
	if hasSolvedProblem(profile.Solved.Items, hidden.ID) || hasActivityProblem(profile.Activities, hidden.ID) {
		t.Fatalf("guest profile leaked hidden problem: %+v", profile)
	}
	if !hasActivity(profile.Activities, "discussion", "Student note") {
		t.Fatalf("guest profile should include discussion activity: %+v", profile.Activities)
	}
	if profile.User.AC != 2 || profile.User.Submit != 6 {
		t.Fatalf("guest profile stats should include all activity: %+v", profile.User)
	}
	today := time.Now().Format("2006-01-02")
	if got := countForDate(profile.Heatmap, today); got != 6 {
		t.Fatalf("guest profile heatmap should include all submissions, got %d", got)
	}
	var rawProfile struct {
		Solved struct {
			Items []map[string]any `json:"items"`
		} `json:"solved"`
	}
	if err := json.Unmarshal(profileRes.Body.Bytes(), &rawProfile); err != nil {
		t.Fatalf("decode raw profile: %v", err)
	}
	if len(rawProfile.Solved.Items) > 0 {
		for _, key := range []string{"visible", "mode", "timeMs", "memoryMb", "discussions", "mine", "latest"} {
			if _, ok := rawProfile.Solved.Items[0][key]; ok {
				t.Fatalf("profile solved problem should not include list-only field %q: %+v", key, rawProfile.Solved.Items[0])
			}
		}
	}

	if res := requestOK(t, e, http.MethodGet, "/api/assignments", ""); len(decodeJSON[PageResult[AssignmentDTO]](t, res).Items) != 0 {
		t.Fatalf("guest assignment list should be empty, body=%s", res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(assignment.ID), 10), nil, nil); res.Code != http.StatusNotFound {
		t.Fatalf("guest assignment detail should be hidden, got %d body=%s", res.Code, res.Body.String())
	}
	outsiderCookies := databaseSession(t, db, outsider.ID)
	if res := requestWithCookies(e, http.MethodGet, "/api/assignments", outsiderCookies, nil); len(decodeJSON[PageResult[AssignmentDTO]](t, res).Items) != 0 {
		t.Fatalf("unassigned assignment list should be empty, body=%s", res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(assignment.ID), 10), outsiderCookies, nil); res.Code != http.StatusNotFound {
		t.Fatalf("unassigned assignment detail should be hidden, got %d body=%s", res.Code, res.Body.String())
	}
	studentAssignment := decodeJSON[AssignmentDetail](t, requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(assignment.ID), 10), databaseSession(t, db, student.ID), nil))
	if !hasProblem(studentAssignment.Problems, hidden.ID) {
		t.Fatalf("student assignment should include assigned hidden problem: %+v", studentAssignment)
	}
	if len(studentAssignment.Problems) != 2 || studentAssignment.Problems[0].Sort != "A" || studentAssignment.Problems[1].Sort != "B" {
		t.Fatalf("student assignment should expose collection problem order: %+v", studentAssignment.Problems)
	}
	if studentAssignment.Assignment.Done != 2 || studentAssignment.Assignment.Total != 2 {
		t.Fatalf("student assignment progress should include hidden problems in aggregate stats: %+v", studentAssignment.Assignment)
	}
	if len(studentAssignment.Progress) != 1 || studentAssignment.Progress[0].User != "student" {
		t.Fatalf("student assignment completion should include assigned student only: %+v", studentAssignment.Progress)
	}
	if len(studentAssignment.Progress[0].Problems) != 2 || studentAssignment.Progress[0].Problems[0].ProblemID != visible.ID || studentAssignment.Progress[0].Problems[1].ProblemID != hidden.ID {
		t.Fatalf("student assignment completion should expose assigned problem statuses: %+v", studentAssignment.Progress[0].Problems)
	}

	contestDetail := decodeJSON[ContestDetail](t, requestOK(t, e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(contest.ID), 10), ""))
	if !hasProblem(contestDetail.Problems, hidden.ID) {
		t.Fatalf("running contest should include linked hidden problem: %+v", contestDetail)
	}
	if len(contestDetail.Problems) != 2 || contestDetail.Problems[0].Sort != "A" || contestDetail.Problems[1].Sort != "B" {
		t.Fatalf("guest contest should expose collection problem order: %+v", contestDetail.Problems)
	}

	adminProfileRes := requestWithCookies(e, http.MethodGet, "/api/users/student", databaseSession(t, db, admin.ID), nil)
	if adminProfileRes.Code != http.StatusOK {
		t.Fatalf("admin profile got %d body=%s", adminProfileRes.Code, adminProfileRes.Body.String())
	}
	adminProfile := decodeJSON[UserProfile](t, adminProfileRes)
	if !hasSolvedProblem(adminProfile.Solved.Items, hidden.ID) || !hasActivityProblem(adminProfile.Activities, hidden.ID) {
		t.Fatalf("admin profile should include hidden problem: %+v", adminProfile)
	}
	if adminProfile.User.AC != 2 || adminProfile.User.Submit != 6 {
		t.Fatalf("admin profile stats should include hidden activity: %+v", adminProfile.User)
	}
	if got := countForDate(adminProfile.Heatmap, today); got != 6 {
		t.Fatalf("admin profile heatmap should include hidden submissions, got %d", got)
	}
}

func TestUserSolvedProblemsArePagedByLatestAC(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	user := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	for index := 0; index < 15; index++ {
		problem := models.Problem{ID: uint(1000 + index), Title: "Problem " + strconv.Itoa(index), Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
		if err := db.Create(&problem).Error; err != nil {
			t.Fatalf("create problem %d: %v", problem.ID, err)
		}
		submission := models.Submission{
			UserID:    user.ID,
			ProblemID: problem.ID,
			Language:  "cpp",
			Code:      "int main(){}",
			Status:    "AC",
			Score:     100,
			Public:    true,
			CreatedAt: base.Add(time.Duration(index) * time.Minute),
		}
		if err := db.Create(&submission).Error; err != nil {
			t.Fatalf("create submission %d: %v", problem.ID, err)
		}
	}

	e := echo.New()
	Register(e, db)
	profile := decodeJSON[UserProfile](t, requestOK(t, e, http.MethodGet, "/api/users/alice?solvedPage=2&solvedPageSize=5", ""))
	if profile.Solved.Page != 2 || profile.Solved.PageSize != 5 || profile.Solved.Total != 15 {
		t.Fatalf("unexpected solved page metadata: %+v", profile.Solved)
	}
	if len(profile.Solved.Items) != 5 {
		t.Fatalf("solved page item count = %d", len(profile.Solved.Items))
	}
	if len(profile.Activities) != userActivityLimit {
		t.Fatalf("activity count = %d, want %d", len(profile.Activities), userActivityLimit)
	}
	if profile.Solved.Items[0].ID != 1009 || profile.Solved.Items[4].ID != 1005 {
		t.Fatalf("solved page order = %+v", profile.Solved.Items)
	}
}

func TestDatabaseRankUsesVisibleSubmissionStatsAndActiveUsers(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	alice := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	bob := models.User{Name: "bob", Mail: "bob@example.com", Auth: "hash"}
	disabled := models.User{Name: "disabled", Mail: "disabled@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	for _, user := range []*models.User{&alice, &bob, &disabled, &admin} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	if err := db.Delete(&disabled).Error; err != nil {
		t.Fatalf("delete disabled user: %v", err)
	}
	visibleA := models.Problem{ID: 1000, Title: "Visible A", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	visibleB := models.Problem{ID: 1001, Title: "Visible B", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	hidden := models.Problem{ID: 1002, Title: "Hidden", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	contestOnly := models.Problem{ID: 1003, Title: "Running Contest", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	for _, problem := range []*models.Problem{&visibleA, &visibleB, &hidden, &contestOnly} {
		if err := db.Create(problem).Error; err != nil {
			t.Fatalf("create problem %s: %v", problem.Title, err)
		}
	}
	now := time.Now()
	contest := models.Contest{Title: "Running OI", Kind: "OI", StartAt: now.Add(-time.Hour), EndAt: now.Add(time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	if err := db.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: contestOnly.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create contest problem: %v", err)
	}
	contestID := contest.ID
	submissions := []models.Submission{
		{UserID: alice.ID, ProblemID: visibleA.ID, Language: "cpp", Code: "a1", Status: "AC", Score: 100, Public: true},
		{UserID: alice.ID, ProblemID: visibleA.ID, Language: "cpp", Code: "a2", Status: "WA", Score: 0, Public: true},
		{UserID: bob.ID, ProblemID: visibleA.ID, Language: "cpp", Code: "b1", Status: "AC", Score: 100, Public: true},
		{UserID: bob.ID, ProblemID: visibleB.ID, Language: "cpp", Code: "b2", Status: "AC", Score: 100, Public: true},
		{UserID: alice.ID, ProblemID: hidden.ID, Language: "cpp", Code: "hidden", Status: "AC", Score: 100, Public: true},
		{UserID: alice.ID, ProblemID: contestOnly.ID, ContestID: &contestID, Language: "cpp", Code: "contest", Status: "AC", Score: 100, Public: true},
		{UserID: disabled.ID, ProblemID: visibleA.ID, Language: "cpp", Code: "disabled", Status: "AC", Score: 100, Public: true},
	}
	for index := range submissions {
		if err := db.Create(&submissions[index]).Error; err != nil {
			t.Fatalf("create submission: %v", err)
		}
	}

	e := echo.New()
	Register(e, db)

	guestRank := decodeJSON[PageResult[RankUserDTO]](t, requestOK(t, e, http.MethodGet, "/api/rank", ""))
	if len(guestRank.Items) != 3 || guestRank.Total != 3 {
		t.Fatalf("rank should include three active users, got %+v", guestRank)
	}
	if guestRank.Items[0].User != "bob" || guestRank.Items[0].AC != 2 || guestRank.Items[0].Submit != 2 {
		t.Fatalf("bob should rank first by visible AC: %+v", guestRank)
	}
	if userInRank(guestRank.Items, "disabled") {
		t.Fatalf("rank should not include disabled users: %+v", guestRank)
	}
	if res := request(e, http.MethodGet, "/api/users/disabled", "", nil); res.Code != http.StatusNotFound {
		t.Fatalf("disabled user profile should be hidden, got %d body=%s", res.Code, res.Body.String())
	}
	aliceGuest, ok := rankByUser(guestRank.Items, "alice")
	if !ok || aliceGuest.AC != 2 || aliceGuest.Submit != 4 {
		t.Fatalf("alice guest stats should include hidden problem but not running OI AC: %+v", guestRank)
	}
	aliceProfile := decodeJSON[UserProfile](t, requestOK(t, e, http.MethodGet, "/api/users/alice", ""))
	if aliceProfile.User.AC != 2 || aliceProfile.User.Submit != 4 {
		t.Fatalf("alice profile should not expose running OI AC: %+v", aliceProfile.User)
	}

	adminRank := decodeJSON[PageResult[RankUserDTO]](t, requestWithCookies(e, http.MethodGet, "/api/rank", databaseSession(t, db, admin.ID), nil))
	aliceAdmin, ok := rankByUser(adminRank.Items, "alice")
	if !ok || aliceAdmin.AC != 3 || aliceAdmin.Submit != 4 {
		t.Fatalf("alice admin stats should include hidden problem: %+v", adminRank)
	}
	adminProfile := decodeJSON[UserProfile](t, requestWithCookies(e, http.MethodGet, "/api/users/alice", databaseSession(t, db, admin.ID), nil))
	if adminProfile.User.AC != 3 || adminProfile.User.Submit != 4 {
		t.Fatalf("admin profile should include running OI AC: %+v", adminProfile.User)
	}
}

func TestDatabaseRankPaginatesAfterRankingAllUsers(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	for i := 0; i < 105; i++ {
		user := models.User{Name: fmt.Sprintf("u%03d", i), Mail: fmt.Sprintf("u%03d@example.com", i), Auth: "hash"}
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
		if i == 104 {
			if err := db.Create(&models.Submission{UserID: user.ID, ProblemID: problem.ID, Language: "cpp", Code: "ok", Status: "AC", Score: 100, Public: true}).Error; err != nil {
				t.Fatalf("create submission: %v", err)
			}
		}
	}
	e := echo.New()
	Register(e, db)

	first := decodeJSON[PageResult[RankUserDTO]](t, requestOK(t, e, http.MethodGet, "/api/rank?page=1&pageSize=20", ""))
	if first.Total != 105 || len(first.Items) != 20 || first.Items[0].User != "u104" || first.Items[0].Rank != 1 {
		t.Fatalf("rank should sort all users before paging: %+v", first)
	}
	last := decodeJSON[PageResult[RankUserDTO]](t, requestOK(t, e, http.MethodGet, "/api/rank?page=6&pageSize=20", ""))
	if len(last.Items) != 5 || last.Items[0].Rank != 101 {
		t.Fatalf("last rank page = %+v", last)
	}
}

func TestDatabaseContestRankUsesContextSubmissions(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	alice := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	bob := models.User{Name: "bob", Mail: "bob@example.com", Auth: "hash"}
	disabled := models.User{Name: "disabled", Mail: "disabled@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	for _, user := range []*models.User{&alice, &bob, &disabled, &admin} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	if err := db.Delete(&disabled).Error; err != nil {
		t.Fatalf("delete disabled user: %v", err)
	}

	visibleA := models.Problem{ID: 1000, Title: "Visible A", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	visibleB := models.Problem{ID: 1001, Title: "Visible B", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	hidden := models.Problem{ID: 1002, Title: "Hidden", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	for _, problem := range []*models.Problem{&visibleA, &visibleB, &hidden} {
		if err := db.Create(problem).Error; err != nil {
			t.Fatalf("create problem %s: %v", problem.Title, err)
		}
	}
	contest := models.Contest{Title: "Round", Kind: "OI", StartAt: time.Now().Add(-time.Hour), EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	for index, problem := range []models.Problem{visibleA, visibleB, hidden} {
		link := models.ContestProblem{ContestID: contest.ID, ProblemID: problem.ID, Sort: problemSort(index)}
		if err := db.Create(&link).Error; err != nil {
			t.Fatalf("create contest link: %v", err)
		}
	}

	contestID := contest.ID
	submissions := []models.Submission{
		{UserID: alice.ID, ProblemID: visibleA.ID, Language: "cpp", Code: "a1", ContestID: &contestID, Status: "AC", Score: 100, Public: true},
		{UserID: alice.ID, ProblemID: visibleA.ID, Language: "cpp", Code: "a2", ContestID: &contestID, Status: "WA", Score: 0, Public: true},
		{UserID: alice.ID, ProblemID: hidden.ID, Language: "cpp", Code: "hidden", ContestID: &contestID, Status: "AC", Score: 100, Public: true},
		{UserID: alice.ID, ProblemID: visibleB.ID, Language: "cpp", Code: "normal", Status: "AC", Score: 100, Public: true},
		{UserID: bob.ID, ProblemID: visibleA.ID, Language: "cpp", Code: "b1", ContestID: &contestID, Status: "AC", Score: 100, Public: true},
		{UserID: bob.ID, ProblemID: visibleB.ID, Language: "cpp", Code: "b2", ContestID: &contestID, Status: "AC", Score: 100, Public: true},
		{UserID: disabled.ID, ProblemID: visibleA.ID, Language: "cpp", Code: "disabled", ContestID: &contestID, Status: "AC", Score: 100, Public: true},
	}
	for index := range submissions {
		if err := db.Create(&submissions[index]).Error; err != nil {
			t.Fatalf("create submission: %v", err)
		}
	}

	e := echo.New()
	Register(e, db)

	guest := decodeJSON[ContestDetail](t, requestOK(t, e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(contest.ID), 10), ""))
	if len(guest.Rank) != 0 {
		t.Fatalf("OI running contest should hide rank from non-admin users: %+v", guest.Rank)
	}
	if !hasProblem(guest.Problems, hidden.ID) {
		t.Fatalf("OI running contest should still expose linked problems: %+v", guest.Problems)
	}
	aliceDetail := decodeJSON[ContestDetail](t, requestWithCookies(e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(contest.ID), 10), databaseSession(t, db, alice.ID), nil))
	if len(aliceDetail.Rank) != 0 {
		t.Fatalf("OI running contest should hide rank from signed-in users: %+v", aliceDetail.Rank)
	}

	adminDetail := decodeJSON[ContestDetail](t, requestWithCookies(e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(contest.ID), 10), databaseSession(t, db, admin.ID), nil))
	aliceAdmin, ok := rankByUser(adminDetail.Rank, "alice")
	if !ok || aliceAdmin.AC != 1 || aliceAdmin.Score != 100 || aliceAdmin.Submit != 3 {
		t.Fatalf("admin OI rank should use last score and include hidden contest submissions: %+v", adminDetail.Rank)
	}
	bobAdmin, ok := rankByUser(adminDetail.Rank, "bob")
	if !ok || bobAdmin.AC != 2 || bobAdmin.Score != 200 {
		t.Fatalf("admin OI rank should include bob contest submissions: %+v", adminDetail.Rank)
	}
	if userInRank(adminDetail.Rank, "disabled") {
		t.Fatalf("contest rank should not include disabled users: %+v", adminDetail.Rank)
	}
}

func TestDatabaseSubmitStoresAndValidatesContext(t *testing.T) {
	db := testWebDB(t)
	student := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	if err := db.Create(&student).Error; err != nil {
		t.Fatalf("create student: %v", err)
	}
	included := models.Problem{ID: 1000, Title: "Included", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	outside := models.Problem{ID: 1001, Title: "Outside", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	for _, problem := range []*models.Problem{&included, &outside} {
		if err := db.Create(problem).Error; err != nil {
			t.Fatalf("create problem %s: %v", problem.Title, err)
		}
	}
	assignment := models.Assignment{Title: "HW", EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	if err := db.Create(&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: included.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create assignment link: %v", err)
	}
	if err := db.Create(&models.AssignmentUser{AssignmentID: assignment.ID, UserID: student.ID}).Error; err != nil {
		t.Fatalf("assign student: %v", err)
	}

	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, student.ID)
	list := decodePageItems[ProblemDTO](t, requestWithCookies(e, http.MethodGet, "/api/problems", cookies, nil))
	if hasProblem(list, included.ID) {
		t.Fatalf("assignment-only hidden problem should not appear in problem list: %+v", list)
	}
	if res := requestWithCookies(e, http.MethodGet, "/api/problems/1000", cookies, nil); res.Code != http.StatusOK {
		t.Fatalf("assigned active assignment problem detail got %d body=%s", res.Code, res.Body.String())
	}
	okBody := `{"problemId":1000,"language":"cpp","code":"int main(){}","public":true}`
	res := requestJSONWithCookies(e, http.MethodPost, "/api/submissions", cookies, okBody)
	if res.Code != http.StatusCreated {
		t.Fatalf("assignment submission got %d body=%s", res.Code, res.Body.String())
	}
	created := decodeJSON[CreatedID](t, res)
	var row models.Submission
	if err := db.First(&row, "problem_id = ?", included.ID).Error; err != nil {
		t.Fatalf("read created submission: %v", err)
	}
	if created.ID != row.ID {
		t.Fatalf("assignment submission should return created id: got %d row %d", created.ID, row.ID)
	}
	if row.AssignmentID == nil || *row.AssignmentID != assignment.ID {
		t.Fatalf("assignment id not inferred: %+v", row)
	}

	normalBody := `{"problemId":1001,"language":"cpp","code":"int main(){}","public":true}`
	res = requestJSONWithCookies(e, http.MethodPost, "/api/submissions", cookies, normalBody)
	if res.Code != http.StatusCreated {
		t.Fatalf("normal submission got %d body=%s", res.Code, res.Body.String())
	}
	created = decodeJSON[CreatedID](t, res)
	if created.ID == 0 {
		t.Fatalf("normal submission should return created id: %+v", created)
	}
}

func TestContestProblemVisibilityIsDerivedFromContestTime(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Contest Only", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	now := time.Now()
	contest := models.Contest{Title: "Future", Kind: "OI", StartAt: now.Add(time.Hour), EndAt: now.Add(2 * time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	if err := db.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: problem.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create contest problem: %v", err)
	}
	e := echo.New()
	Register(e, db)

	guestList := decodePageItems[ProblemDTO](t, requestOK(t, e, http.MethodGet, "/api/problems", ""))
	if hasProblem(guestList, problem.ID) {
		t.Fatalf("upcoming contest problem leaked in problem list: %+v", guestList)
	}
	if res := request(e, http.MethodGet, "/api/problems/1000", "", nil); res.Code != http.StatusNotFound {
		t.Fatalf("upcoming contest problem detail got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, "/api/problems/1000", databaseSession(t, db, admin.ID), nil); res.Code != http.StatusOK {
		t.Fatalf("admin should see upcoming contest problem, got %d body=%s", res.Code, res.Body.String())
	}

	contest.StartAt = now.Add(-time.Hour)
	contest.EndAt = now.Add(time.Hour)
	if err := db.Save(&contest).Error; err != nil {
		t.Fatalf("start contest: %v", err)
	}
	if err := db.Model(&problem).Update("visible", false).Error; err != nil {
		t.Fatalf("hide problem: %v", err)
	}
	guestList = decodePageItems[ProblemDTO](t, requestOK(t, e, http.MethodGet, "/api/problems", ""))
	if hasProblem(guestList, problem.ID) {
		t.Fatalf("running contest problem leaked in problem list: %+v", guestList)
	}
	if res := request(e, http.MethodGet, "/api/problems/1000", "", nil); res.Code != http.StatusOK {
		t.Fatalf("running contest problem detail should be visible, got %d body=%s", res.Code, res.Body.String())
	}
	contestDetail := decodeJSON[ContestDetail](t, requestOK(t, e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(contest.ID), 10), ""))
	if !hasProblem(contestDetail.Problems, problem.ID) {
		t.Fatalf("running contest detail should include linked problem: %+v", contestDetail.Problems)
	}

	contest.EndAt = now.Add(-time.Minute)
	if err := db.Save(&contest).Error; err != nil {
		t.Fatalf("end contest: %v", err)
	}
	if res := request(e, http.MethodGet, "/api/problems/1000", "", nil); res.Code != http.StatusNotFound {
		t.Fatalf("ended hidden contest problem detail got %d body=%s", res.Code, res.Body.String())
	}
	if err := db.Model(&problem).Update("visible", true).Error; err != nil {
		t.Fatalf("show problem: %v", err)
	}
	if res := request(e, http.MethodGet, "/api/problems/1000", "", nil); res.Code != http.StatusOK {
		t.Fatalf("ended visible contest problem detail got %d body=%s", res.Code, res.Body.String())
	}
}

func TestContestFreezeHidesLateResultsFromNonAdmin(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	alice := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	bob := models.User{Name: "bob", Mail: "bob@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	for _, user := range []*models.User{&alice, &bob, &admin} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	problem := models.Problem{ID: 1000, Title: "Frozen", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	now := time.Now()
	freezeAt := now.Add(-time.Hour)
	contest := models.Contest{Title: "Frozen Round", Kind: "ICPC", StartAt: now.Add(-2 * time.Hour), FreezeAt: &freezeAt, EndAt: now.Add(time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	if err := db.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: problem.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create contest problem: %v", err)
	}
	contestID := contest.ID
	before := models.Submission{UserID: alice.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "before", Status: "AC", Score: 100, Public: true, CreatedAt: now.Add(-90 * time.Minute)}
	aliceAfter := models.Submission{UserID: alice.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "alice after", Status: "WA", Score: 0, Public: true, CreatedAt: now.Add(-30 * time.Minute)}
	bobAfter := models.Submission{UserID: bob.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "bob after", Status: "AC", Score: 100, Public: true, CreatedAt: now.Add(-20 * time.Minute)}
	if err := db.Create(&before).Error; err != nil {
		t.Fatalf("create before submission: %v", err)
	}
	if err := db.Create(&aliceAfter).Error; err != nil {
		t.Fatalf("create alice after submission: %v", err)
	}
	if err := db.Create(&bobAfter).Error; err != nil {
		t.Fatalf("create bob after submission: %v", err)
	}

	e := echo.New()
	Register(e, db)
	target := "/api/contests/" + strconv.FormatUint(uint64(contest.ID), 10)
	guest := decodeJSON[ContestDetail](t, requestOK(t, e, http.MethodGet, target, ""))
	if guest.Contest.Status != "frozen" {
		t.Fatalf("contest status should be frozen: %+v", guest.Contest)
	}
	if len(guest.Rank) != 2 || guest.Rank[0].User != "alice" {
		t.Fatalf("guest rank should score pre-freeze submissions but keep pending post-freeze submitters: %+v", guest.Rank)
	}
	bobGuest, ok := rankByUser(guest.Rank, "bob")
	if !ok || bobGuest.AC != 0 || bobGuest.Penalty != 0 || bobGuest.Submit != 0 {
		t.Fatalf("guest rank should not expose bob post-freeze result: %+v", guest.Rank)
	}
	bobProblem, ok := rankProblemByID(bobGuest.Problems, problem.ID)
	if !ok || bobProblem.Status != "pending" || bobProblem.Submit != 1 || bobProblem.Score != 0 || bobProblem.Penalty != 0 {
		t.Fatalf("guest rank should show bob post-freeze submit as pending: %+v", bobGuest.Problems)
	}

	aliceDetail := decodeJSON[ContestDetail](t, requestWithCookies(e, http.MethodGet, target, databaseSession(t, db, alice.ID), nil))
	if len(aliceDetail.Rank) != 2 || aliceDetail.Rank[0].User != "alice" || aliceDetail.Rank[0].AC != 1 {
		t.Fatalf("alice rank should score pre-freeze submissions and keep pending rows: %+v", aliceDetail.Rank)
	}

	bobDetail := decodeJSON[ContestDetail](t, requestWithCookies(e, http.MethodGet, target, databaseSession(t, db, bob.ID), nil))
	if len(bobDetail.Rank) != 2 || bobDetail.Rank[0].User != "alice" {
		t.Fatalf("bob rank should not score his post-freeze accepted submission: %+v", bobDetail.Rank)
	}

	adminDetail := decodeJSON[ContestDetail](t, requestWithCookies(e, http.MethodGet, target, databaseSession(t, db, admin.ID), nil))
	if _, ok := rankByUser(adminDetail.Rank, "bob"); !ok {
		t.Fatalf("admin rank should include post-freeze submitter: %+v", adminDetail.Rank)
	}
	otherView := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, "/api/submissions/"+strconv.FormatUint(uint64(bobAfter.ID), 10), databaseSession(t, db, alice.ID), nil))
	if otherView.Submission.Status != "pending" || otherView.Submission.Score != 0 || otherView.Code != "" || len(otherView.Cases) != 0 || otherView.Progress != nil {
		t.Fatalf("post-freeze result should be hidden from other users: %+v", otherView)
	}
	ownerView := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, "/api/submissions/"+strconv.FormatUint(uint64(bobAfter.ID), 10), databaseSession(t, db, bob.ID), nil))
	if ownerView.Submission.Status != "AC" || ownerView.Submission.Score != 100 || ownerView.Code != "bob after" {
		t.Fatalf("post-freeze result should be visible to owner: %+v", ownerView)
	}
}

func TestContestOIIgnoresFreezeAndUsesLastScoreAfterEnd(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	alice := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	for _, user := range []*models.User{&alice, &admin} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	problem := models.Problem{ID: 1000, Title: "OI", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	now := time.Now()
	freezeAt := now.Add(-90 * time.Minute)
	contest := models.Contest{Title: "OI Round", Kind: "OI", StartAt: now.Add(-2 * time.Hour), FreezeAt: &freezeAt, EndAt: now.Add(-time.Minute)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	if err := db.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: problem.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create contest problem: %v", err)
	}
	contestID := contest.ID
	submissions := []models.Submission{
		{UserID: alice.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "full", Status: "AC", Score: 100, Public: true, CreatedAt: now.Add(-110 * time.Minute)},
		{UserID: alice.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "partial", Status: "WA", Score: 30, Public: true, CreatedAt: now.Add(-80 * time.Minute)},
	}
	for index := range submissions {
		if err := db.Create(&submissions[index]).Error; err != nil {
			t.Fatalf("create submission: %v", err)
		}
	}

	e := echo.New()
	Register(e, db)
	target := "/api/contests/" + strconv.FormatUint(uint64(contest.ID), 10)
	guest := decodeJSON[ContestDetail](t, requestOK(t, e, http.MethodGet, target, ""))
	if guest.Contest.Status == "frozen" {
		t.Fatalf("OI should ignore freeze status: %+v", guest.Contest)
	}
	if guest.Contest.FreezeAt != nil {
		t.Fatalf("OI detail should not expose freezeAt: %+v", guest.Contest)
	}
	aliceRank, ok := rankByUser(guest.Rank, "alice")
	if !ok || aliceRank.Score != 30 || aliceRank.AC != 0 {
		t.Fatalf("OI score should use the last submission score after contest ends: %+v", guest.Rank)
	}
	aliceProblem, ok := rankProblemByID(aliceRank.Problems, problem.ID)
	if !ok || aliceProblem.Status != "tried" || aliceProblem.Score != 30 || aliceProblem.Submit != 2 {
		t.Fatalf("OI rank should expose per-problem score: %+v", aliceRank.Problems)
	}

	createBody := `{"title":"New OI","kind":"OI","startAt":"` + now.Add(time.Hour).UTC().Format(time.RFC3339) + `","endAt":"` + now.Add(2*time.Hour).UTC().Format(time.RFC3339) + `","freezeAt":"` + now.Add(90*time.Minute).UTC().Format(time.RFC3339) + `","problems":[{"id":1000,"sort":"A"}]}`
	res := requestJSONWithCookies(e, http.MethodPost, "/api/contests", databaseSession(t, db, admin.ID), createBody)
	if res.Code != http.StatusCreated {
		t.Fatalf("create OI with freeze got %d body=%s", res.Code, res.Body.String())
	}
	created := decodeJSON[CreatedID](t, res)
	createdDetail := decodeJSON[ContestDetail](t, requestWithCookies(e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(created.ID), 10), databaseSession(t, db, admin.ID), nil))
	if createdDetail.Contest.FreezeAt != nil {
		t.Fatalf("OI create should ignore freezeAt: %+v", createdDetail.Contest)
	}
}

func TestRunningOIContestHidesSubmissionResults(t *testing.T) {
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
	problem := models.Problem{ID: 1000, Title: "OI", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	contest := models.Contest{Title: "OI Round", Kind: "OI", StartAt: time.Now().Add(-time.Hour), EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	if err := db.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: problem.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create contest problem: %v", err)
	}
	contestID := contest.ID
	timeMS := 12
	memoryKB := 345
	submission := models.Submission{UserID: owner.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "secret", Status: "AC", Score: 100, Message: "accepted", TimeMS: &timeMS, MemoryKB: &memoryKB, Public: true}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	if err := db.Create(&models.Case{SubmissionID: submission.ID, No: 1, Status: "AC", TimeMS: &timeMS, MemoryKB: &memoryKB, Message: "ok"}).Error; err != nil {
		t.Fatalf("create case: %v", err)
	}

	e := echo.New()
	Register(e, db)
	target := "/api/submissions/" + strconv.FormatUint(uint64(submission.ID), 10)
	ownerView := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target, databaseSession(t, db, owner.ID), nil))
	if ownerView.Submission.Status != "pending" || ownerView.Submission.Score != 0 || ownerView.Submission.Message != "" || ownerView.Submission.TimeMS != nil || ownerView.Submission.MemoryKB != nil || len(ownerView.Cases) != 0 || ownerView.Code != "secret" {
		t.Fatalf("running OI owner should see source but pending result: %+v", ownerView)
	}
	otherView := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target, databaseSession(t, db, other.ID), nil))
	if otherView.Submission.Status != "pending" || otherView.Code != "" || len(otherView.Cases) != 0 {
		t.Fatalf("running OI other user should see pending detail without source: %+v", otherView)
	}
	adminView := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, target, databaseSession(t, db, admin.ID), nil))
	if adminView.Submission.Status != "AC" || adminView.Submission.Score != 100 || len(adminView.Cases) != 1 || adminView.Code != "secret" {
		t.Fatalf("admin should see running OI result and source: %+v", adminView)
	}
	contestDetail := decodeJSON[ContestDetail](t, requestWithCookies(e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(contest.ID), 10), databaseSession(t, db, owner.ID), nil))
	if contestDetail.Problems[0].Mine != "pending" || contestDetail.Problems[0].Latest == nil || contestDetail.Problems[0].Latest.Status != "pending" || contestDetail.Problems[0].Latest.Score != 0 {
		t.Fatalf("running OI contest problem should not expose mine/latest result: %+v", contestDetail.Problems)
	}
	ownerProfile := decodeJSON[UserProfile](t, requestWithCookies(e, http.MethodGet, "/api/users/owner", databaseSession(t, db, owner.ID), nil))
	if _, ok := activityBySubmission(ownerProfile.Activities, submission.ID); ok {
		t.Fatalf("profile activity should not expose unfinished contest submission: %+v", ownerProfile.Activities)
	}
	api := &API{db: db}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range databaseSession(t, db, owner.ID) {
		req.AddCookie(cookie)
	}
	ctx := echo.New().NewContext(req, httptest.NewRecorder())
	if got, err := api.submissionListItems(ctx, []models.Submission{submission}); err != nil || len(got) != 1 || got[0].Status != "pending" || got[0].TimeMS != nil || got[0].MemoryKB != nil {
		t.Fatalf("submission list item should hide running OI result: items=%+v err=%v", got, err)
	}
}

func TestRunningAssignmentShowsResultsButHidesOtherSource(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	owner := models.User{Name: "owner", Mail: "owner@example.com", Auth: "hash"}
	other := models.User{Name: "other", Mail: "other@example.com", Auth: "hash"}
	for _, user := range []*models.User{&owner, &other} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	problem := models.Problem{ID: 1000, Title: "HW", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	assignment := models.Assignment{Title: "HW", EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	assignmentID := assignment.ID
	submission := models.Submission{UserID: owner.ID, ProblemID: problem.ID, AssignmentID: &assignmentID, Language: "cpp", Code: "homework", Status: "WA", Score: 20, Public: true}
	if err := db.Create(&submission).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}

	e := echo.New()
	Register(e, db)
	got := decodeJSON[SubmissionDetail](t, requestWithCookies(e, http.MethodGet, "/api/submissions/"+strconv.FormatUint(uint64(submission.ID), 10), databaseSession(t, db, other.ID), nil))
	if got.Submission.Status != "WA" || got.Submission.Score != 20 || got.Code != "" {
		t.Fatalf("running assignment should expose result but hide source from other users: %+v", got)
	}
}

func TestContestICPCRankUsesPenalty(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	alice := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	bob := models.User{Name: "bob", Mail: "bob@example.com", Auth: "hash"}
	for _, user := range []*models.User{&alice, &bob} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	problem := models.Problem{ID: 1000, Title: "ICPC", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	otherProblem := models.Problem{ID: 1001, Title: "Outside AC", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	if err := db.Create(&otherProblem).Error; err != nil {
		t.Fatalf("create other problem: %v", err)
	}
	startAt := time.Now().Add(-time.Hour)
	contest := models.Contest{Title: "ICPC Round", Kind: "ICPC", StartAt: startAt, EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}
	if err := db.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: problem.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create contest problem: %v", err)
	}
	if err := db.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: otherProblem.ID, Sort: "B"}).Error; err != nil {
		t.Fatalf("create other contest problem: %v", err)
	}
	contestID := contest.ID
	submissions := []models.Submission{
		{UserID: alice.ID, ProblemID: otherProblem.ID, Language: "cpp", Code: "outside", Status: "AC", Score: 100, Public: true, CreatedAt: startAt.Add(-time.Minute)},
		{UserID: alice.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "ce", Status: "CE", Score: 0, Public: true, CreatedAt: startAt.Add(2 * time.Minute)},
		{UserID: alice.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "wa", Status: "WA", Score: 0, Public: true, CreatedAt: startAt.Add(5 * time.Minute)},
		{UserID: alice.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "ac", Status: "AC", Score: 100, Public: true, CreatedAt: startAt.Add(10 * time.Minute)},
		{UserID: bob.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "se", Status: "SE", Score: 0, Public: true, CreatedAt: startAt.Add(3 * time.Minute)},
		{UserID: bob.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "ac", Status: "AC", Score: 100, Public: true, CreatedAt: startAt.Add(20 * time.Minute)},
		{UserID: bob.ID, ProblemID: problem.ID, ContestID: &contestID, Language: "cpp", Code: "late wa", Status: "WA", Score: 0, Public: true, CreatedAt: startAt.Add(25 * time.Minute)},
	}
	for index := range submissions {
		if err := db.Create(&submissions[index]).Error; err != nil {
			t.Fatalf("create submission: %v", err)
		}
	}

	e := echo.New()
	Register(e, db)
	detail := decodeJSON[ContestDetail](t, requestOK(t, e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(contest.ID), 10), ""))
	if len(detail.Rank) != 2 {
		t.Fatalf("rank size = %+v", detail.Rank)
	}
	if detail.Rank[0].User != "bob" || detail.Rank[0].AC != 1 || detail.Rank[0].Penalty != 20 {
		t.Fatalf("bob should win by lower penalty: %+v", detail.Rank)
	}
	bobProblem, ok := rankProblemByID(detail.Rank[0].Problems, problem.ID)
	if !ok || bobProblem.Status != "ac" || bobProblem.Penalty != 20 || bobProblem.Submit != 1 {
		t.Fatalf("bob ICPC rank should expose per-problem result without post-AC attempts: %+v", detail.Rank[0].Problems)
	}
	if detail.Rank[1].User != "alice" || detail.Rank[1].AC != 1 || detail.Rank[1].Penalty != 30 {
		t.Fatalf("alice penalty should include wrong submission: %+v", detail.Rank)
	}
	aliceProblem, ok := rankProblemByID(detail.Rank[1].Problems, problem.ID)
	if !ok || aliceProblem.Status != "ac" || aliceProblem.Penalty != 30 || aliceProblem.Submit != 2 {
		t.Fatalf("alice ICPC rank should expose wrong-before-AC count: %+v", detail.Rank[1].Problems)
	}
	aliceDetail := decodeJSON[ContestDetail](t, requestWithCookies(e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(contest.ID), 10), databaseSession(t, db, alice.ID), nil))
	if len(aliceDetail.Problems) != 2 {
		t.Fatalf("contest problems = %+v", aliceDetail.Problems)
	}
	if aliceDetail.Problems[0].Mine != "ac" || aliceDetail.Problems[1].Mine != "none" {
		t.Fatalf("contest problem mine should use contest submissions only: %+v", aliceDetail.Problems)
	}
}

func TestAssignmentMembershipCreateUpdateAndVisibility(t *testing.T) {
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	alice := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	bob := models.User{Name: "bob", Mail: "bob@example.com", Auth: "hash"}
	for _, user := range []*models.User{&admin, &alice, &bob} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user %s: %v", user.Name, err)
		}
	}
	group := models.Group{Name: "team"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := db.Create(&models.GroupUser{GroupID: group.ID, UserID: alice.ID}).Error; err != nil {
		t.Fatalf("add alice to group: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Included", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}

	e := echo.New()
	Register(e, db)
	adminCookies := databaseSession(t, db, admin.ID)
	deadline := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	createBody := `{"title":"HW","endAt":"` + deadline + `","problems":[{"id":1000,"sort":"A"}],"users":[],"groups":[` + strconv.FormatUint(uint64(group.ID), 10) + `]}`
	createRes := requestJSONWithCookies(e, http.MethodPost, "/api/assignments", adminCookies, createBody)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create assignment got %d body=%s", createRes.Code, createRes.Body.String())
	}
	created := decodeJSON[CreatedID](t, createRes)
	createdDetail := decodeJSON[AssignmentDetail](t, requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), adminCookies, nil))
	if len(createdDetail.Assignment.Groups) != 1 || createdDetail.Assignment.Groups[0] != group.ID || len(createdDetail.Assignment.Users) != 0 {
		t.Fatalf("created assignment members not persisted: %+v", createdDetail.Assignment)
	}
	adminList := decodeJSON[PageResult[map[string]any]](t, requestWithCookies(e, http.MethodGet, "/api/assignments", adminCookies, nil)).Items
	if len(adminList) != 1 {
		t.Fatalf("admin assignment list got %+v", adminList)
	}
	if _, ok := adminList[0]["users"]; ok {
		t.Fatalf("assignment list should not include user members: %+v", adminList[0])
	}
	if _, ok := adminList[0]["groups"]; ok {
		t.Fatalf("assignment list should not include group members: %+v", adminList[0])
	}

	aliceDetail := requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), databaseSession(t, db, alice.ID), nil)
	if aliceDetail.Code != http.StatusOK {
		t.Fatalf("group member should see assignment, got %d body=%s", aliceDetail.Code, aliceDetail.Body.String())
	}
	bobDetail := requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), databaseSession(t, db, bob.ID), nil)
	if bobDetail.Code != http.StatusNotFound {
		t.Fatalf("unassigned user should not see assignment, got %d body=%s", bobDetail.Code, bobDetail.Body.String())
	}

	updateBody := `{"title":"HW","endAt":"` + deadline + `","problems":[{"id":1000,"sort":"A"}],"users":[` + strconv.FormatUint(uint64(bob.ID), 10) + `],"groups":[]}`
	updateRes := requestJSONWithCookies(e, http.MethodPatch, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), adminCookies, updateBody)
	if updateRes.Code != http.StatusOK {
		t.Fatalf("update assignment got %d body=%s", updateRes.Code, updateRes.Body.String())
	}
	updated := decodeJSON[CreatedID](t, updateRes)
	if updated.ID != created.ID {
		t.Fatalf("updated assignment should return id: %+v", updated)
	}
	updatedDetail := decodeJSON[AssignmentDetail](t, requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), adminCookies, nil))
	if len(updatedDetail.Assignment.Users) != 1 || updatedDetail.Assignment.Users[0] != bob.ID || len(updatedDetail.Assignment.Groups) != 0 {
		t.Fatalf("updated assignment members not persisted: %+v", updatedDetail.Assignment)
	}
	aliceDetail = requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), databaseSession(t, db, alice.ID), nil)
	if aliceDetail.Code != http.StatusNotFound {
		t.Fatalf("removed group member should lose assignment access, got %d body=%s", aliceDetail.Code, aliceDetail.Body.String())
	}
	bobDetail = requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), databaseSession(t, db, bob.ID), nil)
	if bobDetail.Code != http.StatusOK {
		t.Fatalf("directly assigned user should see assignment, got %d body=%s", bobDetail.Code, bobDetail.Body.String())
	}
}

func TestDatabaseAdminInputValidation(t *testing.T) {
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}

	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)
	now := time.Now().UTC()
	startAt := now.Add(time.Hour).Format(time.RFC3339)
	endAt := now.Add(2 * time.Hour).Format(time.RFC3339)
	deadline := now.Add(24 * time.Hour).Format(time.RFC3339)
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{
			name:   "create problem invalid mode",
			method: http.MethodPost,
			path:   "/api/problems",
			body:   `{"title":"Bad Mode","tags":[],"mode":"bad","timeMs":1000,"memoryMb":256}`,
		},
		{
			name:   "update problem invalid mode",
			method: http.MethodPatch,
			path:   "/api/problems/1000",
			body:   `{"mode":"bad"}`,
		},
		{
			name:   "update problem empty patch",
			method: http.MethodPatch,
			path:   "/api/problems/1000",
			body:   `{}`,
		},
		{
			name:   "create assignment duplicate problem",
			method: http.MethodPost,
			path:   "/api/assignments",
			body:   `{"title":"HW","endAt":"` + deadline + `","problems":[{"id":1000,"sort":"A"},{"id":1000,"sort":"B"}]}`,
		},
		{
			name:   "create assignment missing problem",
			method: http.MethodPost,
			path:   "/api/assignments",
			body:   `{"title":"HW","endAt":"` + deadline + `","problems":[{"id":9999,"sort":"A"}]}`,
		},
		{
			name:   "create contest invalid kind",
			method: http.MethodPost,
			path:   "/api/contests",
			body:   `{"title":"Round","kind":"abc","startAt":"` + startAt + `","endAt":"` + endAt + `","problems":[{"id":1000,"sort":"A"}]}`,
		},
		{
			name:   "create contest missing problem",
			method: http.MethodPost,
			path:   "/api/contests",
			body:   `{"title":"Round","kind":"OI","startAt":"` + startAt + `","endAt":"` + endAt + `","problems":[{"id":9999,"sort":"A"}]}`,
		},
	}
	for _, item := range tests {
		t.Run(item.name, func(t *testing.T) {
			res := requestJSONWithCookies(e, item.method, item.path, cookies, item.body)
			if res.Code != http.StatusBadRequest {
				t.Fatalf("%s got %d body=%s", item.name, res.Code, res.Body.String())
			}
		})
	}
}

func TestProblemVisibilityUpdateDoesNotTouchStatementStorage(t *testing.T) {
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	if err := db.Create(&models.Submission{ProblemID: problem.ID, UserID: admin.ID, Language: "cpp", Code: "int main(){}", Status: "AC", Score: 100, Public: true}).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}
	discussionTags, _ := json.Marshal([]string{"P1000"})
	if err := db.Create(&models.Discussion{Title: "Visible discussion", Content: "body", UserID: admin.ID, Tags: discussionTags}).Error; err != nil {
		t.Fatalf("create discussion: %v", err)
	}
	blockedStorage := filepath.Join(t.TempDir(), "storage-file")
	if err := os.WriteFile(blockedStorage, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("create blocked storage marker: %v", err)
	}
	t.Setenv("STORAGE", blockedStorage)

	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)
	res := requestJSONWithCookies(e, http.MethodPatch, "/api/problems/1000/visibility", cookies, `{"visible":false}`)
	if res.Code != http.StatusOK {
		t.Fatalf("visibility update got %d body=%s", res.Code, res.Body.String())
	}
	updated := decodeJSON[ProblemDTO](t, res)
	if updated.Visible {
		t.Fatalf("problem should be hidden after visibility update: %+v", updated)
	}
	if updated.AC != 1 || updated.Submit != 1 || updated.Discussions != 1 || updated.Mine != "ac" {
		t.Fatalf("visibility response should preserve list decorations: %+v", updated)
	}
	if updated.Cases != nil || updated.DataBytes != nil || updated.Statement != "" {
		t.Fatalf("visibility response should stay a list item without storage/detail fields: %+v", updated)
	}
}

func TestProblemPatchOnlyUpdatesProvidedFields(t *testing.T) {
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Original", Tags: datatypes.JSON([]byte(`["old"]`)), Visible: true, Mode: "default", TimeMS: 2000, MemoryMB: 512}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	t.Setenv("STORAGE", t.TempDir())

	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)
	statement := "# Original\n\nBody"
	statementBody := `{"statement":` + strconv.Quote(statement) + `}`
	if res := requestJSONWithCookies(e, http.MethodPatch, "/api/problems/1000", cookies, statementBody); res.Code != http.StatusOK {
		t.Fatalf("statement patch got %d body=%s", res.Code, res.Body.String())
	} else if got := decodeJSON[CreatedID](t, res); got.ID != problem.ID {
		t.Fatalf("statement patch should return problem id: %+v", got)
	}
	res := requestJSONWithCookies(e, http.MethodPatch, "/api/problems/1000", cookies, `{"mode":"strict"}`)
	if res.Code != http.StatusOK {
		t.Fatalf("mode patch got %d body=%s", res.Code, res.Body.String())
	}
	if got := decodeJSON[CreatedID](t, res); got.ID != problem.ID {
		t.Fatalf("mode patch should return problem id: %+v", got)
	}
	updated := decodeJSON[ProblemDTO](t, requestWithCookies(e, http.MethodGet, "/api/problems/1000", cookies, nil))
	if updated.Mode != "strict" || updated.Title != "Original" || updated.Statement != statement || !updated.Visible || updated.TimeMS != 2000 || updated.MemoryMB != 512 {
		t.Fatalf("mode patch should preserve unrelated problem fields: %+v", updated)
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/problems/1000", cookies, `{"visible":false}`)
	if res.Code != http.StatusOK {
		t.Fatalf("visible false patch got %d body=%s", res.Code, res.Body.String())
	}
	if got := decodeJSON[CreatedID](t, res); got.ID != problem.ID {
		t.Fatalf("visible patch should return problem id: %+v", got)
	}
	updated = decodeJSON[ProblemDTO](t, requestWithCookies(e, http.MethodGet, "/api/problems/1000", cookies, nil))
	if updated.Visible || updated.Mode != "strict" || updated.Title != "Original" || updated.Statement != statement {
		t.Fatalf("visible false patch should apply false and preserve unrelated fields: %+v", updated)
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/problems/1000", cookies, `{"tags":[]}`)
	if res.Code != http.StatusOK {
		t.Fatalf("empty tags patch got %d body=%s", res.Code, res.Body.String())
	}
	if got := decodeJSON[CreatedID](t, res); got.ID != problem.ID {
		t.Fatalf("tags patch should return problem id: %+v", got)
	}
	updated = decodeJSON[ProblemDTO](t, requestWithCookies(e, http.MethodGet, "/api/problems/1000", cookies, nil))
	if len(updated.Tags) != 0 || updated.Visible || updated.Mode != "strict" || updated.Statement != statement {
		t.Fatalf("empty tags patch should clear tags and preserve unrelated fields: %+v", updated)
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/problems/1000", cookies, `{"timeMs":0,"memoryMb":0}`)
	if res.Code != http.StatusOK {
		t.Fatalf("zero limit patch got %d body=%s", res.Code, res.Body.String())
	}
	if got := decodeJSON[CreatedID](t, res); got.ID != problem.ID {
		t.Fatalf("limit patch should return problem id: %+v", got)
	}
	updated = decodeJSON[ProblemDTO](t, requestWithCookies(e, http.MethodGet, "/api/problems/1000", cookies, nil))
	if updated.TimeMS != 1000 || updated.MemoryMB != 256 || updated.Mode != "strict" || updated.Statement != statement {
		t.Fatalf("zero limit patch should use defaults and preserve unrelated fields: %+v", updated)
	}
}

func TestProblemCreateDefaultsVisibleAndListSortsByID(t *testing.T) {
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	rows := []models.Problem{
		{ID: 1002, Title: "B", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
		{ID: 1000, Title: "A", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create problem %d: %v", row.ID, err)
		}
	}

	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)
	created := decodeJSON[CreatedID](t, requestJSONWithCookies(e, http.MethodPost, "/api/problems", cookies, `{"title":"Created","tags":[],"mode":"default","timeMs":1000,"memoryMb":256}`))
	createdDetail := decodeJSON[ProblemDTO](t, requestWithCookies(e, http.MethodGet, "/api/problems/"+strconv.FormatUint(uint64(created.ID), 10), cookies, nil))
	if !createdDetail.Visible {
		t.Fatalf("created problem should default to visible: %+v", createdDetail)
	}
	items := decodePageItems[ProblemDTO](t, requestWithCookies(e, http.MethodGet, "/api/problems", cookies, nil))
	if len(items) < 3 {
		t.Fatalf("problem list too short: %+v", items)
	}
	ids := []uint{items[0].ID, items[1].ID, items[2].ID}
	if ids[0] != 1000 || ids[1] != 1002 || ids[2] != created.ID {
		t.Fatalf("problem list should sort by id asc, got %+v", ids)
	}
}

func TestProblemListDoesNotTouchAssetStorage(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	blockedStorage := filepath.Join(t.TempDir(), "storage-file")
	if err := os.WriteFile(blockedStorage, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("create blocked storage marker: %v", err)
	}
	t.Setenv("STORAGE", blockedStorage)

	e := echo.New()
	Register(e, db)
	res := requestOK(t, e, http.MethodGet, "/api/problems", "")
	items := decodePageItems[ProblemDTO](t, res)
	if len(items) != 1 || items[0].ID != problem.ID {
		t.Fatalf("problem list got %+v, want P%d", items, problem.ID)
	}
	if items[0].Cases != nil || items[0].DataBytes != nil {
		t.Fatalf("problem list should not compute storage-derived stats: %+v", items[0])
	}
}

func TestProblemListSearchesByCode(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	rows := []models.Problem{
		{ID: 1288, Title: "Window Median", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
		{ID: 1289, Title: "Deer Tower", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create problem %d: %v", row.ID, err)
		}
	}

	e := echo.New()
	Register(e, db)

	for _, q := range []string{"1289", "P1289"} {
		got := decodePageItems[ProblemDTO](t, requestOK(t, e, http.MethodGet, "/api/problems?q="+url.QueryEscape(q), ""))
		if len(got) != 1 || got[0].ID != 1289 {
			t.Fatalf("problem search %q got %+v, want P1289", q, got)
		}
	}
}

func TestDiscussionProblemTagsAreSoftAssociations(t *testing.T) {
	db := testWebDB(t)
	student := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&student).Error; err != nil {
		t.Fatalf("create student: %v", err)
	}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{
		ID:       1000,
		Title:    "Hidden",
		Tags:     datatypes.JSON([]byte(`["hidden"]`)),
		Visible:  false,
		Mode:     "default",
		TimeMS:   1000,
		MemoryMB: 256,
	}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	discussion := models.Discussion{
		Title:   "Hidden discussion",
		Content: "secret",
		UserID:  admin.ID,
		Tags:    datatypes.JSON([]byte(`["P1000"]`)),
		Locked:  false,
	}
	if err := db.Create(&discussion).Error; err != nil {
		t.Fatalf("create discussion: %v", err)
	}

	e := echo.New()
	Register(e, db)

	studentCookies := databaseSession(t, db, student.ID)
	adminCookies := databaseSession(t, db, admin.ID)
	discussionBody := `{"title":"Hidden tagged discussion","content":"secret","tags":["P1000"]}`
	res := requestJSONWithCookies(e, http.MethodPost, "/api/discussion", studentCookies, discussionBody)
	if res.Code != http.StatusCreated {
		t.Fatalf("student create hidden discussion got %d body=%s", res.Code, res.Body.String())
	}
	res = requestJSONWithCookies(e, http.MethodPost, "/api/discussion", adminCookies, discussionBody)
	if res.Code != http.StatusCreated {
		t.Fatalf("admin create hidden discussion got %d body=%s", res.Code, res.Body.String())
	}

	body := `{"content":"I should not see this"}`
	target := "/api/discussion/" + strconv.FormatUint(uint64(discussion.ID), 10) + "/comments"
	res = requestJSONWithCookies(e, http.MethodPost, target, studentCookies, body)
	if res.Code != http.StatusCreated {
		t.Fatalf("student comment on hidden discussion got %d body=%s", res.Code, res.Body.String())
	}

	res = requestJSONWithCookies(e, http.MethodPost, target, adminCookies, body)
	if res.Code != http.StatusCreated {
		t.Fatalf("admin comment on hidden discussion got %d body=%s", res.Code, res.Body.String())
	}
}

func TestDatabaseDiscussionAuthorsUseNames(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	student := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if err := db.Create(&student).Error; err != nil {
		t.Fatalf("create student: %v", err)
	}
	discussion := models.Discussion{
		Title:   "Named discussion",
		Content: "body",
		UserID:  admin.ID,
		Tags:    datatypes.JSON([]byte(`["general"]`)),
	}
	if err := db.Create(&discussion).Error; err != nil {
		t.Fatalf("create discussion: %v", err)
	}
	comment := models.Comment{DiscussionID: discussion.ID, UserID: student.ID, Content: "reply"}
	if err := db.Create(&comment).Error; err != nil {
		t.Fatalf("create comment: %v", err)
	}

	e := echo.New()
	Register(e, db)

	list := decodePageItems[DiscussionDTO](t, requestOK(t, e, http.MethodGet, "/api/discussion", ""))
	if len(list) != 1 || list[0].Author != "admin" || list[0].Replies != 1 {
		t.Fatalf("discussion list should include author and reply count: %+v", list)
	}
	listRes := requestOK(t, e, http.MethodGet, "/api/discussion", "")
	rawList := decodeJSON[PageResult[map[string]any]](t, listRes).Items
	if _, ok := rawList[0]["content"]; ok {
		t.Fatalf("discussion list should not include content: %+v", rawList[0])
	}

	target := "/api/discussion/" + strconv.FormatUint(uint64(discussion.ID), 10)
	detail := decodeJSON[DiscussionDetail](t, requestOK(t, e, http.MethodGet, target, ""))
	if detail.Discussion.Author != "admin" || len(detail.Comments) != 1 || detail.Comments[0].Author != "student" {
		t.Fatalf("discussion detail authors should be usernames: %+v", detail)
	}

	updated := requestJSONWithCookies(e, http.MethodPatch, target, databaseSession(t, db, admin.ID), `{"pinned":true}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("update discussion got %d body=%s", updated.Code, updated.Body.String())
	}
	updatedID := decodeJSON[CreatedID](t, updated)
	if updatedID.ID != discussion.ID {
		t.Fatalf("partial discussion update should return updated id: %+v", updatedID)
	}
	detail = decodeJSON[DiscussionDetail](t, requestOK(t, e, http.MethodGet, target, ""))
	if !detail.Discussion.Pinned || detail.Discussion.Locked || detail.Discussion.Title != "Named discussion" || detail.Discussion.Replies != 1 || detail.Content != "body" || len(detail.Discussion.Tags) != 1 || detail.Discussion.Tags[0] != "general" {
		t.Fatalf("partial discussion update should preserve content and tags: %+v", detail)
	}
	updated = requestJSONWithCookies(e, http.MethodPatch, target, databaseSession(t, db, admin.ID), `{"pinned":false,"locked":true}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("false/true discussion patch got %d body=%s", updated.Code, updated.Body.String())
	}
	updatedID = decodeJSON[CreatedID](t, updated)
	if updatedID.ID != discussion.ID {
		t.Fatalf("false/true discussion patch should return updated id: %+v", updatedID)
	}
	detail = decodeJSON[DiscussionDetail](t, requestOK(t, e, http.MethodGet, target, ""))
	if detail.Discussion.Pinned || !detail.Discussion.Locked || detail.Discussion.Title != "Named discussion" || detail.Discussion.Replies != 1 {
		t.Fatalf("discussion patch should apply false/true flags and preserve unrelated fields: %+v", detail)
	}
	updated = requestJSONWithCookies(e, http.MethodPatch, target, databaseSession(t, db, admin.ID), `{"locked":false,"tags":[]}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("false/empty tags discussion patch got %d body=%s", updated.Code, updated.Body.String())
	}
	updatedID = decodeJSON[CreatedID](t, updated)
	if updatedID.ID != discussion.ID {
		t.Fatalf("false/empty tags discussion patch should return updated id: %+v", updatedID)
	}
	detail = decodeJSON[DiscussionDetail](t, requestOK(t, e, http.MethodGet, target, ""))
	if detail.Discussion.Locked || len(detail.Discussion.Tags) != 0 || detail.Discussion.Title != "Named discussion" || detail.Discussion.Replies != 1 || detail.Content != "body" {
		t.Fatalf("discussion false/empty tags patch should preserve content: %+v", detail)
	}
	empty := requestJSONWithCookies(e, http.MethodPatch, target, databaseSession(t, db, admin.ID), `{}`)
	if empty.Code != http.StatusBadRequest {
		t.Fatalf("empty discussion patch got %d body=%s", empty.Code, empty.Body.String())
	}
}

func TestDiscussionDeleteAllowsOwnerOrAdmin(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	users := []models.User{
		{Name: "owner", Mail: "owner@example.com", Auth: "hash"},
		{Name: "other", Mail: "other@example.com", Auth: "hash"},
		{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	owner, other, admin := users[0], users[1], users[2]
	discussions := []models.Discussion{
		{Title: "owner post", Content: "body", UserID: owner.ID, Tags: datatypes.JSON([]byte(`[]`))},
		{Title: "admin delete", Content: "body", UserID: owner.ID, Tags: datatypes.JSON([]byte(`[]`))},
	}
	if err := db.Create(&discussions).Error; err != nil {
		t.Fatalf("create discussions: %v", err)
	}

	e := echo.New()
	Register(e, db)
	target := func(id uint) string { return "/api/discussion/" + strconv.FormatUint(uint64(id), 10) }

	if res := requestWithCookies(e, http.MethodDelete, target(discussions[0].ID), nil, nil); res.Code != http.StatusUnauthorized {
		t.Fatalf("guest delete got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodDelete, target(discussions[0].ID), databaseSession(t, db, other.ID), nil); res.Code != http.StatusForbidden {
		t.Fatalf("other user delete got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodDelete, target(discussions[0].ID), databaseSession(t, db, owner.ID), nil); res.Code != http.StatusNoContent {
		t.Fatalf("owner delete got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodDelete, target(discussions[0].ID), databaseSession(t, db, admin.ID), nil); res.Code != http.StatusNotFound {
		t.Fatalf("deleted discussion got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodDelete, target(discussions[1].ID), databaseSession(t, db, admin.ID), nil); res.Code != http.StatusNoContent {
		t.Fatalf("admin delete got %d body=%s", res.Code, res.Body.String())
	}
}

func TestCommentDeleteKeepsFloorSlots(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	users := []models.User{
		{Name: "owner", Mail: "owner@example.com", Auth: "hash"},
		{Name: "other", Mail: "other@example.com", Auth: "hash"},
		{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	owner, other, admin := users[0], users[1], users[2]
	discussion := models.Discussion{Title: "post", Content: "body", UserID: owner.ID, Tags: datatypes.JSON([]byte(`[]`))}
	if err := db.Create(&discussion).Error; err != nil {
		t.Fatalf("create discussion: %v", err)
	}
	comments := []models.Comment{
		{DiscussionID: discussion.ID, UserID: owner.ID, Content: "first"},
		{DiscussionID: discussion.ID, UserID: other.ID, Content: "second"},
		{DiscussionID: discussion.ID, UserID: owner.ID, Content: "third"},
	}
	if err := db.Create(&comments).Error; err != nil {
		t.Fatalf("create comments: %v", err)
	}

	e := echo.New()
	Register(e, db)
	target := func(id uint) string {
		return "/api/discussion/" + strconv.FormatUint(uint64(discussion.ID), 10) + "/comments/" + strconv.FormatUint(uint64(id), 10)
	}

	if res := requestWithCookies(e, http.MethodDelete, target(comments[1].ID), nil, nil); res.Code != http.StatusUnauthorized {
		t.Fatalf("guest delete comment got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodDelete, target(comments[1].ID), databaseSession(t, db, owner.ID), nil); res.Code != http.StatusForbidden {
		t.Fatalf("other user delete comment got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodDelete, target(comments[1].ID), databaseSession(t, db, other.ID), nil); res.Code != http.StatusNoContent {
		t.Fatalf("owner delete comment got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodDelete, target(comments[0].ID), databaseSession(t, db, admin.ID), nil); res.Code != http.StatusNoContent {
		t.Fatalf("admin delete comment got %d body=%s", res.Code, res.Body.String())
	}

	detail := decodeJSON[DiscussionDetail](t, requestOK(t, e, http.MethodGet, "/api/discussion/"+strconv.FormatUint(uint64(discussion.ID), 10), ""))
	if detail.Discussion.Replies != 1 || len(detail.Comments) != 3 {
		t.Fatalf("deleted comments should stay as floor slots but not active replies: %+v", detail)
	}
	if !detail.Comments[0].Deleted || detail.Comments[0].Content != "" || !detail.Comments[1].Deleted || detail.Comments[2].Deleted || detail.Comments[2].Content != "third" {
		t.Fatalf("comment tombstones should preserve order and hide content: %+v", detail.Comments)
	}
}

func TestDiscussionListSearchesTitleContentAndTags(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	rows := []models.Discussion{
		{Title: "Segment tree notes", Content: "body", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["general"]`))},
		{Title: "Other topic", Content: "Fenwick tree detail", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["general"]`))},
		{Title: "Tagged topic", Content: "body", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["P1289"]`))},
		{Title: "Unrelated", Content: "body", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["misc"]`))},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatalf("create discussions: %v", err)
	}

	e := echo.New()
	Register(e, db)

	for _, item := range []struct {
		q    string
		want string
	}{
		{q: "segment", want: "Segment tree notes"},
		{q: "fenwick", want: "Other topic"},
		{q: "p1289", want: "Tagged topic"},
	} {
		got := decodePageItems[DiscussionDTO](t, requestOK(t, e, http.MethodGet, "/api/discussion?q="+url.QueryEscape(item.q), ""))
		if len(got) != 1 || got[0].Title != item.want {
			t.Fatalf("search %q got %+v, want %q", item.q, got, item.want)
		}
	}
}

func TestDynamicSelectSuggestionEndpoints(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	users := []models.User{
		{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true},
		{Name: "alice", Mail: "alice@example.com", Auth: "hash"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	admin := users[0]
	problem := models.Problem{ID: 1000, Title: "A+B", Tags: datatypes.JSON([]byte(`["math","beginner"]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	otherProblem := models.Problem{ID: 1001, Title: "DP", Tags: datatypes.JSON([]byte(`["dp"]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&[]models.Problem{problem, otherProblem}).Error; err != nil {
		t.Fatalf("create problems: %v", err)
	}
	if err := db.Create(&models.Discussion{Title: "General", Content: "body", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["general","P1000"]`))}).Error; err != nil {
		t.Fatalf("create discussion: %v", err)
	}
	assignment := models.Assignment{Title: "Summer Homework", EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	if err := db.Create(&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: problem.ID, Sort: "A"}).Error; err != nil {
		t.Fatalf("create assignment problem: %v", err)
	}
	contest := models.Contest{Title: "Winter Cup", Kind: "OI", StartAt: time.Now().Add(-time.Hour), EndAt: time.Now().Add(time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatalf("create contest: %v", err)
	}

	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)

	problemTags := decodeJSON[[]string](t, requestOK(t, e, http.MethodGet, "/api/tags?kind=problem&q=ma", ""))
	if len(problemTags) != 1 || problemTags[0] != "math" {
		t.Fatalf("problem tag suggestions = %+v", problemTags)
	}
	discussionTags := decodeJSON[[]string](t, requestOK(t, e, http.MethodGet, "/api/tags?kind=discussion&q=gen", ""))
	if len(discussionTags) != 1 || discussionTags[0] != "general" {
		t.Fatalf("discussion tag suggestions = %+v", discussionTags)
	}
	userOptions := decodeJSON[[]UserOptionDTO](t, requestOK(t, e, http.MethodGet, "/api/users?q=ali", ""))
	if len(userOptions) != 1 || userOptions[0].Name != "alice" {
		t.Fatalf("user suggestions = %+v", userOptions)
	}
	assignments := decodePageItems[AssignmentDTO](t, requestWithCookies(e, http.MethodGet, "/api/assignments?q=Summer", cookies, nil))
	if len(assignments) != 1 || assignments[0].Title != "Summer Homework" {
		t.Fatalf("assignment suggestions = %+v", assignments)
	}
	contests := decodePageItems[ContestDTO](t, requestWithCookies(e, http.MethodGet, "/api/contests?q=Winter", cookies, nil))
	if len(contests) != 1 || contests[0].Title != "Winter Cup" {
		t.Fatalf("contest suggestions = %+v", contests)
	}
}

func TestHomeProblemsUseCompactPayload(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	user := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	solved := models.Problem{ID: 1000, Title: "Solved", Tags: datatypes.JSON([]byte(`["math","beginner"]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	unsolved := models.Problem{ID: 1001, Title: "Unsolved", Tags: datatypes.JSON([]byte(`["dp"]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&[]models.Problem{solved, unsolved}).Error; err != nil {
		t.Fatalf("create problems: %v", err)
	}
	if err := db.Create(&models.Submission{UserID: user.ID, ProblemID: solved.ID, Language: "cpp", Code: "code", Status: "AC", Score: 100, Public: true}).Error; err != nil {
		t.Fatalf("create submission: %v", err)
	}

	e := echo.New()
	Register(e, db)

	res := requestWithCookies(e, http.MethodGet, "/api/home", databaseSession(t, db, user.ID), nil)
	home := decodeJSON[Home](t, res)
	if len(home.Problems) != 1 || home.Problems[0].ID != unsolved.ID || home.Problems[0].Title != unsolved.Title {
		t.Fatalf("home problems should include unsolved compact identity fields only: %+v", home.Problems)
	}
	var raw struct {
		Problems []map[string]any `json:"problems"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw home: %v", err)
	}
	for _, key := range []string{"tags", "visible", "mode", "timeMs", "memoryMb", "discussions", "mine", "latest", "ac", "submit"} {
		if _, ok := raw.Problems[0][key]; ok {
			t.Fatalf("home problem should not include list-only field %q: %+v", key, raw.Problems[0])
		}
	}
}

func TestHomeFiltersAssignmentsAndContestsForCurrentUser(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	user := models.User{Name: "alice", Mail: "alice@example.com", Auth: "hash"}
	other := models.User{Name: "bob", Mail: "bob@example.com", Auth: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create other user: %v", err)
	}
	problems := []models.Problem{
		{ID: 1000, Title: "A", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
		{ID: 1001, Title: "B", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
	}
	if err := db.Create(&problems).Error; err != nil {
		t.Fatalf("create problems: %v", err)
	}
	now := time.Now()
	assigned := models.Assignment{Title: "Assigned", EndAt: now.Add(24 * time.Hour)}
	unassigned := models.Assignment{Title: "Unassigned", EndAt: now.Add(48 * time.Hour)}
	if err := db.Create(&assigned).Error; err != nil {
		t.Fatalf("create assigned assignment: %v", err)
	}
	if err := db.Create(&unassigned).Error; err != nil {
		t.Fatalf("create unassigned assignment: %v", err)
	}
	assignmentProblems := []models.AssignmentProblem{
		{AssignmentID: assigned.ID, ProblemID: problems[0].ID, Sort: "A"},
		{AssignmentID: assigned.ID, ProblemID: problems[1].ID, Sort: "B"},
		{AssignmentID: unassigned.ID, ProblemID: problems[0].ID, Sort: "A"},
	}
	if err := db.Create(&assignmentProblems).Error; err != nil {
		t.Fatalf("create assignment problems: %v", err)
	}
	if err := db.Create(&models.AssignmentUser{AssignmentID: assigned.ID, UserID: user.ID}).Error; err != nil {
		t.Fatalf("assign user: %v", err)
	}
	if err := db.Create(&models.AssignmentUser{AssignmentID: unassigned.ID, UserID: other.ID}).Error; err != nil {
		t.Fatalf("assign other: %v", err)
	}
	if err := db.Create(&models.Submission{UserID: user.ID, ProblemID: problems[0].ID, AssignmentID: &assigned.ID, Language: "cpp", Code: "code", Status: "AC", Score: 100, Public: true}).Error; err != nil {
		t.Fatalf("create assignment submission: %v", err)
	}
	running := models.Contest{Title: "Running", Kind: "OI", StartAt: now.Add(-time.Hour), EndAt: now.Add(time.Hour)}
	pending := models.Contest{Title: "Pending", Kind: "OI", StartAt: now.Add(time.Hour), EndAt: now.Add(2 * time.Hour)}
	ended := models.Contest{Title: "Ended", Kind: "OI", StartAt: now.Add(-2 * time.Hour), EndAt: now.Add(-time.Hour)}
	if err := db.Create(&running).Error; err != nil {
		t.Fatalf("create running contest: %v", err)
	}
	if err := db.Create(&pending).Error; err != nil {
		t.Fatalf("create pending contest: %v", err)
	}
	if err := db.Create(&ended).Error; err != nil {
		t.Fatalf("create ended contest: %v", err)
	}
	contestProblems := []models.ContestProblem{
		{ContestID: running.ID, ProblemID: problems[0].ID, Sort: "A"},
		{ContestID: pending.ID, ProblemID: problems[0].ID, Sort: "A"},
		{ContestID: ended.ID, ProblemID: problems[0].ID, Sort: "A"},
	}
	if err := db.Create(&contestProblems).Error; err != nil {
		t.Fatalf("create contest problems: %v", err)
	}

	e := echo.New()
	Register(e, db)
	home := decodeJSON[Home](t, requestWithCookies(e, http.MethodGet, "/api/home", databaseSession(t, db, user.ID), nil))
	if len(home.Assignments) != 1 || home.Assignments[0].ID != assigned.ID || home.Assignments[0].Done != 1 || home.Assignments[0].Total != 2 {
		t.Fatalf("home assignments should include assigned progress only: %+v", home.Assignments)
	}
	contestStatuses := map[string]string{}
	for _, contest := range home.Contests {
		contestStatuses[contest.Title] = contest.Status
	}
	if len(contestStatuses) != 2 || contestStatuses["Running"] != "running" || contestStatuses["Pending"] != "pending" {
		t.Fatalf("home contests should include active contests with status only: %+v", home.Contests)
	}
}

func TestImageUploadUsesRelativeMediaPathsAndHeaders(t *testing.T) {
	t.Setenv("STORAGE", t.TempDir())
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	student := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if err := db.Create(&student).Error; err != nil {
		t.Fatalf("create student: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}

	e := echo.New()
	Register(e, db)
	studentCookies := databaseSession(t, db, student.ID)
	adminCookies := databaseSession(t, db, admin.ID)

	userImage := uploadImageForTest(t, e, "/api/uploads/images", studentCookies, "avatar.png", tinyPNG())
	if !strings.HasPrefix(userImage.URL, "/api/users/") || strings.Contains(userImage.URL, "://") {
		t.Fatalf("user image url should be a relative media path, got %q", userImage.URL)
	}
	res := requestWithCookies(e, http.MethodGet, userImage.URL, studentCookies, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("read user image got %d body=%s", res.Code, res.Body.String())
	}
	if cache := res.Header().Get(echo.HeaderCacheControl); cache != "public, max-age=31536000, immutable" {
		t.Fatalf("user image cache header = %q", cache)
	}
	res = requestWithCookiesAndReferer(e, http.MethodGet, userImage.URL, studentCookies, "https://evil.example/post")
	if res.Code != http.StatusForbidden {
		t.Fatalf("cross-site media request got %d body=%s", res.Code, res.Body.String())
	}

	problemImage := uploadImageForTest(t, e, "/api/problems/1000/assets/images", adminCookies, "statement.png", tinyPNG())
	if !strings.HasPrefix(problemImage.URL, "/api/problems/1000/assets/") || strings.Contains(problemImage.URL, "://") {
		t.Fatalf("problem image url should be a relative media path, got %q", problemImage.URL)
	}
	rel := strings.TrimPrefix(problemImage.URL, "/api/problems/1000/assets/")
	if strings.Contains(rel, "/") {
		t.Fatalf("problem image should not include date folders, got %q", problemImage.URL)
	}
	if _, err := os.Stat(filepath.Join(utils.UploadRoot(), "problems", "1000", "assets", filepath.FromSlash(rel))); err != nil {
		t.Fatalf("problem image should keep the existing object key convention: %v", err)
	}
	res = requestWithCookies(e, http.MethodGet, problemImage.URL, adminCookies, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("read problem image got %d body=%s", res.Code, res.Body.String())
	}
}

func TestProblemAssetDownloadsSupportNestedPathsAndExistingProblems(t *testing.T) {
	t.Setenv("STORAGE", t.TempDir())
	db := testWebDB(t)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}

	e := echo.New()
	Register(e, db)
	adminCookies := databaseSession(t, db, admin.ID)
	nestedPath := filepath.Join(utils.UploadRoot(), "problems", "1000", "data", "cases", "1.in")
	if err := os.MkdirAll(filepath.Dir(nestedPath), 0o755); err != nil {
		t.Fatalf("create nested data dir: %v", err)
	}
	if err := os.WriteFile(nestedPath, []byte("nested input"), 0o644); err != nil {
		t.Fatalf("write nested data file: %v", err)
	}
	for key, body := range map[string]string{
		filepath.Join(utils.UploadRoot(), "problems", "1000", "statement.md"):           "# Visible\n\n![img](./assets/note.png)",
		filepath.Join(utils.UploadRoot(), "problems", "1000", "data", "cases", "1.out"): "nested output",
		filepath.Join(utils.UploadRoot(), "problems", "1000", "judge", "main.cc"):       "int main(){}",
		filepath.Join(utils.UploadRoot(), "problems", "1000", "assets", "note.txt"):     "asset note",
	} {
		if err := os.MkdirAll(filepath.Dir(key), 0o755); err != nil {
			t.Fatalf("create asset dir: %v", err)
		}
		if err := os.WriteFile(key, []byte(body), 0o644); err != nil {
			t.Fatalf("write asset file: %v", err)
		}
	}

	res := requestWithCookies(e, http.MethodGet, "/api/problems/1000/data/cases/1.in", adminCookies, nil)
	if res.Code != http.StatusOK || res.Body.String() != "nested input" {
		t.Fatalf("nested asset download got %d body=%q", res.Code, res.Body.String())
	}
	res = requestWithCookies(e, http.MethodGet, "/api/problems/1000.zip", adminCookies, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("problem zip got %d body=%s", res.Code, res.Body.String())
	}
	if disposition := res.Header().Get(echo.HeaderContentDisposition); !strings.Contains(disposition, `filename="P1000.zip"`) {
		t.Fatalf("problem zip content disposition = %q", disposition)
	}
	reader, err := zip.NewReader(bytes.NewReader(res.Body.Bytes()), int64(res.Body.Len()))
	if err != nil {
		t.Fatalf("read problem zip: %v", err)
	}
	names := map[string]bool{}
	content := map[string]string{}
	for _, file := range reader.File {
		names[file.Name] = true
		body, err := readZipFile(file)
		if err != nil {
			t.Fatalf("read zip file %s: %v", file.Name, err)
		}
		content[file.Name] = string(body)
	}
	for _, name := range []string{"statement.md", "data/cases/1.in", "data/cases/1.out", "judge/main.cc", "assets/note.txt"} {
		if !names[name] {
			t.Fatalf("problem zip missing %s, got %+v", name, names)
		}
	}
	if content["statement.md"] != "# Visible\n\n![img](./assets/note.png)" {
		t.Fatalf("problem zip statement = %q", content["statement.md"])
	}
	res = requestWithCookies(e, http.MethodGet, "/api/problems/404.zip", adminCookies, nil)
	if res.Code != http.StatusNotFound {
		t.Fatalf("missing problem zip got %d body=%s", res.Code, res.Body.String())
	}
}

func readZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func TestProblemAssetsUseCaseFileOrder(t *testing.T) {
	t.Setenv("STORAGE", t.TempDir())
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		t.Fatalf("object store: %v", err)
	}
	for _, name := range []string{"10.out", "2.out", "1.out", "10.in", "readme.txt", "3.ans", "2.in", "input4.txt", "answer4.txt", "3.in", "1.in"} {
		key := path.Join("problems", "1000", "data", name)
		if err := store.Put(context.Background(), key, strings.NewReader(name), int64(len(name)), "text/plain"); err != nil {
			t.Fatalf("put %s: %v", name, err)
		}
	}
	assets, err := problemAssetsFromStore(context.Background(), 1000, store)
	if err != nil {
		t.Fatalf("problem assets: %v", err)
	}
	var got []string
	for _, item := range assets.Data {
		got = append(got, item.Name)
	}
	want := []string{"1.in", "1.out", "2.in", "2.out", "3.in", "3.ans", "input4.txt", "answer4.txt", "10.in", "10.out", "readme.txt"}
	if !slices.Equal(got, want) {
		t.Fatalf("data files = %+v, want %+v", got, want)
	}
	if assets.Cases != 5 {
		t.Fatalf("cases = %d, want 5", assets.Cases)
	}
}

func TestSafeAssetZipNameRejectsUnsafeNames(t *testing.T) {
	name, ok := safeAssetZipName("data", "cases/1.in")
	if !ok || name != "data/cases/1.in" {
		t.Fatalf("safe nested asset name = %q, %v", name, ok)
	}
	for _, unsafe := range []string{"../evil", "cases/../../evil", "/absolute", `cases\..\evil`, "cases//1.in"} {
		if name, ok := safeAssetZipName("data", unsafe); ok {
			t.Fatalf("unsafe asset name %q accepted as %q", unsafe, name)
		}
	}
}

func TestLargeTextAssetIsNotEditableOnline(t *testing.T) {
	t.Setenv("STORAGE", t.TempDir())
	db := testWebDB(t)
	allowGuest(t, db)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	e := echo.New()
	Register(e, db)
	adminCookies := databaseSession(t, db, admin.ID)

	assets := uploadAssetForTest(t, e, "/api/problems/1000/assets/files", adminCookies, "data", "big.txt", strings.Repeat("x", maxEditableAssetBytes+1))
	if len(assets.Data) != 1 {
		t.Fatalf("expected uploaded asset, got %+v", assets)
	}
	if assets.Data[0].Editable {
		t.Fatalf("large text asset should not be editable: %+v", assets.Data[0])
	}

	target := "/api/problems/1000/assets/files/content?key=" + url.QueryEscape(assets.Data[0].Key)
	res := requestWithCookies(e, http.MethodGet, target, adminCookies, nil)
	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large asset content got %d body=%s", res.Code, res.Body.String())
	}
}

func TestAssetContentUpdateRejectsLargeBody(t *testing.T) {
	t.Setenv("STORAGE", t.TempDir())
	db := testWebDB(t)
	allowGuest(t, db)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
	}
	e := echo.New()
	Register(e, db)
	adminCookies := databaseSession(t, db, admin.ID)
	assets := uploadAssetForTest(t, e, "/api/problems/1000/assets/files", adminCookies, "judge", "main.cc", "int main(){}")
	if len(assets.Judge) != 1 || !assets.Judge[0].Editable {
		t.Fatalf("small judge asset should be editable: %+v", assets.Judge)
	}

	body := `{"key":"` + assets.Judge[0].Key + `","content":"` + strings.Repeat("x", maxEditableAssetBytes+1) + `"}`
	res := requestJSONWithCookies(e, http.MethodPatch, "/api/problems/1000/assets/files/content", adminCookies, body)
	if res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large asset update got %d body=%s", res.Code, res.Body.String())
	}
}

func uploadAssetForTest(t *testing.T, e *echo.Echo, target string, cookies []*http.Cookie, section string, name string, content string) ProblemAssets {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("section", section); err != nil {
		t.Fatalf("write section failed: %v", err)
	}
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatalf("create form file failed: %v", err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatalf("write asset failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	addCSRFHeader(req, cookies)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("upload asset got %d body=%s", res.Code, res.Body.String())
	}
	return decodeJSON[ProblemAssets](t, res)
}

func uploadImageForTest(t *testing.T, e *echo.Echo, target string, cookies []*http.Cookie, name string, content []byte) UploadResult {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatalf("create form file failed: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write image failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart failed: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	addCSRFHeader(req, cookies)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("upload image got %d body=%s", res.Code, res.Body.String())
	}
	return decodeJSON[UploadResult](t, res)
}

func requestOK(t *testing.T, e *echo.Echo, method string, target string, role string) *httptest.ResponseRecorder {
	t.Helper()
	res := request(e, method, target, role, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("%s %s as %s got %d body=%s", method, target, role, res.Code, res.Body.String())
	}
	return res
}

func request(e *echo.Echo, method string, target string, role string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, body)
	if role != "" {
		req.Header.Set("X-DOJ-Role", role)
	}
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func requestJSON(e *echo.Echo, method string, target string, role string, body string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if role != "" {
		req.Header.Set("X-DOJ-Role", role)
	}
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func requestJSONWithCookies(e *echo.Echo, method string, target string, cookies []*http.Cookie, body string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	addCSRFHeader(req, cookies)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func requestWithCookies(e *echo.Echo, method string, target string, cookies []*http.Cookie, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, body)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	addCSRFHeader(req, cookies)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func requestWithCookiesAndReferer(e *echo.Echo, method string, target string, cookies []*http.Cookie, referer string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	req.Header.Set("Referer", referer)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	addCSRFHeader(req, cookies)
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	return res
}

func addCSRFHeader(req *http.Request, cookies []*http.Cookie) {
	switch req.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return
	}
	for _, cookie := range cookies {
		if cookie.Name == utils.CSRFCookie {
			req.Header.Set(utils.CSRFHeader, cookie.Value)
			return
		}
	}
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func testWebDB(t *testing.T) *gorm.DB {
	t.Helper()
	startRedis(t)
	utils.ResetCacheForTest()
	t.Cleanup(utils.ResetCacheForTest)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "web.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := models.AutoMigrate(db); err != nil {
		t.Fatalf("migrate sqlite db: %v", err)
	}
	if err := models.EnsureDefaultLanguage(db); err != nil {
		t.Fatalf("seed language: %v", err)
	}
	return db
}

func startRedis(t *testing.T) {
	t.Helper()
	server := miniredis.RunT(t)
	t.Setenv("REDIS", "redis://"+server.Addr()+"/0")
}

func databaseSession(t *testing.T, db *gorm.DB, userID uint) []*http.Cookie {
	t.Helper()
	_ = db
	e := echo.New()
	res := httptest.NewRecorder()
	ctx := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), res)
	if err := utils.CreateUserSession(ctx, userID, time.Now()); err != nil {
		t.Fatalf("create session: %v", err)
	}
	return res.Result().Cookies()
}

func allowGuest(t *testing.T, db *gorm.DB) {
	t.Helper()
	settings := adminsvc.AdminSettings{
		SiteName:                "DOJ",
		AllowRegistration:       false,
		AllowGuestAccess:        true,
		DefaultSubmissionPublic: false,
		Notice:                  "",
	}
	if err := adminsvc.SaveSettings(db, settings); err != nil {
		t.Fatalf("enable guest access: %v", err)
	}
}

func decodeJSON[T any](t *testing.T, res *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(res.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode response failed: %v body=%s", err, res.Body.String())
	}
	return value
}

func decodePageItems[T any](t *testing.T, res *httptest.ResponseRecorder) []T {
	t.Helper()
	return decodeJSON[PageResult[T]](t, res).Items
}

func hasProblem(items []ProblemDTO, id uint) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func hasSolvedProblem(items []SolvedProblem, id uint) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func problemByID(items []ProblemDTO, id uint) (ProblemDTO, bool) {
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return ProblemDTO{}, false
}

func hasSubmissionProblem(items []SubmissionDTO, id uint) bool {
	for _, item := range items {
		if item.ProblemID == id {
			return true
		}
	}
	return false
}

func hasActivityProblem(items []UserActivityDTO, id uint) bool {
	for _, item := range items {
		if item.ProblemID == id {
			return true
		}
	}
	return false
}

func hasActivity(items []UserActivityDTO, kind string, title string) bool {
	for _, item := range items {
		if item.Type == kind && item.Title == title {
			return true
		}
	}
	return false
}

func activityBySubmission(items []UserActivityDTO, id uint) (UserActivityDTO, bool) {
	for _, item := range items {
		if item.Type == "submission" && item.ID == id {
			return item, true
		}
	}
	return UserActivityDTO{}, false
}

func hasSubmission(items []SubmissionDTO, id uint) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func userInRank(items []RankUserDTO, user string) bool {
	_, ok := rankByUser(items, user)
	return ok
}

func rankByUser(items []RankUserDTO, user string) (RankUserDTO, bool) {
	for _, item := range items {
		if item.User == user {
			return item, true
		}
	}
	return RankUserDTO{}, false
}

func rankProblemByID(items []RankProblemDTO, id uint) (RankProblemDTO, bool) {
	for _, item := range items {
		if item.ProblemID == id {
			return item, true
		}
	}
	return RankProblemDTO{}, false
}

func countForDate(items []HeatCell, date string) int {
	for _, item := range items {
		if item.Date == date {
			return item.Count
		}
	}
	return 0
}

func nonzeroHeatmapDays(items []HeatCell) int {
	count := 0
	for _, item := range items {
		if item.Count > 0 {
			count++
		}
	}
	return count
}

func zipHasFile(reader *zip.Reader, name string) bool {
	for _, file := range reader.File {
		if file.Name == name {
			return true
		}
	}
	return false
}

func TestJudgeTemplateUsesDockerfileCMDAndInteractorArgs(t *testing.T) {
	files := judgeTemplateFiles()
	dockerfile := files["Dockerfile"]
	main := files["main.cc"]
	if !strings.Contains(dockerfile, `FROM gcc`) || strings.Contains(dockerfile, `gcc:`) || !strings.Contains(dockerfile, `g++ main.cc -o main`) || !strings.Contains(dockerfile, `CMD ["/src/main"]`) {
		t.Fatalf("Dockerfile template must build and expose the same CMD path:\n%s", dockerfile)
	}
	for _, want := range []string{"argv[1]", "argv[3]", "argv[4]", "thread feeder", "fclose(stdout)", "0 = AC", "1 = WA", "3 = checker/interactor error"} {
		if !strings.Contains(main, want) {
			t.Fatalf("main.cc template missing %s:\n%s", want, main)
		}
	}
}

func tinyPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x63, 0xf8, 0xff, 0xff, 0x3f,
		0x00, 0x05, 0xfe, 0x02, 0xfe, 0xa7, 0x35, 0x81,
		0x84, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}
}
