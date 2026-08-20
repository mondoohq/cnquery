// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/openai/openai-go/v3"
	"go.mondoo.com/mql/llx"
)

func (r *mqlOpenai) users() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client, err := adminPlaneClient(conn, "openai.users")
	if err != nil {
		return nil, err
	}
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Users.ListAutoPaging(ctx, openai.AdminOrganizationUserListParams{})
	var res []any
	for iter.Next() {
		u := iter.Current()
		mqlUser, err := CreateResource(r.MqlRuntime, "openai.organizationUser", map[string]*llx.RawData{
			"__id":             llx.StringData(u.ID),
			"id":               llx.StringData(u.ID),
			"email":            llx.StringData(u.Email),
			"name":             llx.StringData(u.Name),
			"role":             llx.StringData(u.Role),
			"isDefault":        llx.BoolData(u.IsDefault),
			"isScimManaged":    llx.BoolData(u.IsScimManaged),
			"isServiceAccount": llx.BoolData(u.IsServiceAccount),
			"addedAt":          llx.TimeDataPtr(unixToNullableTime(u.AddedAt)),
			"createdAt":        llx.TimeDataPtr(unixToNullableTime(u.Created)),
			"apiKeyLastUsedAt": llx.TimeDataPtr(unixToNullableTime(u.APIKeyLastUsedAt)),
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
	client, err := adminPlaneClient(conn, "openai.invites")
	if err != nil {
		return nil, err
	}
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.Admin.Organization.Invites.ListAutoPaging(ctx, openai.AdminOrganizationInviteListParams{})
	var res []any
	for iter.Next() {
		inv := iter.Current()
		mqlInvite, err := CreateResource(r.MqlRuntime, "openai.invite", map[string]*llx.RawData{
			"__id":       llx.StringData(inv.ID),
			"id":         llx.StringData(inv.ID),
			"email":      llx.StringData(inv.Email),
			"role":       llx.StringData(string(inv.Role)),
			"status":     llx.StringData(string(inv.Status)),
			"createdAt":  llx.TimeDataPtr(unixToNullableTime(inv.CreatedAt)),
			"acceptedAt": llx.TimeDataPtr(unixToNullableTime(inv.AcceptedAt)),
			"expiresAt":  llx.TimeDataPtr(unixToNullableTime(inv.ExpiresAt)),
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

// auditLogActor derives the actor type and a human-readable actor identifier
// from an audit log entry's actor. For a session the identifier is the user's
// email; for an API key it is the key ID; otherwise it falls back to the actor
// type string.
func auditLogActor(actor openai.AdminOrganizationAuditLogListResponseActor) (actorType string, actorID string) {
	actorType = actor.Type
	switch actorType {
	case "session":
		actorID = actor.Session.User.Email
	case "api_key":
		actorID = actor.APIKey.ID
	default:
		actorID = actorType
	}
	return actorType, actorID
}

func (r *mqlOpenai) auditLogs() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client, err := adminPlaneClient(conn, "openai.auditLogs")
	if err != nil {
		return nil, err
	}
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
		actorType, actorID := auditLogActor(entry.Actor)

		mqlLog, err := CreateResource(r.MqlRuntime, "openai.auditLog", map[string]*llx.RawData{
			"__id":        llx.StringData(entry.ID),
			"id":          llx.StringData(entry.ID),
			"type":        llx.StringData(string(entry.Type)),
			"effectiveAt": llx.TimeDataPtr(unixToNullableTime(entry.EffectiveAt)),
			"actorType":   llx.StringData(actorType),
			"actorId":     llx.StringData(actorID),
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
