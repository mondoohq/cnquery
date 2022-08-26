package os

import (
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/os/services"
)

const (
	SERVICE_CACHE_DESCRIPTION = "description"
	SERVICE_CACHE_INSTALLED   = "installed"
	SERVICE_CACHE_TYPE        = "type"
	SERVICE_CACHE_RUNNING     = "running"
	SERVICE_CACHE_ENABLED     = "enabled"
	SERVICE_CACHE_MASKED      = "masked"
)

func (p *mqlService) init(args *resources.Args) (*resources.Args, Service, error) {
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

	obj, err := p.MotorRuntime.CreateResource("services")
	if err != nil {
		return nil, nil, err
	}
	services := obj.(Services)

	_, err = services.List()
	if err != nil {
		return nil, nil, err
	}

	c, ok := services.MqlResource().Cache.Load("_map")
	if !ok {
		return nil, nil, errors.New("cannot get map of services")
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
	(*args)["masked"] = false
	(*args)["type"] = ""

	return args, nil, nil
}

func (p *mqlService) id() (string, error) {
	return p.Name()
}

func (p *mqlService) GetDescription() (string, error) {
	_, ok := p.Cache.Load(SERVICE_CACHE_DESCRIPTION)
	if ok {
		return "", resources.NotReadyError{}
	}

	p.gatherServiceInfo(p.createCallback(SERVICE_CACHE_DESCRIPTION))
	return "", resources.NotReadyError{}
}

func (p *mqlService) GetInstalled() (bool, error) {
	_, ok := p.Cache.Load(SERVICE_CACHE_INSTALLED)
	if ok {
		return false, resources.NotReadyError{}
	}

	p.gatherServiceInfo(p.createCallback(SERVICE_CACHE_INSTALLED))
	return false, resources.NotReadyError{}
}

func (p *mqlService) GetRunning() (bool, error) {
	_, ok := p.Cache.Load(SERVICE_CACHE_RUNNING)
	if ok {
		return false, resources.NotReadyError{}
	}

	p.gatherServiceInfo(p.createCallback(SERVICE_CACHE_RUNNING))
	return false, resources.NotReadyError{}
}

func (p *mqlService) GetEnabled() (bool, error) {
	_, ok := p.Cache.Load(SERVICE_CACHE_ENABLED)
	if ok {
		return false, resources.NotReadyError{}
	}

	p.gatherServiceInfo(p.createCallback(SERVICE_CACHE_ENABLED))
	return false, resources.NotReadyError{}
}

func (p *mqlService) GetMasked() (bool, error) {
	_, ok := p.Cache.Load(SERVICE_CACHE_MASKED)
	if ok {
		return false, resources.NotReadyError{}
	}

	p.gatherServiceInfo(p.createCallback(SERVICE_CACHE_MASKED))
	return false, resources.NotReadyError{}
}

func (p *mqlService) GetType() (string, error) {
	_, ok := p.Cache.Load(SERVICE_CACHE_TYPE)
	if ok {
		return "", resources.NotReadyError{}
	}

	p.gatherServiceInfo(p.createCallback(SERVICE_CACHE_TYPE))
	return "", resources.NotReadyError{}
}

func (p *mqlService) createCallback(field string) ServiceCallbackTrigger {
	return func() {
		err := p.MotorRuntime.Observers.Trigger(p.MqlResource().FieldUID(field))
		if err != nil {
			log.Error().Err(err).Msg("[service]> failed to trigger field '" + field + "'")
		}
	}
}

type ServiceCallbackTrigger func()

func (p *mqlService) gatherServiceInfo(fn ServiceCallbackTrigger) error {
	name, err := p.Name()
	if err != nil {
		return err
	}

	obj, err := p.MotorRuntime.CreateResource("services")
	if err != nil {
		return err
	}
	services := obj.(Services)

	c, ok := services.MqlResource().Cache.Load("_map")
	if !ok {
		return errors.New("cannot get map of services")
	}
	cmap := c.Data.(map[string]Service)

	srv := cmap[name]
	if srv != nil {
		return errors.New("service does not exist")
	}

	p.Cache.Store("name", &resources.CacheEntry{Data: srv.Name, Valid: true, Timestamp: time.Now().Unix()})
	p.Cache.Store("description", &resources.CacheEntry{Data: srv.Description, Valid: true, Timestamp: time.Now().Unix()})
	p.Cache.Store("installed", &resources.CacheEntry{Data: srv.Installed, Valid: true, Timestamp: time.Now().Unix()})
	p.Cache.Store("enabled", &resources.CacheEntry{Data: srv.Enabled, Valid: true, Timestamp: time.Now().Unix()})
	p.Cache.Store("masked", &resources.CacheEntry{Data: srv.Masked, Valid: true, Timestamp: time.Now().Unix()})
	p.Cache.Store("running", &resources.CacheEntry{Data: srv.Running, Valid: true, Timestamp: time.Now().Unix()})
	p.Cache.Store("type", &resources.CacheEntry{Data: srv.Type, Valid: true, Timestamp: time.Now().Unix()})

	// call callback trigger
	if fn != nil {
		fn()
	}

	return nil
}

func (p *mqlServices) id() (string, error) {
	return "services", nil
}

func (p *mqlServices) GetList() ([]interface{}, error) {
	// find suitable service manager
	osm, err := services.ResolveManager(p.MotorRuntime.Motor)
	if osm == nil || err != nil {
		// there are valid cases where this error is happening, eg. you run a service query in
		// asset filters for non-supported transports
		log.Debug().Err(err).Msg("mql[services]> could not retrieve services list")
		return nil, errors.New("cannot find service manager")
	}

	// retrieve all system services
	services, err := osm.List()
	if err != nil {
		log.Debug().Err(err).Msg("mql[services]> could not retrieve service list")
		return nil, errors.New("could not retrieve service list")
	}
	log.Debug().Int("services", len(services)).Msg("mql[services]> running services")

	// convert to interface{}{}
	mqlSrvs := []interface{}{}
	namedMap := map[string]Service{}
	for i := range services {
		srv := services[i]

		mqlSrv, err := p.MotorRuntime.CreateResource("service",
			"name", srv.Name,
			"description", srv.Description,
			"installed", srv.Installed,
			"enabled", srv.Enabled,
			"masked", srv.Masked,
			"running", srv.Running,
			"type", srv.Type,
		)
		if err != nil {
			return nil, err
		}

		mqlSrvs = append(mqlSrvs, mqlSrv.(Service))
		namedMap[srv.Name] = mqlSrv.(Service)
	}

	p.Cache.Store("_map", &resources.CacheEntry{Data: namedMap})

	return mqlSrvs, nil
}
