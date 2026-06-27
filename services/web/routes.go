package web

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/doveccl/doj/models"
	adminsvc "github.com/doveccl/doj/services/admin"
	"github.com/doveccl/doj/services/events"
	"github.com/doveccl/doj/utils"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type API struct {
	db *gorm.DB
}

const (
	maxAssetBytes         = 128 << 20
	maxEditableAssetBytes = 1 << 20
	maxTitleRunes         = models.TitleMax
)

type Home struct {
	Notice      string        `json:"notice"`
	Heatmap     []HeatCell    `json:"heatmap"`
	Problems    []HomeProblem `json:"problems"`
	Assignments []Item        `json:"assignments"`
	Contests    []Item        `json:"contests"`
}

type CreatedID struct {
	ID uint `json:"id"`
}

type NoticeUpdate struct {
	Content string `json:"content"`
}

type HeatCell struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type Item struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Meta  string `json:"meta"`
}

type HomeProblem struct {
	ID     uint     `json:"id"`
	Title  string   `json:"title"`
	Tags   []string `json:"tags"`
	AC     int      `json:"ac"`
	Submit int      `json:"submit"`
}

type PageResult[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
}

type MeDTO struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Mail   string `json:"mail"`
	Bio    string `json:"bio"`
	Avatar string `json:"avatar"`
	Admin  bool   `json:"admin"`
}

type LanguageDTO struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Source string `json:"source"`
}

type MeUpdate struct {
	Mail   *string `json:"mail,omitempty"`
	Bio    *string `json:"bio,omitempty"`
	Avatar *string `json:"avatar,omitempty"`
}

type PasswordUpdate struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Mail     string `json:"mail"`
	Password string `json:"password"`
}

type AssignmentDTO struct {
	ID     uint      `json:"id"`
	Title  string    `json:"title"`
	EndAt  time.Time `json:"endAt"`
	Status string    `json:"status"`
	Total  int       `json:"total"`
	Done   int       `json:"done"`
	Users  []uint    `json:"users,omitempty"`
	Groups []uint    `json:"groups,omitempty"`
}

type AssignmentCreate struct {
	Title    string       `json:"title"`
	EndAt    string       `json:"endAt"`
	Problems []ProblemRef `json:"problems"`
	Users    []uint       `json:"users"`
	Groups   []uint       `json:"groups"`
}

type AssignmentUpdate struct {
	Title    string       `json:"title"`
	EndAt    string       `json:"endAt"`
	Problems []ProblemRef `json:"problems"`
	Users    []uint       `json:"users"`
	Groups   []uint       `json:"groups"`
}

type ProblemRef struct {
	ID   uint   `json:"id"`
	Sort string `json:"sort"`
}

type AssignmentDetail struct {
	Assignment AssignmentDTO           `json:"assignment"`
	Problems   []ProblemDTO            `json:"problems"`
	Progress   []AssignmentProgressDTO `json:"progress"`
}

type AssignmentProgressDTO struct {
	User     string                         `json:"user"`
	AC       int                            `json:"ac"`
	Submit   int                            `json:"submit"`
	Problems []AssignmentProblemProgressDTO `json:"problems"`
}

type AssignmentProblemProgressDTO struct {
	ProblemID uint   `json:"problemId"`
	Status    string `json:"status"`
}

type ContestDTO struct {
	ID       uint       `json:"id"`
	Title    string     `json:"title"`
	Kind     string     `json:"kind"`
	StartAt  time.Time  `json:"startAt"`
	EndAt    time.Time  `json:"endAt"`
	FreezeAt *time.Time `json:"freezeAt"`
	Status   string     `json:"status"`
	Total    int        `json:"total"`
}

type ContestCreate struct {
	Title    string       `json:"title"`
	Kind     string       `json:"kind"`
	StartAt  string       `json:"startAt"`
	EndAt    string       `json:"endAt"`
	FreezeAt string       `json:"freezeAt"`
	Problems []ProblemRef `json:"problems"`
}

type ContestUpdate struct {
	Title    string       `json:"title"`
	Kind     string       `json:"kind"`
	StartAt  string       `json:"startAt"`
	EndAt    string       `json:"endAt"`
	FreezeAt string       `json:"freezeAt"`
	Problems []ProblemRef `json:"problems"`
}

type ContestDetail struct {
	Contest  ContestDTO    `json:"contest"`
	Problems []ProblemDTO  `json:"problems"`
	Rank     []RankUserDTO `json:"rank"`
}

type SubmissionListItem struct {
	ID           uint      `json:"id"`
	ProblemID    uint      `json:"problemId"`
	ProblemTitle string    `json:"problemTitle"`
	User         string    `json:"user"`
	Language     string    `json:"language"`
	Status       string    `json:"status"`
	TimeMS       *int      `json:"timeMs,omitempty"`
	MemoryKB     *int      `json:"memoryKb,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type SubmissionDTO struct {
	ID           uint      `json:"id"`
	ProblemID    uint      `json:"problemId"`
	ProblemTitle string    `json:"problemTitle"`
	User         string    `json:"user"`
	Language     string    `json:"language"`
	Status       string    `json:"status"`
	Score        int       `json:"score"`
	Message      string    `json:"message"`
	TimeMS       *int      `json:"timeMs,omitempty"`
	MemoryKB     *int      `json:"memoryKb,omitempty"`
	Public       bool      `json:"public"`
	CreatedAt    time.Time `json:"createdAt"`
}

type SubmissionDetail struct {
	Submission SubmissionDTO `json:"submission"`
	Code       string        `json:"code"`
	Cases      []CaseDTO     `json:"cases"`
}

type SubmitRequest struct {
	ProblemID uint   `json:"problemId"`
	Language  string `json:"language"`
	Code      string `json:"code"`
	Public    bool   `json:"public"`
}

type SubmissionUpdate struct {
	Public bool `json:"public"`
}

type CaseDTO struct {
	No       int    `json:"no"`
	Status   string `json:"status"`
	TimeMS   *int   `json:"timeMs,omitempty"`
	MemoryKB *int   `json:"memoryKb,omitempty"`
	Message  string `json:"message"`
}

type RankUserDTO struct {
	Rank     int              `json:"rank"`
	User     string           `json:"user"`
	Bio      string           `json:"bio"`
	Avatar   string           `json:"avatar"`
	AC       int              `json:"ac"`
	Submit   int              `json:"submit"`
	Score    int              `json:"score"`
	Penalty  int              `json:"penalty"`
	Problems []RankProblemDTO `json:"problems"`
}

type RankProblemDTO struct {
	ProblemID uint   `json:"problemId"`
	Status    string `json:"status"`
	Submit    int    `json:"submit"`
	Score     int    `json:"score"`
	Penalty   int    `json:"penalty"`
}

type PublicUserDTO struct {
	Name   string `json:"name"`
	Bio    string `json:"bio"`
	Avatar string `json:"avatar"`
	Admin  bool   `json:"admin"`
	AC     int    `json:"ac"`
	Submit int    `json:"submit"`
}

type UserOptionDTO struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type UserProfile struct {
	User       PublicUserDTO     `json:"user"`
	Heatmap    []HeatCell        `json:"heatmap"`
	Solved     []SolvedProblem   `json:"solved"`
	Activities []UserActivityDTO `json:"activities"`
}

type SolvedProblem struct {
	ID     uint     `json:"id"`
	Title  string   `json:"title"`
	Tags   []string `json:"tags"`
	AC     int      `json:"ac"`
	Submit int      `json:"submit"`
}

type UserActivityDTO struct {
	Type         string    `json:"type"`
	ID           uint      `json:"id"`
	Title        string    `json:"title"`
	Status       string    `json:"status,omitempty"`
	ProblemID    uint      `json:"problemId,omitempty"`
	ProblemTitle string    `json:"problemTitle,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type DiscussionDTO struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	Tags      []string  `json:"tags"`
	Pinned    bool      `json:"pinned"`
	Locked    bool      `json:"locked"`
	Replies   int       `json:"replies"`
	CreatedAt time.Time `json:"createdAt"`
}

type DiscussionCreate struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

type DiscussionUpdate struct {
	Title   *string   `json:"title,omitempty"`
	Content *string   `json:"content,omitempty"`
	Tags    *[]string `json:"tags,omitempty"`
	Pinned  *bool     `json:"pinned,omitempty"`
	Locked  *bool     `json:"locked,omitempty"`
}

type DiscussionDetail struct {
	Discussion DiscussionDTO `json:"discussion"`
	Content    string        `json:"content"`
	Comments   []CommentDTO  `json:"comments"`
}

