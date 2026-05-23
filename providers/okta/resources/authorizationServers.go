// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/okta/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOkta) authorizationServers() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()

	ctx := context.Background()
	servers, resp, err := client.AuthorizationServer.ListAuthorizationServers(
		ctx,
		query.NewQueryParams(query.WithLimit(queryLimit)),
	)
	if err != nil {
		return nil, err
	}

	list := []any{}
	appendEntry := func(entries []*okta.AuthorizationServer) error {
		for i := range entries {
			r, err := newMqlOktaAuthorizationServer(o.MqlRuntime, entries[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	if err := appendEntry(servers); err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var page []*okta.AuthorizationServer
		resp, err = resp.Next(ctx, &page)
		if err != nil {
			return nil, err
		}
		if err := appendEntry(page); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func newMqlOktaAuthorizationServer(runtime *plugin.Runtime, entry *okta.AuthorizationServer) (any, error) {
	defaultValue := false
	if entry.Default != nil {
		defaultValue = *entry.Default
	}

	var signingKid, signingRotationMode, signingUse string
	var signingLastRotated, signingNextRotation *time.Time
	if entry.Credentials != nil && entry.Credentials.Signing != nil {
		s := entry.Credentials.Signing
		signingKid = s.Kid
		signingRotationMode = s.RotationMode
		signingUse = s.Use
		signingLastRotated = s.LastRotated
		signingNextRotation = s.NextRotation
	}

	return CreateResource(runtime, "okta.authorizationServer", map[string]*llx.RawData{
		"id":                  llx.StringData(entry.Id),
		"name":                llx.StringData(entry.Name),
		"description":         llx.StringData(entry.Description),
		"status":              llx.StringData(entry.Status),
		"default":             llx.BoolData(defaultValue),
		"issuer":              llx.StringData(entry.Issuer),
		"issuerMode":          llx.StringData(entry.IssuerMode),
		"audiences":           llx.ArrayData(convert.SliceAnyToInterface(entry.Audiences), types.String),
		"signingKid":          llx.StringData(signingKid),
		"signingRotationMode": llx.StringData(signingRotationMode),
		"signingLastRotated":  llx.TimeDataPtr(signingLastRotated),
		"signingNextRotation": llx.TimeDataPtr(signingNextRotation),
		"signingUse":          llx.StringData(signingUse),
		"created":             llx.TimeDataPtr(entry.Created),
		"lastUpdated":         llx.TimeDataPtr(entry.LastUpdated),
	})
}

func (o *mqlOktaAuthorizationServer) id() (string, error) {
	return "okta.authorizationServer/" + o.Id.Data, o.Id.Error
}

func (o *mqlOktaAuthorizationServer) policies() ([]any, error) {
	if o.Id.Error != nil {
		return nil, o.Id.Error
	}
	if o.Id.Data == "" {
		return []any{}, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()
	ctx := context.Background()

	policies, resp, err := client.AuthorizationServer.ListAuthorizationServerPolicies(ctx, o.Id.Data)
	if err != nil {
		return nil, err
	}

	list := []any{}
	appendEntry := func(entries []*okta.AuthorizationServerPolicy) error {
		for i := range entries {
			r, err := newMqlOktaAuthorizationServerPolicy(o.MqlRuntime, o.Id.Data, entries[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	if err := appendEntry(policies); err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var page []*okta.AuthorizationServerPolicy
		resp, err = resp.Next(ctx, &page)
		if err != nil {
			return nil, err
		}
		if err := appendEntry(page); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (o *mqlOktaAuthorizationServer) scopes() ([]any, error) {
	if o.Id.Error != nil {
		return nil, o.Id.Error
	}
	if o.Id.Data == "" {
		return []any{}, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()
	ctx := context.Background()

	scopes, resp, err := client.AuthorizationServer.ListOAuth2Scopes(
		ctx,
		o.Id.Data,
		query.NewQueryParams(query.WithLimit(queryLimit)),
	)
	if err != nil {
		return nil, err
	}

	list := []any{}
	appendEntry := func(entries []*okta.OAuth2Scope) error {
		for i := range entries {
			r, err := newMqlOktaAuthorizationServerScope(o.MqlRuntime, o.Id.Data, entries[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	if err := appendEntry(scopes); err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var page []*okta.OAuth2Scope
		resp, err = resp.Next(ctx, &page)
		if err != nil {
			return nil, err
		}
		if err := appendEntry(page); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (o *mqlOktaAuthorizationServer) claims() ([]any, error) {
	if o.Id.Error != nil {
		return nil, o.Id.Error
	}
	if o.Id.Data == "" {
		return []any{}, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()
	ctx := context.Background()

	claims, resp, err := client.AuthorizationServer.ListOAuth2Claims(ctx, o.Id.Data)
	if err != nil {
		return nil, err
	}

	list := []any{}
	appendEntry := func(entries []*okta.OAuth2Claim) error {
		for i := range entries {
			r, err := newMqlOktaAuthorizationServerClaim(o.MqlRuntime, o.Id.Data, entries[i])
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	if err := appendEntry(claims); err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var page []*okta.OAuth2Claim
		resp, err = resp.Next(ctx, &page)
		if err != nil {
			return nil, err
		}
		if err := appendEntry(page); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (o *mqlOktaAuthorizationServer) keys() ([]any, error) {
	if o.Id.Error != nil {
		return nil, o.Id.Error
	}
	if o.Id.Data == "" {
		return []any{}, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()
	ctx := context.Background()

	keys, _, err := client.AuthorizationServer.ListAuthorizationServerKeys(ctx, o.Id.Data)
	if err != nil {
		return nil, err
	}

	list := []any{}
	for i := range keys {
		k := keys[i]
		if k == nil {
			continue
		}
		r, err := CreateResource(o.MqlRuntime, "okta.authorizationServer.key", map[string]*llx.RawData{
			"authorizationServerId": llx.StringData(o.Id.Data),
			"kid":                   llx.StringData(k.Kid),
			"status":                llx.StringData(k.Status),
			"alg":                   llx.StringData(k.Alg),
			"kty":                   llx.StringData(k.Kty),
			"use":                   llx.StringData(k.Use),
			"keyOps":                llx.ArrayData(convert.SliceAnyToInterface(k.KeyOps), types.String),
			"created":               llx.TimeDataPtr(k.Created),
			"lastUpdated":           llx.TimeDataPtr(k.LastUpdated),
			"expiresAt":             llx.TimeDataPtr(k.ExpiresAt),
			"x5c":                   llx.ArrayData(convert.SliceAnyToInterface(k.X5c), types.String),
			"x5t":                   llx.StringData(k.X5t),
			"x5tS256":               llx.StringData(k.X5tS256),
			"n":                     llx.StringData(k.N),
			"e":                     llx.StringData(k.E),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, nil
}

func (o *mqlOktaAuthorizationServerKey) id() (string, error) {
	return "okta.authorizationServer.key/" + o.AuthorizationServerId.Data + "/" + o.Kid.Data, o.Kid.Error
}

func newMqlOktaAuthorizationServerPolicy(runtime *plugin.Runtime, authServerId string, entry *okta.AuthorizationServerPolicy) (any, error) {
	conditions, err := convert.JsonToDict(entry.Conditions)
	if err != nil {
		return nil, err
	}

	system := false
	if entry.System != nil {
		system = *entry.System
	}

	return CreateResource(runtime, "okta.authorizationServer.policy", map[string]*llx.RawData{
		"authorizationServerId": llx.StringData(authServerId),
		"id":                    llx.StringData(entry.Id),
		"name":                  llx.StringData(entry.Name),
		"description":           llx.StringData(entry.Description),
		"priority":              llx.IntData(entry.Priority),
		"status":                llx.StringData(entry.Status),
		"system":                llx.BoolData(system),
		"type":                  llx.StringData(entry.Type),
		"conditions":            llx.DictData(conditions),
		"created":               llx.TimeDataPtr(entry.Created),
		"lastUpdated":           llx.TimeDataPtr(entry.LastUpdated),
	})
}

func (o *mqlOktaAuthorizationServerPolicy) id() (string, error) {
	return "okta.authorizationServer.policy/" + o.AuthorizationServerId.Data + "/" + o.Id.Data, o.Id.Error
}

func (o *mqlOktaAuthorizationServerPolicy) rules() ([]any, error) {
	if o.Id.Error != nil {
		return nil, o.Id.Error
	}
	if o.AuthorizationServerId.Data == "" || o.Id.Data == "" {
		return []any{}, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OktaConnection)
	client := conn.Client()
	ctx := context.Background()

	rules, resp, err := client.AuthorizationServer.ListAuthorizationServerPolicyRules(
		ctx, o.AuthorizationServerId.Data, o.Id.Data,
	)
	if err != nil {
		return nil, err
	}

	list := []any{}
	appendEntry := func(entries []*okta.AuthorizationServerPolicyRule) error {
		for i := range entries {
			entry := entries[i]
			actions, err := convert.JsonToDict(entry.Actions)
			if err != nil {
				return err
			}
			conditions, err := convert.JsonToDict(entry.Conditions)
			if err != nil {
				return err
			}
			system := false
			if entry.System != nil {
				system = *entry.System
			}

			r, err := CreateResource(o.MqlRuntime, "okta.authorizationServer.policyRule", map[string]*llx.RawData{
				"authorizationServerId": llx.StringData(o.AuthorizationServerId.Data),
				"policyId":              llx.StringData(o.Id.Data),
				"id":                    llx.StringData(entry.Id),
				"name":                  llx.StringData(entry.Name),
				"priority":              llx.IntData(entry.Priority),
				"status":                llx.StringData(entry.Status),
				"system":                llx.BoolData(system),
				"type":                  llx.StringData(entry.Type),
				"actions":               llx.DictData(actions),
				"conditions":            llx.DictData(conditions),
				"created":               llx.TimeDataPtr(entry.Created),
				"lastUpdated":           llx.TimeDataPtr(entry.LastUpdated),
			})
			if err != nil {
				return err
			}
			list = append(list, r)
		}
		return nil
	}

	if err := appendEntry(rules); err != nil {
		return nil, err
	}

	for resp != nil && resp.HasNextPage() {
		var page []*okta.AuthorizationServerPolicyRule
		resp, err = resp.Next(ctx, &page)
		if err != nil {
			return nil, err
		}
		if err := appendEntry(page); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (o *mqlOktaAuthorizationServerPolicyRule) id() (string, error) {
	return "okta.authorizationServer.policyRule/" + o.AuthorizationServerId.Data + "/" + o.PolicyId.Data + "/" + o.Id.Data, o.Id.Error
}

func newMqlOktaAuthorizationServerScope(runtime *plugin.Runtime, authServerId string, entry *okta.OAuth2Scope) (any, error) {
	defaultValue := false
	if entry.Default != nil {
		defaultValue = *entry.Default
	}
	system := false
	if entry.System != nil {
		system = *entry.System
	}

	return CreateResource(runtime, "okta.authorizationServer.scope", map[string]*llx.RawData{
		"authorizationServerId": llx.StringData(authServerId),
		"id":                    llx.StringData(entry.Id),
		"name":                  llx.StringData(entry.Name),
		"displayName":           llx.StringData(entry.DisplayName),
		"description":           llx.StringData(entry.Description),
		"consent":               llx.StringData(entry.Consent),
		"default":               llx.BoolData(defaultValue),
		"metadataPublish":       llx.StringData(entry.MetadataPublish),
		"system":                llx.BoolData(system),
	})
}

func (o *mqlOktaAuthorizationServerScope) id() (string, error) {
	return "okta.authorizationServer.scope/" + o.AuthorizationServerId.Data + "/" + o.Id.Data, o.Id.Error
}

func newMqlOktaAuthorizationServerClaim(runtime *plugin.Runtime, authServerId string, entry *okta.OAuth2Claim) (any, error) {
	alwaysInclude := false
	if entry.AlwaysIncludeInToken != nil {
		alwaysInclude = *entry.AlwaysIncludeInToken
	}
	system := false
	if entry.System != nil {
		system = *entry.System
	}

	var scopes []string
	if entry.Conditions != nil {
		scopes = entry.Conditions.Scopes
	}

	return CreateResource(runtime, "okta.authorizationServer.claim", map[string]*llx.RawData{
		"authorizationServerId": llx.StringData(authServerId),
		"id":                    llx.StringData(entry.Id),
		"name":                  llx.StringData(entry.Name),
		"status":                llx.StringData(entry.Status),
		"system":                llx.BoolData(system),
		"claimType":             llx.StringData(entry.ClaimType),
		"valueType":             llx.StringData(entry.ValueType),
		"value":                 llx.StringData(entry.Value),
		"alwaysIncludeInToken":  llx.BoolData(alwaysInclude),
		"groupFilterType":       llx.StringData(entry.GroupFilterType),
		"scopes":                llx.ArrayData(convert.SliceAnyToInterface(scopes), types.String),
	})
}

func (o *mqlOktaAuthorizationServerClaim) id() (string, error) {
	return "okta.authorizationServer.claim/" + o.AuthorizationServerId.Data + "/" + o.Id.Data, o.Id.Error
}
