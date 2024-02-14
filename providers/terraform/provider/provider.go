// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"errors"
	"strconv"

	"github.com/rs/zerolog/log"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v10/providers/terraform/connection"
	"go.mondoo.com/cnquery/v10/providers/terraform/resources"
)

const (
	// asset connection types, used internally
	StateConnectionType  = "terraform-state"
	PlanConnectionType   = "terraform-plan"
	HclConnectionType    = "terraform-hcl"
	HclGitConnectionType = "terraform-hcl-git"
	// CLI keywords, e.g. `<binary> run terraform plan file.json ...`
	CLIPlan  = "plan"
	CLIHcl   = "hcl"
	CLIState = "state"
)

type Service struct {
	*plugin.Service
}

func Init() *Service {
	return &Service{
		Service: plugin.NewService(),
	}
}

func (s *Service) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	flags := req.Flags
	if flags == nil {
		flags = map[string]*llx.Primitive{}
	}

	conf := &inventory.Config{
		Type:    req.Connector,
		Options: map[string]string{},
	}

	// TODO: somewhere here, parse the args for the previous sub-commands
	// perhaps set the conn.Type here depending on the args (like in the os provider)
	// and later on decide which thing to call based on the conn.Type
	// below in this file we already have something similar:
	// tc.Options["asset-type"] == "state"
	action := req.Args[0]
	switch action {
	case CLIPlan, CLIHcl, CLIState:
		switch action {
		case CLIPlan:
			conf.Type = PlanConnectionType
		case CLIHcl:
			conf.Type = HclConnectionType
		case CLIState:
			conf.Type = StateConnectionType
		}

		if len(req.Args) < 2 {
			return nil, errors.New("no path provided")
		}
		conf.Options["path"] = req.Args[1]

	default:
		if len(req.Args) > 1 {
			return nil, errors.New("unknown set of arguments, use 'state <path>', 'plan <path>' or 'hcl <path>'")
		}
		conf.Type = HclConnectionType
		conf.Options["path"] = req.Args[0]
	}

	if x, ok := flags["ignore-dot-terraform"]; ok {
		if x != nil {
			conf.Options["ignore-dot-terraform"] = strconv.FormatBool(x.RawData().Value.(bool))
			log.Info().Msg("user requested to ignore .terraform")
		}
	}

	asset := &inventory.Asset{
		Connections: []*inventory.Config{conf},
	}

	res := plugin.ParseCLIRes{
		Asset: asset,
	}

	return &res, nil
}

func (s *Service) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	return nil, errors.New("mock connect not yet implemented")
}

func (s *Service) Connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	if req == nil || req.Asset == nil {
		return nil, errors.New("no connection data provided")
	}

	conn, err := s.connect(req, callback)
	if err != nil {
		return nil, err
	}

	// We only need to run the detection step when we don't have any asset information yet.
	if req.Asset.Platform == nil {
		if err := s.detect(req.Asset, conn); err != nil {
			return nil, err
		}
	}

	return &plugin.ConnectRes{
		Id:        uint32(conn.ID()),
		Name:      conn.Name(),
		Asset:     req.Asset,
		Inventory: nil,
	}, nil
}

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*connection.Connection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]

	runtime, err := s.AddRuntime(func(connId uint32) (*plugin.Runtime, error) {
		var conn *connection.Connection
		var err error

		switch conf.Type {
		case HclConnectionType:
			conn, err = connection.NewHclConnection(connId, asset)
			if err != nil {
				return nil, err
			}

		case StateConnectionType:
			conn, err = connection.NewStateConnection(connId, asset)
			if err != nil {
				return nil, err
			}

		case PlanConnectionType:
			conn, err = connection.NewPlanConnection(connId, asset)
			if err != nil {
				return nil, err
			}

		case HclGitConnectionType:
			conn, err = connection.NewHclGitConnection(connId, asset)
			if err != nil {
				return nil, err
			}

		default:
			return nil, errors.New("cannot find connection type " + conf.Type)
		}

		var upstream *upstream.UpstreamClient
		if req.Upstream != nil && !req.Upstream.Incognito {
			upstream, err = req.Upstream.InitClient()
			if err != nil {
				return nil, err
			}
		}

		asset.Connections[0].Id = conn.ID()
		return plugin.NewRuntime(
			conn,
			callback,
			req.HasRecording,
			resources.CreateResource,
			resources.NewResource,
			resources.GetData,
			resources.SetData,
			upstream), nil
	})
	if err != nil {
		return nil, err
	}

	return runtime.Connection.(*connection.Connection), nil
}
