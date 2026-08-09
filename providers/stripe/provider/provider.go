// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package provider

import (
	"context"
	"errors"
	"os"
	"strings"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/inventory"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/upstream"
	"go.mondoo.com/mql/v13/providers-sdk/v1/vault"
	"go.mondoo.com/mql/v13/providers/stripe/connection"
	"go.mondoo.com/mql/v13/providers/stripe/resources"
)

const (
	DefaultConnectionType = "stripe"
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

	if account := stringFlag(flags, connection.OptionAccount); account != "" {
		conf.Options[connection.OptionAccount] = account
	}

	if baseURL := stringFlag(flags, connection.OptionBaseURL); baseURL != "" {
		conf.Options[connection.OptionBaseURL] = baseURL
	}

	token := stringFlag(flags, connection.OptionToken)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("STRIPE_API_KEY"))
	}
	if token == "" {
		token = strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY"))
	}
	if token != "" {
		conf.Credentials = append(conf.Credentials, vault.NewPasswordCredential("", token))
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

	// We only need to run the detection step when we don't have any asset
	// information yet.
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

func (s *Service) connect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*connection.StripeConnection, error) {
	if len(req.Asset.Connections) == 0 {
		return nil, errors.New("no connection options for asset")
	}

	asset := req.Asset
	conf := asset.Connections[0]
	runtime, err := s.AddRuntime(conf, func(connId uint32) (*plugin.Runtime, error) {
		var conn *connection.StripeConnection
		var err error

		switch conf.Type {
		default:
			conn, err = connection.NewStripeConnection(connId, asset, conf)
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

	return runtime.Connection.(*connection.StripeConnection), nil
}

// detect fetches the account so the asset carries a stable platform ID and a
// human-readable name.
func (s *Service) detect(asset *inventory.Asset, conn *connection.StripeConnection) error {
	account, err := conn.DetectAccount(context.Background())
	if err != nil {
		return err
	}

	asset.Platform = connection.NewStripeAccountPlatform(account.ID)
	asset.PlatformIds = []string{connection.NewStripeAccountIdentifier(account.ID)}

	name := account.DisplayName()
	if name == "" {
		name = "Stripe account " + account.ID
	}
	asset.Name = name
	return nil
}

func (s *Service) MockConnect(req *plugin.ConnectReq, callback plugin.ProviderCallback) (*plugin.ConnectRes, error) {
	return nil, errors.New("mock connect not yet implemented")
}

func stringFlag(flags map[string]*llx.Primitive, name string) string {
	if found, ok := flags[name]; ok && found != nil {
		return strings.TrimSpace(string(found.Value))
	}
	return ""
}
