package public

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/doveccl/doj/contract/limits"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/cache"
	"github.com/doveccl/doj/server/settings"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

type API struct {
	db              *gorm.DB
	eventMu         sync.Mutex
	eventTotal      int
	eventByIdentity map[string]int
}

const (
	maxTitleRunes            = models.TitleMax
	homeListLimit            = 5
	userActivityLimit        = 10
	userSolvedPageSize       = 12
	maxEventConnections      = 512
	maxUserEventConnections  = 4
	maxGuestEventConnections = 16
)

func Register(e *echo.Echo, db *gorm.DB) {
	if db == nil {
		panic("web API requires a database")
	}
	api := &API{db: db}
	e.GET("/api/health", api.health)
	e.GET("/api/ready", api.ready)
	e.GET("/api/site", api.site)
	e.POST("/api/auth/login", api.login, api.rateLimit("auth", 20, time.Minute), echomw.BodyLimit(limits.BodyShortText))
	e.POST("/api/auth/register", api.register, api.rateLimit("auth", 20, time.Minute), echomw.BodyLimit(limits.BodyShortText))
	e.POST("/api/auth/logout", api.logout)
	e.GET("/api/me", api.me)
	e.PATCH("/api/me", api.updateMe, echomw.BodyLimit(limits.BodyShortText))
	e.PATCH("/api/me/password", api.updatePassword, echomw.BodyLimit(limits.BodyShortText))

	group := e.Group("/api", api.requireGuestAccess)
	group.GET("/events", api.events)
	group.GET("/home", api.home)
	group.GET("/languages", api.languages)
	group.PATCH("/home/notice", api.updateNotice, echomw.BodyLimit(limits.BodyMarkdown))
	group.POST("/uploads/images", api.uploadImage, api.rateLimit("upload", 20, time.Minute), echomw.BodyLimit(limits.BodyImage))
	group.GET("/rank", api.rank)

	group.GET("/users/:id/:year/:month/:day/*", api.userMedia)
	group.GET("/users", api.users)
	group.GET("/users/:name", api.user)

	group.GET("/problems", api.problems)
	group.POST("/problems", api.createProblem, echomw.BodyLimit(limits.BodyShortText))
	group.GET("/tags", api.tags)
	group.GET("/problem-state", api.problemState)
	group.GET("/problems/:id", api.problem)
	group.PATCH("/problems/:id", api.updateProblem, echomw.BodyLimit(limits.BodyMarkdown))
	group.PATCH("/problems/:id/visibility", api.updateProblemVisibility, echomw.BodyLimit(limits.BodyShortText))
	group.POST("/problems/:id/rejudge", api.rejudgeProblem)
	group.DELETE("/problems/:id", api.deleteProblem)
	group.GET("/problems/:id/package", api.problemPackage)
	group.POST("/problems/:id/assets/images", api.uploadProblemImage, api.rateLimit("problem-upload", 60, time.Minute), echomw.BodyLimit(limits.BodyImage))
	group.GET("/problems/:id/package/data/*", api.problemPrivateData)
	group.GET("/problems/:id/package/judge/*", api.problemPrivateJudge)
	group.POST("/problems/:id/package/files", api.uploadProblemPackage, api.rateLimit("problem-upload", 60, time.Minute), echomw.BodyLimit(limits.BodyProblemPackage))
	group.DELETE("/problems/:id/package/files", api.deleteProblemPackage)
	group.PATCH("/problems/:id/package/cases/score", api.updateProblemCaseScore, echomw.BodyLimit(limits.BodyShortText))
	group.GET("/problems/:id/assets/*", api.problemPublicAsset)

	group.GET("/assignments", api.assignments)
	group.POST("/assignments", api.createAssignment, echomw.BodyLimit(limits.BodyMarkdown))
	group.GET("/assignments/:id", api.assignment)
	group.PATCH("/assignments/:id", api.updateAssignment, echomw.BodyLimit(limits.BodyMarkdown))
	group.DELETE("/assignments/:id", api.deleteAssignment)

	group.GET("/contests", api.contests)
	group.POST("/contests", api.createContest, echomw.BodyLimit(limits.BodyMarkdown))
	group.GET("/contests/:id", api.contest)
	group.PATCH("/contests/:id", api.updateContest, echomw.BodyLimit(limits.BodyMarkdown))
	group.DELETE("/contests/:id", api.deleteContest)

	group.GET("/submissions", api.submissions)
	group.POST("/submissions", api.submit, api.rateLimit("submit", 30, time.Minute), echomw.BodyLimit(limits.BodySource))
	group.GET("/submissions/:id", api.submission)
	group.PATCH("/submissions/:id", api.updateSubmission, echomw.BodyLimit(limits.BodyShortText))
	group.POST("/submissions/:id/rejudge", api.rejudgeSubmission)

	group.GET("/discussion", api.discussions)
	group.POST("/discussion", api.createDiscussion, api.rateLimit("discussion", 30, time.Minute), echomw.BodyLimit(limits.BodyMarkdown))
	group.GET("/discussion/:id", api.discussion)
	group.PATCH("/discussion/:id", api.updateDiscussion, echomw.BodyLimit(limits.BodyMarkdown))
	group.DELETE("/discussion/:id", api.deleteDiscussion)
	group.POST("/discussion/:id/comments", api.createComment, api.rateLimit("comment", 60, time.Minute), echomw.BodyLimit(limits.BodyShortText))
	group.DELETE("/discussion/:id/comments/:commentId", api.deleteComment)

	group.GET("/notifications", api.notifications)
	group.POST("/notifications/read-all", api.readAllNotifications)
	group.POST("/notifications/:id/read", api.readNotification)
}

func (api *API) requireGuestAccess(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if api.role(c) == "guest" && !settings.GuestAllowed(api.db) {
			return echo.NewHTTPError(http.StatusForbidden, "guest access is disabled")
		}
		return next(c)
	}
}

func (api *API) rateLimit(scope string, limit int, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := "doj:rate:" + scope + ":" + api.rateIdentity(c)
			allowed, err := cache.Allow(c.Request().Context(), key, limit, window)
			if err != nil {
				return err
			}
			if !allowed {
				return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
			}
			return next(c)
		}
	}
}

func (api *API) rateIdentity(c echo.Context) string {
	user, err := api.currentUser(c)
	if err == nil && user.ID > 0 {
		return "user:" + strconv.FormatUint(uint64(user.ID), 10)
	}
	ip := strings.TrimSpace(c.RealIP())
	if ip == "" {
		ip = requestHostname(c.Request().RemoteAddr)
	}
	if ip == "" {
		ip = "unknown"
	}
	return "ip:" + ip
}
