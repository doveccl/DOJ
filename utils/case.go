package utils

import (
	"path"
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
	case strings.HasSuffix(lower, ".out"):
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
