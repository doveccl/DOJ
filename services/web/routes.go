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
	imageBodyLimit        = "6M"
	assetBodyLimit        = "130M"
	editAssetBodyLimit    = "2M"
	markdownBodyLimit     = "2M"
	sourceBodyLimit       = "1M"
	shortTextBodyLimit    = "256K"
	maxTitleRunes         = 255
)

type Home struct {
	Notice      string       `json:"notice"`
	Heatmap     []HeatCell   `json:"heatmap"`
	Problems    []ProblemDTO `json:"problems"`
	Assignments []Item       `json:"assignments"`
	Contests    []Item       `json:"contests"`
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
	Mail   string `json:"mail"`
	Bio    string `json:"bio"`
	Avatar string `json:"avatar"`
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
	Users  []uint    `json:"users"`
	Groups []uint    `json:"groups"`
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
	Assignment  AssignmentDTO   `json:"assignment"`
	Problems    []ProblemDTO    `json:"problems"`
	Submissions []SubmissionDTO `json:"submissions"`
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
	Contest     ContestDTO      `json:"contest"`
	Problems    []ProblemDTO    `json:"problems"`
	Rank        []RankUserDTO   `json:"rank"`
	Submissions []SubmissionDTO `json:"submissions"`
}

type SubmissionDTO struct {
	ID           uint      `json:"id"`
	ProblemID    uint      `json:"problemId"`
	ProblemTitle string    `json:"problemTitle"`
	User         string    `json:"user"`
	Language     string    `json:"language"`
	Status       string    `json:"status"`
	Score        int       `json:"score"`
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
	Rank    int    `json:"rank"`
	User    string `json:"user"`
	Bio     string `json:"bio"`
	Avatar  string `json:"avatar"`
	AC      int    `json:"ac"`
	Submit  int    `json:"submit"`
	Score   int    `json:"score"`
	Penalty int    `json:"penalty"`
}

type PublicUserDTO struct {
	Name   string `json:"name"`
	Bio    string `json:"bio"`
	Avatar string `json:"avatar"`
	Admin  bool   `json:"admin"`
	AC     int    `json:"ac"`
	Submit int    `json:"submit"`
}

type UserProfile struct {
	User       PublicUserDTO     `json:"user"`
	Heatmap    []HeatCell        `json:"heatmap"`
	Solved     []ProblemDTO      `json:"solved"`
	Activities []UserActivityDTO `json:"activities"`
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
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
	Pinned  bool     `json:"pinned"`
	Locked  bool     `json:"locked"`
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
	Cases       int        `json:"cases"`
	DataBytes   int64      `json:"dataBytes"`
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
	Mode     string   `json:"mode"`
	TimeMS   int      `json:"timeMs"`
	MemoryMB int      `json:"memoryMb"`
}

