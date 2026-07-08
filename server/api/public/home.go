package public

import (
	"net/http"
	"strings"
	"time"

	"github.com/doveccl/doj/contract/limits"
	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/settings"
	"github.com/labstack/echo/v4"
)

func (api *API) home(c echo.Context) error {
	problems, err := api.homeProblems(c)
	if err != nil {
		return err
	}
	heatmap, err := api.homeHeatmap(c)
	if err != nil {
		return err
	}
	assignments, err := api.homeAssignments(c)
	if err != nil {
		return err
	}
	contests, err := api.homeContests(c)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, contract.Home{
		Notice:      api.notice(),
		Heatmap:     heatmap,
		Problems:    problems,
		Assignments: assignments,
		Contests:    contests,
	})
}

func (api *API) homeProblems(c echo.Context) ([]contract.HomeProblem, error) {
	var rows []models.Problem
	query := api.db.Select("id", "title").Order("id desc").Limit(homeListLimit)
	if !api.isAdmin(c) {
		query = api.applyProblemListVisibility(query)
	}
	if api.role(c) != "guest" {
		user, err := api.currentUser(c)
		if err != nil {
			return nil, err
		}
		query = query.Where(`NOT EXISTS (
			SELECT 1 FROM submissions
			WHERE submissions.problem_id = problems.id
			AND submissions.user_id = ?
			AND submissions.status = ?
		)`, user.ID, "AC")
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]contract.HomeProblem, 0, len(rows))
	for _, row := range rows {
		items = append(items, contract.HomeProblem{
			ID:    row.ID,
			Title: row.Title,
		})
	}
	return items, nil
}

func (api *API) homeHeatmap(c echo.Context) ([]contract.HeatCell, error) {
	if api.role(c) == "guest" {
		return []contract.HeatCell{}, nil
	}

	user, err := api.currentUser(c)
	if err != nil {
		return nil, err
	}
	return api.userHeatmap(user.ID)
}

func (api *API) homeAssignments(c echo.Context) ([]contract.HomeAssignment, error) {
	if api.role(c) == "guest" {
		return []contract.HomeAssignment{}, nil
	}
	user, err := api.currentUser(c)
	if err != nil {
		return nil, err
	}
	query := api.db.Model(&models.Assignment{}).
		Select("assignments.id", "assignments.title", "assignments.end_at").
		Where(`EXISTS (
			SELECT 1 FROM assignment_problems
			WHERE assignment_problems.assignment_id = assignments.id
		)`).
		Where(`(
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
		)`, user.ID, user.ID)
	var rows []models.Assignment
	if err := query.Order("end_at desc").Limit(homeListLimit).Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := assignmentIDs(rows)
	totals, err := api.assignmentTotalMap(ids)
	if err != nil {
		return nil, err
	}
	done, err := api.assignmentDoneMap(c, ids)
	if err != nil {
		return nil, err
	}
	items := make([]contract.HomeAssignment, 0, len(rows))
	for _, row := range rows {
		total := totals[row.ID]
		if total == 0 {
			continue
		}
		items = append(items, contract.HomeAssignment{
			ID:     row.ID,
			Title:  row.Title,
			Status: assignmentStatus(row),
			Total:  total,
			Done:   done[row.ID],
		})
	}
	return items, nil
}

func (api *API) homeContests(c echo.Context) ([]contract.HomeContest, error) {
	var rows []models.Contest
	query := api.db.Model(&models.Contest{}).
		Select("contests.id", "contests.title", "contests.kind", "contests.start_at", "contests.end_at", "contests.freeze_at").
		Where(`EXISTS (
			SELECT 1 FROM contest_problems
			WHERE contest_problems.contest_id = contests.id
		)`).
		Where("contests.end_at > ?", time.Now())
	if !api.isAdmin(c) {
		query = query.Where(`EXISTS (
			SELECT 1 FROM contest_problems
			JOIN problems ON problems.id = contest_problems.problem_id
			WHERE contest_problems.contest_id = contests.id
			AND problems.visible = ?
		)`, true)
	}
	if err := query.Order("start_at desc").Limit(homeListLimit).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]contract.HomeContest, 0, len(rows))
	for _, row := range rows {
		items = append(items, contract.HomeContest{ID: row.ID, Title: row.Title, Status: contestStatus(row)})
	}
	return items, nil
}

func (api *API) updateNotice(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	var req contract.NoticeUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	if strings.TrimSpace(req.Content) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "notice content is required")
	}
	if err := validateTextBytes(req.Content, limits.MaxMarkdownBytes, "notice content is too large"); err != nil {
		return err
	}

	site, err := settings.Get(api.db)
	if err != nil {
		return err
	}
	site.Notice = req.Content
	if err := settings.Save(api.db, site); err != nil {
		return err
	}
	return api.home(c)
}

func (api *API) notice() string {

	site, err := settings.Get(api.db)
	if err != nil {
		return ""
	}
	return site.Notice
}

func heatmapFromCounts(counts map[string]int) []contract.HeatCell {
	today := time.Now()
	cells := make([]contract.HeatCell, 0, 365)
	for i := 364; i >= 0; i-- {
		day := today.AddDate(0, 0, -i)
		date := day.Format("2006-01-02")
		cells = append(cells, contract.HeatCell{Date: date, Count: counts[date]})
	}
	return cells
}
