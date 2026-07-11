package validate

import (
	"strings"
	"testing"
)

func TestPasswordLengthMatchesBcryptBoundary(t *testing.T) {
	for _, value := range []string{strings.Repeat("x", 8), strings.Repeat("x", 72)} {
		if !Password(value) {
			t.Fatalf("valid password length %d rejected", len(value))
		}
	}
	for _, value := range []string{strings.Repeat("x", 7), strings.Repeat("x", 73)} {
		if Password(value) {
			t.Fatalf("invalid password length %d accepted", len(value))
		}
	}
}
