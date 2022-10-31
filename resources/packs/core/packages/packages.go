package packages

import (
	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers/os"
)

type Package struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Arch        string `json:"arch"`
	Status      string `json:"status,omitempty"`
	Description string `json:"description"`

	// this may be the source package or an origin
	// e.g. on alpine it is used for parent  packages
	// o 	Package Origin - https://wiki.alpinelinux.org/wiki/Apk_spec
	Origin string `json:"origin"`
	Format string `json:"format"`
}

// extends Package to store available version
type PackageUpdate struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Arch      string `json:"arch"`
	Available string `json:"available"`
	Repo      string `json:"repo"`
}

type OperatingSystemUpdate struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	Restart     bool   `json:"restart"`
	Format      string `json:"format"`
}

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

	osProvider, isOSProvider := motor.Provider.(os.OperatingSystemProvider)
	if !isOSProvider {
		return nil, errors.New("package manager is not supported for platform: " + pf.Name)
	}

	switch {
	case pf.IsFamily("arch"): // arch family
		pm = &PacmanPkgManager{provider: osProvider}
	case pf.IsFamily("debian"): // debian family
		pm = &DebPkgManager{provider: osProvider}
	case pf.Name == "amazonlinux" || pf.Name == "photon" || pf.Name == "wrlinux":
		fallthrough
	case pf.IsFamily("redhat"): // rhel family
		pm = &RpmPkgManager{provider: osProvider, platform: pf}
	case pf.IsFamily("suse"): // suse handling
		pm = &SusePkgManager{RpmPkgManager{provider: osProvider, platform: pf}}
	case pf.Name == "alpine": // alpine
		pm = &AlpinePkgManager{provider: osProvider}
	case pf.Name == "macos": // mac os family
		pm = &MacOSPkgManager{provider: osProvider}
	case pf.Name == "windows":
		pm = &WinPkgManager{provider: osProvider, platform: pf}
	case pf.Name == "scratch" || pf.Name == "coreos":
		pm = &ScratchPkgManager{provider: osProvider}
	case pf.Name == "openwrt":
		pm = &OpkgPkgManager{provider: osProvider}
	case pf.Name == "solaris":
		pm = &SolarisPkgManager{provider: osProvider}
	case pf.Name == "cos":
		pm = &CosPkgManager{provider: osProvider}
	case pf.Name == "freebsd":
		pm = &FreeBSDPkgManager{provider: osProvider}
	default:
		return nil, errors.New("could not detect suitable package manager for platform: " + pf.Name)
	}

	return pm, nil
}

// TODO: harmonize with ResolveSystemPkgManager
// this will find the right package manager for the operating system
func ResolveSystemUpdateManager(motor *motor.Motor) (OperatingSystemUpdateManager, error) {
	var um OperatingSystemUpdateManager

	pf, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	osProvider, isOSProvider := motor.Provider.(os.OperatingSystemProvider)
	if !isOSProvider {
		return nil, errors.New("package manager is not supported for platform: " + pf.Name)
	}

	// TODO: use OS family and select package manager
	switch pf.Name {
	case "opensuse", "sles", "opensuse-leap", "opensuse-tumbleweed": // suse family
		um = &SuseUpdateManager{provider: osProvider}
	default:
		return nil, errors.New("your platform is not supported by os updates resource")
	}
	return um, nil
}
