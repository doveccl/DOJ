package api

import (
	"context"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/server/api/admin"
	"github.com/doveccl/doj/server/backup"
)

type Health struct {
	Status string `json:"status"`
}

type Site struct {
	SiteName                string `json:"siteName" maxLength:"64"`
	AllowRegistration       bool   `json:"allowRegistration"`
	AllowGuestAccess        bool   `json:"allowGuestAccess"`
	DefaultSubmissionPublic bool   `json:"defaultSubmissionPublic"`
}

type ProblemListItem struct {
	ID       uint     `json:"id"`
	Sort     string   `json:"sort,omitempty" maxLength:"16"`
	Title    string   `json:"title" maxLength:"256"`
	Tags     []string `json:"tags"`
	Visible  bool     `json:"visible"`
	Mode     string   `json:"mode" maxLength:"16"`
	TimeMS   int      `json:"timeMs"`
	MemoryMB int      `json:"memoryMb"`
}

type Problem struct {
	ID        uint     `json:"id"`
	Sort      string   `json:"sort,omitempty" maxLength:"16"`
	Title     string   `json:"title" maxLength:"256"`
	Statement string   `json:"statement,omitempty"`
	Tags      []string `json:"tags"`
	Visible   bool     `json:"visible"`
	Mode      string   `json:"mode" maxLength:"16"`
	TimeMS    int      `json:"timeMs"`
	MemoryMB  int      `json:"memoryMb"`
	Cases     int      `json:"cases"`
	DataBytes int64    `json:"dataBytes"`
}

type ProblemState struct {
	ProblemID   uint                    `json:"problemId"`
	AC          int                     `json:"ac"`
	Submit      int                     `json:"submit"`
	Discussions *int                    `json:"discussions,omitempty"`
	Status      string                  `json:"status" enum:"none,tried,ac,pending"`
	Submission  *contract.ProblemRecord `json:"submission,omitempty"`
}

type Assignment struct {
	ID     uint      `json:"id"`
	Title  string    `json:"title" maxLength:"256"`
	EndAt  time.Time `json:"endAt"`
	Status string    `json:"status"`
	Total  int       `json:"total"`
	Done   int       `json:"done"`
	Users  []uint    `json:"users"`
	Groups []uint    `json:"groups"`
}

type AssignmentListItem struct {
	ID     uint      `json:"id"`
	Title  string    `json:"title" maxLength:"256"`
	EndAt  time.Time `json:"endAt"`
	Status string    `json:"status"`
	Total  int       `json:"total"`
	Done   int       `json:"done"`
}

type AssignmentProgress struct {
	User     string                               `json:"user" maxLength:"32"`
	AC       int                                  `json:"ac"`
	Submit   int                                  `json:"submit"`
	Problems []contract.AssignmentProblemProgress `json:"problems"`
}

type RankUser struct {
	Rank     int                    `json:"rank"`
	User     string                 `json:"user" maxLength:"32"`
	Bio      string                 `json:"bio" maxLength:"256"`
	Avatar   string                 `json:"avatar" maxLength:"512"`
	AC       int                    `json:"ac"`
	Submit   int                    `json:"submit"`
	Score    int                    `json:"score"`
	Penalty  int                    `json:"penalty"`
	Problems []contract.RankProblem `json:"problems"`
}

type assignmentDetail struct {
	Assignment  Assignment           `json:"assignment"`
	Description string               `json:"description"`
	Problems    []ProblemListItem    `json:"problems"`
	Progress    []AssignmentProgress `json:"progress"`
}

type contestDetail struct {
	Contest     contract.Contest  `json:"contest"`
	Description string            `json:"description"`
	Problems    []ProblemListItem `json:"problems"`
	Rank        []RankUser        `json:"rank"`
}

type submissionDetail struct {
	Submission contract.Submission          `json:"submission"`
	Code       string                       `json:"code"`
	Cases      []contract.Case              `json:"cases"`
	Progress   *contract.SubmissionProgress `json:"progress,omitempty"`
}

type userProfile struct {
	User       contract.PublicUser     `json:"user"`
	Heatmap    []contract.HeatCell     `json:"heatmap"`
	Solved     SolvedProblemPage       `json:"solved"`
	Activities []contract.UserActivity `json:"activities"`
}

