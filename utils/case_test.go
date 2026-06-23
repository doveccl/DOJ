package utils

import "testing"

func TestDataCaseStem(t *testing.T) {
	tests := []struct {
		name string
		stem string
		kind string
	}{
		{name: "1.in", stem: "1", kind: "in"},
		{name: "1.out", stem: "1", kind: "out"},
		{name: "sample.in", stem: "sample", kind: "in"},
		{name: "sample.out", stem: "sample", kind: "out"},
		{name: "input1.txt", stem: "1", kind: "in"},
		{name: "answer1.txt", stem: "1", kind: "out"},
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
