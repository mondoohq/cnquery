// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"errors"
	"os"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v10/providers/equinix/connection"
	"go.mondoo.com/cnquery/v10/providers/equinix/resources"
)

const ConnectionType = "equinix"

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

	if len(req.Args) != 2 {
		return nil, errors.New("missing argument, use `equinix project <project-id>`")
	}

	conf := &inventory.Config{
		Type:    req.Connector,
		Options: make(map[string]string),
	}

	// custom flag parsing
	token := ""
	if x, ok := flags["token"]; ok && len(x.Value) != 0 {
		token = string(x.Value)
	}
	if token == "" {
		token = os.Getenv("PACKET_AUTH_TOKEN")
	}
	if token == "" {
		return nil, errors.New("no slack token provided, use --token or PACKET_AUTH_TOKEN")
	}
	conf.Credentials = append(conf.Credentials, vault.NewPasswordCredential("", token))

	switch req.Args[0] {
	case "org":
		conf.Options["org-id"] = req.Args[1]
	case "project":
		conf.Options["project-id"] = req.Args[1]
	default:
		return nil, errors.New("invalid argument, use `equinix project <project-id>`")
	}

	asset := inventory.Asset{
		Connections: []*inventory.Config{conf},
	}

	return &plugin.ParseCLIRes{Asset: &asset}, nil
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
		Id:        conn.ID(),
		Name:      conn.Name(),
		Asset:     req.Asset,
		Inventory: nil,
	}, nil
}

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*connection.EquinixConnection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]
	runtime, err := s.AddRuntime(func(connId uint32) (*plugin.Runtime, error) {
		conn, err := connection.NewEquinixConnection(connId, asset, conf)
		if err != nil {
			return nil, err
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

	return runtime.Connection.(*connection.EquinixConnection), nil
}

func (s *Service) detect(asset *inventory.Asset, conn *connection.EquinixConnection) error {
	asset.Name = conn.Conf.Host

	asset.Platform = &inventory.Platform{
		Name:  "equinix",
		Kind:  "api",
		Title: "Equinix Metal",
	}

	id, err := conn.Identifier()
	if err != nil {
		return err
	}
	asset.PlatformIds = []string{id}
	return nil
}
