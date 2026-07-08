package runner

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
)

const defaultOutputLimit = 64 << 20

var errOutputLimit = errors.New("output limit exceeded")

func BuiltinJudgeMain(args []string) int {
	mode := ModeDefault
	if len(args) > 0 && strings.HasPrefix(args[0], "--mode=") {
		mode = JudgeMode(strings.TrimPrefix(args[0], "--mode="))
		args = args[1:]
	}
	if len(args) != 3 && len(args) != 4 {
		fmt.Fprintln(os.Stderr, "judge requires [--mode=default|strict] <input-file> <output-file> <answer-file> [<result-file>]")
		return 3
	}
	result := ""
	if len(args) == 4 {
		result = args[3]
	}
	exitCode, err := RunBuiltinChecker(mode, args[0], args[1], args[2], result)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 3
	}
	return exitCode
}

func RunBuiltinChecker(mode JudgeMode, inputPath string, outputPath string, answerPath string, resultPath string) (int, error) {
	if _, err := os.Stat(inputPath); err != nil {
		_ = writeResultMessage(resultPath, err.Error())
		return 3, err
	}
	answerBytes, err := os.ReadFile(answerPath)
	if err != nil {
		_ = writeResultMessage(resultPath, err.Error())
		return 3, err
	}
	output, err := os.ReadFile(outputPath)
	if err != nil {
		_ = writeResultMessage(resultPath, err.Error())
		return 3, err
	}
	verdict, message := Compare(mode, answerBytes, output)
	if err := writeResultMessage(resultPath, message); err != nil {
		return 3, err
	}
	switch verdict {
	case VerdictAccepted:
		return 0, nil
	case VerdictWrongAnswer:
		return 1, nil
	case VerdictPresentationError:
		return 2, nil
	default:
		return 3, nil
	}
}

func Compare(mode JudgeMode, answer []byte, output []byte) (Verdict, string) {
	switch mode {
	case ModeDefault, "":
		if equalDefault(answer, output) {
			return VerdictAccepted, ""
		}
		return VerdictWrongAnswer, ""
	case ModeStrict:
		if bytes.Equal(answer, output) {
			return VerdictAccepted, ""
		}
		if equalDefault(answer, output) {
			return VerdictPresentationError, ""
		}
		return VerdictWrongAnswer, ""
	default:
		return VerdictSystemError, "unsupported builtin judge mode"
	}
}

func equalDefault(answer []byte, output []byte) bool {
	return normalizedLines(answer) == normalizedLines(output)
}

func normalizedLines(raw []byte) string {
	value := strings.ReplaceAll(string(raw), "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(value, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

func writeResultMessage(path string, message string) error {
	if path == "" {
		return nil
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(message)
	return err
}
