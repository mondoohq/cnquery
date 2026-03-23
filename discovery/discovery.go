// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package discovery

import (
	"maps"

	"go.mondoo.com/mql/v13/cli/config"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers"
	inventory "go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"google.golang.org/protobuf/proto"
)

type AssetWithRuntime struct {
	Asset   *inventory.Asset
	Runtime *providers.Runtime
}

type AssetWithError struct {
	Asset *inventory.Asset
	Err   error
}

func createRuntimeForAsset(asset *inventory.Asset, upstream *upstream.UpstreamConfig, recording llx.Recording) (*AssetWithRuntime, error) {
	var runtime *providers.Runtime
	var err error
	// Close the runtime if an error occurred
	defer func() {
		if err != nil && runtime != nil {
			runtime.Close()
		}
	}()

	runtime, err = providers.Coordinator.RuntimeFor(asset, providers.DefaultRuntime())
	if err != nil {
		return nil, err
	}

	// If the runtime already has a connection, it means we have a duplicate asset
	if runtime.Provider.Connection != nil {
		return nil, nil
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

	// Clone the asset to create an independent snapshot. The runtime's connection
	// asset may be subject to mutation during subsequent provider connections, so
	// we ensure each discovered asset has its own copy of the platform metadata.
	connAsset := runtime.Provider.Connection.Asset
	clonedAsset := proto.Clone(connAsset).(*inventory.Asset)

	return &AssetWithRuntime{Asset: clonedAsset, Runtime: runtime}, nil
}

// prepareAsset prepares the asset for further processing by adding mondoo-specific labels and annotations
func prepareAsset(a *inventory.Asset, rootAsset *inventory.Asset, runtimeLabels map[string]string) {
	a.AddMondooLabels(rootAsset)
	a.AddAnnotations(rootAsset.GetAnnotations())
	a.ManagedBy = rootAsset.ManagedBy
	a.TraceId = rootAsset.TraceId
	if platform := a.GetPlatform(); platform != nil {
		a.KindString = platform.Kind
	}
	if a.Labels == nil {
		a.Labels = map[string]string{}
	}
	maps.Copy(a.Labels, runtimeLabels)
}
