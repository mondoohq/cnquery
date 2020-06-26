package services

import (
	"errors"
	"regexp"
	"strconv"

	motor "go.mondoo.io/mondoo/motor/motoros"
)

type Service struct {
	Name        string
	Description string
	State       State
	Type        string
	Installed   bool
	Running     bool
	Enabled     bool
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

var amazonlinux1version = regexp.MustCompile(`^20\d\d`)

func ResolveManager(motor *motor.Motor) (OSServiceManager, error) {
	var osm OSServiceManager

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	switch platform.Name {
	case "manjaro", "arch": // arch family
		osm = &SystemDServiceManager{motor: motor}
	case "amzn":
		if amazonlinux1version.MatchString(platform.Release) {
			osm = &UpstartServiceManager{SysVServiceManager{motor: motor}}
		} else {
			osm = &SystemDServiceManager{motor: motor}
		}
	case "centos", "redhat", "scientific", "ol":
		v, err := strconv.ParseFloat(platform.Release, 32)
		if err != nil {
			return nil, errors.New("unknown redhat version: " + platform.Release)
		}
		if v < 7 {
			osm = &UpstartServiceManager{SysVServiceManager{motor: motor}}
		} else {
			osm = &SystemDServiceManager{motor: motor}
		}
	case "fedora":
		v, err := strconv.ParseFloat(platform.Release, 32)
		if err != nil {
			return nil, errors.New("unknown fedora version: " + platform.Release)
		}

		if v < 15 {
			// upstart is only used since fedora 11 but we do not support those older versions
			osm = &UpstartServiceManager{SysVServiceManager{motor: motor}}
		} else {
			osm = &SystemDServiceManager{motor: motor}
		}
	case "ubuntu":
		v, err := strconv.ParseFloat(platform.Release, 32)
		if err != nil {
			return nil, errors.New("unknown ubuntu version: " + platform.Release)
		}

		if v < 15.04 {
			osm = &UpstartServiceManager{SysVServiceManager{motor: motor}}
		} else {
			osm = &SystemDServiceManager{motor: motor}
		}
	case "debian":
		v, err := strconv.ParseFloat(platform.Release, 32)
		if err != nil {
			return nil, errors.New("unknown debian version: " + platform.Release)
		}

		if v < 7 {
			osm = &SysVServiceManager{motor: motor}
		} else {
			osm = &SystemDServiceManager{motor: motor}
		}
	case "suse", "opensuse":
		v, err := strconv.ParseFloat(platform.Release, 32)
		if err != nil {
			return nil, errors.New("unknown suse version: " + platform.Release)
		}

		if v < 12 {
			osm = &SysVServiceManager{motor: motor}
		} else {
			osm = &SystemDServiceManager{motor: motor}
		}
	case "mac_os_x", "darwin":
		osm = &LaunchDServiceManager{motor: motor}
	case "freebsd", "dragonflybsd", "netbsd":
		osm = &BsdInitServiceManager{motor: motor}
	case "openbsd":
		osm = &OpenBsdRcctlServiceManager{motor: motor}
	case "windows":
		osm = &WindowsServiceManager{motor: motor}
	}

	if osm == nil {
		return nil, errors.New("could not detect suitable service manager for platform: " + platform.Name)
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
