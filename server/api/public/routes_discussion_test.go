package public

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

func TestDiscussionTagsDoNotImposeProblemVisibility(t *testing.T) {
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
		t.Fatalf("student created hidden discussion: %d %s", res.Code, res.Body.String())
	}
	res = requestJSONWithCookies(e, http.MethodPost, "/api/discussion", adminCookies, discussionBody)
	if res.Code != http.StatusCreated {
		t.Fatalf("admin create hidden discussion got %d body=%s", res.Code, res.Body.String())
	}
	res = requestJSONWithCookies(e, http.MethodPost, "/api/discussion", adminCookies, `{"title":"Missing problem","content":"secret","tags":["P9999"]}`)
	if res.Code != http.StatusCreated {
		t.Fatalf("admin linked a missing problem: %d %s", res.Code, res.Body.String())
	}
	res = requestJSONWithCookies(e, http.MethodPost, "/api/discussion", adminCookies, `{"title":"Ambiguous problem","content":"secret","tags":["P1000","P1001"]}`)
	if res.Code != http.StatusCreated {
		t.Fatalf("discussion linked multiple problems: %d %s", res.Code, res.Body.String())
	}
	res = requestJSONWithCookies(e, http.MethodPatch, "/api/discussion/"+strconv.FormatUint(uint64(discussion.ID), 10), adminCookies, `{"tags":["P9999"]}`)
	if res.Code != http.StatusOK {
		t.Fatalf("admin updated discussion to a missing problem: %d %s", res.Code, res.Body.String())
	}
	list := decodeJSON[contract.Page[contract.Discussion]](t, requestWithCookies(e, http.MethodGet, "/api/discussion", studentCookies, nil))
	if list.Total != 5 || len(list.Items) != 5 {
		t.Fatalf("tagged discussions should remain visible: %+v", list)
	}
	if res := requestWithCookies(e, http.MethodGet, "/api/discussion/"+strconv.FormatUint(uint64(discussion.ID), 10), studentCookies, nil); res.Code != http.StatusOK {
		t.Fatalf("tagged discussion detail got %d body=%s", res.Code, res.Body.String())
	}

	body := `{"content":"I should not see this"}`
	target := "/api/discussion/" + strconv.FormatUint(uint64(discussion.ID), 10) + "/comments"
	res = requestJSONWithCookies(e, http.MethodPost, target, studentCookies, body)
	if res.Code != http.StatusCreated {
		t.Fatalf("student commented on tagged discussion: %d %s", res.Code, res.Body.String())
	}

	res = requestJSONWithCookies(e, http.MethodPost, target, adminCookies, body)
	if res.Code != http.StatusCreated {
		t.Fatalf("admin comment on hidden discussion got %d body=%s", res.Code, res.Body.String())
	}
}

