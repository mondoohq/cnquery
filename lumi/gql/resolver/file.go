package resolver

import (
	"context"
	"errors"

	"go.mondoo.io/mondoo/lumi/gql"
	"go.mondoo.io/mondoo/motor/motorutil"
)

func (r *queryResolver) File(ctx context.Context, path *string) (*gql.File, error) {
	return &gql.File{
		Path: path,
	}, nil
}

type fileResolver struct{ *Resolver }

func (r *fileResolver) Content(ctx context.Context, obj *gql.File) (*string, error) {
	if obj == nil || obj.Path == nil {
		return nil, errors.New("invalid argument, file and path cannot be nil")
	}

	path := *obj.Path
	f, err := r.Runtime.Motor.Transport.File(path)
	if err != nil {
		return nil, err
	}

	c, err := motorutil.ReadFile(f)
	if err != nil {
		return nil, err
	}

	content := string(c)
	return &content, nil
}

func (r *fileResolver) Exists(ctx context.Context, obj *gql.File) (bool, error) {
	if obj == nil || obj.Path == nil {
		return false, errors.New("invalid argument, file and path cannot be nil")
	}

	path := *obj.Path
	f, err := r.Runtime.Motor.Transport.File(path)
	if err != nil {
		return false, err
	}
	return f.Exists(), nil
}
