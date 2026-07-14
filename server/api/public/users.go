package public

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/validate"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (api *API) users(c echo.Context) error {
	q := strings.TrimSpace(c.QueryParam("q"))
	var rows []models.User
	query := api.db.Select("id", "name").Order("id asc").Limit(50)
	if q != "" {
		like := "%" + q + "%"
		query = query.Where("LOWER(name) LIKE LOWER(?)", like)
	}
	if err := query.Find(&rows).Error; err != nil {
		return err
	}
	items := make([]contract.UserOption, 0, len(rows))
	for _, row := range rows {
		items = append(items, contract.UserOption{ID: row.ID, Name: row.Name})
	}
	return c.JSON(http.StatusOK, items)
}

func (api *API) user(c echo.Context) error {
	nameKey := validate.NameKey(c.Param("name"))
	if nameKey == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user name is required")
	}
	solvedPage, solvedPageSize, solvedOffset, err := parseNamedPage(c, "solved", userSolvedPageSize)
	if err != nil {
		return err
	}

	var row models.User
	if err := api.db.Where("LOWER(name) = ?", nameKey).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return err
	}

	solved, err := api.solvedProblems(c, row.ID, solvedPage, solvedPageSize, solvedOffset)
	if err != nil {
		return err
	}
	activities, err := api.userActivities(c, row.ID)
	if err != nil {
		return err
	}
	heatmap, err := api.userHeatmap(row.ID)
	if err != nil {
		return err
	}
	ac, submit, err := api.userStats(c, row.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, contract.UserProfile{
		User:       contract.PublicUser{Name: row.Name, Bio: row.Bio, Avatar: row.Avatar, Admin: row.Admin, AC: ac, Submit: submit},
		Heatmap:    heatmap,
		Solved:     solved,
		Activities: activities,
	})
}

type userStatsCounts struct {
	AC     int
	Submit int
}

func (api *API) userStats(c echo.Context, userID uint) (int, int, error) {
	submitQuery := api.db.Model(&models.Submission{}).Where("submissions.user_id = ?", userID)
	var submit int64
	if err := submitQuery.Count(&submit).Error; err != nil {
		return 0, 0, err
	}

	acQuery := api.db.Model(&models.Submission{}).
		Where("submissions.user_id = ? AND submissions.status = ?", userID, "AC").
		Distinct("submissions.problem_id")
	acQuery, err := api.filterVisibleResults(c, acQuery)
	if err != nil {
		return 0, 0, err
	}
	var ac int64
	if err := acQuery.Count(&ac).Error; err != nil {
		return 0, 0, err
	}
	return int(ac), int(submit), nil
}

func (api *API) userStatsMap(c echo.Context, userIDs []uint) (map[uint]userStatsCounts, error) {
	userIDs = uniqueUint(userIDs)
	stats := map[uint]userStatsCounts{}
	if len(userIDs) == 0 {
		return stats, nil
	}
	var submits []struct {
		UserID uint
		Count  int64
	}
	submitQuery := api.db.Model(&models.Submission{}).
		Select("user_id, count(*) as count").
		Where("user_id IN ?", userIDs).
		Group("user_id")
	if err := submitQuery.Find(&submits).Error; err != nil {
		return nil, err
	}
	for _, row := range submits {
		item := stats[row.UserID]
		item.Submit = int(row.Count)
		stats[row.UserID] = item
	}
	var acs []struct {
		UserID uint
		Count  int64
	}
	acQuery := api.db.Model(&models.Submission{}).
		Select("user_id, count(DISTINCT problem_id) as count").
		Where("user_id IN ? AND status = ?", userIDs, "AC").
		Group("user_id")
	acQuery, err := api.filterVisibleResults(c, acQuery)
	if err != nil {
		return nil, err
	}
	if err := acQuery.Find(&acs).Error; err != nil {
		return nil, err
	}
	for _, row := range acs {
		item := stats[row.UserID]
		item.AC = int(row.Count)
		stats[row.UserID] = item
	}
	return stats, nil
}

