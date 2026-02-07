// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"fmt"

	"github.com/cockroachdb/errors"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/recording"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

var (
	_ plugin.ProviderPlugin = &recordingProvider{}
	_ plugin.StaticProvider = &recordingProvider{}

	recordingProviderInstance = Provider{
		Provider: &plugin.Provider{
			Name:            "recording",
			ID:              "go.mondoo.com/cnquery/v12/providers/recording",
			Version:         "12.0.0",
			ConnectionTypes: []string{"recording"},
			Connectors: []plugin.Connector{
				{
					Name:    "recording",
					Use:     "recording [flags]",
					MinArgs: 0,
					MaxArgs: 1,
					Short:   "read recording file from disk",
					Flags: []plugin.Flag{
						{
							Long: "recording-path",
							Type: plugin.FlagType_String,
							Desc: "path to the recording file",
						},
					},
				},
			},
		},
	}
)

func (*recordingProvider) StaticName() string {
	return "recording"
}

type RecordingProviderOpt func(*recordingProvider)

// allows to set a custom recording implementation programmatically
func WithRecording(rec llx.Recording) RecordingProviderOpt {
	return func(rp *recordingProvider) {
		rp.recording = rec
	}
}

// allows to set the selected asset
func WithAsset(asset *inventory.Asset) RecordingProviderOpt {
	return func(rp *recordingProvider) {
		rp.selectedAsset = asset
	}
}

func NewRecordingProvider(opts ...RecordingProviderOpt) *recordingProvider {
	rp := &recordingProvider{}
	for _, o := range opts {
		o(rp)
	}
	return rp
}

type recordingProvider struct {
	selectedAsset *inventory.Asset
	recording     llx.Recording
}

func (s *recordingProvider) Heartbeat(req *plugin.HeartbeatReq) (*plugin.HeartbeatRes, error) {
	return nil, nil
}

func (s *recordingProvider) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	filePath := ""

	pathFlag := req.Flags["recording-path"]
	if pathFlag != nil && pathFlag.RawData().Value.(string) != "" {
		filePath = pathFlag.RawData().Value.(string)
	}

	asset := &inventory.Asset{
		Connections: []*inventory.Config{
			{
				Type: "recording",
				Path: filePath,
			},
		},
	}

	res := &plugin.ParseCLIRes{
		Asset: asset,
	}
	return res, nil
}

func (s *recordingProvider) Connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing config")
	}

	// initial connection, expecting the recording file path as part of the config
	if s.recording == nil {
		path := req.Asset.Connections[0].Path
		rec, err := recording.LoadRecordingFile(path)
		if err != nil {
			return nil, err
		}
		s.recording = rec
	}

	inv, err := s.detect(req.GetAsset())
	if err != nil {
		return nil, err
	}
	res := &plugin.ConnectRes{
		Asset:     req.GetAsset(),
		Inventory: inv,
	}

	return res, nil
}

func (s *recordingProvider) detect(asset *inventory.Asset) (*inventory.Inventory, error) {
	if asset.GetPlatform() != nil {
		if !asset.Connections[0].DelayDiscovery {
			if asset.GetMrn() == "" && len(asset.GetPlatformIds()) == 0 {
				return nil, errors.New("missing mrn or platform ids for asset selection")
			}
			s.selectedAsset = asset
		}
		return nil, nil
	}

	assets := []*inventory.Asset{}
	for _, a := range s.recording.GetAssets() {
		a.Connections = []*inventory.Config{
			{
				Type:           "recording",
				DelayDiscovery: true,
			},
		}
		assets = append(assets, a)
	}

	inv := inventory.New(inventory.WithAssets(assets...))
	return inv, nil
}

func (s *recordingProvider) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	return nil, errors.New("the recording provider does not support the mock connect call, this is an internal error")
}

func (s *recordingProvider) Disconnect(req *plugin.DisconnectReq) (*plugin.DisconnectRes, error) {
	return nil, nil
}

func (s *recordingProvider) Shutdown(req *plugin.ShutdownReq) (*plugin.ShutdownRes, error) {
	return nil, nil
}

func (s *recordingProvider) GetData(req *plugin.DataReq) (*plugin.DataRes, error) {
	resource := req.GetResource()
	id := req.GetResourceId()
	field := req.GetField()
	lookup := llx.AssetRecordingLookup{}
	if s.selectedAsset != nil {
		if s.selectedAsset.GetMrn() != "" {
			lookup.Mrn = s.selectedAsset.GetMrn()
		}
		if len(s.selectedAsset.GetPlatformIds()) > 0 {
			lookup.PlatformIds = s.selectedAsset.GetPlatformIds()
		}
	}
	data, ok := s.recording.GetData(lookup, resource, id, field)
	if !ok {
		errMsg := fmt.Sprintf("resource %s (id: %s) doesn't exist", resource, id)
		if f := field; f != "" {
			// prettify the error message if we're asking for a field
			errMsg = fmt.Sprintf("resource %s (id: %s, field: %s) doesn't exist", resource, id, f)
		}

		return nil, errors.New(errMsg)
	}

	res := data.Result().Data
	return &plugin.DataRes{Data: res}, nil
}

func (s *recordingProvider) StoreData(req *plugin.StoreReq) (*plugin.StoreRes, error) {
	return nil, errors.New("the recording provider does not support storing data, this is an internal error")
}
