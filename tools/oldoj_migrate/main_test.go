package main

import (
	"archive/zip"
	"bytes"
	"testing"
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