type CommentDTO struct {
	ID        uint      `json:"id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type CommentCreate struct {
	Content string `json:"content"`
}

type ProblemDTO struct {
	ID          uint       `json:"id"`
	Sort        string     `json:"sort,omitempty"`
	Title       string     `json:"title"`
	Statement   string     `json:"statement,omitempty"`
	Tags        []string   `json:"tags"`
	Visible     bool       `json:"visible"`
	Mode        string     `json:"mode"`
	TimeMS      int        `json:"timeMs"`
	MemoryMB    int        `json:"memoryMb"`
	Cases       *int       `json:"cases,omitempty"`
	DataBytes   *int64     `json:"dataBytes,omitempty"`
	AC          int        `json:"ac"`
	Submit      int        `json:"submit"`
	Discussions int        `json:"discussions"`
	Mine        string     `json:"mine"`
	Latest      *RecordDTO `json:"latest,omitempty"`
}

type RecordDTO struct {
	ID        uint      `json:"id"`
	Status    string    `json:"status"`
	Score     int       `json:"score"`
	CreatedAt time.Time `json:"createdAt"`
}

type ProblemCreate struct {
	Title    string   `json:"title"`
	Tags     []string `json:"tags"`
	Visible  *bool    `json:"visible"`
	Mode     string   `json:"mode"`
	TimeMS   int      `json:"timeMs"`
	MemoryMB int      `json:"memoryMb"`
}

type ProblemUpdate struct {
	Title     *string   `json:"title,omitempty"`
	Statement *string   `json:"statement,omitempty"`
	Tags      *[]string `json:"tags,omitempty"`
	Visible   *bool     `json:"visible,omitempty"`
	Mode      *string   `json:"mode,omitempty"`
	TimeMS    *int      `json:"timeMs,omitempty"`
	MemoryMB  *int      `json:"memoryMb,omitempty"`
}

type ProblemVisibilityUpdate struct {
	Visible bool `json:"visible"`
}

type ProblemAssets struct {
	Data      []AssetFile `json:"data"`
	Judge     []AssetFile `json:"judge"`
	Assets    []AssetFile `json:"assets"`
	Cases     int         `json:"cases"`
	DataBytes int64       `json:"dataBytes"`
}

type AssetFile struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Editable bool   `json:"editable"`
}

type AssetCaseCreate struct {
	Name   string `json:"name"`
	Input  string `json:"input"`
	Output string `json:"output"`
}

type AssetContent struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

type AssetContentUpdate struct {
	Key     string `json:"key"`
	Content string `json:"content"`
}

type UploadResult struct {
	URL string `json:"url"`
}

func Register(e *echo.Echo, db *gorm.DB) {
	if db == nil {
		panic("web API requires a database")
	}
	api := &API{db: db}
	e.GET("/api/health", api.health)
	e.GET("/api/ready", api.ready)
	e.GET("/api/site", api.site)
	e.POST("/api/auth/login", api.login, api.rateLimit("auth", 20, time.Minute), echomw.BodyLimit(utils.BodyLimitShortText))
	e.POST("/api/auth/register", api.register, api.rateLimit("auth", 20, time.Minute), echomw.BodyLimit(utils.BodyLimitShortText))
	e.POST("/api/auth/logout", api.logout)
	e.GET("/api/me", api.me)
	e.PATCH("/api/me", api.updateMe, echomw.BodyLimit(utils.BodyLimitShortText))
	e.PATCH("/api/me/password", api.updatePassword)
	group := e.Group("/api", api.requireGuestAccess)
	group.GET("/events", api.events)
	group.GET("/home", api.home)
	group.GET("/languages", api.languages)
	group.PATCH("/home/notice", api.updateNotice, echomw.BodyLimit(utils.BodyLimitMarkdown))
	group.POST("/uploads/images", api.uploadImage, api.rateLimit("upload", 60, time.Minute), echomw.BodyLimit(utils.BodyLimitImage))
	group.GET("/users/:id/:year/:month/:day/*", api.userMedia)
	group.GET("/problems", api.problems)
	group.POST("/problems", api.createProblem)
	group.GET("/tags", api.tags)
	group.GET("/problems/:id", api.problem)
	group.PATCH("/problems/:id", api.updateProblem, echomw.BodyLimit(utils.BodyLimitMarkdown))
	group.PATCH("/problems/:id/visibility", api.updateProblemVisibility, echomw.BodyLimit(utils.BodyLimitShortText))
	group.DELETE("/problems/:id", api.deleteProblem)
	group.GET("/problems/:id/assets", api.problemAssets)
	group.POST("/problems/:id/assets/images", api.uploadProblemImage, api.rateLimit("problem-upload", 60, time.Minute), echomw.BodyLimit(utils.BodyLimitImage))
	group.GET("/problems/:id/data/*", api.problemPrivateData)
	group.GET("/problems/:id/judge/*", api.problemPrivateJudge)
	group.POST("/problems/:id/assets/files", api.uploadProblemAsset, api.rateLimit("problem-upload", 60, time.Minute), echomw.BodyLimit(utils.BodyLimitAsset))
	group.DELETE("/problems/:id/assets/files", api.deleteProblemAsset)
	group.GET("/problems/:id/assets/files/content", api.problemAssetContent)
	group.PATCH("/problems/:id/assets/files/content", api.updateProblemAssetContent, echomw.BodyLimit(utils.BodyLimitEditAsset))
	group.POST("/problems/:id/assets/cases", api.createProblemCase, echomw.BodyLimit(utils.BodyLimitEditAsset))
	group.POST("/problems/:id/assets/template", api.fillJudgeTemplate, echomw.BodyLimit(utils.BodyLimitEditAsset))
	group.GET("/problems/:id/assets/*", api.problemPublicAsset)
	group.GET("/assignments", api.assignments)
	group.POST("/assignments", api.createAssignment)
	group.GET("/assignments/:id", api.assignment)
	group.PATCH("/assignments/:id", api.updateAssignment)
	group.DELETE("/assignments/:id", api.deleteAssignment)
	group.GET("/contests", api.contests)
	group.POST("/contests", api.createContest)
	group.GET("/contests/:id", api.contest)
	group.PATCH("/contests/:id", api.updateContest)
	group.DELETE("/contests/:id", api.deleteContest)
	group.GET("/submissions", api.submissions)
	group.POST("/submissions", api.submit, api.rateLimit("submit", 30, time.Minute), echomw.BodyLimit(utils.BodyLimitSource))
	group.GET("/submissions/:id", api.submission)
	group.PATCH("/submissions/:id", api.updateSubmission)
	group.POST("/submissions/:id/rejudge", api.rejudgeSubmission)
	group.GET("/rank", api.rank)
	group.GET("/users", api.users)
	group.GET("/users/:name", api.user)
	group.GET("/discussion", api.discussions)
	group.POST("/discussion", api.createDiscussion, api.rateLimit("discussion", 30, time.Minute), echomw.BodyLimit(utils.BodyLimitMarkdown))
	group.GET("/discussion/:id", api.discussion)
	group.PATCH("/discussion/:id", api.updateDiscussion, echomw.BodyLimit(utils.BodyLimitMarkdown))
	group.DELETE("/discussion/:id", api.deleteDiscussion)
	group.POST("/discussion/:id/comments", api.createComment, api.rateLimit("comment", 60, time.Minute), echomw.BodyLimit(utils.BodyLimitShortText))
}

func (api *API) requireGuestAccess(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if api.role(c) == "guest" && !adminsvc.GuestAllowed(api.db) {
			return echo.NewHTTPError(http.StatusForbidden, "guest access is disabled")
		}
		return next(c)
	}
}

func (api *API) rateLimit(scope string, limit int, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := "doj:rate:" + scope + ":" + api.rateIdentity(c)
			allowed, err := utils.CacheAllow(c.Request().Context(), key, limit, window)
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

func (api *API) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (api *API) ready(c echo.Context) error {

	sqlDB, err := api.db.DB()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
	}
	if err := sqlDB.PingContext(c.Request().Context()); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
}

func (api *API) events(c echo.Context) error {
	flusher, ok := c.Response().Writer.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "streaming is not supported")
	}

	response := c.Response()
	response.Header().Set(echo.HeaderContentType, "text/event-stream")
	response.Header().Set(echo.HeaderCacheControl, "no-cache")
	response.Header().Set(echo.HeaderConnection, "keep-alive")
	response.Header().Set("X-Accel-Buffering", "no")
	response.WriteHeader(http.StatusOK)

	if err := writeSSE(response.Writer, "ready", []byte("{}")); err != nil {
		return nil
	}
	flusher.Flush()

	ch, unsubscribe := events.Default.Subscribe()
	defer unsubscribe()
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			if err := writeSSE(response.Writer, event.Type, event.Data); err != nil {
				return nil
			}
			flusher.Flush()
		case <-ticker.C:
			if _, err := io.WriteString(response.Writer, ": ping\n\n"); err != nil {
				return nil
			}
			flusher.Flush()
		}
	}
}

func writeSSE(writer io.Writer, event string, data []byte) error {
	if _, err := fmt.Fprintf(writer, "event: %s\n", event); err != nil {
		return err
	}
	if _, err := writer.Write([]byte("data: ")); err != nil {
		return err
	}
	lines := bytes.Split(data, []byte("\n"))
	for index, line := range lines {
		if index > 0 {
			if _, err := writer.Write([]byte("\ndata: ")); err != nil {
				return err
			}
		}
		if _, err := writer.Write(line); err != nil {
			return err
		}
	}
	_, err := writer.Write([]byte("\n\n"))
	return err
}

func (api *API) site(c echo.Context) error {
	settings, err := adminsvc.Settings(api.db)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, settings)
}

func (api *API) login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	nameKey := userNameKey(req.Name)
	if nameKey == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "username is required")
	}

	now := time.Now()

	var user models.User
	err := api.db.Where("LOWER(name) = ? OR LOWER(mail) = ?", nameKey, nameKey).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid username or password")
		}
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Auth), []byte(req.Password)) != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid username or password")
	}
	if err := api.createSession(c, user, now); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, meDTO(user))
}

func (api *API) register(c echo.Context) error {
	if !adminsvc.RegistrationAllowed(api.db) {
		return echo.NewHTTPError(http.StatusForbidden, "registration is disabled")
	}
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Mail = strings.ToLower(strings.TrimSpace(req.Mail))
	if err := validateRegister(req); err != nil {
		return err
	}

	now := time.Now()

	nameKey := userNameKey(req.Name)
	var count int64
	if err := api.db.Model(&models.User{}).Where("LOWER(name) = ? OR LOWER(mail) = ?", nameKey, req.Mail).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return echo.NewHTTPError(http.StatusConflict, "user already exists")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user := models.User{
		Name: req.Name,
		Mail: req.Mail,
		Auth: string(hash),
	}
	if err := api.db.Create(&user).Error; err != nil {
		return err
	}
	if err := api.createSession(c, user, now); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, meDTO(user))
}

func (api *API) createSession(c echo.Context, user models.User, now time.Time) error {
	return utils.CreateUserSession(c, user.ID, now)
}

func (api *API) logout(c echo.Context) error {

	if err := utils.DeleteSession(c); err != nil {
		return err
	}
	utils.ClearSessionCookie(c)
	return c.NoContent(http.StatusNoContent)
}

func (api *API) me(c echo.Context) error {
	refreshCSRFCookie(c)

	user, err := api.currentUser(c)
	if err != nil {
		return c.JSON(http.StatusOK, guestMe())
	}
	return c.JSON(http.StatusOK, meDTO(user))
}

func refreshCSRFCookie(c echo.Context) {
	token, ok := utils.SessionToken(c)
	if !ok {
		return
	}
	utils.SetCSRFCookie(c, token, utils.SessionExpiresAt(time.Now()))
}

func (api *API) updateMe(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	var req MeUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.Mail == nil && req.Bio == nil && req.Avatar == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "no fields to update")
	}

	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	if req.Mail != nil {
		mail := strings.ToLower(strings.TrimSpace(*req.Mail))
		if err := validateMail(mail); err != nil {
			return err
		}
		if err := api.ensureMailAvailable(mail, user.ID); err != nil {
			return err
		}
		user.Mail = mail
	}
	if req.Bio != nil {
		bio := strings.TrimSpace(*req.Bio)
		if len([]rune(bio)) > models.BioMax {
			return echo.NewHTTPError(http.StatusBadRequest, "bio is too long")
		}
		user.Bio = bio
	}
	if req.Avatar != nil {
		avatar := strings.TrimSpace(*req.Avatar)
		if len(avatar) > models.AvatarMax {
			return echo.NewHTTPError(http.StatusBadRequest, "avatar url is too long")
		}
		user.Avatar = avatar
	}
	if err := api.db.Save(&user).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusOK, meDTO(user))
}

func (api *API) ensureMailAvailable(mail string, currentUserID uint) error {
	var count int64
	if err := api.db.Model(&models.User{}).
		Where("LOWER(mail) = ? AND id <> ?", strings.ToLower(mail), currentUserID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return echo.NewHTTPError(http.StatusConflict, "mail already exists")
	}
	return nil
}

func (api *API) updatePassword(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	var req PasswordUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "old and new password are required")
	}

	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Auth), []byte(req.OldPassword)) != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "old password is invalid")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := api.db.Model(&user).Update("auth", string(hash)).Error; err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

func (api *API) languages(c echo.Context) error {

	var rows []models.Language
	if err := api.db.Order("id asc").Limit(100).Find(&rows).Error; err != nil {
		return err
	}
	items := make([]LanguageDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, langDTO(row))
	}
	return c.JSON(http.StatusOK, items)
}

func (api *API) uploadImage(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "image file is required")
	}
	data, mime, sha, ext, err := readUploadedImage(file)
	if err != nil {
		return err
	}
	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	year, month, day := uploadDateParts(time.Now())
	key := path.Join("users", strconv.Itoa(int(user.ID)), year, month, day, sha+ext)
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	if err := store.Put(c.Request().Context(), key, bytes.NewReader(data), int64(len(data)), mime); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, UploadResult{URL: "/" + path.Join("api", key)})
}

func readUploadedImage(file *multipart.FileHeader) ([]byte, string, string, string, error) {
	src, err := file.Open()
	if err != nil {
		return nil, "", "", "", err
	}
	defer src.Close()

	const maxImageBytes = 5 << 20
	data, err := io.ReadAll(io.LimitReader(src, maxImageBytes+1))
	if err != nil {
		return nil, "", "", "", err
	}
	if len(data) > maxImageBytes {
		return nil, "", "", "", echo.NewHTTPError(http.StatusRequestEntityTooLarge, "image is too large")
	}
	mime := http.DetectContentType(data)
	if !strings.HasPrefix(mime, "image/") {
		return nil, "", "", "", echo.NewHTTPError(http.StatusBadRequest, "image file is required")
	}
	sum := sha256.Sum256(data)
	sha := hex.EncodeToString(sum[:])
	ext := uploadExt(file.Filename, mime)
	return data, mime, sha, ext, nil
}

func uploadDateParts(now time.Time) (string, string, string) {
	return now.Format("2006"), now.Format("01"), now.Format("02")
}

func (api *API) userMedia(c echo.Context) error {
	userID, err := parseID(c, "id", "invalid user id")
	if err != nil {
		return err
	}
	rel, err := utils.CleanObjectKey(path.Join(
		strconv.Itoa(int(userID)),
		c.Param("year"),
		c.Param("month"),
		c.Param("day"),
		c.Param("*"),
	))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "media not found")
	}
	key := path.Join("users", rel)
	if !userUploadKeyAllowed(key) {
		return echo.NewHTTPError(http.StatusNotFound, "media not found")
	}
	return streamMedia(c, key, "media not found")
}

func streamMedia(c echo.Context, key string, notFound string) error {
	if !sameSiteMediaRequest(c) {
		return echo.NewHTTPError(http.StatusForbidden, "media hotlink is not allowed")
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	reader, contentType, err := store.Open(c.Request().Context(), key)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, notFound)
	}
	defer reader.Close()
	c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=31536000, immutable")
	c.Response().Header().Set(echo.HeaderXContentTypeOptions, "nosniff")
	return c.Stream(http.StatusOK, contentType, reader)
}

func userUploadKeyAllowed(key string) bool {
	parts := strings.Split(key, "/")
	return len(parts) >= 5 && parts[0] == "users" && parts[1] != ""
}

func (api *API) home(c echo.Context) error {
	problems, err := api.homeProblems(c)
	if err != nil {
		return err
	}
	heatmap, err := api.homeHeatmap(c)
	if err != nil {
		return err
	}
	assignments, err := api.homeAssignments(c)
	if err != nil {
		return err
	}
	contests, err := api.homeContests(c)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, Home{
		Notice:      api.notice(),
		Heatmap:     heatmap,
		Problems:    problems,
		Assignments: assignments,
		Contests:    contests,
	})
}

func (api *API) homeProblems(c echo.Context) ([]HomeProblem, error) {
	var rows []models.Problem
	query := api.db.Select("id", "title", "tags").Order("id desc").Limit(5)
	if !api.isAdmin(c) {
		query = api.applyProblemListVisibility(query)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]HomeProblem, 0, len(rows))
	for _, row := range rows {
		items = append(items, HomeProblem{
			ID:    row.ID,
			Title: row.Title,
			Tags:  readTags([]byte(row.Tags)),
		})
	}
	if err := api.decorateHomeProblemStats(items); err != nil {
		return nil, err
	}
	return items, nil
}

func (api *API) decorateHomeProblemStats(items []HomeProblem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	submitByProblem, acByProblem, err := api.problemSubmissionStats(ids)
	if err != nil {
		return err
	}
	for index := range items {
		id := items[index].ID
		items[index].Submit = submitByProblem[id]
		items[index].AC = acByProblem[id]
	}
	return nil
}

func (api *API) decorateSolvedProblemStats(items []SolvedProblem) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	submitByProblem, acByProblem, err := api.problemSubmissionStats(ids)
	if err != nil {
		return err
	}
	for index := range items {
		id := items[index].ID
		items[index].Submit = submitByProblem[id]
		items[index].AC = acByProblem[id]
	}
	return nil
}

func (api *API) homeHeatmap(c echo.Context) ([]HeatCell, error) {
	if api.role(c) == "guest" {
		return []HeatCell{}, nil
	}

	user, err := api.currentUser(c)
	if err != nil {
		return nil, err
	}
	return api.userHeatmap(user.ID)
}

func (api *API) homeAssignments(c echo.Context) ([]Item, error) {

	var rows []models.Assignment
	if err := api.db.Order("end_at desc").Limit(5).Find(&rows).Error; err != nil {
		return nil, err
	}
	assignments, err := api.assignmentDTOs(c, rows, false)
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(assignments))
	for _, row := range assignments {
		items = append(items, Item{ID: row.ID, Title: row.Title, Meta: strconv.Itoa(row.Done) + "/" + strconv.Itoa(row.Total)})
	}
	return items, nil
}

func (api *API) homeContests(c echo.Context) ([]Item, error) {

	var rows []models.Contest
	if err := api.db.Order("start_at desc").Limit(5).Find(&rows).Error; err != nil {
		return nil, err
	}
	contests, err := api.contestDTOs(rows, api.isAdmin(c))
	if err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(contests))
	for _, row := range contests {
		items = append(items, Item{ID: row.ID, Title: row.Title, Meta: row.Kind + " · " + strconv.Itoa(row.Total)})
	}
	return items, nil
}

func (api *API) updateNotice(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	var req NoticeUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	if strings.TrimSpace(req.Content) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "notice content is required")
	}
	if err := validateTextBytes(req.Content, utils.MaxMarkdownBytes, "notice content is too large"); err != nil {
		return err
	}

	settings, err := adminsvc.Settings(api.db)
	if err != nil {
		return err
	}
	settings.Notice = req.Content
	if err := adminsvc.SaveSettings(api.db, settings); err != nil {
		return err
	}
	return api.home(c)
}

func (api *API) problems(c echo.Context) error {
	page, pageSize, offset, err := parsePage(c)
	if err != nil {
		return err
	}
	problems, total, err := api.searchProblemPage(c, c.QueryParam("q"), c.QueryParam("tag"), pageSize, offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, PageResult[ProblemDTO]{Items: problems, Page: page, PageSize: pageSize, Total: total})
}

func (api *API) tags(c echo.Context) error {
	items, err := api.searchTags(c, c.QueryParam("kind"), c.QueryParam("q"), 50)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, items)
}

func (api *API) problem(c echo.Context) error {
	if strings.HasSuffix(c.Param("id"), ".zip") {
		return api.downloadProblemAssets(c)
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid problem id")
	}

	var problem models.Problem
	if err := api.db.First(&problem, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	if !api.isAdmin(c) && !api.problemVisibleInDetail(problem) {
		return echo.NewHTTPError(http.StatusNotFound, "problem not found")
	}
	item, err := api.problemDTOWithStatement(c.Request().Context(), problem)
	if err != nil {
		return err
	}
	items := []ProblemDTO{item}
	if err := api.decorateProblemStats(c.Request().Context(), items); err != nil {
		return err
	}
	if err := api.decorateProblemMines(c, items); err != nil {
		return err
	}
	if err := api.decorateProblemDiscussions(c.Request().Context(), items); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, items[0])
}

func (api *API) createProblem(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	var req ProblemCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Mode = strings.TrimSpace(req.Mode)
	req.Tags = normalizeTags(req.Tags)
	if req.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if err := validateTitle(req.Title); err != nil {
		return err
	}
	if req.Mode == "" {
		req.Mode = "default"
	}
	if !validProblemMode(req.Mode) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid judge mode")
	}
	if req.TimeMS <= 0 {
		req.TimeMS = 1000
	}
	if req.MemoryMB <= 0 {
		req.MemoryMB = 256
	}

	tags, _ := json.Marshal(req.Tags)
	visible := true
	if req.Visible != nil {
		visible = *req.Visible
	}

	row := models.Problem{
		Title:    req.Title,
		Tags:     tags,
		Visible:  visible,
		Mode:     req.Mode,
		TimeMS:   req.TimeMS,
		MemoryMB: req.MemoryMB,
	}
	if err := api.db.Create(&row).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, CreatedID{ID: row.ID})
}

func (api *API) updateProblem(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid problem id")
	if err != nil {
		return err
	}
	var req ProblemUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.Title == nil && req.Statement == nil && req.Tags == nil && req.Visible == nil && req.Mode == nil && req.TimeMS == nil && req.MemoryMB == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "no fields to update")
	}

	var row models.Problem
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "title is required")
		}
		if err := validateTitle(title); err != nil {
			return err
		}
		row.Title = title
	}
	if req.Tags != nil {
		tags, _ := json.Marshal(normalizeTags(*req.Tags))
		row.Tags = tags
	}
	if req.Visible != nil {
		row.Visible = *req.Visible
	}
	if req.Mode != nil {
		mode := strings.TrimSpace(*req.Mode)
		if mode == "" {
			mode = "default"
		}
		if !validProblemMode(mode) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid judge mode")
		}
		row.Mode = mode
	}
	if req.TimeMS != nil {
		timeMS := *req.TimeMS
		if timeMS <= 0 {
			timeMS = 1000
		}
		row.TimeMS = timeMS
	}
	if req.MemoryMB != nil {
		memoryMB := *req.MemoryMB
		if memoryMB <= 0 {
			memoryMB = 256
		}
		row.MemoryMB = memoryMB
	}
	var statement *string
	if req.Statement != nil {
		value := strings.TrimSpace(*req.Statement)
		if value == "" && row.Title != "" {
			value = "# " + row.Title
		}
		if err := validateTextBytes(value, utils.MaxMarkdownBytes, "statement is too large"); err != nil {
			return err
		}
		statement = &value
	}
	if err := api.db.Save(&row).Error; err != nil {
		return err
	}
	if statement != nil {
		if err := api.writeProblemStatement(c.Request().Context(), row.ID, *statement); err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, CreatedID{ID: row.ID})
}

func (api *API) updateProblemVisibility(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid problem id")
	if err != nil {
		return err
	}
	var req ProblemVisibilityUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}

	var row models.Problem
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	row.Visible = req.Visible
	if err := api.db.Save(&row).Error; err != nil {
		return err
	}
	items := []ProblemDTO{problemDTO(row)}
	if err := api.decorateProblemSubmissionStats(items); err != nil {
		return err
	}
	if err := api.decorateProblemMines(c, items); err != nil {
		return err
	}
	if err := api.decorateProblemDiscussions(c.Request().Context(), items); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, items[0])
}

func (api *API) deleteProblem(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid problem id")
	if err != nil {
		return err
	}

	if err := api.db.Delete(&models.Problem{}, id).Error; err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (api *API) problemAssets(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	assets, err := api.syncProblemAssets(c, id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, assets)
}

func (api *API) uploadProblemImage(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "image file is required")
	}
	data, mime, sha, ext, err := readUploadedImage(file)
	if err != nil {
		return err
	}
	rel := sha + ext
	key := path.Join("problems", strconv.Itoa(int(id)), "assets", rel)
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	if err := store.Put(c.Request().Context(), key, bytes.NewReader(data), int64(len(data)), mime); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, UploadResult{URL: fmt.Sprintf("/api/problems/%d/assets/%s", id, rel)})
}

func (api *API) problemPrivateData(c echo.Context) error {
	return api.problemPrivateAsset(c, "data")
}

func (api *API) problemPrivateJudge(c echo.Context) error {
	return api.problemPrivateAsset(c, "judge")
}

func (api *API) problemPrivateAsset(c echo.Context, section string) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	rel, err := utils.CleanObjectKey(c.Param("*"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "asset not found")
	}
	key := path.Join("problems", strconv.Itoa(int(id)), section, rel)
	return streamMedia(c, key, "asset not found")
}

func (api *API) problemPublicAsset(c echo.Context) error {
	id, err := parseID(c, "id", "invalid problem id")
	if err != nil {
		return err
	}
	if err := api.requireProblemVisible(c, id); err != nil {
		return err
	}
	rel, err := utils.CleanObjectKey(c.Param("*"))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "media not found")
	}
	key := path.Join("problems", strconv.Itoa(int(id)), "assets", rel)
	return streamMedia(c, key, "media not found")
}

func sameSiteMediaRequest(c echo.Context) bool {
	raw := c.Request().Referer()
	if raw == "" {
		return true
	}
	ref, err := url.Parse(raw)
	if err != nil || ref.Hostname() == "" {
		return false
	}
	return strings.EqualFold(ref.Hostname(), requestHostname(c.Request().Host))
}

func requestHostname(host string) string {
	if value, _, err := net.SplitHostPort(host); err == nil {
		return strings.Trim(value, "[]")
	}
	return strings.Trim(host, "[]")
}

func (api *API) uploadProblemAsset(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	section, err := assetSection(c.FormValue("section"))
	if err != nil {
		return err
	}
	file, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "asset file is required")
	}
	name, err := cleanAssetName(file.Filename)
	if err != nil {
		return err
	}
	if file.Size > maxAssetBytes {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "asset file is too large")
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	buffer := make([]byte, 512)
	n, readErr := src.Read(buffer)
	if readErr != nil && readErr != io.EOF {
		return readErr
	}
	contentType := http.DetectContentType(buffer[:n])
	reader := io.MultiReader(bytes.NewReader(buffer[:n]), src)
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	key := path.Join(problemAssetPrefix(id, section), name)
	if err := store.Put(c.Request().Context(), key, reader, file.Size, contentType); err != nil {
		return err
	}
	clearProblemPackageCacheIfNeeded(c.Request().Context(), id, key)
	assets, err := api.syncProblemAssets(c, id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, assets)
}

func (api *API) deleteProblemAsset(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	key, err := utils.CleanObjectKey(c.QueryParam("key"))
	if err != nil || !problemAssetKeyAllowed(id, key) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid asset key")
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	if err := store.Delete(c.Request().Context(), key); err != nil {
		return err
	}
	clearProblemPackageCacheIfNeeded(c.Request().Context(), id, key)
	assets, err := api.syncProblemAssets(c, id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, assets)
}

func (api *API) problemAssetContent(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	key, err := cleanEditableAssetKey(id, c.QueryParam("key"))
	if err != nil {
		return err
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	reader, _, err := store.Open(c.Request().Context(), key)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "asset not found")
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, maxEditableAssetBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxEditableAssetBytes {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "asset is too large to edit")
	}
	return c.JSON(http.StatusOK, AssetContent{Key: key, Name: path.Base(key), Content: string(data)})
}

func (api *API) updateProblemAssetContent(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	var req AssetContentUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	key, err := cleanEditableAssetKey(id, req.Key)
	if err != nil {
		return err
	}
	if len(req.Content) > maxEditableAssetBytes {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "asset is too large to edit")
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	if err := store.Put(c.Request().Context(), key, strings.NewReader(req.Content), int64(len(req.Content)), "text/plain; charset=utf-8"); err != nil {
		return err
	}
	clearProblemPackageCacheIfNeeded(c.Request().Context(), id, key)
	assets, err := api.syncProblemAssets(c, id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, assets)
}

func (api *API) createProblemCase(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	var req AssetCaseCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	assets, err := problemAssetsFromStore(c.Request().Context(), id, store)
	if err != nil {
		return err
	}
	name, err := caseName(req.Name, assets)
	if err != nil {
		return err
	}
	inputKey := path.Join(problemAssetPrefix(id, "data"), name+".in")
	outputKey := path.Join(problemAssetPrefix(id, "data"), name+".out")
	if err := store.Put(c.Request().Context(), inputKey, strings.NewReader(req.Input), int64(len(req.Input)), "text/plain; charset=utf-8"); err != nil {
		return err
	}
	if err := store.Put(c.Request().Context(), outputKey, strings.NewReader(req.Output), int64(len(req.Output)), "text/plain; charset=utf-8"); err != nil {
		return err
	}
	clearProblemPackageCache(c.Request().Context(), id)
	assets, err = api.syncProblemAssets(c, id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, assets)
}

func (api *API) fillJudgeTemplate(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	for name, body := range judgeTemplateFiles() {
		key := path.Join(problemAssetPrefix(id, "judge"), name)
		if err := store.Put(c.Request().Context(), key, strings.NewReader(body), int64(len(body)), "text/plain; charset=utf-8"); err != nil {
			return err
		}
	}
	clearProblemPackageCache(c.Request().Context(), id)
	assets, err := api.syncProblemAssets(c, id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, assets)
}

func (api *API) downloadProblemAssets(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	rawID := strings.TrimSuffix(c.Param("id"), ".zip")
	parsed, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid problem id")
	}
	id := uint(parsed)
	var problem models.Problem
	if err := api.db.First(&problem, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	statement, err := api.problemStatement(c.Request().Context(), id, problem.Title)
	if err != nil {
		return err
	}
	assets, err := problemAssetsFromStore(c.Request().Context(), id, store)
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("P%d.zip", id)
	c.Response().Header().Set(echo.HeaderContentType, "application/zip")
	c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="`+filename+`"`)
	c.Response().WriteHeader(http.StatusOK)
	writer := zip.NewWriter(c.Response().Writer)
	defer writer.Close()
	if err := writeProblemStatementZipFile(writer, statement); err != nil {
		return err
	}
	if err := writeAssetZipFiles(c.Request().Context(), writer, store, "data", assets.Data); err != nil {
		return err
	}
	if err := writeAssetZipFiles(c.Request().Context(), writer, store, "judge", assets.Judge); err != nil {
		return err
	}
	if err := writeAssetZipFiles(c.Request().Context(), writer, store, "assets", assets.Assets); err != nil {
		return err
	}
	return nil
}

