// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"context"
	"errors"
	"fmt"

	"go.mondoo.com/cnquery/v10/providers-sdk/v1/vault"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v10/providers/oci/connection"
	"go.mondoo.com/cnquery/v10/providers/oci/resources"
)

const ConnectionType = "oci"

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
		Options: make(map[string]string),
	}

	// Do custom flag parsing here
	tenancy := ""
	if x, ok := flags["tenancy"]; ok && len(x.Value) != 0 {
		tenancy = string(x.Value)
	}

	user := ""
	if x, ok := flags["user"]; ok && len(x.Value) != 0 {
		user = string(x.Value)
	}

	region := ""
	if x, ok := flags["user"]; ok && len(x.Value) != 0 {
		region = string(x.Value)
	}

	keyPath := ""
	if x, ok := flags["key-path"]; ok && len(x.Value) != 0 {
		keyPath = string(x.Value)
	}

	fingerprint := ""
	if x, ok := flags["fingerprint"]; ok && len(x.Value) != 0 {
		fingerprint = string(x.Value)
	}

	keySecret := ""
	if x, ok := flags["key-secret"]; ok && len(x.Value) != 0 {
		keySecret = string(x.Value)
	}

	if tenancy != "" {
		conf.Options["tenancy"] = tenancy
	}
	if fingerprint != "" {
		conf.Options["fingerprint"] = fingerprint
	}
	if region != "" {
		conf.Options["region"] = region
	}
	if user != "" {
		conf.Options["user"] = user
	}

	if keyPath != "" {
		conf.Credentials = append(conf.Credentials, &vault.Credential{
			Type:           vault.CredentialType_private_key,
			PrivateKeyPath: keyPath,
			Password:       keySecret,
		})
	}

	asset := inventory.Asset{
		Connections: []*inventory.Config{conf},
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

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*connection.OciConnection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]

	runtime, err := s.AddRuntime(conf, func(connId uint32) (*plugin.Runtime, error) {
		conn, err := connection.NewOciConnection(connId, asset, conf)
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

	return runtime.Connection.(*connection.OciConnection), nil
}

func (s *Service) detect(asset *inventory.Asset, conn *connection.OciConnection) error {
	asset.Name = conn.Conf.Host

	info, err := conn.Tenant(context.Background())
	if err != nil {
		return err
	}
	if info != nil {
		asset.Name = fmt.Sprintf("OCI Tenant %s", *info.Name)
	}

	asset.Platform = &inventory.Platform{
		Name:    "oci",
		Title:   "Oracle Cloud Infrastructure",
		Runtime: "oci",
		Kind:    "api",
		Family:  []string{"oci"},
	}

	platformID := "//platformid.api.mondoo.app/runtime/oci/" + conn.TenantID()
	asset.PlatformIds = []string{platformID}
	return nil
}
