package public

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/doveccl/doj/contract/limits"
	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/events"
	"github.com/doveccl/doj/server/judge"
	"github.com/doveccl/doj/server/validate"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (api *API) submissions(c echo.Context) error {

	page, pageSize, offset, err := parsePage(c)
	if err != nil {
		return err
	}
	var rows []models.Submission
	query := api.db.Model(&models.Submission{})
	if !api.isAdmin(c) {
		query = query.Joins("JOIN problems ON problems.id = submissions.problem_id")
		query = query.Where("problems.deleted_at IS NULL")
	}
	if problem := c.QueryParam("problem"); problem != "" {
		id, err := parseProblemQuery(problem)
		if err != nil {
			return err
		}
		query = query.Where("submissions.problem_id = ?", id)
	}
	if user := strings.TrimSpace(c.QueryParam("user")); user != "" {
		query = query.Joins("JOIN users submission_users ON submission_users.id = submissions.user_id AND submission_users.deleted_at IS NULL").
			Where("LOWER(submission_users.name) = ?", validate.NameKey(user))
	}
	if language := strings.TrimSpace(c.QueryParam("language")); language != "" {
		query = query.Where("submissions.language = ?", language)
	}
	if status := strings.TrimSpace(c.QueryParam("status")); status != "" {
		if !validSubmissionListStatus(status) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid submission status")
		}
		query = query.Where("submissions.status = ?", status)
		if !submissionLive(status) {
			query, err = api.filterVisibleResults(c, query)
			if err != nil {
				return err
			}
		}
	}
	if assignment := c.QueryParam("assignment"); assignment != "" {
		id, err := parseQueryID(assignment, "invalid assignment id")
		if err != nil {
			return err
		}
		query = query.Where("submissions.assignment_id = ?", id)
	}
	if contest := c.QueryParam("contest"); contest != "" {
		id, err := parseQueryID(contest, "invalid contest id")
		if err != nil {
			return err
		}
		query = query.Where("submissions.contest_id = ?", id)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return err
	}
	if err := query.Session(&gorm.Session{}).
		Select("submissions.id", "submissions.problem_id", "submissions.user_id", "submissions.assignment_id", "submissions.contest_id", "submissions.language", "submissions.status", "submissions.score", "submissions.message", "submissions.time_ms", "submissions.memory_kb", "submissions.public", "submissions.created_at").
		Order("submissions.created_at desc").
		Limit(pageSize).
		Offset(offset).
		Find(&rows).Error; err != nil {
		return err
	}
	items, err := api.submissionListItems(c, rows)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, contract.Page[contract.SubmissionListItem]{Items: items, Page: page, PageSize: pageSize, Total: total})
}

func validSubmissionListStatus(status string) bool {
	switch status {
	case "queued", "judging", "AC", "CE", "WA", "PE", "TLE", "MLE", "OLE", "RE", "SE":
		return true
	default:
		return false
	}
}

func submissionLive(status string) bool {
	return status == "queued" || status == "judging"
}

func (api *API) submit(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	var req contract.SubmitRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Language = strings.TrimSpace(req.Language)
	req.Code = strings.TrimSpace(req.Code)
	if req.ProblemID == 0 || req.Language == "" || req.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "problem, language and code are required")
	}
	if err := validateTextBytes(req.Code, limits.MaxSourceBytes, "source code is too large"); err != nil {
		return err
	}

	var problem models.Problem
	if err := api.db.First(&problem, req.ProblemID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	var language models.Language
	if err := api.db.First(&language, "id = ?", req.Language).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusBadRequest, "language not found")
		}
		return err
	}
	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	assignmentID, contestID, err := api.submitContext(user.ID, req.ProblemID, req.AssignmentID, req.ContestID, time.Now())
	if err != nil {
		return err
	}
	if assignmentID == nil && contestID == nil && !user.Admin {
		contests, err := api.problemContestState(problem.ID)
		if err != nil {
			return err
		}
		visible := problem.Visible && !contests.unfinished
		if !visible {
			allowed := contests.ended && !contests.unfinished
			if !allowed {
				allowed, err = api.problemInAssignmentForUser(problem.ID, user.ID, true)
				if err != nil {
					return err
				}
			}
			if !allowed {
				return echo.NewHTTPError(http.StatusNotFound, "problem not found")
			}
		}
	}
	if err := validateProblemReadyForSubmit(problem); err != nil {
		return err
	}
	row := models.Submission{
		UserID:       user.ID,
		ProblemID:    req.ProblemID,
		AssignmentID: assignmentID,
		ContestID:    contestID,
		Language:     req.Language,
		Code:         req.Code,
		Status:       "queued",
		Public:       req.Public,
	}
	if err := api.db.Transaction(func(tx *gorm.DB) error {
		var locked models.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&locked, user.ID).Error; err != nil {
			return err
		}
		var outstanding int64
		if err := tx.Model(&models.Submission{}).Where("user_id = ? AND status IN ?", user.ID, []string{"queued", "judging"}).Count(&outstanding).Error; err != nil {
			return err
		}
		if outstanding >= limits.MaxOutstandingSubmissions {
			return echo.NewHTTPError(http.StatusTooManyRequests, "too many outstanding submissions")
		}
		return tx.Create(&row).Error
	}); err != nil {
		return err
	}
	events.SubmissionChanged()
	return c.JSON(http.StatusCreated, contract.CreatedID{ID: row.ID})
}

