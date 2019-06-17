package docker

import (
	"context"
	"strings"

	docker_types "github.com/docker/docker/api/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/nexus/assets"
)

type Container struct{}

// DockerDisplayNames removes the leading slash of the internal docker name
// @see  https://github.com/moby/moby/issues/6705
func DockerDisplayNames(names []string) []string {
	if names == nil {
		return nil
	}

	displayNames := make([]string, len(names))
	for i := range names {
		displayNames[i] = strings.TrimLeft(names[i], "/")
	}

	return displayNames
}

func (a *Container) List() ([]*assets.Asset, error) {
	cl, err := GetDockerClient()
	if err != nil {
		return nil, err
	}

	dContainers, err := cl.ContainerList(context.Background(), docker_types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	container := make([]*assets.Asset, len(dContainers))
	for i, dContainer := range dContainers {
		name := strings.Join(DockerDisplayNames(dContainer.Names), ",")

		asset := &assets.Asset{
			ReferenceID:       MondooContainerID(dContainer.ID),
			Name:              name,
			ParentReferenceID: dContainer.ImageID,
			Platform: &assets.Platform{
				Kind:    assets.Kind_KIND_CONTAINER,
				Runtime: "docker",
			},
			Connections: []*assets.Connection{
				&assets.Connection{
					Backend: assets.ConnectionBackend_CONNECTION_DOCKER_CONTAINER,
					Host:    dContainer.ID,
				},
			},
			State:  mapContainerState(dContainer.State),
			Labels: make(map[string]string),
		}

		for key := range dContainer.Labels {
			asset.Labels[key] = dContainer.Labels[key]
		}

		// fetch docker specific metadata
		asset.Labels["mondoo.app/instance"] = dContainer.ID
		asset.Labels["mondoo.app/image-id"] = dContainer.ImageID
		asset.Labels["docker.io/image-name"] = dContainer.Image
		asset.Labels["docker.io/names"] = name

		container[i] = asset
	}
	return container, nil
}

func MondooContainerID(id string) string {
	return "docker://container/" + id
}

func mapContainerState(state string) assets.State {
	switch state {
	case "running":
		return assets.State_STATE_RUNNING
	case "created":
		return assets.State_STATE_PENDING
	case "paused":
		return assets.State_STATE_STOPPED
	case "exited":
		return assets.State_STATE_TERMINATED
	case "restarting":
		return assets.State_STATE_PENDING
	case "dead":
		return assets.State_STATE_ERROR
	default:
		log.Warn().Str("state", state).Msg("unknown container state")
		return assets.State_STATE_UNKNOWN
	}
}
