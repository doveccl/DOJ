package web

import (
	"net/http"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (api *API) requireProblemVisible(c echo.Context, id uint) error {
	if api.isAdmin(c) {
		return nil
	}

	var problem models.Problem
	if err := api.db.First(&problem, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	visible, err := api.problemVisibleInDetail(c, problem)
	if err != nil {
		return err
	}
	if !visible {
		return echo.NewHTTPError(http.StatusNotFound, "problem not found")
	}
	return nil
}

func (api *API) problemVisibleInDetail(c echo.Context, problem models.Problem) (bool, error) {
	if api.problemVisibleInList(problem) {
		return true, nil
	}
	if api.problemInRunningContest(problem.ID) {
		return true, nil
	}
	if api.role(c) == "guest" {
		return false, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return false, err
	}
	return api.problemInActiveAssignmentForUser(problem.ID, user.ID)
}

func (api *API) problemInActiveAssignmentForUser(problemID uint, userID uint) (bool, error) {
	var rows []models.Assignment
	if err := api.db.
		Joins("JOIN assignment_problems ON assignment_problems.assignment_id = assignments.id").
		Where("assignment_problems.problem_id = ? AND assignments.end_at >= ?", problemID, time.Now()).
		Find(&rows).Error; err != nil {
		return false, err
	}
	for _, row := range rows {
		ok, err := api.userAssignedTo(row.ID, userID)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func (api *API) problemVisibleInList(problem models.Problem) bool {
	if !problem.Visible {
		return false
	}
	return !api.problemInUnfinishedContest(problem.ID)
}

func (api *API) applyProblemListVisibility(query *gorm.DB) *gorm.DB {
	now := time.Now()
	return query.Where(
		`problems.visible = ? AND NOT EXISTS (
			SELECT 1 FROM contest_problems
			JOIN contests ON contests.id = contest_problems.contest_id
			WHERE contest_problems.problem_id = problems.id AND contests.end_at > ?
		)`,
		true,
		now,
	)
}

func (api *API) problemInUnfinishedContest(problemID uint) bool {
	var count int64
	err := api.db.Model(&models.ContestProblem{}).
		Joins("JOIN contests ON contests.id = contest_problems.contest_id").
		Where("contest_problems.problem_id = ? AND contests.end_at > ?", problemID, time.Now()).
		Count(&count).Error
	return err == nil && count > 0
}

func (api *API) problemVisibleForStats(c echo.Context, id uint) (bool, error) {
	var problem models.Problem
	if err := api.db.First(&problem, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	return api.problemVisibleInDetail(c, problem)
}

func (api *API) problemInRunningContest(problemID uint) bool {
	var count int64
	now := time.Now()
	err := api.db.Model(&models.ContestProblem{}).
		Joins("JOIN contests ON contests.id = contest_problems.contest_id").
		Where("contest_problems.problem_id = ? AND contests.start_at <= ? AND contests.end_at > ?", problemID, now, now).
		Count(&count).Error
	return err == nil && count > 0
}

type submissionView struct {
	Result bool
	Code   bool
}

func (api *API) submissionView(c echo.Context, row models.Submission) (submissionView, error) {
	if api.isAdmin(c) {
		return submissionView{Result: true, Code: true}, nil
	}
	view := submissionView{Result: true}
	userID, err := api.viewerID(c)
	if err != nil {
		return view, err
	}
	owner := userID != 0 && userID == row.UserID
	view.Code = owner
	active, contest, err := api.submissionActiveContest(row)
	if err != nil {
		return view, err
	}
	if active {
		if contest.Kind == "OI" {
			view.Result = false
		} else if !owner && contest.FreezeAt != nil && !row.CreatedAt.Before(*contest.FreezeAt) {
			view.Result = false
		}
		return view, nil
	}
	active, err = api.submissionActiveAssignment(row)
	if err != nil {
		return view, err
	}
	if active {
		return view, nil
	}
	view.Code = owner || row.Public
	return view, nil
}

func (api *API) submissionActiveContest(row models.Submission) (bool, models.Contest, error) {
	var contest models.Contest
	if row.ContestID == nil {
		return false, contest, nil
	}
	if err := api.db.First(&contest, *row.ContestID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, contest, nil
		}
		return false, contest, err
	}
	return !contestEnded(contest), contest, nil
}

func (api *API) submissionActiveAssignment(row models.Submission) (bool, error) {
	if row.AssignmentID == nil {
		return false, nil
	}
	var assignment models.Assignment
	if err := api.db.First(&assignment, *row.AssignmentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	return !assignmentEnded(assignment), nil
}

func (api *API) filterHiddenResultAC(c echo.Context, query *gorm.DB) (*gorm.DB, error) {
	if api.isAdmin(c) {
		return query, nil
	}
	viewerID, err := api.viewerID(c)
	if err != nil {
		return nil, err
	}
	return query.Where(
		`NOT EXISTS (
			SELECT 1 FROM contests
			WHERE contests.id = submissions.contest_id
				AND contests.deleted_at IS NULL
				AND contests.end_at > ?
				AND (
					contests.kind = ?
					OR (
						contests.kind = ?
						AND contests.freeze_at IS NOT NULL
						AND submissions.created_at >= contests.freeze_at
						AND (? = 0 OR submissions.user_id <> ?)
					)
				)
		)`,
		time.Now(),
		"OI",
		"ICPC",
		viewerID,
		viewerID,
	), nil
}
