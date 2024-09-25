// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"context"
	"errors"
	"strings"

	"go.mondoo.com/cnquery/v11"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
	"go.mondoo.com/cnquery/v11/providers/aws/connection/awsec2ebsconn"
	"go.mondoo.com/cnquery/v11/providers/aws/resources"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
)

const (
	DefaultConnectionType = "aws"
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
		knownTagPrefixes := []string{
			"ec2:tag:",
			"exclude:ec2:tag:",
			"ec2:regions",
			"exclude:ec2:regions",
			"all:regions",
			"regions",
			"ec2:instance-ids",
			"exclude:ec2:instance-ids",
			"all:tag:",
			"ecr:tags",
			"ecs:only-running-containers",
			"ecs:discover-instances",
			"ecs:discover-images",
		}
		for k, v := range x.Map {
			for _, prefix := range knownTagPrefixes {
				if strings.HasPrefix(k, prefix) {
					o[k] = string(v.Value)
					break
				}
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

func (s *Service) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	if req == nil || req.Asset == nil {
		return nil, errors.New("no connection data provided")
	}

	// If we get 1 connection that enables fine-grained assets, enable it globally for the provider
	if cnquery.Features(req.Features).IsActive(cnquery.FineGrainedAssets) {
		resources.ENABLE_FINE_GRAINED_ASSETS = true
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
		Id:        uint32(conn.ID()),
		Name:      conn.Name(),
		Asset:     asset,
		Inventory: nil,
	}, nil
}

func (s *Service) Connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	if req == nil || req.Asset == nil {
		return nil, errors.New("no connection data provided")
	}

	// If we get 1 connection that enables fine-grained assets, enable it globally for the provider
	if cnquery.Features(req.Features).IsActive(cnquery.FineGrainedAssets) {
		resources.ENABLE_FINE_GRAINED_ASSETS = true
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

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (shared.Connection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}
	asset := req.Asset
	conf := asset.Connections[0]

	runtime, err := s.AddRuntime(conf, func(connId uint32) (*plugin.Runtime, error) {
		var conn shared.Connection
		var err error

		switch conf.Type {
		case "mock":
			conn = connection.NewMockConnection(connId, asset, conf)

		case string(awsec2ebsconn.EBSConnectionType):
			// An EBS connection is a wrapper around a FilesystemConnection
			// To make sure the connection is later handled by the os provider, override the type
			conf.Type = "filesystem"
			conn, err = awsec2ebsconn.NewAwsEbsConnection(connId, conf, asset)
		default:
			conn, err = connection.NewAwsConnection(connId, asset, conf)
		}
		if err != nil {
			return nil, err
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

func (s *Service) detect(asset *inventory.Asset, conn plugin.Connection) error {
	if len(asset.Connections) > 0 && asset.Connections[0].Type == "ssh" {
		// workaround to make sure we don't assign the aws platform to ec2 instances
		return nil
	}
	if c, ok := conn.(*connection.AwsConnection); ok {
		asset.Name = c.Conf.Host
		asset.Platform = c.PlatformInfo()
	}
	if c, ok := conn.(*awsec2ebsconn.AwsEbsConnection); ok {
		asset.Platform = c.PlatformInfo()
	}
	return nil
}

func (s *Service) discover(conn *connection.AwsConnection) (*inventory.Inventory, error) {
	if conn.Conf.Discover == nil {
		return nil, nil
	}

	runtime, err := s.GetRuntime(conn.ID())
	if err != nil {
		return nil, err
	}

	return resources.Discover(runtime)
}
