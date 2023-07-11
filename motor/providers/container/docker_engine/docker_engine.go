package docker_engine

import (
	"context"
	"errors"
	"io"

	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/motor/providers"
	os_provider "go.mondoo.com/cnquery/motor/providers/os"
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

	// this can later be used for containers build from scratch
	serverVersion, err := dockerClient.ServerVersion(context.Background())
	if err != nil {
		log.Debug().Err(err).Msg("docker> cannot get server version")
	} else {
		log.Debug().Interface("serverVersion", serverVersion).Msg("docker> server version")
		t.PlatformArchitecture = serverVersion.Arch
	}

	t.Fs = &FS{
		dockerClient: t.dockerClient,
		Container:    t.container,
		Provider:     t,
		// catFS:        cat.New(t),
	}
	return t, nil
}

type Provider struct {
	dockerClient *client.Client
	container    string
	Fs           *FS

	PlatformIdentifier   string
	PlatformArchitecture string
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

func (p *Provider) RunCommand(command string) (*os_provider.Command, error) {
	log.Debug().Str("command", command).Msg("docker> run command")
	c := &Command{dockerClient: p.dockerClient, Container: p.container}
	res, err := c.Exec(command)
	// this happens, when we try to run /bin/sh in a container, which does not have it
	if err == nil && res.ExitStatus == 126 {
		output := ""
		b, err := io.ReadAll(res.Stdout)
		if err == nil {
			output = string(b)
		}
		err = errors.New("could not execute command: " + output)
	}
	return res, err
}

func (p *Provider) FS() afero.Fs {
	return p.Fs
}

func (p *Provider) FileInfo(path string) (os_provider.FileInfoDetails, error) {
	fs := p.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return os_provider.FileInfoDetails{}, err
	}

	mode := stat.Mode()

	uid := int64(-1)
	gid := int64(-1)

	if stat, ok := stat.Sys().(*os_provider.FileInfo); ok {
		uid = stat.Uid
		gid = stat.Gid
	}

	return os_provider.FileInfoDetails{
		Mode: os_provider.FileModeDetails{mode},
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