func validateProblemReadyForSubmit(problem models.Problem) error {
	pkg, err := problemPackageView(problem.Package)
	if err != nil {
		return err
	}
	if pkg.Cases == 0 {
		return echo.NewHTTPError(http.StatusConflict, "problem has no test data")
	}
	if problem.Mode == "custom" && !hasCustomJudgeDockerfile(pkg.Judge) {
		return echo.NewHTTPError(http.StatusConflict, "custom judge requires Dockerfile")
	}
	return nil
}

func hasCustomJudgeDockerfile(files []contract.PackageFile) bool {
	for _, file := range files {
		if file.Name == "Dockerfile" {
			return true
		}
	}
	return false
}

func (api *API) submitContext(userID uint, problemID uint, assignmentID *uint, contestID *uint, now time.Time) (*uint, *uint, error) {
	if assignmentID != nil && contestID != nil {
		return nil, nil, echo.NewHTTPError(http.StatusBadRequest, "assignment and contest are mutually exclusive")
	}
	if assignmentID != nil {
		if *assignmentID == 0 {
			return nil, nil, echo.NewHTTPError(http.StatusBadRequest, "invalid assignment id")
		}
		var assignment models.Assignment
		if err := api.db.First(&assignment, *assignmentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, nil, echo.NewHTTPError(http.StatusNotFound, "assignment not found")
			}
			return nil, nil, err
		}
		assigned, err := api.userAssignedTo(assignment.ID, userID)
		if err != nil {
			return nil, nil, err
		}
		if !assigned {
			return nil, nil, echo.NewHTTPError(http.StatusNotFound, "assignment not found")
		}
		if !now.Before(assignment.EndAt) {
			return nil, nil, echo.NewHTTPError(http.StatusForbidden, "assignment has ended")
		}
		var count int64
		if err := api.db.Model(&models.AssignmentProblem{}).Where("assignment_id = ? AND problem_id = ?", assignment.ID, problemID).Count(&count).Error; err != nil {
			return nil, nil, err
		}
		if count == 0 {
			return nil, nil, echo.NewHTTPError(http.StatusBadRequest, "problem is not in assignment")
		}
		return assignmentID, nil, nil
	}
	if contestID != nil {
		if *contestID == 0 {
			return nil, nil, echo.NewHTTPError(http.StatusBadRequest, "invalid contest id")
		}
		var contest models.Contest
		if err := api.db.First(&contest, *contestID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, nil, echo.NewHTTPError(http.StatusNotFound, "contest not found")
			}
			return nil, nil, err
		}
		if now.Before(contest.StartAt) || !now.Before(contest.EndAt) {
			return nil, nil, echo.NewHTTPError(http.StatusForbidden, "contest is not running")
		}
		var count int64
		if err := api.db.Model(&models.ContestProblem{}).Where("contest_id = ? AND problem_id = ?", contest.ID, problemID).Count(&count).Error; err != nil {
			return nil, nil, err
		}
		if count == 0 {
			return nil, nil, echo.NewHTTPError(http.StatusBadRequest, "problem is not in contest")
		}
		return nil, contestID, nil
	}
	return nil, nil, nil
}

