package public

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

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

	guestRank := decodeJSON[contract.Page[contract.RankUser]](t, requestOK(t, e, http.MethodGet, "/api/rank", ""))
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
	aliceProfile := decodeJSON[contract.UserProfile](t, requestOK(t, e, http.MethodGet, "/api/users/alice", ""))
	if aliceProfile.User.AC != 2 || aliceProfile.User.Submit != 4 {
		t.Fatalf("alice profile should not expose running OI AC: %+v", aliceProfile.User)
	}

	adminRank := decodeJSON[contract.Page[contract.RankUser]](t, requestWithCookies(e, http.MethodGet, "/api/rank", databaseSession(t, db, admin.ID), nil))
	aliceAdmin, ok := rankByUser(adminRank.Items, "alice")
	if !ok || aliceAdmin.AC != 3 || aliceAdmin.Submit != 4 {
		t.Fatalf("alice admin stats should include hidden problem: %+v", adminRank)
	}
	adminProfile := decodeJSON[contract.UserProfile](t, requestWithCookies(e, http.MethodGet, "/api/users/alice", databaseSession(t, db, admin.ID), nil))
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

	first := decodeJSON[contract.Page[contract.RankUser]](t, requestOK(t, e, http.MethodGet, "/api/rank?page=1&pageSize=20", ""))
	if first.Total != 105 || len(first.Items) != 20 || first.Items[0].User != "u104" || first.Items[0].Rank != 1 {
		t.Fatalf("rank should sort all users before paging: %+v", first)
	}
	last := decodeJSON[contract.Page[contract.RankUser]](t, requestOK(t, e, http.MethodGet, "/api/rank?page=6&pageSize=20", ""))
	if len(last.Items) != 5 || last.Items[0].Rank != 101 {
		t.Fatalf("last rank page = %+v", last)
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
	profile := decodeJSON[contract.UserProfile](t, requestOK(t, e, http.MethodGet, "/api/users/alice?solvedPage=2&solvedPageSize=5", ""))
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