func (api *API) assignments(c echo.Context) error {

	page, pageSize, offset, err := parsePage(c)
	if err != nil {
		return err
	}
	var rows []models.Assignment
	query := api.db.Model(&models.Assignment{})
	if !api.isAdmin(c) {
		user, err := api.currentUser(c)
		if err != nil {
			return c.JSON(http.StatusOK, PageResult[AssignmentDTO]{Items: []AssignmentDTO{}, Page: page, PageSize: pageSize, Total: 0})
		}
		query = query.Where(`
			EXISTS (
				SELECT 1 FROM assignment_users
				WHERE assignment_users.assignment_id = assignments.id
				AND assignment_users.user_id = ?
				AND assignment_users.deleted_at IS NULL
			)
			OR EXISTS (
				SELECT 1 FROM assignment_groups
				JOIN group_users ON group_users.group_id = assignment_groups.group_id
				WHERE assignment_groups.assignment_id = assignments.id
				AND group_users.user_id = ?
				AND assignment_groups.deleted_at IS NULL
				AND group_users.deleted_at IS NULL
			)
		`, user.ID, user.ID)
	}
	if q := strings.TrimSpace(c.QueryParam("q")); q != "" {
		like := "%" + q + "%"
		if id, err := parseQueryID(q, "invalid assignment id"); err == nil {
			query = query.Where("id = ? OR LOWER(title) LIKE LOWER(?)", id, like)
		} else {
			query = query.Where("LOWER(title) LIKE LOWER(?)", like)
		}
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return err
	}
	if err := query.Session(&gorm.Session{}).Order("end_at desc").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return err
	}
	items, err := api.assignmentDTOs(c, rows, false)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, PageResult[AssignmentDTO]{Items: items, Page: page, PageSize: pageSize, Total: total})
}

func (api *API) createAssignment(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	var req AssignmentCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if err := validateTitle(req.Title); err != nil {
		return err
	}
	endAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid deadline")
	}

	req.Problems = normalizeProblemRefs(req.Problems)
	if err := api.validateProblemRefs(req.Problems); err != nil {
		return err
	}
	req.Users = cleanUintList(req.Users)
	req.Groups = cleanUintList(req.Groups)
	if err := api.validateUserIDs(req.Users); err != nil {
		return err
	}
	if err := api.validateGroupIDs(req.Groups); err != nil {
		return err
	}
	row := models.Assignment{Title: req.Title, EndAt: endAt}
	if err := api.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		for _, item := range req.Problems {
			link := models.AssignmentProblem{AssignmentID: row.ID, ProblemID: item.ID, Sort: item.Sort}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		if err := saveAssignmentMembers(tx, row.ID, req.Users, req.Groups); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, CreatedID{ID: row.ID})
}

func (api *API) updateAssignment(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid assignment id")
	if err != nil {
		return err
	}
	var req AssignmentUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if err := validateTitle(req.Title); err != nil {
		return err
	}
	endAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid deadline")
	}

	var row models.Assignment
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "assignment not found")
		}
		return err
	}
	req.Problems = normalizeProblemRefs(req.Problems)
	if err := api.validateProblemRefs(req.Problems); err != nil {
		return err
	}
	req.Users = cleanUintList(req.Users)
	req.Groups = cleanUintList(req.Groups)
	if err := api.validateUserIDs(req.Users); err != nil {
		return err
	}
	if err := api.validateGroupIDs(req.Groups); err != nil {
		return err
	}
	row.Title = req.Title
	row.EndAt = endAt
	if err := api.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		if err := tx.Where("assignment_id = ?", row.ID).Delete(&models.AssignmentProblem{}).Error; err != nil {
			return err
		}
		for _, item := range req.Problems {
			link := models.AssignmentProblem{AssignmentID: row.ID, ProblemID: item.ID, Sort: item.Sort}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		if err := saveAssignmentMembers(tx, row.ID, req.Users, req.Groups); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, CreatedID{ID: row.ID})
}

func (api *API) deleteAssignment(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid assignment id")
	if err != nil {
		return err
	}

	if err := api.db.Delete(&models.Assignment{}, id).Error; err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (api *API) assignment(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid assignment id")
	}

	var row models.Assignment
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "assignment not found")
		}
		return err
	}
	allowed, err := api.assignmentVisible(c, row.ID)
	if err != nil {
		return err
	}
	if !allowed {
		return echo.NewHTTPError(http.StatusNotFound, "assignment not found")
	}
	var links []models.AssignmentProblem
	if err := api.db.Where("assignment_id = ?", row.ID).Order("sort asc").Find(&links).Error; err != nil {
		return err
	}
	problems, err := api.assignmentProblems(c, links)
	if err != nil {
		return err
	}
	if err := api.decorateProblemMines(c, problems); err != nil {
		return err
	}
	progressRows, err := api.assignmentProgress(row.ID, problems)
	if err != nil {
		return err
	}
	done, err := api.assignmentDoneCount(c, row.ID)
	if err != nil {
		return err
	}
	total, err := api.assignmentProblemCount(row.ID)
	if err != nil {
		return err
	}
	dto, err := api.assignmentDTO(c, row, total, done)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, AssignmentDetail{Assignment: dto, Problems: problems, Progress: progressRows})
}

func (api *API) contests(c echo.Context) error {

	page, pageSize, offset, err := parsePage(c)
	if err != nil {
		return err
	}
	var rows []models.Contest
	query := api.db.Model(&models.Contest{})
	if q := strings.TrimSpace(c.QueryParam("q")); q != "" {
		like := "%" + q + "%"
		if id, err := parseQueryID(q, "invalid contest id"); err == nil {
			query = query.Where("id = ? OR LOWER(title) LIKE LOWER(?)", id, like)
		} else {
			query = query.Where("LOWER(title) LIKE LOWER(?)", like)
		}
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return err
	}
	if err := query.Session(&gorm.Session{}).Order("start_at desc").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return err
	}
	items, err := api.contestDTOs(rows, api.isAdmin(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, PageResult[ContestDTO]{Items: items, Page: page, PageSize: pageSize, Total: total})
}

func (api *API) createContest(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	var req ContestCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Kind = strings.TrimSpace(strings.ToUpper(req.Kind))
	if req.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if err := validateTitle(req.Title); err != nil {
		return err
	}
	if req.Kind == "" {
		req.Kind = "OI"
	}
	if !validContestKind(req.Kind) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid contest kind")
	}
	startAt, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid start time")
	}
	endAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid end time")
	}
	if !endAt.After(startAt) {
		return echo.NewHTTPError(http.StatusBadRequest, "end time must be after start time")
	}
	var freezeAt *time.Time
	if req.Kind == "ICPC" {
		var err error
		freezeAt, err = parseContestFreezeAt(req.FreezeAt, startAt, endAt)
		if err != nil {
			return err
		}
	}

	req.Problems = normalizeProblemRefs(req.Problems)
	if err := api.validateProblemRefs(req.Problems); err != nil {
		return err
	}
	row := models.Contest{Title: req.Title, Kind: req.Kind, StartAt: startAt, EndAt: endAt, FreezeAt: freezeAt}
	if err := api.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		for _, item := range req.Problems {
			link := models.ContestProblem{ContestID: row.ID, ProblemID: item.ID, Sort: item.Sort}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, CreatedID{ID: row.ID})
}

func (api *API) updateContest(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid contest id")
	if err != nil {
		return err
	}
	var req ContestUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Kind = strings.TrimSpace(strings.ToUpper(req.Kind))
	if req.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if err := validateTitle(req.Title); err != nil {
		return err
	}
	if req.Kind == "" {
		req.Kind = "OI"
	}
	if !validContestKind(req.Kind) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid contest kind")
	}
	startAt, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid start time")
	}
	endAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid end time")
	}
	if !endAt.After(startAt) {
		return echo.NewHTTPError(http.StatusBadRequest, "end time must be after start time")
	}
	var freezeAt *time.Time
	if req.Kind == "ICPC" {
		var err error
		freezeAt, err = parseContestFreezeAt(req.FreezeAt, startAt, endAt)
		if err != nil {
			return err
		}
	}

	var row models.Contest
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "contest not found")
		}
		return err
	}
	req.Problems = normalizeProblemRefs(req.Problems)
	if err := api.validateProblemRefs(req.Problems); err != nil {
		return err
	}
	row.Title = req.Title
	row.Kind = req.Kind
	row.StartAt = startAt
	row.EndAt = endAt
	row.FreezeAt = freezeAt
	if err := api.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		if err := tx.Where("contest_id = ?", row.ID).Delete(&models.ContestProblem{}).Error; err != nil {
			return err
		}
		for _, item := range req.Problems {
			link := models.ContestProblem{ContestID: row.ID, ProblemID: item.ID, Sort: item.Sort}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, CreatedID{ID: row.ID})
}

func (api *API) deleteContest(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid contest id")
	if err != nil {
		return err
	}

	if err := api.db.Delete(&models.Contest{}, id).Error; err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (api *API) contest(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid contest id")
	}

	var row models.Contest
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "contest not found")
		}
		return err
	}
	var links []models.ContestProblem
	if err := api.db.Where("contest_id = ?", row.ID).Order("sort asc").Find(&links).Error; err != nil {
		return err
	}
	problems, err := api.contestProblems(c, row, links)
	if err != nil {
		return err
	}
	if err := api.decorateProblemMinesInContest(c, problems, row.ID); err != nil {
		return err
	}
	admin := api.isAdmin(c)
	rank := []RankUserDTO{}
	if row.Kind != "OI" || !contestRunning(row) || admin {
		freezeAt := api.contestFreezeCutoff(c, row)
		contestIncludeHidden := admin || contestRunning(row)
		rank, err = api.contestRank(row, problems, contestIncludeHidden, freezeAt)
		if err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, ContestDetail{Contest: contestDTO(row, len(problems)), Problems: problems, Rank: rank})
}

func (api *API) submissions(c echo.Context) error {

	page, pageSize, offset, err := parsePage(c)
	if err != nil {
		return err
	}
	var rows []models.Submission
	query := api.db.Model(&models.Submission{})
	if !api.isAdmin(c) {
		query = query.Joins("JOIN problems ON problems.id = submissions.problem_id")
		query = api.applyProblemListVisibility(query)
	}
	if problem := c.QueryParam("problem"); problem != "" {
		id, err := parseProblemQuery(problem)
		if err != nil {
			return err
		}
		query = query.Where("submissions.problem_id = ?", id)
	}
	if user := strings.TrimSpace(c.QueryParam("user")); user != "" {
		query = query.Joins("JOIN users submission_users ON submission_users.id = submissions.user_id AND submission_users.deleted_at IS NULL").
			Where("LOWER(submission_users.name) = ?", userNameKey(user))
	}
	if assignment := c.QueryParam("assignment"); assignment != "" {
		id, err := parseQueryID(assignment, "invalid assignment id")
		if err != nil {
			return err
		}
		query = query.Where("submissions.assignment_id = ?", id)
	}
	if contest := c.QueryParam("contest"); contest != "" {
		id, err := parseQueryID(contest, "invalid contest id")
		if err != nil {
			return err
		}
		query = query.Where("submissions.contest_id = ?", id)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return err
	}
	if err := query.Session(&gorm.Session{}).
		Select("submissions.id", "submissions.problem_id", "submissions.user_id", "submissions.language", "submissions.status", "submissions.time_ms", "submissions.memory_kb", "submissions.created_at").
		Order("submissions.created_at desc").
		Limit(pageSize).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return err
	}
	items, err := api.submissionListItems(rows)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, PageResult[SubmissionListItem]{Items: items, Page: page, PageSize: pageSize, Total: total})
}

