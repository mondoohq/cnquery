// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/recording"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

var (
	_ plugin.StaticProvider = &recordingProvider{}
	_ plugin.ProviderPlugin = &recordingProvider{}

	recordingProviderInstance = Provider{
		Provider: &plugin.Provider{
			Name:    "recording",
			ID:      "go.mondoo.com/cnquery/v12/providers/recording",
			Version: "12.0.0",
			Connectors: []plugin.Connector{
				{
					Name:    "recording",
					MinArgs: 0,
					MaxArgs: 1,
					Use:     "recording [flags]",
					Short:   "read recording file on disk",
					Flags: []plugin.Flag{
						{
							Long: "path",
							Type: plugin.FlagType_String,
						},
					},
				},
			},
		},
	}
)

type recordingProvider struct {
	initialized bool
	recording   llx.Recording
	runtime     *plugin.Runtime
}

func (s *recordingProvider) StaticName() string {
	return "recording"
}

func (s *recordingProvider) Heartbeat(req *plugin.HeartbeatReq) (*plugin.HeartbeatRes, error) {
	return nil, nil
}

func (s *recordingProvider) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	filePath := ""

	fp := req.Flags["path"]
	if fp != nil && fp.RawData().Value.(string) != "" {
		filePath = fp.RawData().Value.(string)
	}
	return &plugin.ParseCLIRes{
		Asset: &inventory.Asset{
			Connections: []*inventory.Config{
				{
					Type: "recording",
					Path: filePath,
				},
			},
		},
	}, nil
}

func (s *recordingProvider) Connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing config")
	}
	log.Info().Interface("asset", req.GetAsset()).Msg("connected to recording provider")
	inv, err := s.detect(req.GetAsset())
	if err != nil {
		return nil, err
	}

	path := req.Asset.Connections[0].Path
	rec, err := recording.LoadRecordingFile(path)
	if err != nil {
		return nil, err
	}
	s.recording = rec
	res := &plugin.ConnectRes{
		Asset:     req.GetAsset(),
		Inventory: inv,
	}

	return res, nil
}

func (s *recordingProvider) detect(asset *inventory.Asset) (*inventory.Inventory, error) {
	if asset.GetPlatform() != nil {
		return nil, nil
	}
	path := asset.Connections[0].Path
	rec, err := recording.LoadRecordingFile(path)
	if err != nil {
		return nil, err
	}

	assets := []*inventory.Asset{}
	for _, a := range rec.Assets {
		invAsset := a.Asset.ToInventory()
		invAsset.Connections = []*inventory.Config{
			{
				Type: "recording",
				Path: path,
			},
		}
		assets = append(assets, invAsset)
	}

	inv := inventory.New(inventory.WithAssets(assets...))
	return inv, nil
}

func (s *recordingProvider) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	return nil, errors.New("the sbom provider does not support the mock connect call, this is an internal error")
}

func (s *recordingProvider) Disconnect(req *plugin.DisconnectReq) (*plugin.DisconnectRes, error) {
	return nil, nil
}

func (s *recordingProvider) Shutdown(req *plugin.ShutdownReq) (*plugin.ShutdownRes, error) {
	return nil, nil
}

func (s *recordingProvider) GetData(req *plugin.DataReq) (*plugin.DataRes, error) {
	data, ok := s.recording.GetData(0, req.GetResource(), req.GetResourceId(), req.GetField())
	if !ok {
		return nil, errors.New("resource '" + req.Resource + "' (id: " + req.ResourceId + ") doesn't exist")
	}

	if data.Type.IsResource() {
		dataRes := &plugin.DataRes{Data: data.Result().Data, Id: data.Value.(string)}
		return dataRes, nil
	}

	dataRes := &plugin.DataRes{Data: data.Result().Data}
	return dataRes, nil
}

func (s *recordingProvider) StoreData(req *plugin.StoreReq) (*plugin.StoreRes, error) {
	for _, r := range req.GetResources() {
		for _, f := range r.GetFields() {
			s.recording.AddData(1, r.GetName(), r.GetId(), f.CodeId, f.GetData().RawData())
		}
	}
	return &plugin.StoreRes{}, nil
}

func (s *recordingProvider) Init(running *RunningProvider) {
	if s.initialized {
		return
	}
	s.initialized = true
}
