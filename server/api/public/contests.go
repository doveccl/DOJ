package public

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	contract "github.com/doveccl/doj/contract/web"

	"github.com/doveccl/doj/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func (api *API) contests(c echo.Context) error {

	page, pageSize, offset, err := parsePage(c)
	if err != nil {
		return err
	}
	var rows []models.Contest
	query := api.db.Model(&models.Contest{})
	if q := strings.TrimSpace(c.QueryParam("q")); q != "" {
		like := "%" + q + "%"
		if id, err := parseQueryID(q, "invalid contest id"); err == nil {
			query = query.Where("id = ? OR LOWER(title) LIKE LOWER(?)", id, like)
		} else {
			query = query.Where("LOWER(title) LIKE LOWER(?)", like)
		}
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return err
	}
	if err := query.Session(&gorm.Session{}).Order("start_at desc").Limit(pageSize).Offset(offset).Find(&rows).Error; err != nil {
		return err
	}
	items, err := api.contestViews(rows, api.isAdmin(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, contract.Page[contract.Contest]{Items: items, Page: page, PageSize: pageSize, Total: total})
}

func (api *API) createContest(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	var req contract.ContestCreate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Kind = strings.TrimSpace(strings.ToUpper(req.Kind))
	if req.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if err := validateTitle(req.Title); err != nil {
		return err
	}
	if req.Kind == "" {
		req.Kind = "OI"
	}
	if !validContestKind(req.Kind) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid contest kind")
	}
	startAt, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid start time")
	}
	endAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid end time")
	}
	if !endAt.After(startAt) {
		return echo.NewHTTPError(http.StatusBadRequest, "end time must be after start time")
	}
	var freezeAt *time.Time
	if req.Kind == "ICPC" {
		var err error
		freezeAt, err = parseContestFreezeAt(req.FreezeAt, startAt, endAt)
		if err != nil {
			return err
		}
	}

	req.Problems = normalizeProblemRefs(req.Problems)
	if err := api.validateProblemRefs(req.Problems); err != nil {
		return err
	}
	row := models.Contest{Title: req.Title, Kind: req.Kind, StartAt: startAt, EndAt: endAt, FreezeAt: freezeAt}
	if err := api.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		for _, item := range req.Problems {
			link := models.ContestProblem{ContestID: row.ID, ProblemID: item.ID, Sort: item.Sort}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, contract.CreatedID{ID: row.ID})
}

func (api *API) updateContest(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid contest id")
	if err != nil {
		return err
	}
	var req contract.ContestUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Kind = strings.TrimSpace(strings.ToUpper(req.Kind))
	if req.Title == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "title is required")
	}
	if err := validateTitle(req.Title); err != nil {
		return err
	}
	if req.Kind == "" {
		req.Kind = "OI"
	}
	if !validContestKind(req.Kind) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid contest kind")
	}
	startAt, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid start time")
	}
	endAt, err := time.Parse(time.RFC3339, req.EndAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid end time")
	}
	if !endAt.After(startAt) {
		return echo.NewHTTPError(http.StatusBadRequest, "end time must be after start time")
	}
	var freezeAt *time.Time
	if req.Kind == "ICPC" {
		var err error
		freezeAt, err = parseContestFreezeAt(req.FreezeAt, startAt, endAt)
		if err != nil {
			return err
		}
	}

	var row models.Contest
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "contest not found")
		}
		return err
	}
	req.Problems = normalizeProblemRefs(req.Problems)
	if err := api.validateProblemRefs(req.Problems); err != nil {
		return err
	}
	row.Title = req.Title
	row.Kind = req.Kind
	row.StartAt = startAt
	row.EndAt = endAt
	row.FreezeAt = freezeAt
	if err := api.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&row).Error; err != nil {
			return err
		}
		if err := tx.Where("contest_id = ?", row.ID).Delete(&models.ContestProblem{}).Error; err != nil {
			return err
		}
		for _, item := range req.Problems {
			link := models.ContestProblem{ContestID: row.ID, ProblemID: item.ID, Sort: item.Sort}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, contract.CreatedID{ID: row.ID})
}

func (api *API) deleteContest(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := parseID(c, "id", "invalid contest id")
	if err != nil {
		return err
	}

	if err := api.db.Delete(&models.Contest{}, id).Error; err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (api *API) contest(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid contest id")
	}

	var row models.Contest
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "contest not found")
		}
		return err
	}
	var links []models.ContestProblem
	if err := api.db.Where("contest_id = ?", row.ID).Order("sort asc").Find(&links).Error; err != nil {
		return err
	}
	problems, err := api.contestProblems(c, row, links)
	if err != nil {
		return err
	}
	admin := api.isAdmin(c)
	rank := []contract.RankUser{}
	if row.Kind != "OI" || !contestRunning(row) || admin {
		freezeAt := api.contestFreezeCutoff(c, row)
		contestIncludeHidden := admin || contestRunning(row)
		rank, err = api.contestRank(row, problems, contestIncludeHidden, freezeAt)
		if err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, contract.ContestDetail{Contest: contestView(row, len(problems)), Problems: problems, Rank: rank})
}

func (api *API) activeContestFor(problemID uint, now time.Time) (*uint, error) {
	var row models.Contest
	err := api.db.
		Joins("JOIN contest_problems ON contest_problems.contest_id = contests.id").
		Where("contest_problems.problem_id = ? AND contests.start_at <= ? AND contests.end_at >= ?", problemID, now, now).
		Order("contests.start_at desc").
		First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row.ID, nil
}

