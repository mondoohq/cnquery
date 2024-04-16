// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"context"
	"errors"

	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v11/providers/k8s/connection/admission"
	"go.mondoo.com/cnquery/v11/providers/k8s/connection/api"
	"go.mondoo.com/cnquery/v11/providers/k8s/connection/manifest"
	"go.mondoo.com/cnquery/v11/providers/k8s/connection/shared"
	connectionResources "go.mondoo.com/cnquery/v11/providers/k8s/connection/shared/resources"
	"go.mondoo.com/cnquery/v11/providers/k8s/resources"
)

const ConnectionType = "k8s"

type Service struct {
	*plugin.Service

	discoveryCache *connectionResources.DiscoveryCache
}

func Init() *Service {
	return &Service{
		Service:        plugin.NewService(),
		discoveryCache: connectionResources.NewDiscoveryCache(),
	}
}

func parseDiscover(flags map[string]*llx.Primitive) *inventory.Discovery {
	var targets []string
	if x, ok := flags["discover"]; ok && len(x.Array) != 0 {
		targets = make([]string, 0, len(x.Array))
		for i := range x.Array {
			entry := string(x.Array[i].Value)
			targets = append(targets, entry)
		}
	} else {
		targets = []string{"auto"}
	}
	return &inventory.Discovery{Targets: targets}
}

func (s *Service) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	flags := req.Flags
	if flags == nil {
		flags = map[string]*llx.Primitive{}
	}

	conf := &inventory.Config{
		Discover: parseDiscover(flags),
		Type:     req.Connector,
		Options:  map[string]string{},
	}

	if len(req.Args) == 1 {
		conf.Options[shared.OPTION_MANIFEST] = req.Args[0]
	}

	if context, ok := req.Flags["context"]; ok {
		conf.Options[shared.OPTION_CONTEXT] = string(context.Value)
	}

	if ns, ok := req.Flags[shared.OPTION_NAMESPACE]; ok {
		conf.Options[shared.OPTION_NAMESPACE] = string(ns.Value)
	}

	if ns, ok := req.Flags[shared.OPTION_NAMESPACE_EXCLUDE]; ok {
		conf.Options[shared.OPTION_NAMESPACE_EXCLUDE] = string(ns.Value)
	}

	asset := &inventory.Asset{
		Connections: []*inventory.Config{conf},
	}

	idDetector := "hostname"
	if flag, ok := flags["id-detector"]; ok {
		idDetector = string(flag.Value)
	}
	if idDetector != "" {
		asset.IdDetector = []string{idDetector}
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

	inventory, err := s.discover(conn, req.Features)
	if err != nil {
		return nil, err
	}

	return &plugin.ConnectRes{
		Id:        uint32(conn.ID()),
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

	runtime, err := s.AddRuntime(conf, func(connId uint32) (*plugin.Runtime, error) {
		var conn shared.Connection
		var err error
		if manifestContent, ok := conf.Options[shared.OPTION_IMMEMORY_CONTENT]; ok {
			conn, err = manifest.NewConnection(connId, asset, manifest.WithManifestContent([]byte(manifestContent)))
			if err != nil {
				return nil, err
			}
		} else if manifestFile, ok := conf.Options[shared.OPTION_MANIFEST]; ok {
			conn, err = manifest.NewConnection(connId, asset, manifest.WithManifestFile(manifestFile))
			if err != nil {
				return nil, err
			}
		} else if data, ok := conf.Options[shared.OPTION_ADMISSION]; ok {
			conn, err = admission.NewConnection(connId, asset, data)
			if err != nil {
				return nil, err
			}
		} else {
			conn, err = api.NewConnection(connId, asset, s.discoveryCache)
			if err != nil {
				return nil, err
			}
		}

		var upstream *upstream.UpstreamClient
		if req.Upstream != nil && !req.Upstream.Incognito {
			upstream, err = req.Upstream.InitClient(context.Background())
			if err != nil {
				return nil, err
			}
		}
		asset.Connections[0].Id = connId

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

	return runtime.Connection.(shared.Connection), nil
}

func (s *Service) discover(conn shared.Connection, features cnquery.Features) (*inventory.Inventory, error) {
	if conn.InventoryConfig().Discover == nil {
		return nil, nil
	}

	runtime, err := s.GetRuntime(conn.ID())
	if err != nil {
		return nil, err
	}

	return resources.Discover(runtime, features)
}
