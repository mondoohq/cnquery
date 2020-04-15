package resources

import (
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/services"
	motor "go.mondoo.io/mondoo/motor/motoros"
)

const (
	SERVICE_CACHE_DESCRIPTION = "description"
	SERVICE_CACHE_INSTALLED   = "installed"
	SERVICE_CACHE_TYPE        = "type"
	SERVICE_CACHE_RUNNING     = "running"
	SERVICE_CACHE_ENABLED     = "enabled"
)

func (p *lumiService) init(args *lumi.Args) (*lumi.Args, error) {
	// verify that a service with that name exist
	nameValue, ok := (*args)["name"]

	// check if additional information is already provided,
	// this let us abort testing if provided by a list
	_, iok := (*args)["installed"]

	// if ame was provided, lets collect the info
	if ok && !iok {
		name, ok := nameValue.(string)
		if !ok {
			return nil, errors.New("name has invalid type")
		}

		osm, err := resolveOSServiceManager(p.Runtime.Motor)
		if err != nil {
			return nil, errors.New("cannot find service manager")
		}

		_, err = osm.Service(name)
		if err != nil {
			return nil, errors.New("service " + name + " does not exist")
		}
	}
	return args, nil
}

func (p *lumiService) id() (string, error) {
	return p.Name()
}

func (p *lumiService) GetDescription() (string, error) {
	_, ok := p.Cache.Load(SERVICE_CACHE_DESCRIPTION)
	if ok {
		return "", lumi.NotReadyError{}
	}

	p.gatherServiceInfo(p.createCallback(SERVICE_CACHE_DESCRIPTION))

	return "", lumi.NotReadyError{}
}

func (p *lumiService) GetInstalled() (bool, error) {
	_, ok := p.Cache.Load(SERVICE_CACHE_INSTALLED)
	if ok {
		return false, lumi.NotReadyError{}
	}

	p.gatherServiceInfo(p.createCallback(SERVICE_CACHE_INSTALLED))

	return false, lumi.NotReadyError{}
}

func (p *lumiService) GetRunning() (bool, error) {
	_, ok := p.Cache.Load(SERVICE_CACHE_RUNNING)
	if ok {
		return false, lumi.NotReadyError{}
	}

	p.gatherServiceInfo(p.createCallback(SERVICE_CACHE_RUNNING))

	return false, lumi.NotReadyError{}
}

func (p *lumiService) GetEnabled() (bool, error) {
	_, ok := p.Cache.Load(SERVICE_CACHE_ENABLED)
	if ok {
		return false, lumi.NotReadyError{}
	}

	p.gatherServiceInfo(p.createCallback(SERVICE_CACHE_RUNNING))

	return false, lumi.NotReadyError{}
}

func (p *lumiService) GetType() (string, error) {
	_, ok := p.Cache.Load(SERVICE_CACHE_TYPE)
	if ok {
		return "", lumi.NotReadyError{}
	}

	p.gatherServiceInfo(p.createCallback(SERVICE_CACHE_TYPE))

	return "", lumi.NotReadyError{}
}

func (p *lumiService) createCallback(field string) ServiceCallbackTrigger {
	return func() {
		err := p.Runtime.Observers.Trigger(p.LumiResource().FieldUID(field))
		if err != nil {
			log.Error().Err(err).Msg("[service]> failed to trigger " + field)
		}
	}
}

type ServiceCallbackTrigger func()