type discussionDetail struct {
	Discussion contract.Discussion `json:"discussion"`
	Content    string              `json:"content"`
	Comments   CommentPage         `json:"comments"`
}

type ProblemListPage contract.Page[ProblemListItem]
type AssignmentListPage contract.Page[AssignmentListItem]
type ContestListPage contract.Page[contract.Contest]
type SubmissionListPage contract.Page[contract.SubmissionListItem]
type RankUserPage contract.Page[RankUser]
type SolvedProblemPage contract.Page[contract.SolvedProblem]
type DiscussionListPage contract.Page[contract.Discussion]
type CommentPage contract.Page[contract.Comment]

type AdminMembers struct {
	Users  []AdminUser  `json:"users"`
	Groups []AdminGroup `json:"groups"`
}

type AdminSettings admin.AdminSettings
type AdminSettingsPatch admin.AdminSettingsPatch
type AdminUser admin.User
type AdminUserPage admin.PageResult[AdminUser]
type AdminUserUpdate admin.UserUpdate
type AdminUserCreate admin.UserCreate
type PasswordReset admin.PasswordReset
type AdminGroup admin.Group
type AdminGroupPage admin.PageResult[AdminGroup]
type AdminGroupUpdate admin.GroupUpdate
type AdminLang admin.Language
type AdminLangUpdate admin.LanguageUpdate
type AdminLangCreate admin.LanguageCreate
type AdminJudger admin.Judger
type AdminJudgeQueue admin.JudgeQueue
type AdminJudgers struct {
	Judgers []AdminJudger   `json:"judgers"`
	Queue   AdminJudgeQueue `json:"queue"`
}

type AdminJudgerUpdate struct {
	Name string `json:"name" maxLength:"64"`
	Auth string `json:"auth,omitempty"`
}

type AdminJudgerCreate struct {
	Name string `json:"name" maxLength:"64"`
}

type BackupSettings backup.Settings
type BackupRunning backup.Running
type BackupItem backup.Item
type BackupList struct {
	Running *BackupRunning `json:"running,omitempty"`
	Items   []BackupItem   `json:"items"`
}

type emptyInput struct{}
type emptyOutput struct{}

type jsonOutput[T any] struct {
	Body T
}

type bodyInput[T any] struct {
	Body T `required:"true"`
}

type idPath struct {
	ID uint `path:"id"`
}

type namePath struct {
	Name string `path:"name"`
}

type adminUserPath struct {
	Name string `path:"name"`
}

type idBody[T any] struct {
	ID   uint `path:"id"`
	Body T    `required:"true"`
}

type stringIDBody[T any] struct {
	ID   string `path:"id"`
	Body T      `required:"true"`
}

type nameBody[T any] struct {
	Name string `path:"name"`
	Body T      `required:"true"`
}

type pageQuery struct {
	Q        string `query:"q"`
	Page     int    `query:"page" minimum:"1"`
	PageSize int    `query:"pageSize" minimum:"1" maximum:"100"`
}

type adminMembersQuery struct {
	Q      string `query:"q"`
	Users  string `query:"users"`
	Groups string `query:"groups"`
}

type problemListQuery struct {
	Q        string `query:"q"`
	Tag      string `query:"tag"`
	Page     int    `query:"page"`
	PageSize int    `query:"pageSize"`
}

type tagQuery struct {
	Kind string `query:"kind" enum:"problem,discussion"`
	Q    string `query:"q"`
}

type problemStateQuery struct {
	IDs        string `query:"ids"`
	Assignment int    `query:"assignment"`
	Contest    int    `query:"contest"`
}

type keyQuery struct {
	ID  uint   `path:"id"`
	Key string `query:"key" required:"true"`
}

type assetContentBody struct {
	ID   uint `path:"id"`
	Body contract.AssetContentUpdate
}

type assignmentListQuery struct {
	Q        string `query:"q"`
	Page     int    `query:"page"`
	PageSize int    `query:"pageSize"`
}

