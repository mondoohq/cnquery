// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/cli/config"
	"go.mondoo.com/cnquery/v10/cli/execruntime"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers"
	inventory "go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory/manager"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
)

type AssetWithRuntime struct {
	Asset   *inventory.Asset
	Runtime *providers.Runtime
}

type AssetWithError struct {
	Asset *inventory.Asset
	Err   error
}

type DiscoveredAssets struct {
	platformIds map[string]struct{}
	Assets      []*AssetWithRuntime
	Errors      []*AssetWithError
}

// Add adds an asset and its runtime to the discovered assets list. It returns true if the
// asset has been added, false if it is a duplicate
func (d *DiscoveredAssets) Add(asset *inventory.Asset, runtime *providers.Runtime) bool {
	isDuplicate := false
	for _, platformId := range asset.PlatformIds {
		if _, ok := d.platformIds[platformId]; ok {
			isDuplicate = true
			break
		}
		d.platformIds[platformId] = struct{}{}
	}
	if isDuplicate {
		return false
	}

	d.Assets = append(d.Assets, &AssetWithRuntime{Asset: asset, Runtime: runtime})
	return true
}

func (d *DiscoveredAssets) AddError(asset *inventory.Asset, err error) {
	d.Errors = append(d.Errors, &AssetWithError{Asset: asset, Err: err})
}

func (d *DiscoveredAssets) GetAssets() []*inventory.Asset {
	assets := make([]*inventory.Asset, 0, len(d.Assets))
	for _, a := range d.Assets {
		assets = append(assets, a.Asset)
	}
	return assets
}

func (d *DiscoveredAssets) GetAssetsByPlatformID(platformID string) []*inventory.Asset {
	var assets []*inventory.Asset
	for _, a := range d.Assets {
		for _, p := range a.Asset.PlatformIds {
			if platformID == "" || p == platformID {
				assets = append(assets, a.Asset)
				break
			}
		}
	}
	return assets
}

// DiscoverAssets discovers assets from the given inventory and upstream configuration. Returns only unique assets
func DiscoverAssets(ctx context.Context, inv *inventory.Inventory, upstream *upstream.UpstreamConfig, recording llx.Recording) (*DiscoveredAssets, error) {
	im, err := manager.NewManager(manager.WithInventory(inv, providers.DefaultRuntime()))
	if err != nil {
		return nil, errors.New("failed to resolve inventory for connection")
	}
	invAssets := im.GetAssets()
	if len(invAssets) == 0 {
		return nil, errors.New("could not find an asset that we can connect to")
	}

	runtimeEnv := execruntime.Detect()
	var runtimeLabels map[string]string
	// If the runtime is an automated environment and the root asset is CI/CD, then we are doing a
	// CI/CD scan and we need to apply the runtime labels to the assets
	if runtimeEnv != nil &&
		runtimeEnv.IsAutomatedEnv() &&
		inv.Spec.Assets[0].Category == inventory.AssetCategory_CATEGORY_CICD {
		runtimeLabels = runtimeEnv.Labels()
	}

	discoveredAssets := &DiscoveredAssets{platformIds: map[string]struct{}{}}

	// we connect and perform discovery for each asset in the job inventory
	for _, rootAsset := range invAssets {
		resolvedRootAsset, err := im.ResolveAsset(rootAsset)
		if err != nil {
			return nil, err
		}

		// create runtime for root asset
		rootAssetWithRuntime, err := createRuntimeForAsset(resolvedRootAsset, upstream, recording)
		if err != nil {
			log.Error().Err(err).Str("asset", resolvedRootAsset.Name).Msg("unable to create runtime for asset")
			discoveredAssets.AddError(rootAssetWithRuntime.Asset, err)
			continue
		}

		resolvedRootAsset = rootAssetWithRuntime.Asset // to ensure we get all the information the connect call gave us

		// for all discovered assets, we apply mondoo-specific labels and annotations that come from the root asset
		for _, a := range rootAssetWithRuntime.Runtime.Provider.Connection.Inventory.Spec.Assets {
			// create runtime for root asset
			assetWithRuntime, err := createRuntimeForAsset(a, upstream, recording)
			if err != nil {
				log.Error().Err(err).Str("asset", a.Name).Msg("unable to create runtime for asset")
				discoveredAssets.AddError(assetWithRuntime.Asset, err)
				continue
			}

			resolvedAsset := assetWithRuntime.Runtime.Provider.Connection.Asset
			prepareAsset(resolvedAsset, resolvedRootAsset, runtimeLabels)

			// If the asset has been already added, we should close its runtime
			if !discoveredAssets.Add(resolvedAsset, assetWithRuntime.Runtime) {
				assetWithRuntime.Runtime.Close()
			}
		}
	}

	// if there is exactly one asset, assure that the --asset-name is used
	// TODO: make it so that the --asset-name is set for the root asset only even if multiple assets are there
	// This is a temporary fix that only works if there is only one asset
	if len(discoveredAssets.Assets) == 1 && invAssets[0].Name != "" && invAssets[0].Name != discoveredAssets.Assets[0].Asset.Name {
		log.Debug().Str("asset", discoveredAssets.Assets[0].Asset.Name).Msg("Overriding asset name with --asset-name flag")
		discoveredAssets.Assets[0].Asset.Name = invAssets[0].Name
	}

	return discoveredAssets, nil
}

func createRuntimeForAsset(asset *inventory.Asset, upstream *upstream.UpstreamConfig, recording llx.Recording) (*AssetWithRuntime, error) {
	var runtime *providers.Runtime
	var err error
	// Close the runtime if an error occured
	defer func() {
		if err != nil && runtime != nil {
			runtime.Close()
		}
	}()

	runtime, err = providers.Coordinator.RuntimeFor(asset, providers.DefaultRuntime())
	if err != nil {
		return nil, err
	}
	if err = runtime.SetRecording(recording); err != nil {
		return nil, err
	}

	err = runtime.Connect(&plugin.ConnectReq{
		Features: config.Features,
		Asset:    asset,
		Upstream: upstream,
	})
	if err != nil {
		return nil, err
	}
	return &AssetWithRuntime{Asset: runtime.Provider.Connection.Asset, Runtime: runtime}, nil
}

// prepareAsset prepares the asset for further processing by adding mondoo-specific labels and annotations
func prepareAsset(a *inventory.Asset, rootAsset *inventory.Asset, runtimeLabels map[string]string) {
	a.AddMondooLabels(rootAsset)
	a.AddAnnotations(rootAsset.GetAnnotations())
	a.ManagedBy = rootAsset.ManagedBy
	a.KindString = a.GetPlatform().Kind
	for k, v := range runtimeLabels {
		if a.Labels == nil {
			a.Labels = map[string]string{}
		}
		a.Labels[k] = v
	}
}
