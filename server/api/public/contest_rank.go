package public

import (
	"sort"
	"time"

	contract "github.com/doveccl/doj/contract/web"

	"github.com/doveccl/doj/models"
)

func (api *API) contestRank(contest models.Contest, problems []contract.Problem, until *time.Time) ([]contract.RankUser, error) {
	var rows []models.Submission
	query := api.db.
		Joins("JOIN users ON users.id = submissions.user_id AND users.deleted_at IS NULL").
		Where("submissions.contest_id = ?", contest.ID).
		Order("submissions.created_at asc")
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	users, err := api.rankUsers(rows)
	if err != nil {
		return nil, err
	}
	if contest.Kind == "ICPC" {
		return icpcRank(contest, rows, users, problems, until), nil
	}
	return oiRank(rows, users, problems), nil
}

func (api *API) rankUsers(submissions []models.Submission) (map[uint]models.User, error) {
	ids := map[uint]bool{}
	for _, row := range submissions {
		ids[row.UserID] = true
	}
	if len(ids) == 0 {
		return map[uint]models.User{}, nil
	}
	values := make([]uint, 0, len(ids))
	for id := range ids {
		values = append(values, id)
	}
	var rows []models.User
	if err := api.db.Where("id IN ?", values).Find(&rows).Error; err != nil {
		return nil, err
	}
	users := make(map[uint]models.User, len(rows))
	for _, row := range rows {
		users[row.ID] = row
	}
	return users, nil
}

func oiRank(submissions []models.Submission, users map[uint]models.User, problems []contract.Problem) []contract.RankUser {
	type state struct {
		user    models.User
		submit  int
		score   map[uint]int
		attempt map[uint]int
		done    map[uint]int
		pending map[uint]int
	}
	problemSet := rankProblemSet(problems)
	states := map[uint]*state{}
	for _, row := range submissions {
		if !problemSet[row.ProblemID] {
			continue
		}
		user, ok := users[row.UserID]
		if !ok {
			continue
		}
		got := states[row.UserID]
		if got == nil {
			got = &state{user: user, score: map[uint]int{}, attempt: map[uint]int{}, done: map[uint]int{}, pending: map[uint]int{}}
			states[row.UserID] = got
		}
		got.submit++
		got.attempt[row.ProblemID]++
		if submissionLive(row.Status) {
			got.pending[row.ProblemID]++
			continue
		}
		if got.done[row.ProblemID] == 0 || row.Score > got.score[row.ProblemID] {
			got.score[row.ProblemID] = row.Score
		}
		got.done[row.ProblemID]++
	}
	items := make([]contract.RankUser, 0, len(states))
	for _, got := range states {
		score := 0
		ac := 0
		problemItems := make([]contract.RankProblem, 0, len(problems))
		for _, problem := range problems {
			value := got.score[problem.ID]
			submit := got.attempt[problem.ID]
			status := "none"
			if got.done[problem.ID] == 0 && got.pending[problem.ID] > 0 {
				status = "pending"
			} else if submit > 0 {
				status = "tried"
			}
			if value >= 100 {
				status = "ac"
				ac++
			}
			score += value
			problemItems = append(problemItems, contract.RankProblem{ProblemID: problem.ID, Status: status, Submit: submit, Score: value})
		}
		items = append(items, contract.RankUser{User: got.user.Name, Bio: got.user.Bio, Avatar: got.user.Avatar, AC: ac, Submit: got.submit, Score: score, Problems: problemItems})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		return items[i].User < items[j].User
	})
	for index := range items {
		items[index].Rank = index + 1
	}
	return items
}

func icpcRank(contest models.Contest, submissions []models.Submission, users map[uint]models.User, problems []contract.Problem, freezeAt *time.Time) []contract.RankUser {
	type problemState struct {
		wrong   int
		submit  int
		pending int
		solved  bool
		penalty int
	}
	type state struct {
		user     models.User
		submit   int
		problems map[uint]*problemState
	}
	problemSet := rankProblemSet(problems)
	states := map[uint]*state{}
	for _, row := range submissions {
		if !problemSet[row.ProblemID] {
			continue
		}
		user, ok := users[row.UserID]
		if !ok {
			continue
		}
		got := states[row.UserID]
		if got == nil {
			got = &state{user: user, problems: map[uint]*problemState{}}
			states[row.UserID] = got
		}
		problem := got.problems[row.ProblemID]
		if problem == nil {
			problem = &problemState{}
			got.problems[row.ProblemID] = problem
		}
		if freezeAt != nil && !row.CreatedAt.Before(*freezeAt) {
			problem.pending++
			continue
		}
		if submissionLive(row.Status) {
			problem.pending++
			continue
		}
		got.submit++
		problem.submit++
		if problem.solved {
			continue
		}
		if row.Status == "AC" {
			problem.solved = true
			minutes := int(row.CreatedAt.Sub(contest.StartAt).Minutes())
			if minutes < 0 {
				minutes = 0
			}
			problem.penalty = minutes + problem.wrong*20
			continue
		}
		if penalizable(row.Status) {
			problem.wrong++
		}
	}
	items := make([]contract.RankUser, 0, len(states))
	for _, got := range states {
		ac := 0
		penalty := 0
		problemItems := make([]contract.RankProblem, 0, len(problems))
		for _, contestProblem := range problems {
			problem := got.problems[contestProblem.ID]
			status := "none"
			submit := 0
			problemPenalty := 0
			if problem != nil {
				submit = problem.submit
				if problem.solved {
					status = "ac"
					submit = problem.wrong + 1
					problemPenalty = problem.penalty
					ac++
					penalty += problem.penalty
				} else if problem.pending > 0 {
					status = "pending"
					submit = problem.pending
				} else if problem.submit > 0 {
					status = "tried"
				}
			}
			problemItems = append(problemItems, contract.RankProblem{ProblemID: contestProblem.ID, Status: status, Submit: submit, Score: boolScore(status == "ac"), Penalty: problemPenalty})
		}
		items = append(items, contract.RankUser{User: got.user.Name, Bio: got.user.Bio, Avatar: got.user.Avatar, AC: ac, Submit: got.submit, Score: ac, Penalty: penalty, Problems: problemItems})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].AC != items[j].AC {
			return items[i].AC > items[j].AC
		}
		if items[i].Penalty != items[j].Penalty {
			return items[i].Penalty < items[j].Penalty
		}
		if items[i].Submit != items[j].Submit {
			return items[i].Submit < items[j].Submit
		}
		return items[i].User < items[j].User
	})
	for index := range items {
		items[index].Rank = index + 1
	}
	return items
}

func rankProblemSet(problems []contract.Problem) map[uint]bool {
	items := make(map[uint]bool, len(problems))
	for _, problem := range problems {
		items[problem.ID] = true
	}
	return items
}

func boolScore(value bool) int {
	if value {
		return 1
	}
	return 0
}

func penalizable(status string) bool {
	switch status {
	case "AC", "CE", "SE":
		return false
	default:
		return true
	}
}
