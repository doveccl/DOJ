package public

import (
	"net/http"
	"strings"
	"time"

	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/auth"
	"github.com/doveccl/doj/server/settings"
	"github.com/doveccl/doj/server/validate"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func (api *API) login(c echo.Context) error {
	var req contract.LoginRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	nameKey := validate.NameKey(req.Name)
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
	return c.JSON(http.StatusOK, meView(user))
}

func (api *API) register(c echo.Context) error {
	if !settings.RegistrationAllowed(api.db) {
		return echo.NewHTTPError(http.StatusForbidden, "registration is disabled")
	}
	var req contract.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Mail = strings.ToLower(strings.TrimSpace(req.Mail))
	if err := validateRegister(req); err != nil {
		return err
	}

	now := time.Now()

	nameKey := validate.NameKey(req.Name)
	var count int64
	if err := api.db.Unscoped().Model(&models.User{}).Where("LOWER(name) = ? OR LOWER(mail) = ?", nameKey, req.Mail).Count(&count).Error; err != nil {
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
	return c.JSON(http.StatusCreated, meView(user))
}

func (api *API) createSession(c echo.Context, user models.User, now time.Time) error {
	return auth.CreateUserSession(c, user, now)
}

func (api *API) logout(c echo.Context) error {

	if err := auth.DeleteSession(c); err != nil {
		return err
	}
	auth.ClearSessionCookie(c)
	return c.NoContent(http.StatusNoContent)
}

func (api *API) me(c echo.Context) error {
	refreshCSRFCookie(c)

	user, err := api.currentUser(c)
	if err != nil {
		return c.JSON(http.StatusOK, guestMe())
	}
	return c.JSON(http.StatusOK, meView(user))
}

func refreshCSRFCookie(c echo.Context) {
	token, ok := auth.SessionToken(c)
	if !ok {
		return
	}
	auth.SetCSRFCookie(c, token, auth.SessionExpiresAt(time.Now()))
}

func (api *API) updateMe(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	var req contract.MeUpdate
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
	return c.JSON(http.StatusOK, meView(user))
}

func (api *API) ensureMailAvailable(mail string, currentUserID uint) error {
	var count int64
	if err := api.db.Unscoped().Model(&models.User{}).
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
	var req contract.PasswordUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.OldPassword == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "old password is required")
	}
	if !validate.Password(req.NewPassword) {
		return echo.NewHTTPError(http.StatusBadRequest, "new password must be between 8 and 72 bytes")
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

func (api *API) currentUser(c echo.Context) (models.User, error) {
	viewer := api.requestViewer(c)
	user, err := viewer.user, viewer.err
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return user, echo.NewHTTPError(http.StatusUnauthorized, "sign in required")
		}
		return user, err
	}
	return user, nil
}

func (api *API) viewerID(c echo.Context) (uint, error) {
	viewer := api.requestViewer(c)
	if viewer.err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if viewer.err != nil {
		return 0, viewer.err
	}
	return viewer.user.ID, nil
}

type requestViewer struct {
	user models.User
	err  error
}

const requestViewerKey = "doj.public.viewer"

func (api *API) requestViewer(c echo.Context) requestViewer {
	if viewer, ok := c.Get(requestViewerKey).(requestViewer); ok {
		return viewer
	}
	user, err := auth.UserFromCookie(api.db, c, time.Now())
	viewer := requestViewer{user: user, err: err}
	c.Set(requestViewerKey, viewer)
	return viewer
}

func meView(user models.User) contract.Me {
	return contract.Me{
		ID:                 user.ID,
		Name:               user.Name,
		Mail:               user.Mail,
		Bio:                user.Bio,
		Avatar:             user.Avatar,
		Admin:              user.Admin,
		MustChangePassword: user.Admin && bcrypt.CompareHashAndPassword([]byte(user.Auth), []byte("admin")) == nil,
	}
}

func validateRegister(req contract.RegisterRequest) error {
	if len(req.Name) < models.UserNameMin || len(req.Name) > models.UserNameMax || !validate.Name(req.Name) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid username")
	}
	if err := validateMail(req.Mail); err != nil {
		return err
	}
	if !validate.Password(req.Password) {
		return echo.NewHTTPError(http.StatusBadRequest, "password must be between 8 and 72 bytes")
	}
	return nil
}

func validateMail(value string) error {
	if !validate.Mail(value, models.MailMax, false) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid mail")
	}
	return nil
}

func guestMe() contract.Me {
	return contract.Me{ID: 0, Name: "", Mail: "", Bio: "", Avatar: "", Admin: false, MustChangePassword: false}
}