type ProblemUpdate struct {
	Title     string   `json:"title"`
	Statement string   `json:"statement"`
	Tags      []string `json:"tags"`
	Visible   bool     `json:"visible"`
	Mode      string   `json:"mode"`
	TimeMS    int      `json:"timeMs"`
	MemoryMB  int      `json:"memoryMb"`
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
	e.POST("/api/auth/login", api.login, api.rateLimit("auth", 20, time.Minute), echomw.BodyLimit(shortTextBodyLimit))
	e.POST("/api/auth/register", api.register, api.rateLimit("auth", 20, time.Minute), echomw.BodyLimit(shortTextBodyLimit))
	e.POST("/api/auth/logout", api.logout)
	e.GET("/api/me", api.me)
	e.PATCH("/api/me", api.updateMe, echomw.BodyLimit(shortTextBodyLimit))
	e.PATCH("/api/me/password", api.updatePassword)
	group := e.Group("/api", api.requireGuestAccess)
	group.GET("/events", api.events)
	group.GET("/home", api.home)
	group.GET("/languages", api.languages)
	group.PATCH("/home/notice", api.updateNotice, echomw.BodyLimit(markdownBodyLimit))
	group.POST("/uploads/images", api.uploadImage, api.rateLimit("upload", 60, time.Minute), echomw.BodyLimit(imageBodyLimit))
	group.GET("/users/:id/:year/:month/:day/*", api.userMedia)
	group.GET("/problems", api.problems)
	group.POST("/problems", api.createProblem)
	group.GET("/problems/:id", api.problem)
	group.PATCH("/problems/:id", api.updateProblem, echomw.BodyLimit(markdownBodyLimit))
	group.DELETE("/problems/:id", api.deleteProblem)
	group.GET("/problems/:id/assets", api.problemAssets)
	group.POST("/problems/:id/assets/images", api.uploadProblemImage, api.rateLimit("problem-upload", 60, time.Minute), echomw.BodyLimit(imageBodyLimit))
	group.GET("/problems/:id/data/*", api.problemPrivateData)
	group.GET("/problems/:id/judge/*", api.problemPrivateJudge)
	group.POST("/problems/:id/assets/files", api.uploadProblemAsset, api.rateLimit("problem-upload", 60, time.Minute), echomw.BodyLimit(assetBodyLimit))
	group.DELETE("/problems/:id/assets/files", api.deleteProblemAsset)
	group.GET("/problems/:id/assets/files/content", api.problemAssetContent)
	group.PATCH("/problems/:id/assets/files/content", api.updateProblemAssetContent, echomw.BodyLimit(editAssetBodyLimit))
	group.POST("/problems/:id/assets/cases", api.createProblemCase, echomw.BodyLimit(editAssetBodyLimit))
	group.POST("/problems/:id/assets/template", api.fillJudgeTemplate, echomw.BodyLimit(editAssetBodyLimit))
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
	group.POST("/submissions", api.submit, api.rateLimit("submit", 30, time.Minute), echomw.BodyLimit(sourceBodyLimit))
	group.GET("/submissions/:id", api.submission)
	group.PATCH("/submissions/:id", api.updateSubmission)
	group.GET("/rank", api.rank)
	group.GET("/users/:name", api.user)
	group.GET("/discussion", api.discussions)
	group.POST("/discussion", api.createDiscussion, api.rateLimit("discussion", 30, time.Minute), echomw.BodyLimit(markdownBodyLimit))
	group.GET("/discussion/:id", api.discussion)
	group.PATCH("/discussion/:id", api.updateDiscussion, echomw.BodyLimit(markdownBodyLimit))
	group.DELETE("/discussion/:id", api.deleteDiscussion)
	group.POST("/discussion/:id/comments", api.createComment, api.rateLimit("comment", 60, time.Minute), echomw.BodyLimit(shortTextBodyLimit))
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
	name := strings.ToLower(strings.TrimSpace(req.Name))
	if name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "username is required")
	}

	now := time.Now()

	var user models.User
	err := api.db.Where("LOWER(name) = ? OR LOWER(mail) = ?", name, name).First(&user).Error
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
	req.Name = strings.ToLower(strings.TrimSpace(req.Name))
	req.Mail = strings.ToLower(strings.TrimSpace(req.Mail))
	if err := validateRegister(req); err != nil {
		return err
	}

	now := time.Now()

	var count int64
	if err := api.db.Model(&models.User{}).Where("LOWER(name) = ? OR LOWER(mail) = ?", req.Name, req.Mail).Count(&count).Error; err != nil {
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
	normalizeMeUpdate(&req)
	if err := validateMeUpdate(req); err != nil {
		return err
	}

	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	if err := api.ensureMailAvailable(req.Mail, user.ID); err != nil {
		return err
	}
	user.Mail = req.Mail
	user.Bio = req.Bio
	user.Avatar = req.Avatar
	if err := api.db.Save(&user).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusOK, meDTO(user))
}

