package web

import (
	"encoding/json"
	"fmt"
	"github.com/doveccl/doj/common/authn"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
)

func parseID(c echo.Context, name string, message string) (uint, error) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusBadRequest, message)
	}
	return uint(id), nil
}

func parseQueryID(value string, message string) (uint, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusBadRequest, message)
	}
	return uint(id), nil
}

func parseProblemQuery(value string) (uint, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToUpper(value), "P") {
		value = value[1:]
	}
	return parseQueryID(value, "invalid problem id")
}

func parseIDs(value string, message string) ([]uint, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	ids := []uint{}
	for _, part := range strings.Split(value, ",") {
		id, err := parseQueryID(part, message)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return uniqueUint(ids), nil
}

func parsePage(c echo.Context) (int, int, int, error) {
	page := 1
	if value := strings.TrimSpace(c.QueryParam("page")); value != "" {
		got, err := strconv.Atoi(value)
		if err != nil || got <= 0 {
			return 0, 0, 0, echo.NewHTTPError(http.StatusBadRequest, "invalid page")
		}
		page = got
	}
	pageSize := 20
	if value := strings.TrimSpace(c.QueryParam("pageSize")); value != "" {
		got, err := strconv.Atoi(value)
		if err != nil || got <= 0 {
			return 0, 0, 0, echo.NewHTTPError(http.StatusBadRequest, "invalid page size")
		}
		pageSize = got
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize, (page - 1) * pageSize, nil
}

func parseNamedPage(c echo.Context, prefix string, defaultPageSize int) (int, int, int, error) {
	pageParam := prefix + "Page"
	pageSizeParam := prefix + "PageSize"
	page := 1
	if value := strings.TrimSpace(c.QueryParam(pageParam)); value != "" {
		got, err := strconv.Atoi(value)
		if err != nil || got <= 0 {
			return 0, 0, 0, echo.NewHTTPError(http.StatusBadRequest, "invalid "+pageParam)
		}
		page = got
	}
	pageSize := defaultPageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if value := strings.TrimSpace(c.QueryParam(pageSizeParam)); value != "" {
		got, err := strconv.Atoi(value)
		if err != nil || got <= 0 {
			return 0, 0, 0, echo.NewHTTPError(http.StatusBadRequest, "invalid "+pageSizeParam)
		}
		pageSize = got
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize, (page - 1) * pageSize, nil
}

func (api *API) searchTags(c echo.Context, kind string, q string, limit int) ([]string, error) {
	kind = strings.TrimSpace(kind)
	q = strings.TrimSpace(q)
	if limit <= 0 {
		limit = 50
	}
	switch kind {
	case "problem":
		return api.searchJSONTags(c, "problems", q, limit)
	case "discussion":
		return api.searchJSONTags(c, "discussions", q, limit)
	default:
		return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid tag kind")
	}
}

func (api *API) searchJSONTags(c echo.Context, table string, q string, limit int) ([]string, error) {
	if api.db.Dialector.Name() == "postgres" {
		sql := fmt.Sprintf("SELECT DISTINCT tag.value AS tag FROM %s CROSS JOIN LATERAL jsonb_array_elements_text(tags) AS tag(value) WHERE %s.deleted_at IS NULL", table, table)
		args := []any{}
		if q != "" {
			sql += " AND LOWER(tag.value) LIKE LOWER(?)"
			args = append(args, "%"+q+"%")
		}
		sql += " ORDER BY tag ASC LIMIT ?"
		args = append(args, limit)
		var rows []struct {
			Tag string
		}
		if err := api.db.WithContext(c.Request().Context()).Raw(sql, args...).Scan(&rows).Error; err != nil {
			return nil, err
		}
		items := make([]string, 0, len(rows))
		for _, row := range rows {
			items = append(items, row.Tag)
		}
		return items, nil
	}

	seen := map[string]bool{}
	items := []string{}
	match := strings.ToLower(q)
	switch table {
	case "problems":
		var rows []models.Problem
		if err := api.db.Select("tags").Order("id asc").Limit(500).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			items = appendMatchingTags(items, seen, readTags([]byte(row.Tags)), match, limit)
			if len(items) >= limit {
				return items, nil
			}
		}
	case "discussions":
		var rows []models.Discussion
		if err := api.db.Select("tags").Order("id asc").Limit(500).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			items = appendMatchingTags(items, seen, readTags([]byte(row.Tags)), match, limit)
			if len(items) >= limit {
				return items, nil
			}
		}
	}
	sort.Strings(items)
	return items, nil
}

func appendMatchingTags(items []string, seen map[string]bool, tags []string, match string, limit int) []string {
	for _, tag := range tags {
		if len(items) >= limit {
			return items
		}
		if seen[tag] || (match != "" && !strings.Contains(strings.ToLower(tag), match)) {
			continue
		}
		seen[tag] = true
		items = append(items, tag)
	}
	return items
}

func readTags(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var tags []string
	if err := json.Unmarshal(raw, &tags); err != nil {
		return nil
	}
	return tags
}

func hasTag(tags []string, tag string) bool {
	for _, item := range tags {
		if item == tag {
			return true
		}
	}
	return false
}

func normalizeTags(tags []string) []string {
	seen := map[string]bool{}
	clean := make([]string, 0, len(tags))
	for _, tag := range tags {
		for _, part := range strings.FieldsFunc(tag, func(r rune) bool {
			return r == ',' || r == '，' || r == ' ' || r == '\n' || r == '\t'
		}) {
			part = strings.TrimSpace(part)
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			clean = append(clean, part)
		}
	}
	return clean
}

func (api *API) validateUserIDs(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uint]bool, len(ids))
	for _, id := range ids {
		if id == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
		}
		if seen[id] {
			return echo.NewHTTPError(http.StatusBadRequest, "duplicate user id")
		}
		seen[id] = true
	}
	var count int64
	if err := api.db.Model(&models.User{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return echo.NewHTTPError(http.StatusBadRequest, "user not found")
	}
	return nil
}

