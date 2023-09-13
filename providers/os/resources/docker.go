// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"os"

	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers/aws/resources"
	"go.mondoo.com/cnquery/providers/os/connection"
	"go.mondoo.com/cnquery/types"
)

func (p *mqlDocker) images() ([]interface{}, error) {
	cl, err := dockerClient()
	if err != nil {
		return nil, err
	}

	dImages, err := cl.ImageList(context.Background(), docker_types.ImageListOptions{})
	if err != nil {
		return nil, err
	}

	imgs := make([]interface{}, len(dImages))
	for i, dImg := range dImages {
		labels := make(map[string]interface{})
		for key := range dImg.Labels {
			labels[key] = dImg.Labels[key]
		}

		tags := []interface{}{}
		for i := range dImg.RepoTags {
			tags = append(tags, dImg.RepoTags[i])
		}

		r, err := CreateResource(p.MqlRuntime, "docker.image", map[string]*llx.RawData{
			"id":          llx.StringData(dImg.ID),
			"size":        llx.IntData(dImg.Size),
			"virtualsize": llx.IntData(dImg.VirtualSize),
			"labels":      llx.MapData(labels, types.String),
			"tags":        llx.ArrayData(tags, types.String),
		})
		if err != nil {
			return nil, err
		}

		imgs[i] = r.(*mqlDockerImage)
	}

	return imgs, nil
}

func (p *mqlDocker) containers() ([]interface{}, error) {
	cl, err := dockerClient()
	if err != nil {
		return nil, err
	}

	dContainers, err := cl.ContainerList(context.Background(), docker_types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	container := make([]interface{}, len(dContainers))
	localConn := p.MqlRuntime.Connection.(*connection.LocalConnection)
	localConnId := localConn.ID()

	for i, dContainer := range dContainers {
		labels := make(map[string]interface{})
		for key := range dContainer.Labels {
			labels[key] = dContainer.Labels[key]
		}

		names := []interface{}{}
		for i := range dContainer.Names {
			names = append(names, dContainer.Names[i])
		}

		localConnId++
		containerConf := &inventory.Config{
			Host: dContainer.ID,
			Type: "docker-container",
		}
		containerAsset := &inventory.Asset{
			Name:        dContainer.ID,
			Connections: []*inventory.Config{containerConf},
		}
		containerConn, err := connection.NewDockerEngineContainer(localConnId, containerConf, containerAsset)
		if err != nil {
			return nil, err
		}
		containerAsset.Connections[0].Id = containerConn.ID()
		containerRuntime := &plugin.Runtime{
			Connection:     containerConn,
			Callback:       p.MqlRuntime.Callback,
			HasRecording:   p.MqlRuntime.HasRecording,
			CreateResource: resources.CreateResource,
			Upstream:       p.MqlRuntime.Upstream,
		}
		// s.runtimes[containerConn.ID()] = containerRuntime

		o, err := CreateResource(containerRuntime, "docker.container", map[string]*llx.RawData{
			"id":      llx.StringData(dContainer.ID),
			"image":   llx.StringData(dContainer.Image),
			"imageid": llx.StringData(dContainer.ImageID),
			"command": llx.StringData(dContainer.Command),
			"state":   llx.StringData(dContainer.State),
			"status":  llx.StringData(dContainer.Status),
			"labels":  llx.MapData(labels, types.String),
			"names":   llx.ArrayData(names, types.String),
		})
		if err != nil {
			return nil, err
		}

		container[i] = o.(*mqlDockerContainer)
	}

	return container, nil
}

func (p *mqlDockerContainer) os() (*mqlOsLinux, error) {
	res, err := CreateResource(p.MqlRuntime, "os.linux", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOsLinux), nil
}

func (p *mqlDockerImage) id() (string, error) {
	return p.Id.Data, nil
}

func (p *mqlDockerContainer) id() (string, error) {
	return p.Id.Data, nil
}

func dockerClient() (*client.Client, error) {
	// set docker api version for macos
	os.Setenv("DOCKER_API_VERSION", "1.26")
	// Start new docker container
	return client.NewClientWithOpts(client.FromEnv)
}