func normalizeMeUpdate(req *MeUpdate) {
	req.Mail = strings.ToLower(strings.TrimSpace(req.Mail))
	req.Bio = strings.TrimSpace(req.Bio)
	req.Avatar = strings.TrimSpace(req.Avatar)
}

func validateMeUpdate(req MeUpdate) error {
	if err := validateMail(req.Mail); err != nil {
		return err
	}
	if len([]rune(req.Bio)) > 280 {
		return echo.NewHTTPError(http.StatusBadRequest, "bio is too long")
	}
	if len(req.Avatar) > 512 {
		return echo.NewHTTPError(http.StatusBadRequest, "avatar url is too long")
	}
	return nil
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
	problems, err := api.listProblems(c, 5)
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
	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		allowed, err := api.assignmentVisible(c, row.ID)
		if err != nil {
			return nil, err
		}
		if !allowed {
			continue
		}
		total, err := api.assignmentProblemCount(row.ID)
		if err != nil {
			return nil, err
		}
		if total == 0 {
			continue
		}
		done, err := api.assignmentDoneCount(c, row.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, Item{ID: row.ID, Title: row.Title, Meta: strconv.Itoa(done) + "/" + strconv.Itoa(total)})
	}
	return items, nil
}

func (api *API) homeContests(c echo.Context) ([]Item, error) {

	var rows []models.Contest
	if err := api.db.Order("start_at desc").Limit(5).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(rows))
	for _, row := range rows {
		total, err := api.contestProblemCount(row, api.isAdmin(c))
		if err != nil {
			return nil, err
		}
		items = append(items, Item{ID: row.ID, Title: row.Title, Meta: row.Kind + " · " + strconv.Itoa(total)})
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
	problems, err := api.searchProblems(c, c.QueryParam("q"), c.QueryParam("tag"), 50)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, problems)
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
	row := models.Problem{
		Title:    req.Title,
		Tags:     tags,
		Visible:  false,
		Mode:     req.Mode,
		TimeMS:   req.TimeMS,
		MemoryMB: req.MemoryMB,
	}
	if err := api.db.Create(&row).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, problemDTO(row))
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
	normalizeProblemUpdate(&req)
	if req.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if err := validateTitle(req.Title); err != nil {
		return err
	}
	if !validProblemMode(req.Mode) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid judge mode")
	}

	var row models.Problem
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	tags, _ := json.Marshal(req.Tags)
	row.Title = req.Title
	row.Tags = tags
	row.Visible = req.Visible
	row.Mode = req.Mode
	row.TimeMS = req.TimeMS
	row.MemoryMB = req.MemoryMB
	if err := api.db.Save(&row).Error; err != nil {
		return err
	}
	if err := api.writeProblemStatement(c.Request().Context(), row.ID, req.Statement); err != nil {
		return err
	}
	item, err := api.problemDTOWithStatement(c.Request().Context(), row)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, item)
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

	var rows []models.Assignment
	if err := api.db.Order("end_at desc").Limit(50).Find(&rows).Error; err != nil {
		return err
	}
	items := make([]AssignmentDTO, 0, len(rows))
	for _, row := range rows {
		allowed, err := api.assignmentVisible(c, row.ID)
		if err != nil {
			return err
		}
		if !allowed {
			continue
		}
		total, err := api.assignmentProblemCount(row.ID)
		if err != nil {
			return err
		}
		if total == 0 {
			continue
		}
		done, err := api.assignmentDoneCount(c, row.ID)
		if err != nil {
			return err
		}
		dto, err := api.assignmentDTO(c, row, total, done)
		if err != nil {
			return err
		}
		items = append(items, dto)
	}
	return c.JSON(http.StatusOK, items)
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
	dto, err := api.assignmentDTO(c, row, len(req.Problems), 0)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, dto)
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
	dto, err := api.assignmentDTO(c, row, len(req.Problems), 0)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, dto)
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
	problems := make([]ProblemDTO, 0, len(links))
	for _, link := range links {
		var problem models.Problem
		if err := api.db.First(&problem, link.ProblemID).Error; err == nil {
			if !api.isAdmin(c) && !api.problemVisibleInList(problem) {
				continue
			}
			item := problemDTO(problem)
			item.Sort = link.Sort
			problems = append(problems, item)
		}
	}
	submissions, err := api.contextSubmissions(c, "assignment", row.ID, nil, api.isAdmin(c), 0)
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
	return c.JSON(http.StatusOK, AssignmentDetail{Assignment: dto, Problems: problems, Submissions: submissions})
}

