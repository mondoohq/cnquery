package resolver

import (
	"context"

	"go.mondoo.io/mondoo"
	"go.mondoo.io/mondoo/lumi/gql"
)

func (r *queryResolver) Mondoo(ctx context.Context) (*gql.Mondoo, error) {
	return &gql.Mondoo{
		Build:   mondoo.GetBuild(),
		Version: mondoo.GetVersion(),
	}, nil
}
