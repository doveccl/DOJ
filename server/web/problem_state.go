package web

import (
	"net/http"
	"time"

	"github.com/doveccl/doj/server/web/contract"

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
	items := defaultProblemStateItems(ids)
	if err := api.fillProblemSummaryState(c, items); err != nil {
		return nil, err
	}
	return api.fillProblemUserState(c, items, assignmentID, contestID)
}

func (api *API) fillProblemSummaryState(c echo.Context, items []contract.ProblemState) error {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ProblemID)
	}
	submitByProblem, acByProblem, err := api.problemSubmissionStats(ids)
	if err != nil {
		return err
	}
	discussions, err := api.problemDiscussionCounts(c.Request().Context())
	if err != nil {
		return err
	}
	for index := range items {
		id := items[index].ProblemID
		if !api.isAdmin(c) {
			visible, err := api.problemVisibleForStats(c, id)
			if err != nil {
				return err
			}
			if !visible || api.problemInUnfinishedContest(id) {
				continue
			}
		}
		items[index].Submit = submitByProblem[id]
		items[index].AC = acByProblem[id]
		count := discussions[id]
		items[index].Discussions = &count
	}
	return nil
}

func (api *API) problemSubmissionStats(ids []uint) (map[uint]int, map[uint]int, error) {
	var submits []struct {
		ProblemID uint
		Count     int64
	}
	if err := api.db.Model(&models.Submission{}).
		Select("problem_id, count(*) AS count").
		Where("problem_id IN ?", ids).
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
	if err := api.db.Model(&models.Submission{}).
		Select("problem_id, count(DISTINCT user_id) AS count").
		Where("problem_id IN ? AND status = ?", ids, "AC").
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

func (api *API) fillProblemUserState(c echo.Context, items []contract.ProblemState, assignmentID *uint, contestID *uint) ([]contract.ProblemState, error) {
	if contestID != nil {
		var contest models.Contest
		if err := api.db.First(&contest, *contestID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, echo.NewHTTPError(http.StatusNotFound, "contest not found")
			}
			return nil, err
		}
		return api.fillProblemUserStateInContest(c, items, contest)
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
		query = query.Where("assignment_id = ?", *assignmentID)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	status := map[uint]string{}
	submission := map[uint]contract.ProblemRecord{}
	for _, row := range rows {
		resultVisible := true
		if row.ID != 0 {
			view, err := api.submissionView(c, models.Submission{ID: row.ID, UserID: user.ID, ProblemID: row.ProblemID, AssignmentID: row.AssignmentID, ContestID: row.ContestID, Status: row.Status, Score: row.Score, CreatedAt: row.CreatedAt})
			if err != nil {
				return nil, err
			}
			resultVisible = view.Result
		}
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
		if item, ok := submission[items[index].ProblemID]; ok && assignmentID == nil {
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
		Where("user_id = ? AND contest_id = ? AND problem_id IN ?", user.ID, contest.ID, ids).
		Order("created_at desc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	status := map[uint]string{}
	submission := map[uint]contract.ProblemRecord{}
	for _, row := range rows {
		resultVisible := true
		if row.ID != 0 {
			view, err := api.submissionView(c, models.Submission{ID: row.ID, UserID: user.ID, ProblemID: row.ProblemID, ContestID: &row.ContestID, Status: row.Status, Score: row.Score, CreatedAt: row.CreatedAt})
			if err != nil {
				return nil, err
			}
			resultVisible = view.Result
		}
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
		if contest.Kind == "OI" {
			if status[row.ProblemID] == "" {
				if row.Score >= 100 {
					status[row.ProblemID] = "ac"
				} else {
					status[row.ProblemID] = "tried"
				}
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