func (api *API) submit(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	var req SubmitRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Language = strings.TrimSpace(req.Language)
	req.Code = strings.TrimSpace(req.Code)
	if req.ProblemID == 0 || req.Language == "" || req.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "problem, language and code are required")
	}
	if err := validateTextBytes(req.Code, utils.MaxSourceBytes, "source code is too large"); err != nil {
		return err
	}

	var problem models.Problem
	if err := api.db.First(&problem, req.ProblemID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	if !api.isAdmin(c) && !api.problemVisibleInDetail(problem) {
		return echo.NewHTTPError(http.StatusNotFound, "problem not found")
	}
	var language models.Language
	if err := api.db.First(&language, "id = ?", req.Language).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusBadRequest, "language not found")
		}
		return err
	}
	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	assignmentID, contestID, err := api.inferSubmitScopes(user.ID, req.ProblemID, time.Now())
	if err != nil {
		return err
	}
	row := models.Submission{
		UserID:       user.ID,
		ProblemID:    req.ProblemID,
		AssignmentID: assignmentID,
		ContestID:    contestID,
		Language:     req.Language,
		Code:         req.Code,
		Status:       "queued",
		Public:       req.Public,
	}
	if err := api.db.Create(&row).Error; err != nil {
		return err
	}
	events.SubmissionChanged()
	return c.JSON(http.StatusCreated, CreatedID{ID: row.ID})
}

func (api *API) inferSubmitScopes(userID uint, problemID uint, now time.Time) (*uint, *uint, error) {
	assignmentID, err := api.activeAssignmentFor(userID, problemID, now)
	if err != nil {
		return nil, nil, err
	}
	contestID, err := api.activeContestFor(problemID, now)
	if err != nil {
		return nil, nil, err
	}
	return assignmentID, contestID, nil
}

func (api *API) activeAssignmentFor(userID uint, problemID uint, now time.Time) (*uint, error) {
	var rows []models.Assignment
	if err := api.db.
		Joins("JOIN assignment_problems ON assignment_problems.assignment_id = assignments.id").
		Where("assignment_problems.problem_id = ? AND assignments.end_at >= ?", problemID, now).
		Order("assignments.end_at asc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		ok, err := api.userAssignedTo(row.ID, userID)
		if err != nil {
			return nil, err
		}
		if ok {
			id := row.ID
			return &id, nil
		}
	}
	return nil, nil
}

func (api *API) userAssignedTo(assignmentID uint, userID uint) (bool, error) {
	var direct int64
	if err := api.db.Model(&models.AssignmentUser{}).
		Where("assignment_id = ? AND user_id = ?", assignmentID, userID).
		Count(&direct).Error; err != nil {
		return false, err
	}
	if direct > 0 {
		return true, nil
	}
	var byGroup int64
	if err := api.db.Model(&models.AssignmentGroup{}).
		Joins("JOIN group_users ON group_users.group_id = assignment_groups.group_id").
		Where("assignment_groups.assignment_id = ? AND group_users.user_id = ?", assignmentID, userID).
		Count(&byGroup).Error; err != nil {
		return false, err
	}
	return byGroup > 0, nil
}

func (api *API) assignmentVisible(c echo.Context, assignmentID uint) (bool, error) {
	if api.isAdmin(c) {
		return true, nil
	}
	if api.role(c) == "guest" {
		return false, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return false, err
	}
	return api.userAssignedTo(assignmentID, user.ID)
}

func saveAssignmentMembers(tx *gorm.DB, assignmentID uint, users []uint, groups []uint) error {
	if err := tx.Where("assignment_id = ?", assignmentID).Delete(&models.AssignmentUser{}).Error; err != nil {
		return err
	}
	if err := tx.Where("assignment_id = ?", assignmentID).Delete(&models.AssignmentGroup{}).Error; err != nil {
		return err
	}
	for _, userID := range users {
		if err := tx.Create(&models.AssignmentUser{AssignmentID: assignmentID, UserID: userID}).Error; err != nil {
			return err
		}
	}
	for _, groupID := range groups {
		if err := tx.Create(&models.AssignmentGroup{AssignmentID: assignmentID, GroupID: groupID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (api *API) activeContestFor(problemID uint, now time.Time) (*uint, error) {
	var row models.Contest
	err := api.db.
		Joins("JOIN contest_problems ON contest_problems.contest_id = contests.id").
		Where("contest_problems.problem_id = ? AND contests.start_at <= ? AND contests.end_at >= ?", problemID, now, now).
		Order("contests.start_at desc").
		First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row.ID, nil
}

func (api *API) contestRank(contest models.Contest, problems []ProblemDTO, includeHidden bool, until *time.Time) ([]RankUserDTO, error) {
	var rows []models.Submission
	query := api.db.
		Joins("JOIN users ON users.id = submissions.user_id AND users.deleted_at IS NULL").
		Where("submissions.contest_id = ?", contest.ID).
		Order("submissions.created_at asc")
	if until != nil {
		query = query.Where("submissions.created_at < ?", *until)
	}
	if !includeHidden {
		query = query.Joins("JOIN problems ON problems.id = submissions.problem_id")
		query = api.applyProblemListVisibility(query)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	users, err := api.rankUsers(rows)
	if err != nil {
		return nil, err
	}
	if contest.Kind == "ICPC" {
		return icpcRank(contest, rows, users, problems), nil
	}
	return oiRank(rows, users, problems), nil
}

func (api *API) rankUsers(submissions []models.Submission) (map[uint]models.User, error) {
	ids := map[uint]bool{}
	for _, row := range submissions {
		ids[row.UserID] = true
	}
	if len(ids) == 0 {
		return map[uint]models.User{}, nil
	}
	values := make([]uint, 0, len(ids))
	for id := range ids {
		values = append(values, id)
	}
	var rows []models.User
	if err := api.db.Where("id IN ?", values).Find(&rows).Error; err != nil {
		return nil, err
	}
	users := make(map[uint]models.User, len(rows))
	for _, row := range rows {
		users[row.ID] = row
	}
	return users, nil
}

func oiRank(submissions []models.Submission, users map[uint]models.User, problems []ProblemDTO) []RankUserDTO {
	type state struct {
		user    models.User
		submit  int
		score   map[uint]int
		attempt map[uint]int
	}
	problemSet := rankProblemSet(problems)
	states := map[uint]*state{}
	for _, row := range submissions {
		if !problemSet[row.ProblemID] {
			continue
		}
		user, ok := users[row.UserID]
		if !ok {
			continue
		}
		got := states[row.UserID]
		if got == nil {
			got = &state{user: user, score: map[uint]int{}, attempt: map[uint]int{}}
			states[row.UserID] = got
		}
		got.submit++
		got.attempt[row.ProblemID]++
		got.score[row.ProblemID] = row.Score
	}
	items := make([]RankUserDTO, 0, len(states))
	for _, got := range states {
		score := 0
		ac := 0
		problemItems := make([]RankProblemDTO, 0, len(problems))
		for _, problem := range problems {
			value := got.score[problem.ID]
			submit := got.attempt[problem.ID]
			status := "none"
			if submit > 0 {
				status = "tried"
			}
			if value >= 100 {
				status = "ac"
				ac++
			}
			score += value
			problemItems = append(problemItems, RankProblemDTO{ProblemID: problem.ID, Status: status, Submit: submit, Score: value})
		}
		items = append(items, RankUserDTO{User: got.user.Name, Bio: got.user.Bio, Avatar: got.user.Avatar, AC: ac, Submit: got.submit, Score: score, Problems: problemItems})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		return items[i].User < items[j].User
	})
	for index := range items {
		items[index].Rank = index + 1
	}
	return items
}

func icpcRank(contest models.Contest, submissions []models.Submission, users map[uint]models.User, problems []ProblemDTO) []RankUserDTO {
	type problemState struct {
		wrong   int
		submit  int
		solved  bool
		penalty int
	}
	type state struct {
		user     models.User
		submit   int
		problems map[uint]*problemState
	}
	problemSet := rankProblemSet(problems)
	states := map[uint]*state{}
	for _, row := range submissions {
		if !problemSet[row.ProblemID] {
			continue
		}
		user, ok := users[row.UserID]
		if !ok {
			continue
		}
		got := states[row.UserID]
		if got == nil {
			got = &state{user: user, problems: map[uint]*problemState{}}
			states[row.UserID] = got
		}
		got.submit++
		problem := got.problems[row.ProblemID]
		if problem == nil {
			problem = &problemState{}
			got.problems[row.ProblemID] = problem
		}
		problem.submit++
		if problem.solved {
			continue
		}
		if row.Status == "AC" {
			problem.solved = true
			minutes := int(row.CreatedAt.Sub(contest.StartAt).Minutes())
			if minutes < 0 {
				minutes = 0
			}
			problem.penalty = minutes + problem.wrong*20
			continue
		}
		if penalizable(row.Status) {
			problem.wrong++
		}
	}
	items := make([]RankUserDTO, 0, len(states))
	for _, got := range states {
		ac := 0
		penalty := 0
		problemItems := make([]RankProblemDTO, 0, len(problems))
		for _, contestProblem := range problems {
			problem := got.problems[contestProblem.ID]
			status := "none"
			submit := 0
			problemPenalty := 0
			if problem != nil {
				submit = problem.submit
				if problem.solved {
					status = "ac"
					submit = problem.wrong + 1
					problemPenalty = problem.penalty
					ac++
					penalty += problem.penalty
				} else if problem.submit > 0 {
					status = "tried"
				}
			}
			problemItems = append(problemItems, RankProblemDTO{ProblemID: contestProblem.ID, Status: status, Submit: submit, Score: boolScore(status == "ac"), Penalty: problemPenalty})
		}
		items = append(items, RankUserDTO{User: got.user.Name, Bio: got.user.Bio, Avatar: got.user.Avatar, AC: ac, Submit: got.submit, Score: ac, Penalty: penalty, Problems: problemItems})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].AC != items[j].AC {
			return items[i].AC > items[j].AC
		}
		if items[i].Penalty != items[j].Penalty {
			return items[i].Penalty < items[j].Penalty
		}
		if items[i].Submit != items[j].Submit {
			return items[i].Submit < items[j].Submit
		}
		return items[i].User < items[j].User
	})
	for index := range items {
		items[index].Rank = index + 1
	}
	return items
}

func rankProblemSet(problems []ProblemDTO) map[uint]bool {
	items := make(map[uint]bool, len(problems))
	for _, problem := range problems {
		items[problem.ID] = true
	}
	return items
}

func boolScore(value bool) int {
	if value {
		return 1
	}
	return 0
}

func penalizable(status string) bool {
	switch status {
	case "AC", "CE", "SE":
		return false
	default:
		return true
	}
}

func (api *API) assignmentProgress(id uint, problems []ProblemDTO) ([]AssignmentProgressDTO, error) {
	userIDs, err := api.assignmentProgressUserIDs(id)
	if err != nil {
		return nil, err
	}
	if len(userIDs) == 0 {
		return []AssignmentProgressDTO{}, nil
	}

	var users []models.User
	if err := api.db.Where("id IN ?", userIDs).Order("name asc").Find(&users).Error; err != nil {
		return nil, err
	}

	problemIDs := make([]uint, 0, len(problems))
	for _, problem := range problems {
		problemIDs = append(problemIDs, problem.ID)
	}

	type state struct {
		item      AssignmentProgressDTO
		byProblem map[uint]*AssignmentProblemProgressDTO
	}
	states := map[uint]*state{}
	for _, user := range users {
		item := AssignmentProgressDTO{
			User:     user.Name,
			Problems: make([]AssignmentProblemProgressDTO, 0, len(problems)),
		}
		byProblem := map[uint]*AssignmentProblemProgressDTO{}
		for _, problem := range problems {
			item.Problems = append(item.Problems, AssignmentProblemProgressDTO{ProblemID: problem.ID, Status: "none"})
			byProblem[problem.ID] = &item.Problems[len(item.Problems)-1]
		}
		states[user.ID] = &state{item: item, byProblem: byProblem}
	}

	if len(problemIDs) > 0 {
		var submissions []struct {
			UserID    uint
			ProblemID uint
			Status    string
		}
		if err := api.db.Model(&models.Submission{}).
			Select("user_id, problem_id, status").
			Where("assignment_id = ? AND user_id IN ? AND problem_id IN ?", id, userIDs, problemIDs).
			Find(&submissions).Error; err != nil {
			return nil, err
		}
		for _, submission := range submissions {
			got := states[submission.UserID]
			if got == nil {
				continue
			}
			got.item.Submit++
			problem := got.byProblem[submission.ProblemID]
			if problem == nil {
				continue
			}
			if submission.Status == "AC" {
				problem.Status = "ac"
				continue
			}
			if problem.Status != "ac" {
				problem.Status = "tried"
			}
		}
	}

	items := make([]AssignmentProgressDTO, 0, len(states))
	for _, got := range states {
		for _, problem := range got.item.Problems {
			if problem.Status == "ac" {
				got.item.AC++
			}
		}
		items = append(items, got.item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].AC != items[j].AC {
			return items[i].AC > items[j].AC
		}
		if items[i].Submit != items[j].Submit {
			return items[i].Submit < items[j].Submit
		}
		return items[i].User < items[j].User
	})
	return items, nil
}

func (api *API) assignmentProgressUserIDs(id uint) ([]uint, error) {
	userSet := map[uint]struct{}{}
	var direct []models.AssignmentUser
	if err := api.db.Where("assignment_id = ?", id).Find(&direct).Error; err != nil {
		return nil, err
	}
	for _, row := range direct {
		userSet[row.UserID] = struct{}{}
	}

	var grouped []struct {
		UserID uint
	}
	if err := api.db.Model(&models.GroupUser{}).
		Select("group_users.user_id").
		Joins("JOIN assignment_groups ON assignment_groups.group_id = group_users.group_id").
		Where("assignment_groups.assignment_id = ?", id).
		Find(&grouped).Error; err != nil {
		return nil, err
	}
	for _, row := range grouped {
		userSet[row.UserID] = struct{}{}
	}

	userIDs := make([]uint, 0, len(userSet))
	for id := range userSet {
		userIDs = append(userIDs, id)
	}
	sort.Slice(userIDs, func(i, j int) bool { return userIDs[i] < userIDs[j] })
	return userIDs, nil
}

func (api *API) submission(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid submission id")
	}

	var row models.Submission
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "submission not found")
		}
		return err
	}
	if !api.isAdmin(c) {
		var problem models.Problem
		if err := api.db.First(&problem, row.ProblemID).Error; err != nil {
			return err
		}
		if !api.problemVisibleInDetail(problem) {
			return echo.NewHTTPError(http.StatusNotFound, "submission not found")
		}
	}
	if !api.isAdmin(c) {
		user, err := api.currentUser(c)
		if err != nil || user.ID != row.UserID {
			locked, err := api.submissionSourceLocked(row)
			if err != nil {
				return err
			}
			if locked || !row.Public {
				return echo.NewHTTPError(http.StatusForbidden, "submission source is not public")
			}
		}
	}
	var cases []models.Case
	if err := api.db.Where("submission_id = ?", row.ID).Order("no asc").Find(&cases).Error; err != nil {
		return err
	}
	items := make([]CaseDTO, 0, len(cases))
	for _, item := range cases {
		items = append(items, CaseDTO{No: item.No, Status: item.Status, TimeMS: item.TimeMS, MemoryKB: item.MemoryKB, Message: item.Message})
	}
	return c.JSON(http.StatusOK, SubmissionDetail{Submission: api.submissionDTO(row), Code: row.Code, Cases: items})
}

func (api *API) updateSubmission(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid submission id")
	}
	var req SubmissionUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	var row models.Submission
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "submission not found")
		}
		return err
	}
	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	if !user.Admin && user.ID != row.UserID {
		return echo.NewHTTPError(http.StatusForbidden, "submission can only be updated by owner or admin")
	}
	row.Public = req.Public
	if err := api.db.Model(&row).Update("public", row.Public).Error; err != nil {
		return err
	}
	events.SubmissionChanged()
	return c.JSON(http.StatusOK, CreatedID{ID: row.ID})
}

