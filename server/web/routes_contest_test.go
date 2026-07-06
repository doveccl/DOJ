package web

import (
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
	"net/http"
	"strconv"
	"testing"
	"time"
)

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

func TestContestProblemVisibilityIsDerivedFromContestTime(t *testing.T) {
	db := testWebDB(t)
	allowGuest(t, db)
	admin := models.User{Name: "admin", Mail: "admin@example.com", Auth: "hash", Admin: true}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	problem := models.Problem{ID: 1000, Title: "Contest Only", Tags: datatypes.JSON([]byte(`["math"]`)), Visible: true, Mode: "default", TimeMS: 1000, MemoryMB: 256}
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
	if err := db.Create(&models.Discussion{Title: "Problem discussion", Content: "body", UserID: admin.ID, Tags: datatypes.JSON([]byte(`["P1000"]`))}).Error; err != nil {
		t.Fatalf("create discussion: %v", err)
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
	} else {
		got := decodeJSON[ProblemDTO](t, res)
		if len(got.Tags) != 0 {
			t.Fatalf("running contest problem detail should hide tags: %+v", got)
		}
	}
	state := decodeJSON[[]ProblemStateDTO](t, requestOK(t, e, http.MethodGet, "/api/problem-state?ids=1000", ""))
	if len(state) != 1 || state[0].AC != 0 || state[0].Submit != 0 || state[0].Discussions != nil {
		t.Fatalf("running contest problem state should hide stats and discussions: %+v", state)
	}
	contestDetail := decodeJSON[ContestDetail](t, requestOK(t, e, http.MethodGet, "/api/contests/"+strconv.FormatUint(uint64(contest.ID), 10), ""))
	if !hasProblem(contestDetail.Problems, problem.ID) {
		t.Fatalf("running contest detail should include linked problem: %+v", contestDetail.Problems)
	}
	if len(contestDetail.Problems[0].Tags) != 0 {
		t.Fatalf("running contest detail should hide problem tags: %+v", contestDetail.Problems[0])
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
	aliceRecords := decodeJSON[[]ProblemStateDTO](t, requestWithCookies(e, http.MethodGet, "/api/problem-state?contest="+strconv.FormatUint(uint64(contest.ID), 10)+"&ids=1000", databaseSession(t, db, alice.ID), nil))
	if len(aliceRecords) != 1 || aliceRecords[0].Status != "ac" || aliceRecords[0].Submission == nil || aliceRecords[0].Submission.ID != before.ID {
		t.Fatalf("ICPC problem status should link first AC, not later WA: %+v", aliceRecords)
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
	aliceList := decodeJSON[PageResult[SubmissionListItem]](t, requestWithCookies(e, http.MethodGet, "/api/submissions?contest="+strconv.FormatUint(uint64(contest.ID), 10), databaseSession(t, db, alice.ID), nil))
	if len(aliceList.Items) != 3 || aliceList.Items[0].ID != bobAfter.ID || aliceList.Items[0].Status != "pending" || aliceList.Items[1].ID != aliceAfter.ID || aliceList.Items[1].Status != "WA" {
		t.Fatalf("contest submission list should keep rows but hide only non-owner post-freeze results: %+v", aliceList.Items)
	}
	bobList := decodeJSON[PageResult[SubmissionListItem]](t, requestWithCookies(e, http.MethodGet, "/api/submissions?contest="+strconv.FormatUint(uint64(contest.ID), 10), databaseSession(t, db, bob.ID), nil))
	if len(bobList.Items) != 3 || bobList.Items[0].ID != bobAfter.ID || bobList.Items[0].Status != "AC" || bobList.Items[1].ID != aliceAfter.ID || bobList.Items[1].Status != "pending" {
		t.Fatalf("contest submission list should expose owner post-freeze result only to owner: %+v", bobList.Items)
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
	records := decodeJSON[[]ProblemStateDTO](t, requestWithCookies(e, http.MethodGet, "/api/problem-state?contest="+strconv.FormatUint(uint64(contest.ID), 10)+"&ids=1000,1001", databaseSession(t, db, alice.ID), nil))
	if len(records) != 2 || records[0].Status != "ac" || records[1].Status != "none" {
		t.Fatalf("contest problem state should use contest submissions only: %+v", records)
	}
}
