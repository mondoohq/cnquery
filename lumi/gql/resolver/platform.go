package resolver

import (
	"context"

	"go.mondoo.io/mondoo/lumi/gql"
)

func (r *queryResolver) Platform(ctx context.Context) (*gql.Platform, error) {
	platform, err := r.Runtime.Motor.Platform()
	if err != nil {
		return nil, err
	}

	return &gql.Platform{
		Name:    platform.Name,
		Title:   platform.Title,
		Release: platform.Release,
		Arch:    platform.Arch,
		Family:  platform.Family,
	}, nil
}
