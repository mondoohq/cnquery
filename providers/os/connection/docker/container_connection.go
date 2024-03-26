// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package docker

import (
	"context"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-containerregistry/pkg/v1/mutate"

	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/os/connection/container"
	"go.mondoo.com/cnquery/v10/providers/os/connection/container/image"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/connection/ssh/cat"
	"go.mondoo.com/cnquery/v10/providers/os/connection/tar"
	"go.mondoo.com/cnquery/v10/providers/os/id/containerid"
	dockerDiscovery "go.mondoo.com/cnquery/v10/providers/os/resources/discovery/docker_engine"
)

const (
	ContainerConnectionType shared.ConnectionType = "docker-container"
)

var _ shared.Connection = &ContainerConnection{}

type ContainerConnection struct {
	plugin.Connection
	asset *inventory.Asset

	Client    *client.Client
	container string
	Fs        *FS

	PlatformIdentifier   string
	PlatformArchitecture string
	// optional metadata to store additional information
	Metadata struct {
		Name   string
		Labels map[string]string
	}

	kind    string
	runtime string
}

func NewContainerConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) (*ContainerConnection, error) {
	// expect unix shell by default
	dockerClient, err := GetDockerClient()
	if err != nil {
		return nil, err
	}

	// check if we are having a container
	data, err := dockerClient.ContainerInspect(context.Background(), conf.Host)
	if err != nil {
		return nil, errors.New("cannot find container " + conf.Host)
	}

	if !data.State.Running {
		return nil, errors.New("container " + data.ID + " is not running")
	}

	conn := &ContainerConnection{
		Connection: plugin.NewConnection(id, asset),
		asset:      asset,
		Client:     dockerClient,
		container:  conf.Host,
		kind:       "container",
		runtime:    "docker",
	}

	// this can later be used for containers build from scratch
	serverVersion, err := dockerClient.ServerVersion(context.Background())
	if err != nil {
		log.Debug().Err(err).Msg("docker> cannot get server version")
	} else {
		log.Debug().Interface("serverVersion", serverVersion).Msg("docker> server version")
		conn.PlatformArchitecture = serverVersion.Arch
	}

	conn.Fs = &FS{
		dockerClient: conn.Client,
		Container:    conn.container,
		Connection:   conn,
		catFS:        cat.New(conn),
	}
	return conn, nil
}

func GetDockerClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	cli.NegotiateAPIVersion(context.Background())
	return cli, nil
}

func (c *ContainerConnection) Name() string {
	return string(ContainerConnectionType)
}

func (c *ContainerConnection) Type() shared.ConnectionType {
	return ContainerConnectionType
}

func (c *ContainerConnection) Asset() *inventory.Asset {
	return c.asset
}

func (p *ContainerConnection) UpdateAsset(asset *inventory.Asset) {
	p.asset = asset
}

func (c *ContainerConnection) ContainerId() string {
	return c.container
}

func (c *ContainerConnection) Capabilities() shared.Capabilities {
	return shared.Capability_File | shared.Capability_RunCommand
}