func (api *API) rank(c echo.Context) error {
	page, pageSize, offset, err := parsePage(c)
	if err != nil {
		return err
	}
	var users []models.User
	if err := api.db.Order("id asc").Find(&users).Error; err != nil {
		return err
	}
	userIDs := make([]uint, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}
	stats, err := api.userStatsMap(c, userIDs)
	if err != nil {
		return err
	}
	items := make([]contract.RankUser, 0, len(users))
	for _, user := range users {
		got := stats[user.ID]
		ac := got.AC
		submit := got.Submit
		items = append(items, contract.RankUser{User: user.Name, Bio: user.Bio, Avatar: user.Avatar, AC: ac, Submit: submit})
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
	for index := range items {
		items[index].Rank = index + 1
	}
	total := int64(len(items))
	if offset >= len(items) {
		items = []contract.RankUser{}
	} else {
		end := offset + pageSize
		if end > len(items) {
			end = len(items)
		}
		items = items[offset:end]
	}
	return c.JSON(http.StatusOK, contract.Page[contract.RankUser]{Items: items, Page: page, PageSize: pageSize, Total: total})
}

func contestView(row models.Contest, total int) contract.Contest {
	freezeAt := row.FreezeAt
	if row.Kind == "OI" {
		freezeAt = nil
	}
	return contract.Contest{
		ID:       row.ID,
		Title:    row.Title,
		Kind:     row.Kind,
		StartAt:  row.StartAt,
		EndAt:    row.EndAt,
		FreezeAt: freezeAt,
		Status:   contestStatus(row),
		Total:    total,
	}
}

func (api *API) contestViews(rows []models.Contest, admin bool) ([]contract.Contest, error) {
	if len(rows) == 0 {
		return []contract.Contest{}, nil
	}
	totals, err := api.contestTotalMap(contestIDs(rows), admin)
	if err != nil {
		return nil, err
	}
	items := make([]contract.Contest, 0, len(rows))
	for _, row := range rows {
		items = append(items, contestView(row, totals[row.ID]))
	}
	return items, nil
}

func contestIDs(rows []models.Contest) []uint {
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func (api *API) contestTotalMap(ids []uint, admin bool) (map[uint]int, error) {
	ids = uniqueUint(ids)
	totals := map[uint]int{}
	if len(ids) == 0 {
		return totals, nil
	}
	var rows []struct {
		ContestID uint
		Count     int64
	}
	query := api.db.Model(&models.ContestProblem{}).
		Select("contest_problems.contest_id, count(*) as count").
		Where("contest_problems.contest_id IN ?", ids)
	if !admin {
		query = query.
			Joins("JOIN contests ON contests.id = contest_problems.contest_id").
			Joins("JOIN problems ON problems.id = contest_problems.problem_id AND problems.deleted_at IS NULL").
			Where("contests.end_at > ? OR problems.visible = ?", time.Now(), true)
	}
	if err := query.Group("contest_problems.contest_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		totals[row.ContestID] = int(row.Count)
	}
	return totals, nil
}

func (api *API) contestProblems(c echo.Context, contest models.Contest, links []models.ContestProblem) ([]contract.Problem, error) {
	if len(links) == 0 {
		return []contract.Problem{}, nil
	}
	admin := api.isAdmin(c)
	if !admin && !contestRunning(contest) && !contestEnded(contest) {
		return []contract.Problem{}, nil
	}
	ids := make([]uint, 0, len(links))
	for _, link := range links {
		ids = append(ids, link.ProblemID)
	}
	query := api.db.Model(&models.Problem{}).Where("id IN ?", uniqueUint(ids))
	if !admin && contestEnded(contest) {
		query = query.Where("visible = ?", true)
	}
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
		if !admin && !contestEnded(contest) {
			item.Tags = []string{}
		}
		item.Sort = link.Sort
		items = append(items, item)
	}
	return items, nil
}

func problemRowsByID(rows []models.Problem) map[uint]models.Problem {
	byID := make(map[uint]models.Problem, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	return byID
}

func contestStatus(row models.Contest) string {
	now := time.Now()
	status := "running"
	if row.StartAt.After(now) {
		status = "pending"
	}
	if row.EndAt.Before(now) {
		status = "ended"
	}
	if status == "running" && contestFrozenForUser(row, false) {
		status = "frozen"
	}
	return status
}

func contestFrozenForUser(row models.Contest, admin bool) bool {
	if admin || row.Kind == "OI" || row.FreezeAt == nil {
		return false
	}
	now := time.Now()
	return !now.Before(*row.FreezeAt) && now.Before(row.EndAt)
}

func (api *API) contestFreezeCutoff(c echo.Context, row models.Contest) *time.Time {
	if contestFrozenForUser(row, api.isAdmin(c)) {
		return row.FreezeAt
	}
	return nil
}

func contestRunning(row models.Contest) bool {
	now := time.Now()
	return !now.Before(row.StartAt) && now.Before(row.EndAt)
}

func contestEnded(row models.Contest) bool {
	return !time.Now().Before(row.EndAt)
}

func validContestKind(kind string) bool {
	return kind == "OI" || kind == "ICPC"
}

func parseContestFreezeAt(raw string, startAt time.Time, endAt time.Time) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	freezeAt, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "invalid freeze time")
	}
	if freezeAt.Before(startAt) || !freezeAt.Before(endAt) {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "freeze time must be between start and end")
	}
	return &freezeAt, nil
}
