package worker

import (
	"time"

	"github.com/doveccl/doj/contract/limits"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

type API struct {
	db        *gorm.DB
	leaseWait time.Duration
}

const (
	contextJudgerID     = "judgerID"
	contextJudgerLocal  = "judgerLocal"
	defaultLeaseSeconds = 60
	defaultLeaseWait    = 25 * time.Second
)

func Register(e *echo.Echo, db *gorm.DB) {
	if db == nil {
		panic("judger API requires a database")
	}
	api := &API{db: db, leaseWait: defaultLeaseWait}
	group := e.Group("/api/judger", api.auth)
	group.POST("/lease", api.lease, echomw.BodyLimit(limits.BodyShortText))
	group.GET("/tasks/:id/package", api.taskPackage)
	group.POST("/tasks/:id/heartbeat", api.heartbeat, echomw.BodyLimit(limits.BodyShortText))
	group.POST("/tasks/:id/result", api.result, echomw.BodyLimit(limits.BodyJudgerResult))
}
