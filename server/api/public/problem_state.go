package public

import (
	"net/http"
	"time"

	contract "github.com/doveccl/doj/contract/web"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (api *API) problemState(c echo.Context) error {
	ids, err := parseIDs(c.QueryParam("ids"), "invalid problem ids")
	if err != nil {
		return err
	}
	var assignmentID *uint
	if raw := c.QueryParam("assignment"); raw != "" {
		id, err := parseQueryID(raw, "invalid assignment id")
		if err != nil {
			return err
		}
		assignmentID = &id
	}
	var contestID *uint
	if raw := c.QueryParam("contest"); raw != "" {
		id, err := parseQueryID(raw, "invalid contest id")
		if err != nil {
			return err
		}
		contestID = &id
	}
	if assignmentID != nil && contestID != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "assignment and contest cannot both be set")
	}
	items, err := api.problemStateItems(c, ids, assignmentID, contestID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, items)
}

func (api *API) problemStateItems(c echo.Context, ids []uint, assignmentID *uint, contestID *uint) ([]contract.ProblemState, error) {
	ids = uniqueUint(ids)
	if len(ids) == 0 {
		return []contract.ProblemState{}, nil
	}
	if assignmentID != nil {
		var assignment models.Assignment
		if err := api.db.First(&assignment, *assignmentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, echo.NewHTTPError(http.StatusNotFound, "assignment not found")
			}
			return nil, err
		}
		visible, err := api.assignmentVisible(c, assignment.ID)
		if err != nil {
			return nil, err
		}
		if !visible {
			return nil, echo.NewHTTPError(http.StatusNotFound, "assignment not found")
		}
	}
	var contest *models.Contest
	if contestID != nil {
		var row models.Contest
		if err := api.db.First(&row, *contestID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, echo.NewHTTPError(http.StatusNotFound, "contest not found")
			}
			return nil, err
		}
		contest = &row
	}
	visible, err := api.problemStateVisibleIDs(c, ids, assignmentID, contest)
	if err != nil {
		return nil, err
	}
	items := defaultProblemStateItems(ids)
	if err := api.fillProblemSummaryState(c, items, visible, assignmentID, contestID); err != nil {
		return nil, err
	}
	return api.fillProblemUserState(c, items, assignmentID, contest)
}

func (api *API) problemStateVisibleIDs(c echo.Context, ids []uint, assignmentID *uint, contest *models.Contest) (map[uint]bool, error) {
	visible := map[uint]bool{}
	query := api.db.Model(&models.Problem{}).Select("problems.id").Where("problems.id IN ?", ids)
	if assignmentID != nil {
		query = query.Joins("JOIN assignment_problems ON assignment_problems.problem_id = problems.id").
			Where("assignment_problems.assignment_id = ?", *assignmentID)
	} else if contest != nil {
		if !api.isAdmin(c) && !contestRunning(*contest) && !contestEnded(*contest) {
			return visible, nil
		}
		query = query.Joins("JOIN contest_problems ON contest_problems.problem_id = problems.id").
			Where("contest_problems.contest_id = ?", contest.ID)
		if !api.isAdmin(c) && contestEnded(*contest) {
			query = query.Where(`NOT EXISTS (
				SELECT 1 FROM contest_problems future_links
				JOIN contests future_contests ON future_contests.id = future_links.contest_id
				WHERE future_links.problem_id = problems.id
					AND future_contests.deleted_at IS NULL
					AND future_contests.end_at > ?
			)`, time.Now())
		}
	} else if !api.isAdmin(c) {
		now := time.Now()
		query = query.Where(`NOT EXISTS (
			SELECT 1 FROM contest_problems unfinished_links
			JOIN contests unfinished_contests ON unfinished_contests.id = unfinished_links.contest_id
			WHERE unfinished_links.problem_id = problems.id
				AND unfinished_contests.deleted_at IS NULL
				AND unfinished_contests.end_at > ?
		)`, now)
		baseVisibility := `problems.visible = ? OR EXISTS (
			SELECT 1 FROM contest_problems ended_links
			JOIN contests ended_contests ON ended_contests.id = ended_links.contest_id
			WHERE ended_links.problem_id = problems.id
				AND ended_contests.deleted_at IS NULL
				AND ended_contests.end_at <= ?
		)`
		if api.role(c) == "guest" {
			query = query.Where("("+baseVisibility+")", true, now)
		} else {
			viewerID, err := api.viewerID(c)
			if err != nil {
				return nil, err
			}
			query = query.Where("("+baseVisibility+` OR EXISTS (
				SELECT 1 FROM assignment_problems assigned_links
				JOIN assignments assigned_assignments ON assigned_assignments.id = assigned_links.assignment_id
				WHERE assigned_links.problem_id = problems.id
					AND assigned_assignments.deleted_at IS NULL
					AND (
						EXISTS (
							SELECT 1 FROM assignment_users
							WHERE assignment_users.assignment_id = assigned_assignments.id
								AND assignment_users.user_id = ?
						)
						OR EXISTS (
							SELECT 1 FROM assignment_groups
							JOIN group_users ON group_users.group_id = assignment_groups.group_id
							WHERE assignment_groups.assignment_id = assigned_assignments.id
								AND group_users.user_id = ?
						)
					)
			))`, true, now, viewerID, viewerID)
		}
	}
	var rows []struct{ ID uint }
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		visible[row.ID] = true
	}
	return visible, nil
}