func (api *API) userActivities(c echo.Context, userID uint) ([]contract.UserActivity, error) {
	var submissions []models.Submission
	if err := api.db.
		Select("submissions.id", "submissions.user_id", "submissions.problem_id", "submissions.assignment_id", "submissions.contest_id", "submissions.status", "submissions.score", "submissions.created_at").
		Where("submissions.user_id = ?", userID).
		Order("submissions.created_at desc").
		Limit(userActivityLimit).
		Find(&submissions).Error; err != nil {
		return nil, err
	}
	views, err := api.submissionViews(c, submissions)
	if err != nil {
		return nil, err
	}
	items := make([]contract.UserActivity, 0, len(submissions)+userActivityLimit)
	problemIDs := make([]uint, 0, len(submissions))
	for _, submission := range submissions {
		problemIDs = append(problemIDs, submission.ProblemID)
	}
	titles, err := api.problemTitleMap(problemIDs)
	if err != nil {
		return nil, err
	}
	for index, submission := range submissions {
		title := titles[submission.ProblemID]
		if title == "" {
			title = "P" + strconv.Itoa(int(submission.ProblemID))
		}
		status := submission.Status
		if !views[index].Result {
			status = "pending"
		}
		items = append(items, contract.UserActivity{
			Type:         "submission",
			ID:           submission.ID,
			Title:        title,
			Status:       status,
			ProblemID:    submission.ProblemID,
			ProblemTitle: title,
			CreatedAt:    submission.CreatedAt,
		})
	}

	var discussions []models.Discussion
	discussionQuery := api.db.Model(&models.Discussion{}).Select("id", "title", "created_at").Where("user_id = ?", userID).Order("created_at desc").Limit(userActivityLimit)
	if err := discussionQuery.Find(&discussions).Error; err != nil {
		return nil, err
	}
	for _, row := range discussions {
		items = append(items, contract.UserActivity{
			Type:      "discussion",
			ID:        row.ID,
			Title:     row.Title,
			CreatedAt: row.CreatedAt,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	if len(items) > userActivityLimit {
		items = items[:userActivityLimit]
	}
	return items, nil
}

func (api *API) solvedProblems(c echo.Context, userID uint, page int, pageSize int, offset int) (contract.Page[contract.SolvedProblem], error) {
	base := api.db.Model(&models.Submission{}).
		Where("submissions.user_id = ? AND submissions.status = ?", userID, "AC").
		Distinct("submissions.problem_id")
	base, err := api.filterVisibleResults(c, base)
	if err != nil {
		return contract.Page[contract.SolvedProblem]{}, err
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return contract.Page[contract.SolvedProblem]{}, err
	}
	if total == 0 {
		return contract.Page[contract.SolvedProblem]{Items: []contract.SolvedProblem{}, Page: page, PageSize: pageSize, Total: 0}, nil
	}

	var rows []struct {
		ProblemID uint
	}
	query := api.db.Model(&models.Submission{}).
		Select("submissions.problem_id").
		Where("submissions.user_id = ? AND submissions.status = ?", userID, "AC").
		Group("submissions.problem_id").
		Order("MAX(submissions.created_at) desc").
		Limit(pageSize).
		Offset(offset)
	query, err = api.filterVisibleResults(c, query)
	if err != nil {
		return contract.Page[contract.SolvedProblem]{}, err
	}
	if err := query.Find(&rows).Error; err != nil {
		return contract.Page[contract.SolvedProblem]{}, err
	}
	problemIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		problemIDs = append(problemIDs, row.ProblemID)
	}
	if len(problemIDs) == 0 {
		return contract.Page[contract.SolvedProblem]{Items: []contract.SolvedProblem{}, Page: page, PageSize: pageSize, Total: total}, nil
	}
	problemQuery := api.db.Model(&models.Problem{}).Select("id", "title", "tags").Where("id IN ?", problemIDs)
	var problems []models.Problem
	if err := problemQuery.Find(&problems).Error; err != nil {
		return contract.Page[contract.SolvedProblem]{}, err
	}
	byID := problemRowsByID(problems)
	items := make([]contract.SolvedProblem, 0, len(problemIDs))
	for _, id := range problemIDs {
		problem, ok := byID[id]
		if !ok {
			continue
		}
		items = append(items, contract.SolvedProblem{
			ID:    problem.ID,
			Title: problem.Title,
			Tags:  readTags([]byte(problem.Tags)),
		})
	}
	return contract.Page[contract.SolvedProblem]{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (api *API) userHeatmap(userID uint) ([]contract.HeatCell, error) {
	since := time.Now().AddDate(-1, 0, 0)
	var rows []models.Submission
	query := api.db.Model(&models.Submission{}).Select("created_at").Where("submissions.user_id = ? AND submissions.created_at >= ?", userID, since)
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, row := range rows {
		counts[row.CreatedAt.Format("2006-01-02")]++
	}
	return heatmapFromCounts(counts), nil
}

func (api *API) userName(id uint) string {

	var user models.User
	if err := api.db.First(&user, id).Error; err == nil && user.Name != "" {
		return user.Name
	}

	return strconv.Itoa(int(id))
}

func authorName(id uint, users map[uint]models.User) string {
	if user := users[id]; user.Name != "" {
		return user.Name
	}
	return strconv.Itoa(int(id))
}
