// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1
package provider

import (
	"context"
	"errors"

	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v12/providers/tailscale/connection"
	"go.mondoo.com/cnquery/v12/providers/tailscale/resources"
)

const DefaultConnectionType = "tailscale"

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

	// Pass all flags to the config
	if v, ok := flags[connection.OPTION_BASE_URL]; ok && len(v.Value) != 0 {
		conf.Options[connection.OPTION_BASE_URL] = string(v.Value)
	}
	if v, ok := flags[connection.OPTION_CLIENT_ID]; ok && len(v.Value) != 0 {
		conf.Options[connection.OPTION_CLIENT_ID] = string(v.Value)
	}
	if v, ok := flags[connection.OPTION_CLIENT_SECRET]; ok && len(v.Value) != 0 {
		conf.Options[connection.OPTION_CLIENT_SECRET] = string(v.Value)
	}
	if v, ok := flags[connection.OPTION_TOKEN]; ok && len(v.Value) != 0 {
		conf.Credentials = append(conf.Credentials, vault.NewPasswordCredential("", string(v.Value)))
	}

	// Support to connect to a different tailnet
	if len(req.Args) > 0 {
		conf.Options[connection.OPTION_TAILNET] = req.Args[0]
	}

	return &plugin.ParseCLIRes{
		Asset: &inventory.Asset{
			Name:        "Tailscale",
			Connections: []*inventory.Config{conf},
		},
	}, nil
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

	inv, err := s.discover(conn)
	if err != nil {
		return nil, err
	}

	return &plugin.ConnectRes{
		Id:        conn.ID(),
		Name:      conn.Name(),
		Asset:     req.Asset,
		Inventory: inv,
	}, nil
}

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*connection.TailscaleConnection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]
	runtime, err := s.AddRuntime(conf, func(connId uint32) (*plugin.Runtime, error) {
		conn, err := connection.NewTailscaleConnection(connId, asset, conf)
		if err != nil {
			return nil, err
		}

		// Verify access to Tailscale organization
		if err := conn.Verify(); err != nil {
			return nil, err
		}

		var upstream *upstream.UpstreamClient
		if req.Upstream != nil && !req.Upstream.Incognito {
			upstream, err = req.Upstream.InitClient(context.Background())
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

	return runtime.Connection.(*connection.TailscaleConnection), nil
}

func (s *Service) detect(asset *inventory.Asset, conn *connection.TailscaleConnection) error {
	asset.Id = conn.Conf.Type
	asset.Name = conn.Conf.Host

	platform, err := conn.PlatformInfo()
	if err != nil {
		return err
	}

	asset.Platform = platform
	asset.PlatformIds = []string{conn.Identifier()}
	return nil
}

func (s *Service) discover(conn *connection.TailscaleConnection) (*inventory.Inventory, error) {
	conf := conn.Asset().Connections[0]
	if conf.Discover == nil {
		return nil, nil
	}

	runtime, err := s.GetRuntime(conn.ID())
	if err != nil {
		return nil, err
	}

	return resources.Discover(runtime, conf.Options)
}

func (s *Service) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	return nil, errors.New("mock connect not yet implemented")
}
