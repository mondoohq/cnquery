// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/openai/openai-go/v3"
	"go.mondoo.com/mql/v13/llx"
)

func (r *mqlOpenai) projects() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.AdminClient()
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Projects.ListAutoPaging(ctx, openai.AdminOrganizationProjectListParams{})
	var res []any
	for iter.Next() {
		p := iter.Current()
		created := unixToTime(p.CreatedAt)

		var archivedAt *time.Time
		if p.ArchivedAt != 0 {
			t := unixToTime(p.ArchivedAt)
			archivedAt = &t
		}

		mqlProject, err := CreateResource(r.MqlRuntime, "openai.project", map[string]*llx.RawData{
			"__id":       llx.StringData(p.ID),
			"id":         llx.StringData(p.ID),
			"name":       llx.StringData(p.Name),
			"status":     llx.StringData(string(p.Status)),
			"createdAt":  llx.TimeData(created),
			"archivedAt": llx.TimeDataPtr(archivedAt),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlProject)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	return res, nil
}

func (r *mqlOpenaiProject) apiKeys() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.AdminClient()
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Projects.APIKeys.ListAutoPaging(ctx, r.Id.Data, openai.AdminOrganizationProjectAPIKeyListParams{})
	var res []any
	for iter.Next() {
		k := iter.Current()
		created := unixToTime(k.CreatedAt)
		lastUsed := unixToTime(k.LastUsedAt)

		ownerType := k.Owner.Type
		var ownerName, ownerId string
		switch ownerType {
		case "user":
			ownerName = k.Owner.User.Email
			ownerId = k.Owner.User.ID
		case "service_account":
			ownerName = k.Owner.ServiceAccount.Name
			ownerId = k.Owner.ServiceAccount.ID
		}

		mqlKey, err := CreateResource(r.MqlRuntime, "openai.project.apiKey", map[string]*llx.RawData{
			"__id":          llx.StringData(k.ID),
			"id":            llx.StringData(k.ID),
			"name":          llx.StringData(k.Name),
			"redactedValue": llx.StringData(k.RedactedValue),
			"createdAt":     llx.TimeData(created),
			"lastUsedAt":    llx.TimeData(lastUsed),
			"ownerType":     llx.StringData(ownerType),
			"ownerName":     llx.StringData(ownerName),
			"ownerId":       llx.StringData(ownerId),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlKey)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to list project API keys: %w", err)
	}
	return res, nil
}

func (r *mqlOpenaiProject) serviceAccounts() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.AdminClient()
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Projects.ServiceAccounts.ListAutoPaging(ctx, r.Id.Data, openai.AdminOrganizationProjectServiceAccountListParams{})
	var res []any
	for iter.Next() {
		sa := iter.Current()
		created := unixToTime(sa.CreatedAt)

		mqlSA, err := CreateResource(r.MqlRuntime, "openai.project.serviceAccount", map[string]*llx.RawData{
			"__id":      llx.StringData(sa.ID),
			"id":        llx.StringData(sa.ID),
			"name":      llx.StringData(sa.Name),
			"role":      llx.StringData(string(sa.Role)),
			"createdAt": llx.TimeData(created),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSA)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to list project service accounts: %w", err)
	}
	return res, nil
}
