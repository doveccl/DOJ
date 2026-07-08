package web

import (
	"net/http"
	"testing"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
)

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
