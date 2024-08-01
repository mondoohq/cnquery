// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"context"
	"errors"
	"fmt"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/mondoo/connection"
	"go.mondoo.com/cnquery/v11/providers/mondoo/resources"
)

const (
	DefaultConnectionType = "mondoo"
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
	inventoryConfig := &inventory.Config{
		Type: req.Connector,
	}
	asset := inventory.Asset{
		Connections: []*inventory.Config{inventoryConfig},
	}
	return &plugin.ParseCLIRes{Asset: &asset}, nil
}

func (s *Service) Connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	if req == nil || req.Asset == nil {
		return nil, errors.New("no connection data provided")
	}

	conn, err := s.connect(req, callback)
	if err != nil {
		return nil, err
	}

	return &plugin.ConnectRes{
		Id:        conn.ID(),
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

	runtime, err := s.AddRuntime(conf, func(connId uint32) (*plugin.Runtime, error) {
		if req.Upstream == nil {
			return nil, errors.New("please provide Mondoo credentials (via Mondoo config) to use this provider")
		}

		upstream, err := req.Upstream.InitClient(context.Background())
		if err != nil {
			return nil, err
		}

		// This provider is treated as incognito for the time being
		conn, err := connection.New(connId, asset, conf, upstream)
		if err != nil {
			return nil, err
		}

		fillAsset(conn, asset)

		return plugin.NewRuntime(
			conn,
			callback,
			req.HasRecording,
			resources.CreateResource,
			resources.NewResource,
			resources.GetData,
			resources.SetData,
			nil), nil
	})
	if err != nil {
		return nil, err
	}

	return runtime.Connection.(*connection.Connection), nil
}

func fillAsset(conn *connection.Connection, asset *inventory.Asset) {
	name := connection.MrnBasenameOrMrn(conn.Upstream.SpaceMrn)
	asset.PlatformIds = []string{conn.Upstream.SpaceMrn}
	asset.Name = name
	asset.Connections[0].Id = conn.ID()

	if conn.Type == connection.ConnTypeSpace {
		asset.Name = fmt.Sprintf("Mondoo Space %s", name)
		asset.Platform = &inventory.Platform{
			Name:    "mondoo-space",
			Title:   "Mondoo Space",
			Family:  []string{},
			Kind:    "api",
			Runtime: "mondoo",
			Labels:  map[string]string{},
		}
	} else if conn.Type == connection.ConnTypeOrganization {
		asset.Name = fmt.Sprintf("Mondoo Organization %s", name)
		asset.Platform = &inventory.Platform{
			Name:    "mondoo-organization",
			Title:   "Mondoo Organization",
			Family:  []string{},
			Kind:    "api",
			Runtime: "mondoo",
			Labels:  map[string]string{},
		}
	}
}

func (s *Service) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	return nil, errors.New("don't support recording layers for the Mondoo provider at the moment")
}
