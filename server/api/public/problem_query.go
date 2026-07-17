package public

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	contract "github.com/doveccl/doj/contract/web"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (api *API) searchProblemPage(c echo.Context, q string, tag string, status string, limit int, offset int) ([]contract.Problem, int64, error) {
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
	query, err := api.applyProblemStatusFilter(c, query, status)
	if err != nil {
		return nil, 0, err
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := problemListColumns(query.Session(&gorm.Session{})).Order("id asc").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]contract.Problem, 0, len(rows))
	for _, row := range rows {
		items = append(items, problemView(row))
	}
	return items, total, nil
}

// applyProblemStatusFilter uses the same ac > pending > tried > none priority
// as problem-state so filters and displayed labels stay aligned.
func (api *API) applyProblemStatusFilter(c echo.Context, query *gorm.DB, status string) (*gorm.DB, error) {
	status = strings.TrimSpace(status)
	if status == "" || status == "all" {
		return query, nil
	}
	if status != "none" && status != "tried" && status != "ac" && status != "pending" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid status filter")
	}
	if api.role(c) == "guest" {
		return query, nil
	}
	viewerID, err := api.viewerID(c)
	if err != nil {
		return nil, err
	}
	if viewerID == 0 {
		return query, nil
	}
	anySub, err := api.viewerProblemStatusSubquery(c, viewerID, false)
	if err != nil {
		return nil, err
	}
	acSub, err := api.viewerProblemStatusSubquery(c, viewerID, true)
	if err != nil {
		return nil, err
	}
	liveSub := api.viewerProblemSubmissionSubquery(viewerID).Where("submissions.status IN ?", []string{"queued", "judging"})
	hiddenSub, err := api.filterHiddenResults(c, api.viewerProblemSubmissionSubquery(viewerID).Where("submissions.status NOT IN ?", []string{"queued", "judging"}))
	if err != nil {
		return nil, err
	}
	switch status {
	case "ac":
		return query.Where("EXISTS (?)", acSub), nil
	case "pending":
		return query.Where("NOT EXISTS (?) AND (EXISTS (?) OR EXISTS (?))", acSub, liveSub, hiddenSub), nil
	case "tried":
		return query.Where("EXISTS (?) AND NOT EXISTS (?) AND NOT EXISTS (?) AND NOT EXISTS (?)", anySub, acSub, liveSub, hiddenSub), nil
	default:
		return query.Where("NOT EXISTS (?) AND NOT EXISTS (?) AND NOT EXISTS (?)", anySub, liveSub, hiddenSub), nil
	}
}

// viewerProblemStatusSubquery selects the viewer's visible completed results
// for the outer problem. When acOnly is set it further restricts to AC.
func (api *API) viewerProblemStatusSubquery(c echo.Context, viewerID uint, acOnly bool) (*gorm.DB, error) {
	sub := api.viewerProblemSubmissionSubquery(viewerID).
		Where("submissions.status NOT IN ?", []string{"queued", "judging"})
	if acOnly {
		sub = sub.Where("submissions.status = ?", "AC")
	}
	return api.filterVisibleResults(c, sub)
}

func (api *API) viewerProblemSubmissionSubquery(viewerID uint) *gorm.DB {
	return api.db.Model(&models.Submission{}).
		Select("1").
		Where("submissions.problem_id = problems.id").
		Where("submissions.user_id = ?", viewerID)
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

func problemListColumns(query *gorm.DB) *gorm.DB {
	return query.Select("id", "title", "tags", "visible", "mode", "time_ms", "memory_mb")
}
