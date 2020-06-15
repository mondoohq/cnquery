package services

import (
	"errors"

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
	Service(name string) (*Service, error)
	List() ([]*Service, error)
}

func ResolveManager(motor *motor.Motor) (OSServiceManager, error) {
	var osm OSServiceManager

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	switch platform.Name {
	case "manjaro", "arch": // arch family
		osm = &SystemDServiceManager{motor: motor}
	case "centos", "redhat", "amzn", "ol":
		// TODO: centos6 and amaz1 do not use systemd
		osm = &SystemDServiceManager{motor: motor}
	case "ubuntu":
		osm = &SystemDServiceManager{motor: motor}
	case "debian":
		osm = &SystemDServiceManager{motor: motor}
	case "mac_os_x", "darwin":
		osm = &LaunchDServiceManager{motor: motor}
	case "freebsd":
		osm = &BsdInitServiceManager{motor: motor}
	case "windows":
		osm = &WindowsServiceManager{motor: motor}
	}

	if osm == nil {
		return nil, errors.New("could not detect suitable service manager for platform: " + platform.Name)
	}

	return osm, nil
}

func findService(services []*Service, name string) (*Service, error) {
	// search for name
	for i := range services {
		service := services[i]
		if service.Name == name {
			return service, nil
		}
	}

	return nil, errors.New("service> " + name + " does not exist")
}
