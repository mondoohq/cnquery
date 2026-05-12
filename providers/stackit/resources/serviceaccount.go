// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/stackitcloud/stackit-sdk-go/services/serviceaccount"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func (r *mqlStackit) serviceAccounts() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.ServiceAccount()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListServiceAccountsExecute(bgctx(), c.ProjectID())
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
	resp, err := client.ListServiceAccountsExecute(bgctx(), c.ProjectID())
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
	return args, nil, nil
}

func (r *mqlStackitServiceAccount) accessTokens() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.ServiceAccount()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListAccessTokensExecute(bgctx(), r.ProjectId.Data, r.Email.Data)
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		t := &items[i]
		createdAt, ok1 := t.GetCreatedAtOk()
		validUntil, ok2 := t.GetValidUntilOk()
		entry := map[string]any{
			"id":         t.GetId(),
			"active":     t.GetActive(),
			"createdAt":  timeOrNil(createdAt, ok1),
			"validUntil": timeOrNil(validUntil, ok2),
		}
		out = append(out, entry)
	}
	return out, nil
}

func (r *mqlStackitServiceAccount) keys() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.ServiceAccount()
	if err != nil {
		return nil, err
	}
	resp, err := client.ListServiceAccountKeysExecute(bgctx(), r.ProjectId.Data, r.Email.Data)
	if err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, err
	}
	items, _ := resp.GetItemsOk()
	out := make([]any, 0, len(items))
	for i := range items {
		k := &items[i]
		createdAt, ok1 := k.GetCreatedAtOk()
		validUntil, ok2 := k.GetValidUntilOk()
		entry := map[string]any{
			"id":           k.GetId(),
			"keyType":      k.GetKeyType(),
			"keyAlgorithm": k.GetKeyAlgorithm(),
			"keyOrigin":    k.GetKeyOrigin(),
			"active":       k.GetActive(),
			"createdAt":    timeOrNil(createdAt, ok1),
			"validUntil":   timeOrNil(validUntil, ok2),
		}
		out = append(out, entry)
	}
	return out, nil
}
