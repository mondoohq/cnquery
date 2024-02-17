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

func (d *DiscoveredAssets) GetAssetsByPlatformID(platformID string) []*AssetWithRuntime {
	var assets []*AssetWithRuntime
	for _, a := range d.Assets {
		for _, p := range a.Asset.PlatformIds {
			if platformID == "" || p == platformID {
				assets = append(assets, a)
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
			discoveredAssets.AddError(resolvedRootAsset, err)
			continue
		}

		resolvedRootAsset = rootAssetWithRuntime.Asset // to ensure we get all the information the connect call gave us

		// Make sure the root runtime is closed at the end of the loop if needed. This will close the runtimes for all
		// root assets when the entire loop is done. It is NOT running the close after the current iteration of the loop.
		// This behaviour is fine for now. If we want to close the runtime after each iteration, we need to revisit
		closeRootRuntime := false
		defer func() {
			if closeRootRuntime {
				rootAssetWithRuntime.Runtime.Close()
			}
		}()

		// If the root asset has platform IDs, then it is a scannable asset, so we need to add it
		if len(resolvedRootAsset.PlatformIds) > 0 {
			prepareAsset(resolvedRootAsset, resolvedRootAsset, runtimeLabels)
			if !discoveredAssets.Add(rootAssetWithRuntime.Asset, rootAssetWithRuntime.Runtime) {
				closeRootRuntime = true
			}
		} else {
			closeRootRuntime = true
		}

		// If there is no inventory, no assets have been discovered under the root asset
		if rootAssetWithRuntime.Runtime.Provider.Connection.Inventory == nil {
			continue
		}

		// for all discovered assets, we apply mondoo-specific labels and annotations that come from the root asset
		for _, a := range rootAssetWithRuntime.Runtime.Provider.Connection.Inventory.Spec.Assets {
			// create runtime for root asset
			assetWithRuntime, err := createRuntimeForAsset(a, upstream, recording)
			if err != nil {
				log.Error().Err(err).Str("asset", a.Name).Msg("unable to create runtime for asset")
				discoveredAssets.AddError(a, err)
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
	if a.Labels == nil {
		a.Labels = map[string]string{}
	}
	for k, v := range runtimeLabels {
		a.Labels[k] = v
	}
}
