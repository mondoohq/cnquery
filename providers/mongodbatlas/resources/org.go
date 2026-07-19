// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/types"
	"go.mongodb.org/atlas-sdk/v20250312006/admin"
)

// pageSize is the per-request page size used for the SDK's manual pagination.
const pageSize = 500

// dictTime renders a time for inclusion in a dict field, using nil for the zero
// time so the value round-trips cleanly across the plugin boundary.
func dictTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.Format(time.RFC3339)
}

func (r *mqlMongodbatlas) projects() ([]any, error) {
	client := atlasClient(r.MqlRuntime)
	ctx := context.Background()

	out := []any{}
	for page := 1; ; page++ {
		resp, _, err := client.ProjectsApi.ListProjects(ctx).ItemsPerPage(pageSize).PageNum(page).Execute()
		if err != nil {
			return nil, err
		}
		results := resp.GetResults()
		for i := range results {
			p := results[i]
			res, err := CreateResource(r.MqlRuntime, "mongodbatlas.project", map[string]*llx.RawData{
				"__id":                    llx.StringData("mongodbatlas.project/" + p.GetId()),
				"id":                      llx.StringData(p.GetId()),
				"name":                    llx.StringData(p.GetName()),
				"orgId":                   llx.StringData(p.GetOrgId()),
				"clusterCount":            llx.IntData(p.GetClusterCount()),
				"created":                 llx.TimeDataPtr(timePtr(p.GetCreated())),
				"regionUsageRestrictions": llx.StringData(p.GetRegionUsageRestrictions()),
			})
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
		if len(results) < pageSize {
			break
		}
	}
	return out, nil
}

func (r *mqlMongodbatlas) orgUsers() ([]any, error) {
	oid, err := orgID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	client := atlasClient(r.MqlRuntime)
	ctx := context.Background()

	out := []any{}
	for page := 1; ; page++ {
		resp, _, err := client.MongoDBCloudUsersApi.ListOrganizationUsers(ctx, oid).ItemsPerPage(pageSize).PageNum(page).Execute()
		if err != nil {
			return nil, err
		}
		results := resp.GetResults()
		for i := range results {
			res, err := newMqlMongodbatlasOrgUser(r.MqlRuntime, results[i])
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
		if len(results) < pageSize {
			break
		}
	}
	return out, nil
}

func newMqlMongodbatlasOrgUser(runtime *plugin.Runtime, u admin.OrgUserResponse) (*mqlMongodbatlasOrgUser, error) {
	roles := u.GetRoles()
	res, err := CreateResource(runtime, "mongodbatlas.orgUser", map[string]*llx.RawData{
		"__id":                llx.StringData("mongodbatlas.orgUser/" + u.GetId()),
		"id":                  llx.StringData(u.GetId()),
		"username":            llx.StringData(u.GetUsername()),
		"orgMembershipStatus": llx.StringData(u.GetOrgMembershipStatus()),
		"orgRoles":            llx.ArrayData(strSlice(roles.GetOrgRoles()), types.String),
		"teamIds":             llx.ArrayData(strSlice(u.GetTeamIds()), types.String),
		"lastAuth":            llx.TimeDataPtr(timePtr(u.GetLastAuth())),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMongodbatlasOrgUser), nil
}

// teams resolves the member's team ids to the teams they belong to.
func (r *mqlMongodbatlasOrgUser) teams() ([]any, error) {
	oid, err := orgID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	client := atlasClient(r.MqlRuntime)
	ctx := context.Background()

	out := []any{}
	for _, raw := range r.TeamIds.Data {
		teamID, ok := raw.(string)
		if !ok || teamID == "" {
			continue
		}
		t, _, err := client.TeamsApi.GetTeamById(ctx, oid, teamID).Execute()
		if err != nil {
			return nil, err
		}
		res, err := newMqlMongodbatlasTeam(r.MqlRuntime, *t)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func (r *mqlMongodbatlas) teams() ([]any, error) {
	oid, err := orgID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	client := atlasClient(r.MqlRuntime)
	ctx := context.Background()

	out := []any{}
	for page := 1; ; page++ {
		resp, _, err := client.TeamsApi.ListOrganizationTeams(ctx, oid).ItemsPerPage(pageSize).PageNum(page).Execute()
		if err != nil {
			return nil, err
		}
		results := resp.GetResults()
		for i := range results {
			res, err := newMqlMongodbatlasTeam(r.MqlRuntime, results[i])
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
		if len(results) < pageSize {
			break
		}
	}
	return out, nil
}

func newMqlMongodbatlasTeam(runtime *plugin.Runtime, t admin.TeamResponse) (*mqlMongodbatlasTeam, error) {
	res, err := CreateResource(runtime, "mongodbatlas.team", map[string]*llx.RawData{
		"__id": llx.StringData("mongodbatlas.team/" + t.GetId()),
		"id":   llx.StringData(t.GetId()),
		"name": llx.StringData(t.GetName()),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlMongodbatlasTeam), nil
}

// users resolves the members of the team.
func (r *mqlMongodbatlasTeam) users() ([]any, error) {
	oid, err := orgID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	client := atlasClient(r.MqlRuntime)
	ctx := context.Background()

	out := []any{}
	for page := 1; ; page++ {
		resp, _, err := client.MongoDBCloudUsersApi.ListTeamUsers(ctx, oid, r.Id.Data).ItemsPerPage(pageSize).PageNum(page).Execute()
		if err != nil {
			return nil, err
		}
		results := resp.GetResults()
		for i := range results {
			res, err := newMqlMongodbatlasOrgUser(r.MqlRuntime, results[i])
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
		if len(results) < pageSize {
			break
		}
	}
	return out, nil
}

func (r *mqlMongodbatlas) apiKeys() ([]any, error) {
	oid, err := orgID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	client := atlasClient(r.MqlRuntime)
	ctx := context.Background()

	out := []any{}
	for page := 1; ; page++ {
		resp, _, err := client.ProgrammaticAPIKeysApi.ListApiKeys(ctx, oid).ItemsPerPage(pageSize).PageNum(page).Execute()
		if err != nil {
			return nil, err
		}
		results := resp.GetResults()
		for i := range results {
			k := results[i]
			roles := []any{}
			for _, role := range k.GetRoles() {
				roles = append(roles, map[string]any{
					"roleName": role.GetRoleName(),
					"groupId":  role.GetGroupId(),
					"orgId":    role.GetOrgId(),
				})
			}
			res, err := CreateResource(r.MqlRuntime, "mongodbatlas.apiKey", map[string]*llx.RawData{
				"__id":        llx.StringData("mongodbatlas.apiKey/" + k.GetId()),
				"id":          llx.StringData(k.GetId()),
				"publicKey":   llx.StringData(k.GetPublicKey()),
				"description": llx.StringData(k.GetDesc()),
				"roles":       llx.ArrayData(roles, types.Dict),
			})
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
		if len(results) < pageSize {
			break
		}
	}
	return out, nil
}

func (r *mqlMongodbatlas) serviceAccounts() ([]any, error) {
	oid, err := orgID(r.MqlRuntime)
	if err != nil {
		return nil, err
	}
	client := atlasClient(r.MqlRuntime)
	ctx := context.Background()

	out := []any{}
	for page := 1; ; page++ {
		resp, _, err := client.ServiceAccountsApi.ListServiceAccounts(ctx, oid).ItemsPerPage(pageSize).PageNum(page).Execute()
		if err != nil {
			return nil, err
		}
		results := resp.GetResults()
		for i := range results {
			sa := results[i]
			secrets := []any{}
			for _, s := range sa.GetSecrets() {
				secrets = append(secrets, map[string]any{
					"id":                s.GetId(),
					"createdAt":         dictTime(s.GetCreatedAt()),
					"expiresAt":         dictTime(s.GetExpiresAt()),
					"lastUsedAt":        dictTime(s.GetLastUsedAt()),
					"maskedSecretValue": s.GetMaskedSecretValue(),
				})
			}
			res, err := CreateResource(r.MqlRuntime, "mongodbatlas.serviceAccount", map[string]*llx.RawData{
				"__id":        llx.StringData("mongodbatlas.serviceAccount/" + sa.GetClientId()),
				"clientId":    llx.StringData(sa.GetClientId()),
				"name":        llx.StringData(sa.GetName()),
				"description": llx.StringData(sa.GetDescription()),
				"roles":       llx.ArrayData(strSlice(sa.GetRoles()), types.String),
				"createdAt":   llx.TimeDataPtr(timePtr(sa.GetCreatedAt())),
				"secrets":     llx.ArrayData(secrets, types.Dict),
			})
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
		if len(results) < pageSize {
			break
		}
	}
	return out, nil
}