func (p *lumiService) gatherServiceInfo(fn ServiceCallbackTrigger) error {
	name, err := p.Name()
	if err != nil {
		return errors.New("cannot gather service name")
	}

	osm, err := resolveOSServiceManager(p.Runtime.Motor)
	if err != nil {
		return errors.New("cannot find service manager")
	}

	service, err := osm.Service(name)
	if err != nil {
		return errors.New("cannot gather service details")
	}

	p.Cache.Store("name", &lumi.CacheEntry{Data: service.Name, Valid: true, Timestamp: time.Now().Unix()})
	p.Cache.Store("description", &lumi.CacheEntry{Data: service.Description, Valid: true, Timestamp: time.Now().Unix()})
	p.Cache.Store("installed", &lumi.CacheEntry{Data: service.Installed, Valid: true, Timestamp: time.Now().Unix()})
	p.Cache.Store("enabled", &lumi.CacheEntry{Data: service.Enabled, Valid: true, Timestamp: time.Now().Unix()})
	p.Cache.Store("running", &lumi.CacheEntry{Data: service.Running, Valid: true, Timestamp: time.Now().Unix()})
	p.Cache.Store("type", &lumi.CacheEntry{Data: service.Type, Valid: true, Timestamp: time.Now().Unix()})

	// call callback trigger
	if fn != nil {
		fn()
	}

	return nil
}

func (p *lumiServices) init(args *lumi.Args) (*lumi.Args, error) {
	return args, nil
}

func (p *lumiServices) id() (string, error) {
	return "services", nil
}

func (s *lumiServices) GetList() ([]interface{}, error) {
	// find suitable service manager
	osm, err := resolveOSServiceManager(s.Runtime.Motor)
	if osm == nil || err != nil {
		log.Warn().Err(err).Msg("lumi[services]> could not retrieve services list")
		return nil, errors.New("cannot find service manager")
	}

	// retrieve all system services
	services, err := osm.List()
	if err != nil {
		log.Warn().Err(err).Msg("lumi[services]> could not retrieve service list")
		return nil, errors.New("could not retrieve service list")
	}
	log.Debug().Int("services", len(services)).Msg("lumi[services]> running services")

	// convert to ]interface{}{}
	lumiSrvs := []interface{}{}
	for i := range services {
		srv := services[i]

		// set init arguments for the lumi package resource
		args := make(lumi.Args)
		args["name"] = srv.Name
		args["description"] = srv.Description
		args["installed"] = srv.Installed
		args["enabled"] = srv.Enabled
		args["running"] = srv.Running
		args["type"] = srv.Type

		e, err := newService(s.Runtime, &args)
		if err != nil {
			log.Error().Err(err).Str("service", srv.Name).Msg("lumi[services]> could not create service resource")
			continue
		}

		lumiSrvs = append(lumiSrvs, e.(Service))
	}

	return lumiSrvs, nil
}

func resolveOSServiceManager(motor *motor.Motor) (OSServiceManager, error) {
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
	Service(name string) (*services.Service, error)
	List() ([]*services.Service, error)
}

// Newer linux systems use systemd as service manager
type SystemDServiceManager struct {
	motor *motor.Motor
}

func (s *SystemDServiceManager) Name() string {
	return "systemd Service Manager"
}

func (s *SystemDServiceManager) Service(name string) (*services.Service, error) {
	serviceList, err := s.List()
	if err != nil {
		return nil, err
	}

	// iterate over list and search for the service
	for i := range serviceList {
		service := serviceList[i]
		if service.Name == name {
			return service, nil
		}
	}

	return nil, errors.New("service> " + name + " does not exist")
}

func (s *SystemDServiceManager) List() ([]*services.Service, error) {
	c, err := s.motor.Transport.RunCommand("systemctl --all list-units")
	if err != nil {
		return nil, err
	}
	return services.ParseServiceSystemDUnitFiles(c.Stdout)
}

// MacOS is using launchd as default service manager
type LaunchDServiceManager struct {
	motor *motor.Motor
}

func (s *LaunchDServiceManager) Name() string {
	return "launchd Service Manager"
}

func (s *LaunchDServiceManager) Service(name string) (*services.Service, error) {
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

func (s *LaunchDServiceManager) List() ([]*services.Service, error) {
	c, err := s.motor.Transport.RunCommand("launchctl list")
	if err != nil {
		return nil, err
	}
	return services.ParseServiceLaunchD(c.Stdout)
}
