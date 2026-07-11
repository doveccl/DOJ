package public

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	contract "github.com/doveccl/doj/contract/web"

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
			return c.JSON(http.StatusOK, contract.Page[contract.Assignment]{Items: []contract.Assignment{}, Page: page, PageSize: pageSize, Total: 0})
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
	items, err := api.assignmentViews(c, rows, false)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, contract.Page[contract.Assignment]{Items: items, Page: page, PageSize: pageSize, Total: total})
}

func (api *API) createAssignment(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	var req contract.AssignmentCreate
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
	if !endAt.After(time.Now()) {
		return echo.NewHTTPError(http.StatusBadRequest, "deadline must be in the future")
	}

	req.Problems = normalizeProblemRefs(req.Problems)
	if err := api.validateProblemRefs(req.Problems); err != nil {
		return err
	}
	if err := api.ensureProblemRefsReady(c.Request().Context(), req.Problems); err != nil {
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
	return c.JSON(http.StatusCreated, contract.CreatedID{ID: row.ID})
}

func (api *API) updateAssignment(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid assignment id")
	if err != nil {
		return err
	}
	var req contract.AssignmentUpdate
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
	locked, err := api.assignmentRulesLocked(row)
	if err != nil {
		return err
	}
	if locked {
		var links []models.AssignmentProblem
		if err := api.db.Where("assignment_id = ?", row.ID).Find(&links).Error; err != nil {
			return err
		}
		users, groups, err := api.assignmentMembers(row.ID)
		if err != nil {
			return err
		}
		refs := make([]contract.ProblemRef, 0, len(links))
		for _, link := range links {
			refs = append(refs, contract.ProblemRef{ID: link.ProblemID, Sort: link.Sort})
		}
		if !sameProblemRefs(req.Problems, refs) || !sameUintSet(req.Users, users) || !sameUintSet(req.Groups, groups) {
			return echo.NewHTTPError(http.StatusConflict, "assignment scope is locked after the deadline or first submission")
		}
		if endAt.Before(row.EndAt) || (!time.Now().Before(row.EndAt) && !endAt.Equal(row.EndAt)) {
			return echo.NewHTTPError(http.StatusConflict, "assignment deadline can only be extended while active")
		}
	} else if err := api.ensureProblemRefsReady(c.Request().Context(), req.Problems); err != nil {
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
	return c.JSON(http.StatusOK, contract.CreatedID{ID: row.ID})
}

func (api *API) deleteAssignment(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid assignment id")
	if err != nil {
		return err
	}

	var row models.Assignment
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "assignment not found")
		}
		return err
	}
	locked, err := api.assignmentRulesLocked(row)
	if err != nil {
		return err
	}
	if locked {
		return echo.NewHTTPError(http.StatusConflict, "used assignment cannot be deleted")
	}
	if err := api.db.Transaction(func(tx *gorm.DB) error {
		for _, model := range []any{&models.AssignmentProblem{}, &models.AssignmentUser{}, &models.AssignmentGroup{}} {
			if err := tx.Where("assignment_id = ?", row.ID).Delete(model).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&row).Error
	}); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (api *API) assignmentRulesLocked(row models.Assignment) (bool, error) {
	if !time.Now().Before(row.EndAt) {
		return true, nil
	}
	var submissions int64
	if err := api.db.Model(&models.Submission{}).Where("assignment_id = ?", row.ID).Count(&submissions).Error; err != nil {
		return false, err
	}
	return submissions > 0, nil
}

func sameUintSet(left []uint, right []uint) bool {
	if len(left) != len(right) {
		return false
	}
	values := make(map[uint]struct{}, len(right))
	for _, value := range right {
		values[value] = struct{}{}
	}
	for _, value := range left {
		if _, ok := values[value]; !ok {
			return false
		}
	}
	return true
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
	problems, err := api.assignmentProblems(links)
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
	assignment, err := api.assignmentView(c, row, total, done)
	if err != nil {
		return err
	}
	description, err := api.assignmentDescription(c.Request().Context(), row.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, contract.AssignmentDetail{Assignment: assignment, Description: description, Problems: problems, Progress: progressRows})
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
