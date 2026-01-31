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
					MinArgs: 0,
					MaxArgs: 1,
					Use:     "recording [flags]",
					Short:   "read recording file from disk",
					Flags: []plugin.Flag{
						{
							Long: "recording-path",
							Type: plugin.FlagType_String,
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

// allows to set the connection id of the asset that is selected programmatically
func WithAssetConnectionId(id uint32) RecordingProviderOpt {
	return func(rp *recordingProvider) {
		rp.assetConnectionId = id
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
	// the id of the connection of the asset that is selected
	assetConnectionId uint32
	recording         llx.Recording
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
	// NOTE: this is not an ideal solutionsince we rely on the fact that the shell will
	// set the DelayDiscovery flag to false when a single asset is selected.
	// We cannot use the connection id as the shell will pass in 0 (since it has just one open connection)
	// but assets in the recording are stored under different connection ids.
	// To fix this properly, we need more information about the asset in the Connect request.
	if asset.GetPlatform() != nil && !asset.Connections[0].DelayDiscovery {
		scopeDownRecording, err := s.recording.ScopeToAsset(asset)
		if err != nil {
			return nil, err
		}
		s.recording = scopeDownRecording
		s.assetConnectionId = asset.Connections[0].Id
		return nil, nil
	}

	assets := []*inventory.Asset{}
	for _, a := range s.recording.GetAssets() {
		if len(a.Connections) == 0 {
			continue
		}
		a.Connections = []*inventory.Config{
			{
				Type:           "recording",
				Id:             a.Connections[0].Id,
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
	data, ok := s.recording.GetData(s.assetConnectionId, resource, id, field)
	if !ok {
		errMsg := fmt.Sprintf("resource %s ( id: %s ) doesn't exist", resource, id)
		if f := field; f != "" {
			errMsg = fmt.Sprintf("resource %s ( id: %s, field: %s ) doesn't exist", resource, id, f)
		}

		return nil, errors.New(errMsg)
	}

	res := data.Result().Data
	return &plugin.DataRes{Data: res}, nil
}

func (s *recordingProvider) StoreData(req *plugin.StoreReq) (*plugin.StoreRes, error) {
	return nil, errors.New("the recording provider does not support storing data, this is an internal error")
}
