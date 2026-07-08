package admin

import (
	"net/http"
	"strings"

	"github.com/doveccl/doj/common/limits"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (api *API) getLanguages(c echo.Context) error {
	languages, err := api.languages()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, languages)
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
	updates := map[string]any{"id": req.ID, "name": req.Name, "source": req.Source, "image": req.Image, "compile": req.Compile, "run": req.Run}
	if err := api.db.Model(&models.Language{}).Where("id = ?", row.ID).Updates(updates).Error; err != nil {
		return err
	}
	languages, err := api.languages()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, languages)
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
	row := models.Language{ID: req.ID, Name: req.Name, Source: req.Source, Image: req.Image, Compile: req.Compile, Run: req.Run}
	if err := api.db.Create(&row).Error; err != nil {
		return err
	}
	languages, err := api.languages()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, languages)
}

func (api *API) deleteLanguage(c echo.Context) error {
	deleted := api.db.Delete(&models.Language{ID: c.Param("id")})
	if deleted.Error != nil {
		return deleted.Error
	}
	if deleted.RowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "language not found")
	}
	languages, err := api.languages()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, languages)
}

func (api *API) languages() ([]Language, error) {
	var rows []models.Language
	if err := api.db.Order("id asc").Limit(200).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]Language, 0, len(rows))
	for _, row := range rows {
		items = append(items, Language{ID: row.ID, Name: row.Name, Source: row.Source, Image: row.Image, Compile: row.Compile, Run: row.Run})
	}
	return items, nil
}

func cleanLanguageUpdate(req *LanguageUpdate) {
	req.ID = strings.TrimSpace(req.ID)
	req.Name = strings.TrimSpace(req.Name)
	req.Source = strings.TrimSpace(req.Source)
	req.Image = strings.TrimSpace(req.Image)
	req.Compile = strings.TrimSpace(req.Compile)
	req.Run = strings.TrimSpace(req.Run)
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
	if req.Image == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "language image is required")
	}
	if len([]rune(req.Image)) > 256 {
		return echo.NewHTTPError(http.StatusBadRequest, "language image is too long")
	}
	if req.Run == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "language run command is required")
	}
	if len([]byte(req.Compile)) > limits.MaxLanguageCommandBytes || len([]byte(req.Run)) > limits.MaxLanguageCommandBytes {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, "language command is too large")
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