func (api *API) rejudgeSubmission(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid submission id")
	}
	var row models.Submission
	err = api.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&row, id).Error; err != nil {
			return err
		}
		if err := tx.Where("submission_id = ?", row.ID).Delete(&models.Case{}).Error; err != nil {
			return err
		}
		updates := map[string]any{
			"status":      "queued",
			"score":       0,
			"message":     "",
			"attempt":     gorm.Expr("attempt + 1"),
			"judger_id":   nil,
			"lease_until": nil,
			"time_ms":     nil,
			"memory_kb":   nil,
		}
		if err := tx.Model(&row).Updates(updates).Error; err != nil {
			return err
		}
		return tx.First(&row, row.ID).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "submission not found")
		}
		return err
	}
	events.SubmissionChanged()
	return c.JSON(http.StatusOK, CreatedID{ID: row.ID})
}

func (api *API) submissionSourceLocked(row models.Submission) (bool, error) {
	if row.ContestID != nil {
		var contest models.Contest
		if err := api.db.First(&contest, *row.ContestID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return false, nil
			}
			return false, err
		}
		if !contestEnded(contest) {
			return true, nil
		}
	}
	if row.AssignmentID != nil {
		var assignment models.Assignment
		if err := api.db.First(&assignment, *row.AssignmentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return false, nil
			}
			return false, err
		}
		if !assignmentEnded(assignment) {
			return true, nil
		}
	}
	return false, nil
}

func assignmentEnded(row models.Assignment) bool {
	return !time.Now().Before(row.EndAt)
}

func (api *API) rank(c echo.Context) error {

	var users []models.User
	if err := api.db.Order("id asc").Limit(100).Find(&users).Error; err != nil {
		return err
	}
	userIDs := make([]uint, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}
	stats, err := api.userStatsMap(userIDs)
	if err != nil {
		return err
	}
	items := make([]RankUserDTO, 0, len(users))
	for _, user := range users {
		got := stats[user.ID]
		ac := got.AC
		submit := got.Submit
		items = append(items, RankUserDTO{User: user.Name, Bio: user.Bio, Avatar: user.Avatar, AC: ac, Submit: submit})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].AC != items[j].AC {
			return items[i].AC > items[j].AC
		}
		if items[i].Submit != items[j].Submit {
			return items[i].Submit < items[j].Submit
		}
		return items[i].User < items[j].User
	})
	for index := range items {
		items[index].Rank = index + 1
	}
	return c.JSON(http.StatusOK, items)
}

func (api *API) users(c echo.Context) error {
	q := strings.TrimSpace(c.QueryParam("q"))
	var rows []models.User
	query := api.db.Select("id", "name").Order("id asc").Limit(50)
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("LOWER(name) LIKE LOWER(?)", like)
	}
	if err := query.Find(&rows).Error; err != nil {
		return err
	}
	items := make([]UserOptionDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, UserOptionDTO{ID: row.ID, Name: row.Name})
	}
	return c.JSON(http.StatusOK, items)
}

func (api *API) user(c echo.Context) error {
	nameKey := userNameKey(c.Param("name"))
	if nameKey == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user name is required")
	}

	var row models.User
	if err := api.db.Where("LOWER(name) = ?", nameKey).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return err
	}

	includeHidden := api.isAdmin(c)
	solved, err := api.solvedProblems(row.ID, includeHidden)
	if err != nil {
		return err
	}
	activities, err := api.userActivities(row.ID, includeHidden)
	if err != nil {
		return err
	}
	heatmap, err := api.userHeatmap(row.ID)
	if err != nil {
		return err
	}
	ac, submit, err := api.userStats(c.Request().Context(), row.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, UserProfile{
		User:       PublicUserDTO{Name: row.Name, Bio: row.Bio, Avatar: row.Avatar, Admin: row.Admin, AC: ac, Submit: submit},
		Heatmap:    heatmap,
		Solved:     solved,
		Activities: activities,
	})
}

type userStatsCache struct {
	AC     int `json:"ac"`
	Submit int `json:"submit"`
}

func (api *API) userStats(ctx context.Context, userID uint) (int, int, error) {
	key := userStatsCacheKey(userID)
	var cached userStatsCache
	found, err := utils.CacheGet(ctx, key, &cached)
	if err == nil && found {
		return cached.AC, cached.Submit, nil
	}

	submitQuery := api.db.Model(&models.Submission{}).Where("submissions.user_id = ?", userID)
	var submit int64
	if err := submitQuery.Count(&submit).Error; err != nil {
		return 0, 0, err
	}

	acQuery := api.db.Model(&models.Submission{}).
		Where("submissions.user_id = ? AND submissions.status = ?", userID, "AC").
		Distinct("submissions.problem_id")
	var ac int64
	if err := acQuery.Count(&ac).Error; err != nil {
		return 0, 0, err
	}
	stats := userStatsCache{AC: int(ac), Submit: int(submit)}
	_ = utils.CacheSet(ctx, key, stats, 10*time.Second)
	return stats.AC, stats.Submit, nil
}

func (api *API) userStatsMap(userIDs []uint) (map[uint]userStatsCache, error) {
	userIDs = uniqueUint(userIDs)
	stats := map[uint]userStatsCache{}
	if len(userIDs) == 0 {
		return stats, nil
	}
	var submits []struct {
		UserID uint
		Count  int64
	}
	if err := api.db.Model(&models.Submission{}).
		Select("user_id, count(*) as count").
		Where("user_id IN ?", userIDs).
		Group("user_id").
		Find(&submits).Error; err != nil {
		return nil, err
	}
	for _, row := range submits {
		item := stats[row.UserID]
		item.Submit = int(row.Count)
		stats[row.UserID] = item
	}
	var acs []struct {
		UserID uint
		Count  int64
	}
	if err := api.db.Model(&models.Submission{}).
		Select("user_id, count(DISTINCT problem_id) as count").
		Where("user_id IN ? AND status = ?", userIDs, "AC").
		Group("user_id").
		Find(&acs).Error; err != nil {
		return nil, err
	}
	for _, row := range acs {
		item := stats[row.UserID]
		item.AC = int(row.Count)
		stats[row.UserID] = item
	}
	return stats, nil
}

func userStatsCacheKey(userID uint) string {
	return "doj:user:" + strconv.FormatUint(uint64(userID), 10) + ":stats"
}

func (api *API) userSubmissions(userID uint, includeHidden bool) ([]models.Submission, error) {
	var rows []models.Submission
	query := api.db.
		Select("submissions.id", "submissions.problem_id", "submissions.status", "submissions.created_at").
		Where("submissions.user_id = ?", userID).
		Order("submissions.created_at desc").
		Limit(20)
	if !includeHidden {
		query = query.Joins("JOIN problems ON problems.id = submissions.problem_id")
		query = api.applyProblemListVisibility(query)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (api *API) userActivities(userID uint, includeHidden bool) ([]UserActivityDTO, error) {
	submissions, err := api.userSubmissions(userID, includeHidden)
	if err != nil {
		return nil, err
	}
	items := make([]UserActivityDTO, 0, len(submissions)+20)
	problemIDs := make([]uint, 0, len(submissions))
	for _, submission := range submissions {
		problemIDs = append(problemIDs, submission.ProblemID)
	}
	titles, err := api.problemTitleMap(problemIDs)
	if err != nil {
		return nil, err
	}
	for _, submission := range submissions {
		title := titles[submission.ProblemID]
		if title == "" {
			title = "P" + strconv.Itoa(int(submission.ProblemID))
		}
		items = append(items, UserActivityDTO{
			Type:         "submission",
			ID:           submission.ID,
			Title:        title,
			Status:       submission.Status,
			ProblemID:    submission.ProblemID,
			ProblemTitle: title,
			CreatedAt:    submission.CreatedAt,
		})
	}

	var discussions []models.Discussion
	if err := api.db.Select("id", "title", "created_at").Where("user_id = ?", userID).Order("created_at desc").Limit(20).Find(&discussions).Error; err != nil {
		return nil, err
	}
	for _, row := range discussions {
		items = append(items, UserActivityDTO{
			Type:      "discussion",
			ID:        row.ID,
			Title:     row.Title,
			CreatedAt: row.CreatedAt,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if len(items) > 20 {
		items = items[:20]
	}
	return items, nil
}

func (api *API) solvedProblems(userID uint, includeHidden bool) ([]SolvedProblem, error) {
	var rows []models.Submission
	query := api.db.
		Select("submissions.problem_id", "submissions.created_at").
		Where("submissions.user_id = ? AND submissions.status = ?", userID, "AC").
		Order("submissions.created_at desc").
		Limit(50)
	if !includeHidden {
		query = query.Joins("JOIN problems ON problems.id = submissions.problem_id")
		query = api.applyProblemListVisibility(query)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	seen := map[uint]bool{}
	problemIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		if seen[row.ProblemID] {
			continue
		}
		seen[row.ProblemID] = true
		problemIDs = append(problemIDs, row.ProblemID)
	}
	if len(problemIDs) == 0 {
		return []SolvedProblem{}, nil
	}
	problemQuery := api.db.Model(&models.Problem{}).Select("id", "title", "tags").Where("id IN ?", problemIDs)
	if !includeHidden {
		problemQuery = api.applyProblemListVisibility(problemQuery)
	}
	var problems []models.Problem
	if err := problemQuery.Find(&problems).Error; err != nil {
		return nil, err
	}
	byID := problemRowsByID(problems)
	items := make([]SolvedProblem, 0, len(problemIDs))
	for _, id := range problemIDs {
		problem, ok := byID[id]
		if !ok {
			continue
		}
		items = append(items, SolvedProblem{
			ID:    problem.ID,
			Title: problem.Title,
			Tags:  readTags([]byte(problem.Tags)),
		})
	}
	if err := api.decorateSolvedProblemStats(items); err != nil {
		return nil, err
	}
	return items, nil
}

func (api *API) userHeatmap(userID uint) ([]HeatCell, error) {
	since := time.Now().AddDate(-1, 0, 0)
	var rows []models.Submission
	query := api.db.Select("created_at").Where("submissions.user_id = ? AND submissions.created_at >= ?", userID, since)
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, row := range rows {
		counts[row.CreatedAt.Format("2006-01-02")]++
	}
	return heatmapFromCounts(counts), nil
}

func (api *API) discussions(c echo.Context) error {

	page, pageSize, offset, err := parsePage(c)
	if err != nil {
		return err
	}
	var rows []models.Discussion
	query := api.db.Model(&models.Discussion{})
	if q := strings.TrimSpace(c.QueryParam("q")); q != "" {
		like := "%" + q + "%"
		query = query.Where("LOWER(title) LIKE LOWER(?) OR LOWER(content) LIKE LOWER(?) OR LOWER(CAST(tags AS TEXT)) LIKE LOWER(?)", like, like, like)
	}
	if tag := c.QueryParam("tags"); tag != "" {
		rawTag, _ := json.Marshal([]string{tag})
		query = query.Where("tags @> ?::jsonb", string(rawTag))
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return err
	}
	if err := query.Session(&gorm.Session{}).
		Select("id", "title", "user_id", "tags", "pinned", "locked", "created_at").
		Order("pinned desc, updated_at desc").
		Limit(pageSize).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return err
	}
	authorIDs := make([]uint, 0, len(rows))
	discussionIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		authorIDs = append(authorIDs, row.UserID)
		discussionIDs = append(discussionIDs, row.ID)
	}
	authors, err := api.userNameMap(authorIDs)
	if err != nil {
		return err
	}
	replies, err := api.discussionReplyCounts(discussionIDs)
	if err != nil {
		return err
	}
	items := make([]DiscussionDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, discussionDTOFromRefs(row, authors, replies))
	}
	return c.JSON(http.StatusOK, PageResult[DiscussionDTO]{Items: items, Page: page, PageSize: pageSize, Total: total})
}

func (api *API) createDiscussion(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	var req DiscussionCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	req.Tags = normalizeTags(req.Tags)
	if req.Title == "" || req.Content == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title and content are required")
	}
	if err := validateTitle(req.Title); err != nil {
		return err
	}
	if err := validateTextBytes(req.Content, utils.MaxMarkdownBytes, "discussion content is too large"); err != nil {
		return err
	}
	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	rawTags, _ := json.Marshal(req.Tags)
	row := models.Discussion{
		Title:   req.Title,
		Content: req.Content,
		UserID:  user.ID,
		Tags:    rawTags,
	}
	if err := api.db.Create(&row).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, CreatedID{ID: row.ID})
}

func (api *API) discussion(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid discussion id")
	}

	var row models.Discussion
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "discussion not found")
		}
		return err
	}
	var comments []models.Comment
	if err := api.db.Select("id", "user_id", "content", "created_at").Where("discussion_id = ?", row.ID).Order("created_at asc").Find(&comments).Error; err != nil {
		return err
	}
	authorIDs := []uint{row.UserID}
	for _, item := range comments {
		authorIDs = append(authorIDs, item.UserID)
	}
	authors, err := api.userNameMap(authorIDs)
	if err != nil {
		return err
	}
	items := make([]CommentDTO, 0, len(comments))
	for _, item := range comments {
		items = append(items, CommentDTO{
			ID:        item.ID,
			Author:    authorName(item.UserID, authors),
			Content:   item.Content,
			CreatedAt: item.CreatedAt,
		})
	}
	dto := DiscussionDTO{
		ID:        row.ID,
		Title:     row.Title,
		Author:    authorName(row.UserID, authors),
		Tags:      readTags([]byte(row.Tags)),
		Pinned:    row.Pinned,
		Locked:    row.Locked,
		Replies:   len(items),
		CreatedAt: row.CreatedAt,
	}
	return c.JSON(http.StatusOK, DiscussionDetail{
		Discussion: dto,
		Content:    row.Content,
		Comments:   items,
	})
}

func (api *API) updateDiscussion(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid discussion id")
	if err != nil {
		return err
	}
	var req DiscussionUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.Title == nil && req.Content == nil && req.Tags == nil && req.Pinned == nil && req.Locked == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "no fields to update")
	}

	var row models.Discussion
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "discussion not found")
		}
		return err
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "title is required")
		}
		if err := validateTitle(title); err != nil {
			return err
		}
		row.Title = title
	}
	if req.Content != nil {
		content := strings.TrimSpace(*req.Content)
		if content == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "content is required")
		}
		if err := validateTextBytes(content, utils.MaxMarkdownBytes, "discussion content is too large"); err != nil {
			return err
		}
		row.Content = content
	}
	if req.Tags != nil {
		rawTags, _ := json.Marshal(normalizeTags(*req.Tags))
		row.Tags = rawTags
	}
	if req.Pinned != nil {
		row.Pinned = *req.Pinned
	}
	if req.Locked != nil {
		row.Locked = *req.Locked
	}
	if err := api.db.Save(&row).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusOK, CreatedID{ID: row.ID})
}

func (api *API) deleteDiscussion(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid discussion id")
	if err != nil {
		return err
	}

	if err := api.db.Delete(&models.Discussion{}, id).Error; err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (api *API) createComment(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid discussion id")
	}
	var req CommentCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "content is required")
	}
	if err := validateTextBytes(req.Content, utils.MaxShortTextBytes, "comment content is too large"); err != nil {
		return err
	}

	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	var discussion models.Discussion
	if err := api.db.First(&discussion, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "discussion not found")
		}
		return err
	}
	if discussion.Locked {
		return echo.NewHTTPError(http.StatusForbidden, "discussion is locked")
	}
	row := models.Comment{DiscussionID: uint(id), UserID: user.ID, Content: req.Content}
	if err := api.db.Create(&row).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, CommentDTO{ID: row.ID, Author: user.Name, Content: row.Content, CreatedAt: row.CreatedAt})
}

func discussionDTOFromRefs(row models.Discussion, authors map[uint]string, replies map[uint]int) DiscussionDTO {
	return DiscussionDTO{
		ID:        row.ID,
		Title:     row.Title,
		Author:    authorName(row.UserID, authors),
		Tags:      readTags([]byte(row.Tags)),
		Pinned:    row.Pinned,
		Locked:    row.Locked,
		Replies:   replies[row.ID],
		CreatedAt: row.CreatedAt,
	}
}

func (api *API) discussionReplyCounts(ids []uint) (map[uint]int, error) {
	ids = uniqueUint(ids)
	counts := map[uint]int{}
	if len(ids) == 0 {
		return counts, nil
	}
	var rows []struct {
		DiscussionID uint
		Count        int64
	}
	if err := api.db.Model(&models.Comment{}).
		Select("discussion_id, count(*) as count").
		Where("discussion_id IN ?", ids).
		Group("discussion_id").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.DiscussionID] = int(row.Count)
	}
	return counts, nil
}

func parseID(c echo.Context, name string, message string) (uint, error) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusBadRequest, message)
	}
	return uint(id), nil
}

func parseQueryID(value string, message string) (uint, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusBadRequest, message)
	}
	return uint(id), nil
}

func parseProblemQuery(value string) (uint, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToUpper(value), "P") {
		value = value[1:]
	}
	return parseQueryID(value, "invalid problem id")
}

