// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/docker/docker/client"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection/container/auth"
	"go.mondoo.com/cnquery/v10/providers/os/connection/container/docker_engine"
	"go.mondoo.com/cnquery/v10/providers/os/connection/container/image"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/connection/ssh/cat"
	"go.mondoo.com/cnquery/v10/providers/os/id/containerid"
	docker_discovery "go.mondoo.com/cnquery/v10/providers/os/resources/discovery/docker_engine"
)

const (
	DockerContainer shared.ConnectionType = "docker-container"
)

var _ shared.Connection = &DockerContainerConnection{}

type DockerContainerConnection struct {
	id       uint32
	parentId *uint32
	asset    *inventory.Asset

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

func NewDockerContainerConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) (*DockerContainerConnection, error) {
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

	conn := &DockerContainerConnection{
		id:        id,
		asset:     asset,
		Client:    dockerClient,
		container: conf.Host,
		kind:      "container",
		runtime:   "docker",
	}
	if len(asset.Connections) > 0 && asset.Connections[0].ParentConnectionId > 0 {
		conn.parentId = &asset.Connections[0].ParentConnectionId
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

func (c *DockerContainerConnection) ID() uint32 {
	return c.id
}

func (c *DockerContainerConnection) ParentID() *uint32 {
	return c.parentId
}

func (c *DockerContainerConnection) Name() string {
	return string(DockerContainer)
}

func (c *DockerContainerConnection) Type() shared.ConnectionType {
	return DockerContainer
}

func (c *DockerContainerConnection) Asset() *inventory.Asset {
	return c.asset
}

func (c *DockerContainerConnection) ContainerId() string {
	return c.container
}

func (c *DockerContainerConnection) Capabilities() shared.Capabilities {
	return shared.Capability_File | shared.Capability_RunCommand
}

func (c *DockerContainerConnection) FileInfo(path string) (shared.FileInfoDetails, error) {
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

func (c *DockerContainerConnection) FileSystem() afero.Fs {
	return c.Fs
}

func (c *DockerContainerConnection) RunCommand(command string) (*shared.Command, error) {
	log.Debug().Str("command", command).Msg("docker> run command")
	cmd := &docker_engine.Command{Client: c.Client, Container: c.container}
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

// NewContainerRegistryImage loads a container image from a remote registry
func NewContainerRegistryImage(id uint32, conf *inventory.Config, asset *inventory.Asset) (*TarConnection, error) {
	ref, err := name.ParseReference(conf.Host, name.WeakValidation)
	if err == nil {
		log.Debug().Str("ref", ref.Name()).Msg("found valid container registry reference")

		registryOpts := []image.Option{image.WithInsecure(conf.Insecure)}
		remoteOpts := auth.AuthOption(conf.Credentials)
		registryOpts = append(registryOpts, remoteOpts...)

		img, err := image.LoadImageFromRegistry(ref, registryOpts...)
		if err != nil {
			return nil, err
		}
		if asset.Connections[0].Options == nil {
			asset.Connections[0].Options = map[string]string{}
		}

		conn, err := NewTarConnectionForContainer(id, conf, asset, img)
		if err != nil {
			return nil, err
		}

		var identifier string
		hash, err := img.Digest()
		if err == nil {
			identifier = containerid.MondooContainerImageID(hash.String())
		}

		conn.PlatformIdentifier = identifier
		conn.Metadata.Name = containerid.ShortContainerImageID(hash.String())

		repoName := ref.Context().Name()
		imgDigest := hash.String()
		name := repoName + "@" + containerid.ShortContainerImageID(imgDigest)
		if asset.Name == "" {
			asset.Name = name
		}
		if len(asset.PlatformIds) == 0 {
			asset.PlatformIds = []string{identifier}
		} else {
			asset.PlatformIds = append(asset.PlatformIds, identifier)
		}

		// set the platform architecture using the image configuration
		imgConfig, err := img.ConfigFile()
		if err == nil {
			conn.PlatformArchitecture = imgConfig.Architecture
		}

		labels := map[string]string{}
		labels["docker.io/digests"] = ref.String()

		manifest, err := img.Manifest()
		if err == nil {
			labels["mondoo.com/image-id"] = manifest.Config.Digest.String()
		}

		conn.Metadata.Labels = labels
		asset.Labels = labels

		return conn, err
	}
	log.Debug().Str("image", conf.Host).Msg("Could not detect a valid repository url")
	return nil, err
}

func NewDockerEngineContainer(id uint32, conf *inventory.Config, asset *inventory.Asset) (shared.Connection, error) {
	// could be an image id/name, container id/name or a short reference to an image in docker engine
	ded, err := docker_discovery.NewDockerEngineDiscovery()
	if err != nil {
		return nil, err
	}

	ci, err := ded.ContainerInfo(conf.Host)
	if err != nil {
		return nil, err
	}

	if ci.Running {
		log.Debug().Msg("found running container " + ci.ID)

		conn, err := NewDockerContainerConnection(id, &inventory.Config{
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
		conn, err := NewFromDockerEngine(id, &inventory.Config{
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
	}
}

func NewDockerContainerImageConnection(id uint32, conf *inventory.Config, asset *inventory.Asset) (*TarConnection, error) {
	disableInmemoryCache := false
	if _, ok := conf.Options["disable-cache"]; ok {
		var err error
		disableInmemoryCache, err = strconv.ParseBool(conf.Options["disable-cache"])
		if err != nil {
			return nil, err
		}
	}
	// Determine whether the image is locally present or not.
	resolver := docker_discovery.Resolver{}
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
		return NewContainerRegistryImage(id, conf, asset)
	}

	// could be an image id/name, container id/name or a short reference to an image in docker engine
	ded, err := docker_discovery.NewDockerEngineDiscovery()
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
	_, rc, err := image.LoadImageFromDockerEngine(ii.ID, disableInmemoryCache)
	if err != nil {
		return nil, err
	}

	identifier := containerid.MondooContainerImageID(labelImageId)

	asset.PlatformIds = []string{identifier}
	asset.Name = ii.Name
	asset.Labels = ii.Labels

	tarConn, err := NewWithReader(id, conf, asset, rc)
	if err != nil {
		return nil, err
	}
	tarConn.PlatformIdentifier = identifier
	tarConn.Metadata.Name = ii.Name
	tarConn.Metadata.Labels = ii.Labels
	return tarConn, nil
}

// based on the target, try and find out what kind of connection we are dealing with, this can be either a
// 1. a container, referenced by name or id
// 2. a locally present image, referenced by tag or digest
// 3. a remote image, referenced by tag or digest
func FetchConnectionType(target string) (string, error) {
	ded, err := docker_discovery.NewDockerEngineDiscovery()
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
