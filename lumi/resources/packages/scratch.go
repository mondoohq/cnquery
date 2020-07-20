package packages

import "go.mondoo.io/mondoo/motor"

type ScratchPkgManager struct {
	motor *motor.Motor
}

func (dpm *ScratchPkgManager) Name() string {
	return "Scratch Package Manager"
}

func (dpm *ScratchPkgManager) Format() string {
	return "scratch"
}

func (dpm *ScratchPkgManager) List() ([]Package, error) {
	return []Package{}, nil
}

func (dpm *ScratchPkgManager) Available() (map[string]PackageUpdate, error) {
	return map[string]PackageUpdate{}, nil
}