func parsePage(c echo.Context) (int, int, int, error) {
	page := 1
	if value := strings.TrimSpace(c.QueryParam("page")); value != "" {
		got, err := strconv.Atoi(value)
		if err != nil || got <= 0 {
			return 0, 0, 0, echo.NewHTTPError(http.StatusBadRequest, "invalid page")
		}
		page = got
	}
	pageSize := 20
	if value := strings.TrimSpace(c.QueryParam("pageSize")); value != "" {
		got, err := strconv.Atoi(value)
		if err != nil || got <= 0 {
			return 0, 0, 0, echo.NewHTTPError(http.StatusBadRequest, "invalid page size")
		}
		pageSize = got
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize, (page - 1) * pageSize, nil
}

func (api *API) listProblems(c echo.Context, limit int) ([]ProblemDTO, error) {
	return api.findProblems(c, "", "", limit, "id desc")
}

func (api *API) currentUser(c echo.Context) (models.User, error) {
	var user models.User

	user, err := utils.UserFromCookie(api.db, c, time.Now())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return user, echo.NewHTTPError(http.StatusUnauthorized, "sign in required")
		}
		return user, err
	}
	return user, nil
}

func (api *API) viewerID(c echo.Context) (uint, error) {
	if api.role(c) == "guest" {
		return 0, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}

func meDTO(user models.User) MeDTO {
	return MeDTO{
		ID:     user.ID,
		Name:   user.Name,
		Mail:   user.Mail,
		Bio:    user.Bio,
		Avatar: user.Avatar,
		Admin:  user.Admin,
	}
}

func langDTO(row models.Language) LanguageDTO {
	return LanguageDTO{ID: row.ID, Name: row.Name, Source: row.Source}
}

func uploadExt(_ string, mime string) string {
	switch mime {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".img"
	}
}

func (api *API) searchProblems(c echo.Context, q string, tag string, limit int) ([]ProblemDTO, error) {
	return api.findProblems(c, q, tag, limit, "id asc")
}

func (api *API) searchProblemPage(c echo.Context, q string, tag string, limit int, offset int) ([]ProblemDTO, int64, error) {
	var rows []models.Problem
	query := api.db.Model(&models.Problem{})
	if !api.isAdmin(c) {
		query = api.applyProblemListVisibility(query)
	}
	if q != "" {
		like := "%" + q + "%"
		if id, err := parseProblemQuery(q); err == nil {
			query = query.Where("id = ? OR LOWER(title) LIKE LOWER(?)", id, like)
		} else {
			query = query.Where("LOWER(title) LIKE LOWER(?)", like)
		}
	}
	if tag != "" {
		rawTag, _ := json.Marshal([]string{tag})
		query = query.Where("tags @> ?::jsonb", string(rawTag))
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Session(&gorm.Session{}).Order("id asc").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]ProblemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, problemDTO(row))
	}
	if err := api.decorateProblemSubmissionStats(items); err != nil {
		return nil, 0, err
	}
	if err := api.decorateProblemMines(c, items); err != nil {
		return nil, 0, err
	}
	if err := api.decorateProblemDiscussions(c.Request().Context(), items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (api *API) searchTags(c echo.Context, kind string, q string, limit int) ([]string, error) {
	kind = strings.TrimSpace(kind)
	q = strings.TrimSpace(q)
	if limit <= 0 {
		limit = 50
	}
	switch kind {
	case "problem":
		return api.searchJSONTags(c, "problems", q, limit)
	case "discussion":
		return api.searchJSONTags(c, "discussions", q, limit)
	default:
		return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid tag kind")
	}
}

func (api *API) searchJSONTags(c echo.Context, table string, q string, limit int) ([]string, error) {
	if api.db.Dialector.Name() == "postgres" {
		sql := fmt.Sprintf("SELECT DISTINCT tag.value AS tag FROM %s CROSS JOIN LATERAL jsonb_array_elements_text(tags) AS tag(value) WHERE %s.deleted_at IS NULL", table, table)
		args := []any{}
		if q != "" {
			sql += " AND LOWER(tag.value) LIKE LOWER(?)"
			args = append(args, "%"+q+"%")
		}
		sql += " ORDER BY tag ASC LIMIT ?"
		args = append(args, limit)
		var rows []struct {
			Tag string
		}
		if err := api.db.WithContext(c.Request().Context()).Raw(sql, args...).Scan(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]string, 0, len(rows))
		for _, row := range rows {
			items = append(items, row.Tag)
		}
		return items, nil
	}

	seen := map[string]bool{}
	items := []string{}
	match := strings.ToLower(q)
	switch table {
	case "problems":
		var rows []models.Problem
		if err := api.db.Select("tags").Order("id asc").Limit(500).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			items = appendMatchingTags(items, seen, readTags([]byte(row.Tags)), match, limit)
			if len(items) >= limit {
				return items, nil
			}
		}
	case "discussions":
		var rows []models.Discussion
		if err := api.db.Select("tags").Order("id asc").Limit(500).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			items = appendMatchingTags(items, seen, readTags([]byte(row.Tags)), match, limit)
			if len(items) >= limit {
				return items, nil
			}
		}
	}
	sort.Strings(items)
	return items, nil
}

func appendMatchingTags(items []string, seen map[string]bool, tags []string, match string, limit int) []string {
	for _, tag := range tags {
		if len(items) >= limit {
			return items
		}
		if seen[tag] || (match != "" && !strings.Contains(strings.ToLower(tag), match)) {
			continue
		}
		seen[tag] = true
		items = append(items, tag)
	}
	return items
}

func (api *API) findProblems(c echo.Context, q string, tag string, limit int, order string) ([]ProblemDTO, error) {
	var rows []models.Problem
	query := api.db.Order(order).Limit(limit)
	if !api.isAdmin(c) {
		query = api.applyProblemListVisibility(query)
	}
	if q != "" {
		like := "%" + q + "%"
		if id, err := parseProblemQuery(q); err == nil {
			query = query.Where("id = ? OR LOWER(title) LIKE LOWER(?)", id, like)
		} else {
			query = query.Where("LOWER(title) LIKE LOWER(?)", like)
		}
	}
	if tag != "" {
		rawTag, _ := json.Marshal([]string{tag})
		query = query.Where("tags @> ?::jsonb", string(rawTag))
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]ProblemDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, problemDTO(row))
	}
	if err := api.decorateProblemSubmissionStats(items); err != nil {
		return nil, err
	}
	if err := api.decorateProblemMines(c, items); err != nil {
		return nil, err
	}
	if err := api.decorateProblemDiscussions(c.Request().Context(), items); err != nil {
		return nil, err
	}
	return items, nil
}

func problemDTO(row models.Problem) ProblemDTO {
	return ProblemDTO{
		ID:          row.ID,
		Title:       row.Title,
		Tags:        readTags([]byte(row.Tags)),
		Visible:     row.Visible,
		Mode:        row.Mode,
		TimeMS:      row.TimeMS,
		MemoryMB:    row.MemoryMB,
		Discussions: 0,
		Mine:        "none",
		Latest:      nil,
	}
}

func (api *API) problemDTOWithStatement(ctx context.Context, row models.Problem) (ProblemDTO, error) {
	item := problemDTO(row)
	statement, err := api.problemStatement(ctx, row.ID, row.Title)
	if err != nil {
		return item, err
	}
	item.Statement = statement
	return item, nil
}

func (api *API) problemStatement(ctx context.Context, id uint, title string) (string, error) {
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return "", err
	}
	reader, _, err := store.Open(ctx, problemStatementKey(id))
	if err != nil {
		return "# " + title, nil
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, utils.MaxMarkdownBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > utils.MaxMarkdownBytes {
		return "", echo.NewHTTPError(http.StatusRequestEntityTooLarge, "statement is too large")
	}
	if strings.TrimSpace(string(data)) == "" {
		return "# " + title, nil
	}
	return string(data), nil
}

func (api *API) writeProblemStatement(ctx context.Context, id uint, statement string) error {
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	if err := validateTextBytes(statement, utils.MaxMarkdownBytes, "statement is too large"); err != nil {
		return err
	}
	body := strings.TrimSpace(statement)
	if body == "" {
		body = "# P" + strconv.Itoa(int(id))
	}
	return store.Put(ctx, problemStatementKey(id), strings.NewReader(body), int64(len(body)), "text/markdown; charset=utf-8")
}

func problemStatementKey(id uint) string {
	return fmt.Sprintf("problems/%d/statement.md", id)
}

func (api *API) decorateProblemDiscussions(ctx context.Context, items []ProblemDTO) error {
	if len(items) == 0 {
		return nil
	}
	counts, err := api.problemDiscussionCounts(ctx)
	if err != nil {
		return err
	}

	for index := range items {
		items[index].Discussions = counts[items[index].ID]
	}
	return nil
}

func (api *API) problemDiscussionCounts(ctx context.Context) (map[uint]int, error) {
	counts := map[uint]int{}
	found, err := utils.CacheGet(ctx, problemDiscussionsCacheKey(), &counts)
	if err == nil && found {
		return counts, nil
	}
	var rows []models.Discussion
	if err := api.db.Select("id", "tags", "pinned", "locked").Order("updated_at desc").Limit(1000).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		item := DiscussionDTO{ID: row.ID, Tags: readTags([]byte(row.Tags)), Pinned: row.Pinned, Locked: row.Locked}
		for _, problemID := range discussionProblemIDs(item) {
			counts[problemID]++
		}
	}
	_ = utils.CacheSet(ctx, problemDiscussionsCacheKey(), counts, 10*time.Second)
	return counts, nil
}

func problemDiscussionsCacheKey() string {
	return "doj:problem:discussions"
}

func (api *API) decorateProblemStats(ctx context.Context, items []ProblemDTO) error {
	if len(items) == 0 {
		return nil
	}
	if err := api.decorateProblemSubmissionStats(items); err != nil {
		return err
	}
	return api.decorateProblemAssetStats(ctx, items)
}

func (api *API) decorateProblemSubmissionStats(items []ProblemDTO) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	submitByProblem, acByProblem, err := api.problemSubmissionStats(ids)
	if err != nil {
		return err
	}
	for index := range items {
		id := items[index].ID
		items[index].Submit = submitByProblem[id]
		items[index].AC = acByProblem[id]
	}
	return nil
}

func (api *API) problemSubmissionStats(ids []uint) (map[uint]int, map[uint]int, error) {
	var submits []struct {
		ProblemID uint
		Count     int64
	}
	if err := api.db.Model(&models.Submission{}).
		Select("problem_id, count(*) AS count").
		Where("problem_id IN ?", ids).
		Group("problem_id").
		Find(&submits).Error; err != nil {
		return nil, nil, err
	}
	submitByProblem := map[uint]int{}
	for _, item := range submits {
		submitByProblem[item.ProblemID] = int(item.Count)
	}
	var acs []struct {
		ProblemID uint
		Count     int64
	}
	if err := api.db.Model(&models.Submission{}).
		Select("problem_id, count(DISTINCT user_id) AS count").
		Where("problem_id IN ? AND status = ?", ids, "AC").
		Group("problem_id").
		Find(&acs).Error; err != nil {
		return nil, nil, err
	}
	acByProblem := map[uint]int{}
	for _, item := range acs {
		acByProblem[item.ProblemID] = int(item.Count)
	}
	return submitByProblem, acByProblem, nil
}

