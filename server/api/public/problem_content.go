package public

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/doveccl/doj/contract/limits"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/storage"
	"github.com/labstack/echo/v4"
)

func (api *API) problemStatement(ctx context.Context, id uint, title string) (string, error) {
	store, err := storage.NewFromEnv()
	if err != nil {
		return "", err
	}
	reader, _, err := store.Open(ctx, problemStatementKey(id))
	if err != nil {
		if storage.IsNotFound(err) {
			return "# " + title, nil
		}
		return "", err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, limits.MaxMarkdownBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > limits.MaxMarkdownBytes {
		return "", echo.NewHTTPError(http.StatusRequestEntityTooLarge, "statement is too large")
	}
	if strings.TrimSpace(string(data)) == "" {
		return "# " + title, nil
	}
	return string(data), nil
}

func (api *API) writeProblemStatement(ctx context.Context, id uint, statement string) error {
	store, err := storage.NewFromEnv()
	if err != nil {
		return err
	}
	if err := validateTextBytes(statement, limits.MaxMarkdownBytes, "statement is too large"); err != nil {
		return err
	}
	body := strings.TrimSpace(statement)
	if body == "" {
		body = "# P" + strconv.Itoa(int(id))
	}
	return store.Put(ctx, problemStatementKey(id), strings.NewReader(body), int64(len(body)), "text/markdown; charset=utf-8")
}

func problemStatementKey(id uint) string {
	return fmt.Sprintf("problems/%d/statement.md", id)
}

func (api *API) problemDiscussionCounts(ctx context.Context) (map[uint]int, error) {
	counts := map[uint]int{}
	var rows []models.Discussion
	if err := api.db.WithContext(ctx).Select("tags").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		seen := map[uint]bool{}
		for _, tag := range readTags([]byte(row.Tags)) {
			upper := strings.ToUpper(strings.TrimSpace(tag))
			id, err := strconv.ParseUint(strings.TrimPrefix(upper, "P"), 10, 64)
			if !strings.HasPrefix(upper, "P") || err != nil || id == 0 || seen[uint(id)] {
				continue
			}
			seen[uint(id)] = true
			counts[uint(id)]++
		}
	}
	return counts, nil
}
