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

func (r *mqlOpenai) users() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.AdminClient()
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Users.ListAutoPaging(ctx, openai.AdminOrganizationUserListParams{})
	var res []any
	for iter.Next() {
		u := iter.Current()
		addedAt := unixToTime(u.AddedAt)
		created := unixToTime(u.Created)

		var apiKeyLastUsedAt *time.Time
		if u.APIKeyLastUsedAt != 0 {
			t := unixToTime(u.APIKeyLastUsedAt)
			apiKeyLastUsedAt = &t
		}

		mqlUser, err := CreateResource(r.MqlRuntime, "openai.organizationUser", map[string]*llx.RawData{
			"__id":             llx.StringData(u.ID),
			"id":               llx.StringData(u.ID),
			"email":            llx.StringData(u.Email),
			"name":             llx.StringData(u.Name),
			"role":             llx.StringData(u.Role),
			"isDefault":        llx.BoolData(u.IsDefault),
			"isScimManaged":    llx.BoolData(u.IsScimManaged),
			"isServiceAccount": llx.BoolData(u.IsServiceAccount),
			"addedAt":          llx.TimeData(addedAt),
			"createdAt":        llx.TimeData(created),
			"apiKeyLastUsedAt": llx.TimeDataPtr(apiKeyLastUsedAt),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlUser)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to list organization users: %w", err)
	}
	return res, nil
}

func (r *mqlOpenai) invites() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.AdminClient()
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Invites.ListAutoPaging(ctx, openai.AdminOrganizationInviteListParams{})
	var res []any
	for iter.Next() {
		inv := iter.Current()
		created := unixToTime(inv.CreatedAt)

		var acceptedAt *time.Time
		if inv.AcceptedAt != 0 {
			t := unixToTime(inv.AcceptedAt)
			acceptedAt = &t
		}

		var expiresAt *time.Time
		if inv.ExpiresAt != 0 {
			t := unixToTime(inv.ExpiresAt)
			expiresAt = &t
		}

		mqlInvite, err := CreateResource(r.MqlRuntime, "openai.invite", map[string]*llx.RawData{
			"__id":       llx.StringData(inv.ID),
			"id":         llx.StringData(inv.ID),
			"email":      llx.StringData(inv.Email),
			"role":       llx.StringData(string(inv.Role)),
			"status":     llx.StringData(string(inv.Status)),
			"createdAt":  llx.TimeData(created),
			"acceptedAt": llx.TimeDataPtr(acceptedAt),
			"expiresAt":  llx.TimeDataPtr(expiresAt),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInvite)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to list invites: %w", err)
	}
	return res, nil
}

func (r *mqlOpenai) auditLogs() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.AdminClient()
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	since := time.Now().AddDate(0, 0, -30)
	iter := client.Admin.Organization.AuditLogs.ListAutoPaging(ctx, openai.AdminOrganizationAuditLogListParams{
		EffectiveAt: openai.AdminOrganizationAuditLogListParamsEffectiveAt{
			Gte: openai.Int(since.Unix()),
		},
	})
	var res []any
	for iter.Next() {
		entry := iter.Current()
		effectiveAt := unixToTime(entry.EffectiveAt)

		actorType := entry.Actor.Type
		var actorId string
		switch actorType {
		case "session":
			actorId = entry.Actor.Session.User.Email
		case "api_key":
			actorId = entry.Actor.APIKey.ID
		default:
			actorId = string(actorType)
		}

		mqlLog, err := CreateResource(r.MqlRuntime, "openai.auditLog", map[string]*llx.RawData{
			"__id":        llx.StringData(entry.ID),
			"id":          llx.StringData(entry.ID),
			"type":        llx.StringData(string(entry.Type)),
			"effectiveAt": llx.TimeData(effectiveAt),
			"actorType":   llx.StringData(actorType),
			"actorId":     llx.StringData(actorId),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlLog)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	return res, nil
}
