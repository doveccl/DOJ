package web

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/doveccl/doj/server/web/contract"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (api *API) listProblems(c echo.Context, limit int) ([]contract.Problem, error) {
	return api.findProblems(c, "", "", limit, "id desc")
}

func (api *API) searchProblems(c echo.Context, q string, tag string, limit int) ([]contract.Problem, error) {
	return api.findProblems(c, q, tag, limit, "id asc")
}

func (api *API) searchProblemPage(c echo.Context, q string, tag string, limit int, offset int) ([]contract.Problem, int64, error) {
	var rows []models.Problem
	query := api.db.Model(&models.Problem{})
	if !api.isAdmin(c) {
		query = api.applyProblemListVisibility(query)
	}
	query = applyProblemSearch(query, q)
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
	items := make([]contract.Problem, 0, len(rows))
	for _, row := range rows {
		items = append(items, problemView(row))
	}
	return items, total, nil
}

func (api *API) findProblems(c echo.Context, q string, tag string, limit int, order string) ([]contract.Problem, error) {
	var rows []models.Problem
	query := api.db.Order(order).Limit(limit)
	if !api.isAdmin(c) {
		query = api.applyProblemListVisibility(query)
	}
	query = applyProblemSearch(query, q)
	if tag != "" {
		rawTag, _ := json.Marshal([]string{tag})
		query = query.Where("tags @> ?::jsonb", string(rawTag))
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]contract.Problem, 0, len(rows))
	for _, row := range rows {
		items = append(items, problemView(row))
	}
	return items, nil
}

func applyProblemSearch(query *gorm.DB, q string) *gorm.DB {
	q = strings.TrimSpace(q)
	if q == "" {
		return query
	}
	like := "%" + q + "%"
	id := strings.TrimPrefix(strings.ToUpper(q), "P")
	if _, err := strconv.ParseUint(id, 10, 64); err == nil {
		return query.Where("CAST(id AS TEXT) LIKE ? OR LOWER(title) LIKE LOWER(?)", "%"+id+"%", like)
	}
	return query.Where("LOWER(title) LIKE LOWER(?)", like)
}

func problemView(row models.Problem) contract.Problem {
	return contract.Problem{
		ID:       row.ID,
		Title:    row.Title,
		Tags:     readTags([]byte(row.Tags)),
		Visible:  row.Visible,
		Mode:     row.Mode,
		TimeMS:   row.TimeMS,
		MemoryMB: row.MemoryMB,
	}
}

func (api *API) problemViewWithStatement(ctx context.Context, row models.Problem) (contract.Problem, error) {
	item := problemView(row)
	statement, err := api.problemStatement(ctx, row.ID, row.Title)
	if err != nil {
		return item, err
	}
	item.Statement = statement
	return item, nil
}
