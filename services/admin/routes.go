package admin

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/doveccl/doj/models"
	backupsvc "github.com/doveccl/doj/services/backup"
	judgersvc "github.com/doveccl/doj/services/judger"
	"github.com/doveccl/doj/utils"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
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
	SiteName                string `json:"siteName"`
	AllowRegistration       bool   `json:"allowRegistration"`
	AllowGuestAccess        bool   `json:"allowGuestAccess"`
	DefaultSubmissionPublic bool   `json:"defaultSubmissionPublic"`
	Notice                  string `json:"notice"`
}

type AdminSettingsPatch struct {
	SiteName                *string `json:"siteName,omitempty"`
	AllowRegistration       *bool   `json:"allowRegistration,omitempty"`
	AllowGuestAccess        *bool   `json:"allowGuestAccess,omitempty"`
	DefaultSubmissionPublic *bool   `json:"defaultSubmissionPublic,omitempty"`
	Notice                  *string `json:"notice,omitempty"`
}

type User struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Mail   string `json:"mail"`
	Role   string `json:"role"`
	Groups []uint `json:"groups"`
}

type Group struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Users []uint `json:"users"`
}

type Language struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Source     string `json:"source"`
	Dockerfile string `json:"dockerfile"`
}

type Judger struct {
	ID            uint       `json:"id"`
	Name          string     `json:"name"`
	Token         string     `json:"token,omitempty"`
	Online        bool       `json:"online"`
	ConnectedAt   *time.Time `json:"connectedAt"`
	ActiveAt      *time.Time `json:"activeAt"`
	UptimeSeconds int        `json:"uptimeSeconds"`
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
	Name  string `json:"name"`
	Users []uint `json:"users"`
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

const maxNameRunes = models.NameMax

const (
	settingSiteName                = "site_name"
	settingAllowRegistration       = "allow_registration"
	settingAllowGuestAccess        = "allow_guest_access"
	settingDefaultSubmissionPublic = "default_submission_public"
	settingHomeNotice              = "home_notice"
)

func Register(e *echo.Echo, db *gorm.DB) {
	if db == nil {
		panic("admin API requires a database")
	}
	api := &API{db: db}
	group := e.Group("/api/admin", api.requireAdmin)
	group.GET("", api.overview)
	group.GET("/settings", api.getSettings)
	group.PATCH("/settings", api.updateSettings, echomw.BodyLimit(utils.BodyLimitSettings))
	group.POST("/users", api.createUser)
	group.PATCH("/users/:name", api.updateUser)
	group.DELETE("/users/:name", api.deleteUser)
	group.POST("/users/:name/password", api.resetUserPassword)
	group.POST("/groups", api.createGroup)
	group.PATCH("/groups/:id", api.updateGroup)
	group.DELETE("/groups/:id", api.deleteGroup)
	group.POST("/languages", api.createLanguage, echomw.BodyLimit(utils.BodyLimitDockerfile))
	group.PATCH("/languages/:id", api.updateLanguage, echomw.BodyLimit(utils.BodyLimitDockerfile))
	group.DELETE("/languages/:id", api.deleteLanguage)
	group.POST("/judgers", api.createJudger)
	group.PATCH("/judgers/:id", api.updateJudger)
	group.DELETE("/judgers/:id", api.deleteJudger)
	group.GET("/backups/settings", api.backupSettings)
	group.PATCH("/backups/settings", api.updateBackupSettings, echomw.BodyLimit(utils.BodyLimitSettings))
	group.GET("/backups", api.backups)
	group.POST("/backups", api.createBackup)
	group.GET("/backups/:name/download", api.downloadBackup)
	group.DELETE("/backups/:name", api.deleteBackup)
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

	overview, err := api.readOverview(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, overview)
}

func (api *API) getSettings(c echo.Context) error {
	settings, err := api.settings()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, settings)
}

func (api *API) backupSettings(c echo.Context) error {
	settings, err := backupsvc.ReadSettings(api.db)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, settings)
}

