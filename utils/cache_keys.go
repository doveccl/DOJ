package utils

import "strconv"

func ProblemPackageCacheKey(problemID uint) string {
	return "doj:problem:" + strconv.FormatUint(uint64(problemID), 10) + ":package"
}
