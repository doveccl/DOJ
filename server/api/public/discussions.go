package public

import (
	"encoding/json"
	"github.com/doveccl/doj/contract/limits"
	"net/http"
	"strconv"
	"strings"
	"time"

	contract "github.com/doveccl/doj/contract/web"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (api *API) discussions(c echo.Context) error {

	page, pageSize, offset, err := parsePage(c)
	if err != nil {
		return err
	}
	var rows []models.Discussion
	query := api.db.Model(&models.Discussion{})
	if q := strings.TrimSpace(c.QueryParam("q")); q != "" {
		like := "%" + q + "%"
		query = query.Where("LOWER(title) LIKE LOWER(?) OR LOWER(content) LIKE LOWER(?) OR LOWER(CAST(tags AS TEXT)) LIKE LOWER(?)", like, like, like)
	}
	if tag := c.QueryParam("tags"); tag != "" {
		rawTag, _ := json.Marshal([]string{tag})
		query = query.Where("tags @> ?::jsonb", string(rawTag))
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return err
	}
	if err := query.Session(&gorm.Session{}).
		Select("id", "title", "user_id", "tags", "pinned", "locked", "created_at").
		Order("pinned desc, updated_at desc").
		Limit(pageSize).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return err
	}
	authorIDs := make([]uint, 0, len(rows))
	discussionIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		authorIDs = append(authorIDs, row.UserID)
		discussionIDs = append(discussionIDs, row.ID)
	}
	authors, err := api.userNameMap(authorIDs)
	if err != nil {
		return err
	}
	replies, err := api.discussionReplyCounts(discussionIDs)
	if err != nil {
		return err
	}
	items := make([]contract.Discussion, 0, len(rows))
	for _, row := range rows {
		items = append(items, discussionViewFromRefs(row, authors, replies))
	}
	return c.JSON(http.StatusOK, contract.Page[contract.Discussion]{Items: items, Page: page, PageSize: pageSize, Total: total})
}

func (api *API) createDiscussion(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	var req contract.DiscussionCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	req.Tags = normalizeTags(req.Tags)
	if req.Title == "" || req.Content == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title and content are required")
	}
	if err := validateTitle(req.Title); err != nil {
		return err
	}
	if err := validateTextBytes(req.Content, limits.MaxMarkdownBytes, "discussion content is too large"); err != nil {
		return err
	}
	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	rawTags, _ := json.Marshal(req.Tags)
	row := models.Discussion{
		Title:   req.Title,
		Content: req.Content,
		UserID:  user.ID,
		Tags:    rawTags,
	}
	if err := api.db.Create(&row).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, contract.CreatedID{ID: row.ID})
}

func (api *API) discussion(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid discussion id")
	}

	var row models.Discussion
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "discussion not found")
		}
		return err
	}
	var comments []models.Comment
	if err := api.db.Unscoped().Select("id", "user_id", "content", "created_at", "deleted_at").Where("discussion_id = ?", row.ID).Order("created_at asc").Find(&comments).Error; err != nil {
		return err
	}
	authorIDs := []uint{row.UserID}
	for _, item := range comments {
		authorIDs = append(authorIDs, item.UserID)
	}
	authors, err := api.userNameMap(authorIDs)
	if err != nil {
		return err
	}
	items := make([]contract.Comment, 0, len(comments))
	replies := 0
	for _, item := range comments {
		deleted := item.DeletedAt.Valid
		content := item.Content
		if !deleted {
			replies++
		} else {
			content = ""
		}
		items = append(items, contract.Comment{
			ID:        item.ID,
			Author:    authorName(item.UserID, authors),
			Content:   content,
			Deleted:   deleted,
			CreatedAt: item.CreatedAt,
		})
	}
	item := contract.Discussion{
		ID:        row.ID,
		Title:     row.Title,
		Author:    authorName(row.UserID, authors),
		Tags:      readTags([]byte(row.Tags)),
		Pinned:    row.Pinned,
		Locked:    row.Locked,
		Replies:   replies,
		CreatedAt: row.CreatedAt,
	}
	return c.JSON(http.StatusOK, contract.DiscussionDetail{
		Discussion: item,
		Content:    row.Content,
		Comments:   items,
	})
}

func (api *API) updateDiscussion(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid discussion id")
	if err != nil {
		return err
	}
	var req contract.DiscussionUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.Title == nil && req.Content == nil && req.Tags == nil && req.Pinned == nil && req.Locked == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "no fields to update")
	}

	var row models.Discussion
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "discussion not found")
		}
		return err
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "title is required")
		}
		if err := validateTitle(title); err != nil {
			return err
		}
		row.Title = title
	}
	if req.Content != nil {
		content := strings.TrimSpace(*req.Content)
		if content == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "content is required")
		}
		if err := validateTextBytes(content, limits.MaxMarkdownBytes, "discussion content is too large"); err != nil {
			return err
		}
		row.Content = content
	}
	if req.Tags != nil {
		rawTags, _ := json.Marshal(normalizeTags(*req.Tags))
		row.Tags = rawTags
	}
	if req.Pinned != nil {
		row.Pinned = *req.Pinned
	}
	if req.Locked != nil {
		row.Locked = *req.Locked
	}
	if err := api.db.Save(&row).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusOK, contract.CreatedID{ID: row.ID})
}

