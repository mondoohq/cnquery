// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"context"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v10/explorer/resources"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/recording"
)

var mockProvider = Provider{
	Provider: &plugin.Provider{
		Name:    "mock",
		ID:      "go.mondoo.com/cnquery/v10/providers/mock",
		Version: "9.0.0",
		Connectors: []plugin.Connector{
			{
				Name:  "mock",
				Use:   "mock",
				Short: "use a recording without an active connection",
			},
			{
				Name:     "upstream",
				Use:      "upstream",
				Short:    "use upstream asset data",
				IsHidden: true,
				Flags: []plugin.Flag{
					{
						Long: "asset",
						Type: plugin.FlagType_String,
						Desc: "the upstream asset MRN to connect with",
					},
				},
			},
		},
	},
}

type mockProviderService struct {
	initialized bool
	runtime     *Runtime
}

func (s *mockProviderService) Heartbeat(req *plugin.HeartbeatReq) (*plugin.HeartbeatRes, error) {
	return nil, nil
}

func (s *mockProviderService) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	if req.Connector == "upstream" {
		return s.parseUpstreamCLI(req)
	}

	return &plugin.ParseCLIRes{
		Asset: &inventory.Asset{
			Connections: []*inventory.Config{
				{
					Type: "mock",
				},
			},
		},
	}, nil
}

func (s *mockProviderService) parseUpstreamCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	assetMrn := req.Flags["asset"]
	if assetMrn == nil || len(assetMrn.Value) == 0 {
		return nil, errors.New("please specify an asset MRN")
	}

	return &plugin.ParseCLIRes{
		Asset: &inventory.Asset{
			Mrn: string(assetMrn.Value),
			Connections: []*inventory.Config{
				{
					Type: "mock",
				},
			},
		},
	}, nil
}

// TODO: Replace this entire call with a detector
func assetinfo2providerName(asset *inventory.Asset) (string, error) {
	if asset == nil {
		return "", errors.New("no asset information provided to infer a provider")
	}
	if asset.Platform == nil {
		return "", errors.New("no asset platform information provided to infer a provider")
	}

	switch asset.Platform.Kind {
	case "container-image", "baremetal":
		return "os", nil
	}

	switch asset.Platform.Name {
	case "aws":
		return "aws", nil
	case "azure":
		return "azure", nil
	case "gcp":
		return "gcp", nil
	}

	return "", errors.New("cannot determine provider type for this upstream asset: " + asset.Platform.Name)
}

func (s *mockProviderService) Connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	// If an upstream asset was requested
	if req.Asset.Mrn != "" {
		if req.Upstream == nil {
			return nil, errors.New("missing an upstream configuration")
		}

		ctx := context.Background()
		client, err := req.Upstream.InitClient(ctx)
		if err != nil {
			return nil, err
		}

		explorer, err := resources.NewRemoteServices(client.ApiEndpoint, client.Plugins, client.HttpClient)
		if err != nil {
			return nil, err
		}

		urecording, err := recording.NewUpstreamRecording(ctx, explorer, req.Asset.Mrn)
		if err != nil {
			return nil, err
		}

		asset := urecording.Asset()
		providerName, err := assetinfo2providerName(asset)
		if err != nil {
			return nil, err
		}

		provider := Coordinator.Providers().Lookup(ProviderLookup{ProviderName: providerName})
		if provider == nil {
			return nil, errors.New("failed to look up provider for upstream recording with name=" + providerName)
		}

		addedProvider, err := s.runtime.addProvider(provider.ID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to init provider for connection in recording")
		}

		conn, err := addedProvider.Instance.Plugin.MockConnect(&plugin.ConnectReq{
			Asset:    asset,
			Features: req.Features,
			Upstream: req.Upstream,
		}, callback)
		if err != nil {
			return nil, errors.Wrap(err, "failed to init referenced provider")
		}

		addedProvider.Connection = conn
		err = s.runtime.SetRecording(urecording)
		return conn, err
	}

	// initialize all other providers from all asset connections in the recording
	multiRecording, ok := s.runtime.Recording().(recording.MultiAsset)
	if !ok {
		return nil, errors.New("cannot find assets in recording for mock provider")
	}

	assetRecordings := multiRecording.GetAssetRecordings()
	if len(assetRecordings) == 0 {
		return nil, errors.New("no assets found in recording for mock provider")
	}

	asset := assetRecordings[0]
	if len(asset.Connections) == 0 {
		return nil, errors.New("no connections found in asset")
	}

	var res *plugin.ConnectRes
	for i := range asset.Connections {
		conf := asset.Connections[i]

		provider, err := s.runtime.addProvider(conf.ProviderID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to init provider for connection in recording")
		}

		conn, err := provider.Instance.Plugin.MockConnect(&plugin.ConnectReq{
			Asset:    asset.Asset.ToInventory(),
			Features: req.Features,
			Upstream: req.Upstream,
		}, callback)
		if err != nil {
			return nil, errors.Wrap(err, "failed to init referenced provider")
		}

		provider.Connection = conn
		if i == 0 {
			res = conn
		}
	}

	return res, nil
}

func (s *mockProviderService) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	// Should never happen: the mock provider should not be called with MockConnect.
	// It is the only thing that should ever call MockConnect to other providers
	// (outside of tests).
	return nil, errors.New("the mock provider does not support the mock connect call, this is an internal error")
}

func (s *mockProviderService) Disconnect(req *plugin.DisconnectReq) (*plugin.DisconnectRes, error) {
	// Nothing to do yet...
	return nil, nil
}

func (s *mockProviderService) Shutdown(req *plugin.ShutdownReq) (*plugin.ShutdownRes, error) {
	// Nothing to do yet...
	return nil, nil
}

func (s *mockProviderService) GetData(req *plugin.DataReq) (*plugin.DataRes, error) {
	panic("NO")
}

func (s *mockProviderService) StoreData(req *plugin.StoreReq) (*plugin.StoreRes, error) {
	panic("NO")
}

func (s *mockProviderService) Init(running *RunningProvider) {
	if s.initialized {
		return
	}
	s.initialized = true

	// TODO: Currently not needed, as the coordinator loads all schemas right now.
	// Once it doesn't do that anymore, remember to load all schemas here
	// rt.schema.unsafeLoadAll()
	// rt.schema.unsafeRefresh()
}
