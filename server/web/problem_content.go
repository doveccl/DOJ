package web

import (
	"context"
	"fmt"
	"github.com/doveccl/doj/common/cache"
	"github.com/doveccl/doj/common/limits"
	"github.com/doveccl/doj/common/storage"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	contract "github.com/doveccl/doj/common/web"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
)

const problemDiscussionsCacheKey = "doj:problem:discussions"

func (api *API) problemStatement(ctx context.Context, id uint, title string) (string, error) {
	store, err := storage.NewFromEnv()
	if err != nil {
		return "", err
	}
	reader, _, err := store.Open(ctx, problemStatementKey(id))
	if err != nil {
		return "# " + title, nil
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
	found, err := cache.Get(ctx, problemDiscussionsCacheKey, &counts)
	if err == nil && found {
		return counts, nil
	}
	var rows []models.Discussion
	if err := api.db.Select("id", "tags", "pinned", "locked").Order("updated_at desc").Limit(1000).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		item := contract.Discussion{ID: row.ID, Tags: readTags([]byte(row.Tags)), Pinned: row.Pinned, Locked: row.Locked}
		for _, problemID := range discussionProblemIDs(item) {
			counts[problemID]++
		}
	}
	_ = cache.Set(ctx, problemDiscussionsCacheKey, counts, 10*time.Second)
	return counts, nil
}
