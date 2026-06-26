package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/doveccl/doj/utils"
)

func TestBuildPlanSkipsExistingDataAndMarksEmptyTargetsForCleanup(t *testing.T) {
	oldRows := []oldProblem{
		{ID: 2349, Title: "P2349 - 军训站队", Data: "data-a"},
		{ID: 2348, Title: "P2348 - 魔数", Data: "data-b"},
		{ID: 2347, Title: "P2347 - Reading", Data: ""},
		{ID: 2346, Title: "P2346 - New Real", Data: "data-c"},
	}
	targetRows := map[uint]targetProblem{
		1200: {ID: 1200, Title: "阅读材料"},
		2348: {ID: 2348, Title: "魔数"},
	}
	targetCases := map[uint]int{
		1200: 0,
		2348: 2,
	}

	plan := buildPlan(oldRows, targetRows, targetCases, 100)
	if len(plan.Candidates) != 2 {
		t.Fatalf("candidates = %+v", plan.Candidates)
	}
	if plan.Candidates[0].ID != 2349 || plan.Candidates[1].ID != 2346 {
		t.Fatalf("candidate order = %+v", plan.Candidates)
	}
	if len(plan.Skip) != 1 || plan.Skip[0].ID != 2348 {
		t.Fatalf("skip = %+v", plan.Skip)
	}
	if len(plan.Cleanup) != 1 || plan.Cleanup[0].ID != 1200 {
		t.Fatalf("cleanup = %+v", plan.Cleanup)
	}
}

func TestZipDataFilesCountsCasePairs(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		"0.in":         "1 2\n",
		"0.out":        "3\n",
		"nested/1.in":  "4 5\n",
		"nested/1.out": "9\n",
		"readme.txt":   "ignored",
	} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	files, cases, err := zipDataFiles(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if cases != 2 {
		t.Fatalf("cases = %d files=%v", cases, files)
	}
	if string(files["0.out"]) != "3\n" || string(files["nested/1.in"]) != "4 5\n" {
		t.Fatalf("files = %+v", files)
	}
}

func TestNormalizeTitleIgnoresOldProblemPrefix(t *testing.T) {
	if normalizeTitle("P2349 - 军训站队") != normalizeTitle("军训站队") {
		t.Fatal("problem prefix should not affect title matching")
	}
}

func TestShellCommandQuotesArguments(t *testing.T) {
	got := shellCommand([]string{"psql", "-c", "select 'a b';"})
	want := "'psql' '-c' 'select '\"'\"'a b'\"'\"';'"
	if got != want {
		t.Fatalf("shell command = %q, want %q", got, want)
	}
}

func TestVerifyTargetProblemsCountsDataAndEmptyProblems(t *testing.T) {
	store := memoryStore{objects: map[string]string{
		"problems/1000/data/0.in":       "1 2\n",
		"problems/1000/data/0.out":      "3\n",
		"problems/1001/data/readme.txt": "ignored",
	}}
	targetCases, err := storedCasesByProblem(context.Background(), store)
	if err != nil {
		t.Fatal(err)
	}
	got := verifyTargetProblems(map[uint]targetProblem{
		1000: {ID: 1000, Title: "A+B"},
		1001: {ID: 1001, Title: "Reading"},
	}, targetCases)
	if got.Active != 2 || got.WithData != 1 || got.WithoutData != 1 {
		t.Fatalf("report = %+v", got)
	}
	if len(got.Empty) != 1 || got.Empty[0].ID != 1001 {
		t.Fatalf("empty = %+v", got.Empty)
	}
}

type memoryStore struct {
	objects map[string]string
}

func (store memoryStore) Put(context.Context, string, io.Reader, int64, string) error {
	return fmt.Errorf("not implemented")
}

func (store memoryStore) Open(context.Context, string) (io.ReadCloser, string, error) {
	return nil, "", fmt.Errorf("not implemented")
}

func (store memoryStore) List(_ context.Context, prefix string) ([]utils.ObjectInfo, error) {
	var items []utils.ObjectInfo
	for key, body := range store.objects {
		if strings.HasPrefix(key, prefix) {
			items = append(items, utils.ObjectInfo{Key: key, Size: int64(len(body))})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Key < items[j].Key })
	return items, nil
}

func (store memoryStore) Delete(context.Context, string) error {
	return fmt.Errorf("not implemented")
}
