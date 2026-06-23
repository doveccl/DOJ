package judger

import "errors"

var ErrCgroupUnsupported = errors.New("cgroup v2 is only available on linux")

type CgroupConfig struct {
	Root         string
	SubmissionID string
	CaseID       string
	MemoryMax    int64
	PidsMax      int
}

type CgroupStats struct {
	MemoryPeak   int64
	MemoryOOM    bool
	CPUUsageUSec int64
	PidsCurrent  int64
	PidsMaxed    bool
}

type CgroupCase struct {
	Path string
}