func (api *API) decorateProblemAssetStats(ctx context.Context, items []ProblemDTO) error {
	if len(items) == 0 {
		return nil
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	for index := range items {
		id := items[index].ID
		assets, err := api.problemAssetsCached(ctx, id, store)
		if err != nil {
			return err
		}
		cases := assets.Cases
		dataBytes := assets.DataBytes
		items[index].Cases = &cases
		items[index].DataBytes = &dataBytes
	}
	return nil
}

func (api *API) decorateProblemMines(c echo.Context, items []ProblemDTO) error {
	if len(items) == 0 {
		return nil
	}
	for index := range items {
		items[index].Mine = "none"
	}
	if api.role(c) == "guest" {
		return nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	var rows []struct {
		ProblemID uint
		ID        uint
		Status    string
		Score     int
		CreatedAt time.Time
	}
	if err := api.db.Model(&models.Submission{}).
		Select("problem_id, id, status, score, created_at").
		Where("user_id = ? AND problem_id IN ?", user.ID, ids).
		Order("created_at desc").
		Find(&rows).Error; err != nil {
		return err
	}
	mine := map[uint]string{}
	latest := map[uint]RecordDTO{}
	for _, row := range rows {
		if _, ok := latest[row.ProblemID]; !ok {
			latest[row.ProblemID] = RecordDTO{ID: row.ID, Status: row.Status, Score: row.Score, CreatedAt: row.CreatedAt}
		}
		if row.Status == "AC" {
			mine[row.ProblemID] = "ac"
			continue
		}
		if mine[row.ProblemID] == "" {
			mine[row.ProblemID] = "tried"
		}
	}
	for index := range items {
		items[index].Mine = mine[items[index].ID]
		if items[index].Mine == "" {
			items[index].Mine = "none"
		}
		if item, ok := latest[items[index].ID]; ok {
			items[index].Latest = &item
		}
	}
	return nil
}

func (api *API) decorateProblemMinesInContest(c echo.Context, items []ProblemDTO, contestID uint) error {
	if len(items) == 0 {
		return nil
	}
	for index := range items {
		items[index].Mine = "none"
	}
	if api.role(c) == "guest" {
		return nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	var rows []struct {
		ProblemID uint
		ID        uint
		Status    string
		Score     int
		CreatedAt time.Time
	}
	if err := api.db.Model(&models.Submission{}).
		Select("problem_id, id, status, score, created_at").
		Where("user_id = ? AND contest_id = ? AND problem_id IN ?", user.ID, contestID, ids).
		Order("created_at desc").
		Find(&rows).Error; err != nil {
		return err
	}
	mine := map[uint]string{}
	latest := map[uint]RecordDTO{}
	for _, row := range rows {
		if _, ok := latest[row.ProblemID]; !ok {
			latest[row.ProblemID] = RecordDTO{ID: row.ID, Status: row.Status, Score: row.Score, CreatedAt: row.CreatedAt}
		}
		if row.Status == "AC" {
			mine[row.ProblemID] = "ac"
			continue
		}
		if mine[row.ProblemID] == "" {
			mine[row.ProblemID] = "tried"
		}
	}
	for index := range items {
		items[index].Mine = mine[items[index].ID]
		if items[index].Mine == "" {
			items[index].Mine = "none"
		}
		if item, ok := latest[items[index].ID]; ok {
			items[index].Latest = &item
		}
	}
	return nil
}

func (api *API) requireProblemAdmin(c echo.Context) (uint, error) {
	if err := api.requireAdmin(c); err != nil {
		return 0, err
	}
	id, err := parseID(c, "id", "invalid problem id")
	if err != nil {
		return 0, err
	}

	var count int64
	if err := api.db.Model(&models.Problem{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, echo.NewHTTPError(http.StatusNotFound, "problem not found")
	}
	return id, nil
}

func (api *API) requireProblemVisible(c echo.Context, id uint) error {
	if api.isAdmin(c) {
		return nil
	}

	var problem models.Problem
	if err := api.db.First(&problem, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	if !api.problemVisibleInDetail(problem) {
		return echo.NewHTTPError(http.StatusNotFound, "problem not found")
	}
	return nil
}

func (api *API) problemVisibleInDetail(problem models.Problem) bool {
	if api.problemVisibleInList(problem) {
		return true
	}
	return api.problemInRunningContest(problem.ID)
}

func (api *API) problemVisibleInList(problem models.Problem) bool {
	if !problem.Visible {
		return false
	}
	return !api.problemInUnfinishedContest(problem.ID)
}

func (api *API) applyProblemListVisibility(query *gorm.DB) *gorm.DB {
	now := time.Now()
	return query.Where(
		`problems.visible = ? AND NOT EXISTS (
			SELECT 1 FROM contest_problems
			JOIN contests ON contests.id = contest_problems.contest_id
			WHERE contest_problems.problem_id = problems.id AND contests.end_at > ?
		)`,
		true,
		now,
	)
}

func (api *API) problemInUnfinishedContest(problemID uint) bool {
	var count int64
	err := api.db.Model(&models.ContestProblem{}).
		Joins("JOIN contests ON contests.id = contest_problems.contest_id").
		Where("contest_problems.problem_id = ? AND contests.end_at > ?", problemID, time.Now()).
		Count(&count).Error
	return err == nil && count > 0
}

func (api *API) problemInRunningContest(problemID uint) bool {
	var count int64
	now := time.Now()
	err := api.db.Model(&models.ContestProblem{}).
		Joins("JOIN contests ON contests.id = contest_problems.contest_id").
		Where("contest_problems.problem_id = ? AND contests.start_at <= ? AND contests.end_at > ?", problemID, now, now).
		Count(&count).Error
	return err == nil && count > 0
}

func (api *API) syncProblemAssets(c echo.Context, id uint) (ProblemAssets, error) {
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return ProblemAssets{}, err
	}
	assets, err := problemAssetsFromStore(c.Request().Context(), id, store)
	if err != nil {
		return ProblemAssets{}, err
	}
	api.cacheProblemAssets(c.Request().Context(), id, assets)
	return assets, nil
}

func (api *API) problemAssetsCached(ctx context.Context, id uint, store utils.ObjectStore) (ProblemAssets, error) {
	var cached ProblemAssets
	found, err := utils.CacheGet(ctx, problemAssetsCacheKey(id), &cached)
	if err == nil && found {
		return cached, nil
	}
	assets, err := problemAssetsFromStore(ctx, id, store)
	if err != nil {
		return ProblemAssets{}, err
	}
	api.cacheProblemAssets(ctx, id, assets)
	return assets, nil
}

func (api *API) cacheProblemAssets(ctx context.Context, id uint, assets ProblemAssets) {
	_ = utils.CacheSet(ctx, problemAssetsCacheKey(id), assets, time.Minute)
}

func problemAssetsCacheKey(id uint) string {
	return "doj:problem:" + strconv.FormatUint(uint64(id), 10) + ":assets"
}

func clearProblemPackageCacheIfNeeded(ctx context.Context, id uint, key string) {
	data := problemAssetPrefix(id, "data") + "/"
	judge := problemAssetPrefix(id, "judge") + "/"
	if strings.HasPrefix(key, data) || strings.HasPrefix(key, judge) {
		clearProblemPackageCache(ctx, id)
	}
}

func clearProblemPackageCache(ctx context.Context, id uint) {
	_ = utils.CacheDelete(ctx, utils.ProblemPackageCacheKey(id))
}

func cleanEditableAssetKey(id uint, raw string) (string, error) {
	key, err := utils.CleanObjectKey(raw)
	if err != nil || !problemAssetKeyAllowed(id, key) {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid asset key")
	}
	if !editableAssetName(key) {
		return "", echo.NewHTTPError(http.StatusBadRequest, "asset is not editable")
	}
	return key, nil
}

func problemAssetsFromStore(ctx context.Context, id uint, store utils.ObjectStore) (ProblemAssets, error) {
	data, err := assetFiles(ctx, store, problemAssetPrefix(id, "data"))
	if err != nil {
		return ProblemAssets{}, err
	}
	judge, err := assetFiles(ctx, store, problemAssetPrefix(id, "judge"))
	if err != nil {
		return ProblemAssets{}, err
	}
	assets, err := assetFiles(ctx, store, problemAssetPrefix(id, "assets"))
	if err != nil {
		return ProblemAssets{}, err
	}
	cases, dataBytes := dataStats(data)
	return ProblemAssets{Data: data, Judge: judge, Assets: assets, Cases: cases, DataBytes: dataBytes}, nil
}

func writeProblemStatementZipFile(writer *zip.Writer, statement string) error {
	file, err := writer.CreateHeader(&zip.FileHeader{Name: "statement.md", Method: zip.Deflate})
	if err != nil {
		return err
	}
	_, err = io.WriteString(file, statement)
	return err
}

func writeAssetZipFiles(ctx context.Context, writer *zip.Writer, store utils.ObjectStore, section string, files []AssetFile) error {
	for _, item := range files {
		zipName, ok := safeAssetZipName(section, item.Name)
		if !ok {
			continue
		}
		reader, _, err := store.Open(ctx, item.Key)
		if err != nil {
			return err
		}
		header := &zip.FileHeader{Name: zipName, Method: zip.Deflate}
		file, err := writer.CreateHeader(header)
		if err != nil {
			_ = reader.Close()
			return err
		}
		if _, err := io.Copy(file, reader); err != nil {
			_ = reader.Close()
			return err
		}
		if err := reader.Close(); err != nil {
			return err
		}
	}
	return nil
}

func safeAssetZipName(section string, name string) (string, bool) {
	normalized := strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
	clean, err := utils.CleanObjectKey(normalized)
	if err != nil || clean != normalized {
		return "", false
	}
	return path.Join(section, clean), true
}

func assetFiles(ctx context.Context, store utils.ObjectStore, prefix string) ([]AssetFile, error) {
	objects, err := store.List(ctx, prefix)
	if err != nil {
		return nil, err
	}
	fullPrefix := strings.TrimSuffix(prefix, "/") + "/"
	items := make([]AssetFile, 0, len(objects))
	for _, object := range objects {
		if !strings.HasPrefix(object.Key, fullPrefix) {
			continue
		}
		name := strings.TrimPrefix(object.Key, fullPrefix)
		if name == "" {
			continue
		}
		items = append(items, AssetFile{
			Key:      object.Key,
			Name:     name,
			Size:     object.Size,
			Editable: editableAsset(name, object.Size),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func dataStats(files []AssetFile) (int, int64) {
	inputs := map[string]bool{}
	outputs := map[string]bool{}
	var bytes int64
	for _, file := range files {
		bytes += file.Size
		stem, kind := utils.DataCaseStem(file.Name)
		switch kind {
		case "in":
			inputs[stem] = true
		case "out":
			outputs[stem] = true
		}
	}
	cases := 0
	for stem := range inputs {
		if outputs[stem] {
			cases++
		}
	}
	return cases, bytes
}

func editableAsset(name string, size int64) bool {
	if size > maxEditableAssetBytes {
		return false
	}
	return editableAssetName(name)
}

func editableAssetName(name string) bool {
	switch strings.ToLower(path.Base(name)) {
	case "dockerfile", "makefile":
		return true
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".c", ".cc", ".cpp", ".cxx", ".go", ".rs", ".py", ".java", ".js", ".ts", ".txt", ".md", ".json", ".yaml", ".yml", ".toml", ".in", ".out":
		return true
	default:
		return false
	}
}

func problemAssetPrefix(id uint, section string) string {
	return fmt.Sprintf("problems/%d/%s", id, section)
}

func problemAssetKeyAllowed(id uint, key string) bool {
	data := problemAssetPrefix(id, "data") + "/"
	judge := problemAssetPrefix(id, "judge") + "/"
	assets := problemAssetPrefix(id, "assets") + "/"
	return strings.HasPrefix(key, data) || strings.HasPrefix(key, judge) || strings.HasPrefix(key, assets)
}

func assetSection(raw string) (string, error) {
	switch strings.TrimSpace(raw) {
	case "data", "judge", "assets":
		return strings.TrimSpace(raw), nil
	default:
		return "", echo.NewHTTPError(http.StatusBadRequest, "asset section must be data, judge or assets")
	}
}

func cleanAssetName(raw string) (string, error) {
	name := path.Base(strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/"))
	if name == "" || name == "." || name == ".." {
		return "", echo.NewHTTPError(http.StatusBadRequest, "asset file name is required")
	}
	if _, err := utils.CleanObjectKey(name); err != nil {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid asset file name")
	}
	return name, nil
}

func caseName(raw string, assets ProblemAssets) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		name = nextCaseName(assets)
	}
	name = strings.TrimSuffix(strings.TrimSuffix(name, ".in"), ".out")
	var out []rune
	for _, char := range name {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-' || char == '_' {
			out = append(out, char)
		}
	}
	if len(out) == 0 {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid case name")
	}
	return string(out), nil
}

func nextCaseName(assets ProblemAssets) string {
	used := map[string]bool{}
	for _, file := range assets.Data {
		stem, kind := utils.DataCaseStem(file.Name)
		if kind != "" {
			used[stem] = true
		}
	}
	for i := 1; ; i++ {
		name := strconv.Itoa(i)
		if !used[name] {
			return name
		}
	}
}

func judgeTemplateFiles() map[string]string {
	return map[string]string{
		"Dockerfile": `FROM gcc:14
WORKDIR /src
COPY main.cc .
RUN g++ -O2 -pipe -static -s main.cc -o /out/judge
`,
		"main.cc": `#include <bits/stdc++.h>
using namespace std;

static string read_file(const char* path) {
  ifstream file(path, ios::binary);
  return string((istreambuf_iterator<char>(file)), istreambuf_iterator<char>());
}

static string trim_right(string s) {
  while (!s.empty() && (s.back() == '\n' || s.back() == '\r' || s.back() == ' ' || s.back() == '\t')) {
    s.pop_back();
  }
  return s;
}

int main() {
  const char* inputPath = getenv("INPUT");
  const char* answerPath = getenv("ANSWER");
  const char* resultPath = getenv("RESULT");
  if (!inputPath || !answerPath || !resultPath) return 2;

  cout << read_file(inputPath);
  cout.flush();
  fclose(stdout);

  string expected = read_file(answerPath);
  string output((istreambuf_iterator<char>(cin)), istreambuf_iterator<char>());

  bool ok = trim_right(expected) == trim_right(output);

  ofstream result(resultPath);
  result << "{\"verdict\":\"" << (ok ? "AC" : "WA") << "\",\"score\":" << (ok ? 100 : 0)
         << ",\"message\":\"" << (ok ? "" : "output differs") << "\"}\n";
  return 0;
}
`,
	}
}

func (api *API) assignmentDTO(c echo.Context, row models.Assignment, total int, done int) (AssignmentDTO, error) {
	members := assignmentMembersDTO{}
	admin := api.isAdmin(c)
	if admin {
		users, groups, err := api.assignmentMembers(row.ID)
		if err != nil {
			return AssignmentDTO{}, err
		}
		members = assignmentMembersDTO{Users: users, Groups: groups}
	}
	return assignmentDTOFromParts(row, total, done, members, admin), nil
}

type assignmentMembersDTO struct {
	Users  []uint
	Groups []uint
}

func (api *API) assignmentDTOs(c echo.Context, rows []models.Assignment, includeMembers bool) ([]AssignmentDTO, error) {
	if len(rows) == 0 {
		return []AssignmentDTO{}, nil
	}
	ids := assignmentIDs(rows)
	visible, err := api.assignmentVisibleMap(c, ids)
	if err != nil {
		return nil, err
	}
	totals, err := api.assignmentTotalMap(ids)
	if err != nil {
		return nil, err
	}
	done, err := api.assignmentDoneMap(c, ids)
	if err != nil {
		return nil, err
	}
	admin := api.isAdmin(c)
	includeMemberFields := includeMembers && admin
	members := map[uint]assignmentMembersDTO{}
	if includeMemberFields {
		members, err = api.assignmentMembersMap(ids)
		if err != nil {
			return nil, err
		}
	}
	items := make([]AssignmentDTO, 0, len(rows))
	for _, row := range rows {
		if !visible[row.ID] {
			continue
		}
		total := totals[row.ID]
		if total == 0 {
			continue
		}
		items = append(items, assignmentDTOFromParts(row, total, done[row.ID], members[row.ID], includeMemberFields))
	}
	return items, nil
}

func assignmentDTOFromParts(row models.Assignment, total int, done int, members assignmentMembersDTO, includeMembers bool) AssignmentDTO {
	dto := AssignmentDTO{
		ID:     row.ID,
		Title:  row.Title,
		EndAt:  row.EndAt,
		Status: assignmentStatus(row),
		Total:  total,
		Done:   done,
	}
	if includeMembers {
		dto.Users = cleanUintList(members.Users)
		if dto.Users == nil {
			dto.Users = []uint{}
		}
		dto.Groups = cleanUintList(members.Groups)
		if dto.Groups == nil {
			dto.Groups = []uint{}
		}
	}
	return dto
}

func assignmentStatus(row models.Assignment) string {
	if row.EndAt.Before(time.Now()) {
		return "ended"
	}
	return "running"
}

func assignmentIDs(rows []models.Assignment) []uint {
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func (api *API) assignmentVisibleMap(c echo.Context, ids []uint) (map[uint]bool, error) {
	ids = uniqueUint(ids)
	visible := map[uint]bool{}
	if len(ids) == 0 {
		return visible, nil
	}
	if api.isAdmin(c) {
		for _, id := range ids {
			visible[id] = true
		}
		return visible, nil
	}
	if api.role(c) == "guest" {
		return visible, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return nil, err
	}
	var direct []models.AssignmentUser
	if err := api.db.Where("assignment_id IN ? AND user_id = ?", ids, user.ID).Find(&direct).Error; err != nil {
		return nil, err
	}
	for _, row := range direct {
		visible[row.AssignmentID] = true
	}
	var grouped []struct {
		AssignmentID uint
	}
	if err := api.db.Model(&models.AssignmentGroup{}).
		Select("assignment_groups.assignment_id").
		Joins("JOIN group_users ON group_users.group_id = assignment_groups.group_id").
		Where("assignment_groups.assignment_id IN ? AND group_users.user_id = ?", ids, user.ID).
		Find(&grouped).Error; err != nil {
		return nil, err
	}
	for _, row := range grouped {
		visible[row.AssignmentID] = true
	}
	return visible, nil
}

func (api *API) assignmentMembers(id uint) ([]uint, []uint, error) {
	var users []models.AssignmentUser
	if err := api.db.Where("assignment_id = ?", id).Order("user_id asc").Find(&users).Error; err != nil {
		return nil, nil, err
	}
	var groups []models.AssignmentGroup
	if err := api.db.Where("assignment_id = ?", id).Order("group_id asc").Find(&groups).Error; err != nil {
		return nil, nil, err
	}
	userIDs := make([]uint, 0, len(users))
	for _, row := range users {
		userIDs = append(userIDs, row.UserID)
	}
	groupIDs := make([]uint, 0, len(groups))
	for _, row := range groups {
		groupIDs = append(groupIDs, row.GroupID)
	}
	return userIDs, groupIDs, nil
}

func (api *API) assignmentMembersMap(ids []uint) (map[uint]assignmentMembersDTO, error) {
	ids = uniqueUint(ids)
	members := map[uint]assignmentMembersDTO{}
	if len(ids) == 0 {
		return members, nil
	}
	var users []models.AssignmentUser
	if err := api.db.Where("assignment_id IN ?", ids).Order("user_id asc").Find(&users).Error; err != nil {
		return nil, err
	}
	for _, row := range users {
		item := members[row.AssignmentID]
		item.Users = append(item.Users, row.UserID)
		members[row.AssignmentID] = item
	}
	var groups []models.AssignmentGroup
	if err := api.db.Where("assignment_id IN ?", ids).Order("group_id asc").Find(&groups).Error; err != nil {
		return nil, err
	}
	for _, row := range groups {
		item := members[row.AssignmentID]
		item.Groups = append(item.Groups, row.GroupID)
		members[row.AssignmentID] = item
	}
	return members, nil
}

func (api *API) assignmentProblemCount(id uint) (int, error) {
	var count int64
	query := api.db.Model(&models.AssignmentProblem{}).Where("assignment_id = ?", id)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (api *API) assignmentTotalMap(ids []uint) (map[uint]int, error) {
	ids = uniqueUint(ids)
	totals := map[uint]int{}
	if len(ids) == 0 {
		return totals, nil
	}
	var rows []struct {
		AssignmentID uint
		Count        int64
	}
	if err := api.db.Model(&models.AssignmentProblem{}).
		Select("assignment_id, count(*) as count").
		Where("assignment_id IN ?", ids).
		Group("assignment_id").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		totals[row.AssignmentID] = int(row.Count)
	}
	return totals, nil
}

func (api *API) assignmentDoneCount(c echo.Context, id uint) (int, error) {
	if api.role(c) == "guest" {
		return 0, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return 0, err
	}
	var rows []struct {
		ProblemID uint
	}
	query := api.db.Model(&models.Submission{}).
		Select("DISTINCT submissions.problem_id").
		Joins("JOIN assignment_problems ON assignment_problems.problem_id = submissions.problem_id").
		Where("assignment_problems.assignment_id = ? AND submissions.assignment_id = ? AND submissions.user_id = ? AND submissions.status = ?", id, id, user.ID, "AC")
	if err := query.Find(&rows).Error; err != nil {
		return 0, err
	}
	return len(rows), nil
}

func (api *API) assignmentDoneMap(c echo.Context, ids []uint) (map[uint]int, error) {
	ids = uniqueUint(ids)
	done := map[uint]int{}
	if len(ids) == 0 || api.role(c) == "guest" {
		return done, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		AssignmentID uint
		Count        int64
	}
	if err := api.db.Model(&models.Submission{}).
		Select("submissions.assignment_id, count(DISTINCT submissions.problem_id) as count").
		Joins("JOIN assignment_problems ON assignment_problems.assignment_id = submissions.assignment_id AND assignment_problems.problem_id = submissions.problem_id").
		Where("submissions.assignment_id IN ? AND submissions.user_id = ? AND submissions.status = ?", ids, user.ID, "AC").
		Group("submissions.assignment_id").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		done[row.AssignmentID] = int(row.Count)
	}
	return done, nil
}

func (api *API) assignmentProblems(c echo.Context, links []models.AssignmentProblem) ([]ProblemDTO, error) {
	if len(links) == 0 {
		return []ProblemDTO{}, nil
	}
	ids := make([]uint, 0, len(links))
	for _, link := range links {
		ids = append(ids, link.ProblemID)
	}
	query := api.db.Model(&models.Problem{}).Where("id IN ?", uniqueUint(ids))
	if !api.isAdmin(c) {
		query = api.applyProblemListVisibility(query)
	}
	var rows []models.Problem
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	byID := problemRowsByID(rows)
	items := make([]ProblemDTO, 0, len(links))
	for _, link := range links {
		problem, ok := byID[link.ProblemID]
		if !ok {
			continue
		}
		item := problemDTO(problem)
		item.Sort = link.Sort
		items = append(items, item)
	}
	return items, nil
}

func contestDTO(row models.Contest, total int) ContestDTO {
	freezeAt := row.FreezeAt
	if row.Kind == "OI" {
		freezeAt = nil
	}
	return ContestDTO{
		ID:       row.ID,
		Title:    row.Title,
		Kind:     row.Kind,
		StartAt:  row.StartAt,
		EndAt:    row.EndAt,
		FreezeAt: freezeAt,
		Status:   contestStatus(row),
		Total:    total,
	}
}

func (api *API) contestDTOs(rows []models.Contest, admin bool) ([]ContestDTO, error) {
	if len(rows) == 0 {
		return []ContestDTO{}, nil
	}
	totals, err := api.contestTotalMap(contestIDs(rows), admin)
	if err != nil {
		return nil, err
	}
	items := make([]ContestDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, contestDTO(row, totals[row.ID]))
	}
	return items, nil
}

func contestIDs(rows []models.Contest) []uint {
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func (api *API) contestTotalMap(ids []uint, admin bool) (map[uint]int, error) {
	ids = uniqueUint(ids)
	totals := map[uint]int{}
	if len(ids) == 0 {
		return totals, nil
	}
	var rows []struct {
		ContestID uint
		Count     int64
	}
	query := api.db.Model(&models.ContestProblem{}).
		Select("contest_problems.contest_id, count(*) as count").
		Where("contest_problems.contest_id IN ?", ids)
	if !admin {
		query = query.
			Joins("JOIN contests ON contests.id = contest_problems.contest_id").
			Joins("JOIN problems ON problems.id = contest_problems.problem_id AND problems.deleted_at IS NULL").
			Where("contests.end_at > ? OR problems.visible = ?", time.Now(), true)
	}
	if err := query.Group("contest_problems.contest_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		totals[row.ContestID] = int(row.Count)
	}
	return totals, nil
}

func (api *API) contestProblems(c echo.Context, contest models.Contest, links []models.ContestProblem) ([]ProblemDTO, error) {
	if len(links) == 0 {
		return []ProblemDTO{}, nil
	}
	admin := api.isAdmin(c)
	if !admin && !contestRunning(contest) && !contestEnded(contest) {
		return []ProblemDTO{}, nil
	}
	ids := make([]uint, 0, len(links))
	for _, link := range links {
		ids = append(ids, link.ProblemID)
	}
	query := api.db.Model(&models.Problem{}).Where("id IN ?", uniqueUint(ids))
	if !admin && contestEnded(contest) {
		query = query.Where("visible = ?", true)
	}
	var rows []models.Problem
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	byID := problemRowsByID(rows)
	items := make([]ProblemDTO, 0, len(links))
	for _, link := range links {
		problem, ok := byID[link.ProblemID]
		if !ok {
			continue
		}
		item := problemDTO(problem)
		item.Sort = link.Sort
		items = append(items, item)
	}
	return items, nil
}

func problemRowsByID(rows []models.Problem) map[uint]models.Problem {
	byID := make(map[uint]models.Problem, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	return byID
}

func contestStatus(row models.Contest) string {
	now := time.Now()
	status := "running"
	if row.StartAt.After(now) {
		status = "pending"
	}
	if row.EndAt.Before(now) {
		status = "ended"
	}
	if status == "running" && contestFrozenForUser(row, false) {
		status = "frozen"
	}
	return status
}

func contestFrozenForUser(row models.Contest, admin bool) bool {
	if admin || row.Kind == "OI" || row.FreezeAt == nil {
		return false
	}
	now := time.Now()
	return !now.Before(*row.FreezeAt) && now.Before(row.EndAt)
}

func (api *API) contestFreezeCutoff(c echo.Context, row models.Contest) *time.Time {
	if contestFrozenForUser(row, api.isAdmin(c)) {
		return row.FreezeAt
	}
	return nil
}

func contestRunning(row models.Contest) bool {
	now := time.Now()
	return !now.Before(row.StartAt) && now.Before(row.EndAt)
}

func contestEnded(row models.Contest) bool {
	return !time.Now().Before(row.EndAt)
}

func (api *API) submissionDTO(row models.Submission) SubmissionDTO {
	items, err := api.submissionDTOs([]models.Submission{row})
	if err == nil && len(items) > 0 {
		return items[0]
	}
	return submissionDTOFromRefs(row, nil, nil)
}

func (api *API) submissionListItems(rows []models.Submission) ([]SubmissionListItem, error) {
	items, err := api.submissionDTOs(rows)
	if err != nil {
		return nil, err
	}
	list := make([]SubmissionListItem, 0, len(items))
	for _, item := range items {
		list = append(list, submissionListItemFromDTO(item))
	}
	return list, nil
}

func (api *API) submissionDTOs(rows []models.Submission) ([]SubmissionDTO, error) {
	if len(rows) == 0 {
		return []SubmissionDTO{}, nil
	}
	problemIDs := make([]uint, 0, len(rows))
	userIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		problemIDs = append(problemIDs, row.ProblemID)
		userIDs = append(userIDs, row.UserID)
	}
	titles, err := api.problemTitleMap(problemIDs)
	if err != nil {
		return nil, err
	}
	users, err := api.userNameMap(userIDs)
	if err != nil {
		return nil, err
	}
	items := make([]SubmissionDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, submissionDTOFromRefs(row, titles, users))
	}
	return items, nil
}

func submissionListItemFromDTO(row SubmissionDTO) SubmissionListItem {
	return SubmissionListItem{
		ID:           row.ID,
		ProblemID:    row.ProblemID,
		ProblemTitle: row.ProblemTitle,
		User:         row.User,
		Language:     row.Language,
		Status:       row.Status,
		TimeMS:       row.TimeMS,
		MemoryKB:     row.MemoryKB,
		CreatedAt:    row.CreatedAt,
	}
}

func submissionDTOFromRefs(row models.Submission, titles map[uint]string, users map[uint]string) SubmissionDTO {
	title := titles[row.ProblemID]
	userName := users[row.UserID]
	if title == "" {
		title = "P" + strconv.Itoa(int(row.ProblemID))
	}
	if userName == "" {
		userName = strconv.Itoa(int(row.UserID))
	}
	return SubmissionDTO{
		ID:           row.ID,
		ProblemID:    row.ProblemID,
		ProblemTitle: title,
		User:         userName,
		Language:     row.Language,
		Status:       row.Status,
		Score:        row.Score,
		Message:      row.Message,
		TimeMS:       row.TimeMS,
		MemoryKB:     row.MemoryKB,
		Public:       row.Public,
		CreatedAt:    row.CreatedAt,
	}
}

func (api *API) problemTitleMap(ids []uint) (map[uint]string, error) {
	ids = uniqueUint(ids)
	titles := map[uint]string{}
	if len(ids) == 0 {
		return titles, nil
	}
	var rows []models.Problem
	if err := api.db.Select("id", "title").Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		titles[row.ID] = row.Title
	}
	return titles, nil
}

func (api *API) userNameMap(ids []uint) (map[uint]string, error) {
	ids = uniqueUint(ids)
	names := map[uint]string{}
	if len(ids) == 0 {
		return names, nil
	}
	var rows []models.User
	if err := api.db.Select("id", "name").Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		names[row.ID] = row.Name
	}
	return names, nil
}

func (api *API) notice() string {

	settings, err := adminsvc.Settings(api.db)
	if err != nil {
		return ""
	}
	return settings.Notice
}

func readTags(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var tags []string
	if err := json.Unmarshal(raw, &tags); err != nil {
		return nil
	}
	return tags
}

func filterProblems(items []ProblemDTO, q string, tag string) []ProblemDTO {
	if q == "" && tag == "" {
		return items
	}
	filtered := make([]ProblemDTO, 0, len(items))
	for _, item := range items {
		if q != "" && !containsFold(item.Title, q) {
			continue
		}
		if tag != "" && !hasTag(item.Tags, tag) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func visibleProblems(items []ProblemDTO) []ProblemDTO {
	filtered := make([]ProblemDTO, 0, len(items))
	for _, item := range items {
		if item.Visible {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func containsFold(value string, needle string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(needle))
}

func hasTag(tags []string, tag string) bool {
	for _, item := range tags {
		if item == tag {
			return true
		}
	}
	return false
}

func normalizeTags(tags []string) []string {
	seen := map[string]bool{}
	clean := make([]string, 0, len(tags))
	for _, tag := range tags {
		for _, part := range strings.FieldsFunc(tag, func(r rune) bool {
			return r == ',' || r == '，' || r == ' ' || r == '\n' || r == '\t'
		}) {
			part = strings.TrimSpace(part)
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			clean = append(clean, part)
		}
	}
	return clean
}

func validProblemMode(mode string) bool {
	return mode == "default" || mode == "strict" || mode == "custom"
}

func validContestKind(kind string) bool {
	return kind == "OI" || kind == "ICPC"
}

func parseContestFreezeAt(raw string, startAt time.Time, endAt time.Time) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	freezeAt, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid freeze time")
	}
	if freezeAt.Before(startAt) || !freezeAt.Before(endAt) {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "freeze time must be between start and end")
	}
	return &freezeAt, nil
}

func problemSort(index int) string {
	if index >= 0 && index < 26 {
		return string(rune('A' + index))
	}
	return strconv.Itoa(index + 1)
}

func normalizeProblemRefs(items []ProblemRef) []ProblemRef {
	for index := range items {
		items[index].Sort = strings.TrimSpace(items[index].Sort)
		if items[index].Sort == "" {
			items[index].Sort = problemSort(index)
		}
	}
	return items
}

func (api *API) validateProblemRefs(items []ProblemRef) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(items))
	seen := make(map[uint]bool, len(items))
	for _, item := range items {
		if item.ID == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid problem id")
		}
		if seen[item.ID] {
			return echo.NewHTTPError(http.StatusBadRequest, "duplicate problem id")
		}
		if len([]rune(item.Sort)) > models.SortMax {
			return echo.NewHTTPError(http.StatusBadRequest, "problem sort is too long")
		}
		seen[item.ID] = true
		ids = append(ids, item.ID)
	}
	var count int64
	if err := api.db.Model(&models.Problem{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return echo.NewHTTPError(http.StatusBadRequest, "problem not found")
	}
	return nil
}

func (api *API) validateUserIDs(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uint]bool, len(ids))
	for _, id := range ids {
		if id == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
		}
		if seen[id] {
			return echo.NewHTTPError(http.StatusBadRequest, "duplicate user id")
		}
		seen[id] = true
	}
	var count int64
	if err := api.db.Model(&models.User{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return echo.NewHTTPError(http.StatusBadRequest, "user not found")
	}
	return nil
}

