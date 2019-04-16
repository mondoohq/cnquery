package resolver

import (
	"context"

	"go.mondoo.io/mondoo"
	"go.mondoo.io/mondoo/lumi/gql"
)

func New() *Resolver {
	return &Resolver{}
}

type Resolver struct{}

func (r *Resolver) Query() gql.QueryResolver {
	return &queryResolver{r}
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Mondoo(ctx context.Context) (*gql.Mondoo, error) {
	return &gql.Mondoo{
		Build:   mondoo.GetBuild(),
		Version: mondoo.GetVersion(),
	}, nil
}
