package admin

import (
	"net/http"
	"strings"

	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/validate"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func (api *API) usersPage(c echo.Context) error {
	page, pageSize, err := parsePage(c)
	if err != nil {
		return err
	}
	items, total, err := api.searchUsersPage(c.QueryParam("q"), pageSize, (page-1)*pageSize)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, PageResult[User]{Items: items, Page: page, PageSize: pageSize, Total: total})
}

func (api *API) createUser(c echo.Context) error {
	var req UserCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	cleanUserCreate(&req)
	if err := validateUserCreate(req); err != nil {
		return err
	}
	if err := api.ensureGroups(req.Groups); err != nil {
		return err
	}

	var count int64
	if err := api.db.Model(&models.User{}).Where("LOWER(name) = ? OR LOWER(mail) = ?", validate.NameKey(req.Name), req.Mail).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return echo.NewHTTPError(http.StatusConflict, "user already exists")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = api.db.Transaction(func(tx *gorm.DB) error {
		user := models.User{
			Name:  req.Name,
			Mail:  req.Mail,
			Auth:  string(hash),
			Admin: req.Role == "admin",
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		items := make([]models.GroupUser, 0, len(req.Groups))
		for _, id := range req.Groups {
			items = append(items, models.GroupUser{UserID: user.ID, GroupID: id})
		}
		if len(items) > 0 {
			return tx.Create(&items).Error
		}
		return nil
	})
	if err != nil {
		return err
	}
	members, err := api.readMembers()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, members)
}

func (api *API) updateUser(c echo.Context) error {
	var req UserUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Role = strings.TrimSpace(req.Role)
	if req.Role != "admin" && req.Role != "user" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role")
	}
	req.Groups = cleanUintList(req.Groups)

	row, err := api.userByName(c.Param("name"))
	if err != nil {
		return err
	}
	if row.Admin && req.Role != "admin" {
		if err := api.ensureOtherAdmin(row.ID); err != nil {
			return err
		}
	}
	if err := api.ensureGroups(req.Groups); err != nil {
		return err
	}
	err = api.db.Transaction(func(tx *gorm.DB) error {
		row.Admin = req.Role == "admin"
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", row.ID).Delete(&models.GroupUser{}).Error; err != nil {
			return err
		}
		items := make([]models.GroupUser, 0, len(req.Groups))
		for _, id := range req.Groups {
			items = append(items, models.GroupUser{UserID: row.ID, GroupID: id})
		}
		if len(items) > 0 {
			return tx.Create(&items).Error
		}
		return nil
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

func (api *API) deleteUser(c echo.Context) error {
	row, err := api.userByName(c.Param("name"))
	if err != nil {
		return err
	}
	if row.Admin {
		if err := api.ensureOtherAdmin(row.ID); err != nil {
			return err
		}
	}
	if err := api.db.Delete(&row).Error; err != nil {
		return err
	}
	members, err := api.readMembers()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, members)
}

func (api *API) resetUserPassword(c echo.Context) error {
	row, err := api.userByName(c.Param("name"))
	if err != nil {
		return err
	}
	password, err := randomPassword()
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	updated := api.db.Model(&models.User{}).Where("id = ?", row.ID).Update("auth", string(hash))
	if updated.Error != nil {
		return updated.Error
	}
	return c.JSON(http.StatusOK, PasswordReset{Password: password})
}

func (api *API) ensureOtherAdmin(userID uint) error {
	var otherAdmins int64
	if err := api.db.Model(&models.User{}).Where("id <> ? AND admin = ?", userID, true).Count(&otherAdmins).Error; err != nil {
		return err
	}
	if otherAdmins == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "cannot remove the last admin")
	}
	return nil
}

func (api *API) userByName(name string) (models.User, error) {
	var row models.User
	nameKey := validate.NameKey(name)
	if nameKey == "" {
		return row, echo.NewHTTPError(http.StatusBadRequest, "user name is required")
	}
	if err := api.db.Where("LOWER(name) = ?", nameKey).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return row, echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return row, err
	}
	return row, nil
}

func (api *API) users() ([]User, error) {
	return api.searchUsers("", nil, 200)
}

func (api *API) searchUsers(q string, includeIDs []uint, limit int) ([]User, error) {
	var rows []models.User
	query := api.db.Order("id asc").Limit(limit)
	q = strings.TrimSpace(q)
	includeIDs = cleanUintList(includeIDs)
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("LOWER(name) LIKE LOWER(?) OR LOWER(mail) LIKE LOWER(?) OR id IN ?", like, like, includeIDsOrZero(includeIDs))
	} else if len(includeIDs) > 0 {
		query = query.Where("id IN ?", includeIDs)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return api.userViews(rows)
}

func (api *API) searchUsersPage(q string, limit int, offset int) ([]User, int64, error) {
	query := api.db.Model(&models.User{})
	q = strings.TrimSpace(q)
	if q != "" {
		like := "%" + q + "%"
		query = query.Where(`
			LOWER(users.name) LIKE LOWER(?)
			OR LOWER(users.mail) LIKE LOWER(?)
			OR EXISTS (
				SELECT 1 FROM group_users
				JOIN groups ON groups.id = group_users.group_id
				WHERE group_users.user_id = users.id
				AND LOWER(groups.name) LIKE LOWER(?)
			)
		`, like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.User
	if err := query.Order("users.id asc").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	items, err := api.userViews(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (api *API) userViews(rows []models.User) ([]User, error) {
	userIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		userIDs = append(userIDs, row.ID)
	}
	groupMap := map[uint][]uint{}
	if len(userIDs) > 0 {
		var links []models.GroupUser
		err := api.db.Table("group_users").
			Select("group_users.group_id, group_users.user_id").
			Joins("JOIN groups ON groups.id = group_users.group_id").
			Where("group_users.user_id IN ?", userIDs).
			Find(&links).Error
		if err != nil {
			return nil, err
		}
		for _, link := range links {
			groupMap[link.UserID] = append(groupMap[link.UserID], link.GroupID)
		}
	}
	items := make([]User, 0, len(rows))
	for _, row := range rows {
		role := "user"
		if row.Admin {
			role = "admin"
		}
		items = append(items, User{ID: row.ID, Name: row.Name, Mail: row.Mail, Role: role, Groups: cleanUintList(groupMap[row.ID])})
	}
	return items, nil
}

func cleanUserCreate(req *UserCreate) {
	req.Name = strings.TrimSpace(req.Name)
	req.Mail = strings.ToLower(strings.TrimSpace(req.Mail))
	req.Role = strings.TrimSpace(req.Role)
	req.Groups = cleanUintList(req.Groups)
}

func validateUserCreate(req UserCreate) error {
	if len(req.Name) < models.UserNameMin || len(req.Name) > models.UserNameMax || !validate.Name(req.Name) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid username")
	}
	if !validate.Mail(req.Mail, models.MailMax, false) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid mail")
	}
	if len(req.Password) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "password is too short")
	}
	if req.Role != "admin" && req.Role != "user" {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role")
	}
	return nil
}
