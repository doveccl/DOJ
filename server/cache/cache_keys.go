package cache

import "strconv"

func ProblemPackageKey(problemID uint) string {
	return "doj:problem:" + strconv.FormatUint(uint64(problemID), 10) + ":package"
}
