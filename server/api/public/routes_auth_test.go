package public

import (
	"net/http"
	"net/http/httptest"
	"testing"

	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/settings"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
)

func TestDefaultAdminPasswordWarningIsDerivedFromHash(t *testing.T) {
	db := testWebDB(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: string(hash), Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	Register(e, db)
	cookies := databaseSession(t, db, admin.ID)
	me := decodeJSON[contract.Me](t, requestWithCookies(e, http.MethodGet, "/api/me", cookies, nil))
	if !me.MustChangePassword {
		t.Fatal("default admin password warning was not derived")
	}
	res := requestJSONWithCookies(e, http.MethodPatch, "/api/me/password", cookies, `{"oldPassword":"admin","newPassword":"new-admin-password"}`)
	if res.Code != http.StatusNoContent {
		t.Fatalf("change password got %d body=%s", res.Code, res.Body.String())
	}
	login := requestJSON(e, http.MethodPost, "/api/auth/login", "", `{"name":"admin","password":"new-admin-password"}`)
	if login.Code != http.StatusOK {
		t.Fatalf("new password login got %d body=%s", login.Code, login.Body.String())
	}
	if next := decodeJSON[contract.Me](t, login); next.MustChangePassword {
		t.Fatal("default password warning remained after password change")
	}
}

func TestPasswordChangeInvalidatesAllSessions(t *testing.T) {
	db := testWebDB(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := models.User{Name: "student", Mail: "student@example.com", Auth: string(hash)}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	e := echo.New()
	Register(e, db)
	first := databaseSession(t, db, user.ID)
	second := databaseSession(t, db, user.ID)

	res := requestJSONWithCookies(e, http.MethodPatch, "/api/me/password", first, `{"oldPassword":"old-password","newPassword":"new-password"}`)
	if res.Code != http.StatusNoContent {
		t.Fatalf("change password got %d body=%s", res.Code, res.Body.String())
	}
	for index, cookies := range [][]*http.Cookie{first, second} {
		me := decodeJSON[contract.Me](t, requestWithCookies(e, http.MethodGet, "/api/me", cookies, nil))
		if me.ID != 0 {
			t.Fatalf("session %d survived password change: %+v", index+1, me)
		}
	}
	if res := requestJSON(e, http.MethodPost, "/api/auth/login", "", `{"name":"student","password":"old-password"}`); res.Code != http.StatusUnauthorized {
		t.Fatalf("old password login got %d body=%s", res.Code, res.Body.String())
	}
	if res := requestJSON(e, http.MethodPost, "/api/auth/login", "", `{"name":"student","password":"new-password"}`); res.Code != http.StatusOK {
		t.Fatalf("new password login got %d body=%s", res.Code, res.Body.String())
	}
}

func TestRequestViewerIsLoadedOncePerRequest(t *testing.T) {
	db := testWebDB(t)
	user := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range databaseSession(t, db, user.ID) {
		req.AddCookie(cookie)
	}
	c := echo.New().NewContext(req, httptest.NewRecorder())
	api := &API{db: db}
	if got, err := api.currentUser(c); err != nil || got.ID != user.ID {
		t.Fatalf("first viewer = %+v, %v", got, err)
	}
	if err := db.Delete(&user).Error; err != nil {
		t.Fatal(err)
	}
	if got, err := api.currentUser(c); err != nil || got.ID != user.ID {
		t.Fatalf("cached viewer = %+v, %v", got, err)
	}
}

func TestUsernamePreservesCaseAndMatchesCaseInsensitively(t *testing.T) {
	db := testWebDB(t)
	if err := settings.Save(db, settings.Settings{
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
	for _, key := range []string{"message", "public"} {
		if _, ok := rawSubmissions[0][key]; ok {
			t.Fatalf("submission list should not include detail field %q: %+v", key, rawSubmissions[0])
		}
	}
	if err := db.Delete(&user).Error; err != nil {
		t.Fatalf("delete user: %v", err)
	}
	reserved := requestJSON(e, http.MethodPost, "/api/auth/register", "", `{"name":"ALICE_ONE","mail":"new@example.com","password":"password123"}`)
	if reserved.Code != http.StatusConflict {
		t.Fatalf("deleted identity was reused: %d body=%s", reserved.Code, reserved.Body.String())
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
