// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package docker_engine

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers/os/id/containerid"
)

// be aware that images are prefixed with sha256:, while containers are not
func (e *dockerEngineDiscovery) imageList() ([]types.ImageSummary, error) {
	dc, err := e.client()
	if err != nil {
		return nil, err
	}

	return dc.ImageList(context.Background(), types.ImageListOptions{})
}

func (e *dockerEngineDiscovery) ListImageShas() ([]string, error) {
	images, err := e.imageList()
	if err != nil {
		return []string{}, err
	}

	imagesShas := []string{}
	for i := range images {
		imagesShas = append(imagesShas, images[i].ID)
	}

	return imagesShas, nil
}

func (e *dockerEngineDiscovery) ListImages() ([]*inventory.Asset, error) {
	dImages, err := e.imageList()
	if err != nil {
		return nil, err
	}
	imgs := make([]*inventory.Asset, len(dImages))
	for i, dImg := range dImages {

		// TODO: we need to use the digest sha
		// docker does not always have a repo sha: docker images --digests
		digest := digest(dImg.RepoDigests)
		// fallback to docker id
		if len(digest) == 0 {
			digest = dImg.ID
		}

		asset := &inventory.Asset{
			Name:        strings.Join(dImg.RepoTags, ","),
			PlatformIds: []string{containerid.MondooContainerImageID(digest)},
			Platform: &inventory.Platform{
				Kind:    "container-image",
				Runtime: "docker-image",
			},
			Connections: []*inventory.Config{
				{
					Backend: "docker-image",
					Host:    dImg.ID,
				},
			},
			State: inventory.State_STATE_ONLINE,
		}
		log.Debug().Str("container", dImg.ID).Msg("discovered container-image")

		// update labels
		labels := map[string]string{}
		for key := range dImg.Labels {
			labels[key] = dImg.Labels[key]
		}

		labels["mondoo.com/image-id"] = dImg.ID
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
