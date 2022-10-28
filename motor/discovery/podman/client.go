package podman

import (
	"context"

	"github.com/containers/common/pkg/config"
	"github.com/containers/podman/v4/pkg/bindings"
	"github.com/pkg/errors"
)

type podmanClient struct {
	conn context.Context
}

func NewPodmanDiscovery() (*podmanClient, error) {
	cfg, err := config.ReadCustomConfig()
	if err != nil {
		return nil, errors.Wrap(err, "podman config not found")
	}
	url, identity, isMachine, err := cfg.ActiveDestination()
	if err != nil {
		return nil, err
	}
	conn, err := bindings.NewConnectionWithIdentity(context.Background(), url, identity, isMachine)
	if err != nil {
		return nil, err
	}

	return &podmanClient{
		conn: conn,
	}, nil
}
