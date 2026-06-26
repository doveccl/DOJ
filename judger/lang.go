package judger

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type preparedLang struct {
	Image   string
	Command string
	Cleanup func()
}

const defaultLanguageBuildTimeout = 2 * time.Minute

var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)

type languageBuildError struct {
	Message string
}

func (err languageBuildError) Error() string {
	return err.Message
}

func prepareLanguageImage(ctx context.Context, work string, lang Lang, source string, limits Limits, submissionID uint, attempt int, logf func(format string, args ...any)) (preparedLang, error) {
	if strings.TrimSpace(lang.Dockerfile) == "" {
		return preparedLang{}, fmt.Errorf("language Dockerfile is required")
	}
	if strings.TrimSpace(lang.Source) == "" {
		return preparedLang{}, fmt.Errorf("language source file is required")
	}
	command, err := dockerfileCommand(lang.Dockerfile)
	if err != nil {
		return preparedLang{}, err
	}
	contextStartedAt := time.Now()
	dir, err := os.MkdirTemp(work, "lang-build-")
	if err != nil {
		return preparedLang{}, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	if err := writeLanguageBuildContext(dir, lang, source); err != nil {
		cleanup()
		return preparedLang{}, err
	}
	logStep(logf, submissionID, attempt, "language_build_context", contextStartedAt)
	buildCtx, cancel := context.WithTimeout(ctx, defaultLanguageBuildTimeout)
	defer cancel()
	outputLimit := int64(defaultCompileOutputLimit)
	if limits.OutputKB > 0 {
		outputLimit = int64(limits.OutputKB) * 1024
	}
	image, out, err := dockerBuildImageTimed(buildCtx, dir, "Dockerfile", outputLimit, dockerBuildTiming{
		Logf:         logf,
		SubmissionID: submissionID,
		Attempt:      attempt,
	})
	if err != nil {
		cleanup()
		message := cleanBuildMessage(out)
		if message == "" {
			message = err.Error()
			return preparedLang{}, fmt.Errorf("build language image: %s", message)
		}
		return preparedLang{}, languageBuildError{Message: "build language image: " + message}
	}
	return preparedLang{
		Image:   image,
		Command: command,
		Cleanup: func() {
			removeStartedAt := time.Now()
			dockerRemoveImage(context.Background(), image)
			logStep(logf, submissionID, attempt, "remove_language_image", removeStartedAt)
			cleanup()
		},
	}, nil
}

func writeLanguageBuildContext(dir string, lang Lang, source string) error {
	sourceName := filepath.Clean(lang.Source)
	if sourceName == "." || filepath.IsAbs(sourceName) || sourceName == ".." || strings.HasPrefix(sourceName, ".."+string(filepath.Separator)) {
		return fmt.Errorf("unsafe language source path %q", lang.Source)
	}
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(lang.Dockerfile), 0o600); err != nil {
		return err
	}
	rootSource := filepath.Join(dir, sourceName)
	if err := os.MkdirAll(filepath.Dir(rootSource), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(rootSource, []byte(source), 0o644); err != nil {
		return err
	}
	srcSource := filepath.Join(dir, "src", sourceName)
	if srcSource != rootSource {
		if err := os.MkdirAll(filepath.Dir(srcSource), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(srcSource, []byte(source), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func cleanBuildMessage(message string) string {
	return strings.TrimSpace(ansiEscapePattern.ReplaceAllString(message, ""))
}

func dockerfileCommand(dockerfile string) (string, error) {
	for _, line := range reverseLines(dockerfile) {
		raw := strings.TrimSpace(line)
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		upper := strings.ToUpper(raw)
		if !strings.HasPrefix(upper, "CMD") || len(raw) == 3 {
			continue
		}
		value := strings.TrimSpace(raw[3:])
		if value == "" {
			continue
		}
		if strings.HasPrefix(value, "[") {
			var parts []string
			if err := json.Unmarshal([]byte(value), &parts); err != nil {
				return "", fmt.Errorf("invalid Dockerfile CMD: %w", err)
			}
			quoted := make([]string, 0, len(parts))
			for _, part := range parts {
				quoted = append(quoted, shellQuote(part))
			}
			if len(quoted) == 0 {
				break
			}
			return strings.Join(quoted, " "), nil
		}
		return value, nil
	}
	return "", fmt.Errorf("language Dockerfile requires a CMD")
}

func reverseLines(value string) []string {
	lines := strings.Split(value, "\n")
	for left, right := 0, len(lines)-1; left < right; left, right = left+1, right-1 {
		lines[left], lines[right] = lines[right], lines[left]
	}
	return lines
}
