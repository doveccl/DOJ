package public

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/doveccl/doj/contract/limits"
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
	query, err = api.applyDiscussionVisibility(c, query)
	if err != nil {
		return err
	}
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
		Select("id", "problem_id", "title", "user_id", "tags", "pinned", "locked", "created_at").
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
	problemID, err := discussionProblemID(req.Tags)
	if err != nil {
		return err
	}
	if problemID != nil {
		if err := api.requireDiscussionProblem(c, problemID); err != nil {
			return err
		}
	}
	rawTags, _ := json.Marshal(req.Tags)
	row := models.Discussion{
		ProblemID: problemID,
		Title:     req.Title,
		Content:   req.Content,
		UserID:    user.ID,
		Tags:      rawTags,
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
	page, pageSize, offset, err := parsePage(c)
	if err != nil {
		return err
	}

	row, err := api.discussionByID(c, uint(id))
	if err != nil {
		return err
	}
	commentsQuery := api.db.Unscoped().Model(&models.Comment{}).Where("discussion_id = ?", row.ID)
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
	authors, err := api.userNameMap(authorIDs)
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
		ProblemID: row.ProblemID,
		Title:     row.Title,
		Author:    authorName(row.UserID, authors),
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

	row, err := api.discussionByID(c, id)
	if err != nil {
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
		tags := normalizeTags(*req.Tags)
		problemID, err := discussionProblemID(tags)
		if err != nil {
			return err
		}
		if err := api.requireDiscussionProblem(c, problemID); err != nil {
			return err
		}
		rawTags, _ := json.Marshal(tags)
		row.Tags = rawTags
		row.ProblemID = problemID
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

	row, err := api.discussionByID(c, id)
	if err != nil {
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
	discussion, err := api.discussionByID(c, uint(id))
	if err != nil {
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
	if _, err := api.discussionByID(c, discussionID); err != nil {
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
		ProblemID: row.ProblemID,
		Title:     row.Title,
		Author:    authorName(row.UserID, authors),
		Tags:      readTags([]byte(row.Tags)),
		Pinned:    row.Pinned,
		Locked:    row.Locked,
		Replies:   replies[row.ID],
		CreatedAt: row.CreatedAt,
	}
}

func (api *API) applyDiscussionVisibility(c echo.Context, query *gorm.DB) (*gorm.DB, error) {
	clause, args, err := api.discussionVisibility(c)
	if err != nil {
		return nil, err
	}
	return query.Where(clause, args...), nil
}

func (api *API) discussionVisibility(c echo.Context) (string, []any, error) {
	if api.isAdmin(c) {
		return `discussions.problem_id IS NULL OR EXISTS (
			SELECT 1 FROM problems
			WHERE problems.id = discussions.problem_id AND problems.deleted_at IS NULL
		)`, nil, nil
	}
	viewerID, err := api.viewerID(c)
	if err != nil {
		return "", nil, err
	}
	now := time.Now()
	clause := `discussions.problem_id IS NULL OR EXISTS (
		SELECT 1 FROM problems
		WHERE problems.id = discussions.problem_id
			AND problems.deleted_at IS NULL
			AND NOT EXISTS (
				SELECT 1 FROM contest_problems
				JOIN contests ON contests.id = contest_problems.contest_id
				WHERE contest_problems.problem_id = problems.id
					AND contests.deleted_at IS NULL
					AND contests.end_at > ?
			)
			AND (
				problems.visible = ?
				OR EXISTS (
					SELECT 1 FROM contest_problems ended_links
					JOIN contests ended_contests ON ended_contests.id = ended_links.contest_id
					WHERE ended_links.problem_id = problems.id
						AND ended_contests.deleted_at IS NULL
						AND ended_contests.end_at <= ?
				)
				OR (? <> 0 AND EXISTS (
					SELECT 1 FROM assignment_problems
					JOIN assignments ON assignments.id = assignment_problems.assignment_id
					WHERE assignment_problems.problem_id = problems.id
						AND assignments.deleted_at IS NULL
						AND (
							EXISTS (SELECT 1 FROM assignment_users WHERE assignment_users.assignment_id = assignments.id AND assignment_users.user_id = ?)
							OR EXISTS (
								SELECT 1 FROM assignment_groups
								JOIN group_users ON group_users.group_id = assignment_groups.group_id
								WHERE assignment_groups.assignment_id = assignments.id AND group_users.user_id = ?
							)
						)
				))
			)
	)`
	return clause, []any{now, true, now, viewerID, viewerID, viewerID}, nil
}

func (api *API) discussionByID(c echo.Context, id uint) (models.Discussion, error) {
	query, err := api.applyDiscussionVisibility(c, api.db.Model(&models.Discussion{}))
	if err != nil {
		return models.Discussion{}, err
	}
	var row models.Discussion
	if err := query.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return row, echo.NewHTTPError(http.StatusNotFound, "discussion not found")
		}
		return row, err
	}
	return row, nil
}

func (api *API) requireDiscussionProblem(c echo.Context, problemID *uint) error {
	if problemID == nil {
		return nil
	}
	if !api.isAdmin(c) {
		if api.problemInUnfinishedContest(*problemID) {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return api.requireProblemVisible(c, *problemID)
	}
	var count int64
	if err := api.db.Model(&models.Problem{}).Where("id = ?", *problemID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "problem not found")
	}
	return nil
}

func discussionProblemID(tags []string) (*uint, error) {
	var problemID *uint
	for _, tag := range tags {
		upper := strings.ToUpper(strings.TrimSpace(tag))
		if !strings.HasPrefix(upper, "P") {
			continue
		}
		id, err := strconv.ParseUint(strings.TrimPrefix(upper, "P"), 10, 64)
		if err != nil || id == 0 {
			continue
		}
		value := uint(id)
		if problemID != nil && *problemID != value {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "discussion can reference only one problem")
		}
		problemID = &value
	}
	return problemID, nil
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
