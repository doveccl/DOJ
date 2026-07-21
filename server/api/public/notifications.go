package public

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/events"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const maxMentions = 10

var mentionPattern = regexp.MustCompile(`@[A-Za-z0-9_-]+`)

func mentionKeys(content string) map[string]bool {
	keys := map[string]bool{}
	for _, match := range mentionPattern.FindAllStringIndex(content, -1) {
		if match[0] > 0 && mentionNameByte(content[match[0]-1]) {
			continue
		}
		name := content[match[0]+1 : match[1]]
		if len(name) < models.UserNameMin || len(name) > models.UserNameMax {
			continue
		}
		keys[strings.ToLower(name)] = true
	}
	return keys
}

func mentionNameByte(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' || value == '_' || value == '-' || value == '@'
}

func (api *API) createDiscussionNotifications(tx *gorm.DB, actorID uint, discussion models.Discussion, commentID uint, content string, previous string, reply bool) (bool, error) {
	targets := map[uint]string{}
	if reply && discussion.UserID != actorID {
		targets[discussion.UserID] = "reply"
	}

	current := mentionKeys(content)
	previousMentions := mentionKeys(previous)
	keys := make([]string, 0, len(current))
	for key := range current {
		if !previousMentions[key] {
			keys = append(keys, key)
		}
	}
	if len(keys) > maxMentions {
		return false, echo.NewHTTPError(http.StatusBadRequest, "too many mentions")
	}
	if len(keys) > 0 {
		var users []models.User
		if err := tx.Select("id", "name").Where("LOWER(name) IN ?", keys).Find(&users).Error; err != nil {
			return false, err
		}
		for _, user := range users {
			if user.ID == actorID {
				continue
			}
			targets[user.ID] = "mention"
		}
	}

	if len(targets) == 0 {
		return false, nil
	}
	rows := make([]models.Notification, 0, len(targets))
	for userID, kind := range targets {
		rows = append(rows, models.Notification{
			UserID:       userID,
			ActorID:      actorID,
			Kind:         kind,
			DiscussionID: discussion.ID,
			CommentID:    commentID,
		})
	}
	result := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "discussion_id"}, {Name: "comment_id"}},
		DoNothing: true,
	}).Create(&rows)
	return result.RowsAffected > 0, result.Error
}

func (api *API) notifications(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	page, pageSize, offset, err := parsePage(c)
	if err != nil {
		return err
	}
	query := api.db.Model(&models.Notification{}).Where("user_id = ?", user.ID)
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return err
	}
	var unread int64
	if err := query.Session(&gorm.Session{}).Where("read_at IS NULL").Count(&unread).Error; err != nil {
		return err
	}
	var rows []models.Notification
	if err := query.Session(&gorm.Session{}).
		Order("created_at desc, id desc").
		Limit(pageSize).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return err
	}
	actorIDs := make([]uint, 0, len(rows))
	discussionIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		actorIDs = append(actorIDs, row.ActorID)
		discussionIDs = append(discussionIDs, row.DiscussionID)
	}
	actors, err := api.userMap(actorIDs)
	if err != nil {
		return err
	}
	var discussions []models.Discussion
	if len(discussionIDs) > 0 {
		if err := api.db.Select("id", "title").Where("id IN ?", uniqueUint(discussionIDs)).Find(&discussions).Error; err != nil {
			return err
		}
	}
	titles := make(map[uint]string, len(discussions))
	for _, discussion := range discussions {
		titles[discussion.ID] = discussion.Title
	}
	items := make([]contract.Notification, 0, len(rows))
	for _, row := range rows {
		var commentID *uint
		if row.CommentID > 0 {
			value := row.CommentID
			commentID = &value
		}
		items = append(items, contract.Notification{
			ID:              row.ID,
			Kind:            row.Kind,
			Actor:           authorName(row.ActorID, actors),
			Avatar:          actors[row.ActorID].Avatar,
			DiscussionID:    row.DiscussionID,
			DiscussionTitle: titles[row.DiscussionID],
			CommentID:       commentID,
			Read:            row.ReadAt != nil,
			CreatedAt:       row.CreatedAt,
		})
	}
	return c.JSON(http.StatusOK, contract.NotificationPage{Items: items, Unread: unread, Page: page, PageSize: pageSize, Total: total})
}

func (api *API) readNotification(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid notification id")
	if err != nil {
		return err
	}
	result := api.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ? AND read_at IS NULL", id, user.ID).
		Update("read_at", time.Now())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		events.NotificationChanged()
	}
	return c.NoContent(http.StatusNoContent)
}

func (api *API) readAllNotifications(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	result := api.db.Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL", user.ID).
		Update("read_at", time.Now())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		events.NotificationChanged()
	}
	return c.NoContent(http.StatusNoContent)
}
