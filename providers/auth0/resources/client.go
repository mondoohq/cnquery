// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/auth0/go-auth0/management"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/auth0/connection"
	"go.mondoo.com/mql/v13/types"
)

// clients lists every application (OAuth/OIDC client) registered in the tenant.
func (a *mqlAuth0) clients() ([]any, error) {
	conn := a.conn()
	client := conn.Client()

	var all []any
	page := 0
	for {
		list, err := client.Client.List(context.Background(),
			management.Page(page),
			management.PerPage(50),
			management.IncludeTotals(true),
		)
		if err != nil {
			return nil, err
		}
		for _, c := range list.Clients {
			r, err := newMqlAuth0Client(a.MqlRuntime, c)
			if err != nil {
				return nil, err
			}
			all = append(all, r)
		}
		if !list.HasNext() {
			break
		}
		page++
	}
	return all, nil
}

// newMqlAuth0Client maps a single SDK client to its MQL resource.
func newMqlAuth0Client(runtime *plugin.Runtime, c *management.Client) (plugin.Resource, error) {
	jwt := map[string]any{}
	if c.JWTConfiguration != nil {
		if c.JWTConfiguration.Algorithm != nil {
			jwt["alg"] = *c.JWTConfiguration.Algorithm
		}
		if c.JWTConfiguration.LifetimeInSeconds != nil {
			jwt["lifetimeInSeconds"] = *c.JWTConfiguration.LifetimeInSeconds
		}
	}

	meta := map[string]any{}
	if c.ClientMetadata != nil {
		for k, v := range *c.ClientMetadata {
			meta[k] = fmt.Sprintf("%v", v)
		}
	}

	var rotationType, expirationType *string
	var leeway, tokenLifetime, idleTokenLifetime *int
	if c.RefreshToken != nil {
		rotationType = c.RefreshToken.RotationType
		expirationType = c.RefreshToken.ExpirationType
		leeway = c.RefreshToken.Leeway
		tokenLifetime = c.RefreshToken.TokenLifetime
		idleTokenLifetime = c.RefreshToken.IdleTokenLifetime
	}

	r, err := CreateResource(runtime, "auth0.client", map[string]*llx.RawData{
		"id":                         llx.StringDataPtr(c.ClientID),
		"name":                       llx.StringDataPtr(c.Name),
		"description":                llx.StringDataPtr(c.Description),
		"appType":                    llx.StringDataPtr(c.AppType),
		"isFirstParty":               llx.BoolDataPtr(c.IsFirstParty),
		"oidcConformant":             llx.BoolDataPtr(c.OIDCConformant),
		"callbacks":                  llx.ArrayData(strList(c.Callbacks), types.String),
		"allowedLogoutUrls":          llx.ArrayData(strList(c.AllowedLogoutURLs), types.String),
		"allowedOrigins":             llx.ArrayData(strList(c.AllowedOrigins), types.String),
		"webOrigins":                 llx.ArrayData(strList(c.WebOrigins), types.String),
		"grantTypes":                 llx.ArrayData(strList(c.GrantTypes), types.String),
		"tokenEndpointAuthMethod":    llx.StringDataPtr(c.TokenEndpointAuthMethod),
		"refreshTokenRotationType":   llx.StringDataPtr(rotationType),
		"refreshTokenExpirationType": llx.StringDataPtr(expirationType),
		"refreshTokenLeeway":         llx.IntDataPtr(leeway),
		"refreshTokenLifetime":       llx.IntDataPtr(tokenLifetime),
		"refreshTokenIdleLifetime":   llx.IntDataPtr(idleTokenLifetime),
		"ssoDisabled":                llx.BoolDataPtr(c.SSODisabled),
		"crossOriginAuth":            llx.BoolDataPtr(c.CrossOriginAuth),
		"initiateLoginUri":           llx.StringDataPtr(c.InitiateLoginURI),
		"jwtConfiguration":           llx.DictData(jwt),
		"clientMetadata":             llx.MapData(meta, types.String),
	})
	if err != nil {
		return nil, err
	}
	return r, nil
}

// initAuth0Client resolves an application by its client ID on demand, for typed
// references (e.g. auth0.connection.enabledClients) and direct lookups.
func initAuth0Client(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// fast path: the caller already provided a fully populated resource
	if len(args) > 2 {
		return args, nil, nil
	}

	idArg, ok := args["id"]
	if !ok {
		return args, nil, nil
	}
	id, ok := idArg.Value.(string)
	if !ok || id == "" {
		return nil, nil, fmt.Errorf("auth0.client requires a valid id")
	}

	conn := runtime.Connection.(*connection.Auth0Connection)
	c, err := conn.Client().Client.Read(context.Background(), id)
	if err != nil {
		return nil, nil, fmt.Errorf("auth0.client with id %q not found: %w", id, err)
	}

	res, err := newMqlAuth0Client(runtime, c)
	if err != nil {
		return nil, nil, err
	}
	return args, res, nil
}

// enabledConnections resolves the identity connections this application is
// enabled on, as typed auth0.connection references.
func (c *mqlAuth0Client) enabledConnections() ([]any, error) {
	conn := c.MqlRuntime.Connection.(*connection.Auth0Connection)

	list, err := conn.Client().Client.ReadEnabledConnections(context.Background(), c.Id.Data)
	if err != nil {
		return nil, err
	}

	var result []any
	for _, connection := range list.Connections {
		r, err := newMqlAuth0Connection(c.MqlRuntime, connection)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}
