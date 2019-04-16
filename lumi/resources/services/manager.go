package services

import (
	"errors"

	"go.mondoo.io/mondoo/motor"
)

func ResolveManager(motor *motor.Motor) (OSServiceManager, error) {
	var osm OSServiceManager

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	switch platform.Name {
	case "manjaro", "arch": // arch family
		osm = &SystemDServiceManager{motor: motor}
	case "centos", "redhat": // redhat family
		osm = &SystemDServiceManager{motor: motor}
	case "ubuntu":
		osm = &SystemDServiceManager{motor: motor}
	case "debian":
		osm = &SystemDServiceManager{motor: motor}
	case "mac_os_x", "darwin":
		osm = &LaunchDServiceManager{motor: motor}
	}

	return osm, nil
}

type OSServiceManager interface {
	Name() string
	Service(name string) (*Service, error)
	List() ([]*Service, error)
}

// Newer linux systems use systemd as service manager
type SystemDServiceManager struct {
	motor *motor.Motor
}

func (s *SystemDServiceManager) Name() string {
	return "systemd Service Manager"
}

func (s *SystemDServiceManager) Service(name string) (*Service, error) {
	services, err := s.List()
	if err != nil {
		return nil, err
	}

	// iterate over list and search for the service
	for i := range services {
		service := services[i]
		if service.Name == name {
			return service, nil
		}
	}

	return nil, errors.New("service> " + name + " does not exist")
}

func (s *SystemDServiceManager) List() ([]*Service, error) {
	c, err := s.motor.Transport.RunCommand("systemctl --all list-units")
	if err != nil {
		return nil, err
	}
	return ParseServiceSystemDUnitFiles(c.Stdout)
}

// MacOS is using launchd as default service manager
type LaunchDServiceManager struct {
	motor *motor.Motor
}

func (s *LaunchDServiceManager) Name() string {
	return "launchd Service Manager"
}

func (s *LaunchDServiceManager) Service(name string) (*Service, error) {
	services, err := s.List()
	if err != nil {
		return nil, err
	}

	// iterate over list and search for the service
	for i := range services {
		service := services[i]
		if service.Name == name {
			return service, nil
		}
	}

	return nil, errors.New("service> " + name + " does not exist")
}

func (s *LaunchDServiceManager) List() ([]*Service, error) {
	c, err := s.motor.Transport.RunCommand("launchctl list")
	if err != nil {
		return nil, err
	}
	return ParseServiceLaunchD(c.Stdout)
}
