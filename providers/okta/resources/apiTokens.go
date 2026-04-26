// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/okta/connection"
	"go.mondoo.com/mql/v13/providers/okta/resources/sdk"
)

// apiTokens lists API tokens for the org. Requires Super Admin privileges
// (the /api/v1/api-tokens endpoint).
func (o *mqlOkta) apiTokens() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)

	ctx := context.Background()
	tokens, err := sdk.ListApiTokens(ctx, conn.OrganizationID(), conn.Token())
	if err != nil {
		return nil, err
	}

	list := []any{}
	for _, t := range tokens {
		r, err := CreateResource(o.MqlRuntime, "okta.api.token", map[string]*llx.RawData{
			"id":          llx.StringData(t.Id),
			"name":        llx.StringData(t.Name),
			"userId":      llx.StringData(t.UserId),
			"clientName":  llx.StringData(t.ClientName),
			"created":     llx.TimeDataPtr(t.Created),
			"expiresAt":   llx.TimeDataPtr(t.ExpiresAt),
			"lastUpdated": llx.TimeDataPtr(t.LastUpdated),
			"tokenWindow": llx.StringData(t.TokenWindow),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, nil
}

func (o *mqlOktaApiToken) id() (string, error) {
	return "okta.api.token/" + o.Id.Data, o.Id.Error
}

// user resolves the typed user this API token was issued for.
func (o *mqlOktaApiToken) user() (*mqlOktaUser, error) {
	if o.UserId.Error != nil {
		return nil, o.UserId.Error
	}
	if o.UserId.Data == "" {
		o.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	usr, _, err := client.User.GetUser(ctx, o.UserId.Data)
	if err != nil {
		return nil, err
	}
	return newMqlOktaUser(o.MqlRuntime, usr)
}
