package podman

import (
	"context"

	"github.com/containers/podman/v4/pkg/bindings"
)

type podmanClient struct {
	conn context.Context
}

func NewPodmanDiscovery() (*podmanClient, error) {
	conn, err := bindings.NewConnection(context.Background(), "unix://run/podman/podman.sock")
	if err != nil {
		return nil, err
	}

	return &podmanClient{
		conn: conn,
	}, nil
}
