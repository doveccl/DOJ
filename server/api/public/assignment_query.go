package public

import (
	"time"

	contract "github.com/doveccl/doj/contract/web"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
)

func (api *API) assignmentView(c echo.Context, row models.Assignment, total int, done int) (contract.Assignment, error) {
	members := assignmentMembers{}
	admin := api.isAdmin(c)
	if admin {
		users, groups, err := api.assignmentMembers(row.ID)
		if err != nil {
			return contract.Assignment{}, err
		}
		members = assignmentMembers{Users: users, Groups: groups}
	}
	return assignmentViewFromParts(row, total, done, members, admin), nil
}

type assignmentMembers struct {
	Users  []uint
	Groups []uint
}

func (api *API) assignmentViews(c echo.Context, rows []models.Assignment, includeMembers bool) ([]contract.Assignment, error) {
	if len(rows) == 0 {
		return []contract.Assignment{}, nil
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
	members := map[uint]assignmentMembers{}
	if includeMemberFields {
		members, err = api.assignmentMembersMap(ids)
		if err != nil {
			return nil, err
		}
	}
	items := make([]contract.Assignment, 0, len(rows))
	for _, row := range rows {
		if !visible[row.ID] {
			continue
		}
		total := totals[row.ID]
		if total == 0 {
			continue
		}
		items = append(items, assignmentViewFromParts(row, total, done[row.ID], members[row.ID], includeMemberFields))
	}
	return items, nil
}

func assignmentViewFromParts(row models.Assignment, total int, done int, members assignmentMembers, includeMembers bool) contract.Assignment {
	item := contract.Assignment{
		ID:     row.ID,
		Title:  row.Title,
		EndAt:  row.EndAt,
		Status: assignmentStatus(row),
		Total:  total,
		Done:   done,
	}
	if includeMembers {
		item.Users = cleanUintList(members.Users)
		if item.Users == nil {
			item.Users = []uint{}
		}
		item.Groups = cleanUintList(members.Groups)
		if item.Groups == nil {
			item.Groups = []uint{}
		}
	}
	return item
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

func (api *API) assignmentMembersMap(ids []uint) (map[uint]assignmentMembers, error) {
	ids = uniqueUint(ids)
	members := map[uint]assignmentMembers{}
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

func (api *API) assignmentProblems(links []models.AssignmentProblem) ([]contract.Problem, error) {
	if len(links) == 0 {
		return []contract.Problem{}, nil
	}
	ids := make([]uint, 0, len(links))
	for _, link := range links {
		ids = append(ids, link.ProblemID)
	}
	query := problemListColumns(api.db.Model(&models.Problem{})).Where("id IN ?", uniqueUint(ids))
	var rows []models.Problem
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	byID := problemRowsByID(rows)
	items := make([]contract.Problem, 0, len(links))
	for _, link := range links {
		problem, ok := byID[link.ProblemID]
		if !ok {
			continue
		}
		item := problemView(problem)
		item.Sort = link.Sort
		items = append(items, item)
	}
	return items, nil
}