func (api *API) validateGroupIDs(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[uint]bool, len(ids))
	for _, id := range ids {
		if id == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid group id")
		}
		if seen[id] {
			return echo.NewHTTPError(http.StatusBadRequest, "duplicate group id")
		}
		seen[id] = true
	}
	var count int64
	if err := api.db.Model(&models.Group{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return echo.NewHTTPError(http.StatusBadRequest, "group not found")
	}
	return nil
}

func cleanUintList(values []uint) []uint {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[uint]bool, len(values))
	items := make([]uint, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		items = append(items, value)
	}
	return items
}

func uniqueUint(values []uint) []uint {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[uint]bool, len(values))
	items := make([]uint, 0, len(values))
	for _, value := range values {
		if value == 0 || seen[value] {
			continue
		}
		seen[value] = true
		items = append(items, value)
	}
	return items
}

func validateTitle(value string) error {
	if len([]rune(value)) > maxTitleRunes {
		return echo.NewHTTPError(http.StatusBadRequest, "title is too long")
	}
	return nil
}

func validateTextBytes(value string, max int, message string) error {
	if len([]byte(value)) > max {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, message)
	}
	return nil
}

func (api *API) requireSignedIn(c echo.Context) error {
	if api.role(c) == "guest" {
		return echo.NewHTTPError(http.StatusUnauthorized, "sign in required")
	}
	return nil
}

func (api *API) requireAdmin(c echo.Context) error {
	if !api.isAdmin(c) {
		return echo.NewHTTPError(http.StatusForbidden, "admin required")
	}
	return nil
}

func (api *API) isAdmin(c echo.Context) bool {
	return api.role(c) == "admin"
}

func (api *API) role(c echo.Context) string {

	user, err := authn.UserFromCookie(api.db, c, time.Now())
	if err != nil {
		return "guest"
	}
	if user.Admin {
		return "admin"
	}
	return "user"
}
