package web

import (
	"net/http"
	"sort"
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

func (api *API) assignmentProgress(c echo.Context, id uint, problems []ProblemDTO) ([]AssignmentProgressDTO, error) {
	userIDs, err := api.assignmentProgressUserIDs(id)
	if err != nil {
		return nil, err
	}
	if len(userIDs) == 0 {
		return []AssignmentProgressDTO{}, nil
	}

	var users []models.User
	if err := api.db.Where("id IN ?", userIDs).Order("name asc").Find(&users).Error; err != nil {
		return nil, err
	}

	problemIDs := make([]uint, 0, len(problems))
	for _, problem := range problems {
		problemIDs = append(problemIDs, problem.ID)
	}

	type state struct {
		item      AssignmentProgressDTO
		byProblem map[uint]*AssignmentProblemProgressDTO
	}
	states := map[uint]*state{}
	for _, user := range users {
		item := AssignmentProgressDTO{
			User:     user.Name,
			Problems: make([]AssignmentProblemProgressDTO, 0, len(problems)),
		}
		byProblem := map[uint]*AssignmentProblemProgressDTO{}
		for _, problem := range problems {
			item.Problems = append(item.Problems, AssignmentProblemProgressDTO{ProblemID: problem.ID, Status: "none"})
			byProblem[problem.ID] = &item.Problems[len(item.Problems)-1]
		}
		states[user.ID] = &state{item: item, byProblem: byProblem}
	}

	if len(problemIDs) > 0 {
		var submissions []struct {
			UserID    uint
			ProblemID uint
			ContestID *uint
			Status    string
			Score     int
			CreatedAt time.Time
		}
		if err := api.db.Model(&models.Submission{}).
			Select("user_id, problem_id, contest_id, status, score, created_at").
			Where("assignment_id = ? AND user_id IN ? AND problem_id IN ?", id, userIDs, problemIDs).
			Find(&submissions).Error; err != nil {
			return nil, err
		}
		contestIDs := make([]uint, 0, len(submissions))
		for _, submission := range submissions {
			if submission.ContestID != nil {
				contestIDs = append(contestIDs, *submission.ContestID)
			}
		}
		contests := map[uint]models.Contest{}
		if len(contestIDs) > 0 {
			var rows []models.Contest
			if err := api.db.Where("id IN ?", uniqueUint(contestIDs)).Find(&rows).Error; err != nil {
				return nil, err
			}
			for _, row := range rows {
				contests[row.ID] = row
			}
		}
		admin := api.isAdmin(c)
		viewerID := uint(0)
		if !admin {
			id, err := api.viewerID(c)
			if err != nil {
				return nil, err
			}
			viewerID = id
		}
		for _, submission := range submissions {
			got := states[submission.UserID]
			if got == nil {
				continue
			}
			got.item.Submit++
			problem := got.byProblem[submission.ProblemID]
			if problem == nil {
				continue
			}
			if assignmentProgressResultHidden(admin, viewerID, submission.UserID, submission.ContestID, submission.CreatedAt, contests) {
				if problem.Status != "ac" {
					problem.Status = "pending"
					problem.Score = nil
				}
				continue
			}
			if problem.Status == "pending" && submission.Status != "AC" {
				continue
			}
			if problem.Score == nil || submission.Score > *problem.Score {
				score := submission.Score
				problem.Score = &score
			}
			if submission.Status == "AC" {
				problem.Status = "ac"
				continue
			}
			if problem.Status != "ac" {
				problem.Status = "tried"
			}
		}
	}

	items := make([]AssignmentProgressDTO, 0, len(states))
	for _, got := range states {
		for _, problem := range got.item.Problems {
			if problem.Status == "ac" {
				got.item.AC++
			}
		}
		items = append(items, got.item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].AC != items[j].AC {
			return items[i].AC > items[j].AC
		}
		if items[i].Submit != items[j].Submit {
			return items[i].Submit < items[j].Submit
		}
		return items[i].User < items[j].User
	})
	return items, nil
}

func assignmentProgressResultHidden(admin bool, viewerID uint, userID uint, contestID *uint, createdAt time.Time, contests map[uint]models.Contest) bool {
	if admin || contestID == nil {
		return false
	}
	contest, ok := contests[*contestID]
	if !ok || contestEnded(contest) {
		return false
	}
	if contest.Kind == "OI" {
		return true
	}
	return viewerID != userID && contest.FreezeAt != nil && !createdAt.Before(*contest.FreezeAt)
}