func (c *ContainerConnection) FileInfo(path string) (shared.FileInfoDetails, error) {
	fs := c.FileSystem()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return shared.FileInfoDetails{}, err
	}

	mode := stat.Mode()

	uid := int64(-1)
	gid := int64(-1)

	if stat, ok := stat.Sys().(*shared.FileInfo); ok {
		uid = stat.Uid
		gid = stat.Gid
	}

	return shared.FileInfoDetails{
		Mode: shared.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (c *ContainerConnection) FileSystem() afero.Fs {
	return c.Fs
}

func (c *ContainerConnection) RunCommand(command string) (*shared.Command, error) {
	log.Debug().Str("command", command).Msg("docker> run command")
	cmd := &Command{Client: c.Client, Container: c.container}
	res, err := cmd.Exec(command)
	// this happens, when we try to run /bin/sh in a container, which does not have it
	if err == nil && res.ExitStatus == 126 {
		output := ""
		var b []byte
		b, err = io.ReadAll(res.Stdout)
		if err == nil {
			output = string(b)
		}
		err = errors.New("could not execute command: " + output)
	}
	return res, err
}

func NewDockerEngineContainer(id uint32, conf *inventory.Config, asset *inventory.Asset) (shared.Connection, error) {
	// could be an image id/name, container id/name or a short reference to an image in docker engine
	ded, err := dockerDiscovery.NewDockerEngineDiscovery()
	if err != nil {
		return nil, err
	}

	ci, err := ded.ContainerInfo(conf.Host)
	if err != nil {
		return nil, err
	}

	if ci.Running {
		log.Debug().Msg("found running container " + ci.ID)

		conn, err := NewContainerConnection(id, &inventory.Config{
			Host: ci.ID,
		}, asset)
		if err != nil {
			return nil, err
		}
		conn.PlatformIdentifier = containerid.MondooContainerID(ci.ID)
		conn.Metadata.Name = containerid.ShortContainerImageID(ci.ID)
		conn.Metadata.Labels = ci.Labels
		asset.Name = ci.Name
		asset.PlatformIds = []string{containerid.MondooContainerID(ci.ID)}
		return conn, nil
	} else {
		log.Debug().Msg("found stopped container " + ci.ID)
		conn, err := NewSnapshotConnection(id, &inventory.Config{
			Host: ci.ID,
		}, asset)
		if err != nil {
			return nil, err
		}
		conn.PlatformIdentifier = containerid.MondooContainerID(ci.ID)
		conn.Metadata.Name = containerid.ShortContainerImageID(ci.ID)
		conn.Metadata.Labels = ci.Labels
		// FIXME: DEPRECATED, remove in v12.0 vv
		// The DelayDiscovery flag should always be set from v12
		if conf.Options == nil || conf.Options[plugin.DISABLE_DELAYED_DISCOVERY_OPTION] == "" {
			conf.DelayDiscovery = true // Delay discovery, to make sure we don't directly download the image
		}
		// ^^
		asset.Name = ci.Name
		asset.PlatformIds = []string{containerid.MondooContainerID(ci.ID)}
		return conn, nil
	}
}

func NewContainerImageConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) (*tar.Connection, error) {
	disableInmemoryCache := false
	if _, ok := conf.Options["disable-cache"]; ok {
		var err error
		disableInmemoryCache, err = strconv.ParseBool(conf.Options["disable-cache"])
		if err != nil {
			return nil, err
		}
	}
	if conf.Options == nil {
		conf.Options = map[string]string{}
	}
	// FIXME: DEPRECATED, remove in v12.0 vv
	// The DelayDiscovery flag should always be set from v12
	if conf.Options == nil || conf.Options[plugin.DISABLE_DELAYED_DISCOVERY_OPTION] == "" {
		conf.DelayDiscovery = true // Delay discovery, to make sure we don't directly download the image
	}
	// ^^
	// Determine whether the image is locally present or not.
	resolver := dockerDiscovery.Resolver{}
	resolvedAssets, err := resolver.Resolve(context.Background(), asset, conf, nil)
	if err != nil {
		return nil, err
	}

	if len(resolvedAssets) > 1 {
		return nil, errors.New("provided image name resolved to more than one container image")
	}

	// The requested image isn't locally available, but we can pull it from a remote registry.
	if len(resolvedAssets) > 0 && resolvedAssets[0].Connections[0].Type == "container-registry" {
		asset.Name = resolvedAssets[0].Name
		asset.PlatformIds = resolvedAssets[0].PlatformIds
		asset.Labels = resolvedAssets[0].Labels
		return container.NewRegistryImage(id, conf, asset)
	}

	// could be an image id/name, container id/name or a short reference to an image in docker engine
	ded, err := dockerDiscovery.NewDockerEngineDiscovery()
	if err != nil {
		return nil, err
	}

	ii, err := ded.ImageInfo(conf.Host)
	if err != nil {
		return nil, err
	}

	labelImageId := ii.ID
	splitLabels := strings.Split(ii.Labels["docker.io/digests"], ",")
	if len(splitLabels) > 1 {
		labelImageIdFull := splitLabels[0]
		splitFullLabel := strings.Split(labelImageIdFull, "@")
		if len(splitFullLabel) > 1 {
			labelImageId = strings.Split(labelImageIdFull, "@")[1]
		}
	}

	// This is the image id that is used to pull the image from the registry.
	log.Debug().Msg("found docker engine image " + labelImageId)
	if ii.Size > 1024 && !disableInmemoryCache { // > 1GB
		log.Warn().Int64("size", ii.Size).Msg("Because the image is larger than 1 GB, this task will require a lot of memory. Consider disabling the in-memory cache by adding this flag to the command: `--disable-cache=true`")
	}

	identifier := containerid.MondooContainerImageID(labelImageId)

	asset.PlatformIds = []string{identifier}
	asset.Name = ii.Name
	asset.Labels = ii.Labels

	// cache file locally
	var filename string
	tmpFile, err := tar.RandomFile()
	if err != nil {
		return nil, err
	}
	filename = tmpFile.Name()

	conf.Options[tar.OPTION_FILE] = filename

	tarConn, err := tar.NewConnection(
		id,
		conf,
		asset,
		tar.WithFetchFn(func() (string, error) {
			img, err := image.LoadImageFromDockerEngine(ii.ID, disableInmemoryCache)
			if err != nil {
				return filename, err
			}
			err = tar.StreamToTmpFile(mutate.Extract(img), tmpFile)
			if err != nil {
				_ = os.Remove(filename)
				return filename, err
			}
			return filename, nil
		}),
		tar.WithCloseFn(func() {
			log.Debug().Str("tar", filename).Msg("tar> remove temporary tar file on connection close")
			_ = os.Remove(filename)
		}))
	if err != nil {
		return nil, err
	}
	tarConn.PlatformIdentifier = identifier
	tarConn.Metadata.Name = ii.Name
	tarConn.Metadata.Labels = ii.Labels
	return tarConn, nil
}

// FindDockerObjectConnectionType tries to find out what kind of connection we are dealing with, this can be either a
// 1. a container, referenced by name or id
// 2. a locally present image, referenced by tag or digest
// 3. a remote image, referenced by tag or digest
func FindDockerObjectConnectionType(target string) (string, error) {
	ded, err := dockerDiscovery.NewDockerEngineDiscovery()
	if err != nil {
		return "", err
	}

	if ded != nil {
		_, err = ded.ContainerInfo(target)
		if err == nil {
			return "docker-container", nil
		}
		_, err = ded.ImageInfo(target)
		if err == nil {
			return "docker-image", nil
		}
	}
	_, err = name.ParseReference(target, name.WeakValidation)
	if err == nil {
		return "docker-image", nil
	}

	return "", errors.New("could not find container or image " + target)
}
