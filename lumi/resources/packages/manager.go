package packages

import (
	"github.com/pkg/errors"

	motor "go.mondoo.io/mondoo/motor/motoros"
)

type OperatingSystemPkgManager interface {
	Name() string
	Format() string
	List() ([]Package, error)
	Available() (map[string]PackageUpdate, error)
}

// this will find the right package manager for the operating system
func ResolveSystemPkgManager(motor *motor.Motor) (OperatingSystemPkgManager, error) {
	var pm OperatingSystemPkgManager

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	// TODO: use OS family and select package manager
	switch platform.Name {
	case "manjaro", "arch": // arch family
		pm = &PacmanPkgManager{motor: motor}
	case "ubuntu", "debian", "raspbian", "kali": // debian family
		pm = &DebPkgManager{motor: motor}
	case "redhat", "centos", "fedora", "amzn", "ol", "scientific", "photon", "wrlinux": // rhel family
		pm = &RpmPkgManager{motor: motor, platform: &platform}
	case "opensuse", "sles", "opensuse-leap", "opensuse-tumbleweed": // suse handling
		pm = &SusePkgManager{RpmPkgManager{motor: motor, platform: &platform}}
	case "alpine": // alpine family
		pm = &AlpinePkgManager{motor: motor}
	case "mac_os_x": // mac os family
		pm = &MacOSPkgManager{motor: motor}
	case "windows":
		pm = &WinPkgManager{motor: motor}
	case "scratch", "coreos":
		pm = &ScratchPkgManager{motor: motor}
	case "openwrt":
		pm = &OpkgPkgManager{motor: motor}
	case "solaris":
		pm = &SolarisPkgManager{motor: motor}
	default:
		return nil, errors.New("could not detect suitable package manager for platform: " + platform.Name)
	}

	return pm, nil
}

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
