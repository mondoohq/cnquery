// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/providers-sdk/v1/vault"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/providers/ipmi/connection"
	"go.mondoo.com/cnquery/providers/ipmi/resources"
)

type Service struct {
	runtimes         map[uint32]*plugin.Runtime
	lastConnectionID uint32
}

func Init() *Service {
	return &Service{
		runtimes:         map[uint32]*plugin.Runtime{},
		lastConnectionID: 0,
	}
}

func (s *Service) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	flags := req.Flags
	if flags == nil {
		flags = map[string]*llx.Primitive{}
	}

	conn := &inventory.Config{
		Type: req.Connector,
	}

	// custom flag parsing

	user := ""
	port := 623
	if len(req.Args) != 0 {
		target := req.Args[0]
		if !strings.Contains(target, "://") {
			target = "ssh://" + target
		}

		x, err := url.Parse(target)
		if err != nil {
			return nil, errors.New("incorrect format of target, please use user@host:port")
		}

		user = x.User.Username()
		conn.Host = x.Hostname()
		conn.Path = x.Path

		if sPort := x.Port(); sPort != "" {
			port, err = strconv.Atoi(x.Port())
			if err != nil {
				return nil, errors.New("port '" + x.Port() + "'is incorrectly formatted, must be a number")
			}
		}
	}

	if port > 0 {
		conn.Port = int32(port)
	}

	if x, ok := flags["password"]; ok && len(x.Value) != 0 {
		conn.Credentials = append(conn.Credentials, vault.NewPasswordCredential(user, string(x.Value)))
	}

	asset := inventory.Asset{
		Connections: []*inventory.Config{conn},
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

	return &plugin.ConnectRes{
		Id:        conn.ID(),
		Name:      conn.Name(),
		Asset:     req.Asset,
		Inventory: nil,
	}, nil
}

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*connection.IpmiConnection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]
	var conn *connection.IpmiConnection
	var err error

	switch conf.Type {
	default:
		s.lastConnectionID++
		conn, err = connection.NewIpmiConnection(s.lastConnectionID, asset, conf)
	}

	if err != nil {
		return nil, err
	}

	var upstream *upstream.UpstreamClient
	if req.Upstream != nil {
		upstream, err = req.Upstream.InitClient()
		if err != nil {
			return nil, err
		}
	}

	asset.Connections[0].Id = conn.ID()
	s.runtimes[conn.ID()] = &plugin.Runtime{
		Connection:     conn,
		Resources:      map[string]plugin.Resource{},
		Callback:       callback,
		HasRecording:   req.HasRecording,
		CreateResource: resources.CreateResource,
		Upstream:       upstream,
	}

	return conn, err
}

func (s *Service) detect(asset *inventory.Asset, conn *connection.IpmiConnection) error {
	asset.Id = conn.Conf.Type
	asset.Name = conn.Conf.Host

	asset.Platform = &inventory.Platform{
		Name:  "ipmi",
		Title: "IPMI",
		Kind:  "api",
	}

	// TODO: consider using the ipmi vendor id and product id
	if asset.Name == "" {
		asset.Name = "IPMI device " + conn.Guid()
	}

	asset.PlatformIds = []string{conn.Identifier()}
	return nil
}

func (s *Service) GetData(req *plugin.DataReq) (*plugin.DataRes, error) {
	runtime, ok := s.runtimes[req.Connection]
	if !ok {
		return nil, errors.New("connection " + strconv.FormatUint(uint64(req.Connection), 10) + " not found")
	}

	args := plugin.PrimitiveArgsToRawDataArgs(req.Args, runtime)

	if req.ResourceId == "" && req.Field == "" {
		res, err := resources.NewResource(runtime, req.Resource, args)
		if err != nil {
			return nil, err
		}

		rd := llx.ResourceData(res, res.MqlName()).Result()
		return &plugin.DataRes{
			Data: rd.Data,
		}, nil
	}

	resource, ok := runtime.Resources[req.Resource+"\x00"+req.ResourceId]
	if !ok {
		// Note: Since resources are internally always created, there are only very
		// few cases where we arrive here:
		// 1. The caller is wrong. Possibly a mixup with IDs
		// 2. The resource was loaded from a recording, but the field is not
		// in the recording. Thus the resource was never created inside the
		// plugin. We will attempt to create the resource and see if the field
		// can be computed.
		if !runtime.HasRecording {
			return nil, errors.New("resource '" + req.Resource + "' (id: " + req.ResourceId + ") doesn't exist")
		}

		args, err := runtime.ResourceFromRecording(req.Resource, req.ResourceId)
		if err != nil {
			return nil, errors.New("attempted to load resource '" + req.Resource + "' (id: " + req.ResourceId + ") from recording failed: " + err.Error())
		}

		resource, err = resources.CreateResource(runtime, req.Resource, args)
		if err != nil {
			return nil, errors.New("attempted to create resource '" + req.Resource + "' (id: " + req.ResourceId + ") from recording failed: " + err.Error())
		}
	}

	return resources.GetData(resource, req.Field, args), nil
}

func (s *Service) StoreData(req *plugin.StoreReq) (*plugin.StoreRes, error) {
	return nil, errors.New("not yet implemented")
}
