package resources

import (
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/services"
)

const (
	SERVICE_CACHE_DESCRIPTION = "description"
	SERVICE_CACHE_INSTALLED   = "installed"
	SERVICE_CACHE_TYPE        = "type"
	SERVICE_CACHE_RUNNING     = "running"
	SERVICE_CACHE_ENABLED     = "enabled"
)

func (p *lumiService) init(args *lumi.Args) (*lumi.Args, Service, error) {
	// verify that a service with that name exist
	rawName, ok := (*args)["name"]
	if !ok {
		return args, nil, nil
	}
	name, ok := rawName.(string)
	if !ok {
		return args, nil, errors.New("name has invalid type")
	}

	// check if additional information is already provided,
	// this let us abort testing if provided by a list
	if _, iok := (*args)["installed"]; iok {
		return args, nil, nil
	}

	obj, err := p.Runtime.CreateResource("services")
	if err != nil {
		return nil, nil, err
	}
	services := obj.(Services)

	c, ok := services.LumiResource().Cache.Load("_map")
	if !ok {
		return nil, nil, errors.New("Cannot get map of packages")
	}
	cmap := c.Data.(map[string]Service)

	srv := cmap[name]
	if srv != nil {
		return nil, srv, nil
	}

	// if the service doesn't exist, init it to empty
	(*args)["description"] = ""
	(*args)["installed"] = false
	(*args)["running"] = false
	(*args)["enabled"] = false
	(*args)["type"] = ""

	return args, nil, nil
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

	p.gatherServiceInfo(p.createCallback(SERVICE_CACHE_ENABLED))

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
			log.Error().Err(err).Msg("[service]> failed to trigger field '" + field + "'")
		}
	}
}

type ServiceCallbackTrigger func()

func (p *lumiService) gatherServiceInfo(fn ServiceCallbackTrigger) error {
	name, err := p.Name()
	if err != nil {
		return errors.New("cannot gather service name")
	}

	osm, err := services.ResolveManager(p.Runtime.Motor)
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

func (p *lumiServices) id() (string, error) {
	return "services", nil
}

func (p *lumiServices) GetList() ([]interface{}, error) {
	// find suitable service manager
	osm, err := services.ResolveManager(p.Runtime.Motor)
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

	// convert to interface{}{}
	lumiSrvs := []interface{}{}
	namedMap := map[string]Service{}
	for i := range services {
		srv := services[i]

		lumiSrv, err := p.Runtime.CreateResource("service",
			"name", srv.Name,
			"description", srv.Description,
			"installed", srv.Installed,
			"enabled", srv.Enabled,
			"running", srv.Running,
			"type", srv.Type,
		)
		if err != nil {
			return nil, err
		}

		lumiSrvs = append(lumiSrvs, lumiSrv.(Service))
		namedMap[srv.Name] = lumiSrv.(Service)
	}

	p.Cache.Store("_map", &lumi.CacheEntry{Data: namedMap})

	return lumiSrvs, nil
}
