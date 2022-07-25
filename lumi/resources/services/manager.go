package services

import (
	"errors"
	"regexp"

	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/platform"
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

	switch {
	case pf.IsFamily("arch"): // arch family
		osm = ResolveSystemdServiceManager(motor)
	case pf.Name == "amazonlinux":
		if amazonlinux1version.MatchString(pf.Version) {
			osm = &UpstartServiceManager{SysVServiceManager{motor: motor}}
		} else {
			osm = ResolveSystemdServiceManager(motor)
		}
	case pf.Name == "photon":
		osm = ResolveSystemdServiceManager(motor)
	// NOTE: we need to check fedora before rhel family, since its also rhel family
	case pf.Name == "fedora":
		rv := platform.ParseOsVersion(pf.Version)
		v, err := rv.MajorAtoi()
		if err != nil {
			return nil, errors.New("unknown fedora version: " + pf.Version)
		}

		if v < 15 {
			// upstart is only used since fedora 11 but we do not support those older versions
			osm = &UpstartServiceManager{SysVServiceManager{motor: motor}}
		} else {
			osm = ResolveSystemdServiceManager(motor)
		}
	case pf.IsFamily("redhat"):
		rv := platform.ParseOsVersion(pf.Version)
		v, err := rv.MajorAtoi()
		if err != nil {
			return nil, errors.New("unknown redhat version: " + pf.Version)
		}
		if v < 7 {
			osm = &UpstartServiceManager{SysVServiceManager{motor: motor}}
		} else {
			osm = ResolveSystemdServiceManager(motor)
		}
	case pf.Name == "ubuntu" || pf.Name == "linuxmint" || pf.Name == "pop":
		rv := platform.ParseOsVersion(pf.Version)
		v, err := rv.MajorAtoi()
		if err != nil {
			return nil, errors.New("unknown ubuntu version: " + pf.Version)
		}

		if v < 15 {
			osm = &UpstartServiceManager{SysVServiceManager{motor: motor}}
		} else {
			osm = ResolveSystemdServiceManager(motor)
		}
	case pf.Name == "debian":
		rv := platform.ParseOsVersion(pf.Version)
		v, err := rv.MajorAtoi()
		if err != nil {
			return nil, errors.New("unknown debian version: " + pf.Version)
		}

		if v < 7 {
			osm = &SysVServiceManager{motor: motor}
		} else {
			osm = ResolveSystemdServiceManager(motor)
		}
	case pf.Name == "suse-microos": // it is suse family but uses a different version scheme
		osm = ResolveSystemdServiceManager(motor)
	case pf.IsFamily("suse"):
		rv := platform.ParseOsVersion(pf.Version)
		v, err := rv.MajorAtoi()
		if err != nil {
			return nil, errors.New("unknown suse version: " + pf.Version)
		}

		// NOTE: opensuse-tumbleweed uses version numbers like 20200622
		if v < 12 {
			osm = &SysVServiceManager{motor: motor}
		} else {
			osm = ResolveSystemdServiceManager(motor)
		}
	case pf.IsFamily("darwin"): // "macos", "darwin"
		osm = &LaunchDServiceManager{motor: motor}
	case pf.Name == "freebsd" || pf.Name == "dragonflybsd" || pf.Name == "netbsd":
		osm = &BsdInitServiceManager{motor: motor}
	case pf.Name == "openbsd":
		osm = &OpenBsdRcctlServiceManager{motor: motor}
	case pf.Name == "windows":
		osm = &WindowsServiceManager{motor: motor}
	case pf.Name == "alpine":
		osm = &AlpineOpenrcServiceManager{motor: motor}
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
