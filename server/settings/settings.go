package settings

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/doveccl/doj/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	keySiteName                = "site_name"
	keyAllowRegistration       = "allow_registration"
	keyAllowGuestAccess        = "allow_guest_access"
	keyDefaultSubmissionPublic = "default_submission_public"
	keyHomeNotice              = "home_notice"
)

var (
	ErrEmptyPatch      = errors.New("settings patch is empty")
	ErrSiteNameEmpty   = errors.New("site name is required")
	ErrSiteNameTooLong = errors.New("site name is too long")
)

type Settings struct {
	SiteName                string `json:"siteName"`
	AllowRegistration       bool   `json:"allowRegistration"`
	AllowGuestAccess        bool   `json:"allowGuestAccess"`
	DefaultSubmissionPublic bool   `json:"defaultSubmissionPublic"`
	Notice                  string `json:"notice"`
}

type Patch struct {
	SiteName                *string `json:"siteName,omitempty"`
	AllowRegistration       *bool   `json:"allowRegistration,omitempty"`
	AllowGuestAccess        *bool   `json:"allowGuestAccess,omitempty"`
	DefaultSubmissionPublic *bool   `json:"defaultSubmissionPublic,omitempty"`
	Notice                  *string `json:"notice,omitempty"`
}

func Get(db *gorm.DB) (Settings, error) {
	settings := defaults()
	var rows []models.Setting
	if err := db.Find(&rows, "key IN ?", keys()).Error; err != nil {
		return settings, err
	}
	for _, row := range rows {
		if err := apply(&settings, row); err != nil {
			return settings, err
		}
	}
	if settings.SiteName == "" {
		settings.SiteName = "DOJ"
	}
	return settings, nil
}

func Update(db *gorm.DB, req Patch) (Settings, error) {
	current, err := Get(db)
	if err != nil {
		return current, err
	}
	changed := map[string]any{}
	if req.SiteName != nil {
		siteName := strings.TrimSpace(*req.SiteName)
		if siteName == "" {
			return current, ErrSiteNameEmpty
		}
		if len([]rune(siteName)) > models.NameMax {
			return current, ErrSiteNameTooLong
		}
		current.SiteName = siteName
		changed[keySiteName] = siteName
	}
	if req.AllowRegistration != nil {
		current.AllowRegistration = *req.AllowRegistration
		changed[keyAllowRegistration] = *req.AllowRegistration
	}
	if req.AllowGuestAccess != nil {
		current.AllowGuestAccess = *req.AllowGuestAccess
		changed[keyAllowGuestAccess] = *req.AllowGuestAccess
	}
	if req.DefaultSubmissionPublic != nil {
		current.DefaultSubmissionPublic = *req.DefaultSubmissionPublic
		changed[keyDefaultSubmissionPublic] = *req.DefaultSubmissionPublic
	}
	if req.Notice != nil {
		current.Notice = *req.Notice
		changed[keyHomeNotice] = *req.Notice
	}
	if len(changed) == 0 {
		return current, ErrEmptyPatch
	}
	return current, saveChanged(db, changed)
}

func Save(db *gorm.DB, settings Settings) error {
	return saveChanged(db, map[string]any{
		keySiteName:                settings.SiteName,
		keyAllowRegistration:       settings.AllowRegistration,
		keyAllowGuestAccess:        settings.AllowGuestAccess,
		keyDefaultSubmissionPublic: settings.DefaultSubmissionPublic,
		keyHomeNotice:              settings.Notice,
	})
}

func GuestAllowed(db *gorm.DB) bool {
	settings, err := Get(db)
	return err == nil && settings.AllowGuestAccess
}

func RegistrationAllowed(db *gorm.DB) bool {
	settings, err := Get(db)
	return err == nil && settings.AllowRegistration
}

func defaults() Settings {
	return Settings{SiteName: "DOJ"}
}

func keys() []string {
	return []string{keySiteName, keyAllowRegistration, keyAllowGuestAccess, keyDefaultSubmissionPublic, keyHomeNotice}
}

func saveChanged(db *gorm.DB, changed map[string]any) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, key := range keys() {
			value, ok := changed[key]
			if !ok {
				continue
			}
			if err := save(tx, key, value); err != nil {
				return err
			}
		}
		return nil
	})
}

func save(db *gorm.DB, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return db.Save(&models.Setting{Key: key, Value: datatypes.JSON(raw)}).Error
}

func apply(settings *Settings, row models.Setting) error {
	switch row.Key {
	case keySiteName:
		return json.Unmarshal(row.Value, &settings.SiteName)
	case keyAllowRegistration:
		return json.Unmarshal(row.Value, &settings.AllowRegistration)
	case keyAllowGuestAccess:
		return json.Unmarshal(row.Value, &settings.AllowGuestAccess)
	case keyDefaultSubmissionPublic:
		return json.Unmarshal(row.Value, &settings.DefaultSubmissionPublic)
	case keyHomeNotice:
		return json.Unmarshal(row.Value, &settings.Notice)
	default:
		return nil
	}
}
