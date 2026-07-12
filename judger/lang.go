package judger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/doveccl/doj/judger/runner"
)

type preparedLang struct {
	Image          string
	CompileCommand string
	RunCommand     string
	SourceName     string
}

const languageSourceDir = "src"

func prepareLanguageRuntime(work string, lang runner.Lang, source string) (preparedLang, error) {
	if strings.TrimSpace(lang.Image) == "" {
		return preparedLang{}, fmt.Errorf("language image is required")
	}
	if strings.TrimSpace(lang.Source) == "" {
		return preparedLang{}, fmt.Errorf("language source file is required")
	}
	if strings.TrimSpace(lang.Run) == "" {
		return preparedLang{}, fmt.Errorf("language run command is required")
	}
	sourceName, err := cleanLanguageSource(lang.Source)
	if err != nil {
		return preparedLang{}, err
	}
	sourceRoot := filepath.Join(work, languageSourceDir)
	if err := os.RemoveAll(sourceRoot); err != nil {
		return preparedLang{}, err
	}
	sourcePath := filepath.Join(sourceRoot, sourceName)
	if err := privateDir(filepath.Dir(sourcePath)); err != nil {
		return preparedLang{}, err
	}
	if err := os.WriteFile(sourcePath, []byte(source), 0o600); err != nil {
		return preparedLang{}, err
	}
	return preparedLang{
		Image:          strings.TrimSpace(lang.Image),
		CompileCommand: strings.TrimSpace(lang.Compile),
		RunCommand:     strings.TrimSpace(lang.Run),
		SourceName:     sourceName,
	}, nil
}

func cleanLanguageSource(source string) (string, error) {
	sourceName := filepath.Clean(strings.TrimSpace(source))
	if sourceName == "." || filepath.IsAbs(sourceName) || sourceName == ".." || sourceName != filepath.Base(sourceName) {
		return "", fmt.Errorf("unsafe language source path %q", source)
	}
	return sourceName, nil
}
