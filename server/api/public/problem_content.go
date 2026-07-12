package public

import (
	"context"
	"strconv"
	"strings"

	"github.com/doveccl/doj/contract/limits"
	"github.com/doveccl/doj/models"
)

func problemStatement(row models.Problem) string {
	return defaultProblemStatement(row.ID, row.Title, row.Content)
}

func normalizeProblemStatement(id uint, title string, statement string) (string, error) {
	body := strings.TrimSpace(statement)
	if body == "" {
		if title != "" {
			body = "# " + title
		} else {
			body = "# P" + strconv.Itoa(int(id))
		}
	}
	if err := validateTextBytes(body, limits.MaxMarkdownBytes, "statement is too large"); err != nil {
		return "", err
	}
	return body, nil
}

func defaultProblemStatement(id uint, title string, statement string) string {
	body := strings.TrimSpace(statement)
	if body != "" {
		return body
	}
	if title != "" {
		return "# " + title
	}
	return "# P" + strconv.Itoa(int(id))
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
