// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/okta/okta-sdk-golang/v5/okta"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/okta/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOkta) identityProviders() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	idps, resp, err := client.IdentityProviderAPI.ListIdentityProviders(ctx).Limit(queryLimit).Execute()
	if err != nil {
		return nil, err
	}

	list := []any{}
	appendEntry := func(entries []okta.IdentityProvider) error {
		for i := range entries {
			r, err := newMqlOktaIdentityProvider(o.MqlRuntime, &entries[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	if err := appendEntry(idps); err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var page []okta.IdentityProvider
		resp, err = resp.Next(&page)
		if err != nil {
			return nil, err
		}
		if err := appendEntry(page); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaIdentityProvider(runtime *plugin.Runtime, entry *okta.IdentityProvider) (any, error) {
	protocol, err := convert.JsonToDict(entry.Protocol)
	if err != nil {
		return nil, err
	}

	policy, err := convert.JsonToDict(entry.Policy)
	if err != nil {
		return nil, err
	}

	return CreateResource(runtime, "okta.identityProvider", map[string]*llx.RawData{
		"id":          llx.StringData(oktaStr(entry.Id)),
		"name":        llx.StringData(oktaStr(entry.Name)),
		"type":        llx.StringData(oktaStr(entry.Type)),
		"status":      llx.StringData(oktaStr(entry.Status)),
		"issuerMode":  llx.StringData(oktaStr(entry.IssuerMode)),
		"protocol":    llx.DictData(protocol),
		"policy":      llx.DictData(policy),
		"created":     llx.TimeDataPtr(entry.Created.Get()),
		"lastUpdated": llx.TimeDataPtr(entry.LastUpdated),
	})
}

func (o *mqlOktaIdentityProvider) id() (string, error) {
	return "okta.identityProvider/" + o.Id.Data, o.Id.Error
}

func (o *mqlOktaIdentityProvider) signingKeys() ([]any, error) {
	if o.Id.Error != nil {
		return nil, o.Id.Error
	}
	if o.Id.Data == "" {
		return []any{}, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	keys, resp, err := client.IdentityProviderAPI.ListIdentityProviderSigningKeys(ctx, o.Id.Data).Execute()
	if err != nil {
		return nil, err
	}

	list := []any{}
	appendKeys := func(entries []okta.JsonWebKey) error {
		for i := range entries {
			k := entries[i]
			r, err := CreateResource(o.MqlRuntime, "okta.identityProvider.key", map[string]*llx.RawData{
				"identityProviderId": llx.StringData(o.Id.Data),
				"kid":                llx.StringData(oktaStr(k.Kid)),
				"status":             llx.StringData(oktaStr(k.Status)),
				"alg":                llx.StringData(oktaStr(k.Alg)),
				"kty":                llx.StringData(oktaStr(k.Kty)),
				"use":                llx.StringData(oktaStr(k.Use)),
				"keyOps":             llx.ArrayData(convert.SliceAnyToInterface(k.KeyOps), types.String),
				"created":            llx.TimeDataPtr(k.Created),
				"lastUpdated":        llx.TimeDataPtr(k.LastUpdated),
				"expiresAt":          llx.TimeDataPtr(k.ExpiresAt),
				"x5c":                llx.ArrayData(convert.SliceAnyToInterface(k.X5c), types.String),
				"x5t":                llx.StringData(oktaStr(k.X5t)),
				"x5tS256":            llx.StringData(oktaStr(k.X5tS256)),
				"n":                  llx.StringData(oktaStr(k.N)),
				"e":                  llx.StringData(oktaStr(k.E)),
			})
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	if err := appendKeys(keys); err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var page []okta.JsonWebKey
		resp, err = resp.Next(&page)
		if err != nil {
			return nil, err
		}
		if err := appendKeys(page); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (o *mqlOktaIdentityProviderKey) id() (string, error) {
	return "okta.identityProvider.key/" + o.IdentityProviderId.Data + "/" + o.Kid.Data, o.Kid.Error
}
