package docker

import (
	"context"
	"strings"

	docker_types "github.com/docker/docker/api/types"
	"go.mondoo.io/mondoo/nexus/assets"
)

type Images struct{}

func (a *Images) List() ([]*assets.Asset, error) {
	cl, err := GetDockerClient()
	if err != nil {
		return nil, err
	}

	dImages, err := cl.ImageList(context.Background(), docker_types.ImageListOptions{})
	if err != nil {
		return nil, err
	}

	imgs := make([]*assets.Asset, len(dImages))
	for i, dImg := range dImages {
		asset := &assets.Asset{
			ReferenceIDs: []string{MondooContainerImageID(dImg.ID)},
			Name:         strings.Join(dImg.RepoTags, ","),
			Platform: &assets.Platform{
				Kind:    assets.Kind_KIND_CONTAINER_IMAGE,
				Runtime: "docker",
			},
			Connections: []*assets.Connection{
				&assets.Connection{
					Backend: assets.ConnectionBackend_CONNECTION_DOCKER_IMAGE,
					Host:    dImg.ID,
				},
			},
			State:  assets.State_STATE_ONLINE,
			Labels: make(map[string]string),
		}

		for key := range dImg.Labels {
			asset.Labels[key] = dImg.Labels[key]
		}

		asset.Labels["mondoo.app/image-id"] = dImg.ID
		asset.Labels["docker.io/tags"] = strings.Join(dImg.RepoTags, ",")
		asset.Labels["docker.io/digests"] = strings.Join(dImg.RepoDigests, ",")

		imgs[i] = asset
	}

	return imgs, nil
}

func MondooContainerImageID(id string) string {
	id = strings.Replace(id, "sha256:", "", -1)
	return "//platformid.api.mondoo.app/runtime/docker/images/" + id
}
