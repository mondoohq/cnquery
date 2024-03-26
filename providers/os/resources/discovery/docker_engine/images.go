// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package docker_engine

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
)

// be aware that images are prefixed with sha256:, while containers are not
func (e *dockerEngineDiscovery) imageList() ([]image.Summary, error) {
	dc, err := e.client()
	if err != nil {
		return nil, err
	}

	return dc.ImageList(context.Background(), image.ListOptions{})
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
			Connections: []*inventory.Config{
				{
					Type: "docker-image",
					Host: dImg.ID,
				},
			},
		}
		log.Debug().Str("container", dImg.ID).Msg("discovered container-image")

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
