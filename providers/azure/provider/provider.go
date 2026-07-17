// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"context"
	"errors"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/providers/azure/connection/azureinstancesnapshot"
	"go.mondoo.com/mql/v13/providers/azure/connection/shared"
	"go.mondoo.com/mql/v13/providers/azure/resources"
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

// flagBytes safely reads a flag's raw value. Unset flags (including keys the
// CLI never registers, such as the legacy singular "subscription") are absent
// from the map and therefore nil pointers, so a direct .Value dereference
// panics. Returning an empty slice for those keeps ParseCLI robust.
func flagBytes(flags map[string]*llx.Primitive, key string) []byte {
	if p, ok := flags[key]; ok && p != nil {
		return p.Value
	}
	return nil
}

func (s *Service) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
	flags := req.GetFlags()

	tenantId := flagBytes(flags, "tenant-id")
	clientId := flagBytes(flags, "client-id")
	clientSecret := flagBytes(flags, "client-secret")
	certificatePath := flagBytes(flags, "certificate-path")
	certificateSecret := flagBytes(flags, "certificate-secret")
	federatedTokenFile := flagBytes(flags, "federated-token-file")
	opts := map[string]string{}
	creds := []*vault.Credential{}

	opts["tenant-id"] = string(tenantId)
	opts["client-id"] = string(clientId)
	if len(federatedTokenFile) > 0 {
		opts[connection.OptionFederatedTokenFile] = string(federatedTokenFile)
	}
	if len(clientSecret) > 0 {
		creds = append(creds, &vault.Credential{
			Type:   vault.CredentialType_password,
			Secret: clientSecret,
		})
	} else if len(certificatePath) > 0 {
		creds = append(creds, &vault.Credential{
			Type:           vault.CredentialType_pkcs12,
			PrivateKeyPath: string(certificatePath),
			Password:       string(certificateSecret),
		})
	}
	config := &inventory.Config{
		Type:        "azure",
		Discover:    parseDiscover(flags, parseFlagsToFiltersOpts(flags)),
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

func parseDiscover(flags map[string]*llx.Primitive, filterOpts map[string]string) *inventory.Discovery {
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
	return &inventory.Discovery{Targets: targets, Filter: filterOpts}
}

// parseFlagsToFiltersOpts builds the discovery filter options map from both the
// --filters key/value flag and the dedicated --subscription* flags, then stores
// it on inventory.Discovery.Filter (mirroring the AWS provider). The dedicated
// flags take precedence over their --filters counterparts, and the plural
// --subscriptions overrides the singular --subscription (preserving the
// historical precedence). Keys are matched exactly, not by prefix, because
// "subscriptions" is a prefix of "subscriptions-exclude".
func parseFlagsToFiltersOpts(flags map[string]*llx.Primitive) map[string]string {
	o := map[string]string{}

	// base: the --filters key/value flag (allowlisted keys only)
	if x, ok := flags["filters"]; ok && len(x.Map) != 0 {
		for k, v := range x.Map {
			if k == "subscriptions" || k == "subscriptions-exclude" {
				o[k] = string(v.Value)
			}
		}
	}

	// overlay: dedicated flags win over their --filters counterparts
	if v := flagBytes(flags, "subscription"); len(v) > 0 {
		o["subscriptions"] = string(v)
	}
	if v := flagBytes(flags, "subscriptions"); len(v) > 0 {
		o["subscriptions"] = string(v)
	}
	if v := flagBytes(flags, "subscriptions-exclude"); len(v) > 0 {
		o["subscriptions-exclude"] = string(v)
	}

	return o
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

	runtime, err := s.AddRuntime(conf, func(connId uint32) (*plugin.Runtime, error) {
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
			upstream, err = req.Upstream.InitClient(context.Background())
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

	if len(conn.Config().Discover.Targets) == 0 {
		return nil, nil
	}

	runtime, err := s.GetRuntime(conn.ID())
	if err != nil {
		return nil, err
	}

	return resources.Discover(runtime, conn.Config())
}
