package cases

import (
	"slices"
	"sort"
	"testing"
)

func TestDataCaseStem(t *testing.T) {
	tests := []struct {
		name string
		stem string
		kind string
	}{
		{name: "1.in", stem: "1", kind: "in"},
		{name: "1.out", stem: "1", kind: "out"},
		{name: "1.ans", stem: "1", kind: "out"},
		{name: "sample.in", stem: "sample", kind: "in"},
		{name: "sample.out", stem: "sample", kind: "out"},
		{name: "sample.ans", stem: "sample", kind: "out"},
		{name: "input1.txt", stem: "1", kind: "in"},
		{name: "answer1.txt", stem: "1", kind: "out"},
		{name: "ans1.txt", stem: "1", kind: "out"},
		{name: "output02.txt", stem: "02", kind: "out"},
		{name: "case-03-input.txt", stem: "03", kind: "in"},
		{name: "readme.txt"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stem, kind := DataCaseStem(test.name)
			if stem != test.stem || kind != test.kind {
				t.Fatalf("DataCaseStem(%q) = %q, %q; want %q, %q", test.name, stem, kind, test.stem, test.kind)
			}
		})
	}
}

func TestDataCaseFileLess(t *testing.T) {
	files := []string{"10.out", "2.out", "1.out", "10.in", "readme.txt", "2.in", "1.in", "3.ans", "3.in"}
	sort.Slice(files, func(i, j int) bool { return DataCaseFileLess(files[i], files[j]) })
	want := []string{"1.in", "1.out", "2.in", "2.out", "3.in", "3.ans", "10.in", "10.out", "readme.txt"}
	if !slices.Equal(files, want) {
		t.Fatalf("files = %+v, want %+v", files, want)
	}
}
