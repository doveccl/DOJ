//go:build !linux

package runner

func DefaultCgroupRoot() string {
	return "/sys/fs/cgroup/doj"
}

func PrepareCgroup(cfg CgroupConfig) (*CgroupCase, error) {
	return nil, ErrCgroupUnsupported
}

func (cg *CgroupCase) Add(pid int) error {
	return ErrCgroupUnsupported
}

func (cg *CgroupCase) Stats() (CgroupStats, error) {
	return CgroupStats{}, ErrCgroupUnsupported
}

func (cg *CgroupCase) Cleanup() error {
	return nil
}

func (cg *CgroupCase) killAll() error {
	return nil
}

func (cg *CgroupCase) KillAll() error {
	return cg.killAll()
}
