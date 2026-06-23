package admin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/utils"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type API struct {
	db *gorm.DB
}

type Overview struct {
	Settings  AdminSettings `json:"settings"`
	Users     []User        `json:"users"`
	Groups    []Group       `json:"groups"`
	Languages []Language    `json:"languages"`
	Judgers   []Judger      `json:"judgers"`
	Queue     JudgeQueue    `json:"queue"`
}

type AdminSettings struct {
	SiteName            string `json:"siteName"`
	Registration        bool   `json:"registration"`
	Guest               bool   `json:"guest"`
	DefaultPublicSource bool   `json:"defaultPublicSource"`
	Notice              string `json:"notice"`
}

type User struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Mail   string `json:"mail"`
	Role   string `json:"role"`
	Groups []uint `json:"groups"`
}

type Group struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type Language struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Source     string `json:"source"`
	Dockerfile string `json:"dockerfile"`
}

type Judger struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Token     string    `json:"token,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type JudgeQueue struct {
	Queued  int `json:"queued"`
	Running int `json:"running"`
	Done    int `json:"done"`
}

type UserUpdate struct {
	Role   string `json:"role"`
	Groups []uint `json:"groups"`
}

type UserCreate struct {
	Name     string `json:"name"`
	Mail     string `json:"mail"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Groups   []uint `json:"groups"`
}

type PasswordReset struct {
	Password string `json:"password"`
}

type GroupUpdate struct {
	Name string `json:"name"`
}

type LanguageUpdate struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Source     string `json:"source"`
	Dockerfile string `json:"dockerfile"`
}

type LanguageCreate struct {
	LanguageUpdate
}

type JudgerUpdate struct {
	Name string `json:"name"`
	Auth string `json:"auth"`
}

type JudgerCreate struct {
	JudgerUpdate
}

func Register(e *echo.Echo, db *gorm.DB) {
	if db == nil {
		panic("admin API requires a database")
	}
	api := &API{db: db}
	group := e.Group("/api/admin", api.requireAdmin)
	group.GET("", api.overview)
	group.PATCH("/settings", api.updateSettings)
	group.POST("/users", api.createUser)
	group.PATCH("/users/:name", api.updateUser)
	group.DELETE("/users/:name", api.deleteUser)
	group.POST("/users/:name/password", api.resetUserPassword)
	group.POST("/groups", api.createGroup)
	group.PATCH("/groups/:id", api.updateGroup)
	group.DELETE("/groups/:id", api.deleteGroup)
	group.POST("/languages", api.createLanguage)
	group.PATCH("/languages/:id", api.updateLanguage)
	group.DELETE("/languages/:id", api.deleteLanguage)
	group.POST("/judgers", api.createJudger)
	group.PATCH("/judgers/:id", api.updateJudger)
	group.DELETE("/judgers/:id", api.deleteJudger)
}

func (api *API) requireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !api.isAdmin(c) {
			return echo.NewHTTPError(http.StatusForbidden, "admin required")
		}
		return next(c)
	}
}

func (api *API) isAdmin(c echo.Context) bool {

	user, err := utils.UserFromCookie(api.db, c, time.Now())
	return err == nil && user.Admin
}

func (api *API) overview(c echo.Context) error {

	overview, err := api.readOverview()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, overview)
}

func (api *API) readOverview() (Overview, error) {
	settings, err := api.settings()
	if err != nil {
		return Overview{}, err
	}
	users, err := api.users()
	if err != nil {
		return Overview{}, err
	}
	groups, err := api.groups()
	if err != nil {
		return Overview{}, err
	}
	languages, err := api.languages()
	if err != nil {
		return Overview{}, err
	}
	judgers, err := api.judgers()
	if err != nil {
		return Overview{}, err
	}
	queue, err := api.queue()
	if err != nil {
		return Overview{}, err
	}
	return Overview{Settings: settings, Users: users, Groups: groups, Languages: languages, Judgers: judgers, Queue: queue}, nil
}

func (api *API) updateSettings(c echo.Context) error {
	var req AdminSettings
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.SiteName = strings.TrimSpace(req.SiteName)
	if req.SiteName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "site name is required")
	}

	if err := SaveSettings(api.db, req); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, req)
}

func (api *API) createUser(c echo.Context) error {
	var req UserCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	cleanUserCreate(&req)
	if err := validateUserCreate(req); err != nil {
		return err
	}
	if err := api.ensureGroups(req.Groups); err != nil {
		return err
	}

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
	err = api.db.Transaction(func(tx *gorm.DB) error {
		user := models.User{
			Name:  req.Name,
			Mail:  req.Mail,
			Auth:  string(hash),
			Admin: req.Role == "admin",
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		items := make([]models.GroupUser, 0, len(req.Groups))
		for _, id := range req.Groups {
			items = append(items, models.GroupUser{UserID: user.ID, GroupID: id})
		}
		if len(items) > 0 {
			return tx.Create(&items).Error
		}
		return nil
	})
	if err != nil {
		return err
	}
	overview, err := api.readOverview()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, overview)
}