type submissionListQuery struct {
	Problem    string `query:"problem"`
	User       string `query:"user"`
	Language   string `query:"language"`
	Status     string `query:"status"`
	Assignment string `query:"assignment"`
	Contest    string `query:"contest"`
	Page       int    `query:"page"`
	PageSize   int    `query:"pageSize"`
}

type userProfilePath struct {
	Name           string `path:"name"`
	SolvedPage     int    `query:"solvedPage"`
	SolvedPageSize int    `query:"solvedPageSize"`
}

type discussionListQuery struct {
	Q        string `query:"q"`
	Tags     string `query:"tags"`
	Page     int    `query:"page"`
	PageSize int    `query:"pageSize"`
}

type discussionDetailQuery struct {
	ID       uint `path:"id"`
	Page     int  `query:"page"`
	PageSize int  `query:"pageSize"`
}

type commentPath struct {
	ID        uint `path:"id"`
	CommentID uint `path:"commentId"`
}

type userMediaPath struct {
	ID    uint   `path:"id"`
	Year  string `path:"year"`
	Month string `path:"month"`
	Day   string `path:"day"`
	Name  string `path:"name"`
}

type uploadImageInput struct {
	RawBody multipart.Form
}

type uploadProblemAssetInput struct {
	ID      uint `path:"id"`
	RawBody multipart.Form
}

func NewOpenAPI() huma.API {
	arrayNullable := huma.DefaultArrayNullable
	huma.DefaultArrayNullable = false
	defer func() {
		huma.DefaultArrayNullable = arrayNullable
	}()

	registry := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	config := huma.Config{
		OpenAPI: &huma.OpenAPI{
			OpenAPI: "3.1.0",
			Info: &huma.Info{
				Title:   "DOJ Web API",
				Version: "0.1.0",
			},
			Components: &huma.Components{Schemas: registry},
		},
		Formats:       huma.DefaultFormats,
		DefaultFormat: "application/json",
	}
	api := humago.New(http.NewServeMux(), config)
	RegisterOpenAPI(api)
	return api
}

