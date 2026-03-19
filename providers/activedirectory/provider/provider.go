// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"context"
	"errors"
	"strconv"

	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"go.mondoo.com/mql/v13/providers/activedirectory/connection"
	"go.mondoo.com/mql/v13/providers/activedirectory/resources"
)

const ConnectionType = "activedirectory"

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

	dc := flags["dc"]
	user := flags["user"]
	password := flags["password"]
	domain := flags["domain"]
	baseDN := flags["base-dn"]
	ldaps := flags["ldaps"]
	port := flags["port"]
	insecure := flags["insecure"]
	backend := flags["backend"]

	if len(dc.Value) == 0 {
		return nil, errors.New("dc flag is required: specify the domain controller hostname or IP address")
	}

	opts := map[string]string{}
	opts[connection.OptionDC] = string(dc.Value)

	if len(domain.Value) > 0 {
		opts[connection.OptionDomain] = string(domain.Value)
	}
	if len(baseDN.Value) > 0 {
		opts[connection.OptionBaseDN] = string(baseDN.Value)
	}
	if len(ldaps.Value) > 0 {
		opts[connection.OptionLDAPS] = string(ldaps.Value)
	}
	if len(port.Value) > 0 {
		// Validate port is a valid integer before storing it.
		if _, err := strconv.Atoi(string(port.Value)); err != nil {
			return nil, errors.New("port flag must be a valid integer: " + err.Error())
		}
		opts[connection.OptionPort] = string(port.Value)
	}
	if len(insecure.Value) > 0 {
		opts[connection.OptionInsecure] = string(insecure.Value)
	}
	if len(backend.Value) > 0 {
		b := string(backend.Value)
		if b != "ldap" && b != "rsat" {
			return nil, errors.New("backend flag must be 'ldap' or 'rsat'")
		}
		opts[connection.OptionBackend] = b
	}

	creds := []*vault.Credential{}
	if len(password.Value) > 0 {
		creds = append(creds, &vault.Credential{
			Type:   vault.CredentialType_password,
			User:   string(user.Value),
			Secret: password.Value,
		})
	}

	config := &inventory.Config{
		Type:        ConnectionType,
		Credentials: creds,
		Options:     opts,
	}
	asset := inventory.Asset{
		Connections: []*inventory.Config{config},
	}

	return &plugin.ParseCLIRes{Asset: &asset}, nil
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

	if req.Asset.Platform == nil {
		if err := s.detect(req.Asset, conn); err != nil {
			return nil, err
		}
	}

	return &plugin.ConnectRes{
		Id:    uint32(conn.ID()),
		Name:  conn.Name(),
		Asset: req.Asset,
	}, nil
}

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*connection.ActiveDirectoryConnection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]
	runtime, err := s.AddRuntime(conf, func(connId uint32) (*plugin.Runtime, error) {
		conn, err := connection.NewActiveDirectoryConnection(connId, asset, conf)
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

	return runtime.Connection.(*connection.ActiveDirectoryConnection), nil
}

func (s *Service) detect(asset *inventory.Asset, conn *connection.ActiveDirectoryConnection) error {
	asset.Name = "Active Directory " + conn.FQDN()
	asset.Platform = &inventory.Platform{
		Name:                  "activedirectory",
		Runtime:               "activedirectory",
		Family:                []string{"directory-service"},
		Kind:                  "api",
		Title:                 "Active Directory Domain Services",
		TechnologyUrlSegments: []string{"directory-service", "activedirectory"},
	}
	asset.PlatformIds = []string{conn.PlatformId()}

	return nil
}