func (api *API) updateUser(c echo.Context) error {
	var req UserUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Role = strings.TrimSpace(req.Role)
	if req.Role != "admin" && req.Role != "user" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role")
	}
	req.Groups = cleanUintList(req.Groups)

	var row models.User
	if err := api.db.First(&row, "name = ?", c.Param("name")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return err
	}
	if row.Admin && req.Role != "admin" {
		if err := api.ensureOtherAdmin(row.ID); err != nil {
			return err
		}
	}
	if err := api.ensureGroups(req.Groups); err != nil {
		return err
	}
	err := api.db.Transaction(func(tx *gorm.DB) error {
		row.Admin = req.Role == "admin"
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", row.ID).Delete(&models.GroupUser{}).Error; err != nil {
			return err
		}
		items := make([]models.GroupUser, 0, len(req.Groups))
		for _, id := range req.Groups {
			items = append(items, models.GroupUser{UserID: row.ID, GroupID: id})
		}
		if len(items) > 0 {
			return tx.Create(&items).Error
		}
		return nil
	})
	if err != nil {
		return err
	}
	return api.overview(c)
}

func (api *API) deleteUser(c echo.Context) error {

	var row models.User
	if err := api.db.First(&row, "name = ?", c.Param("name")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return err
	}
	if row.Admin {
		if err := api.ensureOtherAdmin(row.ID); err != nil {
			return err
		}
	}
	if err := api.db.Delete(&row).Error; err != nil {
		return err
	}
	return api.overview(c)
}

func (api *API) resetUserPassword(c echo.Context) error {
	password, err := randomPassword()
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	updated := api.db.Model(&models.User{}).Where("name = ?", c.Param("name")).Update("auth", string(hash))
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}
	return c.JSON(http.StatusOK, PasswordReset{Password: password})
}

func (api *API) createGroup(c echo.Context) error {
	var req GroupUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "group name is required")
	}

	if err := api.db.Create(&models.Group{Name: req.Name}).Error; err != nil {
		return err
	}
	overview, err := api.readOverview()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, overview)
}

func (api *API) updateGroup(c echo.Context) error {
	id, err := parseUintParam(c, "id", "invalid group id")
	if err != nil {
		return err
	}
	var req GroupUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "group name is required")
	}

	updated := api.db.Model(&models.Group{}).Where("id = ?", id).Update("name", req.Name)
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "group not found")
	}
	return api.overview(c)
}

func (api *API) deleteGroup(c echo.Context) error {
	id, err := parseUintParam(c, "id", "invalid group id")
	if err != nil {
		return err
	}

	var row models.Group
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "group not found")
		}
		return err
	}
	if err := api.db.Delete(&row).Error; err != nil {
		return err
	}
	return api.overview(c)
}

func (api *API) updateLanguage(c echo.Context) error {
	var req LanguageUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	cleanLanguageUpdate(&req)
	if err := validateLanguage(req); err != nil {
		return err
	}

	var row models.Language
	if err := api.db.First(&row, "id = ?", c.Param("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "language not found")
		}
		return err
	}
	if req.ID != row.ID {
		var count int64
		if err := api.db.Model(&models.Language{}).Where("id = ?", req.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return echo.NewHTTPError(http.StatusConflict, "language already exists")
		}
	}
	updates := map[string]any{"id": req.ID, "name": req.Name, "source": req.Source, "dockerfile": req.Dockerfile}
	if err := api.db.Model(&models.Language{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		return err
	}
	return api.overview(c)
}

func (api *API) createLanguage(c echo.Context) error {
	var req LanguageCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	cleanLanguageUpdate(&req.LanguageUpdate)
	if err := validateLanguage(req.LanguageUpdate); err != nil {
		return err
	}
	row := models.Language{ID: req.ID, Name: req.Name, Source: req.Source, Dockerfile: req.Dockerfile}
	if err := api.db.Create(&row).Error; err != nil {
		return err
	}
	overview, err := api.readOverview()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, overview)
}

func (api *API) deleteLanguage(c echo.Context) error {

	deleted := api.db.Delete(&models.Language{ID: c.Param("id")})
	if deleted.Error != nil {
		return deleted.Error
	}
	if deleted.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "language not found")
	}
	return api.overview(c)
}

