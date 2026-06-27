package judger

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

const defaultOutputLimit = 64 << 20

var errOutputLimit = errors.New("output limit exceeded")

type JudgeReport struct {
	Verdict Verdict `json:"verdict"`
	Score   int     `json:"score"`
	Message string  `json:"message"`
}

func BuiltinJudgeMain(args []string) int {
	flags := flag.NewFlagSet("judge", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	mode := flags.String("mode", string(ModeDefault), "judge mode")
	input := flags.String("input", "", "input file")
	answer := flags.String("answer", "", "answer file")
	result := flags.String("result", "", "result file")
	outputLimit := flags.Int64("output-limit", defaultOutputLimit, "user output limit in bytes")
	if err := flags.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if *input == "" || *answer == "" || *result == "" {
		fmt.Fprintln(os.Stderr, "judge requires --input, --answer, and --result")
		return 2
	}
	signal.Ignore(syscall.SIGPIPE)

	inputBytes, err := os.ReadFile(*input)
	if err != nil {
		writeReport(*result, JudgeReport{Verdict: VerdictSystemError, Message: err.Error()})
		return 2
	}
	answerBytes, err := os.ReadFile(*answer)
	if err != nil {
		writeReport(*result, JudgeReport{Verdict: VerdictSystemError, Message: err.Error()})
		return 2
	}
	inputErr := make(chan error, 1)
	go func() {
		_, err := os.Stdout.Write(inputBytes)
		if errors.Is(err, syscall.EPIPE) {
			err = nil
		}
		if closeErr := os.Stdout.Close(); err == nil && closeErr != nil && !errors.Is(closeErr, syscall.EPIPE) {
			err = closeErr
		}
		inputErr <- err
	}()

	output, err := readLimited(os.Stdin, *outputLimit)
	if err != nil {
		writeReport(*result, JudgeReport{Verdict: VerdictOutputLimit, Message: err.Error()})
		return 0
	}
	if err := <-inputErr; err != nil {
		writeReport(*result, JudgeReport{Verdict: VerdictSystemError, Message: err.Error()})
		return 2
	}
	verdict, message := Compare(JudgeMode(*mode), answerBytes, output)
	score := 0
	if verdict == VerdictAccepted {
		score = 100
	}
	if err := writeReport(*result, JudgeReport{Verdict: verdict, Score: score, Message: message}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	return 0
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

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		limit = defaultOutputLimit
	}
	var buf bytes.Buffer
	n, err := io.CopyN(&buf, r, limit+1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if n > limit {
		return nil, errOutputLimit
	}
	return buf.Bytes(), nil
}

func writeReport(path string, report JudgeReport) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(report)
}