func RegisterOpenAPI(api huma.API) {
	get[Health](api, "/api/health", "health", "Server health")
	get[Health](api, "/api/ready", "ready", "Server dependencies are ready")
	getText(api, "/api/events", "watchEvents", "Server-sent events for lightweight live updates", "text/event-stream")
	get[Site](api, "/api/site", "getSite", "Public site settings")
	get[contract.Me](api, "/api/me", "getMe", "Current user")
	patch[bodyInput[contract.MeUpdate], contract.Me](api, "/api/me", "updateMe", "Updated current user")
	noContent[bodyInput[contract.PasswordUpdate]](api, http.MethodPatch, "/api/me/password", "updatePassword", "Password updated")
	post[bodyInput[contract.LoginRequest], contract.Me](api, "/api/auth/login", "login", "Signed in user")
	postCreated[bodyInput[contract.RegisterRequest], contract.Me](api, "/api/auth/register", "register", "Registered user")
	noContent[emptyInput](api, http.MethodPost, "/api/auth/logout", "logout", "Signed out")
	get[[]contract.Language](api, "/api/languages", "getLangs", "Submit languages")
	postCreated[uploadImageInput, contract.UploadResult](api, "/api/uploads/images", "uploadImage", "Uploaded image")

	get[AdminSettings](api, "/api/admin/settings", "getAdminSettings", "Admin settings")
	patch[bodyInput[AdminSettingsPatch], AdminSettings](api, "/api/admin/settings", "updateAdminSettings", "Updated admin settings")
	getWith[adminMembersQuery, AdminMembers](api, "/api/admin/members", "getAdminMembers", "Admin member options")
	getWith[pageQuery, AdminUserPage](api, "/api/admin/users", "getAdminUsers", "Admin users")
	postCreated[bodyInput[AdminUserCreate], AdminMembers](api, "/api/admin/users", "createAdminUser", "Created user")
	patch[nameBody[AdminUserUpdate], AdminMembers](api, "/api/admin/users/{name}", "updateAdminUser", "Updated user")
	deleteWith[adminUserPath, AdminMembers](api, "/api/admin/users/{name}", "deleteAdminUser", "Deleted user", http.StatusOK)
	post[adminUserPath, PasswordReset](api, "/api/admin/users/{name}/password", "resetAdminUserPassword", "Reset user password")
	getWith[pageQuery, AdminGroupPage](api, "/api/admin/groups", "getAdminGroups", "Admin groups")
	postCreated[bodyInput[AdminGroupUpdate], AdminMembers](api, "/api/admin/groups", "createAdminGroup", "Created group")
	patch[idBody[AdminGroupUpdate], AdminMembers](api, "/api/admin/groups/{id}", "updateAdminGroup", "Updated group")
	deleteWith[idPath, AdminMembers](api, "/api/admin/groups/{id}", "deleteAdminGroup", "Deleted group", http.StatusOK)
	get[[]AdminLang](api, "/api/admin/languages", "getAdminLangs", "Admin languages")
	postCreated[bodyInput[AdminLangCreate], []AdminLang](api, "/api/admin/languages", "createAdminLang", "Created language")
	patch[stringIDBody[AdminLangUpdate], []AdminLang](api, "/api/admin/languages/{id}", "updateAdminLang", "Updated language")
	deleteWith[stringIDPath, []AdminLang](api, "/api/admin/languages/{id}", "deleteAdminLang", "Deleted language", http.StatusOK)
	get[AdminJudgers](api, "/api/admin/judgers", "getAdminJudgers", "Admin judgers")
	postCreated[bodyInput[AdminJudgerCreate], AdminJudgers](api, "/api/admin/judgers", "createAdminJudger", "Created judger")
	patch[idBody[AdminJudgerUpdate], AdminJudgers](api, "/api/admin/judgers/{id}", "updateAdminJudger", "Updated judger")
	deleteWith[idPath, AdminJudgers](api, "/api/admin/judgers/{id}", "deleteAdminJudger", "Deleted judger", http.StatusOK)
	get[BackupSettings](api, "/api/admin/backups/settings", "getBackupSettings", "Backup settings")
	patch[bodyInput[BackupSettings], BackupSettings](api, "/api/admin/backups/settings", "updateBackupSettings", "Updated backup settings")
	get[BackupList](api, "/api/admin/backups", "getBackups", "Backup list")
	postCreated[emptyInput, BackupItem](api, "/api/admin/backups", "createBackup", "Created backup")
	getBinary[namePath](api, "/api/admin/backups/{name}/download", "downloadBackup", "Backup file", "application/gzip")
	noContent[namePath](api, http.MethodDelete, "/api/admin/backups/{name}", "deleteBackup", "Deleted backup")

	get[contract.Home](api, "/api/home", "home", "Home data")
	patch[bodyInput[contract.NoticeUpdate], contract.Home](api, "/api/home/notice", "updateNotice", "Updated notice")
	getBinary[userMediaPath](api, "/api/users/{id}/{year}/{month}/{day}/{name}", "userMedia", "User uploaded media", "application/octet-stream")
	getWith[problemListQuery, ProblemListPage](api, "/api/problems", "listProblems", "Problem list")
	postCreated[bodyInput[contract.ProblemCreate], contract.CreatedID](api, "/api/problems", "createProblem", "Problem created")
	getWith[tagQuery, []string](api, "/api/tags", "listTags", "Tag suggestions")
	getWith[problemStateQuery, []ProblemState](api, "/api/problem-state", "getProblemState", "Problem state for current user and page context")
	getWith[idPath, Problem](api, "/api/problems/{id}", "getProblem", "Problem detail")
	patch[idBody[contract.ProblemUpdate], contract.CreatedID](api, "/api/problems/{id}", "updateProblem", "Problem updated")
	noContent[idPath](api, http.MethodDelete, "/api/problems/{id}", "deleteProblem", "Problem deleted")
	patch[idBody[contract.ProblemVisibilityUpdate], ProblemListItem](api, "/api/problems/{id}/visibility", "updateProblemVisibility", "Problem visibility updated")
	post[idPath, contract.CountResult](api, "/api/problems/{id}/rejudge", "rejudgeProblem", "Problem submissions requeued")
	getWith[idPath, contract.ProblemAssets](api, "/api/problems/{id}/assets", "getProblemAssets", "Problem assets")
	postCreated[uploadProblemAssetInput, contract.UploadResult](api, "/api/problems/{id}/assets/images", "uploadProblemImage", "Problem image uploaded")
	getBinary[nameAssetPath](api, "/api/problems/{id}/assets/{name}", "problemPublicAsset", "Public problem asset", "application/octet-stream")
	getBinary[nameAssetPath](api, "/api/problems/{id}/data/{name}", "problemPrivateData", "Private problem data file", "application/octet-stream")
	getBinary[nameAssetPath](api, "/api/problems/{id}/judge/{name}", "problemPrivateJudge", "Private problem judge file", "application/octet-stream")
	postCreated[uploadProblemAssetInput, contract.ProblemAssets](api, "/api/problems/{id}/assets/files", "uploadProblemAsset", "Problem asset uploaded")
	deleteWith[keyQuery, contract.ProblemAssets](api, "/api/problems/{id}/assets/files", "deleteProblemAsset", "Problem asset deleted", http.StatusOK)
	getWith[keyQuery, contract.AssetContent](api, "/api/problems/{id}/assets/files/content", "getProblemAssetContent", "Text asset content")
	patch[assetContentBody, contract.ProblemAssets](api, "/api/problems/{id}/assets/files/content", "updateProblemAssetContent", "Text asset updated")
	postCreated[idBody[contract.AssetCaseCreate], contract.ProblemAssets](api, "/api/problems/{id}/assets/cases", "createProblemCase", "Problem case created")
	postCreated[idPath, contract.ProblemAssets](api, "/api/problems/{id}/assets/template", "fillJudgeTemplate", "Judge template written")
	getBinary[idPath](api, "/api/problems/{id}/zip", "downloadProblemAssets", "Problem assets zip", "application/zip")
	renamePath(api, "/api/problems/{id}/zip", "/api/problems/{id}.zip")

	getWith[assignmentListQuery, AssignmentListPage](api, "/api/assignments", "listAssignments", "Assignment list")
	postCreated[bodyInput[contract.AssignmentCreate], contract.CreatedID](api, "/api/assignments", "createAssignment", "Assignment created")
	getWith[idPath, assignmentDetail](api, "/api/assignments/{id}", "getAssignment", "Assignment detail")
	patch[idBody[contract.AssignmentUpdate], contract.CreatedID](api, "/api/assignments/{id}", "updateAssignment", "Assignment updated")
	patch[idBody[contract.DescriptionUpdate], contract.DescriptionUpdate](api, "/api/assignments/{id}/description", "updateAssignmentDescription", "Assignment description updated")
	noContent[idPath](api, http.MethodDelete, "/api/assignments/{id}", "deleteAssignment", "Assignment deleted")
	getWith[assignmentListQuery, ContestListPage](api, "/api/contests", "listContests", "Contest list")
	postCreated[bodyInput[contract.ContestCreate], contract.CreatedID](api, "/api/contests", "createContest", "Contest created")
	getWith[idPath, contestDetail](api, "/api/contests/{id}", "getContest", "Contest detail")
	patch[idBody[contract.ContestUpdate], contract.CreatedID](api, "/api/contests/{id}", "updateContest", "Contest updated")
	patch[idBody[contract.DescriptionUpdate], contract.DescriptionUpdate](api, "/api/contests/{id}/description", "updateContestDescription", "Contest description updated")
	noContent[idPath](api, http.MethodDelete, "/api/contests/{id}", "deleteContest", "Contest deleted")

	getWith[submissionListQuery, SubmissionListPage](api, "/api/submissions", "listSubmissions", "Submission list")
	postCreated[bodyInput[contract.SubmitRequest], contract.CreatedID](api, "/api/submissions", "createSubmission", "Submission created")
	getWith[idPath, submissionDetail](api, "/api/submissions/{id}", "getSubmission", "Submission detail")
	patch[idBody[contract.SubmissionUpdate], contract.CreatedID](api, "/api/submissions/{id}", "updateSubmission", "Submission updated")
	post[idPath, contract.CreatedID](api, "/api/submissions/{id}/rejudge", "rejudgeSubmission", "Submission requeued")
	getWith[pageQuery, RankUserPage](api, "/api/rank", "getRank", "Rank list")
	getWith[struct {
		Q string `query:"q"`
	}, []contract.UserOption](api, "/api/users", "searchUsers", "User suggestions")
	getWith[userProfilePath, userProfile](api, "/api/users/{name}", "getUser", "User profile")
	getWith[discussionListQuery, DiscussionListPage](api, "/api/discussion", "listDiscussions", "Discussion list")
	postCreated[bodyInput[contract.DiscussionCreate], contract.CreatedID](api, "/api/discussion", "createDiscussion", "Discussion created")
	getWith[discussionDetailQuery, discussionDetail](api, "/api/discussion/{id}", "getDiscussion", "Discussion detail")
	patch[idBody[contract.DiscussionUpdate], contract.CreatedID](api, "/api/discussion/{id}", "updateDiscussion", "Discussion updated")
	noContent[idPath](api, http.MethodDelete, "/api/discussion/{id}", "deleteDiscussion", "Discussion deleted")
	postCreated[idBody[contract.CommentCreate], contract.Comment](api, "/api/discussion/{id}/comments", "createComment", "Comment created")
	noContent[commentPath](api, http.MethodDelete, "/api/discussion/{id}/comments/{commentId}", "deleteComment", "Comment deleted")
}

