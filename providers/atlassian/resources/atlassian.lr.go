// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Code generated by resources. DO NOT EDIT.

package resources

import (
	"errors"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/types"
)

var resourceFactories map[string]plugin.ResourceFactory

func init() {
	resourceFactories = map[string]plugin.ResourceFactory {
		"atlassian": {
			// to override args, implement: initAtlassian(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createAtlassian,
		},
		"atlassian.admin": {
			// to override args, implement: initAtlassianAdmin(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createAtlassianAdmin,
		},
		"atlassian.admin.organization": {
			// to override args, implement: initAtlassianAdminOrganization(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createAtlassianAdminOrganization,
		},
		"atlassian.admin.organization.user": {
			// to override args, implement: initAtlassianAdminOrganizationUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createAtlassianAdminOrganizationUser,
		},
		"atlassian.jira": {
			// to override args, implement: initAtlassianJira(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createAtlassianJira,
		},
		"atlassian.jira.user": {
			// to override args, implement: initAtlassianJiraUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createAtlassianJiraUser,
		},
		"atlassian.jira.user.group": {
			// to override args, implement: initAtlassianJiraUserGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createAtlassianJiraUserGroup,
		},
		"atlassian.confluence": {
			// to override args, implement: initAtlassianConfluence(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createAtlassianConfluence,
		},
		"atlassian.confluence.user": {
			// to override args, implement: initAtlassianConfluenceUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createAtlassianConfluenceUser,
		},
	}
}

// NewResource is used by the runtime of this plugin to create new resources.
// Its arguments may be provided by users. This function is generally not
// used by initializing resources from recordings or from lists.
func NewResource(runtime *plugin.Runtime, name string, args map[string]*llx.RawData) (plugin.Resource, error) {
	f, ok := resourceFactories[name]
	if !ok {
		return nil, errors.New("cannot find resource " + name + " in this provider")
	}

	if f.Init != nil {
		cargs, res, err := f.Init(runtime, args)
		if err != nil {
			return res, err
		}

		if res != nil {
			id := name+"\x00"+res.MqlID()
			if x, ok := runtime.Resources.Get(id); ok {
				return x, nil
			}
			runtime.Resources.Set(id, res)
			return res, nil
		}

		args = cargs
	}

	res, err := f.Create(runtime, args)
	if err != nil {
		return nil, err
	}

	id := name+"\x00"+res.MqlID()
	if x, ok := runtime.Resources.Get(id); ok {
		return x, nil
	}

	runtime.Resources.Set(id, res)
	return res, nil
}

// CreateResource is used by the runtime of this plugin to create resources.
// Its arguments must be complete and pre-processed. This method is used
// for initializing resources from recordings or from lists.
func CreateResource(runtime *plugin.Runtime, name string, args map[string]*llx.RawData) (plugin.Resource, error) {
	f, ok := resourceFactories[name]
	if !ok {
		return nil, errors.New("cannot find resource " + name + " in this provider")
	}

	res, err := f.Create(runtime, args)
	if err != nil {
		return nil, err
	}

	id := name+"\x00"+res.MqlID()
	if x, ok := runtime.Resources.Get(id); ok {
		return x, nil
	}

	runtime.Resources.Set(id, res)
	return res, nil
}