func (api *API) updateJudger(c echo.Context) error {
	id, err := parseUintParam(c, "id", "invalid judger id")
	if err != nil {
		return err
	}
	var req JudgerUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	cleanJudgerUpdate(&req)
	if err := validateJudger(req, false); err != nil {
		return err
	}

	updates := map[string]any{"name": req.Name}
	if req.Auth != "" {
		updates["auth"] = tokenHash(req.Auth)
	}
	updated := api.db.Model(&models.Judger{}).Where("id = ?", id).Updates(updates)
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "judger not found")
	}
	return api.overview(c)
}

func (api *API) createJudger(c echo.Context) error {
	var req JudgerCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	cleanJudgerUpdate(&req.JudgerUpdate)
	if err := validateJudger(req.JudgerUpdate, false); err != nil {
		return err
	}

	token, err := utils.NewToken()
	if err != nil {
		return err
	}
	row := models.Judger{Name: req.Name, Auth: tokenHash(token)}
	if err := api.db.Create(&row).Error; err != nil {
		return err
	}
	overview, err := api.readOverview()
	if err != nil {
		return err
	}
	for index := range overview.Judgers {
		if overview.Judgers[index].ID == row.ID {
			overview.Judgers[index].Token = token
			break
		}
	}
	return c.JSON(http.StatusCreated, overview)
}

func (api *API) deleteJudger(c echo.Context) error {
	id, err := parseUintParam(c, "id", "invalid judger id")
	if err != nil {
		return err
	}

	deleted := api.db.Delete(&models.Judger{}, id)
	if deleted.Error != nil {
		return deleted.Error
	}
	if deleted.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "judger not found")
	}
	return api.overview(c)
}

func (api *API) ensureOtherAdmin(userID uint) error {
	var otherAdmins int64
	if err := api.db.Model(&models.User{}).Where("id <> ? AND admin = ?", userID, true).Count(&otherAdmins).Error; err != nil {
		return err
	}
	if otherAdmins == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "cannot remove the last admin")
	}
	return nil
}

func (api *API) settings() (AdminSettings, error) {
	settings := AdminSettings{SiteName: "DOJ", Registration: false, Guest: false, DefaultPublicSource: false, Notice: ""}
	var row models.Setting
	if err := api.db.First(&row, "key = ?", "site").Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return settings, nil
		}
		return settings, err
	}
	_ = json.Unmarshal(row.Value, &settings)
	if settings.SiteName == "" {
		settings.SiteName = "DOJ"
	}
	return settings, nil
}

func Settings(db *gorm.DB) (AdminSettings, error) {
	api := API{db: db}
	return api.settings()
}

func SaveSettings(db *gorm.DB, settings AdminSettings) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	row := models.Setting{Key: "site", Value: datatypes.JSON(raw)}
	return db.Save(&row).Error
}

func (api *API) users() ([]User, error) {
	var rows []models.User
	if err := api.db.Order("id asc").Limit(200).Find(&rows).Error; err != nil {
		return nil, err
	}
	userIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, row.ID)
	}
	groupMap := map[uint][]uint{}
	if len(userIDs) > 0 {
		var links []models.GroupUser
		err := api.db.Table("group_users").
			Select("group_users.group_id, group_users.user_id").
			Joins("JOIN groups ON groups.id = group_users.group_id AND groups.deleted_at IS NULL").
			Where("group_users.user_id IN ?", userIDs).
			Find(&links).Error
		if err != nil {
			return nil, err
		}
		for _, link := range links {
			groupMap[link.UserID] = append(groupMap[link.UserID], link.GroupID)
		}
	}
	items := make([]User, 0, len(rows))
	for _, row := range rows {
		role := "user"
		if row.Admin {
			role = "admin"
		}
		items = append(items, User{ID: row.ID, Name: row.Name, Mail: row.Mail, Role: role, Groups: cleanUintList(groupMap[row.ID])})
	}
	return items, nil
}

func (api *API) groups() ([]Group, error) {
	var rows []models.Group
	if err := api.db.Order("id asc").Limit(200).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]Group, 0, len(rows))
	for _, row := range rows {
		items = append(items, Group{ID: row.ID, Name: row.Name})
	}
	return items, nil
}

func (api *API) languages() ([]Language, error) {
	var rows []models.Language
	if err := api.db.Order("id asc").Limit(200).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]Language, 0, len(rows))
	for _, row := range rows {
		items = append(items, Language{ID: row.ID, Name: row.Name, Source: row.Source, Dockerfile: row.Dockerfile})
	}
	return items, nil
}

func (api *API) judgers() ([]Judger, error) {
	var rows []models.Judger
	if err := api.db.Order("id asc").Limit(200).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]Judger, 0, len(rows))
	for _, row := range rows {
		items = append(items, Judger{ID: row.ID, Name: row.Name, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt})
	}
	return items, nil
}

