package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (api *API) assignments(c echo.Context) error {

	page, pageSize, offset, err := parsePage(c)
	if err != nil {
		return err
	}
	var rows []models.Assignment
	query := api.db.Model(&models.Assignment{})
	if !api.isAdmin(c) {
		user, err := api.currentUser(c)
		if err != nil {
			return c.JSON(http.StatusOK, PageResult[AssignmentDTO]{Items: []AssignmentDTO{}, Page: page, PageSize: pageSize, Total: 0})
		}
		query = query.Where(`
			EXISTS (
				SELECT 1 FROM assignment_users
				WHERE assignment_users.assignment_id = assignments.id
				AND assignment_users.user_id = ?
			)
			OR EXISTS (
				SELECT 1 FROM assignment_groups
				JOIN group_users ON group_users.group_id = assignment_groups.group_id
				WHERE assignment_groups.assignment_id = assignments.id
				AND group_users.user_id = ?
			)
		`, user.ID, user.ID)
	}
	if q := strings.TrimSpace(c.QueryParam("q")); q != "" {
		like := "%" + q + "%"
		if id, err := parseQueryID(q, "invalid assignment id"); err == nil {
			query = query.Where("id = ? OR LOWER(title) LIKE LOWER(?)", id, like)
		} else {
			query = query.Where("LOWER(title) LIKE LOWER(?)", like)
		}
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return err
	}
	if err := query.Session(&gorm.Session{}).Order("end_at desc").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return err
	}
	items, err := api.assignmentDTOs(c, rows, false)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, PageResult[AssignmentDTO]{Items: items, Page: page, PageSize: pageSize, Total: total})
}

func (api *API) createAssignment(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	var req AssignmentCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if err := validateTitle(req.Title); err != nil {
		return err
	}
	endAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid deadline")
	}

	req.Problems = normalizeProblemRefs(req.Problems)
	if err := api.validateProblemRefs(req.Problems); err != nil {
		return err
	}
	req.Users = cleanUintList(req.Users)
	req.Groups = cleanUintList(req.Groups)
	if err := api.validateUserIDs(req.Users); err != nil {
		return err
	}
	if err := api.validateGroupIDs(req.Groups); err != nil {
		return err
	}
	row := models.Assignment{Title: req.Title, EndAt: endAt}
	if err := api.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		for _, item := range req.Problems {
			link := models.AssignmentProblem{AssignmentID: row.ID, ProblemID: item.ID, Sort: item.Sort}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		if err := saveAssignmentMembers(tx, row.ID, req.Users, req.Groups); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, CreatedID{ID: row.ID})
}

func (api *API) updateAssignment(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid assignment id")
	if err != nil {
		return err
	}
	var req AssignmentUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if err := validateTitle(req.Title); err != nil {
		return err
	}
	endAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid deadline")
	}

	var row models.Assignment
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "assignment not found")
		}
		return err
	}
	req.Problems = normalizeProblemRefs(req.Problems)
	if err := api.validateProblemRefs(req.Problems); err != nil {
		return err
	}
	req.Users = cleanUintList(req.Users)
	req.Groups = cleanUintList(req.Groups)
	if err := api.validateUserIDs(req.Users); err != nil {
		return err
	}
	if err := api.validateGroupIDs(req.Groups); err != nil {
		return err
	}
	row.Title = req.Title
	row.EndAt = endAt
	if err := api.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		if err := tx.Where("assignment_id = ?", row.ID).Delete(&models.AssignmentProblem{}).Error; err != nil {
			return err
		}
		for _, item := range req.Problems {
			link := models.AssignmentProblem{AssignmentID: row.ID, ProblemID: item.ID, Sort: item.Sort}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		if err := saveAssignmentMembers(tx, row.ID, req.Users, req.Groups); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, CreatedID{ID: row.ID})
}

func (api *API) deleteAssignment(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid assignment id")
	if err != nil {
		return err
	}

	if err := api.db.Delete(&models.Assignment{}, id).Error; err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (api *API) assignment(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid assignment id")
	}

	var row models.Assignment
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "assignment not found")
		}
		return err
	}
	allowed, err := api.assignmentVisible(c, row.ID)
	if err != nil {
		return err
	}
	if !allowed {
		return echo.NewHTTPError(http.StatusNotFound, "assignment not found")
	}
	var links []models.AssignmentProblem
	if err := api.db.Where("assignment_id = ?", row.ID).Order("sort asc").Find(&links).Error; err != nil {
		return err
	}
	problems, err := api.assignmentProblems(c, row, links)
	if err != nil {
		return err
	}
	progressRows, err := api.assignmentProgress(c, row.ID, problems)
	if err != nil {
		return err
	}
	done, err := api.assignmentDoneCount(c, row.ID)
	if err != nil {
		return err
	}
	total, err := api.assignmentProblemCount(row.ID)
	if err != nil {
		return err
	}
	dto, err := api.assignmentDTO(c, row, total, done)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, AssignmentDetail{Assignment: dto, Problems: problems, Progress: progressRows})
}

func (api *API) activeAssignmentFor(userID uint, problemID uint, now time.Time) (*uint, error) {
	var rows []models.Assignment
	if err := api.db.
		Joins("JOIN assignment_problems ON assignment_problems.assignment_id = assignments.id").
		Where("assignment_problems.problem_id = ? AND assignments.end_at >= ?", problemID, now).
		Order("assignments.end_at asc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		ok, err := api.userAssignedTo(row.ID, userID)
		if err != nil {
			return nil, err
		}
		if ok {
			id := row.ID
			return &id, nil
		}
	}
	return nil, nil
}

func (api *API) userAssignedTo(assignmentID uint, userID uint) (bool, error) {
	var direct int64
	if err := api.db.Model(&models.AssignmentUser{}).
		Where("assignment_id = ? AND user_id = ?", assignmentID, userID).
		Count(&direct).Error; err != nil {
		return false, err
	}
	if direct > 0 {
		return true, nil
	}
	var byGroup int64
	if err := api.db.Model(&models.AssignmentGroup{}).
		Joins("JOIN group_users ON group_users.group_id = assignment_groups.group_id").
		Where("assignment_groups.assignment_id = ? AND group_users.user_id = ?", assignmentID, userID).
		Count(&byGroup).Error; err != nil {
		return false, err
	}
	return byGroup > 0, nil
}

func (api *API) assignmentVisible(c echo.Context, assignmentID uint) (bool, error) {
	if api.isAdmin(c) {
		return true, nil
	}
	if api.role(c) == "guest" {
		return false, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return false, err
	}
	return api.userAssignedTo(assignmentID, user.ID)
}

func saveAssignmentMembers(tx *gorm.DB, assignmentID uint, users []uint, groups []uint) error {
	if err := tx.Where("assignment_id = ?", assignmentID).Delete(&models.AssignmentUser{}).Error; err != nil {
		return err
	}
	if err := tx.Where("assignment_id = ?", assignmentID).Delete(&models.AssignmentGroup{}).Error; err != nil {
		return err
	}
	for _, userID := range users {
		if err := tx.Create(&models.AssignmentUser{AssignmentID: assignmentID, UserID: userID}).Error; err != nil {
			return err
		}
	}
	for _, groupID := range groups {
		if err := tx.Create(&models.AssignmentGroup{AssignmentID: assignmentID, GroupID: groupID}).Error; err != nil {
			return err
		}
	}
	return nil
}

func assignmentEnded(row models.Assignment) bool {
	return !time.Now().Before(row.EndAt)
}
