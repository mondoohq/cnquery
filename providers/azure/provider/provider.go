// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"errors"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v9/providers/azure/connection"
	"go.mondoo.com/cnquery/v9/providers/azure/connection/azureinstancesnapshot"
	"go.mondoo.com/cnquery/v9/providers/azure/connection/shared"
	"go.mondoo.com/cnquery/v9/providers/azure/resources"
)

const (
	defaultConnection uint32 = 1
	ConnectionType           = "azure"
)

type Service struct {
	plugin.Service
	runtimes         map[uint32]*plugin.Runtime
	lastConnectionID uint32
}

func Init() *Service {
	return &Service{
		runtimes: map[uint32]*plugin.Runtime{},
	}
}

func (s *Service) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	flags := req.GetFlags()

	tenantId := flags["tenant-id"]
	clientId := flags["client-id"]
	clientSecret := flags["client-secret"]
	subscriptionId := flags["subscription"]
	subscriptions := flags["subscriptions"]
	subscriptionsToExclude := flags["subscriptions-exclude"]
	certificatePath := flags["certificate-path"]
	certificateSecret := flags["certificate-secret"]

	opts := map[string]string{}
	creds := []*vault.Credential{}

	opts["tenant-id"] = string(tenantId.Value)
	opts["client-id"] = string(clientId.Value)
	if len(subscriptionId.Value) > 0 {
		opts["subscriptions"] = string(subscriptionId.Value)
	}
	if len(subscriptions.Value) > 0 {
		opts["subscriptions"] = string(subscriptions.Value)
	}
	if len(subscriptionsToExclude.Value) > 0 {
		opts["subscriptions-exclude"] = string(subscriptionsToExclude.Value)
	}

	if len(clientSecret.Value) > 0 {
		creds = append(creds, &vault.Credential{
			Type:   vault.CredentialType_password,
			Secret: clientSecret.Value,
		})
	} else if len(certificatePath.Value) > 0 {
		creds = append(creds, &vault.Credential{
			Type:           vault.CredentialType_pkcs12,
			PrivateKeyPath: string(certificatePath.Value),
			Password:       string(certificateSecret.Value),
		})
	}
	config := &inventory.Config{
		Type:        "azure",
		Discover:    parseDiscover(flags),
		Credentials: creds,
		Options:     opts,
	}

	// handle azure subcommands
	if len(req.Args) >= 3 && req.Args[0] == "compute" {
		err := handleAzureComputeSubcommands(req.Args, config)
		if err != nil {
			return nil, err
		}
	}

	asset := inventory.Asset{
		Connections: []*inventory.Config{config},
	}

	return &plugin.ParseCLIRes{Asset: &asset}, nil
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
		targets = []string{resources.DiscoveryAuto}
	}
	return &inventory.Discovery{Targets: targets}
}

func handleAzureComputeSubcommands(args []string, config *inventory.Config) error {
	switch args[1] {
	case "instance":
		config.Type = string(azureinstancesnapshot.SnapshotConnectionType)
		config.Discover = nil
		config.Options["type"] = "instance"
		config.Options["target"] = args[2]
		return nil
	case "snapshot":
		config.Type = string(azureinstancesnapshot.SnapshotConnectionType)
		config.Options["type"] = "snapshot"
		config.Options["target"] = args[2]
		config.Discover = nil
		return nil
	default:
		return errors.New("unknown subcommand " + args[1])
	}
}

// Shutdown is automatically called when the shell closes.
// It is not necessary to implement this method.
// If you want to do some cleanup, you can do it here.
func (s *Service) Shutdown(req *plugin.ShutdownReq) (*plugin.ShutdownRes, error) {
	for i := range s.runtimes {
		runtime := s.runtimes[i]
		sharedConn := runtime.Connection.(shared.AzureConnection)
		if sharedConn.Type() == azureinstancesnapshot.SnapshotConnectionType {
			conn := runtime.Connection.(*azureinstancesnapshot.AzureSnapshotConnection)
			conn.Close()
		}
	}
	return &plugin.ShutdownRes{}, nil
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

	// discovery assets for further scanning
	inventory, err := s.discover(conn)
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

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (shared.AzureConnection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]
	s.lastConnectionID++
	var conn shared.AzureConnection
	var err error

	switch conf.Type {
	case string(azureinstancesnapshot.SnapshotConnectionType):
		// An AzureSnapshotConnection is a wrapper around a FilesystemConnection
		// To make sure the connection is later handled by the os provider, override the type
		conf.Type = "filesystem"
		s.lastConnectionID++
		conn, err = azureinstancesnapshot.NewAzureSnapshotConnection(s.lastConnectionID, conf, asset)
	default:
		s.lastConnectionID++
		conn, err = connection.NewAzureConnection(s.lastConnectionID, asset, conf)
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

func (s *Service) detect(asset *inventory.Asset, conn shared.AzureConnection) error {
	// TODO: what do i put here
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
	runtime, ok := s.runtimes[req.Connection]
	if !ok {
		return nil, errors.New("connection " + strconv.FormatUint(uint64(req.Connection), 10) + " not found")
	}

	var errs []string
	for i := range req.Resources {
		info := req.Resources[i]

		args, err := plugin.ProtoArgsToRawDataArgs(info.Fields)
		if err != nil {
			errs = append(errs, "failed to add cached "+info.Name+" (id: "+info.Id+"), failed to parse arguments")
			continue
		}

		resource, ok := runtime.Resources.Get(info.Name + "\x00" + info.Id)
		if !ok {
			resource, err = resources.CreateResource(runtime, info.Name, args)
			if err != nil {
				errs = append(errs, "failed to add cached "+info.Name+" (id: "+info.Id+"), creation failed: "+err.Error())
				continue
			}

			runtime.Resources.Set(info.Name+"\x00"+info.Id, resource)
		}

		for k, v := range args {
			if err := resources.SetData(resource, k, v); err != nil {
				errs = append(errs, "failed to add cached "+info.Name+" (id: "+info.Id+"), field error: "+err.Error())
			}
		}
	}

	if len(errs) != 0 {
		return nil, errors.New(strings.Join(errs, ", "))
	}
	return &plugin.StoreRes{}, nil
}

func (s *Service) discover(conn shared.AzureConnection) (*inventory.Inventory, error) {
	if conn.Config().Discover == nil {
		return nil, nil
	}

	runtime, ok := s.runtimes[conn.ID()]
	if !ok {
		// no connection found, this should never happen
		return nil, errors.New("connection " + strconv.FormatUint(uint64(conn.ID()), 10) + " not found")
	}

	return resources.Discover(runtime, conn.Config())
}
