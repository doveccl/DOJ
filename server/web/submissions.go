package web

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/doveccl/doj/models"
	"github.com/doveccl/doj/server/events"
	judgersvc "github.com/doveccl/doj/server/judger"
	"github.com/doveccl/doj/utils"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
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
			Where("LOWER(submission_users.name) = ?", utils.NameKey(user))
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
	return c.JSON(http.StatusOK, PageResult[SubmissionListItem]{Items: items, Page: page, PageSize: pageSize, Total: total})
}

func (api *API) submit(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	var req SubmitRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	req.Language = strings.TrimSpace(req.Language)
	req.Code = strings.TrimSpace(req.Code)
	if req.ProblemID == 0 || req.Language == "" || req.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "problem, language and code are required")
	}
	if err := validateTextBytes(req.Code, utils.MaxSourceBytes, "source code is too large"); err != nil {
		return err
	}

	var problem models.Problem
	if err := api.db.First(&problem, req.ProblemID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
		return err
	}
	if !api.isAdmin(c) {
		visible, err := api.problemVisibleInDetail(c, problem)
		if err != nil {
			return err
		}
		if !visible {
			return echo.NewHTTPError(http.StatusNotFound, "problem not found")
		}
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
	assignmentID, contestID, err := api.inferSubmitScopes(user.ID, req.ProblemID, time.Now())
	if err != nil {
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
	if err := api.db.Create(&row).Error; err != nil {
		return err
	}
	events.SubmissionChanged()
	return c.JSON(http.StatusCreated, CreatedID{ID: row.ID})
}

func (api *API) inferSubmitScopes(userID uint, problemID uint, now time.Time) (*uint, *uint, error) {
	assignmentID, err := api.activeAssignmentFor(userID, problemID, now)
	if err != nil {
		return nil, nil, err
	}
	contestID, err := api.activeContestFor(problemID, now)
	if err != nil {
		return nil, nil, err
	}
	return assignmentID, contestID, nil
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
	items := make([]CaseDTO, 0, len(cases))
	for _, item := range cases {
		items = append(items, CaseDTO{No: item.No, Status: item.Status, TimeMS: item.TimeMS, MemoryKB: item.MemoryKB, Message: item.Message})
	}
	code := ""
	if view.Code {
		code = row.Code
	}
	var progress *ProgressDTO
	if view.Result {
		progress = api.submissionProgress(c.Request().Context(), row)
	}
	submission, err := api.submissionDTO(c, row)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, SubmissionDetail{Submission: submission, Code: code, Cases: items, Progress: progress})
}

func (api *API) updateSubmission(c echo.Context) error {
	if err := api.requireSignedIn(c); err != nil {
		return err
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid submission id")
	}
	var req SubmissionUpdate
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
	return c.JSON(http.StatusOK, CreatedID{ID: row.ID})
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
	judgersvc.DeleteProgress(c.Request().Context(), row.ID)
	events.SubmissionChanged()
	return c.JSON(http.StatusOK, CreatedID{ID: row.ID})
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
		judgersvc.DeleteProgress(c.Request().Context(), id)
	}
	events.SubmissionChanged()
	return c.JSON(http.StatusOK, CountResult{Count: count})
}

func (api *API) submissionProgress(ctx context.Context, row models.Submission) *ProgressDTO {
	if row.Status != "queued" && row.Status != "judging" {
		return nil
	}
	progress, err := judgersvc.ReadProgress(ctx, row.ID, row.Attempt)
	if err != nil || progress == nil {
		return nil
	}
	return &ProgressDTO{Stage: progress.Stage, Done: progress.Done, Total: progress.Total, UpdatedAt: progress.UpdatedAt}
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

func (api *API) submissionDTO(c echo.Context, row models.Submission) (SubmissionDTO, error) {
	items, err := api.submissionDTOs(c, []models.Submission{row})
	if err != nil {
		return SubmissionDTO{}, err
	}
	if len(items) > 0 {
		return items[0], nil
	}
	return submissionDTOFromRefs(row, nil, nil), nil
}

func (api *API) submissionListItems(c echo.Context, rows []models.Submission) ([]SubmissionListItem, error) {
	items, err := api.submissionDTOs(c, rows)
	if err != nil {
		return nil, err
	}
	list := make([]SubmissionListItem, 0, len(items))
	for _, item := range items {
		list = append(list, submissionListItemFromDTO(item))
	}
	return list, nil
}

func (api *API) submissionDTOs(c echo.Context, rows []models.Submission) ([]SubmissionDTO, error) {
	if len(rows) == 0 {
		return []SubmissionDTO{}, nil
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
	users, err := api.userNameMap(userIDs)
	if err != nil {
		return nil, err
	}
	items := make([]SubmissionDTO, 0, len(rows))
	for _, row := range rows {
		dto := submissionDTOFromRefs(row, titles, users)
		view, err := api.submissionView(c, row)
		if err != nil {
			return nil, err
		}
		if !view.Result {
			hideSubmissionResult(&dto)
		}
		items = append(items, dto)
	}
	return items, nil
}

func submissionListItemFromDTO(row SubmissionDTO) SubmissionListItem {
	return SubmissionListItem{
		ID:           row.ID,
		ProblemID:    row.ProblemID,
		ProblemTitle: row.ProblemTitle,
		User:         row.User,
		Language:     row.Language,
		Status:       row.Status,
		TimeMS:       row.TimeMS,
		MemoryKB:     row.MemoryKB,
		CreatedAt:    row.CreatedAt,
	}
}

func submissionDTOFromRefs(row models.Submission, titles map[uint]string, users map[uint]string) SubmissionDTO {
	title := titles[row.ProblemID]
	userName := users[row.UserID]
	if title == "" {
		title = "P" + strconv.Itoa(int(row.ProblemID))
	}
	if userName == "" {
		userName = strconv.Itoa(int(row.UserID))
	}
	return SubmissionDTO{
		ID:           row.ID,
		ProblemID:    row.ProblemID,
		ProblemTitle: title,
		User:         userName,
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

func hideSubmissionResult(row *SubmissionDTO) {
	row.Status = "pending"
	row.Score = 0
	row.Message = ""
	row.TimeMS = nil
	row.MemoryKB = nil
}

func pendingRecord(row RecordDTO) RecordDTO {
	row.Status = "pending"
	row.Score = 0
	return row
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

func (api *API) userNameMap(ids []uint) (map[uint]string, error) {
	ids = uniqueUint(ids)
	names := map[uint]string{}
	if len(ids) == 0 {
		return names, nil
	}
	var rows []models.User
	if err := api.db.Select("id", "name").Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		names[row.ID] = row.Name
	}
	return names, nil
}
