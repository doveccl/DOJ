package judger

import (
	"strings"
	"testing"
)

func TestReadDockerErrorStream(t *testing.T) {
	if err := readDockerErrorStream(strings.NewReader(`{"status":"Pulling fs layer"}` + "\n")); err != nil {
		t.Fatalf("status stream returned error: %v", err)
	}
	err := readDockerErrorStream(strings.NewReader(`{"errorDetail":{"message":"pull failed"},"error":"pull failed"}` + "\n"))
	if err == nil || err.Error() != "pull failed" {
		t.Fatalf("error stream = %v", err)
	}
}