func (api *API) contests(c echo.Context) error {

	var rows []models.Contest
	if err := api.db.Order("start_at desc").Limit(50).Find(&rows).Error; err != nil {
		return err
	}
	items := make([]ContestDTO, 0, len(rows))
	for _, row := range rows {
		total, err := api.contestProblemCount(row, api.isAdmin(c))
		if err != nil {
			return err
		}
		items = append(items, contestDTO(row, total))
	}
	return c.JSON(http.StatusOK, items)
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
	return c.JSON(http.StatusCreated, contestDTO(row, len(req.Problems)))
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
	return c.JSON(http.StatusOK, contestDTO(row, len(req.Problems)))
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
	problems := make([]ProblemDTO, 0, len(links))
	for _, link := range links {
		var problem models.Problem
		if err := api.db.First(&problem, link.ProblemID).Error; err == nil {
			if !api.contestProblemVisible(row, problem, api.isAdmin(c)) {
				continue
			}
			item := problemDTO(problem)
			item.Sort = link.Sort
			problems = append(problems, item)
		}
	}
	admin := api.isAdmin(c)
	submissions := []SubmissionDTO{}
	rank := []RankUserDTO{}
	if row.Kind != "OI" || !contestRunning(row) || admin {
		freezeAt := api.contestFreezeCutoff(c, row)
		contestIncludeHidden := admin || contestRunning(row)
		viewerID, err := api.viewerID(c)
		if err != nil {
			return err
		}
		submissions, err = api.contextSubmissions(c, "contest", row.ID, freezeAt, contestIncludeHidden, viewerID)
		if err != nil {
			return err
		}
		rank, err = api.contestRank(row, contestIncludeHidden, freezeAt)
		if err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, ContestDetail{Contest: contestDTO(row, len(problems)), Problems: problems, Rank: rank, Submissions: submissions})
}

func (api *API) submissions(c echo.Context) error {

	var rows []models.Submission
	query := api.db.Order("created_at desc").Limit(50)
	if !api.isAdmin(c) {
		query = query.Joins("JOIN problems ON problems.id = submissions.problem_id")
		query = api.applyProblemListVisibility(query)
	}
	if status := c.QueryParam("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if problem := c.QueryParam("problem"); problem != "" {
		if strings.HasPrefix(strings.ToUpper(problem), "P") {
			problem = problem[1:]
		}
		query = query.Where("problem_id = ?", problem)
	}
	if err := query.Find(&rows).Error; err != nil {
		return err
	}
	items := make([]SubmissionDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, api.submissionDTO(row))
	}
	return c.JSON(http.StatusOK, items)
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
	return c.JSON(http.StatusCreated, api.submissionDTO(row))
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

func (api *API) contextSubmissions(c echo.Context, context string, id uint, until *time.Time, includeHidden bool, viewerID uint) ([]SubmissionDTO, error) {
	var rows []models.Submission
	query := api.db.Order("submissions.created_at desc").Limit(50)
	switch context {
	case "assignment":
		query = query.Where("assignment_id = ?", id)
	case "contest":
		query = query.Where("contest_id = ?", id)
	default:
		query = query.Where("1 = 0")
	}
	if until != nil {
		if viewerID > 0 {
			query = query.Where("(submissions.created_at < ? OR submissions.user_id = ?)", *until, viewerID)
		} else {
			query = query.Where("submissions.created_at < ?", *until)
		}
	}
	if !includeHidden {
		query = query.Joins("JOIN problems ON problems.id = submissions.problem_id")
		query = api.applyProblemListVisibility(query)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]SubmissionDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, api.submissionDTO(row))
	}
	return items, nil
}

