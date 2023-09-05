// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gcr

import (
	"context"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/vault"
)

type GcrResolver struct{}

func (r *GcrResolver) Name() string {
	return "GCP Container Registry Resolver"
}

func (r *GcrResolver) AvailableDiscoveryTargets() []string {
	return []string{"auto", "all"}
}

// func (r *GcrResolver) Resolve(ctx context.Context, root *inventory.Asset, t *inventory.Config, credsResolver vault.Resolver, sfn common.QuerySecretFn, userIdDetectors ...providers.PlatformIdDetector) ([]*inventory.Asset, error) {
func (r *GcrResolver) Resolve(ctx context.Context, root *inventory.Asset, conf *inventory.Config, credsResolver vault.Resolver) ([]*inventory.Asset, error) {
	resolved := []*inventory.Asset{}
	repository := conf.Host

	log.Debug().Str("registry", repository).Msg("fetch meta information from gcr registry")
	gcrImages := NewGCRImages()
	assetList, err := gcrImages.ListRepository(repository, true)
	if err != nil {
		log.Error().Err(err).Msg("could not fetch gcr images")
		return nil, err
	}

	for i := range assetList {
		log.Debug().Str("name", assetList[i].Name).Str("image", assetList[i].Connections[0].Host+assetList[i].Connections[0].Path).Msg("resolved image")
		resolved = append(resolved, assetList[i])
	}

	return resolved, nil
}
