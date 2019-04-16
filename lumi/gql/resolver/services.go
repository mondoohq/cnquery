package resolver

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/gql"
	"go.mondoo.io/mondoo/lumi/resources/services"
)

func (r *queryResolver) Services(ctx context.Context) ([]gql.Service, error) {
	// find suitable service manager
	osm, err := services.ResolveManager(r.Runtime.Motor)
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

	lumiSrvs := []gql.Service{}
	for i := range services {
		srv := services[i]

		lumiSrvs = append(lumiSrvs, gql.Service{
			Name:        srv.Name,
			Description: srv.Description,
			Installed:   srv.Installed,
			Running:     srv.Running,
			Enabled:     srv.Enabled,
			Type:        srv.Type,
		})
	}

	return lumiSrvs, nil
}

func (r *queryResolver) Service(ctx context.Context, name string) (*gql.Service, error) {
	osm, err := services.ResolveManager(r.Runtime.Motor)
	if err != nil {
		return nil, errors.New("cannot find service manager")
	}

	_, err = osm.Service(name)
	if err != nil {
		return nil, errors.New("service " + name + " does not exist")
	}

	return &gql.Service{}, nil
}
