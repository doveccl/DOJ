package web

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"

	contract "github.com/doveccl/doj/common/web"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

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

	list := decodePageItems[contract.Discussion](t, requestOK(t, e, http.MethodGet, "/api/discussion", ""))
	if len(list) != 1 || list[0].Author != "admin" || list[0].Replies != 1 {
		t.Fatalf("discussion list should include author and reply count: %+v", list)
	}
	listRes := requestOK(t, e, http.MethodGet, "/api/discussion", "")
	rawList := decodeJSON[contract.Page[map[string]any]](t, listRes).Items
	if _, ok := rawList[0]["content"]; ok {
		t.Fatalf("discussion list should not include content: %+v", rawList[0])
	}

	target := "/api/discussion/" + strconv.FormatUint(uint64(discussion.ID), 10)
	detail := decodeJSON[contract.DiscussionDetail](t, requestOK(t, e, http.MethodGet, target, ""))
	if detail.Discussion.Author != "admin" || len(detail.Comments) != 1 || detail.Comments[0].Author != "student" {
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

	detail := decodeJSON[contract.DiscussionDetail](t, requestOK(t, e, http.MethodGet, "/api/discussion/"+strconv.FormatUint(uint64(discussion.ID), 10), ""))
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
		got := decodePageItems[contract.Discussion](t, requestOK(t, e, http.MethodGet, "/api/discussion?q="+url.QueryEscape(item.q), ""))
		if len(got) != 1 || got[0].Title != item.want {
			t.Fatalf("search %q got %+v, want %q", item.q, got, item.want)
		}
	}
}
