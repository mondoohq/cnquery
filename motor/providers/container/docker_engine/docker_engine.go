package docker_engine

import (
	"context"
	"errors"

	"go.mondoo.io/mondoo/motor/providers/os"

	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/ssh/cat"
)

var (
	_ providers.Instance           = (*Provider)(nil)
	_ providers.PlatformIdentifier = (*Provider)(nil)
)

func New(container string) (*Provider, error) {
	// TODO: harmonize docker client establishment with docker engine discovery
	dockerClient, err := GetDockerClient()
	if err != nil {
		return nil, err
	}

	// check if we are having container
	data, err := dockerClient.ContainerInspect(context.Background(), container)
	if err != nil {
		return nil, errors.New("cannot find container " + container)
	}

	if !data.State.Running {
		return nil, errors.New("container " + data.ID + " is not running")
	}

	t := &Provider{
		dockerClient: dockerClient,
		container:    container,
		kind:         providers.Kind_KIND_CONTAINER,
		runtime:      providers.RUNTIME_DOCKER_CONTAINER,
	}
	t.Fs = &FS{
		dockerClient: t.dockerClient,
		Container:    t.container,
		Transport:    t,
		catFS:        cat.New(t),
	}
	return t, nil
}

type Provider struct {
	dockerClient *client.Client
	container    string
	Fs           *FS

	PlatformIdentifier string
	// optional metadata to store additional information
	Metadata struct {
		Name   string
		Labels map[string]string
	}

	kind    providers.Kind
	runtime string
}

func (p *Provider) DockerClient() *client.Client {
	return p.dockerClient
}

func (p *Provider) ContainerId() string {
	return p.container
}

func (p *Provider) Identifier() (string, error) {
	return p.PlatformIdentifier, nil
}

func (p *Provider) Labels() map[string]string {
	return p.Metadata.Labels
}

func (p *Provider) PlatformName() string {
	return p.Metadata.Name
}

func (p *Provider) RunCommand(command string) (*os.Command, error) {
	log.Debug().Str("command", command).Msg("docker> run command")
	c := &Command{dockerClient: p.dockerClient, Container: p.container}
	res, err := c.Exec(command)
	return res, err
}

func (p *Provider) FS() afero.Fs {
	return p.Fs
}

func (p *Provider) FileInfo(path string) (os.FileInfoDetails, error) {
	fs := p.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return os.FileInfoDetails{}, err
	}

	uid := int64(-1)
	gid := int64(-1)
	mode := stat.Mode()

	return os.FileInfoDetails{
		Mode: os.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (p *Provider) Close() {
	p.dockerClient.Close()
}

func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		providers.Capability_RunCommand,
		providers.Capability_File,
	}
}

func GetDockerClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	cli.NegotiateAPIVersion(context.Background())
	return cli, nil
}

func (p *Provider) Kind() providers.Kind {
	return p.kind
}

func (p *Provider) Runtime() string {
	return p.runtime
}

func (p *Provider) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}
