package web

import (
	"net/http"
	"strconv"
	"testing"
	"time"

	contract "github.com/doveccl/doj/common/web"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

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
	created := decodeJSON[contract.CreatedID](t, createRes)
	createdDetail := decodeJSON[contract.AssignmentDetail](t, requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), adminCookies, nil))
	if len(createdDetail.Assignment.Groups) != 1 || createdDetail.Assignment.Groups[0] != group.ID || len(createdDetail.Assignment.Users) != 0 {
		t.Fatalf("created assignment members not persisted: %+v", createdDetail.Assignment)
	}
	adminList := decodeJSON[contract.Page[map[string]any]](t, requestWithCookies(e, http.MethodGet, "/api/assignments", adminCookies, nil)).Items
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
	updated := decodeJSON[contract.CreatedID](t, updateRes)
	if updated.ID != created.ID {
		t.Fatalf("updated assignment should return id: %+v", updated)
	}
	updatedDetail := decodeJSON[contract.AssignmentDetail](t, requestWithCookies(e, http.MethodGet, "/api/assignments/"+strconv.FormatUint(uint64(created.ID), 10), adminCookies, nil))
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