func (api *API) submission(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid submission id")
	}

	var row models.Submission
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "submission not found")
		}
		return err
	}
	if !api.isAdmin(c) {
		var count int64
		if err := api.db.Model(&models.Problem{}).Where("id = ?", row.ProblemID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return echo.NewHTTPError(http.StatusNotFound, "submission not found")
		}
	}
	view, err := api.submissionView(c, row)
	if err != nil {
		return err
	}
	var cases []models.Case
	if view.Result {
		if err := api.db.Where("submission_id = ?", row.ID).Order("no asc").Find(&cases).Error; err != nil {
			return err
		}
	}
	items := make([]contract.Case, 0, len(cases))
	for _, item := range cases {
		items = append(items, contract.Case{No: item.No, ID: item.CaseID, Status: item.Status, Score: item.Score, TimeMS: item.TimeMS, MemoryKB: item.MemoryKB, Message: item.Message})
	}
	code := ""
	if view.Code {
		code = row.Code
	}
	var progress *contract.SubmissionProgress
	if view.Result {
		progress = api.submissionProgress(c.Request().Context(), row)
	}
	submission, err := api.submissionPayload(c, row)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, contract.SubmissionDetail{Submission: submission, Code: code, Cases: items, Progress: progress})
}

func (api *API) updateSubmission(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid submission id")
	}
	var req contract.SubmissionUpdate
	if err := c.Bind(&req); err != nil {
		return err
	}
	var row models.Submission
	if err := api.db.First(&row, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "submission not found")
		}
		return err
	}
	user, err := api.currentUser(c)
	if err != nil {
		return err
	}
	if !user.Admin && user.ID != row.UserID {
		return echo.NewHTTPError(http.StatusForbidden, "submission can only be updated by owner or admin")
	}
	row.Public = req.Public
	if err := api.db.Model(&row).Update("public", row.Public).Error; err != nil {
		return err
	}
	events.SubmissionChanged()
	return c.JSON(http.StatusOK, contract.CreatedID{ID: row.ID})
}

func (api *API) rejudgeSubmission(c echo.Context) error {
	if err := api.requireAdmin(c); err != nil {
		return err
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid submission id")
	}
	var row models.Submission
	err = api.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&row, id).Error; err != nil {
			return err
		}
		if err := tx.Where("submission_id = ?", row.ID).Delete(&models.Case{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&row).Updates(rejudgeUpdates()).Error; err != nil {
			return err
		}
		return tx.First(&row, row.ID).Error
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "submission not found")
		}
		return err
	}
	judge.DeleteProgress(c.Request().Context(), row.ID)
	events.SubmissionChanged()
	return c.JSON(http.StatusOK, contract.CreatedID{ID: row.ID})
}

func (api *API) rejudgeProblem(c echo.Context) error {
	id, err := api.requireProblemAdmin(c)
	if err != nil {
		return err
	}
	var count int64
	var ids []uint
	err = api.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Submission{}).Where("problem_id = ?", id).Pluck("id", &ids).Error; err != nil {
			return err
		}
		count = int64(len(ids))
		if len(ids) == 0 {
			return nil
		}
		if err := tx.Where("submission_id IN ?", ids).Delete(&models.Case{}).Error; err != nil {
			return err
		}
		return tx.Model(&models.Submission{}).Where("id IN ?", ids).Updates(rejudgeUpdates()).Error
	})
	if err != nil {
		return err
	}
	for _, id := range ids {
		judge.DeleteProgress(c.Request().Context(), id)
	}
	events.SubmissionChanged()
	return c.JSON(http.StatusOK, contract.CountResult{Count: count})
}

func (api *API) submissionProgress(ctx context.Context, row models.Submission) *contract.SubmissionProgress {
	if !submissionLive(row.Status) {
		return nil
	}
	progress, err := judge.ReadProgress(ctx, row.ID, row.Attempt)
	if err != nil || progress == nil {
		return nil
	}
	return &contract.SubmissionProgress{Stage: progress.Stage, Done: progress.Done, Total: progress.Total, UpdatedAt: progress.UpdatedAt}
}

func rejudgeUpdates() map[string]any {
	return map[string]any{
		"status":      "queued",
		"score":       0,
		"message":     "",
		"attempt":     gorm.Expr("attempt + 1"),
		"judger_id":   nil,
		"lease_until": nil,
		"time_ms":     nil,
		"memory_kb":   nil,
	}
}

