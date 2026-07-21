package public

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/doveccl/doj/contract/limits"
	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/events"
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
	authors, err := api.userMap(authorIDs)
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
	notified := false
	if err := api.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		var err error
		notified, err = api.createDiscussionNotifications(tx, user.ID, row, 0, row.Content, "", false)
		return err
	}); err != nil {
		return err
	}
	if notified {
		events.NotificationChanged()
	}
	return c.JSON(http.StatusCreated, contract.CreatedID{ID: row.ID})
}

func (api *API) discussion(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid discussion id")
	}
	page, pageSize, offset, err := parsePage(c)
	if err != nil {
		return err
	}

	row, err := api.discussionByID(uint(id))
	if err != nil {
		return err
	}
	commentsQuery := api.db.Unscoped().Model(&models.Comment{}).Where("discussion_id = ?", row.ID)
	if value := strings.TrimSpace(c.QueryParam("comment")); value != "" {
		commentID, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid comment id")
		}
		var target models.Comment
		if err := commentsQuery.Session(&gorm.Session{}).Where("id = ?", commentID).First(&target).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return echo.NewHTTPError(http.StatusNotFound, "comment not found")
			}
			return err
		}
		var position int64
		if err := commentsQuery.Session(&gorm.Session{}).
			Where("(created_at < ? OR (created_at = ? AND id <= ?))", target.CreatedAt, target.CreatedAt, target.ID).
			Count(&position).Error; err != nil {
			return err
		}
		page = int((position-1)/int64(pageSize)) + 1
		offset = (page - 1) * pageSize
	}
	var total int64
	if err := commentsQuery.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return err
	}
	var comments []models.Comment
	if err := commentsQuery.Session(&gorm.Session{}).
		Select("id", "user_id", "content", "created_at", "deleted_at").
		Order("created_at asc, id asc").
		Limit(pageSize).
		Offset(offset).
		Find(&comments).Error; err != nil {
		return err
	}
	authorIDs := []uint{row.UserID}
	for _, item := range comments {
		authorIDs = append(authorIDs, item.UserID)
	}
	authors, err := api.userMap(authorIDs)
	if err != nil {
		return err
	}
	items := make([]contract.Comment, 0, len(comments))
	for _, item := range comments {
		deleted := item.DeletedAt.Valid
		content := item.Content
		if deleted {
			content = ""
		}
		items = append(items, contract.Comment{
			ID:        item.ID,
			Author:    authorName(item.UserID, authors),
			Avatar:    authors[item.UserID].Avatar,
			Content:   content,
			Deleted:   deleted,
			CreatedAt: item.CreatedAt,
		})
	}
	replies, err := api.discussionReplyCounts([]uint{row.ID})
	if err != nil {
		return err
	}
	item := contract.Discussion{
		ID:        row.ID,
		Title:     row.Title,
		Author:    authorName(row.UserID, authors),
		Avatar:    authors[row.UserID].Avatar,
		Tags:      readTags([]byte(row.Tags)),
		Pinned:    row.Pinned,
		Locked:    row.Locked,
		Replies:   replies[row.ID],
		CreatedAt: row.CreatedAt,
	}
	return c.JSON(http.StatusOK, contract.DiscussionDetail{
		Discussion: item,
		Content:    row.Content,
		Comments:   contract.Page[contract.Comment]{Items: items, Page: page, PageSize: pageSize, Total: total},
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

	row, err := api.discussionByID(id)
	if err != nil {
		return err
	}
	actor, err := api.currentUser(c)
	if err != nil {
		return err
	}
	previousContent := row.Content
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
		tags := normalizeTags(*req.Tags)
		rawTags, _ := json.Marshal(tags)
		row.Tags = rawTags
	}
	if req.Pinned != nil {
		row.Pinned = *req.Pinned
	}
	if req.Locked != nil {
		row.Locked = *req.Locked
	}
	notified := false
	if err := api.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		if req.Content == nil {
			return nil
		}
		var err error
		notified, err = api.createDiscussionNotifications(tx, actor.ID, row, 0, row.Content, previousContent, false)
		return err
	}); err != nil {
		return err
	}
	if notified {
		events.NotificationChanged()
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

	row, err := api.discussionByID(id)
	if err != nil {
		return err
	}
	if !user.Admin && row.UserID != user.ID {
		return echo.NewHTTPError(http.StatusForbidden, "discussion owner required")
	}
	notified := false
	if err := api.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("discussion_id = ?", row.ID).Delete(&models.Notification{})
		if result.Error != nil {
			return result.Error
		}
		notified = result.RowsAffected > 0
		return tx.Delete(&row).Error
	}); err != nil {
		return err
	}
	if notified {
		events.NotificationChanged()
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
	discussion, err := api.discussionByID(uint(id))
	if err != nil {
		return err
	}
	if discussion.Locked {
		return echo.NewHTTPError(http.StatusForbidden, "discussion is locked")
	}
	row := models.Comment{DiscussionID: uint(id), UserID: user.ID, Content: req.Content}
	notified := false
	if err := api.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		var err error
		notified, err = api.createDiscussionNotifications(tx, user.ID, discussion, row.ID, row.Content, "", true)
		return err
	}); err != nil {
		return err
	}
	if notified {
		events.NotificationChanged()
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
	if _, err := api.discussionByID(discussionID); err != nil {
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
	notified := false
	if err := api.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("discussion_id = ? AND comment_id = ?", discussionID, commentID).Delete(&models.Notification{})
		if result.Error != nil {
			return result.Error
		}
		notified = result.RowsAffected > 0
		return tx.Delete(&comment).Error
	}); err != nil {
		return err
	}
	if notified {
		events.NotificationChanged()
	}
	return c.NoContent(http.StatusNoContent)
}

func discussionViewFromRefs(row models.Discussion, authors map[uint]models.User, replies map[uint]int) contract.Discussion {
	return contract.Discussion{
		ID:        row.ID,
		Title:     row.Title,
		Author:    authorName(row.UserID, authors),
		Avatar:    authors[row.UserID].Avatar,
		Tags:      readTags([]byte(row.Tags)),
		Pinned:    row.Pinned,
		Locked:    row.Locked,
		Replies:   replies[row.ID],
		CreatedAt: row.CreatedAt,
	}
}

func (api *API) discussionByID(id uint) (models.Discussion, error) {
	var row models.Discussion
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return row, echo.NewHTTPError(http.StatusNotFound, "discussion not found")
		}
		return row, err
	}
	return row, nil
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
