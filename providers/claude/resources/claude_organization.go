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

// lookupWorkspace returns the organization workspace carrying the given id.
func lookupWorkspace(runtime *plugin.Runtime, id string) (*mqlClaudeOrganizationWorkspace, bool, error) {
	return lookupOrganizationChild[*mqlClaudeOrganizationWorkspace](runtime, id, (*mqlClaudeOrganization).GetWorkspaces)
}

// lookupMember returns the organization member carrying the given id.
func lookupMember(runtime *plugin.Runtime, id string) (*mqlClaudeOrganizationMember, bool, error) {
	return lookupOrganizationChild[*mqlClaudeOrganizationMember](runtime, id, (*mqlClaudeOrganization).GetMembers)
}

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
		createdAt, err := parseTime(w.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing workspace createdAt: %w", err)
		}
		var archivedAt time.Time
		if w.ArchivedAt != nil {
			archivedAt, err = parseTime(*w.ArchivedAt)
			if err != nil {
				return nil, fmt.Errorf("parsing workspace archivedAt: %w", err)
			}
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
		addedAt, err := parseTime(u.AddedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing member addedAt: %w", err)
		}

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
		invitedAt, err := parseTime(inv.InvitedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing invite invitedAt: %w", err)
		}
		expiresAt, err := parseTime(inv.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("parsing invite expiresAt: %w", err)
		}

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
		createdAt, err := parseTime(k.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing apiKey createdAt: %w", err)
		}
		var expiresAt time.Time
		if k.ExpiresAt != nil {
			expiresAt, err = parseTime(*k.ExpiresAt)
			if err != nil {
				return nil, fmt.Errorf("parsing apiKey expiresAt: %w", err)
			}
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
	member, ok, err := lookupMember(r.MqlRuntime, r.cacheCreatedByID)
	if err != nil {
		return nil, err
	}
	if !ok {
		r.CreatedBy.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return member, nil
}

func (r *mqlClaudeOrganizationApiKey) workspace() (*mqlClaudeOrganizationWorkspace, error) {
	var workspaceID string
	if r.cacheWorkspaceID != nil {
		workspaceID = *r.cacheWorkspaceID
	}

	ws, ok, err := lookupWorkspace(r.MqlRuntime, workspaceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		r.Workspace.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return ws, nil
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
			"__id":                  llx.StringData(prefix + "/ratelimit/" + l.GroupType),
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

type mqlClaudeOrganizationWorkspaceMemberInternal struct {
	cacheUserID      string
	cacheWorkspaceID string
}

func (r *mqlClaudeOrganizationWorkspaceMember) user() (*mqlClaudeOrganizationMember, error) {
	member, ok, err := lookupMember(r.MqlRuntime, r.cacheUserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		r.User.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return member, nil
}

func (r *mqlClaudeOrganizationWorkspaceMember) workspace() (*mqlClaudeOrganizationWorkspace, error) {
	ws, ok, err := lookupWorkspace(r.MqlRuntime, r.cacheWorkspaceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		r.Workspace.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return ws, nil
}

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

		wsMember := mqlMember.(*mqlClaudeOrganizationWorkspaceMember)
		wsMember.cacheUserID = m.UserID
		wsMember.cacheWorkspaceID = m.WorkspaceID

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

type mqlClaudeOrganizationUsageEntryInternal struct {
	cacheWorkspaceID string
}

func (r *mqlClaudeOrganizationUsageEntry) workspace() (*mqlClaudeOrganizationWorkspace, error) {
	ws, ok, err := lookupWorkspace(r.MqlRuntime, r.cacheWorkspaceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		r.Workspace.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return ws, nil
}

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
		startingAt, err := parseTime(b.StartingAt)
		if err != nil {
			return nil, fmt.Errorf("parsing usage startingAt: %w", err)
		}
		endingAt, err := parseTime(b.EndingAt)
		if err != nil {
			return nil, fmt.Errorf("parsing usage endingAt: %w", err)
		}

		for _, result := range b.Results {
			mqlEntry, err := CreateResource(r.MqlRuntime, "claude.organization.usageEntry", map[string]*llx.RawData{
				"__id":                 llx.StringData("usage/" + b.StartingAt + "/" + result.Model + "/" + result.WorkspaceID + "/" + result.ServiceTier),
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

			mqlEntry.(*mqlClaudeOrganizationUsageEntry).cacheWorkspaceID = result.WorkspaceID

			res = append(res, mqlEntry)
		}
	}

	return res, nil
}

// claude.organization.costEntry

type mqlClaudeOrganizationCostEntryInternal struct {
	cacheWorkspaceID string
}

func (r *mqlClaudeOrganizationCostEntry) workspace() (*mqlClaudeOrganizationWorkspace, error) {
	ws, ok, err := lookupWorkspace(r.MqlRuntime, r.cacheWorkspaceID)
	if err != nil {
		return nil, err
	}
	if !ok {
		r.Workspace.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return ws, nil
}

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
		startingAt, err := parseTime(b.StartingAt)
		if err != nil {
			return nil, fmt.Errorf("parsing cost startingAt: %w", err)
		}
		endingAt, err := parseTime(b.EndingAt)
		if err != nil {
			return nil, fmt.Errorf("parsing cost endingAt: %w", err)
		}

		for _, result := range b.Results {
			mqlEntry, err := CreateResource(r.MqlRuntime, "claude.organization.costEntry", map[string]*llx.RawData{
				"__id":        llx.StringData("cost/" + b.StartingAt + "/" + result.CostType + "/" + result.Model + "/" + result.WorkspaceID),
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

			mqlEntry.(*mqlClaudeOrganizationCostEntry).cacheWorkspaceID = result.WorkspaceID

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
		createdAt, err := parseTime(a.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing activity createdAt: %w", err)
		}

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
