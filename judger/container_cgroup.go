package judger

import (
	"time"

	"github.com/doveccl/doj/judger/runner"
)

func (client runnerClient) prepareUserCgroup(pid *runner.UserPID) (*runner.CgroupCase, error) {
	hostPID, err := MapInnerPID(client.procRoot, client.initPID, pid.PID)
	if err != nil {
		return nil, err
	}
	cg, err := runner.PrepareCgroup(runner.CgroupConfig{
		Root:         client.cgroupRoot,
		SubmissionID: client.taskID,
		CaseID:       runner.SafeCaseID(pid.CaseID),
		MemoryMax:    int64(client.limits.MemoryKB) * 1024,
		PidsMax:      client.limits.Pids,
	})
	if err != nil {
		return nil, err
	}
	if err := cg.Add(hostPID); err != nil {
		_ = cg.Cleanup()
		return nil, err
	}
	return cg, nil
}

func watchCgroupMemoryLimit(cgroup *runner.CgroupCase) func() {
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				stats, err := cgroup.Stats()
				if err == nil && runner.CgroupMemoryLimitReached(stats) {
					_ = cgroup.KillAll()
					return
				}
			}
		}
	}()
	return func() {
		close(stop)
		<-done
	}
}

func applyCgroupStats(result *runner.CaseResult, cgroup *runner.CgroupCase) {
	stats, err := cgroup.Stats()
	if err != nil {
		return
	}
	applyCgroupStatsSnapshot(result, stats)
}

func applyCgroupStatsSnapshot(result *runner.CaseResult, stats runner.CgroupStats) {
	if timeMS := cgroupCPUTimeMS(stats); timeMS > 0 && result.Verdict != runner.VerdictTimeLimit {
		result.TimeMS = timeMS
	}
	if stats.MemoryPeak > 0 {
		result.MemoryKB = int((stats.MemoryPeak + 1023) / 1024)
	}
	if runner.CgroupMemoryLimitReached(stats) {
		result.Verdict = runner.VerdictMemoryLimit
		result.Score = 0
		result.Message = ""
		return
	}
	if stats.PidsMaxed && result.Verdict == runner.VerdictAccepted {
		result.Verdict = runner.VerdictRuntimeError
		result.Score = 0
		result.Message = "process limit exceeded"
	}
}

func cgroupCPUTimeMS(stats runner.CgroupStats) int {
	if stats.CPUUsageUSec <= 0 {
		return 0
	}
	return int((stats.CPUUsageUSec + 999) / 1000)
}
