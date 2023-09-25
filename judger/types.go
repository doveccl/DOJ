package judger

import (
	"encoding/json"
	"fmt"
)

type Status int

const (
	AC Status = iota
	WA
	PE
	TLE
	MLE
	OLE
	CE
	RE
	SE
)

type Config struct {
	Points      int
	TimeLimit   int64 // ms
	MemoryLimit int64 // mbytes
}

type Result struct {
	Status Status
	Time   int64
	Detail string
}

func (s Status) String() string {
	return []string{"AC", "WA", "PE", "TLE", "MLE", "OLE", "CE", "RE", "SE"}[s]
}

func (r Result) String() string {
	if r.Detail != "" {
		b, _ := json.Marshal(r.Detail)
		return fmt.Sprintf("%v(%vms %s)", r.Status, r.Time, b)
	}
	return fmt.Sprintf("%v(%vms)", r.Status, r.Time)
}