func TestDiscussionTagsRemainSoftAcrossReadSurfaces(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	users := []models.User{
		{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true},
		{Name: "student", Mail: "student@example.com", Auth: "hash"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	admin, student := users[0], users[1]
	problems := []models.Problem{
		{ID: 1000, Title: "Hidden", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256},
		{ID: 1001, Title: "Upcoming", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
		{ID: 1002, Title: "Visible", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256},
	}
	if err := db.Create(&problems).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	contest := models.Contest{Title: "Upcoming", Kind: "OI", StartAt: now.Add(time.Hour), EndAt: now.Add(2 * time.Hour)}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: problems[1].ID, Sort: "A"}).Error; err != nil {
		t.Fatal(err)
	}
	discussions := []models.Discussion{
		{Title: "General", Content: "public", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["general-secret"]`))},
		{Title: "Hidden topic", Content: "hidden", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["P1000","hidden-secret"]`))},
		{Title: "Upcoming topic", Content: "upcoming", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["P1001","upcoming-secret"]`))},
		{Title: "Visible topic", Content: "visible", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["P1002","visible-secret"]`))},
	}
	if err := db.Create(&discussions).Error; err != nil {
		t.Fatal(err)
	}
	hiddenComment := models.Comment{DiscussionID: discussions[1].ID, UserID: student.ID, Content: "old reply"}
	if err := db.Create(&hiddenComment).Error; err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	Register(e, db)
	studentCookies := databaseSession(t, db, student.ID)
	for _, viewer := range []struct {
		name    string
		cookies []*http.Cookie
	}{{name: "guest"}, {name: "student", cookies: studentCookies}} {
		list := decodeJSON[contract.Page[contract.Discussion]](t, requestWithCookies(e, http.MethodGet, "/api/discussion", viewer.cookies, nil))
		if list.Total != 4 || len(list.Items) != 4 || discussionByTitle(list.Items, "Hidden topic") == nil || discussionByTitle(list.Items, "Upcoming topic") == nil {
			t.Fatalf("%s should see soft-linked discussions: %+v", viewer.name, list)
		}
		search := decodeJSON[contract.Page[contract.Discussion]](t, requestWithCookies(e, http.MethodGet, "/api/discussion?q=hidden-secret", viewer.cookies, nil))
		if search.Total != 1 || len(search.Items) != 1 {
			t.Fatalf("%s discussion search lost tagged result: %+v", viewer.name, search)
		}
		visible := discussionByTitle(list.Items, "Visible topic")
		if visible == nil || !hasTag(visible.Tags, "P1002") {
			t.Fatalf("%s visible discussion lost problem tag: %+v", viewer.name, list.Items)
		}
		for _, row := range discussions[1:3] {
			path := "/api/discussion/" + strconv.FormatUint(uint64(row.ID), 10)
			if res := requestWithCookies(e, http.MethodGet, path, viewer.cookies, nil); res.Code != http.StatusOK {
				t.Fatalf("%s read %q: %d %s", viewer.name, row.Title, res.Code, res.Body.String())
			}
		}
		tags := decodeJSON[[]string](t, requestWithCookies(e, http.MethodGet, "/api/tags?kind=discussion&q=secret", viewer.cookies, nil))
		if !hasTag(tags, "hidden-secret") || !hasTag(tags, "upcoming-secret") || !hasTag(tags, "general-secret") || !hasTag(tags, "visible-secret") {
			t.Fatalf("%s discussion tags should remain searchable: %+v", viewer.name, tags)
		}
		profile := decodeJSON[contract.UserProfile](t, requestWithCookies(e, http.MethodGet, "/api/users/admin", viewer.cookies, nil))
		if !hasActivity(profile.Activities, "discussion", "Hidden topic") || !hasActivity(profile.Activities, "discussion", "Upcoming topic") || !hasActivity(profile.Activities, "discussion", "General") || !hasActivity(profile.Activities, "discussion", "Visible topic") {
			t.Fatalf("%s profile should retain soft-linked discussions: %+v", viewer.name, profile.Activities)
		}
	}

	hiddenPath := "/api/discussion/" + strconv.FormatUint(uint64(discussions[1].ID), 10)
	if res := requestJSONWithCookies(e, http.MethodPost, hiddenPath+"/comments", studentCookies, `{"content":"new reply"}`); res.Code != http.StatusCreated {
		t.Fatalf("student commented on hidden discussion: %d %s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodDelete, hiddenPath+"/comments/"+strconv.FormatUint(uint64(hiddenComment.ID), 10), studentCookies, nil); res.Code != http.StatusNoContent {
		t.Fatalf("student deleted comment through hidden discussion: %d %s", res.Code, res.Body.String())
	}
	state := decodeJSON[[]contract.ProblemState](t, requestWithCookies(e, http.MethodGet, "/api/problem-state?ids=1000,1001,1002", studentCookies, nil))
	if len(state) != 3 || state[0].Discussions != nil || state[1].Discussions != nil || state[2].Discussions == nil || *state[2].Discussions != 1 {
		t.Fatalf("problem discussion counts leaked hidden context: %+v", state)
	}

	adminCookies := databaseSession(t, db, admin.ID)
	adminList := decodeJSON[contract.Page[contract.Discussion]](t, requestWithCookies(e, http.MethodGet, "/api/discussion", adminCookies, nil))
	if adminList.Total != 4 || len(adminList.Items) != 4 {
		t.Fatalf("admin should see valid hidden problem discussions: %+v", adminList)
	}
	if err := db.Delete(&problems[0]).Error; err != nil {
		t.Fatal(err)
	}
	if res := requestWithCookies(e, http.MethodGet, hiddenPath, adminCookies, nil); res.Code != http.StatusOK {
		t.Fatalf("soft-linked discussion disappeared with deleted problem: %d %s", res.Code, res.Body.String())
	}
	adminTags := decodeJSON[[]string](t, requestWithCookies(e, http.MethodGet, "/api/tags?kind=discussion&q=hidden-secret", adminCookies, nil))
	if !hasTag(adminTags, "hidden-secret") {
		t.Fatalf("soft discussion tag disappeared with deleted problem: %+v", adminTags)
	}
}

