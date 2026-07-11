package public

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/doveccl/doj/contract/limits"
	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/storage"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (api *API) assignmentDescription(ctx context.Context, id uint) (string, error) {
	return contextDescription(ctx, "assignments", id)
}

func (api *API) contestDescription(ctx context.Context, id uint) (string, error) {
	return contextDescription(ctx, "contests", id)
}

func contextDescription(ctx context.Context, kind string, id uint) (string, error) {
	store, err := storage.NewFromEnv()
	if err != nil {
		return "", err
	}
	reader, _, err := store.Open(ctx, contextDescriptionKey(kind, id))
	if err != nil {
		if storage.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, limits.MaxMarkdownBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > limits.MaxMarkdownBytes {
		return "", echo.NewHTTPError(http.StatusRequestEntityTooLarge, "description is too large")
	}
	return string(data), nil
}

func (api *API) updateAssignmentDescription(c echo.Context) error {
	return api.updateContextDescription(c, "assignments", "assignment", &models.Assignment{})
}

func (api *API) updateContestDescription(c echo.Context) error {
	return api.updateContextDescription(c, "contests", "contest", &models.Contest{})
}

func (api *API) updateContextDescription(c echo.Context, kind string, label string, model any) error {
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
	store, err := storage.NewFromEnv()
	if err != nil {
		return err
	}
	key := contextDescriptionKey(kind, id)
	if req.Description == "" {
		if err := store.Delete(c.Request().Context(), key); err != nil {
			return err
		}
	} else if err := store.Put(c.Request().Context(), key, strings.NewReader(req.Description), int64(len(req.Description)), "text/markdown; charset=utf-8"); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, req)
}

func contextDescriptionKey(kind string, id uint) string {
	return fmt.Sprintf("%s/%d/description.md", kind, id)
}
