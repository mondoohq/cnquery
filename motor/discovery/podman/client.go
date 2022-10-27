package podman

import (
	"context"

	"github.com/containers/podman/v4/pkg/bindings"
)

type podmanClient struct {
	conn context.Context
}

func NewPodmanDiscovery() (*podmanClient, error) {
	conn, err := bindings.NewConnectionWithIdentity(context.Background(), "socket here", "identity", true)
	if err != nil {
		return nil, err
	}

	return &podmanClient{
		conn: conn,
	}, nil
}