type stringIDPath struct {
	ID string `path:"id"`
}

type nameAssetPath struct {
	ID   uint   `path:"id"`
	Name string `path:"name"`
}

func get[O any](api huma.API, path string, id string, summary string) {
	getWith[emptyInput, O](api, path, id, summary)
}

func getWith[I any, O any](api huma.API, path string, id string, summary string) {
	register[I, jsonOutput[O]](api, http.MethodGet, path, id, summary, http.StatusOK)
}

func post[I any, O any](api huma.API, path string, id string, summary string) {
	register[I, jsonOutput[O]](api, http.MethodPost, path, id, summary, http.StatusOK)
}

func postCreated[I any, O any](api huma.API, path string, id string, summary string) {
	register[I, jsonOutput[O]](api, http.MethodPost, path, id, summary, http.StatusCreated)
}

func patch[I any, O any](api huma.API, path string, id string, summary string) {
	register[I, jsonOutput[O]](api, http.MethodPatch, path, id, summary, http.StatusOK)
}

func deleteWith[I any, O any](api huma.API, path string, id string, summary string, status int) {
	register[I, jsonOutput[O]](api, http.MethodDelete, path, id, summary, status)
}

func noContent[I any](api huma.API, method string, path string, id string, summary string) {
	register[I, emptyOutput](api, method, path, id, summary, http.StatusNoContent)
}

