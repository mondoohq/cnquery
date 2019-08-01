package resolver

import (
	"context"
	"errors"

	"go.mondoo.io/mondoo/lumi/gql"
	"go.mondoo.io/mondoo/lumi/resources/sshd"
)

func (r *queryResolver) Sshd(ctx context.Context) (*gql.Sshd, error) {
	return &gql.Sshd{}, nil
}

type sshdResolver struct{ *Resolver }

func (r *sshdResolver) Config(ctx context.Context, obj *gql.Sshd, path *string) (*gql.SSHConfig, error) {
	if obj == nil {
		return nil, errors.New("no parent object defined")
	}

	configPath := "/etc/ssh/sshd_config"
	if path != nil {
		configPath = *path
	}

	return &gql.SSHConfig{
		Path: configPath,
	}, nil
}

type sshConfigResolver struct{ *Resolver }

func (r *sshConfigResolver) File(ctx context.Context, obj *gql.SSHConfig) (*gql.File, error) {
	if obj == nil {
		return nil, errors.New("no parent object defined")
	}

	return &gql.File{
		Path: &obj.Path,
	}, nil
}

func (r *sshConfigResolver) Params(ctx context.Context, obj *gql.SSHConfig) ([]*gql.KeyValue, error) {
	if obj == nil {
		return nil, errors.New("no parent object defined")
	}

	f, err := r.Runtime.Motor.Transport.File(obj.Path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sshParams, err := sshd.Params(f)
	if err != nil {
		return nil, err
	}

	entries := []*gql.KeyValue{}
	for k := range sshParams {
		// we need a local variable, otherwise go reuses the k pointer
		key := k
		val := sshParams[k]
		entries = append(entries, &gql.KeyValue{
			Key:   &key,
			Value: &val,
		})
	}

	return entries, nil
}
