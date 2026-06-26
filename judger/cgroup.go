package judger

import "errors"

var ErrCgroupUnsupported = errors.New("doj judger requires linux cgroup v2")

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
	MemoryMaxed  bool
	CPUUsageUSec int64
	PidsCurrent  int64
	PidsMaxed    bool
}

type CgroupCase struct {
	Path string
}

func cgroupMemoryLimitReached(stats CgroupStats) bool {
	return stats.MemoryOOM || stats.MemoryMaxed
}
