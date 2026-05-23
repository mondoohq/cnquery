// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/applicationcredentials"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/domains"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/groups"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/projects"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/roles"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/users"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// ---- openstack.project ----

type mqlOpenstackProjectInternal struct {
	cacheDomainID string
	cacheParentID string
}

func (r *mqlOpenstackProject) id() (string, error) {
	return "openstack.project/" + r.Id.Data, nil
}

func initOpenstackProject(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := stringArg(args, "id")
	if !ok || id == "" {
		return args, nil, nil
	}
	root, err := CreateResource(runtime, "openstack", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	list := root.(*mqlOpenstack).GetProjects()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		p := raw.(*mqlOpenstackProject)
		if p.Id.Data == id {
			return args, p, nil
		}
	}
	initSyntheticID("openstack.project", "id", args)
	return args, nil, nil
}

func (o *mqlOpenstack) projects() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.IdentityClient()
	if err != nil {
		return nil, err
	}
	pages, err := projects.List(client, projects.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := projects.ExtractProjects(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for _, p := range items {
		res, err := newMqlOpenstackProject(o.MqlRuntime, &p)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func newMqlOpenstackProject(runtime *plugin.Runtime, p *projects.Project) (*mqlOpenstackProject, error) {
	res, err := CreateResource(runtime, "openstack.project", map[string]*llx.RawData{
		"__id":        llx.StringData("openstack.project/" + p.ID),
		"id":          llx.StringData(p.ID),
		"name":        llx.StringData(p.Name),
		"description": llx.StringData(p.Description),
		"enabled":     llx.BoolData(p.Enabled),
		"isDomain":    llx.BoolData(p.IsDomain),
		"tags":        stringSliceData(p.Tags),
	})
	if err != nil {
		return nil, err
	}
	mqlProject := res.(*mqlOpenstackProject)
	mqlProject.cacheDomainID = p.DomainID
	mqlProject.cacheParentID = p.ParentID
	return mqlProject, nil
}

func (r *mqlOpenstackProject) domain() (*mqlOpenstackDomain, error) {
	if r.cacheDomainID == "" {
		r.Domain.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.domain", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheDomainID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackDomain), nil
}

func (r *mqlOpenstackProject) parent() (*mqlOpenstackProject, error) {
	if r.cacheParentID == "" {
		r.Parent.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.project", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheParentID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackProject), nil
}

// ---- openstack.user ----

type mqlOpenstackUserInternal struct {
	cacheDomainID         string
	cacheDefaultProjectID string
}

func (r *mqlOpenstackUser) id() (string, error) {
	return "openstack.user/" + r.Id.Data, nil
}

func initOpenstackUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := stringArg(args, "id")
	if !ok || id == "" {
		return args, nil, nil
	}
	root, err := CreateResource(runtime, "openstack", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	list := root.(*mqlOpenstack).GetUsers()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		u := raw.(*mqlOpenstackUser)
		if u.Id.Data == id {
			return args, u, nil
		}
	}
	initSyntheticID("openstack.user", "id", args)
	return args, nil, nil
}

func (o *mqlOpenstack) users() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.IdentityClient()
	if err != nil {
		return nil, err
	}
	pages, err := users.List(client, users.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := users.ExtractUsers(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for i := range items {
		mqlUser, err := newMqlOpenstackUser(o.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, mqlUser)
	}
	return out, nil
}

func newMqlOpenstackUser(runtime *plugin.Runtime, u *users.User) (*mqlOpenstackUser, error) {
	res, err := CreateResource(runtime, "openstack.user", map[string]*llx.RawData{
		"__id":                         llx.StringData("openstack.user/" + u.ID),
		"id":                           llx.StringData(u.ID),
		"name":                         llx.StringData(u.Name),
		"enabled":                      llx.BoolData(u.Enabled),
		"description":                  llx.StringData(u.Description),
		"passwordExpiresAt":            llx.TimeDataPtr(timePtr(u.PasswordExpiresAt)),
		"ignoreLockoutFailureAttempts": llx.BoolData(userOptionBool(u.Options, "ignore_lockout_failure_attempts")),
	})
	if err != nil {
		return nil, err
	}
	mqlUser := res.(*mqlOpenstackUser)
	mqlUser.cacheDomainID = u.DomainID
	mqlUser.cacheDefaultProjectID = u.DefaultProjectID
	return mqlUser, nil
}

func (r *mqlOpenstackUser) domain() (*mqlOpenstackDomain, error) {
	if r.cacheDomainID == "" {
		r.Domain.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.domain", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheDomainID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackDomain), nil
}

func (r *mqlOpenstackUser) defaultProject() (*mqlOpenstackProject, error) {
	if r.cacheDefaultProjectID == "" {
		r.DefaultProject.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.project", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheDefaultProjectID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackProject), nil
}

func (r *mqlOpenstackUser) roles() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.IdentityClient()
	if err != nil {
		return nil, err
	}
	effective := true
	pages, err := roles.ListAssignments(client, roles.ListAssignmentsOpts{
		UserID:    r.Id.Data,
		Effective: &effective,
	}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := roles.ExtractRoleAssignments(pages)
	if err != nil {
		return nil, err
	}
	return resolveAssignedRoles(r.MqlRuntime, items)
}

// ---- openstack.role ----

type mqlOpenstackRoleInternal struct {
	cacheDomainID string
}

func (r *mqlOpenstackRole) id() (string, error) {
	return "openstack.role/" + r.Id.Data, nil
}

func initOpenstackRole(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := stringArg(args, "id")
	if !ok || id == "" {
		return args, nil, nil
	}
	root, err := CreateResource(runtime, "openstack", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	list := root.(*mqlOpenstack).GetRoles()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		r := raw.(*mqlOpenstackRole)
		if r.Id.Data == id {
			return args, r, nil
		}
	}
	initSyntheticID("openstack.role", "id", args)
	return args, nil, nil
}

func (o *mqlOpenstack) roles() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.IdentityClient()
	if err != nil {
		return nil, err
	}
	pages, err := roles.List(client, roles.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := roles.ExtractRoles(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for i := range items {
		role := &items[i]
		res, err := CreateResource(o.MqlRuntime, "openstack.role", map[string]*llx.RawData{
			"__id":        llx.StringData("openstack.role/" + role.ID),
			"id":          llx.StringData(role.ID),
			"name":        llx.StringData(role.Name),
			"description": llx.StringData(role.Description),
		})
		if err != nil {
			return nil, err
		}
		mqlRole := res.(*mqlOpenstackRole)
		mqlRole.cacheDomainID = role.DomainID
		out = append(out, mqlRole)
	}
	return out, nil
}

func (r *mqlOpenstackRole) domain() (*mqlOpenstackDomain, error) {
	if r.cacheDomainID == "" {
		r.Domain.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.domain", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheDomainID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackDomain), nil
}

// ---- openstack.domain ----

func (r *mqlOpenstackDomain) id() (string, error) {
	return "openstack.domain/" + r.Id.Data, nil
}

func initOpenstackDomain(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := stringArg(args, "id")
	if !ok || id == "" {
		return args, nil, nil
	}
	root, err := CreateResource(runtime, "openstack", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	list := root.(*mqlOpenstack).GetDomains()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		d := raw.(*mqlOpenstackDomain)
		if d.Id.Data == id {
			return args, d, nil
		}
	}
	initSyntheticID("openstack.domain", "id", args)
	return args, nil, nil
}

func (o *mqlOpenstack) domains() ([]any, error) {
	c := conn(o.MqlRuntime)
	client, err := c.IdentityClient()
	if err != nil {
		return nil, err
	}
	pages, err := domains.List(client, domains.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := domains.ExtractDomains(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for _, d := range items {
		res, err := CreateResource(o.MqlRuntime, "openstack.domain", map[string]*llx.RawData{
			"__id":        llx.StringData("openstack.domain/" + d.ID),
			"id":          llx.StringData(d.ID),
			"name":        llx.StringData(d.Name),
			"description": llx.StringData(d.Description),
			"enabled":     llx.BoolData(d.Enabled),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// ---- openstack.group ----

type mqlOpenstackGroupInternal struct {
	cacheDomainID string
}

func (r *mqlOpenstackGroup) id() (string, error) {
	return "openstack.group/" + r.Id.Data, nil
}

func initOpenstackGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := stringArg(args, "id")
	if !ok || id == "" {
		return args, nil, nil
	}
	root, err := CreateResource(runtime, "openstack", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	list := root.(*mqlOpenstack).GetGroups()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		g := raw.(*mqlOpenstackGroup)
		if g.Id.Data == id {
			return args, g, nil
		}
	}
	initSyntheticID("openstack.group", "id", args)
	return args, nil, nil
}

func (o *mqlOpenstack) groups() ([]any, error) {
	return listKeystoneGroups(o.MqlRuntime, groups.ListOpts{})
}

func (r *mqlOpenstackDomain) groups() ([]any, error) {
	return listKeystoneGroups(r.MqlRuntime, groups.ListOpts{DomainID: r.Id.Data})
}

func listKeystoneGroups(runtime *plugin.Runtime, opts groups.ListOpts) ([]any, error) {
	c := conn(runtime)
	client, err := c.IdentityClient()
	if err != nil {
		return nil, err
	}
	pages, err := groups.List(client, opts).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := groups.ExtractGroups(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for i := range items {
		mqlGroup, err := newMqlOpenstackGroup(runtime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, mqlGroup)
	}
	return out, nil
}

func newMqlOpenstackGroup(runtime *plugin.Runtime, g *groups.Group) (*mqlOpenstackGroup, error) {
	res, err := CreateResource(runtime, "openstack.group", map[string]*llx.RawData{
		"__id":        llx.StringData("openstack.group/" + g.ID),
		"id":          llx.StringData(g.ID),
		"name":        llx.StringData(g.Name),
		"description": llx.StringData(g.Description),
	})
	if err != nil {
		return nil, err
	}
	mqlGroup := res.(*mqlOpenstackGroup)
	mqlGroup.cacheDomainID = g.DomainID
	return mqlGroup, nil
}

func (r *mqlOpenstackGroup) domain() (*mqlOpenstackDomain, error) {
	if r.cacheDomainID == "" {
		r.Domain.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.domain", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheDomainID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackDomain), nil
}

func (r *mqlOpenstackGroup) users() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.IdentityClient()
	if err != nil {
		return nil, err
	}
	pages, err := users.ListInGroup(client, r.Id.Data, users.ListOpts{}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := users.ExtractUsers(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for i := range items {
		mqlUser, err := newMqlOpenstackUser(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, mqlUser)
	}
	return out, nil
}

func (r *mqlOpenstackGroup) roles() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.IdentityClient()
	if err != nil {
		return nil, err
	}
	pages, err := roles.ListAssignments(client, roles.ListAssignmentsOpts{
		GroupID: r.Id.Data,
	}).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := roles.ExtractRoleAssignments(pages)
	if err != nil {
		return nil, err
	}
	return resolveAssignedRoles(r.MqlRuntime, items)
}

func (r *mqlOpenstackUser) groups() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.IdentityClient()
	if err != nil {
		return nil, err
	}
	pages, err := users.ListGroups(client, r.Id.Data).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := groups.ExtractGroups(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for i := range items {
		mqlGroup, err := newMqlOpenstackGroup(r.MqlRuntime, &items[i])
		if err != nil {
			return nil, err
		}
		out = append(out, mqlGroup)
	}
	return out, nil
}

// resolveAssignedRoles dedupes role assignments by role ID and, when the
// caller has already triggered openstack.roles, resolves each role from the
// cached list so the returned resources carry full data (name, domain).
// ListAssignments returns only role IDs — accepting that as-is leaves a
// blank `name` field on every assignment-derived role.
func resolveAssignedRoles(runtime *plugin.Runtime, items []roles.RoleAssignment) ([]any, error) {
	root, err := CreateResource(runtime, "openstack", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	list := root.(*mqlOpenstack).GetRoles()
	var byID map[string]*mqlOpenstackRole
	if list.Error == nil {
		byID = make(map[string]*mqlOpenstackRole, len(list.Data))
		for _, raw := range list.Data {
			role := raw.(*mqlOpenstackRole)
			byID[role.Id.Data] = role
		}
	}

	seen := make(map[string]struct{}, len(items))
	out := make([]any, 0, len(items))
	for _, a := range items {
		if a.Role.ID == "" {
			continue
		}
		if _, dup := seen[a.Role.ID]; dup {
			continue
		}
		seen[a.Role.ID] = struct{}{}
		if role, ok := byID[a.Role.ID]; ok {
			out = append(out, role)
			continue
		}
		res, err := NewResource(runtime, "openstack.role", map[string]*llx.RawData{
			"id": llx.StringData(a.Role.ID),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// ---- openstack.applicationCredential ----

type mqlOpenstackApplicationCredentialInternal struct {
	cacheUserID    string
	cacheProjectID string
}

func (r *mqlOpenstackApplicationCredential) id() (string, error) {
	return "openstack.applicationCredential/" + r.Id.Data, nil
}

func initOpenstackApplicationCredential(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	id, ok := stringArg(args, "id")
	if !ok || id == "" {
		return args, nil, nil
	}
	initSyntheticID("openstack.applicationCredential", "id", args)
	return args, nil, nil
}

func (r *mqlOpenstackUser) applicationCredentials() ([]any, error) {
	c := conn(r.MqlRuntime)
	client, err := c.IdentityClient()
	if err != nil {
		return nil, err
	}
	pages, err := applicationcredentials.List(client, r.Id.Data, nil).AllPages(ctx())
	if err != nil {
		if translateOpenstackError(err) == nil {
			return []any{}, nil
		}
		return nil, err
	}
	items, err := applicationcredentials.ExtractApplicationCredentials(pages)
	if err != nil {
		return nil, err
	}

	out := make([]any, 0, len(items))
	for _, ac := range items {
		roleNames := make([]string, 0, len(ac.Roles))
		for _, role := range ac.Roles {
			roleNames = append(roleNames, role.Name)
		}
		rules := make([]any, 0, len(ac.AccessRules))
		for _, rule := range ac.AccessRules {
			rules = append(rules, map[string]any{
				"id":      rule.ID,
				"service": rule.Service,
				"method":  rule.Method,
				"path":    rule.Path,
			})
		}
		res, err := CreateResource(r.MqlRuntime, "openstack.applicationCredential", map[string]*llx.RawData{
			"__id":         llx.StringData("openstack.applicationCredential/" + ac.ID),
			"id":           llx.StringData(ac.ID),
			"name":         llx.StringData(ac.Name),
			"description":  llx.StringData(ac.Description),
			"unrestricted": llx.BoolData(ac.Unrestricted),
			"userId":       llx.StringData(r.Id.Data),
			"projectId":    llx.StringData(ac.ProjectID),
			"roleNames":    stringSliceData(roleNames),
			"accessRules":  dictSliceData(rules),
			"expiresAt":    llx.TimeDataPtr(timePtr(ac.ExpiresAt)),
		})
		if err != nil {
			return nil, err
		}
		mqlAC := res.(*mqlOpenstackApplicationCredential)
		mqlAC.cacheUserID = r.Id.Data
		mqlAC.cacheProjectID = ac.ProjectID
		out = append(out, mqlAC)
	}
	return out, nil
}

func (r *mqlOpenstackApplicationCredential) user() (*mqlOpenstackUser, error) {
	if r.cacheUserID == "" {
		r.User.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.user", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheUserID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackUser), nil
}

func (r *mqlOpenstackApplicationCredential) project() (*mqlOpenstackProject, error) {
	if r.cacheProjectID == "" {
		r.Project.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openstack.project", map[string]*llx.RawData{
		"id": llx.StringData(r.cacheProjectID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenstackProject), nil
}