func (api *API) updateBackupSettings(c echo.Context) error {
	var req backupsvc.Settings
	if err := c.Bind(&req); err != nil {
		return err
	}
	settings, err := backupsvc.CleanSettings(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := backupsvc.SaveSettings(api.db, settings); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, settings)
}

func (api *API) backups(c echo.Context) error {
	manager := backupsvc.Manager{DB: api.db}
	list, err := manager.List(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, list)
}

func (api *API) createBackup(c echo.Context) error {
	manager := backupsvc.Manager{DB: api.db}
	item, err := manager.BackupNow(c.Request().Context())
	if err != nil {
		if errors.Is(err, backupsvc.ErrRunning) {
			return echo.NewHTTPError(http.StatusConflict, "backup already running")
		}
		return err
	}
	return c.JSON(http.StatusCreated, item)
}

func (api *API) downloadBackup(c echo.Context) error {
	name := c.Param("name")
	manager := backupsvc.Manager{DB: api.db}
	reader, contentType, err := manager.Open(c.Request().Context(), name)
	if err != nil {
		return err
	}
	defer reader.Close()
	if contentType == "" {
		contentType = "application/gzip"
	}
	c.Response().Header().Set(echo.HeaderContentDisposition, `attachment; filename="`+name+`"`)
	return c.Stream(http.StatusOK, contentType, reader)
}

