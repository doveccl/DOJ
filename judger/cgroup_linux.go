//go:build linux

package judger

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func DefaultCgroupRoot() string {
	for _, root := range []string{"/sys/fs/cgroup", "/sys/fs/cgroup/unified"} {
		if isCgroupV2Root(root) {
			return filepath.Join(root, "doj")
		}
	}
	return filepath.Join("/sys/fs/cgroup", "doj")
}

func isCgroupV2Root(root string) bool {
	info, err := os.Stat(filepath.Join(root, "cgroup.controllers"))
	return err == nil && !info.IsDir()
}

func PrepareCgroup(cfg CgroupConfig) (*CgroupCase, error) {
	if cfg.Root == "" {
		cfg.Root = DefaultCgroupRoot()
	}
	if cfg.SubmissionID == "" || cfg.CaseID == "" {
		return nil, fmt.Errorf("cgroup requires submission and case id")
	}
	submissionPath := filepath.Join(cfg.Root, cfg.SubmissionID)
	casePath := filepath.Join(submissionPath, cfg.CaseID)
	userPath := filepath.Join(casePath, "user")
	if err := os.MkdirAll(cfg.Root, 0o755); err != nil {
		return nil, err
	}
	if err := enableControllers(cfg.Root); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(submissionPath, 0o755); err != nil {
		return nil, err
	}
	if err := enableControllers(submissionPath); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(casePath, 0o755); err != nil {
		return nil, err
	}
	if err := enableControllers(casePath); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(userPath, 0o755); err != nil {
		return nil, err
	}
	if cfg.MemoryMax > 0 {
		if err := os.WriteFile(filepath.Join(userPath, "memory.max"), []byte(strconv.FormatInt(cfg.MemoryMax, 10)), 0o644); err != nil {
			return nil, err
		}
	}
	if cfg.PidsMax > 0 {
		if err := os.WriteFile(filepath.Join(userPath, "pids.max"), []byte(strconv.Itoa(cfg.PidsMax)), 0o644); err != nil {
			return nil, err
		}
	}
	return &CgroupCase{Path: userPath}, nil
}

func (cg *CgroupCase) Add(pid int) error {
	return os.WriteFile(filepath.Join(cg.Path, "cgroup.procs"), []byte(strconv.Itoa(pid)), 0o644)
}

func (cg *CgroupCase) Stats() (CgroupStats, error) {
	var stats CgroupStats
	peak, err := readInt(filepath.Join(cg.Path, "memory.peak"))
	if err == nil {
		stats.MemoryPeak = peak
	}
	events, err := os.ReadFile(filepath.Join(cg.Path, "memory.events"))
	if err == nil {
		for _, line := range strings.Split(string(events), "\n") {
			fields := strings.Fields(line)
			if len(fields) != 2 || fields[1] == "0" {
				continue
			}
			switch fields[0] {
			case "max":
				stats.MemoryMaxed = true
			case "oom", "oom_kill", "oom_group_kill":
				stats.MemoryOOM = true
			}
		}
	}
	pidsCurrent, err := readInt(filepath.Join(cg.Path, "pids.current"))
	if err == nil {
		stats.PidsCurrent = pidsCurrent
	}
	pidsEvents, err := os.ReadFile(filepath.Join(cg.Path, "pids.events"))
	if err == nil {
		for _, line := range strings.Split(string(pidsEvents), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == "max" && fields[1] != "0" {
				stats.PidsMaxed = true
			}
		}
	}
	cpu, err := os.ReadFile(filepath.Join(cg.Path, "cpu.stat"))
	if err == nil {
		for _, line := range strings.Split(string(cpu), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == "usage_usec" {
				stats.CPUUsageUSec, _ = strconv.ParseInt(fields[1], 10, 64)
			}
		}
	}
	return stats, nil
}

func (cg *CgroupCase) Cleanup() error {
	_ = cg.killAll()
	err := os.Remove(cg.Path)
	_ = os.Remove(filepath.Dir(cg.Path))
	_ = os.Remove(filepath.Dir(filepath.Dir(cg.Path)))
	return err
}

func (cg *CgroupCase) killAll() error {
	procsPath := filepath.Join(cg.Path, "cgroup.procs")
	for attempt := 0; attempt < 20; attempt++ {
		raw, err := os.ReadFile(procsPath)
		if err != nil {
			return err
		}
		pids := strings.Fields(string(raw))
		if len(pids) == 0 {
			return nil
		}
		for _, pidText := range pids {
			pid, err := strconv.Atoi(pidText)
			if err != nil || pid <= 0 {
				continue
			}
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
		time.Sleep(25 * time.Millisecond)
	}
	return nil
}

func readInt(path string) (int64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
}

func enableControllers(path string) error {
	availableRaw, err := os.ReadFile(filepath.Join(path, "cgroup.controllers"))
	if err != nil {
		return err
	}
	available := map[string]bool{}
	for _, item := range strings.Fields(string(availableRaw)) {
		available[item] = true
	}
	currentRaw, err := os.ReadFile(filepath.Join(path, "cgroup.subtree_control"))
	if err != nil {
		return err
	}
	current := map[string]bool{}
	for _, item := range strings.Fields(string(currentRaw)) {
		current[strings.TrimPrefix(item, "+")] = true
	}
	for _, controller := range []string{"cpu", "memory", "pids"} {
		if !available[controller] {
			return fmt.Errorf("cgroup v2 controller %s is unavailable at %s", controller, path)
		}
		if current[controller] {
			continue
		}
		if err := os.WriteFile(filepath.Join(path, "cgroup.subtree_control"), []byte("+"+controller), 0o644); err != nil {
			return fmt.Errorf("enable cgroup controller %s at %s: %w", controller, path, err)
		}
	}
	return nil
}