func (api *API) submissionPayload(c echo.Context, row models.Submission) (contract.Submission, error) {
	items, err := api.submissionPayloads(c, []models.Submission{row})
	if err != nil {
		return contract.Submission{}, err
	}
	if len(items) > 0 {
		return items[0], nil
	}
	return submissionPayloadFromRefs(row, nil, nil), nil
}

func (api *API) submissionListItems(c echo.Context, rows []models.Submission) ([]contract.SubmissionListItem, error) {
	items, err := api.submissionPayloads(c, rows)
	if err != nil {
		return nil, err
	}
	list := make([]contract.SubmissionListItem, 0, len(items))
	for _, item := range items {
		list = append(list, submissionListItemFromPayload(item))
	}
	return list, nil
}

func (api *API) submissionPayloads(c echo.Context, rows []models.Submission) ([]contract.Submission, error) {
	if len(rows) == 0 {
		return []contract.Submission{}, nil
	}
	problemIDs := make([]uint, 0, len(rows))
	userIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		problemIDs = append(problemIDs, row.ProblemID)
		userIDs = append(userIDs, row.UserID)
	}
	titles, err := api.problemTitleMap(problemIDs)
	if err != nil {
		return nil, err
	}
	users, err := api.userMap(userIDs)
	if err != nil {
		return nil, err
	}
	views, err := api.submissionViews(c, rows)
	if err != nil {
		return nil, err
	}
	items := make([]contract.Submission, 0, len(rows))
	for index, row := range rows {
		item := submissionPayloadFromRefs(row, titles, users)
		if !views[index].Result {
			hideSubmissionResult(&item)
		}
		items = append(items, item)
	}
	return items, nil
}

func submissionListItemFromPayload(row contract.Submission) contract.SubmissionListItem {
	return contract.SubmissionListItem{
		ID:           row.ID,
		ProblemID:    row.ProblemID,
		ProblemTitle: row.ProblemTitle,
		AssignmentID: row.AssignmentID,
		ContestID:    row.ContestID,
		User:         row.User,
		Avatar:       row.Avatar,
		Language:     row.Language,
		Status:       row.Status,
		Score:        row.Score,
		TimeMS:       row.TimeMS,
		MemoryKB:     row.MemoryKB,
		CreatedAt:    row.CreatedAt,
	}
}

func submissionPayloadFromRefs(row models.Submission, titles map[uint]string, users map[uint]models.User) contract.Submission {
	title := titles[row.ProblemID]
	user := users[row.UserID]
	if title == "" {
		title = "P" + strconv.Itoa(int(row.ProblemID))
	}
	if user.Name == "" {
		user.Name = strconv.Itoa(int(row.UserID))
	}
	return contract.Submission{
		ID:           row.ID,
		ProblemID:    row.ProblemID,
		ProblemTitle: title,
		AssignmentID: row.AssignmentID,
		ContestID:    row.ContestID,
		User:         user.Name,
		Avatar:       user.Avatar,
		Language:     row.Language,
		Status:       row.Status,
		Score:        row.Score,
		Message:      row.Message,
		TimeMS:       row.TimeMS,
		MemoryKB:     row.MemoryKB,
		Public:       row.Public,
		CreatedAt:    row.CreatedAt,
	}
}

func hideSubmissionResult(row *contract.Submission) {
	row.Status = "pending"
	row.Score = 0
	row.Message = ""
	row.TimeMS = nil
	row.MemoryKB = nil
}

func (api *API) problemTitleMap(ids []uint) (map[uint]string, error) {
	ids = uniqueUint(ids)
	titles := map[uint]string{}
	if len(ids) == 0 {
		return titles, nil
	}
	var rows []models.Problem
	if err := api.db.Select("id", "title").Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		titles[row.ID] = row.Title
	}
	return titles, nil
}

func (api *API) userMap(ids []uint) (map[uint]models.User, error) {
	ids = uniqueUint(ids)
	users := map[uint]models.User{}
	if len(ids) == 0 {
		return users, nil
	}
	var rows []models.User
	if err := api.db.Select("id", "name", "avatar").Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		users[row.ID] = row
	}
	return users, nil
}