func (api *API) deleteBackup(c echo.Context) error {
	manager := backupsvc.Manager{DB: api.db}
	if err := manager.Delete(c.Request().Context(), c.Param("name")); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (api *API) readOverview(ctx context.Context) (Overview, error) {
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
	judgers, err := api.judgers(ctx)
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
	var req AdminSettingsPatch
	if err := c.Bind(&req); err != nil {
		return err
	}

	current, err := api.settings()
	if err != nil {
		return err
	}
	changed := map[string]any{}
	if req.SiteName != nil {
		siteName := strings.TrimSpace(*req.SiteName)
		if siteName == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "site name is required")
		}
		if len([]rune(siteName)) > maxNameRunes {
			return echo.NewHTTPError(http.StatusBadRequest, "site name is too long")
		}
		current.SiteName = siteName
		changed[settingSiteName] = siteName
	}
	if req.AllowRegistration != nil {
		current.AllowRegistration = *req.AllowRegistration
		changed[settingAllowRegistration] = *req.AllowRegistration
	}
	if req.AllowGuestAccess != nil {
		current.AllowGuestAccess = *req.AllowGuestAccess
		changed[settingAllowGuestAccess] = *req.AllowGuestAccess
	}
	if req.DefaultSubmissionPublic != nil {
		current.DefaultSubmissionPublic = *req.DefaultSubmissionPublic
		changed[settingDefaultSubmissionPublic] = *req.DefaultSubmissionPublic
	}
	if req.Notice != nil {
		current.Notice = *req.Notice
		changed[settingHomeNotice] = *req.Notice
	}
	if len(changed) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "settings patch is empty")
	}

	if err := api.db.Transaction(func(tx *gorm.DB) error {
		for key, value := range changed {
			if err := saveSetting(tx, key, value); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, current)
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
	if err := api.db.Model(&models.User{}).Where("LOWER(name) = ? OR LOWER(mail) = ?", userNameKey(req.Name), req.Mail).Count(&count).Error; err != nil {
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
	overview, err := api.readOverview(c.Request().Context())
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

	row, err := api.userByName(c.Param("name"))
	if err != nil {
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
	err = api.db.Transaction(func(tx *gorm.DB) error {
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

	row, err := api.userByName(c.Param("name"))
	if err != nil {
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
	row, err := api.userByName(c.Param("name"))
	if err != nil {
		return err
	}
	password, err := randomPassword()
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	updated := api.db.Model(&models.User{}).Where("id = ?", row.ID).Update("auth", string(hash))
	if updated.Error != nil {
		return updated.Error
	}
	return c.JSON(http.StatusOK, PasswordReset{Password: password})
}

func (api *API) createGroup(c echo.Context) error {
	var req GroupUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Users = cleanUintList(req.Users)
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "group name is required")
	}
	if len([]rune(req.Name)) > maxNameRunes {
		return echo.NewHTTPError(http.StatusBadRequest, "group name is too long")
	}
	if err := api.ensureUsers(req.Users); err != nil {
		return err
	}

	if err := api.db.Transaction(func(tx *gorm.DB) error {
		group := models.Group{Name: req.Name}
		if err := tx.Create(&group).Error; err != nil {
			return err
		}
		return saveGroupUsers(tx, group.ID, req.Users)
	}); err != nil {
		return err
	}
	overview, err := api.readOverview(c.Request().Context())
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
	req.Users = cleanUintList(req.Users)
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "group name is required")
	}
	if len([]rune(req.Name)) > maxNameRunes {
		return echo.NewHTTPError(http.StatusBadRequest, "group name is too long")
	}
	if err := api.ensureUsers(req.Users); err != nil {
		return err
	}

	err = api.db.Transaction(func(tx *gorm.DB) error {
		updated := tx.Model(&models.Group{}).Where("id = ?", id).Update("name", req.Name)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected == 0 {
			return echo.NewHTTPError(http.StatusNotFound, "group not found")
		}
		return saveGroupUsers(tx, id, req.Users)
	})
	if err != nil {
		return err
	}
	return api.overview(c)
}

func (api *API) deleteGroup(c echo.Context) error {
	id, err := parseUintParam(c, "id", "invalid group id")
	if err != nil {
		return err
	}

	err = api.db.Transaction(func(tx *gorm.DB) error {
		var row models.Group
		if err := tx.First(&row, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return echo.NewHTTPError(http.StatusNotFound, "group not found")
			}
			return err
		}
		if err := tx.Where("group_id = ?", row.ID).Delete(&models.GroupUser{}).Error; err != nil {
			return err
		}
		if err := tx.Where("group_id = ?", row.ID).Delete(&models.AssignmentGroup{}).Error; err != nil {
			return err
		}
		return tx.Delete(&row).Error
	})
	if err != nil {
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
	overview, err := api.readOverview(c.Request().Context())
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
	overview, err := api.readOverview(c.Request().Context())
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

func (api *API) userByName(name string) (models.User, error) {
	var row models.User
	nameKey := userNameKey(name)
	if nameKey == "" {
		return row, echo.NewHTTPError(http.StatusBadRequest, "user name is required")
	}
	if err := api.db.Where("LOWER(name) = ?", nameKey).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return row, echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return row, err
	}
	return row, nil
}

func (api *API) settings() (AdminSettings, error) {
	settings := defaultSettings()
	var rows []models.Setting
	if err := api.db.Find(&rows, "key IN ?", settingKeys()).Error; err != nil {
		return settings, err
	}
	for _, row := range rows {
		if err := applySetting(&settings, row); err != nil {
			return settings, err
		}
	}
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
	return db.Transaction(func(tx *gorm.DB) error {
		values := map[string]any{
			settingSiteName:                settings.SiteName,
			settingAllowRegistration:       settings.AllowRegistration,
			settingAllowGuestAccess:        settings.AllowGuestAccess,
			settingDefaultSubmissionPublic: settings.DefaultSubmissionPublic,
			settingHomeNotice:              settings.Notice,
		}
		for _, key := range settingKeys() {
			if err := saveSetting(tx, key, values[key]); err != nil {
				return err
			}
		}
		return nil
	})
}

func defaultSettings() AdminSettings {
	return AdminSettings{SiteName: "DOJ", AllowRegistration: false, AllowGuestAccess: false, DefaultSubmissionPublic: false, Notice: ""}
}

func settingKeys() []string {
	return []string{settingSiteName, settingAllowRegistration, settingAllowGuestAccess, settingDefaultSubmissionPublic, settingHomeNotice}
}

func saveSetting(db *gorm.DB, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return db.Save(&models.Setting{Key: key, Value: datatypes.JSON(raw)}).Error
}

func applySetting(settings *AdminSettings, row models.Setting) error {
	switch row.Key {
	case settingSiteName:
		return json.Unmarshal(row.Value, &settings.SiteName)
	case settingAllowRegistration:
		return json.Unmarshal(row.Value, &settings.AllowRegistration)
	case settingAllowGuestAccess:
		return json.Unmarshal(row.Value, &settings.AllowGuestAccess)
	case settingDefaultSubmissionPublic:
		return json.Unmarshal(row.Value, &settings.DefaultSubmissionPublic)
	case settingHomeNotice:
		return json.Unmarshal(row.Value, &settings.Notice)
	default:
		return nil
	}
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
			Joins("JOIN groups ON groups.id = group_users.group_id").
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
	groupIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		groupIDs = append(groupIDs, row.ID)
	}
	userMap := map[uint][]uint{}
	if len(groupIDs) > 0 {
		var links []models.GroupUser
		if err := api.db.Table("group_users").
			Select("group_users.group_id, group_users.user_id").
			Joins("JOIN users ON users.id = group_users.user_id AND users.deleted_at IS NULL").
			Where("group_users.group_id IN ?", groupIDs).
			Order("group_users.user_id asc").
			Find(&links).Error; err != nil {
			return nil, err
		}
		for _, link := range links {
			userMap[link.GroupID] = append(userMap[link.GroupID], link.UserID)
		}
	}
	items := make([]Group, 0, len(rows))
	for _, row := range rows {
		items = append(items, Group{ID: row.ID, Name: row.Name, Users: cleanUintList(userMap[row.ID])})
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

func (api *API) judgers(ctx context.Context) ([]Judger, error) {
	var rows []models.Judger
	if err := api.db.Order("id asc").Limit(200).Find(&rows).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	items := make([]Judger, 0, len(rows))
	for _, row := range rows {
		status := judgersvc.ReadStatus(ctx, row.ID, now)
		items = append(items, Judger{ID: row.ID, Name: row.Name, Online: status.Online, ConnectedAt: status.ConnectedAt, ActiveAt: status.ActiveAt, UptimeSeconds: status.UptimeSeconds})
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

func (api *API) ensureUsers(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	var count int64
	if err := api.db.Model(&models.User{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user")
	}
	return nil
}

func saveGroupUsers(tx *gorm.DB, groupID uint, users []uint) error {
	if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupUser{}).Error; err != nil {
		return err
	}
	items := make([]models.GroupUser, 0, len(users))
	for _, id := range users {
		items = append(items, models.GroupUser{GroupID: groupID, UserID: id})
	}
	if len(items) > 0 {
		return tx.Create(&items).Error
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
	req.Name = strings.TrimSpace(req.Name)
	req.Mail = strings.ToLower(strings.TrimSpace(req.Mail))
	req.Role = strings.TrimSpace(req.Role)
	req.Groups = cleanUintList(req.Groups)
}

func validateUserCreate(req UserCreate) error {
	if len(req.Name) < models.UserNameMin || len(req.Name) > models.UserNameMax || !validUserName(req.Name) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid username")
	}
	if req.Mail == "" || len(req.Mail) > models.MailMax {
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
	if len([]rune(req.Name)) > maxNameRunes {
		return echo.NewHTTPError(http.StatusBadRequest, "language name is too long")
	}
	if req.Source == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "language source is required")
	}
	if len([]rune(req.Source)) > models.SourceMax {
		return echo.NewHTTPError(http.StatusBadRequest, "language source is too long")
	}
	if req.Dockerfile == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "language dockerfile is required")
	}
	if len([]byte(req.Dockerfile)) > utils.MaxDockerfileBytes {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "language dockerfile is too large")
	}
	return nil
}

func validLanguageID(id string) bool {
	if len(id) == 0 || len(id) > models.LanguageIDMax {
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
	if len([]rune(req.Name)) > maxNameRunes {
		return echo.NewHTTPError(http.StatusBadRequest, "judger name is too long")
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
	return settings.AllowGuestAccess
}

func RegistrationAllowed(db *gorm.DB) bool {
	settings, err := Settings(db)
	if err != nil {
		return false
	}
	return settings.AllowRegistration
}

func ValidateMail(value string) error {
	if value == "" {
		return nil
	}
	if len(value) > models.MailMax {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid mail")
	}
	if _, err := mail.ParseAddress(value); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid mail")
	}
	return nil
}