func (api *API) contestRank(contest models.Contest, includeHidden bool, until *time.Time) ([]RankUserDTO, error) {
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
		return icpcRank(contest, rows, users), nil
	}
	return oiRank(rows, users), nil
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

func oiRank(submissions []models.Submission, users map[uint]models.User) []RankUserDTO {
	type state struct {
		user   models.User
		submit int
		best   map[uint]int
	}
	states := map[uint]*state{}
	for _, row := range submissions {
		user, ok := users[row.UserID]
		if !ok {
			continue
		}
		got := states[row.UserID]
		if got == nil {
			got = &state{user: user, best: map[uint]int{}}
			states[row.UserID] = got
		}
		got.submit++
		got.best[row.ProblemID] = row.Score
	}
	items := make([]RankUserDTO, 0, len(states))
	for _, got := range states {
		score := 0
		ac := 0
		for _, value := range got.best {
			score += value
			if value >= 100 {
				ac++
			}
		}
		items = append(items, RankUserDTO{User: got.user.Name, Bio: got.user.Bio, Avatar: got.user.Avatar, AC: ac, Submit: got.submit, Score: score})
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

func icpcRank(contest models.Contest, submissions []models.Submission, users map[uint]models.User) []RankUserDTO {
	type problemState struct {
		wrong   int
		solved  bool
		penalty int
	}
	type state struct {
		user     models.User
		submit   int
		problems map[uint]*problemState
	}
	states := map[uint]*state{}
	for _, row := range submissions {
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
		for _, problem := range got.problems {
			if problem.solved {
				ac++
				penalty += problem.penalty
			}
		}
		items = append(items, RankUserDTO{User: got.user.Name, Bio: got.user.Bio, Avatar: got.user.Avatar, AC: ac, Submit: got.submit, Score: ac, Penalty: penalty})
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

func penalizable(status string) bool {
	switch status {
	case "AC", "CE", "SE":
		return false
	default:
		return true
	}
}

func (api *API) contextRank(context string, id uint, includeHidden bool, until *time.Time) ([]RankUserDTO, error) {
	var users []models.User
	query := api.db.Model(&models.User{}).
		Joins("JOIN submissions ON submissions.user_id = users.id").
		Distinct("users.id", "users.name", "users.bio", "users.avatar").
		Order("users.id asc")
	switch context {
	case "assignment":
		query = query.Where("submissions.assignment_id = ?", id)
	case "contest":
		query = query.Where("submissions.contest_id = ?", id)
	default:
		query = query.Where("1 = 0")
	}
	if until != nil {
		query = query.Where("submissions.created_at < ?", *until)
	}
	if !includeHidden {
		query = query.Joins("JOIN problems ON problems.id = submissions.problem_id")
		query = api.applyProblemListVisibility(query)
	}
	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	items := make([]RankUserDTO, 0, len(users))
	for _, user := range users {
		ac, submit, err := api.contextUserStats(user.ID, context, id, includeHidden, until)
		if err != nil {
			return nil, err
		}
		if submit == 0 {
			continue
		}
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
	return items, nil
}

func (api *API) contextUserStats(userID uint, context string, id uint, includeHidden bool, until *time.Time) (int, int, error) {
	submitQuery := api.db.Model(&models.Submission{}).
		Where("submissions.user_id = ?", userID)
	switch context {
	case "assignment":
		submitQuery = submitQuery.Where("submissions.assignment_id = ?", id)
	case "contest":
		submitQuery = submitQuery.Where("submissions.contest_id = ?", id)
	default:
		submitQuery = submitQuery.Where("1 = 0")
	}
	if until != nil {
		submitQuery = submitQuery.Where("submissions.created_at < ?", *until)
	}
	if !includeHidden {
		submitQuery = submitQuery.Joins("JOIN problems ON problems.id = submissions.problem_id")
		submitQuery = api.applyProblemListVisibility(submitQuery)
	}
	var submit int64
	if err := submitQuery.Count(&submit).Error; err != nil {
		return 0, 0, err
	}

	acQuery := api.db.Model(&models.Submission{}).
		Where("submissions.user_id = ? AND submissions.status = ?", userID, "AC").
		Distinct("submissions.problem_id")
	switch context {
	case "assignment":
		acQuery = acQuery.Where("submissions.assignment_id = ?", id)
	case "contest":
		acQuery = acQuery.Where("submissions.contest_id = ?", id)
	default:
		acQuery = acQuery.Where("1 = 0")
	}
	if until != nil {
		acQuery = acQuery.Where("submissions.created_at < ?", *until)
	}
	if !includeHidden {
		acQuery = acQuery.Joins("JOIN problems ON problems.id = submissions.problem_id")
		acQuery = api.applyProblemListVisibility(acQuery)
	}
	var ac int64
	if err := acQuery.Count(&ac).Error; err != nil {
		return 0, 0, err
	}
	return int(ac), int(submit), nil
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
	return c.JSON(http.StatusOK, api.submissionDTO(row))
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
	items := make([]RankUserDTO, 0, len(users))
	for _, user := range users {
		ac, submit, err := api.userStats(c.Request().Context(), user.ID)
		if err != nil {
			return err
		}
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

func (api *API) user(c echo.Context) error {
	name := c.Param("name")
	if name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user name is required")
	}

	var row models.User
	if err := api.db.Where("name = ?", name).First(&row).Error; err != nil {
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

func userStatsCacheKey(userID uint) string {
	return "doj:user:" + strconv.FormatUint(uint64(userID), 10) + ":stats"
}

func (api *API) userSubmissions(userID uint, includeHidden bool) ([]models.Submission, error) {
	var rows []models.Submission
	query := api.db.Where("user_id = ?", userID).Order("submissions.created_at desc").Limit(20)
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
	for _, row := range submissions {
		submission := api.submissionDTO(row)
		items = append(items, UserActivityDTO{
			Type:         "submission",
			ID:           submission.ID,
			Title:        submission.ProblemTitle,
			Status:       submission.Status,
			ProblemID:    submission.ProblemID,
			ProblemTitle: submission.ProblemTitle,
			CreatedAt:    submission.CreatedAt,
		})
	}

	var discussions []models.Discussion
	if err := api.db.Where("user_id = ?", userID).Order("created_at desc").Limit(20).Find(&discussions).Error; err != nil {
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

func (api *API) solvedProblems(userID uint, includeHidden bool) ([]ProblemDTO, error) {
	var rows []models.Submission
	query := api.db.Where("submissions.user_id = ? AND submissions.status = ?", userID, "AC").Order("submissions.created_at desc").Limit(50)
	if !includeHidden {
		query = query.Joins("JOIN problems ON problems.id = submissions.problem_id")
		query = api.applyProblemListVisibility(query)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	seen := map[uint]bool{}
	items := make([]ProblemDTO, 0, len(rows))
	for _, row := range rows {
		if seen[row.ProblemID] {
			continue
		}
		seen[row.ProblemID] = true
		var problem models.Problem
		if err := api.db.First(&problem, row.ProblemID).Error; err == nil {
			if !includeHidden && !api.problemVisibleInList(problem) {
				continue
			}
			items = append(items, problemDTO(problem))
		}
	}
	return items, nil
}

func (api *API) userHeatmap(userID uint) ([]HeatCell, error) {
	since := time.Now().AddDate(-1, 0, 0)
	var rows []models.Submission
	query := api.db.Where("submissions.user_id = ? AND submissions.created_at >= ?", userID, since)
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

	var rows []models.Discussion
	query := api.db.Order("pinned desc, updated_at desc").Limit(50)
	if tag := c.QueryParam("tags"); tag != "" {
		rawTag, _ := json.Marshal([]string{tag})
		query = query.Where("tags @> ?::jsonb", string(rawTag))
	}
	if err := query.Find(&rows).Error; err != nil {
		return err
	}
	items := make([]DiscussionDTO, 0, len(rows))
	for _, row := range rows {
		item := DiscussionDTO{
			ID:        row.ID,
			Title:     row.Title,
			Author:    api.userName(row.UserID),
			Tags:      readTags([]byte(row.Tags)),
			Pinned:    row.Pinned,
			Locked:    row.Locked,
			CreatedAt: row.CreatedAt,
		}
		items = append(items, item)
	}
	return c.JSON(http.StatusOK, items)
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
	return c.JSON(http.StatusCreated, DiscussionDTO{
		ID:        row.ID,
		Title:     row.Title,
		Author:    user.Name,
		Tags:      req.Tags,
		Pinned:    row.Pinned,
		Locked:    row.Locked,
		Replies:   0,
		CreatedAt: row.CreatedAt,
	})
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
	if err := api.db.Where("discussion_id = ?", row.ID).Order("created_at asc").Find(&comments).Error; err != nil {
		return err
	}
	items := make([]CommentDTO, 0, len(comments))
	for _, item := range comments {
		items = append(items, CommentDTO{
			ID:        item.ID,
			Author:    api.userName(item.UserID),
			Content:   item.Content,
			CreatedAt: item.CreatedAt,
		})
	}
	dto := DiscussionDTO{
		ID:        row.ID,
		Title:     row.Title,
		Author:    api.userName(row.UserID),
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
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	req.Tags = normalizeTags(req.Tags)
	if req.Title == "" || req.Content == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title and content are required")
	}
	if err := validateTitle(req.Title); err != nil {
		return err
	}

	var row models.Discussion
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "discussion not found")
		}
		return err
	}
	rawTags, _ := json.Marshal(req.Tags)
	row.Title = req.Title
	row.Content = req.Content
	row.Tags = rawTags
	row.Pinned = req.Pinned
	row.Locked = req.Locked
	if err := api.db.Save(&row).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusOK, DiscussionDTO{
		ID:        row.ID,
		Title:     row.Title,
		Author:    api.userName(row.UserID),
		Tags:      req.Tags,
		Pinned:    row.Pinned,
		Locked:    row.Locked,
		CreatedAt: row.CreatedAt,
	})
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

func parseID(c echo.Context, name string, message string) (uint, error) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusBadRequest, message)
	}
	return uint(id), nil
}

func (api *API) listProblems(c echo.Context, limit int) ([]ProblemDTO, error) {
	return api.searchProblems(c, "", "", limit)
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

	var rows []models.Problem
	query := api.db.Order("id desc").Limit(limit)
	if !api.isAdmin(c) {
		query = api.applyProblemListVisibility(query)
	}
	if q != "" {
		query = query.Where("title ILIKE ?", "%"+q+"%")
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
	if err := api.decorateProblemStats(c.Request().Context(), items); err != nil {
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
	const maxStatementBytes = 2 << 20
	data, err := io.ReadAll(io.LimitReader(reader, maxStatementBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxStatementBytes {
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
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	var submits []struct {
		ProblemID uint
		Count     int64
	}
	if err := api.db.Model(&models.Submission{}).
		Select("problem_id, count(*) AS count").
		Where("problem_id IN ?", ids).
		Group("problem_id").
		Find(&submits).Error; err != nil {
		return err
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
		return err
	}
	acByProblem := map[uint]int{}
	for _, item := range acs {
		acByProblem[item.ProblemID] = int(item.Count)
	}
	store, err := utils.NewObjectStoreFromEnv()
	if err != nil {
		return err
	}
	for index := range items {
		id := items[index].ID
		items[index].Submit = submitByProblem[id]
		items[index].AC = acByProblem[id]
		assets, err := api.problemAssetsCached(ctx, id, store)
		if err != nil {
			return err
		}
		items[index].Cases = assets.Cases
		items[index].DataBytes = assets.DataBytes
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
	now := time.Now()
	status := "running"
	if row.EndAt.Before(now) {
		status = "ended"
	}
	dto := AssignmentDTO{
		ID:     row.ID,
		Title:  row.Title,
		EndAt:  row.EndAt,
		Status: status,
		Total:  total,
		Done:   done,
		Users:  []uint{},
		Groups: []uint{},
	}
	if api.isAdmin(c) {
		users, groups, err := api.assignmentMembers(row.ID)
		if err != nil {
			return AssignmentDTO{}, err
		}
		dto.Users = users
		dto.Groups = groups
	}
	return dto, nil
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

func (api *API) assignmentProblemCount(id uint) (int, error) {
	var count int64
	query := api.db.Model(&models.AssignmentProblem{}).Where("assignment_id = ?", id)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
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

func (api *API) contestProblemCount(row models.Contest, admin bool) (int, error) {
	var links []models.ContestProblem
	if err := api.db.Where("contest_id = ?", row.ID).Find(&links).Error; err != nil {
		return 0, err
	}
	if admin || !contestEnded(row) {
		return len(links), nil
	}
	total := 0
	for _, link := range links {
		var problem models.Problem
		if err := api.db.First(&problem, link.ProblemID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return 0, err
		}
		if api.contestProblemVisible(row, problem, admin) {
			total++
		}
	}
	return total, nil
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

func (api *API) contestProblemVisible(row models.Contest, problem models.Problem, admin bool) bool {
	if admin {
		return true
	}
	if contestRunning(row) {
		return true
	}
	if contestEnded(row) {
		return problem.Visible
	}
	return false
}

func (api *API) submissionDTO(row models.Submission) SubmissionDTO {
	title := ""
	userName := ""

	var problem models.Problem
	if err := api.db.First(&problem, row.ProblemID).Error; err == nil {
		title = problem.Title
	}
	var user models.User
	if err := api.db.First(&user, row.UserID).Error; err == nil {
		userName = user.Name
	}

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
		TimeMS:       row.TimeMS,
		MemoryKB:     row.MemoryKB,
		Public:       row.Public,
		CreatedAt:    row.CreatedAt,
	}
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

func normalizeProblemUpdate(req *ProblemUpdate) {
	req.Title = strings.TrimSpace(req.Title)
	req.Statement = strings.TrimSpace(req.Statement)
	req.Mode = strings.TrimSpace(req.Mode)
	req.Tags = normalizeTags(req.Tags)
	if req.Statement == "" && req.Title != "" {
		req.Statement = "# " + req.Title
	}
	if req.Mode == "" {
		req.Mode = "default"
	}
	if req.TimeMS <= 0 {
		req.TimeMS = 1000
	}
	if req.MemoryMB <= 0 {
		req.MemoryMB = 256
	}
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
		if len([]rune(item.Sort)) > 16 {
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

func validateRegister(req RegisterRequest) error {
	if len(req.Name) < 3 || len(req.Name) > 32 || !validName(req.Name) {
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

func validateMail(value string) error {
	if value == "" || len(value) > 255 {
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

func filterSubmissions(items []SubmissionDTO, problem string, status string) []SubmissionDTO {
	if problem == "" && status == "" {
		return items
	}
	problem = strings.TrimPrefix(strings.ToUpper(problem), "P")
	filtered := make([]SubmissionDTO, 0, len(items))
	for _, item := range items {
		if problem != "" && strconv.Itoa(int(item.ProblemID)) != problem {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
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
	item.Discussion.Title = req.Title
	item.Discussion.Tags = req.Tags
	item.Discussion.Pinned = req.Pinned
	item.Discussion.Locked = req.Locked
	item.Content = req.Content
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