func (api *API) assignmentProgressUserIDs(id uint) ([]uint, error) {
	userSet := map[uint]struct{}{}
	var direct []models.AssignmentUser
	if err := api.db.Where("assignment_id = ?", id).Find(&direct).Error; err != nil {
		return nil, err
	}
	for _, row := range direct {
		userSet[row.UserID] = struct{}{}
	}

	var grouped []struct {
		UserID uint
	}
	if err := api.db.Model(&models.GroupUser{}).
		Select("group_users.user_id").
		Joins("JOIN assignment_groups ON assignment_groups.group_id = group_users.group_id").
		Where("assignment_groups.assignment_id = ?", id).
		Find(&grouped).Error; err != nil {
		return nil, err
	}
	for _, row := range grouped {
		userSet[row.UserID] = struct{}{}
	}

	userIDs := make([]uint, 0, len(userSet))
	for id := range userSet {
		userIDs = append(userIDs, id)
	}
	sort.Slice(userIDs, func(i, j int) bool { return userIDs[i] < userIDs[j] })
	return userIDs, nil
}

func assignmentEnded(row models.Assignment) bool {
	return !time.Now().Before(row.EndAt)
}

func (api *API) assignmentDTO(c echo.Context, row models.Assignment, total int, done int) (AssignmentDTO, error) {
	members := assignmentMembersDTO{}
	admin := api.isAdmin(c)
	if admin {
		users, groups, err := api.assignmentMembers(row.ID)
		if err != nil {
			return AssignmentDTO{}, err
		}
		members = assignmentMembersDTO{Users: users, Groups: groups}
	}
	return assignmentDTOFromParts(row, total, done, members, admin), nil
}

type assignmentMembersDTO struct {
	Users  []uint
	Groups []uint
}

func (api *API) assignmentDTOs(c echo.Context, rows []models.Assignment, includeMembers bool) ([]AssignmentDTO, error) {
	if len(rows) == 0 {
		return []AssignmentDTO{}, nil
	}
	ids := assignmentIDs(rows)
	visible, err := api.assignmentVisibleMap(c, ids)
	if err != nil {
		return nil, err
	}
	totals, err := api.assignmentTotalMap(ids)
	if err != nil {
		return nil, err
	}
	done, err := api.assignmentDoneMap(c, ids)
	if err != nil {
		return nil, err
	}
	admin := api.isAdmin(c)
	includeMemberFields := includeMembers && admin
	members := map[uint]assignmentMembersDTO{}
	if includeMemberFields {
		members, err = api.assignmentMembersMap(ids)
		if err != nil {
			return nil, err
		}
	}
	items := make([]AssignmentDTO, 0, len(rows))
	for _, row := range rows {
		if !visible[row.ID] {
			continue
		}
		total := totals[row.ID]
		if total == 0 {
			continue
		}
		items = append(items, assignmentDTOFromParts(row, total, done[row.ID], members[row.ID], includeMemberFields))
	}
	return items, nil
}

func assignmentDTOFromParts(row models.Assignment, total int, done int, members assignmentMembersDTO, includeMembers bool) AssignmentDTO {
	dto := AssignmentDTO{
		ID:     row.ID,
		Title:  row.Title,
		EndAt:  row.EndAt,
		Status: assignmentStatus(row),
		Total:  total,
		Done:   done,
	}
	if includeMembers {
		dto.Users = cleanUintList(members.Users)
		if dto.Users == nil {
			dto.Users = []uint{}
		}
		dto.Groups = cleanUintList(members.Groups)
		if dto.Groups == nil {
			dto.Groups = []uint{}
		}
	}
	return dto
}

func assignmentStatus(row models.Assignment) string {
	if row.EndAt.Before(time.Now()) {
		return "ended"
	}
	return "running"
}

