package connection

import (
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
)

const (
	DockerSnapshot shared.ConnectionType = "docker-snapshot"
)

type DockerSnapshotConnection struct {
	TarConnection
}

func NewDockerSnapshotConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) (*DockerSnapshotConnection, error) {
	// expect unix shell by default
	res := DockerSnapshotConnection{
		TarConnection: TarConnection{
			id:    id,
			asset: asset,
		},
	}

	panic("Not yet migrated")

	return &res, nil
}

func (p *DockerSnapshotConnection) ID() uint32 {
	return p.id
}

func (p *DockerSnapshotConnection) Name() string {
	return string(DockerSnapshot)
}

func (p *DockerSnapshotConnection) Type() shared.ConnectionType {
	return DockerSnapshot
}
