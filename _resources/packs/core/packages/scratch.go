package packages

import (
	"go.mondoo.com/cnquery/motor/providers/os"
)

type ScratchPkgManager struct {
	provider os.OperatingSystemProvider
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
