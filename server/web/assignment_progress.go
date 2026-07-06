package web

import (
	"sort"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
)

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