func (api *API) deleteDiscussion(c echo.Context) error {
	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid discussion id")
	if err != nil {
		return err
	}

	var row models.Discussion
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "discussion not found")
		}
		return err
	}
	if !user.Admin && row.UserID != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, "discussion owner required")
	}
	if err := api.db.Delete(&row).Error; err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (api *API) createComment(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid discussion id")
	}
	var req contract.CommentCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "content is required")
	}
	if err := validateTextBytes(req.Content, limits.MaxShortTextBytes, "comment content is too large"); err != nil {
		return err
	}

	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	var discussion models.Discussion
	if err := api.db.First(&discussion, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "discussion not found")
		}
		return err
	}
	if discussion.Locked {
		return echo.NewHTTPError(http.StatusForbidden, "discussion is locked")
	}
	row := models.Comment{DiscussionID: uint(id), UserID: user.ID, Content: req.Content}
	if err := api.db.Create(&row).Error; err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, contract.Comment{ID: row.ID, Author: user.Name, Content: row.Content, CreatedAt: row.CreatedAt})
}

func (api *API) deleteComment(c echo.Context) error {
	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	discussionID, err := parseID(c, "id", "invalid discussion id")
	if err != nil {
		return err
	}
	commentID, err := parseID(c, "commentId", "invalid comment id")
	if err != nil {
		return err
	}
	var comment models.Comment
	if err := api.db.First(&comment, "id = ? AND discussion_id = ?", commentID, discussionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "comment not found")
		}
		return err
	}
	if !user.Admin && comment.UserID != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, "comment owner required")
	}
	if err := api.db.Delete(&comment).Error; err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func discussionViewFromRefs(row models.Discussion, authors map[uint]string, replies map[uint]int) contract.Discussion {
	return contract.Discussion{
		ID:        row.ID,
		Title:     row.Title,
		Author:    authorName(row.UserID, authors),
		Tags:      readTags([]byte(row.Tags)),
		Pinned:    row.Pinned,
		Locked:    row.Locked,
		Replies:   replies[row.ID],
		CreatedAt: row.CreatedAt,
	}
}

func (api *API) discussionReplyCounts(ids []uint) (map[uint]int, error) {
	ids = uniqueUint(ids)
	counts := map[uint]int{}
	if len(ids) == 0 {
		return counts, nil
	}
	var rows []struct {
		DiscussionID uint
		Count        int64
	}
	if err := api.db.Model(&models.Comment{}).
		Select("discussion_id, count(*) as count").
		Where("discussion_id IN ?", ids).
		Group("discussion_id").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.DiscussionID] = int(row.Count)
	}
	return counts, nil
}

func discussionProblemIDs(item contract.Discussion) []uint {
	ids := []uint{}
	for _, tag := range item.Tags {
		upper := strings.ToUpper(strings.TrimSpace(tag))
		if !strings.HasPrefix(upper, "P") {
			continue
		}
		id, err := strconv.ParseUint(strings.TrimPrefix(upper, "P"), 10, 64)
		if err == nil {
			ids = append(ids, uint(id))
		}
	}
	return ids
}

func baseDiscussionDetails(now time.Time) []contract.DiscussionDetail {
	return []contract.DiscussionDetail{
		{
			Discussion: contract.Discussion{ID: 1, Title: "A+B Problem 有哪些边界情况？", Author: "admin", Tags: []string{"P1000", "beginner"}, Pinned: true, CreatedAt: now.Add(-3 * time.Hour)},
			Content:    "这题主要覆盖输入输出链路，也用来做第一批评测 smoke。\n\n```cpp\nlong long a, b;\ncin >> a >> b;\ncout << a + b << '\\n';\n```",
			Comments: []contract.Comment{
				{ID: 1, Author: "student", Content: "需要考虑负数吗？", CreatedAt: now.Add(-2 * time.Hour)},
				{ID: 2, Author: "admin", Content: "需要，数据范围会包含负数。", CreatedAt: now.Add(-90 * time.Minute)},
			},
		},
		{
			Discussion: contract.Discussion{ID: 2, Title: "Limits Hash 的数据范围讨论", Author: "student", Tags: []string{"P1001"}, CreatedAt: now.Add(-24 * time.Hour)},
			Content:    "这题重点是哈希边界和时间限制，建议先用朴素实现确认结果，再优化。",
			Comments: []contract.Comment{
				{ID: 3, Author: "student", Content: "严格模式下空格不一致会怎样？", CreatedAt: now.Add(-22 * time.Hour)},
			},
		},
		{
			Discussion: contract.Discussion{ID: 3, Title: "交互题提交时需要注意什么？", Author: "admin", Tags: []string{"P1002", "interactive"}, Locked: true, CreatedAt: now.Add(-48 * time.Hour)},
			Content:    "交互题会使用同一套 JudgeProgram/UserProgram pipe 模型。提交时要及时 flush 输出。",
			Comments:   []contract.Comment{},
		},
	}
}

func updatedDiscussion(item contract.DiscussionDetail, req contract.DiscussionUpdate) contract.DiscussionDetail {
	if req.Title != nil {
		item.Discussion.Title = *req.Title
	}
	if req.Tags != nil {
		item.Discussion.Tags = *req.Tags
	}
	if req.Pinned != nil {
		item.Discussion.Pinned = *req.Pinned
	}
	if req.Locked != nil {
		item.Discussion.Locked = *req.Locked
	}
	if req.Content != nil {
		item.Content = *req.Content
	}
	return item
}

func filterDiscussions(items []contract.Discussion, tag string) []contract.Discussion {
	if tag == "" {
		return items
	}
	filtered := make([]contract.Discussion, 0, len(items))
	for _, item := range items {
		if hasTag(item.Tags, tag) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
