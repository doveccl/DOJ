package public

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/doveccl/doj/contract/limits"
	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
)

func TestContextDescriptionsAreSeparateAndAdminManaged(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	student := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&student).Error; err != nil {
		t.Fatal(err)
	}
	assignment := models.Assignment{Title: "Homework", EndAt: time.Now().Add(time.Hour)}
	contest := models.Contest{Title: "Round", Kind: "OI", StartAt: time.Now().Add(time.Hour), EndAt: time.Now().Add(2 * time.Hour)}
	if err := db.Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.AssignmentUser{AssignmentID: assignment.ID, UserID: student.ID}).Error; err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	Register(e, db)
	assignmentPath := "/api/assignments/" + strconv.FormatUint(uint64(assignment.ID), 10)
	contestPath := "/api/contests/" + strconv.FormatUint(uint64(contest.ID), 10)
	studentCookies := databaseSession(t, db, student.ID)
	adminCookies := databaseSession(t, db, admin.ID)
	if res := requestJSONWithCookies(e, http.MethodPatch, assignmentPath, studentCookies, `{}`); res.Code != http.StatusForbidden {
		t.Fatalf("student updated description: %d body=%s", res.Code, res.Body.String())
	}
	assignmentBody := func(description string) string {
		return fmt.Sprintf(`{"title":"Homework","description":%q,"endAt":%q,"problems":[],"users":[%d],"groups":[]}`,
			description, assignment.EndAt.Format(time.RFC3339), student.ID)
	}
	contestBody := func(description string) string {
		return fmt.Sprintf(`{"title":"Round","description":%q,"kind":"OI","startAt":%q,"endAt":%q,"freezeAt":"","problems":[]}`,
			description, contest.StartAt.Format(time.RFC3339), contest.EndAt.Format(time.RFC3339))
	}
	for path, body := range map[string]string{
		assignmentPath: assignmentBody("# Assignment rules"),
		contestPath:    contestBody("# Contest rules"),
	} {
		if res := requestJSONWithCookies(e, http.MethodPatch, path, adminCookies, body); res.Code != http.StatusOK {
			t.Fatalf("update %s: %d body=%s", path, res.Code, res.Body.String())
		}
	}
	assignmentDetail := decodeJSON[contract.AssignmentDetail](t, requestWithCookies(e, http.MethodGet, assignmentPath, studentCookies, nil))
	if assignmentDetail.Description != "# Assignment rules" {
		t.Fatalf("assignment description = %q", assignmentDetail.Description)
	}
	contestDetail := decodeJSON[contract.ContestDetail](t, requestWithCookies(e, http.MethodGet, contestPath, nil, nil))
	if contestDetail.Description != "# Contest rules" {
		t.Fatalf("contest description = %q", contestDetail.Description)
	}
	tooLarge := contestBody(strings.Repeat("x", limits.MaxMarkdownBytes+1))
	if res := requestJSONWithCookies(e, http.MethodPatch, contestPath, adminCookies, tooLarge); res.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large description got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestJSONWithCookies(e, http.MethodPatch, assignmentPath, adminCookies, assignmentBody("")); res.Code != http.StatusOK {
		t.Fatalf("clear description: %d body=%s", res.Code, res.Body.String())
	}
	assignmentDetail = decodeJSON[contract.AssignmentDetail](t, requestWithCookies(e, http.MethodGet, assignmentPath, studentCookies, nil))
	if assignmentDetail.Description != "" {
		t.Fatalf("cleared assignment description = %q", assignmentDetail.Description)
	}
}
