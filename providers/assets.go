// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/cli/config"
	"go.mondoo.com/cnquery/v10/logger"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	pp "go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
)

func ProcessAssetCandidates(runtime *Runtime, connectRes *pp.ConnectRes, upstreamConfig *upstream.UpstreamConfig, platformID string) ([]*inventory.Asset, error) {
	var assetCandidates []*inventory.Asset
	if connectRes.Inventory == nil || connectRes.Inventory.Spec == nil {
		return []*inventory.Asset{connectRes.Asset}, nil
	} else {
		logger.DebugDumpJSON("inventory-resolved", connectRes.Inventory)
		assetCandidates = connectRes.Inventory.Spec.Assets
	}
	log.Debug().Msgf("resolved %d assets", len(assetCandidates))

	if err := detectAssets(runtime, assetCandidates, upstreamConfig); err != nil {
		return nil, err
	}

	if platformID != "" {
		res, err := filterAssetByPlatformID(assetCandidates, platformID)
		if err != nil {
			return nil, err
		}
		return []*inventory.Asset{res}, nil
	}

	return filterUniqueAssets(assetCandidates), nil
}

// detectAssets connects to all assets that do not have a platform ID yet
func detectAssets(runtime *Runtime, assetCandidates []*inventory.Asset, upstreamConfig *upstream.UpstreamConfig) error {
	for i := range assetCandidates {
		asset := assetCandidates[i]
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
			continue
		}
		// Use the updated asset
		assetCandidates[i] = runtime.Provider.Connection.Asset
	}
	return nil
}

func filterAssetByPlatformID(assetList []*inventory.Asset, selectionID string) (*inventory.Asset, error) {
	var foundAsset *inventory.Asset
	for i := range assetList {
		assetObj := assetList[i]
		for j := range assetObj.PlatformIds {
			if assetObj.PlatformIds[j] == selectionID {
				return assetObj, nil
			}
		}
	}

	if foundAsset == nil {
		return nil, errors.New("could not find an asset with the provided identifier: " + selectionID)
	}
	return foundAsset, nil
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
				log.Debug().Msgf("skipping asset %s with duplicate platform ID %s", asset.Name, platformId)
				break
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
