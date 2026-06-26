package judger

import (
	"errors"
	"testing"
)

func TestDockerfileCommandExecForm(t *testing.T) {
	got, err := dockerfileCommand("FROM gcc:14\nCMD [\"python3\", \"/src/main.py\"]\n")
	if err != nil {
		t.Fatalf("dockerfile command: %v", err)
	}
	if got != "'python3' '/src/main.py'" {
		t.Fatalf("command = %q", got)
	}
}

func TestDockerfileCommandShellFormUsesLastCMD(t *testing.T) {
	got, err := dockerfileCommand("CMD old\nFROM alpine\nCMD /main --flag\n")
	if err != nil {
		t.Fatalf("dockerfile command: %v", err)
	}
	if got != "/main --flag" {
		t.Fatalf("command = %q", got)
	}
}

func TestDockerfileCommandRequired(t *testing.T) {
	if _, err := dockerfileCommand("FROM alpine\n"); err == nil {
		t.Fatal("expected missing CMD error")
	}
}

func TestLanguageBuildErrorBecomesCompileError(t *testing.T) {
	task := Task{SubmissionID: 14, Attempt: 2}
	result, ok := taskResultForLanguageBuildError(task, languageBuildError{Message: "build language image: main.cc:1: error"})
	if !ok {
		t.Fatal("language build error should be handled as a compile result")
	}
	if result.SubmissionID != task.SubmissionID || result.Attempt != task.Attempt || result.Verdict != VerdictCompileError {
		t.Fatalf("result = %#v", result)
	}
	if result.Message != "build language image: main.cc:1: error" {
		t.Fatalf("message = %q", result.Message)
	}
}

func TestPlainLanguagePrepareErrorStaysUnhandled(t *testing.T) {
	if _, ok := taskResultForLanguageBuildError(Task{}, errors.New("docker unavailable")); ok {
		t.Fatal("plain infrastructure errors should not be converted to compile errors")
	}
}

func TestCleanBuildMessageStripsANSI(t *testing.T) {
	got := cleanBuildMessage("\x1b[91mmain.cc:1: error\x1b[0m\n")
	if got != "main.cc:1: error" {
		t.Fatalf("cleanBuildMessage = %q", got)
	}
}
