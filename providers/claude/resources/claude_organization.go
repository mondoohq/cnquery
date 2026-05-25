// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/claude/connection"
	"go.mondoo.com/mql/v13/types"
)

// claude.organization

func (r *mqlClaudeOrganization) workspaces() ([]interface{}, error) {
	admin, err := requireAdmin(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	workspaces, err := admin.ListWorkspaces(context.Background())
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0, len(workspaces))
	for _, w := range workspaces {
		createdAt, _ := time.Parse(time.RFC3339, w.CreatedAt)
		var archivedAt time.Time
		if w.ArchivedAt != nil {
			archivedAt, _ = time.Parse(time.RFC3339, *w.ArchivedAt)
		}

		var workspaceGeo, defaultInferenceGeo string
		var allowedInferenceGeos []interface{}
		if w.DataResidency != nil {
			workspaceGeo = w.DataResidency.WorkspaceGeo
			defaultInferenceGeo = w.DataResidency.DefaultInferenceGeo
			allowedInferenceGeos = make([]interface{}, len(w.DataResidency.AllowedInferenceGeos))
			for i, g := range w.DataResidency.AllowedInferenceGeos {
				allowedInferenceGeos[i] = g
			}
		}
		if allowedInferenceGeos == nil {
			allowedInferenceGeos = []interface{}{}
		}

		mqlWs, err := CreateResource(r.MqlRuntime, "claude.organization.workspace", map[string]*llx.RawData{
			"__id":                 llx.StringData(w.ID),
			"id":                   llx.StringData(w.ID),
			"name":                 llx.StringData(w.Name),
			"displayColor":         llx.StringData(w.DisplayColor),
			"createdAt":            llx.TimeData(createdAt),
			"archivedAt":           llx.TimeData(archivedAt),
			"workspaceGeo":         llx.StringData(workspaceGeo),
			"defaultInferenceGeo":  llx.StringData(defaultInferenceGeo),
			"allowedInferenceGeos": llx.ArrayData(allowedInferenceGeos, types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlWs)
	}

	return res, nil
}

func (r *mqlClaudeOrganization) members() ([]interface{}, error) {
	admin, err := requireAdmin(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	users, err := admin.ListUsers(context.Background())
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0, len(users))
	for _, u := range users {
		addedAt, _ := time.Parse(time.RFC3339, u.AddedAt)

		mqlMember, err := CreateResource(r.MqlRuntime, "claude.organization.member", map[string]*llx.RawData{
			"__id":    llx.StringData(u.ID),
			"id":      llx.StringData(u.ID),
			"name":    llx.StringData(u.Name),
			"email":   llx.StringData(u.Email),
			"role":    llx.StringData(u.Role),
			"addedAt": llx.TimeData(addedAt),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlMember)
	}

	return res, nil
}

func (r *mqlClaudeOrganization) invites() ([]interface{}, error) {
	admin, err := requireAdmin(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	invites, err := admin.ListInvites(context.Background())
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0, len(invites))
	for _, inv := range invites {
		invitedAt, _ := time.Parse(time.RFC3339, inv.InvitedAt)
		expiresAt, _ := time.Parse(time.RFC3339, inv.ExpiresAt)

		mqlInvite, err := CreateResource(r.MqlRuntime, "claude.organization.invite", map[string]*llx.RawData{
			"__id":      llx.StringData(inv.ID),
			"id":        llx.StringData(inv.ID),
			"email":     llx.StringData(inv.Email),
			"role":      llx.StringData(inv.Role),
			"status":    llx.StringData(inv.Status),
			"invitedAt": llx.TimeData(invitedAt),
			"expiresAt": llx.TimeData(expiresAt),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInvite)
	}

	return res, nil
}

type mqlClaudeOrganizationApiKeyInternal struct {
	cacheWorkspaceID *string
	cacheCreatedByID string
}

func (r *mqlClaudeOrganization) apiKeys() ([]interface{}, error) {
	admin, err := requireAdmin(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	keys, err := admin.ListAPIKeys(context.Background())
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0, len(keys))
	for _, k := range keys {
		createdAt, _ := time.Parse(time.RFC3339, k.CreatedAt)
		var expiresAt time.Time
		if k.ExpiresAt != nil {
			expiresAt, _ = time.Parse(time.RFC3339, *k.ExpiresAt)
		}

		mqlKey, err := CreateResource(r.MqlRuntime, "claude.organization.apiKey", map[string]*llx.RawData{
			"__id":           llx.StringData(k.ID),
			"id":             llx.StringData(k.ID),
			"name":           llx.StringData(k.Name),
			"status":         llx.StringData(k.Status),
			"createdAt":      llx.TimeData(createdAt),
			"expiresAt":      llx.TimeData(expiresAt),
			"partialKeyHint": llx.StringData(k.PartialKeyHint),
		})
		if err != nil {
			return nil, err
		}

		apiKey := mqlKey.(*mqlClaudeOrganizationApiKey)
		apiKey.cacheWorkspaceID = k.WorkspaceID
		apiKey.cacheCreatedByID = k.CreatedBy.ID

		res = append(res, mqlKey)
	}

	return res, nil
}

func (r *mqlClaudeOrganizationApiKey) createdBy() (*mqlClaudeOrganizationMember, error) {
	if r.cacheCreatedByID == "" {
		r.CreatedBy.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	res, err := CreateResource(r.MqlRuntime, "claude.organization", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	org := res.(*mqlClaudeOrganization)
	members, err := org.members()
	if err != nil {
		return nil, err
	}
	for _, m := range members {
		member := m.(*mqlClaudeOrganizationMember)
		if member.Id.Data == r.cacheCreatedByID {
			return member, nil
		}
	}

	r.CreatedBy.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (r *mqlClaudeOrganizationApiKey) workspace() (*mqlClaudeOrganizationWorkspace, error) {
	if r.cacheWorkspaceID == nil || *r.cacheWorkspaceID == "" {
		r.Workspace.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	res, err := CreateResource(r.MqlRuntime, "claude.organization", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	org := res.(*mqlClaudeOrganization)
	workspaces, err := org.workspaces()
	if err != nil {
		return nil, err
	}
	for _, w := range workspaces {
		ws := w.(*mqlClaudeOrganizationWorkspace)
		if ws.Id.Data == *r.cacheWorkspaceID {
			return ws, nil
		}
	}

	r.Workspace.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

// claude.organization.rateLimit

func (r *mqlClaudeOrganization) rateLimits() ([]interface{}, error) {
	admin, err := requireAdmin(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	limits, err := admin.ListRateLimits(context.Background())
	if err != nil {
		return nil, err
	}

	return convertRateLimits(r.MqlRuntime, limits, "org")
}

func convertRateLimits(runtime *plugin.Runtime, limits []connection.AdminRateLimit, prefix string) ([]interface{}, error) {
	res := make([]interface{}, 0, len(limits))
	for _, l := range limits {
		models := make([]interface{}, len(l.Models))
		for j, m := range l.Models {
			models[j] = m
		}

		mqlLimit, err := CreateResource(runtime, "claude.organization.rateLimit", map[string]*llx.RawData{
			"__id":                  llx.StringData(fmt.Sprintf("%s/ratelimit/%s", prefix, l.GroupType)),
			"groupType":             llx.StringData(l.GroupType),
			"models":                llx.ArrayData(models, types.String),
			"requestsPerMinute":     llx.IntData(l.LimitValue("requests_per_minute")),
			"inputTokensPerMinute":  llx.IntData(l.LimitValue("input_tokens_per_minute_cache_aware")),
			"outputTokensPerMinute": llx.IntData(l.LimitValue("output_tokens_per_minute")),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlLimit)
	}
	return res, nil
}

// claude.organization.workspace members and rate limits

type mqlClaudeOrganizationWorkspaceInternal struct{}

func (r *mqlClaudeOrganizationWorkspace) members() ([]interface{}, error) {
	admin, err := requireAdmin(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	wsID := r.GetId().Data
	members, err := admin.ListWorkspaceMembers(context.Background(), wsID)
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0, len(members))
	for _, m := range members {
		mqlMember, err := CreateResource(r.MqlRuntime, "claude.organization.workspace.member", map[string]*llx.RawData{
			"__id":          llx.StringData(m.WorkspaceID + "/" + m.UserID),
			"userId":        llx.StringData(m.UserID),
			"workspaceId":   llx.StringData(m.WorkspaceID),
			"workspaceRole": llx.StringData(m.WorkspaceRole),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlMember)
	}

	return res, nil
}

func (r *mqlClaudeOrganizationWorkspace) rateLimits() ([]interface{}, error) {
	admin, err := requireAdmin(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	wsID := r.GetId().Data
	limits, err := admin.ListWorkspaceRateLimits(context.Background(), wsID)
	if err != nil {
		return nil, err
	}

	return convertRateLimits(r.MqlRuntime, limits, "ws/"+wsID)
}

// claude.organization.usageEntry

func (r *mqlClaudeOrganization) usageReport() ([]interface{}, error) {
	admin, err := requireAdmin(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	buckets, err := admin.ListUsageReport(context.Background())
	if err != nil {
		return nil, err
	}

	var res []interface{}
	for _, b := range buckets {
		startingAt, _ := time.Parse(time.RFC3339, b.StartingAt)
		endingAt, _ := time.Parse(time.RFC3339, b.EndingAt)

		for _, result := range b.Results {
			mqlEntry, err := CreateResource(r.MqlRuntime, "claude.organization.usageEntry", map[string]*llx.RawData{
				"__id":                 llx.StringData(fmt.Sprintf("usage/%s/%s/%s/%s", b.StartingAt, result.Model, result.WorkspaceID, result.ServiceTier)),
				"startingAt":           llx.TimeData(startingAt),
				"endingAt":             llx.TimeData(endingAt),
				"model":                llx.StringData(result.Model),
				"workspaceId":          llx.StringData(result.WorkspaceID),
				"serviceTier":          llx.StringData(result.ServiceTier),
				"uncachedInputTokens":  llx.IntData(result.UncachedInputTokens),
				"cacheReadInputTokens": llx.IntData(result.CacheReadInputTokens),
				"outputTokens":         llx.IntData(result.OutputTokens),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlEntry)
		}
	}

	return res, nil
}

// claude.organization.costEntry

func (r *mqlClaudeOrganization) costReport() ([]interface{}, error) {
	admin, err := requireAdmin(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	buckets, err := admin.ListCostReport(context.Background())
	if err != nil {
		return nil, err
	}

	var res []interface{}
	for _, b := range buckets {
		startingAt, _ := time.Parse(time.RFC3339, b.StartingAt)
		endingAt, _ := time.Parse(time.RFC3339, b.EndingAt)

		for _, result := range b.Results {
			mqlEntry, err := CreateResource(r.MqlRuntime, "claude.organization.costEntry", map[string]*llx.RawData{
				"__id":        llx.StringData(fmt.Sprintf("cost/%s/%s/%s/%s", b.StartingAt, result.CostType, result.Model, result.WorkspaceID)),
				"startingAt":  llx.TimeData(startingAt),
				"endingAt":    llx.TimeData(endingAt),
				"amount":      llx.StringData(result.Amount),
				"currency":    llx.StringData(result.Currency),
				"costType":    llx.StringData(result.CostType),
				"model":       llx.StringData(result.Model),
				"workspaceId": llx.StringData(result.WorkspaceID),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlEntry)
		}
	}

	return res, nil
}

// claude.organization.activity

func (r *mqlClaudeOrganization) activities() ([]interface{}, error) {
	admin, err := requireAdmin(r.MqlRuntime)
	if err != nil {
		return nil, err
	}

	activities, err := admin.ListActivities(context.Background())
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0, len(activities))
	for _, a := range activities {
		createdAt, _ := time.Parse(time.RFC3339, a.CreatedAt)

		mqlActivity, err := CreateResource(r.MqlRuntime, "claude.organization.activity", map[string]*llx.RawData{
			"__id":       llx.StringData(a.ID),
			"id":         llx.StringData(a.ID),
			"type":       llx.StringData(a.Type),
			"actorEmail": llx.StringData(a.Actor.Email),
			"actorId":    llx.StringData(a.Actor.ID),
			"createdAt":  llx.TimeData(createdAt),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlActivity)
	}

	return res, nil
}