func (api *API) queue() (JudgeQueue, error) {
	var queued int64
	if err := api.db.Model(&models.Submission{}).Where("status = ?", "queued").Count(&queued).Error; err != nil {
		return JudgeQueue{}, err
	}
	var running int64
	if err := api.db.Model(&models.Submission{}).Where("status = ?", "judging").Count(&running).Error; err != nil {
		return JudgeQueue{}, err
	}
	var done int64
	if err := api.db.Model(&models.Submission{}).Where("status NOT IN ?", []string{"queued", "judging"}).Count(&done).Error; err != nil {
		return JudgeQueue{}, err
	}
	return JudgeQueue{Queued: int(queued), Running: int(running), Done: int(done)}, nil
}

func (api *API) ensureGroups(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	var count int64
	if err := api.db.Model(&models.Group{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid group")
	}
	return nil
}

func cleanUintList(values []uint) []uint {
	if len(values) == 0 {
		return []uint{}
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	items := values[:0]
	var last uint
	for index, value := range values {
		if value == 0 {
			continue
		}
		if index > 0 && value == last {
			continue
		}
		items = append(items, value)
		last = value
	}
	if len(items) == 0 {
		return []uint{}
	}
	return items
}

func cleanUserCreate(req *UserCreate) {
	req.Name = strings.ToLower(strings.TrimSpace(req.Name))
	req.Mail = strings.ToLower(strings.TrimSpace(req.Mail))
	req.Role = strings.TrimSpace(req.Role)
	req.Groups = cleanUintList(req.Groups)
}

func validateUserCreate(req UserCreate) error {
	if len(req.Name) < 3 || len(req.Name) > 32 || !validUserName(req.Name) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid username")
	}
	if req.Mail == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid mail")
	}
	addr, err := mail.ParseAddress(req.Mail)
	if err != nil || addr.Address != req.Mail {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid mail")
	}
	if len(req.Password) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "password is too short")
	}
	if req.Role != "admin" && req.Role != "user" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role")
	}
	return nil
}

func validUserName(name string) bool {
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

func cleanLanguageUpdate(req *LanguageUpdate) {
	req.ID = strings.TrimSpace(req.ID)
	req.Name = strings.TrimSpace(req.Name)
	req.Source = strings.TrimSpace(req.Source)
	req.Dockerfile = strings.TrimSpace(req.Dockerfile)
}

func validateLanguage(req LanguageUpdate) error {
	if req.ID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "language id is required")
	}
	if !validLanguageID(req.ID) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid language id")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "language name is required")
	}
	if req.Source == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "language source is required")
	}
	if req.Dockerfile == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "language dockerfile is required")
	}
	return nil
}

func validLanguageID(id string) bool {
	if len(id) == 0 || len(id) > 32 {
		return false
	}
	for _, char := range id {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' {
			continue
		}
		switch char {
		case '+', '#', '.', '_', '-':
			continue
		default:
			return false
		}
	}
	return true
}

func cleanJudgerUpdate(req *JudgerUpdate) {
	req.Name = strings.TrimSpace(req.Name)
	req.Auth = strings.TrimSpace(req.Auth)
}

func validateJudger(req JudgerUpdate, requireAuth bool) error {
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "judger name is required")
	}
	if requireAuth && req.Auth == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "judger auth is required")
	}
	return nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomPassword() (string, error) {
	token, err := utils.NewToken()
	if err != nil {
		return "", err
	}
	if len(token) > 20 {
		token = token[:20]
	}
	return token, nil
}

func parseUintParam(c echo.Context, name string, message string) (uint, error) {
	value, err := strconvParseUint(c.Param(name))
	if err != nil || value == 0 {
		return 0, echo.NewHTTPError(http.StatusBadRequest, message)
	}
	return value, nil
}

func strconvParseUint(raw string) (uint, error) {
	var value uint64
	for _, char := range strings.TrimSpace(raw) {
		if char < '0' || char > '9' {
			return 0, echo.NewHTTPError(http.StatusBadRequest, "invalid id")
		}
		value = value*10 + uint64(char-'0')
	}
	if value == 0 {
		return 0, echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	return uint(value), nil
}

func languageIDExists(id string, except string, items []Language) bool {
	for _, item := range items {
		if item.ID == id && item.ID != except {
			return true
		}
	}
	return false
}

func GuestAllowed(db *gorm.DB) bool {
	settings, err := Settings(db)
	if err != nil {
		return false
	}
	return settings.Guest
}

func RegistrationAllowed(db *gorm.DB) bool {
	settings, err := Settings(db)
	if err != nil {
		return false
	}
	return settings.Registration
}

func ValidateMail(value string) error {
	if value == "" {
		return nil
	}
	if _, err := mail.ParseAddress(value); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid mail")
	}
	return nil
}
