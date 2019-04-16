package resolver

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/gql"
	"go.mondoo.io/mondoo/lumi/resources/users"
)

func (r *queryResolver) Users(ctx context.Context) ([]gql.User, error) {
	// find suitable user manager
	um, err := users.ResolveManager(r.Runtime.Motor)
	if um == nil || err != nil {
		log.Warn().Err(err).Msg("lumi[users]> could not retrieve users list")
		return nil, errors.New("cannot find users manager")
	}

	// retrieve all system users
	users, err := um.List()
	if err != nil {
		log.Warn().Err(err).Msg("lumi[users]> could not retrieve users list")
		return nil, errors.New("could not retrieve users list")
	}
	log.Debug().Int("users", len(users)).Msg("lumi[users]> found users")

	lumiUsers := []gql.User{}
	for i := range users {
		user := users[i]

		if user != nil {
			lumiUsers = append(lumiUsers, gql.User{
				Uid:         user.Uid,
				Gid:         user.Gid,
				Username:    user.Username,
				Description: user.Description,
				Shell:       user.Shell,
				Home:        user.Home,
				Enabled:     user.Enabled,
			})
		}
	}

	return lumiUsers, nil
}

func (r *queryResolver) User(ctx context.Context, uidVal int) (*gql.User, error) {
	uid := int64(uidVal)
	um, err := users.ResolveManager(r.Runtime.Motor)
	if err != nil {
		return nil, errors.New("user> cannot find user manager")
	}

	// search for the user
	user, err := um.User(uid)
	if err != nil {
		return nil, err
	}

	return &gql.User{
		Uid:         user.Uid,
		Gid:         user.Gid,
		Username:    user.Username,
		Description: user.Description,
		Shell:       user.Shell,
		Home:        user.Home,
		Enabled:     user.Enabled,
	}, nil
}
