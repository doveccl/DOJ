package admin

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	settingSiteName                = "site_name"
	settingAllowRegistration       = "allow_registration"
	settingAllowGuestAccess        = "allow_guest_access"
	settingDefaultSubmissionPublic = "default_submission_public"
	settingHomeNotice              = "home_notice"
)

func (api *API) getSettings(c echo.Context) error {
	settings, err := api.settings()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, settings)
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
