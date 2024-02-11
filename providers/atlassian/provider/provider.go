// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"errors"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v10/providers/atlassian/connection"
	"go.mondoo.com/cnquery/v10/providers/atlassian/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/atlassian/resources"
)

const (
	defaultConnection     uint32 = 1
	DefaultConnectionType        = "atlassian"
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

	if len(req.Args) == 0 {
		return nil, errors.New("missing argument, use `atlassian jira`, `atlassian admin`, `atlassian confluence`, or `atlassian scim {directoryID}`")
	}

	if req.Args[0] == "scim" {
		if len(req.Args) != 2 {
			return nil, errors.New("missing argument, scim requires a directory id `atlassian scim {directoryID}`")
		}
	}

	conf := &inventory.Config{
		Type:    req.Connector,
		Options: map[string]string{},
	}

	// flags
	if x, ok := flags["user-token"]; ok && len(x.Value) != 0 {
		conf.Options["user-token"] = string(x.Value)
	}
	if x, ok := flags["user"]; ok && len(x.Value) != 0 {
		conf.Options["user"] = string(x.Value)
	}
	if x, ok := flags["host"]; ok && len(x.Value) != 0 {
		conf.Options["host"] = string(x.Value)
	}
	if x, ok := flags["admin-token"]; ok && len(x.Value) != 0 {
		conf.Options["admin-token"] = string(x.Value)
	}
	if x, ok := flags["scim-token"]; ok && len(x.Value) != 0 {
		conf.Options["scim-token"] = string(x.Value)
	}

	// discovery flags
	discoverTargets := []string{}
	if x, ok := flags["discover"]; ok && len(x.Array) != 0 {
		for i := range x.Array {
			entry := string(x.Array[i].Value)
			discoverTargets = append(discoverTargets, entry)
		}
	} else {
		discoverTargets = []string{"auto"}
	}
	conf.Discover = &inventory.Discovery{Targets: discoverTargets}

	switch req.Args[0] {
	case "admin":
		conf.Options["product"] = req.Args[0]
	case "jira":
		conf.Options["product"] = req.Args[0]
	case "confluence":
		conf.Options["product"] = req.Args[0]
	case "scim":
		conf.Options["product"] = req.Args[0]
		conf.Options["directory-id"] = req.Args[1]
	}

	asset := inventory.Asset{
		Connections: []*inventory.Config{conf},
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

	// We only need to run the detection step when we don't have any asset information yet.
	if req.Asset.Platform == nil {
		if err := s.detect(req.Asset, conn); err != nil {
			return nil, err
		}
	}

	inventory := &inventory.Inventory{
		Spec: &inventory.InventorySpec{
			Assets: []*inventory.Asset{req.Asset},
		},
	}

	return &plugin.ConnectRes{
		Id:        conn.ID(),
		Name:      conn.Name(),
		Asset:     req.Asset,
		Inventory: inventory,
	}, nil
}

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (shared.Connection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]
	runtime, err := s.AddRuntime(func(connId uint32) (*plugin.Runtime, error) {
		conn, err := connection.NewConnection(connId, asset, conf)
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

		return &plugin.Runtime{
			Connection:     conn,
			Callback:       callback,
			HasRecording:   req.HasRecording,
			CreateResource: resources.CreateResource,
			NewResource:    resources.NewResource,
			GetData:        resources.GetData,
			SetData:        resources.SetData,
			Upstream:       upstream,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	return runtime.Connection.(shared.Connection), nil
}

func (s *Service) detect(asset *inventory.Asset, conn shared.Connection) error {
	asset.Id = string(conn.Type())
	asset.Name = conn.Name()

	asset.Platform = conn.PlatformInfo()
	asset.PlatformIds = []string{conn.PlatformID()}
	return nil
}

func (s *Service) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	return nil, errors.New("mock connect not yet implemented")
}
