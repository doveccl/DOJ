package public

import (
	"testing"
	"time"

	contract "github.com/doveccl/doj/contract/web"
	"github.com/doveccl/doj/models"
)

func TestOIRankKeepsLastCompletedScoreWhileJudging(t *testing.T) {
	users := map[uint]models.User{
		1: {ID: 1, Name: "alice"},
		2: {ID: 2, Name: "bob"},
	}
	rows := []models.Submission{
		{UserID: 1, ProblemID: 1000, Status: "WA", Score: 60},
		{UserID: 1, ProblemID: 1000, Status: "judging"},
		{UserID: 2, ProblemID: 1000, Status: "queued"},
	}
	rank := oiRank(rows, users, []contract.Problem{{ID: 1000}})
	if len(rank) != 2 || rank[0].User != "alice" || rank[0].Score != 60 || rank[0].Problems[0].Status != "tried" {
		t.Fatalf("completed score was replaced by a live submission: %+v", rank)
	}
	if rank[1].User != "bob" || rank[1].Score != 0 || rank[1].Problems[0].Status != "pending" {
		t.Fatalf("live-only participant should be pending: %+v", rank)
	}
}

func TestICPCRankDoesNotPenalizeLiveSubmission(t *testing.T) {
	now := time.Now()
	contest := models.Contest{Kind: "ICPC", StartAt: now.Add(-time.Hour), EndAt: now.Add(time.Hour)}
	users := map[uint]models.User{1: {ID: 1, Name: "alice"}}
	rows := []models.Submission{{UserID: 1, ProblemID: 1000, Status: "judging", CreatedAt: now}}
	rank := icpcRank(contest, rows, users, []contract.Problem{{ID: 1000}}, nil)
	if len(rank) != 1 || rank[0].AC != 0 || rank[0].Penalty != 0 || rank[0].Submit != 0 || rank[0].Problems[0].Status != "pending" {
		t.Fatalf("live ICPC submission was counted as a wrong attempt: %+v", rank)
	}
}
