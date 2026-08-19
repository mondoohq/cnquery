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
	"go.mondoo.com/mql/v13/providers/minio/connection"
	"go.mondoo.com/mql/v13/providers/minio/resources"
)

const (
	DefaultConnectionType = "minio"
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

	conf := &inventory.Config{
		Type:    req.Connector,
		Options: map[string]string{},
	}

	for _, name := range []string{
		connection.OptionEndpoint,
		connection.OptionRegion,
		connection.OptionCACert,
		connection.OptionTLSSkipVerify,
	} {
		if flag, ok := flags[name]; ok && len(flag.Value) > 0 {
			conf.Options[name] = string(flag.Value)
		}
	}

	// The secret key is a secret, so the key pair travels as a credential
	// rather than as options. The access key rides along in the user field,
	// which is where the credential store expects the public half.
	accessKey := ""
	if flag, ok := flags[connection.OptionAccessKey]; ok && len(flag.Value) > 0 {
		accessKey = string(flag.Value)
	}
	if flag, ok := flags["secret-key"]; ok && len(flag.Value) > 0 {
		conf.Credentials = append(conf.Credentials, &vault.Credential{
			Type:   vault.CredentialType_password,
			User:   accessKey,
			Secret: flag.Value,
		})
	} else if accessKey != "" {
		conf.Options[connection.OptionAccessKey] = accessKey
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

	return &plugin.ConnectRes{
		Id:        conn.ID(),
		Name:      conn.Name(),
		Asset:     req.Asset,
		Inventory: nil,
	}, nil
}

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*connection.MinioConnection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]
	runtime, err := s.AddRuntime(conf, func(connId uint32) (*plugin.Runtime, error) {
		var conn *connection.MinioConnection
		var err error

		switch conf.Type {
		default:
			conn, err = connection.NewMinioConnection(connId, asset, conf)
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

	return runtime.Connection.(*connection.MinioConnection), nil
}

func (s *Service) detect(asset *inventory.Asset, conn *connection.MinioConnection) error {
	asset.Name = "MinIO " + conn.Host()
	asset.Platform = connection.NewMinioPlatform(conn.Host())

	// The version and deployment ID come from the admin API. A failure here is
	// not fatal: the asset is still identified by its address, and the
	// resources report the error when they are queried.
	deploymentID := ""
	if info, err := conn.Admin().ServerInfo(context.Background()); err == nil {
		deploymentID = info.DeploymentID
		if len(info.Servers) > 0 {
			asset.Platform.Version = info.Servers[0].Version
		}
	}

	id := connection.NewMinioIdentifier(deploymentID, conn.Host())
	asset.PlatformIds = []string{id}
	asset.Id = id
	return nil
}

func (s *Service) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	return nil, errors.New("mock connect not yet implemented")
}
