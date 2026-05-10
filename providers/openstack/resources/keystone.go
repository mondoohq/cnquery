// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/domains"
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
	if list.Error == nil {
		for _, raw := range list.Data {
			p := raw.(*mqlOpenstackProject)
			if p.Id.Data == id {
				return args, p, nil
			}
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
	if list.Error == nil {
		for _, raw := range list.Data {
			u := raw.(*mqlOpenstackUser)
			if u.Id.Data == id {
				return args, u, nil
			}
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
		u := &items[i]
		res, err := CreateResource(o.MqlRuntime, "openstack.user", map[string]*llx.RawData{
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
		out = append(out, mqlUser)
	}
	return out, nil
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
		res, err := NewResource(r.MqlRuntime, "openstack.role", map[string]*llx.RawData{
			"id":   llx.StringData(a.Role.ID),
			"name": llx.StringData(a.Role.Name),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
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
	if list.Error == nil {
		for _, raw := range list.Data {
			r := raw.(*mqlOpenstackRole)
			if r.Id.Data == id {
				return args, r, nil
			}
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
	if list.Error == nil {
		for _, raw := range list.Data {
			d := raw.(*mqlOpenstackDomain)
			if d.Id.Data == id {
				return args, d, nil
			}
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
