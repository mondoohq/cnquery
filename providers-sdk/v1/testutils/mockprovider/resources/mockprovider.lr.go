// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Code generated by resources. DO NOT EDIT.

package resources

import (
	"errors"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/types"
)

var resourceFactories map[string]plugin.ResourceFactory

func init() {
	resourceFactories = map[string]plugin.ResourceFactory {
		"muser": {
			// to override args, implement: initMuser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createMuser,
		},
		"mgroup": {
			// to override args, implement: initMgroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createMgroup,
		},
		"mos": {
			// to override args, implement: initMos(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createMos,
		},
		"customGroups": {
			// to override args, implement: initCustomGroups(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error)
			Create: createCustomGroups,
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
	"muser.name": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlMuser).GetName()).ToDataRes(types.String)
	},
	"muser.group": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlMuser).GetGroup()).ToDataRes(types.Resource("mgroup"))
	},
	"muser.nullgroup": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlMuser).GetNullgroup()).ToDataRes(types.Resource("mgroup"))
	},
	"muser.nullstring": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlMuser).GetNullstring()).ToDataRes(types.String)
	},
	"muser.groups": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlMuser).GetGroups()).ToDataRes(types.Array(types.Resource("mgroup")))
	},
	"muser.dict": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlMuser).GetDict()).ToDataRes(types.Dict)
	},
	"mgroup.name": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlMgroup).GetName()).ToDataRes(types.String)
	},
	"mos.groups": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlMos).GetGroups()).ToDataRes(types.Resource("customGroups"))
	},
	"customGroups.length": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCustomGroups).GetLength()).ToDataRes(types.Int)
	},
	"customGroups.list": func(r plugin.Resource) *plugin.DataRes {
		return (r.(*mqlCustomGroups).GetList()).ToDataRes(types.Array(types.Resource("mgroup")))
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
	"muser.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlMuser).__id, ok = v.Value.(string)
			return
		},
	"muser.name": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlMuser).Name, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"muser.group": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlMuser).Group, ok = plugin.RawToTValue[*mqlMgroup](v.Value, v.Error)
		return
	},
	"muser.nullgroup": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlMuser).Nullgroup, ok = plugin.RawToTValue[*mqlMgroup](v.Value, v.Error)
		return
	},
	"muser.nullstring": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlMuser).Nullstring, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"muser.groups": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlMuser).Groups, ok = plugin.RawToTValue[[]interface{}](v.Value, v.Error)
		return
	},
	"muser.dict": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlMuser).Dict, ok = plugin.RawToTValue[interface{}](v.Value, v.Error)
		return
	},
	"mgroup.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlMgroup).__id, ok = v.Value.(string)
			return
		},
	"mgroup.name": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlMgroup).Name, ok = plugin.RawToTValue[string](v.Value, v.Error)
		return
	},
	"mos.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlMos).__id, ok = v.Value.(string)
			return
		},
	"mos.groups": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlMos).Groups, ok = plugin.RawToTValue[*mqlCustomGroups](v.Value, v.Error)
		return
	},
	"customGroups.__id": func(r plugin.Resource, v *llx.RawData) (ok bool) {
			r.(*mqlCustomGroups).__id, ok = v.Value.(string)
			return
		},
	"customGroups.length": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCustomGroups).Length, ok = plugin.RawToTValue[int64](v.Value, v.Error)
		return
	},
	"customGroups.list": func(r plugin.Resource, v *llx.RawData) (ok bool) {
		r.(*mqlCustomGroups).List, ok = plugin.RawToTValue[[]interface{}](v.Value, v.Error)
		return
	},
}

