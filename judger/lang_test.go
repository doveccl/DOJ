package judger

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareLanguageRuntimeWritesSourceAndCommands(t *testing.T) {
	work := t.TempDir()
	lang, err := prepareLanguageRuntime(work, Lang{
		ID:      "cpp",
		Source:  "main.cc",
		Image:   "gcc:14",
		Compile: "g++ main.cc -o /work/main",
		Run:     "./main",
	}, "int main(){}\n")
	if err != nil {
		t.Fatalf("prepare language runtime: %v", err)
	}
	if lang.Image != "gcc:14" || lang.CompileCommand != "g++ main.cc -o /work/main" || lang.RunCommand != "./main" {
		t.Fatalf("runtime = %#v", lang)
	}
	got, err := os.ReadFile(filepath.Join(work, "src", "main.cc"))
	if err != nil || string(got) != "int main(){}\n" {
		t.Fatalf("source = %q, %v", got, err)
	}
}

func TestPrepareLanguageRuntimeRejectsUnsafeSource(t *testing.T) {
	for _, source := range []string{"../main.cc", "/main.cc", ".", "src/main.cc"} {
		t.Run(source, func(t *testing.T) {
			if _, err := prepareLanguageRuntime(t.TempDir(), Lang{Source: source, Image: "gcc:14", Run: "./main"}, ""); err == nil {
				t.Fatal("expected unsafe source to be rejected")
			}
		})
	}
}

func TestCompileLanguageRuntimeWithoutCompileCopiesSource(t *testing.T) {
	work := t.TempDir()
	lang, err := prepareLanguageRuntime(work, Lang{
		ID:     "py",
		Source: "main.py",
		Image:  "python:3",
		Run:    "python3 main.py",
	}, "print(42)\n")
	if err != nil {
		t.Fatalf("prepare language runtime: %v", err)
	}
	got, err := compileLanguageRuntime(context.Background(), lang, work, Limits{}, 1, 1, nil)
	if err != nil {
		t.Fatalf("compile language runtime: %v", err)
	}
	if !got.OK {
		t.Fatalf("compile result = %#v", got)
	}
	source, err := os.ReadFile(filepath.Join(work, "main.py"))
	if err != nil || string(source) != "print(42)\n" {
		t.Fatalf("runtime source = %q, %v", source, err)
	}
}

func TestCleanBuildMessageStripsANSI(t *testing.T) {
	got := cleanBuildMessage("\x1b[91mmain.cc:1: error\x1b[0m\n")
	if got != "main.cc:1: error" {
		t.Fatalf("cleanBuildMessage = %q", got)
	}
}
