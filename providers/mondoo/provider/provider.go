// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"context"
	"errors"
	"strings"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/mrn"
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
	flags := req.Flags
	if flags == nil {
		flags = map[string]*llx.Primitive{}
	}

	inventoryConfig := &inventory.Config{
		Type: req.Connector,
	}
	asset := inventory.Asset{
		Connections: []*inventory.Config{inventoryConfig},
	}
	return &plugin.ParseCLIRes{Asset: &asset}, nil
}

func parseFlagsToFiltersOpts(m map[string]*llx.Primitive) map[string]string {
	o := make(map[string]string, 0)

	if x, ok := m["filters"]; ok && len(x.Map) != 0 {
		for k, v := range x.Map {
			if strings.Contains(k, "tag:") {
				o[k] = string(v.Value)
			}
			if k == "instance-id" {
				o[k] = string(v.Value)
			}
			if strings.Contains(k, "region") {
				o[k] = string(v.Value)
			}
		}
	}

	return o
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

		name := mrnBasenameOrMrn(upstream.SpaceMrn)

		asset.Connections[0].Id = connId
		asset.Name = "Mondoo Space " + name
		asset.PlatformIds = []string{upstream.SpaceMrn}
		asset.Platform = &inventory.Platform{
			Name:    "mondoo-space",
			Title:   "Mondoo Space",
			Family:  []string{},
			Kind:    "api",
			Runtime: "mondoo",
			Labels:  map[string]string{},
		}

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

	if err := s.initResources(runtime); err != nil {
		return nil, err
	}

	return runtime.Connection.(*connection.Connection), nil
}

func mrnBasenameOrMrn(m string) string {
	parsed, err := mrn.NewMRN(m)
	if err != nil {
		return m
	}
	base := parsed.Basename()
	if base == "" {
		return m
	}
	return base
}

func (s *Service) initResources(runtime *plugin.Runtime) error {
	conn := runtime.Connection.(*connection.Connection)
	var err error

	_, err = resources.CreateResource(runtime, "mondoo.client", map[string]*llx.RawData{
		"mrn": llx.StringData(conn.Upstream.Creds.Mrn),
	})
	if err != nil {
		return err
	}

	spaceMrn := conn.Upstream.SpaceMrn
	_, err = resources.CreateResource(runtime, "mondoo.space", map[string]*llx.RawData{
		"name": llx.StringData(mrnBasenameOrMrn(spaceMrn)),
		"mrn":  llx.StringData(spaceMrn),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	return nil, errors.New("don't support recording layers for the Mondoo provider at the moment")
}
