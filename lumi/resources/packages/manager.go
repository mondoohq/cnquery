package packages

import (
	"github.com/cockroachdb/errors"

	"go.mondoo.io/mondoo/motor"
)

type OperatingSystemPkgManager interface {
	Name() string
	Format() string
	List() ([]Package, error)
	Available() (map[string]PackageUpdate, error)
}

type OperatingSystemUpdateManager interface {
	Name() string
	Format() string
	List() ([]OperatingSystemUpdate, error)
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
	case "ubuntu", "debian", "raspbian", "kali", "linuxmint": // debian family
		pm = &DebPkgManager{motor: motor}
	case "redhat", "centos", "fedora", "amazonlinux", "oraclelinux", "scientific", "photon", "wrlinux": // rhel family
		pm = &RpmPkgManager{motor: motor, platform: platform}
	case "opensuse", "sles", "opensuse-leap", "opensuse-tumbleweed": // suse handling
		pm = &SusePkgManager{RpmPkgManager{motor: motor, platform: platform}}
	case "alpine": // alpine family
		pm = &AlpinePkgManager{motor: motor}
	case "macos": // mac os family
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
