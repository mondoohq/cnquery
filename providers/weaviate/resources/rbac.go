// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/weaviate/weaviate-go-client/v5/weaviate/rbac"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/types"
)

func (r *mqlWeaviateInstance) roles() ([]any, error) {
	conn := weaviateConnection(r.MqlRuntime)
	client, err := conn.Client()
	if err != nil {
		return nil, err
	}
	roles, err := client.Roles().AllGetter().Do(weaviateContext())
	if err != nil {
		// RBAC disabled or the credential cannot read roles: no visible roles.
		if isForbidden(err) {
			return []any{}, nil
		}
		return nil, err
	}

	serverID := conn.ServerID()
	list := []any{}
	for i := range roles {
		role := roles[i]
		mqlRole, err := newWeaviateRole(r.MqlRuntime, serverID, &role)
		if err != nil {
			return nil, err
		}
		list = append(list, mqlRole)
	}
	return list, nil
}

// permEntry is one flattened (action, collection) grant of a role.
type permEntry struct {
	action     string
	collection string
}

// flattenPermissions expands a role's per-domain permission slices into one
// entry per action, carrying the collection scope where the domain has one.
func flattenPermissions(role *rbac.Role) []permEntry {
	var out []permEntry
	add := func(actions []string, collection string) {
		for _, a := range actions {
			out = append(out, permEntry{action: a, collection: collection})
		}
	}
	for _, p := range role.Data {
		add(p.Actions, p.Collection)
	}
	for _, p := range role.Collections {
		add(p.Actions, p.Collection)
	}
	for _, p := range role.Nodes {
		add(p.Actions, p.Collection)
	}
	for _, p := range role.Tenants {
		// TenantsPermission carries no collection scope in this client version.
		add(p.Actions, "")
	}
	for _, p := range role.Roles {
		add(p.Actions, "")
	}
	for _, p := range role.Users {
		add(p.Actions, "")
	}
	for _, p := range role.Cluster {
		add(p.Actions, "")
	}
	for _, p := range role.Backups {
		add(p.Actions, "")
	}
	for _, p := range role.Alias {
		add(p.Actions, "")
	}
	for _, p := range role.Replicate {
		add(p.Actions, "")
	}
	for _, p := range role.Groups {
		add(p.Actions, "")
	}
	for _, p := range role.MCP {
		add(p.Actions, "")
	}
	return out
}

func (r *mqlWeaviateRole) permissions() ([]any, error) {
	if r.cacheRole == nil {
		return []any{}, nil
	}
	// Derive each permission's id from its (action, collection) so it is stable
	// regardless of the order the API returns permissions in. A repeated pair
	// gets a counter suffix so distinct entries never collide.
	seen := map[string]int{}
	list := []any{}
	for _, p := range flattenPermissions(r.cacheRole) {
		key := p.action + "/" + p.collection
		suffix := key
		if n := seen[key]; n > 0 {
			suffix = key + "#" + intToStr(int64(n))
		}
		seen[key]++
		res, err := CreateResource(r.MqlRuntime, "weaviate.role.permission", map[string]*llx.RawData{
			"__id":       llx.StringData(r.__id + "/perm/" + suffix),
			"action":     llx.StringData(p.action),
			"collection": llx.StringData(p.collection),
		})
		if err != nil {
			return nil, err
		}
		list = append(list, res)
	}
	return list, nil
}

func (r *mqlWeaviateRole) assignedUsers() ([]any, error) {
	conn := weaviateConnection(r.MqlRuntime)
	client, err := conn.Client()
	if err != nil {
		return nil, err
	}
	assignments, err := client.Roles().UserAssignmentGetter().WithRole(r.Name.Data).Do(weaviateContext())
	if err != nil {
		if isForbidden(err) {
			return []any{}, nil
		}
		return nil, err
	}
	// A user can be assigned under more than one user type (db, oidc), which
	// returns it once per type; report each distinct user id once.
	seen := map[string]struct{}{}
	list := []any{}
	for _, a := range assignments {
		if _, dup := seen[a.UserID]; dup {
			continue
		}
		seen[a.UserID] = struct{}{}
		list = append(list, a.UserID)
	}
	return list, nil
}

func (r *mqlWeaviateInstance) users() ([]any, error) {
	conn := weaviateConnection(r.MqlRuntime)
	client, err := conn.Client()
	if err != nil {
		return nil, err
	}
	infos, err := client.Users().DB().Lister().Do(weaviateContext())
	if err != nil {
		// User management disabled or the credential cannot read users.
		if isForbidden(err) {
			return []any{}, nil
		}
		return nil, err
	}

	serverID := conn.ServerID()
	list := []any{}
	for i := range infos {
		info := infos[i]
		res, err := CreateResource(r.MqlRuntime, "weaviate.user", map[string]*llx.RawData{
			"__id":     llx.StringData(userResourceID(serverID, info.UserID)),
			"userId":   llx.StringData(info.UserID),
			"userType": llx.StringData(string(info.UserType)),
			"active":   llx.BoolData(info.Active),
		})
		if err != nil {
			return nil, err
		}
		mqlUser := res.(*mqlWeaviateUser)
		mqlUser.cacheRoles = info.Roles
		list = append(list, mqlUser)
	}
	return list, nil
}

func (r *mqlWeaviateUser) roles() ([]any, error) {
	serverID := weaviateConnection(r.MqlRuntime).ServerID()
	list := []any{}
	for _, role := range r.cacheRoles {
		if role == nil {
			continue
		}
		mqlRole, err := newWeaviateRole(r.MqlRuntime, serverID, role)
		if err != nil {
			return nil, err
		}
		list = append(list, mqlRole)
	}
	return list, nil
}

var _ = types.String
