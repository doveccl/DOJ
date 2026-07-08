package cases

import (
	"path"
	"strconv"
	"strings"
)

func DataCaseStem(name string) (string, string) {
	base := path.Base(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
	lower := strings.ToLower(base)
	if stem := firstDigitRun(base); stem != "" {
		switch {
		case strings.Contains(lower, "in"):
			return stem, "in"
		case strings.Contains(lower, "out") || strings.Contains(lower, "ans"):
			return stem, "out"
		}
	}
	switch {
	case strings.HasSuffix(lower, ".in"):
		stem := base[:len(base)-3]
		if stem != "" {
			return stem, "in"
		}
	case strings.HasSuffix(lower, ".out"), strings.HasSuffix(lower, ".ans"):
		stem := base[:len(base)-4]
		if stem != "" {
			return stem, "out"
		}
	}
	return "", ""
}

func firstDigitRun(value string) string {
	start := -1
	for index, char := range value {
		if char >= '0' && char <= '9' {
			if start < 0 {
				start = index
			}
			continue
		}
		if start >= 0 {
			return value[start:index]
		}
	}
	if start >= 0 {
		return value[start:]
	}
	return ""
}

func CaseStemLess(a string, b string) bool {
	aInt, aErr := strconv.Atoi(a)
	bInt, bErr := strconv.Atoi(b)
	if aErr == nil && bErr == nil && aInt != bInt {
		return aInt < bInt
	}
	return a < b
}

func DataCaseFileLess(a string, b string) bool {
	aStem, aKind := DataCaseStem(a)
	bStem, bKind := DataCaseStem(b)
	if aKind != "" && bKind != "" {
		if aStem != bStem {
			return CaseStemLess(aStem, bStem)
		}
		if aKind != bKind {
			return aKind == "in"
		}
	}
	return a < b
}