var getDataFields = map[string]func(r plugin.Resource) *plugin.DataRes{
	"atlassian.admin.organizations": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianAdmin).GetOrganizations()).ToDataRes(types.Array(types.Resource("atlassian.admin.organization")))
	},
	"atlassian.admin.organization.id": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianAdminOrganization).GetId()).ToDataRes(types.String)
	},
	"atlassian.admin.organization.type": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianAdminOrganization).GetType()).ToDataRes(types.String)
	},
	"atlassian.admin.organization.users": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianAdminOrganization).GetUsers()).ToDataRes(types.Array(types.Resource("atlassian.admin.organization.user")))
	},
	"atlassian.admin.organization.user.id": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianAdminOrganizationUser).GetId()).ToDataRes(types.String)
	},
	"atlassian.admin.organization.user.type": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianAdminOrganizationUser).GetType()).ToDataRes(types.String)
	},
	"atlassian.admin.organization.user.status": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianAdminOrganizationUser).GetStatus()).ToDataRes(types.String)
	},
	"atlassian.admin.organization.user.name": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianAdminOrganizationUser).GetName()).ToDataRes(types.String)
	},
	"atlassian.admin.organization.user.picture": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianAdminOrganizationUser).GetPicture()).ToDataRes(types.String)
	},
	"atlassian.admin.organization.user.email": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianAdminOrganizationUser).GetEmail()).ToDataRes(types.String)
	},
	"atlassian.admin.organization.user.accessBillable": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianAdminOrganizationUser).GetAccessBillable()).ToDataRes(types.Bool)
	},
	"atlassian.admin.organization.user.lastActive": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianAdminOrganizationUser).GetLastActive()).ToDataRes(types.String)
	},
	"atlassian.jira.users": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianJira).GetUsers()).ToDataRes(types.Array(types.Resource("atlassian.jira.user")))
	},
	"atlassian.jira.user.id": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianJiraUser).GetId()).ToDataRes(types.String)
	},
	"atlassian.jira.user.name": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianJiraUser).GetName()).ToDataRes(types.String)
	},
	"atlassian.jira.user.type": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianJiraUser).GetType()).ToDataRes(types.String)
	},
	"atlassian.jira.user.picture": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianJiraUser).GetPicture()).ToDataRes(types.String)
	},
	"atlassian.jira.user.groups": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianJiraUser).GetGroups()).ToDataRes(types.Array(types.Resource("atlassian.jira.user.group")))
	},
	"atlassian.jira.user.group.id": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianJiraUserGroup).GetId()).ToDataRes(types.String)
	},
	"atlassian.confluence.users": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianConfluence).GetUsers()).ToDataRes(types.Array(types.Resource("atlassian.confluence.user")))
	},
	"atlassian.confluence.user.id": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianConfluenceUser).GetId()).ToDataRes(types.String)
	},
	"atlassian.confluence.user.name": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianConfluenceUser).GetName()).ToDataRes(types.String)
	},
	"atlassian.confluence.user.type": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlAtlassianConfluenceUser).GetType()).ToDataRes(types.String)
	},
}

func GetData(resource plugin.Resource, field string, args map[string]*llx.RawData) *plugin.DataRes {
	f, ok := getDataFields[resource.MqlName()+"."+field]
	if !ok {
		return &plugin.DataRes{Error: "cannot find '" + field + "' in resource '" + resource.MqlName() + "'"}
	}

	return f(resource)
}

var setDataFields = map[string]func(r plugin.Resource, v *llx.RawData) bool {
	"atlassian.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlAtlassian).__id, ok = v.Value.(string)
			return
		},
	"atlassian.admin.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlAtlassianAdmin).__id, ok = v.Value.(string)
			return
		},
	"atlassian.admin.organizations": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianAdmin).Organizations, ok = plugin.RawToTValue[[]interface{}](v.Value, v.Error)
		return
	},
	"atlassian.admin.organization.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlAtlassianAdminOrganization).__id, ok = v.Value.(string)
			return
		},
	"atlassian.admin.organization.id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianAdminOrganization).Id, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"atlassian.admin.organization.type": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianAdminOrganization).Type, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"atlassian.admin.organization.users": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianAdminOrganization).Users, ok = plugin.RawToTValue[[]interface{}](v.Value, v.Error)
		return
	},
	"atlassian.admin.organization.user.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlAtlassianAdminOrganizationUser).__id, ok = v.Value.(string)
			return
		},
	"atlassian.admin.organization.user.id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianAdminOrganizationUser).Id, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"atlassian.admin.organization.user.type": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianAdminOrganizationUser).Type, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"atlassian.admin.organization.user.status": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianAdminOrganizationUser).Status, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"atlassian.admin.organization.user.name": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianAdminOrganizationUser).Name, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"atlassian.admin.organization.user.picture": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianAdminOrganizationUser).Picture, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"atlassian.admin.organization.user.email": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianAdminOrganizationUser).Email, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"atlassian.admin.organization.user.accessBillable": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianAdminOrganizationUser).AccessBillable, ok = plugin.RawToTValue[bool](v.Value, v.Error)
		return
	},
	"atlassian.admin.organization.user.lastActive": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianAdminOrganizationUser).LastActive, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"atlassian.jira.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlAtlassianJira).__id, ok = v.Value.(string)
			return
		},
	"atlassian.jira.users": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianJira).Users, ok = plugin.RawToTValue[[]interface{}](v.Value, v.Error)
		return
	},
	"atlassian.jira.user.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlAtlassianJiraUser).__id, ok = v.Value.(string)
			return
		},
	"atlassian.jira.user.id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianJiraUser).Id, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"atlassian.jira.user.name": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianJiraUser).Name, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"atlassian.jira.user.type": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianJiraUser).Type, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"atlassian.jira.user.picture": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianJiraUser).Picture, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"atlassian.jira.user.groups": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianJiraUser).Groups, ok = plugin.RawToTValue[[]interface{}](v.Value, v.Error)
		return
	},
	"atlassian.jira.user.group.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlAtlassianJiraUserGroup).__id, ok = v.Value.(string)
			return
		},
	"atlassian.jira.user.group.id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianJiraUserGroup).Id, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"atlassian.confluence.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlAtlassianConfluence).__id, ok = v.Value.(string)
			return
		},
	"atlassian.confluence.users": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianConfluence).Users, ok = plugin.RawToTValue[[]interface{}](v.Value, v.Error)
		return
	},
	"atlassian.confluence.user.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlAtlassianConfluenceUser).__id, ok = v.Value.(string)
			return
		},
	"atlassian.confluence.user.id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianConfluenceUser).Id, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"atlassian.confluence.user.name": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianConfluenceUser).Name, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"atlassian.confluence.user.type": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlAtlassianConfluenceUser).Type, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
}

