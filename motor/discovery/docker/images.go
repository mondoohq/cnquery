package docker

import (
	"context"
	"strings"

	docker_types "github.com/docker/docker/api/types"
	"go.mondoo.io/mondoo/motor/runtime"
	"go.mondoo.io/mondoo/motor/transports"
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

		// TODO: we need to use the digest sha
		// docker does not always have a repo sha: docker images --digests
		digest := digest(dImg.RepoDigests)
		// fallback to docker id
		if len(digest) == 0 {
			digest = dImg.ID
		}

		asset := &assets.Asset{
			ReferenceIDs: []string{MondooContainerImageID(digest)},
			Name:         strings.Join(dImg.RepoTags, ","),
			Platform: &assets.Platform{
				Kind:    transports.Kind_KIND_CONTAINER_IMAGE,
				Runtime: runtime.RUNTIME_DOCKER_IMAGE,
			},
			Connections: []*transports.TransportConfig{
				&transports.TransportConfig{
					Backend: transports.TransportBackend_CONNECTION_DOCKER_IMAGE,
					Host:    dImg.ID,
				},
			},
			State:  assets.State_STATE_ONLINE,
			Labels: make(map[string]string),
		}

		for key := range dImg.Labels {
			asset.Labels[key] = dImg.Labels[key]
		}

		labels := map[string]string{}
		labels["mondoo.app/image-id"] = dImg.ID
		// project/repo:5e664d0e,gcr.io/project/repo:5e664d0e
		labels["docker.io/tags"] = strings.Join(dImg.RepoTags, ",")
		// gcr.io/project/repo@sha256:5248...2bee
		labels["docker.io/digests"] = strings.Join(dImg.RepoDigests, ",")
		asset.Labels = labels
		imgs[i] = asset
	}

	return imgs, nil
}

func digest(repoDigest []string) string {
	for i := range repoDigest {

		m := strings.Split(repoDigest[i], "sha256:")
		if len(m) == 2 {
			return "sha256:" + m[1]
		}
	}

	return ""
}

func MondooContainerImageID(id string) string {
	id = strings.Replace(id, "sha256:", "", -1)
	return "//platformid.api.mondoo.app/runtime/docker/images/" + id
}
