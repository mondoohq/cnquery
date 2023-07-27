package resources

import (
	"errors"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
	"go.mondoo.com/cnquery/providers/os/resources/services"
)

func initService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) != 1 {
		return args, nil, nil
	}

	x, ok := args["name"]
	if !ok {
		return nil, nil, errors.New("cannot initialize service, need at least a name to look it up")
	}

	name := x.Value.(string)
	if name == "" {
		return nil, nil, errors.New("cannot look for a service with an empty name")
	}

	raw, err := CreateResource(runtime, "services", nil)
	if err != nil {
		return nil, nil, err
	}
	services := raw.(*mqlServices)

	if err := services.refreshCache(nil); err != nil {
		return nil, nil, err
	}

	srv := services.namedServices[name]
	if srv == nil {
		return nil, nil, errors.New("service '" + name + "' does not exist")
	}

	return nil, srv, nil
}

func (p *mqlService) id() (string, error) {
	return p.Name.Data, nil
}

type mqlServicesInternal struct {
	lock          sync.Mutex
	namedServices map[string]*mqlService
}

func (p *mqlServices) list() ([]interface{}, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	// find suitable service manager
	conn := p.MqlRuntime.Connection.(shared.Connection)
	osm, err := services.ResolveManager(conn)
	if osm == nil || err != nil {
		// there are valid cases where this error is happening, eg. you run a service query in
		// asset filters for non-supported providers
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

	for i := range services {
		srv := services[i]

		mqlSrv, err := CreateResource(p.MqlRuntime, "service", map[string]*llx.RawData{
			"name":        llx.StringData(srv.Name),
			"description": llx.StringData(srv.Description),
			"installed":   llx.BoolData(srv.Installed),
			"enabled":     llx.BoolData(srv.Enabled),
			"masked":      llx.BoolData(srv.Masked),
			"running":     llx.BoolData(srv.Running),
			"type":        llx.StringData(srv.Type),
		})
		if err != nil {
			return nil, err
		}

		mqlSrvs = append(mqlSrvs, mqlSrv.(*mqlService))
	}

	return mqlSrvs, p.refreshCache(mqlSrvs)
}

func (p *mqlServices) refreshCache(all []interface{}) error {
	if all == nil {
		raw := p.GetList()
		if raw.Error != nil {
			return raw.Error
		}
		all = raw.Data
	}

	namedMap := map[string]*mqlService{}
	for i := range all {
		service := all[i].(*mqlService)
		namedMap[service.Name.Data] = service
	}
	p.namedServices = namedMap
	return nil
}
