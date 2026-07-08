package runner

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