func (api *API) unfinishedContestProblemSet(ids []uint) (map[uint]bool, error) {
	unfinished := map[uint]bool{}
	var rows []struct{ ProblemID uint }
	if err := api.db.Model(&models.ContestProblem{}).
		Select("DISTINCT contest_problems.problem_id").
		Joins("JOIN contests ON contests.id = contest_problems.contest_id").
		Where("contest_problems.problem_id IN ? AND contests.deleted_at IS NULL AND contests.end_at > ?", ids, time.Now()).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		unfinished[row.ProblemID] = true
	}
	return unfinished, nil
}

func (api *API) fillProblemSummaryState(c echo.Context, items []contract.ProblemState, visible map[uint]bool, assignmentID *uint, contestID *uint) error {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		if visible[item.ProblemID] {
			ids = append(ids, item.ProblemID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	submitByProblem, acByProblem, err := api.problemSubmissionStats(c, ids, assignmentID, contestID)
	if err != nil {
		return err
	}
	discussions, err := api.problemDiscussionCounts(c.Request().Context())
	if err != nil {
		return err
	}
	unfinished := map[uint]bool{}
	if !api.isAdmin(c) {
		unfinished, err = api.unfinishedContestProblemSet(ids)
		if err != nil {
			return err
		}
	}
	for index := range items {
		id := items[index].ProblemID
		if !visible[id] {
			continue
		}
		items[index].Submit = submitByProblem[id]
		items[index].AC = acByProblem[id]
		if !unfinished[id] {
			count := discussions[id]
			items[index].Discussions = &count
		}
	}
	return nil
}

func (api *API) problemSubmissionStats(c echo.Context, ids []uint, assignmentID *uint, contestID *uint) (map[uint]int, map[uint]int, error) {
	var submits []struct {
		ProblemID uint
		Count     int64
	}
	submitQuery, err := api.applySubmissionAccess(c, api.problemStateSubmissionQuery(ids, assignmentID, contestID))
	if err != nil {
		return nil, nil, err
	}
	if err := submitQuery.
		Select("submissions.problem_id, count(*) AS count").
		Group("problem_id").
		Find(&submits).Error; err != nil {
		return nil, nil, err
	}
	submitByProblem := map[uint]int{}
	for _, item := range submits {
		submitByProblem[item.ProblemID] = int(item.Count)
	}
	var acs []struct {
		ProblemID uint
		Count     int64
	}
	acQuery, err := api.applySubmissionStatsVisibility(c, api.problemStateSubmissionQuery(ids, assignmentID, contestID).Where("submissions.status = ?", "AC"))
	if err != nil {
		return nil, nil, err
	}
	if err := acQuery.
		Select("submissions.problem_id, count(*) AS count").
		Group("problem_id").
		Find(&acs).Error; err != nil {
		return nil, nil, err
	}
	acByProblem := map[uint]int{}
	for _, item := range acs {
		acByProblem[item.ProblemID] = int(item.Count)
	}
	return submitByProblem, acByProblem, nil
}

func (api *API) problemStateSubmissionQuery(ids []uint, assignmentID *uint, contestID *uint) *gorm.DB {
	query := api.db.Model(&models.Submission{}).Where("submissions.problem_id IN ?", ids)
	if assignmentID != nil {
		return query.Where("submissions.assignment_id = ? AND submissions.contest_id IS NULL", *assignmentID)
	}
	if contestID != nil {
		return query.Where("submissions.contest_id = ? AND submissions.assignment_id IS NULL", *contestID)
	}
	return query.Where("submissions.assignment_id IS NULL AND submissions.contest_id IS NULL")
}

func (api *API) fillProblemUserState(c echo.Context, items []contract.ProblemState, assignmentID *uint, contest *models.Contest) ([]contract.ProblemState, error) {
	if contest != nil {
		return api.fillProblemUserStateInContest(c, items, *contest)
	}
	if api.role(c) == "guest" {
		return items, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ProblemID    uint
		ID           uint
		AssignmentID *uint
		ContestID    *uint
		Status       string
		Score        int
		CreatedAt    time.Time
	}
	ids := problemStateIDs(items)
	query := api.db.Model(&models.Submission{}).
		Select("problem_id, id, assignment_id, contest_id, status, score, created_at").
		Where("user_id = ? AND problem_id IN ?", user.ID, ids).
		Order("created_at desc")
	if assignmentID != nil {
		query = query.Where("assignment_id = ? AND contest_id IS NULL", *assignmentID)
	} else {
		query = query.Where("assignment_id IS NULL AND contest_id IS NULL")
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	viewRows := make([]models.Submission, len(rows))
	for index, row := range rows {
		viewRows[index] = models.Submission{ID: row.ID, UserID: user.ID, ProblemID: row.ProblemID, AssignmentID: row.AssignmentID, ContestID: row.ContestID, Status: row.Status, Score: row.Score, CreatedAt: row.CreatedAt}
	}
	views, err := api.submissionViews(c, viewRows)
	if err != nil {
		return nil, err
	}
	status := map[uint]string{}
	submission := map[uint]contract.ProblemRecord{}
	for index, row := range rows {
		resultVisible := row.ID == 0 || views[index].Result
		submissionSet := false
		if _, ok := submission[row.ProblemID]; !ok {
			submission[row.ProblemID] = contract.ProblemRecord{ID: row.ID, Status: row.Status, Score: row.Score, CreatedAt: row.CreatedAt}
			submissionSet = true
		}
		if !resultVisible {
			if submissionSet {
				submission[row.ProblemID] = pendingRecord(submission[row.ProblemID])
			}
			if status[row.ProblemID] == "" {
				status[row.ProblemID] = "pending"
			}
			continue
		}
		if submissionLive(row.Status) {
			if status[row.ProblemID] == "" {
				status[row.ProblemID] = "pending"
			}
			continue
		}
		if row.Status == "AC" {
			status[row.ProblemID] = "ac"
			continue
		}
		if status[row.ProblemID] == "" {
			status[row.ProblemID] = "tried"
		}
	}
	for index := range items {
		items[index].Status = status[items[index].ProblemID]
		if items[index].Status == "" {
			items[index].Status = "none"
		}
		if item, ok := submission[items[index].ProblemID]; ok {
			items[index].Submission = &item
		}
	}
	return items, nil
}

func (api *API) fillProblemUserStateInContest(c echo.Context, items []contract.ProblemState, contest models.Contest) ([]contract.ProblemState, error) {
	if api.role(c) == "guest" {
		return items, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return nil, err
	}
	var rows []struct {
		ProblemID uint
		ID        uint
		ContestID uint
		Status    string
		Score     int
		CreatedAt time.Time
	}
	ids := problemStateIDs(items)
	if err := api.db.Model(&models.Submission{}).
		Select("problem_id, id, contest_id, status, score, created_at").
		Where("user_id = ? AND contest_id = ? AND assignment_id IS NULL AND problem_id IN ?", user.ID, contest.ID, ids).
		Order("created_at desc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	viewRows := make([]models.Submission, len(rows))
	for index, row := range rows {
		contestID := row.ContestID
		viewRows[index] = models.Submission{ID: row.ID, UserID: user.ID, ProblemID: row.ProblemID, ContestID: &contestID, Status: row.Status, Score: row.Score, CreatedAt: row.CreatedAt}
	}
	views, err := api.submissionViews(c, viewRows)
	if err != nil {
		return nil, err
	}
	status := map[uint]string{}
	submission := map[uint]contract.ProblemRecord{}
	oiBest := map[uint]int{}
	oiDone := map[uint]bool{}
	for index, row := range rows {
		resultVisible := row.ID == 0 || views[index].Result
		submissionSet := false
		if _, ok := submission[row.ProblemID]; !ok {
			submission[row.ProblemID] = contract.ProblemRecord{ID: row.ID, Status: row.Status, Score: row.Score, CreatedAt: row.CreatedAt}
			submissionSet = true
		}
		if !resultVisible {
			if submissionSet {
				submission[row.ProblemID] = pendingRecord(submission[row.ProblemID])
			}
			if status[row.ProblemID] == "" {
				status[row.ProblemID] = "pending"
			}
			continue
		}
		if submissionLive(row.Status) {
			if status[row.ProblemID] == "" {
				status[row.ProblemID] = "pending"
			}
			continue
		}
		if contest.Kind == "OI" {
			if !oiDone[row.ProblemID] || row.Score > oiBest[row.ProblemID] {
				oiBest[row.ProblemID] = row.Score
				submission[row.ProblemID] = contract.ProblemRecord{ID: row.ID, Status: row.Status, Score: row.Score, CreatedAt: row.CreatedAt}
			}
			oiDone[row.ProblemID] = true
			if oiBest[row.ProblemID] >= 100 {
				status[row.ProblemID] = "ac"
			} else {
				status[row.ProblemID] = "tried"
			}
			continue
		}
		if status[row.ProblemID] == "ac" {
			continue
		}
		if row.Status == "AC" {
			status[row.ProblemID] = "ac"
			submission[row.ProblemID] = contract.ProblemRecord{ID: row.ID, Status: row.Status, Score: row.Score, CreatedAt: row.CreatedAt}
			continue
		}
		if status[row.ProblemID] == "" {
			status[row.ProblemID] = "tried"
		}
	}
	for index := range items {
		items[index].Status = status[items[index].ProblemID]
		if items[index].Status == "" {
			items[index].Status = "none"
		}
		if item, ok := submission[items[index].ProblemID]; ok {
			items[index].Submission = &item
		}
	}
	return items, nil
}

func defaultProblemStateItems(ids []uint) []contract.ProblemState {
	items := make([]contract.ProblemState, 0, len(ids))
	for _, id := range ids {
		items = append(items, contract.ProblemState{ProblemID: id, Status: "none"})
	}
	return items
}

func problemStateIDs(items []contract.ProblemState) []uint {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ProblemID)
	}
	return ids
}
