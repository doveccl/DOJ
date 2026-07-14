package public

import (
	"net/http"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

const problemListVisibilitySQL = `problems.visible = ? AND NOT EXISTS (
	SELECT 1 FROM contest_problems
	JOIN contests ON contests.id = contest_problems.contest_id
	WHERE contest_problems.problem_id = problems.id
		AND contests.deleted_at IS NULL
		AND contests.end_at > ?
)`

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
	if api.problemInEndedContest(problem.ID) {
		return true, nil
	}
	if api.role(c) == "guest" {
		return false, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return false, err
	}
	return api.problemInAssignmentForUser(problem.ID, user.ID, false)
}

func (api *API) problemInAssignmentForUser(problemID uint, userID uint, endedOnly bool) (bool, error) {
	var rows []models.Assignment
	query := api.db.
		Joins("JOIN assignment_problems ON assignment_problems.assignment_id = assignments.id").
		Where("assignment_problems.problem_id = ?", problemID)
	if endedOnly {
		query = query.Where("assignments.end_at <= ?", time.Now())
	}
	if err := query.Find(&rows).Error; err != nil {
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
	return query.Where(problemListVisibilitySQL, true, time.Now())
}

func (api *API) problemInUnfinishedContest(problemID uint) bool {
	var count int64
	err := api.db.Model(&models.ContestProblem{}).
		Joins("JOIN contests ON contests.id = contest_problems.contest_id").
		Where("contest_problems.problem_id = ? AND contests.deleted_at IS NULL AND contests.end_at > ?", problemID, time.Now()).
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
		Where("contest_problems.problem_id = ? AND contests.deleted_at IS NULL AND contests.start_at <= ? AND contests.end_at > ?", problemID, now, now).
		Count(&count).Error
	return err == nil && count > 0
}

func (api *API) problemInEndedContest(problemID uint) bool {
	if api.problemInUnfinishedContest(problemID) {
		return false
	}
	var count int64
	err := api.db.Model(&models.ContestProblem{}).
		Joins("JOIN contests ON contests.id = contest_problems.contest_id").
		Where("contest_problems.problem_id = ? AND contests.deleted_at IS NULL AND contests.end_at <= ?", problemID, time.Now()).
		Count(&count).Error
	return err == nil && count > 0
}

type submissionView struct {
	Result bool
	Code   bool
}

func (api *API) submissionView(c echo.Context, row models.Submission) (submissionView, error) {
	views, err := api.submissionViews(c, []models.Submission{row})
	if err != nil {
		return submissionView{}, err
	}
	return views[0], nil
}

func (api *API) submissionViews(c echo.Context, rows []models.Submission) ([]submissionView, error) {
	views := make([]submissionView, len(rows))
	if len(rows) == 0 {
		return views, nil
	}
	if api.isAdmin(c) {
		for index := range views {
			views[index] = submissionView{Result: true, Code: true}
		}
		return views, nil
	}
	viewerID, err := api.viewerID(c)
	if err != nil {
		return nil, err
	}
	problemIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		problemIDs = append(problemIDs, row.ProblemID)
	}
	unfinishedProblems, err := api.unfinishedContestProblemSet(uniqueUint(problemIDs))
	if err != nil {
		return nil, err
	}
	contestIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		if row.ContestID != nil {
			contestIDs = append(contestIDs, *row.ContestID)
		}
	}
	contests := map[uint]models.Contest{}
	if ids := uniqueUint(contestIDs); len(ids) > 0 {
		var found []models.Contest
		if err := api.db.Where("id IN ?", ids).Find(&found).Error; err != nil {
			return nil, err
		}
		for _, row := range found {
			contests[row.ID] = row
		}
	}
	now := time.Now()
	for index, row := range rows {
		owner := viewerID != 0 && viewerID == row.UserID
		view := submissionView{Result: true, Code: owner}
		sourceLocked := unfinishedProblems[row.ProblemID]
		if row.ContestID != nil {
			if contest, ok := contests[*row.ContestID]; ok && now.Before(contest.EndAt) {
				sourceLocked = true
				if contest.Kind == "OI" || (!owner && contest.FreezeAt != nil && !row.CreatedAt.Before(*contest.FreezeAt)) {
					view.Result = false
				}
			}
		}
		if !sourceLocked {
			view.Code = owner || row.Public
		}
		views[index] = view
	}
	return views, nil
}

const visibleSubmissionResultSQL = `NOT EXISTS (
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
)`

func (api *API) filterVisibleResults(c echo.Context, query *gorm.DB) (*gorm.DB, error) {
	if api.isAdmin(c) {
		return query, nil
	}
	viewerID, err := api.viewerID(c)
	if err != nil {
		return nil, err
	}
	return query.Where(visibleSubmissionResultSQL, submissionResultVisibilityArgs(viewerID)...), nil
}

func (api *API) filterHiddenResults(c echo.Context, query *gorm.DB) (*gorm.DB, error) {
	if api.isAdmin(c) {
		return query.Where("1 = 0"), nil
	}
	viewerID, err := api.viewerID(c)
	if err != nil {
		return nil, err
	}
	return query.Where("NOT ("+visibleSubmissionResultSQL+")", submissionResultVisibilityArgs(viewerID)...), nil
}

func submissionResultVisibilityArgs(viewerID uint) []any {
	return []any{time.Now(), "OI", "ICPC", viewerID, viewerID}
}
