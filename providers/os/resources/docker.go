// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"strings"

	"github.com/moby/moby/client"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/types"
)

func (p *mqlDocker) images() ([]any, error) {
	cl, err := dockerClient()
	if err != nil {
		return nil, err
	}

	imageListRes, err := cl.ImageList(context.Background(), client.ImageListOptions{})
	if err != nil {
		return nil, err
	}
	dImages := imageListRes.Items

	imgs := make([]any, len(dImages))
	for i, dImg := range dImages {
		labels := make(map[string]any)
		for key := range dImg.Labels {
			labels[key] = dImg.Labels[key]
		}

		tags := []any{}
		for i := range dImg.RepoTags {
			tags = append(tags, dImg.RepoTags[i])
		}

		r, err := CreateResource(p.MqlRuntime, "docker.image", map[string]*llx.RawData{
			"id":          llx.StringData(dImg.ID),
			"size":        llx.IntData(dImg.Size),
			"repoDigests": llx.ArrayData(llx.TArr2Raw(dImg.RepoDigests), types.String),
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

func (p *mqlDocker) containers() ([]any, error) {
	cl, err := dockerClient()
	if err != nil {
		return nil, err
	}

	containerListRes, err := cl.ContainerList(context.Background(), client.ContainerListOptions{})
	if err != nil {
		return nil, err
	}
	dContainers := containerListRes.Items

	container := make([]any, len(dContainers))

	for i, dContainer := range dContainers {
		labels := make(map[string]any)
		for key := range dContainer.Labels {
			labels[key] = dContainer.Labels[key]
		}

		names := []any{}
		for i := range dContainer.Names {
			name := dContainer.Names[i]
			name = strings.TrimPrefix(name, "/")
			names = append(names, name)
		}

		/*
			FIXME: ??? not used?
			conn, err := connection.NewDockerEngineContainer(dContainer.ID)
			if err != nil {
				return nil, err
			}
		*/

		o, err := CreateResource(p.MqlRuntime, "docker.container", map[string]*llx.RawData{
			"id":      llx.StringData(dContainer.ID),
			"image":   llx.StringData(dContainer.Image),
			"imageid": llx.StringData(dContainer.ImageID),
			"command": llx.StringData(dContainer.Command),
			"state":   llx.StringData(string(dContainer.State)),
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

func (p *mqlDockerContainer) hostConfig() (any, error) {
	cl, err := dockerClient()
	if err != nil {
		return nil, err
	}

	res, err := cl.ContainerInspect(context.Background(), p.Id.Data, client.ContainerInspectOptions{})
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(res.Container.HostConfig)
}

func dockerClient() (*client.Client, error) {
	// No explicit NegotiateAPIVersion call: the method was removed from *Client in
	// moby/moby's v29 client rewrite. API version negotiation now happens
	// automatically on the first request (WithAPIVersionNegotiation is a
	// documented no-op kept only for backward compatibility).
	cl, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}
	log.Debug().Msgf("docker client> negotiated API version %s", cl.ClientVersion())
	return cl, nil
}
