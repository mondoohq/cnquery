package packages

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/motor"
)

type OperatingSystemPkgManager interface {
	Name() string
	List() ([]Package, error)
	Available() (map[string]PackageUpdate, error)
}

type OperatingSystemUpdateManager interface {
	Name() string
	List() ([]OperatingSystemUpdate, error)
}

// this will find the right package manager for the operating system
func ResolveSystemPkgManager(motor *motor.Motor) (OperatingSystemPkgManager, error) {
	var pm OperatingSystemPkgManager

	pf, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	switch {
	case pf.IsFamily("arch"): // arch family
		pm = &PacmanPkgManager{motor: motor}
	case pf.IsFamily("debian"): // debian family
		pm = &DebPkgManager{motor: motor}
	case pf.Name == "amazonlinux" || pf.Name == "photon" || pf.Name == "wrlinux":
		fallthrough
	case pf.IsFamily("redhat"): // rhel family
		pm = &RpmPkgManager{motor: motor, platform: pf}
	case pf.IsFamily("suse"): // suse handling
		pm = &SusePkgManager{RpmPkgManager{motor: motor, platform: pf}}
	case pf.Name == "alpine": // alpine
		pm = &AlpinePkgManager{motor: motor}
	case pf.Name == "macos": // mac os family
		pm = &MacOSPkgManager{motor: motor}
	case pf.Name == "windows":
		pm = &WinPkgManager{motor: motor}
	case pf.Name == "scratch" || pf.Name == "coreos":
		pm = &ScratchPkgManager{motor: motor}
	case pf.Name == "openwrt":
		pm = &OpkgPkgManager{motor: motor}
	case pf.Name == "solaris":
		pm = &SolarisPkgManager{motor: motor}
	default:
		return nil, errors.New("could not detect suitable package manager for platform: " + pf.Name)
	}

	return pm, nil
}

// TODO: harmonize with ResolveSystemPkgManager
// this will find the right package manager for the operating system
func ResolveSystemUpdateManager(motor *motor.Motor) (OperatingSystemUpdateManager, error) {
	var um OperatingSystemUpdateManager

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	// TODO: use OS family and select package manager
	switch platform.Name {
	case "opensuse", "sles", "opensuse-leap", "opensuse-tumbleweed": // suse family
		um = &SuseUpdateManager{Motor: motor}
	default:
		return nil, errors.New("your platform is not supported by os updates resource")
	}
	return um, nil
}
