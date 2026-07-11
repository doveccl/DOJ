package public

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
)

func TestEventConnectionsAreBoundedAndReleased(t *testing.T) {
	db := testWebDB(t)
	user := models.User{Name: "student", Mail: "student@example.com", Auth: "hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	api := &API{db: db}
	e := echo.New()
	contextFor := func(address string) echo.Context {
		req := httptest.NewRequest("GET", "/api/events", nil)
		req.RemoteAddr = address + ":1234"
		return e.NewContext(req, httptest.NewRecorder())
	}

	releases := make([]func(), 0, maxEventConnections)
	for i := 0; i < maxGuestEventConnections; i++ {
		release, ok := api.acquireEventConnection(contextFor("192.0.2.1"))
		if !ok {
			t.Fatalf("connection %d for one address was rejected early", i)
		}
		releases = append(releases, release)
	}
	if _, ok := api.acquireEventConnection(contextFor("192.0.2.1")); ok {
		t.Fatal("guest per-address event connection limit was bypassed")
	}
	for _, release := range releases {
		release()
	}
	releases = releases[:0]

	cookies := databaseSession(t, db, user.ID)
	userContext := func() echo.Context {
		ctx := contextFor("192.0.2.2")
		for _, cookie := range cookies {
			ctx.Request().AddCookie(cookie)
		}
		return ctx
	}
	for i := 0; i < maxUserEventConnections; i++ {
		release, ok := api.acquireEventConnection(userContext())
		if !ok {
			t.Fatalf("connection %d for one user was rejected early", i)
		}
		releases = append(releases, release)
	}
	if _, ok := api.acquireEventConnection(userContext()); ok {
		t.Fatal("per-user event connection limit was bypassed")
	}
	for _, release := range releases {
		release()
	}
	releases = releases[:0]

	for i := len(releases); i < maxEventConnections; i++ {
		address := fmt.Sprintf("198.51.%d.%d", i/250, i%250+1)
		release, ok := api.acquireEventConnection(contextFor(address))
		if !ok {
			t.Fatalf("global connection %d was rejected early", i)
		}
		releases = append(releases, release)
	}
	if _, ok := api.acquireEventConnection(contextFor("203.0.113.1")); ok {
		t.Fatal("global event connection limit was bypassed")
	}

	releases[0]()
	if release, ok := api.acquireEventConnection(contextFor("203.0.113.1")); !ok {
		t.Fatal("released event connection capacity was not reusable")
	} else {
		release()
	}
	for _, release := range releases[1:] {
		release()
	}
}