func SetData(resource plugin.Resource, field string, val *llx.RawData) error {
	f, ok := setDataFields[resource.MqlName() + "." + field]
	if !ok {
		return errors.New("[mockprovider] cannot set '"+field+"' in resource '"+resource.MqlName()+"', field not found")
	}

	if ok := f(resource, val); !ok {
		return errors.New("[mockprovider] cannot set '"+field+"' in resource '"+resource.MqlName()+"', type does not match")
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

// mqlMuser for the muser resource
type mqlMuser struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlMuserInternal it will be used here
	Name plugin.TValue[string]
	Group plugin.TValue[*mqlMgroup]
	Nullgroup plugin.TValue[*mqlMgroup]
	Nullstring plugin.TValue[string]
	Groups plugin.TValue[[]interface{}]
	Dict plugin.TValue[interface{}]
}

// createMuser creates a new instance of this resource
func createMuser(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlMuser{
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
		args, err = runtime.ResourceFromRecording("muser", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlMuser) MqlName() string {
	return "muser"
}

func (c *mqlMuser) MqlID() string {
	return c.__id
}

func (c *mqlMuser) GetName() *plugin.TValue[string] {
	return &c.Name
}

func (c *mqlMuser) GetGroup() *plugin.TValue[*mqlMgroup] {
	return plugin.GetOrCompute[*mqlMgroup](&c.Group, func() (*mqlMgroup, error) {
		if c.MqlRuntime.HasRecording {
			d, err := c.MqlRuntime.FieldResourceFromRecording("muser", c.__id, "group")
			if err != nil {
				return nil, err
			}
			if d != nil {
				return d.Value.(*mqlMgroup), nil
			}
		}

		return c.group()
	})
}

func (c *mqlMuser) GetNullgroup() *plugin.TValue[*mqlMgroup] {
	return plugin.GetOrCompute[*mqlMgroup](&c.Nullgroup, func() (*mqlMgroup, error) {
		if c.MqlRuntime.HasRecording {
			d, err := c.MqlRuntime.FieldResourceFromRecording("muser", c.__id, "nullgroup")
			if err != nil {
				return nil, err
			}
			if d != nil {
				return d.Value.(*mqlMgroup), nil
			}
		}

		return c.nullgroup()
	})
}

func (c *mqlMuser) GetNullstring() *plugin.TValue[string] {
	return plugin.GetOrCompute[string](&c.Nullstring, func() (string, error) {
		return c.nullstring()
	})
}

func (c *mqlMuser) GetGroups() *plugin.TValue[[]interface{}] {
	return plugin.GetOrCompute[[]interface{}](&c.Groups, func() ([]interface{}, error) {
		if c.MqlRuntime.HasRecording {
			d, err := c.MqlRuntime.FieldResourceFromRecording("muser", c.__id, "groups")
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

func (c *mqlMuser) GetDict() *plugin.TValue[interface{}] {
	return plugin.GetOrCompute[interface{}](&c.Dict, func() (interface{}, error) {
		return c.dict()
	})
}

// mqlMgroup for the mgroup resource
type mqlMgroup struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlMgroupInternal it will be used here
	Name plugin.TValue[string]
}

// createMgroup creates a new instance of this resource
func createMgroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlMgroup{
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
		args, err = runtime.ResourceFromRecording("mgroup", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlMgroup) MqlName() string {
	return "mgroup"
}

func (c *mqlMgroup) MqlID() string {
	return c.__id
}

func (c *mqlMgroup) GetName() *plugin.TValue[string] {
	return &c.Name
}

// mqlMos for the mos resource
type mqlMos struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlMosInternal it will be used here
	Groups plugin.TValue[*mqlCustomGroups]
}

// createMos creates a new instance of this resource
func createMos(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlMos{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	// to override __id implement: id() (string, error)

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("mos", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlMos) MqlName() string {
	return "mos"
}

func (c *mqlMos) MqlID() string {
	return c.__id
}

func (c *mqlMos) GetGroups() *plugin.TValue[*mqlCustomGroups] {
	return plugin.GetOrCompute[*mqlCustomGroups](&c.Groups, func() (*mqlCustomGroups, error) {
		if c.MqlRuntime.HasRecording {
			d, err := c.MqlRuntime.FieldResourceFromRecording("mos", c.__id, "groups")
			if err != nil {
				return nil, err
			}
			if d != nil {
				return d.Value.(*mqlCustomGroups), nil
			}
		}

		return c.groups()
	})
}

// mqlCustomGroups for the customGroups resource
type mqlCustomGroups struct {
	MqlRuntime *plugin.Runtime
	__id string
	// optional: if you define mqlCustomGroupsInternal it will be used here
	Length plugin.TValue[int64]
	List plugin.TValue[[]interface{}]
}

// createCustomGroups creates a new instance of this resource
func createCustomGroups(runtime *plugin.Runtime, args map[string]*llx.RawData) (plugin.Resource, error) {
	res := &mqlCustomGroups{
		MqlRuntime: runtime,
	}

	err := SetAllData(res, args)
	if err != nil {
		return res, err
	}

	// to override __id implement: id() (string, error)

	if runtime.HasRecording {
		args, err = runtime.ResourceFromRecording("customGroups", res.__id)
		if err != nil || args == nil {
			return res, err
		}
		return res, SetAllData(res, args)
	}

	return res, nil
}

func (c *mqlCustomGroups) MqlName() string {
	return "customGroups"
}

func (c *mqlCustomGroups) MqlID() string {
	return c.__id
}

func (c *mqlCustomGroups) GetLength() *plugin.TValue[int64] {
	return plugin.GetOrCompute[int64](&c.Length, func() (int64, error) {
		return c.length()
	})
}

func (c *mqlCustomGroups) GetList() *plugin.TValue[[]interface{}] {
	return plugin.GetOrCompute[[]interface{}](&c.List, func() ([]interface{}, error) {
		if c.MqlRuntime.HasRecording {
			d, err := c.MqlRuntime.FieldResourceFromRecording("customGroups", c.__id, "list")
			if err != nil {
				return nil, err
			}
			if d != nil {
				return d.Value.([]interface{}), nil
			}
		}

		return c.list()
	})
}
