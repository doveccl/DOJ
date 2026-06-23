package judger

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func MapInnerPID(procRoot string, containerInitHostPID int, innerPID int) (int, error) {
	if procRoot == "" {
		procRoot = "/proc"
	}
	if containerInitHostPID <= 0 {
		return 0, fmt.Errorf("container init host pid is required")
	}
	if innerPID <= 0 {
		return 0, fmt.Errorf("inner pid is required")
	}
	targetNS, err := pidNamespace(procRoot, containerInitHostPID)
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		hostPID, err := strconv.Atoi(entry.Name())
		if err != nil || hostPID <= 0 {
			continue
		}
		ns, err := pidNamespace(procRoot, hostPID)
		if err != nil || ns != targetNS {
			continue
		}
		ids, err := nspids(procRoot, hostPID)
		if err != nil || len(ids) == 0 {
			continue
		}
		if ids[len(ids)-1] == innerPID {
			return hostPID, nil
		}
	}
	return 0, fmt.Errorf("map inner pid %d in namespace %s: not found", innerPID, targetNS)
}

func pidNamespace(procRoot string, hostPID int) (string, error) {
	path := filepath.Join(procRoot, strconv.Itoa(hostPID), "ns", "pid")
	target, err := os.Readlink(path)
	if err != nil {
		return "", fmt.Errorf("read pid namespace for host pid %d: %w", hostPID, err)
	}
	if target == "" {
		return "", fmt.Errorf("pid namespace for host pid %d is empty", hostPID)
	}
	return target, nil
}

func nspids(procRoot string, hostPID int) ([]int, error) {
	raw, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(hostPID), "status"))
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok || key != "NSpid" {
			continue
		}
		fields := strings.Fields(value)
		ids := make([]int, 0, len(fields))
		for _, field := range fields {
			id, err := strconv.Atoi(field)
			if err != nil || id <= 0 {
				return nil, fmt.Errorf("invalid NSpid field %q for host pid %d", field, hostPID)
			}
			ids = append(ids, id)
		}
		return ids, nil
	}
	return nil, fmt.Errorf("NSpid not found for host pid %d", hostPID)
}
