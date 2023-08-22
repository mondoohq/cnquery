// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/providers/network/connection"
	"go.mondoo.com/cnquery/providers/network/resources"
	"go.mondoo.com/cnquery/providers/network/resources/domain"
)

const defaultConnection uint32 = 1

// This is a small selection of common ports that are supported.
// Outside of this range, users will have to specify ports explicitly.
// We could expand this to cover more of IANA.
var commonPorts = map[string]int{
	"https":  443,
	"http":   80,
	"ssh":    22,
	"ftp":    21,
	"telnet": 23,
	"smtp":   25,
	"dns":    53,
	"pop3":   110,
	"imap4":  143,
}

type Service struct {
	runtimes         map[uint32]*plugin.Runtime
	lastConnectionID uint32
}

func Init() *Service {
	return &Service{
		runtimes: map[uint32]*plugin.Runtime{},
	}
}

func (s *Service) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	target := req.Args[0]
	if i := strings.Index(target, "://"); i == -1 {
		target = "http://" + target
	}

	url, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	host, port := domain.SplitHostPort(url.Host)
	if port == 0 {
		port = commonPorts[url.Scheme]
	}

	insecure := false
	if found, ok := req.Flags["insecure"]; ok {
		insecure, _ = found.RawData().Value.(bool)
	}

	asset := inventory.Asset{
		Connections: []*inventory.Config{{
			Type:     "host",
			Port:     int32(port),
			Host:     host,
			Path:     url.Path,
			Insecure: insecure,
		}},
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

	// TODO: discovery of related assets and use them in the inventory below

	return &plugin.ConnectRes{
		Id:        uint32(conn.ID()),
		Name:      conn.Name(),
		Asset:     req.Asset,
		Inventory: nil,
	}, nil
}

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*connection.HostConnection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]
	var conn *connection.HostConnection
	var err error

	switch conf.Type {
	case "host":
		s.lastConnectionID++
		conn = connection.NewHostConnection(s.lastConnectionID, asset, conf)

	default:
		// generic host connection, without anything else
		s.lastConnectionID++
		conn = connection.NewHostConnection(s.lastConnectionID, asset, conf)
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

func (s *Service) detect(asset *inventory.Asset, conn *connection.HostConnection) error {
	if conn.Conf.Port == 0 {
		return errors.New("a port for the network connection is required")
	}

	asset.Id = conn.Conf.Type + "://" + conn.Conf.Host + ":" + strconv.Itoa(int(conn.Conf.Port)) + conn.Conf.Path
	asset.Name = conn.Conf.Host
	asset.Platform = &inventory.Platform{
		Name:   "host",
		Family: []string{"network"},
		Kind:   "network",
		Title:  "Network API",
	}

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
