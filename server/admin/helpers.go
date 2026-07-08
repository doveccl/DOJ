package admin

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/doveccl/doj/common/authn"
	"github.com/labstack/echo/v4"
)

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

func parseUintCSV(raw string) ([]uint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	items := []uint{}
	for _, part := range strings.Split(raw, ",") {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		id, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid id list")
		}
		items = append(items, uint(id))
	}
	return cleanUintList(items), nil
}

func parsePage(c echo.Context) (int, int, error) {
	page, err := positiveIntQuery(c, "page", 1)
	if err != nil {
		return 0, 0, err
	}
	pageSize, err := positiveIntQuery(c, "pageSize", 20)
	if err != nil {
		return 0, 0, err
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize, nil
}

func positiveIntQuery(c echo.Context, name string, fallback int) (int, error) {
	raw := strings.TrimSpace(c.QueryParam(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, echo.NewHTTPError(http.StatusBadRequest, "invalid "+name)
	}
	return value, nil
}

func randomPassword() (string, error) {
	token, err := authn.NewToken()
	if err != nil {
		return "", err
	}
	if len(token) > 20 {
		token = token[:20]
	}
	return token, nil
}

func parseUintParam(c echo.Context, name string, message string) (uint, error) {
	value, err := strconv.ParseUint(strings.TrimSpace(c.Param(name)), 10, 64)
	if err != nil || value == 0 {
		return 0, echo.NewHTTPError(http.StatusBadRequest, message)
	}
	return uint(value), nil
}
