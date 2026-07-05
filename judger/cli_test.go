package judger

import (
	"context"
	"testing"
)

func TestJudgerCLIRejectsRuntimeFlags(t *testing.T) {
	if code := JudgerCLI(context.Background(), []string{"serve"}); code != 2 {
		t.Fatalf("JudgerCLI serve code = %d, want 2", code)
	}
	if code := JudgerCLI(context.Background(), []string{"--server", "http://example.test"}); code != 2 {
		t.Fatalf("JudgerCLI flag code = %d, want 2", code)
	}
}

func TestJudgerCLIVersion(t *testing.T) {
	if code := JudgerCLI(context.Background(), []string{"version"}); code != 0 {
		t.Fatalf("JudgerCLI version code = %d, want 0", code)
	}
}
