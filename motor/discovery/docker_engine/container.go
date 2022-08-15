package docker_engine

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/motorid/containerid"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/providers"
)

func (e *dockerEngineDiscovery) containerList() ([]types.Container, error) {
	dc, err := e.client()
	if err != nil {
		return nil, err
	}

	return dc.ContainerList(context.Background(), types.ContainerListOptions{})
}

func (e *dockerEngineDiscovery) ListContainerShas() ([]string, error) {
	containers, err := e.containerList()
	if err != nil {
		return []string{}, err
	}

	containerShas := []string{}
	for i := range containers {
		containerShas = append(containerShas, containers[i].ID)
	}

	return containerShas, nil
}

type ContainerInfo struct {
	ID         string
	Name       string
	PlatformID string
	Running    bool
	Labels     map[string]string
	Arch       string
}

// will resolve name and id to a container id
func (e *dockerEngineDiscovery) ContainerInfo(name string) (ContainerInfo, error) {
	ci := ContainerInfo{}
	dc, err := e.client()
	if err != nil {
		return ci, err
	}

	cdata, err := dc.ContainerInspect(context.Background(), name)
	if err != nil {
		return ci, err
	}

	cName := cdata.Name
	cName = strings.TrimPrefix(cName, "/")
	if len(cName) == 0 {
		cName = containerid.ShortContainerID(cdata.ID)
	}

	ci.ID = cdata.ID
	ci.Name = cName
	ci.PlatformID = containerid.MondooContainerID(ci.ID)
	ci.Running = cdata.State.Running

	// fetch docker specific metadata
	labels := map[string]string{}
	labels["mondoo.com/instance"] = cdata.ID
	labels["docker.io/container-id"] = cdata.ID
	labels["docker.io/image-name"] = cdata.Image
	// labels["mondoo.app/image-id"] = cdata.ImageID
	labels["docker.io/names"] = cName

	ci.Labels = labels

	return ci, nil
}

type ImageInfo struct {
	ID         string
	Name       string
	PlatformID string
	Labels     map[string]string
	Arch       string
}

func (e *dockerEngineDiscovery) ImageInfo(name string) (ImageInfo, error) {
	ii := ImageInfo{}
	dc, err := e.client()
	if err != nil {
		return ii, err
	}

	res, _, err := dc.ImageInspectWithRaw(context.Background(), name)
	if err != nil {
		return ii, err
	}

	switch res.Architecture {
	case "amd64":
		ii.Arch = "x86_64"
	}

	labels := map[string]string{}
	labels["mondoo.com/image-id"] = res.ID
	labels["docker.io/tags"] = strings.Join(res.RepoTags, ",")
	labels["docker.io/digests"] = strings.Join(res.RepoDigests, ",")

	ii.Name = containerid.ShortContainerImageID(res.ID)
	ii.ID = res.ID
	ii.Labels = labels
	ii.PlatformID = containerid.MondooContainerImageID(res.ID)
	return ii, nil
}

func (e *dockerEngineDiscovery) ListContainer() ([]*asset.Asset, error) {
	dContainers, err := e.containerList()
	if err != nil {
		return nil, err
	}

	container := make([]*asset.Asset, len(dContainers))
	for i, dContainer := range dContainers {
		name := strings.Join(DockerDisplayNames(dContainer.Names), ",")
		asset := &asset.Asset{
			Name:        name,
			PlatformIds: []string{containerid.MondooContainerID(dContainer.ID)},
			Platform: &platform.Platform{
				Kind:    providers.Kind_KIND_CONTAINER,
				Runtime: providers.RUNTIME_DOCKER_CONTAINER,
			},
			Connections: []*providers.Config{
				{
					Backend: providers.ProviderType_DOCKER_ENGINE_CONTAINER,
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
		labels["mondoo.com/instance"] = dContainer.ID
		labels["mondoo.com/image-id"] = dContainer.ImageID
		labels["docker.io/image-name"] = dContainer.Image
		labels["docker.io/names"] = name
		asset.Labels = labels

		container[i] = asset
	}
	return container, nil
}

func mapContainerState(state string) asset.State {
	switch state {
	case "running":
		return asset.State_STATE_RUNNING
	case "created":
		return asset.State_STATE_PENDING
	case "paused":
		return asset.State_STATE_STOPPED
	case "exited":
		return asset.State_STATE_TERMINATED
	case "restarting":
		return asset.State_STATE_PENDING
	case "dead":
		return asset.State_STATE_ERROR
	default:
		log.Warn().Str("state", state).Msg("unknown container state")
		return asset.State_STATE_UNKNOWN
	}
}

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
