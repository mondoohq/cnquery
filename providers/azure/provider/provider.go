// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"errors"
	"fmt"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"
	"go.mondoo.com/cnquery/v10/providers/azure/connection"
	"go.mondoo.com/cnquery/v10/providers/azure/connection/azureinstancesnapshot"
	"go.mondoo.com/cnquery/v10/providers/azure/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/azure/resources"
)

const (
	ConnectionType = "azure"
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
	flags := req.GetFlags()

	tenantId := flags["tenant-id"]
	clientId := flags["client-id"]
	clientSecret := flags["client-secret"]
	subscriptionId := flags["subscription"]
	subscriptions := flags["subscriptions"]
	subscriptionsToExclude := flags["subscriptions-exclude"]
	certificatePath := flags["certificate-path"]
	certificateSecret := flags["certificate-secret"]
	skipSnapshotCleanup := flags["skip-snapshot-cleanup"]
	skipSnapshotSetup := flags["skip-snapshot-setup"]
	lun := flags["lun"]
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
	if len(lun.Value) > 0 {
		opts[azureinstancesnapshot.Lun] = fmt.Sprint(lun.RawData().Value.(int64))
	}
	// the presence of the flag indicates that we should skip cleanup
	if skipCleanup := skipSnapshotCleanup.RawData().Value.(bool); skipCleanup {
		opts[azureinstancesnapshot.SkipCleanup] = "true"
	}
	// the presence of the flag indicates that we should skip setup. the disk we're trying to scan
	// is already attached. Rely on the lun parameter to give us a hint as to the location of the disk
	if skipSetup := skipSnapshotSetup.RawData().Value.(bool); skipSetup {
		opts[azureinstancesnapshot.SkipSetup] = "true"
		// we cannot detach the disk if we didn't attach it.
		// we cannot delete the disk as we do not know it's azure resource id
		// explicitly set the cleanup flag to false for clarity
		opts[azureinstancesnapshot.SkipCleanup] = "true"
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
		config.Options["type"] = azureinstancesnapshot.InstanceTargetType
		config.Options["target"] = args[2]
		return nil
	case "snapshot":
		config.Type = string(azureinstancesnapshot.SnapshotConnectionType)
		config.Options["type"] = azureinstancesnapshot.SnapshotTargetType
		config.Options["target"] = args[2]
		config.Discover = nil
		return nil
	case "disk":
		config.Type = string(azureinstancesnapshot.SnapshotConnectionType)
		config.Options["type"] = azureinstancesnapshot.DiskTargetType
		config.Options["target"] = args[2]
		config.Discover = nil
		return nil
	default:
		return errors.New("unknown subcommand " + args[1])
	}
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

	runtime, err := s.AddRuntime(func(connId uint32) (*plugin.Runtime, error) {
		var conn shared.AzureConnection
		var err error

		switch conf.Type {
		case string(azureinstancesnapshot.SnapshotConnectionType):
			// An AzureSnapshotConnection is a wrapper around a FilesystemConnection
			// To make sure the connection is later handled by the os provider, override the type
			conf.Type = "filesystem"
			conn, err = azureinstancesnapshot.NewAzureSnapshotConnection(connId, conf, asset)
		default:
			conn, err = connection.NewAzureConnection(connId, asset, conf)
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

	return runtime.Connection.(shared.AzureConnection), nil
}

func (s *Service) detect(asset *inventory.Asset, conn shared.AzureConnection) error {
	return nil
}

func (s *Service) discover(conn shared.AzureConnection) (*inventory.Inventory, error) {
	if conn.Config().Discover == nil {
		return nil, nil
	}

	runtime, err := s.GetRuntime(conn.ID())
	if err != nil {
		return nil, err
	}

	return resources.Discover(runtime, conn.Config())
}
