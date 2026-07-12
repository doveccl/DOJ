package runner

import (
	"regexp"
	"strings"
)

const languageSourceDir = "src"

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)

func CleanBuildMessage(message string) string {
	return strings.TrimSpace(ansiEscapePattern.ReplaceAllString(message, ""))
}
