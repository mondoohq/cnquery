package packages

import (
	"errors"
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

type OperatingSystemPkgManager interface {
	Name() string
	List() ([]Package, error)
	Available() (map[string]PackageUpdate, error)
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
	case pf.IsFamily("linux"):
		// no clear package manager for linux platform found
		// most likely we land here if we have a yocto-based system
		opkgPaths := []string{"/bin/opkg", "/usr/bin/opkg"}
		for i := range opkgPaths {
			_, err = osProvider.FS().Stat(opkgPaths[i])
			if err == nil {
				pm = &OpkgPkgManager{provider: osProvider}
				break
			}
		}
	}

	if pm == nil {
		return nil, errors.New("could not detect suitable package manager for platform: " + pf.Name)
	}

	return pm, nil
}
