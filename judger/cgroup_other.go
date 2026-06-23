//go:build !linux

package judger

func PrepareCgroup(CgroupConfig) (*CgroupCase, error) {
	return nil, ErrCgroupUnsupported
}

func (cg *CgroupCase) Add(int) error {
	return ErrCgroupUnsupported
}

func (cg *CgroupCase) Stats() (CgroupStats, error) {
	return CgroupStats{}, ErrCgroupUnsupported
}

func (cg *CgroupCase) Cleanup() error {
	return ErrCgroupUnsupported
}
