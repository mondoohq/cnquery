package connection

import (
	"context"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
)

const (
	DockerContainer shared.ConnectionType = "docker-container"
)

var _ shared.Connection = &DockerContainerConnection{}

type DockerContainerConnection struct {
	id    uint32
	asset *inventory.Asset

	Client      *client.Client
	ContainerID string
}

func NewDockerContainerConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) (*DockerContainerConnection, error) {
	// expect unix shell by default

	panic("Not yet migrated")

	return nil, nil
}

func GetDockerClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	cli.NegotiateAPIVersion(context.Background())
	return cli, nil
}

func (p *DockerContainerConnection) ID() uint32 {
	return p.id
}

func (p *DockerContainerConnection) Name() string {
	return string(DockerContainer)
}

func (p *DockerContainerConnection) Type() shared.ConnectionType {
	return DockerContainer
}

func (p *DockerContainerConnection) Asset() *inventory.Asset {
	return p.asset
}

func (p *DockerContainerConnection) Capabilities() shared.Capabilities {
	return shared.Capability_File | shared.Capability_RunCommand
}

func (p *DockerContainerConnection) FileInfo(path string) (shared.FileInfoDetails, error) {
	return shared.FileInfoDetails{}, errors.New("Not yet migrated")
}

func (p *DockerContainerConnection) FileSystem() afero.Fs {
	panic("not yet migerated")
	return afero.NewOsFs()
}

func (p *DockerContainerConnection) RunCommand(command string) (*shared.Command, error) {
	return nil, errors.New("not yet migrated")
}