func SetData(resource plugin.Resource, field string, val *llx.RawData) error {
	f, ok := setDataFields[resource.MqlName() + "." + field]
	if !ok {
		return errors.New("[atlassian] cannot set '"+field+"' in resource '"+resource.MqlName()+"', field not found")
	}

	if ok := f(resource, val); !ok {
		return errors.New("[atlassian] cannot set '"+field+"' in resource '"+resource.MqlName()+"', type does not match")
	}
	return nil
}

func SetAllData(resource plugin.Resource, args map[string]*llx.RawData) error {
	var err error
	for k, v := range args {
		if err = SetData(resource, k, v); err != nil {
			return err
		}
	}
	return nil
}

// mqlAtlassian for the atlassian resource
type mqlAtlassian struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlAtlassianInternal it will be used here
}

// createAtlassian creates a new instance of this resource
func createAtlassian(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlAtlassian{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	if res.__id == "" {
	res.__id, err = res.id()
		if err != nil {
			return nil, err
		}
	}

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("atlassian", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlAtlassian) MqlName() string {
	return "atlassian"
}

func (c *mqlAtlassian) MqlID() string {
	return c.__id
}

// mqlAtlassianAdmin for the atlassian.admin resource
type mqlAtlassianAdmin struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlAtlassianAdminInternal it will be used here
	Organizations plugin.TValue[[]interface{}]
}

// createAtlassianAdmin creates a new instance of this resource
func createAtlassianAdmin(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlAtlassianAdmin{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	if res.__id == "" {
	res.__id, err = res.id()
		if err != nil {
			return nil, err
		}
	}

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("atlassian.admin", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlAtlassianAdmin) MqlName() string {
	return "atlassian.admin"
}

func (c *mqlAtlassianAdmin) MqlID() string {
	return c.__id
}

func (c *mqlAtlassianAdmin) GetOrganizations() *plugin.TValue[[]interface{}] {
	return plugin.GetOrCompute[[]interface{}](&c.Organizations, func() ([]interface{}, error) {
		if c.MqlRuntime.HasRecording {
			d, err := c.MqlRuntime.FieldResourceFromRecording("atlassian.admin", c.__id, "organizations")
			if err != nil {
				return nil, err
			}
			if d != nil {
				return d.Value.([]interface{}), nil
			}
		}

		return c.organizations()
	})
}

// mqlAtlassianAdminOrganization for the atlassian.admin.organization resource
type mqlAtlassianAdminOrganization struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlAtlassianAdminOrganizationInternal it will be used here
	Id plugin.TValue[string]
	Type plugin.TValue[string]
	Users plugin.TValue[[]interface{}]
}

// createAtlassianAdminOrganization creates a new instance of this resource
func createAtlassianAdminOrganization(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlAtlassianAdminOrganization{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	// to override __id implement: id() (string, error)

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("atlassian.admin.organization", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlAtlassianAdminOrganization) MqlName() string {
	return "atlassian.admin.organization"
}

func (c *mqlAtlassianAdminOrganization) MqlID() string {
	return c.__id
}

func (c *mqlAtlassianAdminOrganization) GetId() *plugin.TValue[string] {
	return &c.Id
}

func (c *mqlAtlassianAdminOrganization) GetType() *plugin.TValue[string] {
	return &c.Type
}

func (c *mqlAtlassianAdminOrganization) GetUsers() *plugin.TValue[[]interface{}] {
	return plugin.GetOrCompute[[]interface{}](&c.Users, func() ([]interface{}, error) {
		if c.MqlRuntime.HasRecording {
			d, err := c.MqlRuntime.FieldResourceFromRecording("atlassian.admin.organization", c.__id, "users")
			if err != nil {
				return nil, err
			}
			if d != nil {
				return d.Value.([]interface{}), nil
			}
		}

		return c.users()
	})
}

// mqlAtlassianAdminOrganizationUser for the atlassian.admin.organization.user resource
type mqlAtlassianAdminOrganizationUser struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlAtlassianAdminOrganizationUserInternal it will be used here
	Id plugin.TValue[string]
	Type plugin.TValue[string]
	Status plugin.TValue[string]
	Name plugin.TValue[string]
	Picture plugin.TValue[string]
	Email plugin.TValue[string]
	AccessBillable plugin.TValue[bool]
	LastActive plugin.TValue[string]
}

// createAtlassianAdminOrganizationUser creates a new instance of this resource
func createAtlassianAdminOrganizationUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlAtlassianAdminOrganizationUser{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	// to override __id implement: id() (string, error)

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("atlassian.admin.organization.user", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlAtlassianAdminOrganizationUser) MqlName() string {
	return "atlassian.admin.organization.user"
}

func (c *mqlAtlassianAdminOrganizationUser) MqlID() string {
	return c.__id
}

func (c *mqlAtlassianAdminOrganizationUser) GetId() *plugin.TValue[string] {
	return &c.Id
}

func (c *mqlAtlassianAdminOrganizationUser) GetType() *plugin.TValue[string] {
	return &c.Type
}

func (c *mqlAtlassianAdminOrganizationUser) GetStatus() *plugin.TValue[string] {
	return &c.Status
}

func (c *mqlAtlassianAdminOrganizationUser) GetName() *plugin.TValue[string] {
	return &c.Name
}

func (c *mqlAtlassianAdminOrganizationUser) GetPicture() *plugin.TValue[string] {
	return &c.Picture
}

func (c *mqlAtlassianAdminOrganizationUser) GetEmail() *plugin.TValue[string] {
	return &c.Email
}

func (c *mqlAtlassianAdminOrganizationUser) GetAccessBillable() *plugin.TValue[bool] {
	return &c.AccessBillable
}

func (c *mqlAtlassianAdminOrganizationUser) GetLastActive() *plugin.TValue[string] {
	return &c.LastActive
}

// mqlAtlassianJira for the atlassian.jira resource
type mqlAtlassianJira struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlAtlassianJiraInternal it will be used here
	Users plugin.TValue[[]interface{}]
}

// createAtlassianJira creates a new instance of this resource
func createAtlassianJira(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlAtlassianJira{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	if res.__id == "" {
	res.__id, err = res.id()
		if err != nil {
			return nil, err
		}
	}

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("atlassian.jira", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlAtlassianJira) MqlName() string {
	return "atlassian.jira"
}

func (c *mqlAtlassianJira) MqlID() string {
	return c.__id
}

func (c *mqlAtlassianJira) GetUsers() *plugin.TValue[[]interface{}] {
	return plugin.GetOrCompute[[]interface{}](&c.Users, func() ([]interface{}, error) {
		if c.MqlRuntime.HasRecording {
			d, err := c.MqlRuntime.FieldResourceFromRecording("atlassian.jira", c.__id, "users")
			if err != nil {
				return nil, err
			}
			if d != nil {
				return d.Value.([]interface{}), nil
			}
		}

		return c.users()
	})
}

// mqlAtlassianJiraUser for the atlassian.jira.user resource
type mqlAtlassianJiraUser struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlAtlassianJiraUserInternal it will be used here
	Id plugin.TValue[string]
	Name plugin.TValue[string]
	Type plugin.TValue[string]
	Picture plugin.TValue[string]
	Groups plugin.TValue[[]interface{}]
}

// createAtlassianJiraUser creates a new instance of this resource
func createAtlassianJiraUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlAtlassianJiraUser{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	if res.__id == "" {
	res.__id, err = res.id()
		if err != nil {
			return nil, err
		}
	}

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("atlassian.jira.user", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlAtlassianJiraUser) MqlName() string {
	return "atlassian.jira.user"
}

func (c *mqlAtlassianJiraUser) MqlID() string {
	return c.__id
}

func (c *mqlAtlassianJiraUser) GetId() *plugin.TValue[string] {
	return &c.Id
}

func (c *mqlAtlassianJiraUser) GetName() *plugin.TValue[string] {
	return &c.Name
}

func (c *mqlAtlassianJiraUser) GetType() *plugin.TValue[string] {
	return &c.Type
}

func (c *mqlAtlassianJiraUser) GetPicture() *plugin.TValue[string] {
	return &c.Picture
}

func (c *mqlAtlassianJiraUser) GetGroups() *plugin.TValue[[]interface{}] {
	return plugin.GetOrCompute[[]interface{}](&c.Groups, func() ([]interface{}, error) {
		if c.MqlRuntime.HasRecording {
			d, err := c.MqlRuntime.FieldResourceFromRecording("atlassian.jira.user", c.__id, "groups")
			if err != nil {
				return nil, err
			}
			if d != nil {
				return d.Value.([]interface{}), nil
			}
		}

		return c.groups()
	})
}

// mqlAtlassianJiraUserGroup for the atlassian.jira.user.group resource
type mqlAtlassianJiraUserGroup struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlAtlassianJiraUserGroupInternal it will be used here
	Id plugin.TValue[string]
}

// createAtlassianJiraUserGroup creates a new instance of this resource
func createAtlassianJiraUserGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlAtlassianJiraUserGroup{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	if res.__id == "" {
	res.__id, err = res.id()
		if err != nil {
			return nil, err
		}
	}

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("atlassian.jira.user.group", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlAtlassianJiraUserGroup) MqlName() string {
	return "atlassian.jira.user.group"
}

func (c *mqlAtlassianJiraUserGroup) MqlID() string {
	return c.__id
}

func (c *mqlAtlassianJiraUserGroup) GetId() *plugin.TValue[string] {
	return &c.Id
}

// mqlAtlassianConfluence for the atlassian.confluence resource
type mqlAtlassianConfluence struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlAtlassianConfluenceInternal it will be used here
	Users plugin.TValue[[]interface{}]
}

// createAtlassianConfluence creates a new instance of this resource
func createAtlassianConfluence(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlAtlassianConfluence{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	if res.__id == "" {
	res.__id, err = res.id()
		if err != nil {
			return nil, err
		}
	}

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("atlassian.confluence", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlAtlassianConfluence) MqlName() string {
	return "atlassian.confluence"
}

func (c *mqlAtlassianConfluence) MqlID() string {
	return c.__id
}

func (c *mqlAtlassianConfluence) GetUsers() *plugin.TValue[[]interface{}] {
	return plugin.GetOrCompute[[]interface{}](&c.Users, func() ([]interface{}, error) {
		if c.MqlRuntime.HasRecording {
			d, err := c.MqlRuntime.FieldResourceFromRecording("atlassian.confluence", c.__id, "users")
			if err != nil {
				return nil, err
			}
			if d != nil {
				return d.Value.([]interface{}), nil
			}
		}

		return c.users()
	})
}

// mqlAtlassianConfluenceUser for the atlassian.confluence.user resource
type mqlAtlassianConfluenceUser struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlAtlassianConfluenceUserInternal it will be used here
	Id plugin.TValue[string]
	Name plugin.TValue[string]
	Type plugin.TValue[string]
}

// createAtlassianConfluenceUser creates a new instance of this resource
func createAtlassianConfluenceUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlAtlassianConfluenceUser{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	if res.__id == "" {
	res.__id, err = res.id()
		if err != nil {
			return nil, err
		}
	}

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("atlassian.confluence.user", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlAtlassianConfluenceUser) MqlName() string {
	return "atlassian.confluence.user"
}

func (c *mqlAtlassianConfluenceUser) MqlID() string {
	return c.__id
}

func (c *mqlAtlassianConfluenceUser) GetId() *plugin.TValue[string] {
	return &c.Id
}

func (c *mqlAtlassianConfluenceUser) GetName() *plugin.TValue[string] {
	return &c.Name
}

func (c *mqlAtlassianConfluenceUser) GetType() *plugin.TValue[string] {
	return &c.Type
}
