// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"go.mondoo.com/cnquery/cli/config"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	pp "go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/upstream"
)

func ProcessAssetCandidates(runtime *Runtime, assetCandidates []*inventory.Asset, upstreamConfig *upstream.UpstreamConfig) ([]*inventory.Asset, error) {
	if err := detectAssets(runtime, assetCandidates, upstreamConfig); err != nil {
		return nil, err
	}

	return filterUniqueAssets(assetCandidates), nil
}

// detectAssets connects to all assets that do not have a platform ID yet
func detectAssets(runtime *Runtime, assetCandidates []*inventory.Asset, upstreamConfig *upstream.UpstreamConfig) error {
	for _, asset := range assetCandidates {
		// If the assets have platform IDs, then we have already connected to them via the
		// current provider.
		if len(asset.PlatformIds) > 0 {
			continue
		}

		// Make sure the provider for the asset is present
		if err := runtime.DetectProvider(asset); err != nil {
			return err
		}

		err := runtime.Connect(&pp.ConnectReq{
			Features: config.Features,
			Asset:    asset,
			Upstream: upstreamConfig,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// filterUniqueAssets filters assets with duplicate platform IDs
func filterUniqueAssets(assetCandidates []*inventory.Asset) []*inventory.Asset {
	uniqueAssets := []*inventory.Asset{}
	platformIds := map[string]struct{}{}
	for _, asset := range assetCandidates {
		found := false
		for _, platformId := range asset.PlatformIds {
			if _, ok := platformIds[platformId]; ok {
				found = true
			}
		}
		if found {
			continue
		}

		uniqueAssets = append(uniqueAssets, asset)
		for _, platformId := range asset.PlatformIds {
			platformIds[platformId] = struct{}{}
		}
	}
	return uniqueAssets
}
