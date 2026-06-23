package judger

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type preparedLang struct {
	Image   string
	Command string
	Cleanup func()
}

func prepareLanguageImage(ctx context.Context, work string, lang Lang, source string, limits Limits) (preparedLang, error) {
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
	dir, err := os.MkdirTemp(work, "lang-build-")
	if err != nil {
		return preparedLang{}, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	if err := writeLanguageBuildContext(dir, lang, source); err != nil {
		cleanup()
		return preparedLang{}, err
	}
	iidFile := filepath.Join(dir, "image.iid")
	timeout := 30 * time.Second
	if limits.TimeMS > 0 {
		timeout = time.Duration(limits.TimeMS) * time.Millisecond
	}
	buildCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	outputLimit := int64(defaultCompileOutputLimit)
	if limits.OutputKB > 0 {
		outputLimit = int64(limits.OutputKB) * 1024
	}
	out, err := runDockerStep(buildCtx, dir, outputLimit, "build", "--iidfile", iidFile, ".")
	if err != nil {
		cleanup()
		message := strings.TrimSpace(out)
		if message == "" {
			message = err.Error()
		}
		return preparedLang{}, fmt.Errorf("build language image: %s", message)
	}
	imageID, err := os.ReadFile(iidFile)
	if err != nil {
		cleanup()
		return preparedLang{}, fmt.Errorf("read language image id: %w", err)
	}
	image := strings.TrimSpace(string(imageID))
	if image == "" {
		cleanup()
		return preparedLang{}, fmt.Errorf("language image build produced an empty image id")
	}
	return preparedLang{
		Image:   image,
		Command: command,
		Cleanup: func() {
			dockerCleanup("rmi", "-f", image)
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
