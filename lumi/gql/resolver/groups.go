package resolver

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/gql"
	"go.mondoo.io/mondoo/lumi/resources/groups"
	motor "go.mondoo.io/mondoo/motor/motoros"
)

func osGroups(motor *motor.Motor) ([]*groups.Group, error) {
	// find suitable groups manager
	gm, err := groups.ResolveManager(motor)
	if gm == nil || err != nil {
		return nil, err
	}

	// retrieve all system groups
	return gm.List()
}

func (r *queryResolver) Groups(ctx context.Context) ([]gql.Group, error) {
	groups, err := osGroups(r.Runtime.Motor)
	if err != nil {
		log.Warn().Err(err).Msg("lumi[groups]> could not retrieve groups list")
		return nil, errors.New("could not retrieve groups list")
	}

	lumiGroups := []gql.Group{}
	for i := range groups {
		group := groups[i]

		lumiGroups = append(lumiGroups, gql.Group{
			Name: group.Name,
			Gid:  group.Gid,
		})
	}

	return lumiGroups, nil
}

func (r *queryResolver) Group(ctx context.Context, gidVal int) (*gql.Group, error) {
	gid := int64(gidVal)

	groups, err := osGroups(r.Runtime.Motor)
	if err != nil {
		log.Warn().Err(err).Msg("lumi[groups]> could not retrieve groups list")
		return nil, errors.New("could not retrieve groups list")
	}

	for i := range groups {
		group := groups[i]
		if group.Gid == gid {
			return &gql.Group{
				Name: group.Name,
				Gid:  group.Gid,
			}, nil
		}
	}

	return nil, nil
}