func (api *API) validateGroupIDs(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uint]bool, len(ids))
	for _, id := range ids {
		if id == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
		}
		if seen[id] {
			return echo.NewHTTPError(http.StatusBadRequest, "duplicate group id")
		}
		seen[id] = true
	}
	var count int64
	if err := api.db.Model(&models.Group{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return echo.NewHTTPError(http.StatusBadRequest, "group not found")
	}
	return nil
}

func cleanUintList(values []uint) []uint {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[uint]bool, len(values))
	items := make([]uint, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		items = append(items, value)
	}
	return items
}

func uniqueUint(values []uint) []uint {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[uint]bool, len(values))
	items := make([]uint, 0, len(values))
	for _, value := range values {
		if value == 0 || seen[value] {
			continue
		}
		seen[value] = true
		items = append(items, value)
	}
	return items
}

func validateRegister(req RegisterRequest) error {
	if len(req.Name) < models.UserNameMin || len(req.Name) > models.UserNameMax || !validName(req.Name) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid username")
	}
	if err := validateMail(req.Mail); err != nil {
		return err
	}
	if len(req.Password) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "password is too short")
	}
	return nil
}

func validName(name string) bool {
	for _, r := range name {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func userNameKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func validateMail(value string) error {
	if value == "" || len(value) > models.MailMax {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid mail")
	}
	addr, err := mail.ParseAddress(value)
	if err != nil || addr.Address != value {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid mail")
	}
	return nil
}

func validateTitle(value string) error {
	if len([]rune(value)) > maxTitleRunes {
		return echo.NewHTTPError(http.StatusBadRequest, "title is too long")
	}
	return nil
}

func validateTextBytes(value string, max int, message string) error {
	if len([]byte(value)) > max {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, message)
	}
	return nil
}

func discussionProblemIDs(item DiscussionDTO) []uint {
	ids := []uint{}
	for _, tag := range item.Tags {
		upper := strings.ToUpper(strings.TrimSpace(tag))
		if !strings.HasPrefix(upper, "P") {
			continue
		}
		id, err := strconv.ParseUint(strings.TrimPrefix(upper, "P"), 10, 64)
		if err == nil {
			ids = append(ids, uint(id))
		}
	}
	return ids
}

func guestMe() MeDTO {
	return MeDTO{ID: 0, Name: "", Mail: "", Bio: "", Avatar: "", Admin: false}
}

func (api *API) requireSignedIn(c echo.Context) error {
	if api.role(c) == "guest" {
		return echo.NewHTTPError(http.StatusUnauthorized, "sign in required")
	}
	return nil
}

func (api *API) requireAdmin(c echo.Context) error {
	if !api.isAdmin(c) {
		return echo.NewHTTPError(http.StatusForbidden, "admin required")
	}
	return nil
}

func (api *API) userName(id uint) string {

	var user models.User
	if err := api.db.First(&user, id).Error; err == nil && user.Name != "" {
		return user.Name
	}

	return strconv.Itoa(int(id))
}

func authorName(id uint, names map[uint]string) string {
	if name := names[id]; name != "" {
		return name
	}
	return strconv.Itoa(int(id))
}

func (api *API) isAdmin(c echo.Context) bool {
	return api.role(c) == "admin"
}

func (api *API) role(c echo.Context) string {

	user, err := utils.UserFromCookie(api.db, c, time.Now())
	if err != nil {
		return "guest"
	}
	if user.Admin {
		return "admin"
	}
	return "user"
}

func baseDiscussionDetails(now time.Time) []DiscussionDetail {
	return []DiscussionDetail{
		{
			Discussion: DiscussionDTO{ID: 1, Title: "A+B Problem 有哪些边界情况？", Author: "admin", Tags: []string{"P1000", "beginner"}, Pinned: true, CreatedAt: now.Add(-3 * time.Hour)},
			Content:    "这题主要覆盖输入输出链路，也用来做第一批评测 smoke。\n\n```cpp\nlong long a, b;\ncin >> a >> b;\ncout << a + b << '\\n';\n```",
			Comments: []CommentDTO{
				{ID: 1, Author: "student", Content: "需要考虑负数吗？", CreatedAt: now.Add(-2 * time.Hour)},
				{ID: 2, Author: "admin", Content: "需要，数据范围会包含负数。", CreatedAt: now.Add(-90 * time.Minute)},
			},
		},
		{
			Discussion: DiscussionDTO{ID: 2, Title: "Limits Hash 的数据范围讨论", Author: "student", Tags: []string{"P1001"}, CreatedAt: now.Add(-24 * time.Hour)},
			Content:    "这题重点是哈希边界和时间限制，建议先用朴素实现确认结果，再优化。",
			Comments: []CommentDTO{
				{ID: 3, Author: "student", Content: "严格模式下空格不一致会怎样？", CreatedAt: now.Add(-22 * time.Hour)},
			},
		},
		{
			Discussion: DiscussionDTO{ID: 3, Title: "交互题提交时需要注意什么？", Author: "admin", Tags: []string{"P1002", "interactive"}, Locked: true, CreatedAt: now.Add(-48 * time.Hour)},
			Content:    "交互题会使用同一套 JudgeProgram/UserProgram pipe 模型。提交时要及时 flush 输出。",
			Comments:   []CommentDTO{},
		},
	}
}

func updatedDiscussion(item DiscussionDetail, req DiscussionUpdate) DiscussionDetail {
	if req.Title != nil {
		item.Discussion.Title = *req.Title
	}
	if req.Tags != nil {
		item.Discussion.Tags = *req.Tags
	}
	if req.Pinned != nil {
		item.Discussion.Pinned = *req.Pinned
	}
	if req.Locked != nil {
		item.Discussion.Locked = *req.Locked
	}
	if req.Content != nil {
		item.Content = *req.Content
	}
	return item
}

func filterDiscussions(items []DiscussionDTO, tag string) []DiscussionDTO {
	if tag == "" {
		return items
	}
	filtered := make([]DiscussionDTO, 0, len(items))
	for _, item := range items {
		if hasTag(item.Tags, tag) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func heatmapFromCounts(counts map[string]int) []HeatCell {
	today := time.Now()
	cells := make([]HeatCell, 0, 365)
	for i := 364; i >= 0; i-- {
		day := today.AddDate(0, 0, -i)
		date := day.Format("2006-01-02")
		cells = append(cells, HeatCell{Date: date, Count: counts[date]})
	}
	return cells
}