func TestContestDoesNotControlSoftLinkedDiscussionVisibility(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	student := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&[]models.User{student, admin}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("name = ?", student.Name).First(&student).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("name = ?", admin.Name).First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	problem := models.Problem{ID: 1000, Title: "Contest problem", Tags: datatypes.JSON([]byte(`[]`)), Visible: false, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	now := time.Now()
	contest := models.Contest{Title: "Round", Kind: "OI", StartAt: now.Add(-time.Hour), EndAt: now.Add(time.Hour)}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&contest).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ContestProblem{ContestID: contest.ID, ProblemID: problem.ID, Sort: "A"}).Error; err != nil {
		t.Fatal(err)
	}
	discussion := models.Discussion{Title: "Solution", Content: "secret", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["P1000"]`))}
	if err := db.Create(&discussion).Error; err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	Register(e, db)
	studentCookies := databaseSession(t, db, student.ID)
	adminCookies := databaseSession(t, db, admin.ID)
	path := "/api/discussion/" + strconv.FormatUint(uint64(discussion.ID), 10)
	if got := decodeJSON[contract.Page[contract.Discussion]](t, requestWithCookies(e, http.MethodGet, "/api/discussion", studentCookies, nil)); got.Total != 1 {
		t.Fatalf("soft-linked discussion missing from list: %+v", got)
	}
	if res := requestWithCookies(e, http.MethodGet, path, studentCookies, nil); res.Code != http.StatusOK {
		t.Fatalf("running contest discussion detail got %d", res.Code)
	}
	body := `{"title":"Spoiler","content":"secret","tags":["P1000"]}`
	if res := requestJSONWithCookies(e, http.MethodPost, "/api/discussion", studentCookies, body); res.Code != http.StatusCreated {
		t.Fatalf("contestant created running contest discussion: %d %s", res.Code, res.Body.String())
	}
	if res := requestWithCookies(e, http.MethodGet, path, adminCookies, nil); res.Code != http.StatusOK {
		t.Fatalf("admin discussion detail got %d body=%s", res.Code, res.Body.String())
	}
	contest.EndAt = now.Add(-time.Minute)
	if err := db.Save(&contest).Error; err != nil {
		t.Fatal(err)
	}
	if res := requestWithCookies(e, http.MethodGet, path, studentCookies, nil); res.Code != http.StatusOK {
		t.Fatalf("ended contest discussion stayed hidden: %d body=%s", res.Code, res.Body.String())
	}
	if res := requestJSONWithCookies(e, http.MethodPost, "/api/discussion", studentCookies, body); res.Code != http.StatusCreated {
		t.Fatalf("ended contest discussion create got %d body=%s", res.Code, res.Body.String())
	}
}

func discussionByTitle(items []contract.Discussion, title string) *contract.Discussion {
	for index := range items {
		if items[index].Title == title {
			return &items[index]
		}
	}
	return nil
}

func TestDatabaseDiscussionAuthorsUseProfiles(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Avatar: "/admin.png", Auth: "hash", Admin: true}
	student := models.User{Name: "student", Mail: "student@example.com", Avatar: "/student.png", Auth: "hash"}
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

	list := decodePageItems[contract.Discussion](t, requestOK(t, e, http.MethodGet, "/api/discussion", ""))
	if len(list) != 1 || list[0].Author != "admin" || list[0].Avatar != admin.Avatar || list[0].Replies != 1 {
		t.Fatalf("discussion list should include author and reply count: %+v", list)
	}
	listRes := requestOK(t, e, http.MethodGet, "/api/discussion", "")
	rawList := decodeJSON[contract.Page[map[string]any]](t, listRes).Items
	if _, ok := rawList[0]["content"]; ok {
		t.Fatalf("discussion list should not include content: %+v", rawList[0])
	}

	target := "/api/discussion/" + strconv.FormatUint(uint64(discussion.ID), 10)
	detail := decodeJSON[contract.DiscussionDetail](t, requestOK(t, e, http.MethodGet, target, ""))
	if detail.Discussion.Author != "admin" || detail.Discussion.Avatar != admin.Avatar || len(detail.Comments.Items) != 1 || detail.Comments.Items[0].Author != "student" || detail.Comments.Items[0].Avatar != student.Avatar {
		t.Fatalf("discussion detail authors should be usernames: %+v", detail)
	}

	updated := requestJSONWithCookies(e, http.MethodPatch, target, databaseSession(t, db, admin.ID), `{"pinned":true}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("update discussion got %d body=%s", updated.Code, updated.Body.String())
	}
	updatedID := decodeJSON[contract.CreatedID](t, updated)
	if updatedID.ID != discussion.ID {
		t.Fatalf("partial discussion update should return updated id: %+v", updatedID)
	}
	detail = decodeJSON[contract.DiscussionDetail](t, requestOK(t, e, http.MethodGet, target, ""))
	if !detail.Discussion.Pinned || detail.Discussion.Locked || detail.Discussion.Title != "Named discussion" || detail.Discussion.Replies != 1 || detail.Content != "body" || len(detail.Discussion.Tags) != 1 || detail.Discussion.Tags[0] != "general" {
		t.Fatalf("partial discussion update should preserve content and tags: %+v", detail)
	}
	updated = requestJSONWithCookies(e, http.MethodPatch, target, databaseSession(t, db, admin.ID), `{"pinned":false,"locked":true}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("false/true discussion patch got %d body=%s", updated.Code, updated.Body.String())
	}
	updatedID = decodeJSON[contract.CreatedID](t, updated)
	if updatedID.ID != discussion.ID {
		t.Fatalf("false/true discussion patch should return updated id: %+v", updatedID)
	}
	detail = decodeJSON[contract.DiscussionDetail](t, requestOK(t, e, http.MethodGet, target, ""))
	if detail.Discussion.Pinned || !detail.Discussion.Locked || detail.Discussion.Title != "Named discussion" || detail.Discussion.Replies != 1 {
		t.Fatalf("discussion patch should apply false/true flags and preserve unrelated fields: %+v", detail)
	}
	updated = requestJSONWithCookies(e, http.MethodPatch, target, databaseSession(t, db, admin.ID), `{"locked":false,"tags":[]}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("false/empty tags discussion patch got %d body=%s", updated.Code, updated.Body.String())
	}
	updatedID = decodeJSON[contract.CreatedID](t, updated)
	if updatedID.ID != discussion.ID {
		t.Fatalf("false/empty tags discussion patch should return updated id: %+v", updatedID)
	}
	detail = decodeJSON[contract.DiscussionDetail](t, requestOK(t, e, http.MethodGet, target, ""))
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

	path := "/api/discussion/" + strconv.FormatUint(uint64(discussion.ID), 10)
	detail := decodeJSON[contract.DiscussionDetail](t, requestOK(t, e, http.MethodGet, path+"?page=1&pageSize=2", ""))
	if detail.Discussion.Replies != 1 || detail.Comments.Total != 3 || detail.Comments.Page != 1 || detail.Comments.PageSize != 2 || len(detail.Comments.Items) != 2 {
		t.Fatalf("deleted comments should stay as floor slots but not active replies: %+v", detail)
	}
	if !detail.Comments.Items[0].Deleted || detail.Comments.Items[0].Content != "" || !detail.Comments.Items[1].Deleted || detail.Comments.Items[1].Content != "" {
		t.Fatalf("comment tombstones should preserve order and hide content: %+v", detail.Comments.Items)
	}
	detail = decodeJSON[contract.DiscussionDetail](t, requestOK(t, e, http.MethodGet, path+"?page=2&pageSize=2", ""))
	if detail.Comments.Total != 3 || len(detail.Comments.Items) != 1 || detail.Comments.Items[0].Deleted || detail.Comments.Items[0].Content != "third" {
		t.Fatalf("discussion comments should be fetched one server page at a time: %+v", detail.Comments)
	}
}

func TestDiscussionListSearchesTitleContentAndTags(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{ID: 1289, Title: "Tagged", Tags: datatypes.JSON([]byte(`[]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
	if err := db.Create(&problem).Error; err != nil {
		t.Fatalf("create problem: %v", err)
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
		got := decodePageItems[contract.Discussion](t, requestOK(t, e, http.MethodGet, "/api/discussion?q="+url.QueryEscape(item.q), ""))
		if len(got) != 1 || got[0].Title != item.want {
			t.Fatalf("search %q got %+v, want %q", item.q, got, item.want)
		}
	}
}
