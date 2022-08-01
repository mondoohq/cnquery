package docker_engine

import (
	"context"
	"errors"

	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/ssh/cat"
)

var (
	_ providers.Transport                   = (*Transport)(nil)
	_ providers.TransportPlatformIdentifier = (*Transport)(nil)
)

func New(container string) (*Transport, error) {
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

	t := &Transport{
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

type Transport struct {
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

func (t *Transport) DockerClient() *client.Client {
	return t.dockerClient
}

func (t *Transport) ContainerId() string {
	return t.container
}

func (t *Transport) Identifier() (string, error) {
	return t.PlatformIdentifier, nil
}

func (t *Transport) Labels() map[string]string {
	return t.Metadata.Labels
}

func (t *Transport) PlatformName() string {
	return t.Metadata.Name
}

func (t *Transport) RunCommand(command string) (*providers.Command, error) {
	log.Debug().Str("command", command).Msg("docker> run command")
	c := &Command{dockerClient: t.dockerClient, Container: t.container}
	res, err := c.Exec(command)
	return res, err
}

func (t *Transport) FS() afero.Fs {
	return t.Fs
}

func (t *Transport) FileInfo(path string) (providers.FileInfoDetails, error) {
	fs := t.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return providers.FileInfoDetails{}, err
	}

	uid := int64(-1)
	gid := int64(-1)
	mode := stat.Mode()

	return providers.FileInfoDetails{
		Mode: providers.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (t *Transport) Close() {
	t.dockerClient.Close()
}

func (t *Transport) Capabilities() providers.Capabilities {
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

func (t *Transport) Kind() providers.Kind {
	return t.kind
}

func (t *Transport) Runtime() string {
	return t.runtime
}

func (t *Transport) PlatformIdDetectors() []providers.PlatformIdDetector {
	return []providers.PlatformIdDetector{
		providers.TransportPlatformIdentifierDetector,
	}
}
