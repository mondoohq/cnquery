// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"errors"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v10/providers/aws/connection"
	"go.mondoo.com/cnquery/v10/providers/aws/connection/awsec2ebsconn"
	"go.mondoo.com/cnquery/v10/providers/aws/resources"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
)

const (
	defaultConnection     uint32 = 1
	DefaultConnectionType        = "aws"
)

type Service struct {
	plugin.Service
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
	opts := parseFlagsToOptions(flags)

	// handle aws subcommands
	if len(req.Args) >= 3 && req.Args[0] == "ec2" {
		return &plugin.ParseCLIRes{Asset: handleAwsEc2Subcommands(req.Args, opts)}, nil
	}

	inventoryConfig := &inventory.Config{
		Type: req.Connector,
	}
	// discovery flags
	discoverTargets := []string{}
	if x, ok := flags["discover"]; ok && len(x.Array) != 0 {
		for i := range x.Array {
			entry := string(x.Array[i].Value)
			discoverTargets = append(discoverTargets, entry)
		}
	}
	filterOpts := parseFlagsToFiltersOpts(flags)

	inventoryConfig.Discover = &inventory.Discovery{Targets: discoverTargets, Filter: filterOpts}
	asset := inventory.Asset{
		Connections: []*inventory.Config{inventoryConfig},
		Options:     opts,
	}
	return &plugin.ParseCLIRes{Asset: &asset}, nil
}

func handleAwsEc2Subcommands(args []string, opts map[string]string) *inventory.Asset {
	asset := &inventory.Asset{}
	switch args[1] {
	case "instance-connect":
		return resources.InstanceConnectAsset(args, opts)
	case "ssm":
		return resources.SSMConnectAsset(args, opts)
	case "ebs":
		return resources.EbsConnectAsset(args, opts)
	}
	return asset
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
			if strings.Contains(k, ":region") {
				o[k] = string(v.Value)
			}
		}
	}

	return o
}

func parseFlagsToOptions(m map[string]*llx.Primitive) map[string]string {
	o := make(map[string]string, 0)
	for k, v := range m {
		if k == "profile" || k == "region" || k == "role" || k == "endpoint-url" || k == "no-setup" || k == "scope" {
			if val := string(v.Value); val != "" {
				o[k] = string(v.Value)
			}
		}
	}
	return o
}

// Shutdown is automatically called when the shell closes.
// It is not necessary to implement this method.
// If you want to do some cleanup, you can do it here.
func (s *Service) Shutdown(req *plugin.ShutdownReq) (*plugin.ShutdownRes, error) {
	for i := range s.runtimes {
		runtime := s.runtimes[i]
		if conn, ok := runtime.Connection.(shared.Connection); ok {
			if conn.Type() == awsec2ebsconn.EBSConnectionType {
				conn := runtime.Connection.(*awsec2ebsconn.AwsEbsConnection)
				conn.Close()
			}
		}
	}
	return &plugin.ShutdownRes{}, nil
}

func (s *Service) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	if req == nil || req.Asset == nil {
		return nil, errors.New("no connection data provided")
	}

	asset := &inventory.Asset{
		PlatformIds: req.Asset.PlatformIds,
		Platform:    req.Asset.Platform,
		Connections: []*inventory.Config{{
			Type: "mock",
		}},
	}

	conn, err := s.connect(&plugin.ConnectReq{
		Features: req.Features,
		Upstream: req.Upstream,
		Asset:    asset,
	}, callback)
	if err != nil {
		return nil, err
	}

	return &plugin.ConnectRes{
		Id:        uint32(conn.(shared.Connection).ID()),
		Name:      conn.(shared.Connection).Name(),
		Asset:     asset,
		Inventory: nil,
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
	inventory := &inventory.Inventory{
		Spec: &inventory.InventorySpec{
			Assets: []*inventory.Asset{req.Asset},
		},
	}

	if c, ok := conn.(*connection.AwsConnection); ok {
		if req.Asset.Platform != nil {
			c.PlatformOverride = req.Asset.Platform.Name
		}
		inventory, err = s.discover(c)
		if err != nil {
			return nil, err
		}
	}

	return &plugin.ConnectRes{
		Id:        uint32(conn.(shared.Connection).ID()),
		Name:      conn.(shared.Connection).Name(),
		Asset:     req.Asset,
		Inventory: inventory,
	}, nil
}

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (plugin.Connection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}
	asset := req.Asset
	conf := asset.Connections[0]
	var conn shared.Connection
	var err error

	switch conf.Type {
	case "mock":
		s.lastConnectionID++
		conn = connection.NewMockConnection(s.lastConnectionID, asset, conf)

	case string(awsec2ebsconn.EBSConnectionType):
		s.lastConnectionID++
		conn, err = awsec2ebsconn.NewAwsEbsConnection(s.lastConnectionID, conf, asset)
		if conn.Asset() != nil && len(conn.Asset().Connections) > 0 && conn.Asset().Connections[0].Options["mounted"] != "" {
			// if we've already done all the mounting work, then reassign the connection
			// to be the filesystem connection so we use the right connection down the line
			fsconn := conn.(*awsec2ebsconn.AwsEbsConnection).FsProvider
			conn = fsconn
			req.Asset = fsconn.Asset()
			req.Asset.Connections[0] = fsconn.Conf
			asset = req.Asset
		}
	default:
		s.lastConnectionID++
		conn, err = connection.NewAwsConnection(s.lastConnectionID, asset, conf)
	}
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
	s.runtimes[conn.ID()] = &plugin.Runtime{
		Connection:     conn,
		Callback:       callback,
		HasRecording:   req.HasRecording,
		CreateResource: resources.CreateResource,
		Upstream:       upstream,
	}

	return conn, err
}

func (s *Service) detect(asset *inventory.Asset, conn plugin.Connection) error {
	if len(asset.Connections) > 0 && asset.Connections[0].Type == "ssh" {
		// workaround to make sure we dont assign the aws platform to ec2 instances
		return nil
	}
	if c, ok := conn.(*connection.AwsConnection); ok {
		asset.Name = c.Conf.Host
		asset.Platform = c.PlatformInfo()
		// TODO: do not do this here but in discovery
		asset.PlatformIds = []string{"//platformid.api.mondoo.app/runtime/aws/accounts/" + c.AccountId()}
	}
	if c, ok := conn.(*awsec2ebsconn.AwsEbsConnection); ok {
		asset.Platform = c.PlatformInfo()
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

	resource, ok := runtime.Resources.Get(req.Resource + "\x00" + req.ResourceId)
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

func (s *Service) discover(conn *connection.AwsConnection) (*inventory.Inventory, error) {
	if conn.Conf.Discover == nil {
		return nil, nil
	}

	runtime, ok := s.runtimes[conn.ID()]
	if !ok {
		// no connection found, this should never happen
		return nil, errors.New("connection " + strconv.FormatUint(uint64(conn.ID()), 10) + " not found")
	}

	return resources.Discover(runtime, conn.Filters)
}
