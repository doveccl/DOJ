package admin

import (
	"net/http"
	"strings"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (api *API) groupsPage(c echo.Context) error {
	page, pageSize, err := parsePage(c)
	if err != nil {
		return err
	}
	items, total, err := api.searchGroupsPage(c.QueryParam("q"), pageSize, (page-1)*pageSize)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, PageResult[Group]{Items: items, Page: page, PageSize: pageSize, Total: total})
}

func (api *API) createGroup(c echo.Context) error {
	var req GroupUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Users = cleanUintList(req.Users)
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "group name is required")
	}
	if len([]rune(req.Name)) > maxNameRunes {
		return echo.NewHTTPError(http.StatusBadRequest, "group name is too long")
	}
	if err := api.ensureUsers(req.Users); err != nil {
		return err
	}

	if err := api.db.Transaction(func(tx *gorm.DB) error {
		group := models.Group{Name: req.Name}
		if err := tx.Create(&group).Error; err != nil {
			return err
		}
		return saveGroupUsers(tx, group.ID, req.Users)
	}); err != nil {
		return err
	}
	members, err := api.readMembers()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, members)
}

func (api *API) updateGroup(c echo.Context) error {
	id, err := parseUintParam(c, "id", "invalid group id")
	if err != nil {
		return err
	}
	var req GroupUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Users = cleanUintList(req.Users)
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "group name is required")
	}
	if len([]rune(req.Name)) > maxNameRunes {
		return echo.NewHTTPError(http.StatusBadRequest, "group name is too long")
	}
	if err := api.ensureUsers(req.Users); err != nil {
		return err
	}

	err = api.db.Transaction(func(tx *gorm.DB) error {
		updated := tx.Model(&models.Group{}).Where("id = ?", id).Update("name", req.Name)
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected == 0 {
			return echo.NewHTTPError(http.StatusNotFound, "group not found")
		}
		return saveGroupUsers(tx, id, req.Users)
	})
	if err != nil {
		return err
	}
	members, err := api.readMembers()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, members)
}

func (api *API) deleteGroup(c echo.Context) error {
	id, err := parseUintParam(c, "id", "invalid group id")
	if err != nil {
		return err
	}

	err = api.db.Transaction(func(tx *gorm.DB) error {
		var row models.Group
		if err := tx.First(&row, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return echo.NewHTTPError(http.StatusNotFound, "group not found")
			}
			return err
		}
		if err := tx.Where("group_id = ?", row.ID).Delete(&models.GroupUser{}).Error; err != nil {
			return err
		}
		if err := tx.Where("group_id = ?", row.ID).Delete(&models.AssignmentGroup{}).Error; err != nil {
			return err
		}
		return tx.Delete(&row).Error
	})
	if err != nil {
		return err
	}
	members, err := api.readMembers()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, members)
}

func (api *API) groups() ([]Group, error) {
	return api.searchGroups("", nil, 200)
}

func (api *API) searchGroups(q string, includeIDs []uint, limit int) ([]Group, error) {
	var rows []models.Group
	query := api.db.Order("id asc").Limit(limit)
	q = strings.TrimSpace(q)
	includeIDs = cleanUintList(includeIDs)
	if q != "" {
		query = query.Where("LOWER(name) LIKE LOWER(?) OR id IN ?", "%"+q+"%", includeIDsOrZero(includeIDs))
	} else if len(includeIDs) > 0 {
		query = query.Where("id IN ?", includeIDs)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return api.groupDTOs(rows)
}

func (api *API) searchGroupsPage(q string, limit int, offset int) ([]Group, int64, error) {
	query := api.db.Model(&models.Group{})
	q = strings.TrimSpace(q)
	if q != "" {
		like := "%" + q + "%"
		query = query.Where(`
			LOWER(groups.name) LIKE LOWER(?)
			OR EXISTS (
				SELECT 1 FROM group_users
				JOIN users ON users.id = group_users.user_id AND users.deleted_at IS NULL
				WHERE group_users.group_id = groups.id
				AND LOWER(users.name) LIKE LOWER(?)
			)
		`, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.Group
	if err := query.Order("groups.id asc").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items, err := api.groupDTOs(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (api *API) groupDTOs(rows []models.Group) ([]Group, error) {
	groupIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		groupIDs = append(groupIDs, row.ID)
	}
	userMap := map[uint][]uint{}
	if len(groupIDs) > 0 {
		var links []models.GroupUser
		if err := api.db.Table("group_users").
			Select("group_users.group_id, group_users.user_id").
			Joins("JOIN users ON users.id = group_users.user_id AND users.deleted_at IS NULL").
			Where("group_users.group_id IN ?", groupIDs).
			Order("group_users.user_id asc").
			Find(&links).Error; err != nil {
			return nil, err
		}
		for _, link := range links {
			userMap[link.GroupID] = append(userMap[link.GroupID], link.UserID)
		}
	}
	items := make([]Group, 0, len(rows))
	for _, row := range rows {
		items = append(items, Group{ID: row.ID, Name: row.Name, Users: cleanUintList(userMap[row.ID])})
	}
	return items, nil
}

func (api *API) ensureGroups(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	var count int64
	if err := api.db.Model(&models.Group{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid group")
	}
	return nil
}

func (api *API) ensureUsers(ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	var count int64
	if err := api.db.Model(&models.User{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user")
	}
	return nil
}

func saveGroupUsers(tx *gorm.DB, groupID uint, users []uint) error {
	if err := tx.Where("group_id = ?", groupID).Delete(&models.GroupUser{}).Error; err != nil {
		return err
	}
	items := make([]models.GroupUser, 0, len(users))
	for _, id := range users {
		items = append(items, models.GroupUser{GroupID: groupID, UserID: id})
	}
	if len(items) > 0 {
		return tx.Create(&items).Error
	}
	return nil
}
