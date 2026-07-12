package judger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/doveccl/doj/judger/runner"
)

func TestPrepareLanguageRuntimeWritesSourceAndCommands(t *testing.T) {
	work := t.TempDir()
	lang, err := prepareLanguageRuntime(work, runner.Lang{
		ID:        "cpp",
		Source:    "main.cc",
		Image:     "gcc:14",
		Compile:   "g++ main.cc -o main",
		CompileMS: 10000,
		Run:       "./main",
	}, "int main(){}\n")
	if err != nil {
		t.Fatalf("prepare language runtime: %v", err)
	}
	if lang.Image != "gcc:14" || lang.CompileCommand != "g++ main.cc -o main" || lang.CompileMS != 10000 || lang.RunCommand != "./main" {
		t.Fatalf("runtime = %#v", lang)
	}
	got, err := os.ReadFile(filepath.Join(work, "src", "main.cc"))
	if err != nil || string(got) != "int main(){}\n" {
		t.Fatalf("source = %q, %v", got, err)
	}
	for path, want := range map[string]os.FileMode{
		filepath.Join(work, "src"):            0o700,
		filepath.Join(work, "src", "main.cc"): 0o600,
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Fatalf("%s mode = %o, want %o", path, got, want)
		}
	}
}

func TestPrepareLanguageRuntimeRejectsUnsafeSource(t *testing.T) {
	for _, source := range []string{"../main.cc", "/main.cc", ".", "src/main.cc"} {
		t.Run(source, func(t *testing.T) {
			if _, err := prepareLanguageRuntime(t.TempDir(), runner.Lang{Source: source, Image: "gcc:14", Run: "./main"}, ""); err == nil {
				t.Fatal("expected unsafe source to be rejected")
			}
		})
	}
}

func TestCleanBuildMessageStripsANSI(t *testing.T) {
	got := runner.CleanBuildMessage("\x1b[91mmain.cc:1: error\x1b[0m\n")
	if got != "main.cc:1: error" {
		t.Fatalf("cleanBuildMessage = %q", got)
	}
}
