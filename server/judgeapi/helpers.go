package judgeapi

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func parseTaskID(c echo.Context) (uint, error) {
	raw := strings.TrimSpace(c.Param("id"))
	var value uint64
	for _, char := range raw {
		if char < '0' || char > '9' {
			return 0, echo.NewHTTPError(http.StatusBadRequest, "invalid task id")
		}
		value = value*10 + uint64(char-'0')
	}
	if value == 0 {
		return 0, echo.NewHTTPError(http.StatusBadRequest, "invalid task id")
	}
	return uint(value), nil
}
