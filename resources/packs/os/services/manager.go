package services

import (
	"errors"
	"regexp"

	"go.mondoo.com/cnquery/motor/providers/os"

	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/platform"
)

type Service struct {
	Name        string
	Description string
	State       State
	Type        string
	Installed   bool
	Running     bool
	Enabled     bool
	Masked      bool
}

type State string

const (
	ServiceContinuePending State = "ServiceContinuePending"
	ServicePausePending    State = "ServicePausePending"
	ServicePaused          State = "ServicePaused"
	ServiceRunning         State = "ServiceRunning"
	ServiceStartPending    State = "ServiceStartPending"
	ServiceStopPending     State = "ServiceStopPending"
	ServiceStopped         State = "ServiceStopped"
	ServiceUnknown         State = "ServiceUnknown"
)

type OSServiceManager interface {
	Name() string
	List() ([]*Service, error)
}

var amazonlinux1version = regexp.MustCompile(`^201\d`)

func ResolveManager(motor *motor.Motor) (OSServiceManager, error) {
	var osm OSServiceManager

	pf, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	osProvider, isOSProvider := motor.Provider.(os.OperatingSystemProvider)
	if !isOSProvider {
		return nil, errors.New("services manager is not supported for platform: " + pf.Name)
	}

	switch {
	case pf.IsFamily("arch"): // arch family
		osm = ResolveSystemdServiceManager(osProvider)
	case pf.Name == "amazonlinux":
		if amazonlinux1version.MatchString(pf.Version) {
			osm = &UpstartServiceManager{SysVServiceManager{provider: osProvider}}
		} else {
			osm = ResolveSystemdServiceManager(osProvider)
		}
	case pf.Name == "photon":
		osm = ResolveSystemdServiceManager(osProvider)
	// NOTE: we need to check fedora before rhel family, since its also rhel family
	case pf.Name == "fedora":
		rv := platform.ParseOsVersion(pf.Version)
		v, err := rv.MajorAtoi()
		if err != nil {
			return nil, errors.New("unknown fedora version: " + pf.Version)
		}

		if v < 15 {
			// upstart is only used since fedora 11 but we do not support those older versions
			osm = &UpstartServiceManager{SysVServiceManager{provider: osProvider}}
		} else {
			osm = ResolveSystemdServiceManager(osProvider)
		}
	case pf.IsFamily("redhat"):
		rv := platform.ParseOsVersion(pf.Version)
		v, err := rv.MajorAtoi()
		if err != nil {
			return nil, errors.New("unknown redhat version: " + pf.Version)
		}
		if v < 7 {
			osm = &UpstartServiceManager{SysVServiceManager{provider: osProvider}}
		} else {
			osm = ResolveSystemdServiceManager(osProvider)
		}
	case pf.Name == "ubuntu" || pf.Name == "linuxmint" || pf.Name == "pop":
		rv := platform.ParseOsVersion(pf.Version)
		v, err := rv.MajorAtoi()
		if err != nil {
			return nil, errors.New("unknown ubuntu version: " + pf.Version)
		}

		if v < 15 {
			osm = &UpstartServiceManager{SysVServiceManager{provider: osProvider}}
		} else {
			osm = ResolveSystemdServiceManager(osProvider)
		}
	case pf.Name == "debian":
		rv := platform.ParseOsVersion(pf.Version)
		v, err := rv.MajorAtoi()
		if err != nil {
			return nil, errors.New("unknown debian version: " + pf.Version)
		}

		if v < 7 {
			osm = &SysVServiceManager{provider: osProvider}
		} else {
			osm = ResolveSystemdServiceManager(osProvider)
		}
	case pf.Name == "suse-microos": // it is suse family but uses a different version scheme
		osm = ResolveSystemdServiceManager(osProvider)
	case pf.IsFamily("suse"):
		rv := platform.ParseOsVersion(pf.Version)
		v, err := rv.MajorAtoi()
		if err != nil {
			return nil, errors.New("unknown suse version: " + pf.Version)
		}

		// NOTE: opensuse-tumbleweed uses version numbers like 20200622
		if v < 12 {
			osm = &SysVServiceManager{provider: osProvider}
		} else {
			osm = ResolveSystemdServiceManager(osProvider)
		}
	case pf.IsFamily("darwin"): // "macos", "darwin"
		osm = &LaunchDServiceManager{provider: osProvider}
	case pf.Name == "freebsd" || pf.Name == "dragonflybsd" || pf.Name == "netbsd":
		osm = &BsdInitServiceManager{provider: osProvider}
	case pf.Name == "openbsd":
		osm = &OpenBsdRcctlServiceManager{provider: osProvider}
	case pf.Name == "windows":
		osm = &WindowsServiceManager{provider: osProvider}
	case pf.Name == "alpine":
		osm = &AlpineOpenrcServiceManager{provider: osProvider}
	case pf.Name == "cos":
		osm = ResolveSystemdServiceManager(osProvider)
	}

	if osm == nil {
		return nil, errors.New("could not detect suitable service manager for platform: " + pf.Name)
	}

	return osm, nil
}

func FindService(services []*Service, name string) (*Service, error) {
	// search for name
	for i := range services {
		service := services[i]
		if service.Name == name {
			return service, nil
		}
	}

	return nil, errors.New("service> " + name + " does not exist")
}