func assignmentIDs(rows []models.Assignment) []uint {
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func (api *API) assignmentVisibleMap(c echo.Context, ids []uint) (map[uint]bool, error) {
	ids = uniqueUint(ids)
	visible := map[uint]bool{}
	if len(ids) == 0 {
		return visible, nil
	}
	if api.isAdmin(c) {
		for _, id := range ids {
			visible[id] = true
		}
		return visible, nil
	}
	if api.role(c) == "guest" {
		return visible, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return nil, err
	}
	var direct []models.AssignmentUser
	if err := api.db.Where("assignment_id IN ? AND user_id = ?", ids, user.ID).Find(&direct).Error; err != nil {
		return nil, err
	}
	for _, row := range direct {
		visible[row.AssignmentID] = true
	}
	var grouped []struct {
		AssignmentID uint
	}
	if err := api.db.Model(&models.AssignmentGroup{}).
		Select("assignment_groups.assignment_id").
		Joins("JOIN group_users ON group_users.group_id = assignment_groups.group_id").
		Where("assignment_groups.assignment_id IN ? AND group_users.user_id = ?", ids, user.ID).
		Find(&grouped).Error; err != nil {
		return nil, err
	}
	for _, row := range grouped {
		visible[row.AssignmentID] = true
	}
	return visible, nil
}

func (api *API) assignmentMembers(id uint) ([]uint, []uint, error) {
	var users []models.AssignmentUser
	if err := api.db.Where("assignment_id = ?", id).Order("user_id asc").Find(&users).Error; err != nil {
		return nil, nil, err
	}
	var groups []models.AssignmentGroup
	if err := api.db.Where("assignment_id = ?", id).Order("group_id asc").Find(&groups).Error; err != nil {
		return nil, nil, err
	}
	userIDs := make([]uint, 0, len(users))
	for _, row := range users {
		userIDs = append(userIDs, row.UserID)
	}
	groupIDs := make([]uint, 0, len(groups))
	for _, row := range groups {
		groupIDs = append(groupIDs, row.GroupID)
	}
	return userIDs, groupIDs, nil
}

func (api *API) assignmentMembersMap(ids []uint) (map[uint]assignmentMembersDTO, error) {
	ids = uniqueUint(ids)
	members := map[uint]assignmentMembersDTO{}
	if len(ids) == 0 {
		return members, nil
	}
	var users []models.AssignmentUser
	if err := api.db.Where("assignment_id IN ?", ids).Order("user_id asc").Find(&users).Error; err != nil {
		return nil, err
	}
	for _, row := range users {
		item := members[row.AssignmentID]
		item.Users = append(item.Users, row.UserID)
		members[row.AssignmentID] = item
	}
	var groups []models.AssignmentGroup
	if err := api.db.Where("assignment_id IN ?", ids).Order("group_id asc").Find(&groups).Error; err != nil {
		return nil, err
	}
	for _, row := range groups {
		item := members[row.AssignmentID]
		item.Groups = append(item.Groups, row.GroupID)
		members[row.AssignmentID] = item
	}
	return members, nil
}

func (api *API) assignmentProblemCount(id uint) (int, error) {
	var count int64
	query := api.db.Model(&models.AssignmentProblem{}).Where("assignment_id = ?", id)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (api *API) assignmentTotalMap(ids []uint) (map[uint]int, error) {
	ids = uniqueUint(ids)
	totals := map[uint]int{}
	if len(ids) == 0 {
		return totals, nil
	}
	var rows []struct {
		AssignmentID uint
		Count        int64
	}
	if err := api.db.Model(&models.AssignmentProblem{}).
		Select("assignment_id, count(*) as count").
		Where("assignment_id IN ?", ids).
		Group("assignment_id").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		totals[row.AssignmentID] = int(row.Count)
	}
	return totals, nil
}

func (api *API) assignmentDoneCount(c echo.Context, id uint) (int, error) {
	if api.role(c) == "guest" {
		return 0, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return 0, err
	}
	var rows []struct {
		ProblemID uint
	}
	query := api.db.Model(&models.Submission{}).
		Select("DISTINCT submissions.problem_id").
		Joins("JOIN assignment_problems ON assignment_problems.problem_id = submissions.problem_id").
		Where("assignment_problems.assignment_id = ? AND submissions.assignment_id = ? AND submissions.user_id = ? AND submissions.status = ?", id, id, user.ID, "AC")
	query, err = api.filterHiddenResultAC(c, query)
	if err != nil {
		return 0, err
	}
	if err := query.Find(&rows).Error; err != nil {
		return 0, err
	}
	return len(rows), nil
}

func (api *API) assignmentDoneMap(c echo.Context, ids []uint) (map[uint]int, error) {
	ids = uniqueUint(ids)
	done := map[uint]int{}
	if len(ids) == 0 || api.role(c) == "guest" {
		return done, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		AssignmentID uint
		Count        int64
	}
	query := api.db.Model(&models.Submission{}).
		Select("submissions.assignment_id, count(DISTINCT submissions.problem_id) as count").
		Joins("JOIN assignment_problems ON assignment_problems.assignment_id = submissions.assignment_id AND assignment_problems.problem_id = submissions.problem_id").
		Where("submissions.assignment_id IN ? AND submissions.user_id = ? AND submissions.status = ?", ids, user.ID, "AC").
		Group("submissions.assignment_id")
	query, err = api.filterHiddenResultAC(c, query)
	if err != nil {
		return nil, err
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		done[row.AssignmentID] = int(row.Count)
	}
	return done, nil
}

func (api *API) assignmentProblems(c echo.Context, assignment models.Assignment, links []models.AssignmentProblem) ([]ProblemDTO, error) {
	if len(links) == 0 {
		return []ProblemDTO{}, nil
	}
	ids := make([]uint, 0, len(links))
	for _, link := range links {
		ids = append(ids, link.ProblemID)
	}
	query := api.db.Model(&models.Problem{}).Where("id IN ?", uniqueUint(ids))
	var rows []models.Problem
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	byID := problemRowsByID(rows)
	items := make([]ProblemDTO, 0, len(links))
	for _, link := range links {
		problem, ok := byID[link.ProblemID]
		if !ok {
			continue
		}
		if !api.assignmentShouldIncludeHiddenProblem(c, assignment) && !api.problemVisibleInList(problem) {
			continue
		}
		item := problemDTO(problem)
		item.Sort = link.Sort
		items = append(items, item)
	}
	return items, nil
}

func (api *API) assignmentShouldIncludeHiddenProblem(c echo.Context, assignment models.Assignment) bool {
	return api.isAdmin(c) || !assignment.EndAt.Before(time.Now())
}
