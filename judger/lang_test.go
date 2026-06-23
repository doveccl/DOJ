package judger

import "testing"

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
