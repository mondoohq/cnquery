package docker

import (
	"context"
	"strings"

	docker_types "github.com/docker/docker/api/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/runtime"
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
			ReferenceIDs:      []string{MondooContainerID(dContainer.ID)},
			Name:              name,
			ParentReferenceID: dContainer.ImageID,
			Platform: &assets.Platform{
				Kind:    assets.Kind_KIND_CONTAINER,
				Runtime: runtime.RUNTIME_DOCKER,
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
		labels := map[string]string{}
		labels["mondoo.app/instance"] = dContainer.ID
		labels["mondoo.app/image-id"] = dContainer.ImageID
		labels["docker.io/image-name"] = dContainer.Image
		labels["docker.io/names"] = name
		asset.Labels = labels

		container[i] = asset
	}
	return container, nil
}

func MondooContainerID(id string) string {
	return "//platformid.api.mondoo.app/runtime/docker/containers/" + id
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
