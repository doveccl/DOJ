package runner

import (
	"fmt"
	"strings"
)

func parseCommand(command string) (string, []string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", nil, fmt.Errorf("command is required")
	}
	if strings.ContainsAny(command, "\"'`") {
		return "", nil, fmt.Errorf("command must use plain space-separated arguments")
	}
	parts := strings.Fields(command)
	return parts[0], parts[1:], nil
}
