// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	serviceaccount "github.com/stackitcloud/stackit-sdk-go/services/serviceaccount/v2api"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlStackit) serviceAccounts() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.ServiceAccount()
	if err != nil {
		return nil, err
	}
	resp, err := client.DefaultAPI.ListServiceAccounts(bgctx(), c.ProjectID()).Execute()
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		res, err := buildServiceAccount(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlStackitServiceAccount) id() (string, error) {
	return "stackit.serviceAccount/" + r.Email.Data, nil
}

func buildServiceAccount(runtime *plugin.Runtime, sa *serviceaccount.ServiceAccount) (plugin.Resource, error) {
	return CreateResource(runtime, "stackit.serviceAccount", map[string]*llx.RawData{
		"email":     llx.StringData(sa.GetEmail()),
		"projectId": llx.StringData(sa.GetProjectId()),
		"id":        llx.StringData(sa.GetId()),
	})
}

func initStackitServiceAccount(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	email, ok := idArg(args, "email")
	if !ok {
		return args, nil, nil
	}
	c := conn(runtime)
	client, err := c.ServiceAccount()
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.DefaultAPI.ListServiceAccounts(bgctx(), c.ProjectID()).Execute()
	if err != nil {
		return nil, nil, err
	}
	items, _ := resp.GetItemsOk()
	for i := range items {
		if items[i].GetEmail() != email {
			continue
		}
		res, err := buildServiceAccount(runtime, &items[i])
		if err != nil {
			return nil, nil, err
		}
		return nil, res, nil
	}
	return nil, nil, fmt.Errorf("stackit.serviceAccount with email %q not found", email)
}

func (r *mqlStackitServiceAccount) accessTokens() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.ServiceAccount()
	if err != nil {
		return nil, err
	}
	resp, err := client.DefaultAPI.ListAccessTokens(bgctx(), r.ProjectId.Data, r.Email.Data).Execute()
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		out = append(out, serviceAccountTokenEntry(&items[i]))
	}
	return out, nil
}

// serviceAccountTokenEntry maps an access-token metadata record into a
// dict-native map. Timestamps are RFC3339 strings (a `dict` cannot carry a
// *time.Time) so the entry serializes cleanly for the `accessTokens []dict`
// field.
func serviceAccountTokenEntry(t *serviceaccount.AccessTokenMetadata) map[string]any {
	createdAt, ok1 := t.GetCreatedAtOk()
	validUntil, ok2 := t.GetValidUntilOk()
	return map[string]any{
		"id":         t.GetId(),
		"active":     t.GetActive(),
		"createdAt":  rfc3339OrNil(createdAt, ok1),
		"validUntil": rfc3339OrNil(validUntil, ok2),
	}
}

func (r *mqlStackitServiceAccount) keys() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.ServiceAccount()
	if err != nil {
		return nil, err
	}
	resp, err := client.DefaultAPI.ListServiceAccountKeys(bgctx(), r.ProjectId.Data, r.Email.Data).Execute()
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		out = append(out, serviceAccountKeyEntry(&items[i]))
	}
	return out, nil
}

// mqlStackitServiceAccountFederatedIdentityProviderInternal caches the owning
// service account's email so the back-reference resolves without the schema
// repeating it as a raw field.
type mqlStackitServiceAccountFederatedIdentityProviderInternal struct {
	cacheServiceAccountEmail string
}

func (r *mqlStackitServiceAccount) federatedIdentityProviders() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.ServiceAccount()
	if err != nil {
		return nil, err
	}
	resp, err := client.DefaultAPI.ListFederatedIdentityProviders(bgctx(), r.ProjectId.Data, r.Email.Data).Execute()
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		// A service account with federation never configured answers 404
		// rather than an empty list.
		if isNotFound(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetResourcesOk()
	out := make([]any, 0, len(items))
	for i := range items {
		fip := &items[i]
		createdAt, ok1 := fip.GetCreatedAtOk()
		updatedAt, ok2 := fip.GetUpdatedAtOk()

		assertions := make([]any, 0, len(fip.GetAssertions()))
		for _, a := range fip.GetAssertions() {
			assertions = append(assertions, map[string]any{
				"item":     a.GetItem(),
				"operator": a.GetOperator(),
				"value":    a.GetValue(),
			})
		}

		args := map[string]*llx.RawData{
			"id":         llx.StringData(fip.GetId()),
			"name":       llx.StringData(fip.GetName()),
			"issuer":     llx.StringData(fip.GetIssuer()),
			"assertions": llx.ArrayData(assertions, types.Dict),
			"createdAt":  llx.TimeDataPtr(timeOrNil(createdAt, ok1)),
			"updatedAt":  llx.TimeDataPtr(timeOrNil(updatedAt, ok2)),
		}
		res, err := CreateResource(r.MqlRuntime, "stackit.serviceAccount.federatedIdentityProvider", args)
		if err != nil {
			return nil, err
		}
		if mfip, ok := res.(*mqlStackitServiceAccountFederatedIdentityProvider); ok {
			mfip.cacheServiceAccountEmail = r.Email.Data
		}
		out = append(out, res)
	}
	return out, nil
}

// id qualifies the provider UUID with the service account it belongs to. The
// SDK marks the id optional, so fall back to the name, which is unique per
// service account.
func (r *mqlStackitServiceAccountFederatedIdentityProvider) id() (string, error) {
	key := r.Id.Data
	if key == "" {
		key = r.Name.Data
	}
	return "stackit.serviceAccount.federatedIdentityProvider/" + r.cacheServiceAccountEmail + "/" + key, nil
}

func (r *mqlStackitServiceAccountFederatedIdentityProvider) serviceAccount() (*mqlStackitServiceAccount, error) {
	if r.cacheServiceAccountEmail == "" {
		return markNull[mqlStackitServiceAccount](&r.ServiceAccount)
	}
	res, err := NewResource(r.MqlRuntime, "stackit.serviceAccount", map[string]*llx.RawData{
		"email": llx.StringData(r.cacheServiceAccountEmail),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlStackitServiceAccount), nil
}

// serviceAccountKeyEntry maps a service-account key record into a dict-native
// map. The key* fields are named string enums in the SDK, cast to plain
// strings, and the timestamps are RFC3339 strings so the entry serializes
// cleanly for the `keys []dict` field (a `dict` cannot carry a *time.Time or a
// defined string type).
func serviceAccountKeyEntry(k *serviceaccount.ServiceAccountKeyListResponse) map[string]any {
	createdAt, ok1 := k.GetCreatedAtOk()
	validUntil, ok2 := k.GetValidUntilOk()
	return map[string]any{
		"id":           k.GetId(),
		"keyType":      string(k.GetKeyType()),
		"keyAlgorithm": string(k.GetKeyAlgorithm()),
		"keyOrigin":    string(k.GetKeyOrigin()),
		"active":       k.GetActive(),
		"createdAt":    rfc3339OrNil(createdAt, ok1),
		"validUntil":   rfc3339OrNil(validUntil, ok2),
	}
}
