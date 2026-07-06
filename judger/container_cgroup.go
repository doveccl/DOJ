package judger

import "time"

func (client runnerClient) prepareUserCgroup(pid *UserPID) (*CgroupCase, error) {
	hostPID, err := MapInnerPID(client.procRoot, client.initPID, pid.PID)
	if err != nil {
		return nil, err
	}
	cg, err := PrepareCgroup(CgroupConfig{
		Root:         client.cgroupRoot,
		SubmissionID: client.taskID,
		CaseID:       safeCaseID(pid.CaseID),
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

func watchCgroupMemoryLimit(cgroup *CgroupCase) func() {
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
				if err == nil && cgroupMemoryLimitReached(stats) {
					_ = cgroup.killAll()
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

func applyCgroupStats(result *CaseResult, cgroup *CgroupCase) {
	stats, err := cgroup.Stats()
	if err != nil {
		return
	}
	applyCgroupStatsSnapshot(result, stats)
}

func applyCgroupStatsSnapshot(result *CaseResult, stats CgroupStats) {
	if stats.MemoryPeak > 0 {
		result.MemoryKB = int((stats.MemoryPeak + 1023) / 1024)
	}
	if cgroupMemoryLimitReached(stats) {
		result.Verdict = VerdictMemoryLimit
		result.Score = 0
		result.Message = ""
		return
	}
	if stats.PidsMaxed && result.Verdict == VerdictAccepted {
		result.Verdict = VerdictRuntimeError
		result.Score = 0
		result.Message = "process limit exceeded"
	}
}
