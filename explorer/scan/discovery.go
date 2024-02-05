package scan

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10"
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
	assets      []*AssetWithRuntime
	errors      []*AssetWithError
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

	d.assets = append(d.assets, &AssetWithRuntime{Asset: asset, Runtime: runtime})
	return true
}

func (d *DiscoveredAssets) AddError(asset *inventory.Asset, err error) {
	d.errors = append(d.errors, &AssetWithError{Asset: asset, Err: err})
}

func (d *DiscoveredAssets) GetAssets() []*inventory.Asset {
	assets := make([]*inventory.Asset, 0, len(d.assets))
	for _, a := range d.assets {
		assets = append(assets, a.Asset)
	}
	return assets
}

func (d *DiscoveredAssets) GetAssetsByPlatformID(platformID string) []*inventory.Asset {
	var assets []*inventory.Asset
	for _, a := range d.assets {
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

		rootRuntime, err := providers.Coordinator.RuntimeFor(resolvedRootAsset, providers.DefaultRuntime())
		if err != nil {
			log.Error().Err(err).Str("asset", resolvedRootAsset.Name).Msg("unable to create runtime for asset")
			discoveredAssets.AddError(resolvedRootAsset, err)
			continue
		}
		if err := rootRuntime.SetRecording(recording); err != nil {
			discoveredAssets.AddError(resolvedRootAsset, err)
			continue
		}

		if err := rootRuntime.Connect(&plugin.ConnectReq{
			Features: cnquery.GetFeatures(ctx),
			Asset:    resolvedRootAsset,
			Upstream: upstream,
		}); err != nil {
			log.Error().Err(err).Msg("unable to connect to asset")
			discoveredAssets.AddError(resolvedRootAsset, err)
			continue
		}
		resolvedRootAsset = rootRuntime.Provider.Connection.Asset // to ensure we get all the information the connect call gave us

		// for all discovered assets, we apply mondoo-specific labels and annotations that come from the root asset
		for _, a := range rootRuntime.Provider.Connection.Inventory.Spec.Assets {
			runtime, err := providers.Coordinator.RuntimeFor(resolvedRootAsset, providers.DefaultRuntime())
			if err != nil {
				discoveredAssets.AddError(a, err)
				continue
			}
			if err := runtime.SetRecording(recording); err != nil {
				discoveredAssets.AddError(resolvedRootAsset, err)
				runtime.Close()
				continue
			}

			err = runtime.Connect(&plugin.ConnectReq{
				Features: config.Features,
				Asset:    a,
				Upstream: upstream,
			})
			if err != nil {
				discoveredAssets.AddError(a, err)
				runtime.Close()
				continue
			}

			resolvedAsset := runtime.Provider.Connection.Asset
			prepareAsset(resolvedAsset, resolvedRootAsset, runtimeLabels)

			// If the asset has been already added, we should close its runtime
			if !discoveredAssets.Add(resolvedAsset, runtime) {
				runtime.Close()
			}
		}
	}

	// if there is exactly one asset, assure that the --asset-name is used
	// TODO: make it so that the --asset-name is set for the root asset only even if multiple assets are there
	// This is a temporary fix that only works if there is only one asset
	if len(discoveredAssets.assets) == 1 && invAssets[0].Name != "" && invAssets[0].Name != discoveredAssets.assets[0].Asset.Name {
		log.Debug().Str("asset", discoveredAssets.assets[0].Asset.Name).Msg("Overriding asset name with --asset-name flag")
		discoveredAssets.assets[0].Asset.Name = invAssets[0].Name
	}

	return discoveredAssets, nil
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
