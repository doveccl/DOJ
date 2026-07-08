package web

import (
	"net/http"
	"testing"

	"github.com/doveccl/doj/models"
	adminsvc "github.com/doveccl/doj/server/admin"
	"github.com/doveccl/doj/server/web/contract"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

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
	registered := decodeJSON[contract.Me](t, registerRes)
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
	loggedIn := decodeJSON[contract.Me](t, loginRes)
	if loggedIn.Name != "Alice_One" {
		t.Fatalf("login should return stored username case: %+v", loggedIn)
	}

	profile := decodeJSON[contract.UserProfile](t, requestOK(t, e, http.MethodGet, "/api/users/alice_one", ""))
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
	submissions := decodePageItems[contract.SubmissionListItem](t, submissionRes)
	if len(submissions) != 1 || submissions[0].User != "Alice_One" {
		t.Fatalf("submission user filter should be case-insensitive: %+v", submissions)
	}
	rawSubmissions := decodeJSON[contract.Page[map[string]any]](t, submissionRes).Items
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
	avatar := decodeJSON[contract.Me](t, avatarRes)
	if avatar.Mail != "student@example.com" || avatar.Bio != "old bio" || avatar.Avatar != "/new.png" {
		t.Fatalf("avatar patch should preserve mail and bio: %+v", avatar)
	}

	mailRes := requestJSONWithCookies(e, http.MethodPatch, "/api/me", cookies, `{"mail":"Next@Example.com"}`)
	if mailRes.Code != http.StatusOK {
		t.Fatalf("mail patch got %d body=%s", mailRes.Code, mailRes.Body.String())
	}
	mail := decodeJSON[contract.Me](t, mailRes)
	if mail.Mail != "next@example.com" || mail.Bio != "old bio" || mail.Avatar != "/new.png" {
		t.Fatalf("mail patch should preserve bio and avatar: %+v", mail)
	}
	clearRes := requestJSONWithCookies(e, http.MethodPatch, "/api/me", cookies, `{"bio":"","avatar":""}`)
	if clearRes.Code != http.StatusOK {
		t.Fatalf("clear profile fields got %d body=%s", clearRes.Code, clearRes.Body.String())
	}
	cleared := decodeJSON[contract.Me](t, clearRes)
	if cleared.Mail != "next@example.com" || cleared.Bio != "" || cleared.Avatar != "" {
		t.Fatalf("empty string patch should clear provided profile fields only: %+v", cleared)
	}
	emptyRes := requestJSONWithCookies(e, http.MethodPatch, "/api/me", cookies, `{}`)
	if emptyRes.Code != http.StatusBadRequest {
		t.Fatalf("empty profile patch got %d body=%s", emptyRes.Code, emptyRes.Body.String())
	}
}
