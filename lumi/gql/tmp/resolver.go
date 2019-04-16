package tmp

import (
	"context"

	"go.mondoo.io/mondoo/lumi/gql"
)

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

type Resolver struct{}

func (r *Resolver) Query() gql.QueryResolver {
	return &queryResolver{r}
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Mondoo(ctx context.Context) (*gql.Mondoo, error) {
	panic("not implemented")
}
