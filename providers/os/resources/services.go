// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/services"
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

	cleanServiceName := strings.TrimSuffix(name, ".service")

	if srv, ok := services.namedServices[cleanServiceName]; ok {
		return nil, srv, nil
	}

	res := &mqlService{}
	res.Name = plugin.TValue[string]{Data: name, State: plugin.StateIsSet}
	res.Description.State = plugin.StateIsSet | plugin.StateIsNull
	res.Installed = plugin.TValue[bool]{Data: false, State: plugin.StateIsSet}
	res.Running = plugin.TValue[bool]{Data: false, State: plugin.StateIsSet}
	res.Type.State = plugin.StateIsSet | plugin.StateIsNull
	res.Enabled = plugin.TValue[bool]{Data: false, State: plugin.StateIsSet}
	res.Masked = plugin.TValue[bool]{Data: false, State: plugin.StateIsSet}
	return nil, res, nil
}

func (x *mqlService) id() (string, error) {
	return x.Name.Data, nil
}

type mqlServicesInternal struct {
	lock          sync.Mutex
	namedServices map[string]*mqlService
}

func (x *mqlServices) list() ([]interface{}, error) {
	x.lock.Lock()
	defer x.lock.Unlock()

	// find suitable service manager
	conn := x.MqlRuntime.Connection.(shared.Connection)
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

		mqlSrv, err := CreateResource(x.MqlRuntime, "service", map[string]*llx.RawData{
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

	return mqlSrvs, x.refreshCache(mqlSrvs)
}

func (x *mqlServices) refreshCache(all []interface{}) error {
	if all == nil {
		raw := x.GetList()
		if raw.Error != nil {
			return raw.Error
		}
		all = raw.Data
	}

	x.namedServices = map[string]*mqlService{}
	for i := range all {
		service := all[i].(*mqlService)
		x.namedServices[service.Name.Data] = service
	}

	return nil
}
