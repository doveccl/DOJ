package public

import (
	"net/http"
	"strings"

	"github.com/doveccl/doj/contract/limits"
	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (api *API) updateAssignmentDescription(c echo.Context) error {
	return api.updateContextDescription(c, "assignment", &models.Assignment{})
}

func (api *API) updateContestDescription(c echo.Context) error {
	return api.updateContextDescription(c, "contest", &models.Contest{})
}

func (api *API) updateContextDescription(c echo.Context, label string, model any) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid "+label+" id")
	if err != nil {
		return err
	}
	if err := api.db.First(model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, label+" not found")
		}
		return err
	}
	var req contract.DescriptionUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Description = strings.TrimSpace(req.Description)
	if err := validateTextBytes(req.Description, limits.MaxMarkdownBytes, "description is too large"); err != nil {
		return err
	}
	if err := api.db.Model(model).Update("description", req.Description).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusOK, req)
}