func getText(api huma.API, path string, id string, summary string, contentType string) {
	register[emptyInput, struct {
		Body string `contentType:"text/event-stream"`
	}](api, http.MethodGet, path, id, summary, http.StatusOK)
	op := api.OpenAPI().Paths[path].Get
	response := op.Responses["200"]
	response.Content = map[string]*huma.MediaType{
		contentType: {Schema: &huma.Schema{Type: "string"}},
	}
}

func getBinary[I any](api huma.API, path string, id string, summary string, contentType string) {
	register[I, struct {
		Body []byte `contentType:"application/octet-stream"`
	}](api, http.MethodGet, path, id, summary, http.StatusOK)
	op := api.OpenAPI().Paths[path].Get
	response := op.Responses["200"]
	response.Content = map[string]*huma.MediaType{
		contentType: {Schema: &huma.Schema{Type: "string", Format: "binary"}},
	}
}

func register[I any, O any](api huma.API, method string, path string, id string, summary string, status int) {
	huma.Register(api, huma.Operation{
		OperationID:   id,
		Method:        method,
		Path:          path,
		Summary:       summary,
		DefaultStatus: status,
		Errors:        []int{},
	}, func(context.Context, *I) (*O, error) {
		return new(O), nil
	})
}

func renamePath(api huma.API, from string, to string) {
	api.OpenAPI().Paths[to] = api.OpenAPI().Paths[from]
	delete(api.OpenAPI().Paths, from)
}
