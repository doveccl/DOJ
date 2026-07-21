package public

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

func TestMentionKeysUseExplicitUsernameTokens(t *testing.T) {
	got := mentionKeys("@Alice @test-user a@example.com @@double `@Code_User` @ab")
	for _, key := range []string{"alice", "test-user", "code_user"} {
		if !got[key] {
			t.Fatalf("missing mention %q: %+v", key, got)
		}
	}
	for _, key := range []string{"example", "double", "ab"} {
		if got[key] {
			t.Fatalf("unexpected mention %q: %+v", key, got)
		}
	}
}

func TestDiscussionReplyNotificationsArePrivateDeduplicatedAndReadable(t *testing.T) {
	db := testWebDB(t)
	users := []models.User{
		{Name: "owner", Mail: "owner@example.com", Auth: "hash"},
		{Name: "actor", Mail: "actor@example.com", Auth: "hash"},
		{Name: "mentioned", Mail: "mentioned@example.com", Auth: "hash"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	owner, actor, mentioned := users[0], users[1], users[2]
	discussion := models.Discussion{Title: "Post", Content: "Body", UserID: owner.ID, Tags: datatypes.JSON([]byte(`[]`))}
	if err := db.Create(&discussion).Error; err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	Register(e, db)
	actorCookies := databaseSession(t, db, actor.ID)
	ownerCookies := databaseSession(t, db, owner.ID)
	mentionedCookies := databaseSession(t, db, mentioned.ID)
	commentsPath := "/api/discussion/" + strconv.FormatUint(uint64(discussion.ID), 10) + "/comments"
	res := requestJSONWithCookies(e, http.MethodPost, commentsPath, actorCookies, `{"content":"hello @OWNER, @mentioned and @actor"}`)
	if res.Code != http.StatusCreated {
		t.Fatalf("create comment got %d body=%s", res.Code, res.Body.String())
	}
	comment := decodeJSON[contract.Comment](t, res)

	ownerPage := decodeJSON[contract.NotificationPage](t, requestWithCookies(e, http.MethodGet, "/api/notifications", ownerCookies, nil))
	if ownerPage.Total != 1 || ownerPage.Unread != 1 || ownerPage.Items[0].Kind != "mention" || ownerPage.Items[0].CommentID == nil || *ownerPage.Items[0].CommentID != comment.ID {
		t.Fatalf("owner should receive one mention, not a duplicate reply: %+v", ownerPage)
	}
	mentionedPage := decodeJSON[contract.NotificationPage](t, requestWithCookies(e, http.MethodGet, "/api/notifications", mentionedCookies, nil))
	if mentionedPage.Total != 1 || mentionedPage.Items[0].Kind != "mention" || mentionedPage.Items[0].Actor != actor.Name || mentionedPage.Items[0].DiscussionTitle != discussion.Title {
		t.Fatalf("mentioned user notification mismatch: %+v", mentionedPage)
	}
	actorPage := decodeJSON[contract.NotificationPage](t, requestWithCookies(e, http.MethodGet, "/api/notifications", actorCookies, nil))
	if actorPage.Total != 0 {
		t.Fatalf("actor should not notify self: %+v", actorPage)
	}

	readPath := "/api/notifications/" + strconv.FormatUint(uint64(ownerPage.Items[0].ID), 10) + "/read"
	if got := requestWithCookies(e, http.MethodPost, readPath, actorCookies, nil); got.Code != http.StatusNoContent {
		t.Fatalf("other user read notification got %d body=%s", got.Code, got.Body.String())
	}
	ownerPage = decodeJSON[contract.NotificationPage](t, requestWithCookies(e, http.MethodGet, "/api/notifications", ownerCookies, nil))
	if ownerPage.Unread != 1 || ownerPage.Items[0].Read {
		t.Fatalf("other user changed notification: %+v", ownerPage)
	}
	if got := requestWithCookies(e, http.MethodPost, readPath, ownerCookies, nil); got.Code != http.StatusNoContent {
		t.Fatalf("read notification got %d body=%s", got.Code, got.Body.String())
	}
	ownerPage = decodeJSON[contract.NotificationPage](t, requestWithCookies(e, http.MethodGet, "/api/notifications", ownerCookies, nil))
	if ownerPage.Unread != 0 || !ownerPage.Items[0].Read {
		t.Fatalf("notification was not marked read: %+v", ownerPage)
	}

	if got := requestWithCookies(e, http.MethodDelete, commentsPath+"/"+strconv.FormatUint(uint64(comment.ID), 10), actorCookies, nil); got.Code != http.StatusNoContent {
		t.Fatalf("delete comment got %d body=%s", got.Code, got.Body.String())
	}
	mentionedPage = decodeJSON[contract.NotificationPage](t, requestWithCookies(e, http.MethodGet, "/api/notifications", mentionedCookies, nil))
	if mentionedPage.Total != 0 {
		t.Fatalf("deleted comment notifications remained: %+v", mentionedPage)
	}
}

func TestMentionLimitRejectsBeforeUserLookup(t *testing.T) {
	db := testWebDB(t)
	users := []models.User{
		{Name: "owner", Mail: "owner@example.com", Auth: "hash"},
		{Name: "actor", Mail: "actor@example.com", Auth: "hash"},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	owner, actor := users[0], users[1]
	mentions := make([]string, 0, maxMentions+1)
	for index := 0; index <= maxMentions; index++ {
		mentions = append(mentions, fmt.Sprintf("@missing%02d", index))
	}
	discussion := models.Discussion{Title: "Post", Content: "Body", UserID: owner.ID, Tags: datatypes.JSON([]byte(`[]`))}
	if err := db.Create(&discussion).Error; err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	Register(e, db)
	path := "/api/discussion/" + strconv.FormatUint(uint64(discussion.ID), 10) + "/comments"
	body := `{"content":"` + strings.Join(mentions, " ") + `"}`
	res := requestJSONWithCookies(e, http.MethodPost, path, databaseSession(t, db, actor.ID), body)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("too many mentions got %d body=%s", res.Code, res.Body.String())
	}
	var count int64
	if err := db.Model(&models.Comment{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("rejected comment was not rolled back: count=%d err=%v", count, err)
	}
}

func TestDiscussionCommentTargetSelectsContainingPage(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	user := models.User{Name: "owner", Mail: "owner@example.com", Auth: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	discussion := models.Discussion{Title: "Post", Content: "Body", UserID: user.ID, Tags: datatypes.JSON([]byte(`[]`))}
	if err := db.Create(&discussion).Error; err != nil {
		t.Fatal(err)
	}
	comments := make([]models.Comment, 25)
	for index := range comments {
		comments[index] = models.Comment{DiscussionID: discussion.ID, UserID: user.ID, Content: fmt.Sprintf("comment %d", index+1)}
	}
	if err := db.Create(&comments).Error; err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	Register(e, db)
	target := fmt.Sprintf("/api/discussion/%d?page=1&pageSize=10&comment=%d", discussion.ID, comments[22].ID)
	page := decodeJSON[contract.DiscussionDetail](t, requestWithCookies(e, http.MethodGet, target, nil, nil))
	if page.Comments.Page != 3 || len(page.Comments.Items) != 5 || page.Comments.Items[2].ID != comments[22].ID {
		t.Fatalf("target comment page mismatch: %+v", page.Comments)
	}
}
