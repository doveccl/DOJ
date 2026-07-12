package judger

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/doveccl/doj/judger/runner"
)

func TestTarDirectoryStreamsFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	body := tarDirectory(dir)
	defer body.Close()
	reader := tar.NewReader(body)
	header, err := reader.Next()
	if err != nil || header.Name != "Dockerfile" {
		t.Fatalf("tar header = %#v, %v", header, err)
	}
	data, err := io.ReadAll(reader)
	if err != nil || string(data) != "FROM scratch\n" {
		t.Fatalf("tar data = %q, %v", data, err)
	}
}

func TestReadDockerErrorStream(t *testing.T) {
	if err := readDockerErrorStream(strings.NewReader(`{"status":"Pulling fs layer"}` + "\n")); err != nil {
		t.Fatalf("status stream returned error: %v", err)
	}
	err := readDockerErrorStream(strings.NewReader(`{"errorDetail":{"message":"pull failed"},"error":"pull failed"}` + "\n"))
	if err == nil || err.Error() != "pull failed" {
		t.Fatalf("error stream = %v", err)
	}
}

func TestRunnerHostConfigBoundsResources(t *testing.T) {
	config := runnerHostConfig(runner.Limits{MemoryKB: 256 << 10, Pids: 64, FileKB: 64 << 10}, []string{"work:/work"})
	if config.Memory != runnerMemoryFloor || config.NanoCPUs != runnerNanoCPUs || config.PidsLimit != runnerPidsFloor {
		t.Fatalf("runner resources = memory %d cpu %d pids %d", config.Memory, config.NanoCPUs, config.PidsLimit)
	}
	if config.FileSize != 64<<20 || !config.ReadonlyRootfs || !strings.Contains(config.Tmpfs["/tmp"], "exec") || !strings.Contains(config.Tmpfs["/var/tmp"], "exec") {
		t.Fatalf("runner filesystem limits = %#v", config)
	}
	if len(config.CapDrop) != 1 || config.CapDrop[0] != "ALL" || len(config.CapAdd) == 0 {
		t.Fatalf("runner capabilities = drop %#v add %#v", config.CapDrop, config.CapAdd)
	}
}

func TestApplyContainerCgroupStatsUsesCPUTimeForNonTimeout(t *testing.T) {
	result := runner.CaseResult{CaseID: "1", Verdict: runner.VerdictWrongAnswer, TimeMS: 1200}
	applyCgroupStatsSnapshot(&result, runner.CgroupStats{CPUUsageUSec: 12_345})
	if result.TimeMS != 13 {
		t.Fatalf("time ms = %d", result.TimeMS)
	}
}

func TestApplyContainerCgroupStatsKeepsWallTimeForTimeout(t *testing.T) {
	result := runner.CaseResult{CaseID: "1", Verdict: runner.VerdictTimeLimit, TimeMS: 1001}
	applyCgroupStatsSnapshot(&result, runner.CgroupStats{CPUUsageUSec: 12_345})
	if result.TimeMS != 1001 {
		t.Fatalf("time ms = %d", result.TimeMS)
	}
}
