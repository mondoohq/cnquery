// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"encoding/json"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v11/utils/multierr"
	"go.mondoo.com/mondoo-go"
	"go.mondoo.com/mondoo-go/option"
)

type Connection struct {
	plugin.Connection
	Upstream *upstream.UpstreamClient
	Client   *mondoogql.Client
}

func New(id uint32, asset *inventory.Asset, conf *inventory.Config, upstream *upstream.UpstreamClient) (*Connection, error) {
	creds := upstream.GetCreds()
	rawCreds, err := json.Marshal(creds)
	if err != nil {
		return nil, multierr.Wrap(err, "failed to wrap credentials for Mondoo API client")
	}

	client, err := mondoogql.NewClient(
		option.UseUSRegion(),
		// option.WithAPIToken(os.Getenv("MONDOO_API_TOKEN")),
		option.WithHTTPClient(upstream.HttpClient),
		option.WithServiceAccount(rawCreds),
	)
	if err != nil {
		return nil, multierr.Wrap(err, "failed to initialize Mondoo API client")
	}

	return &Connection{
		Connection: plugin.NewConnection(id, asset),
		Client:     client,
		Upstream:   upstream,
	}, nil
}

func (c Connection) Name() string {
	return "mondoo"
}
