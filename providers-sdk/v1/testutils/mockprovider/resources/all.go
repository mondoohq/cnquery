// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
)

func (c *mqlMuser) id() (string, error) {
	return c.Name.Data, nil
}

func (c *mqlMuser) group() (*mqlMgroup, error) {
	o, err := CreateResource(c.MqlRuntime, "mgroup", map[string]*llx.RawData{
		"name": llx.StringData("group one"),
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlMgroup), nil
}

func (c *mqlMuser) nullgroup() (*mqlMgroup, error) {
	c.Nullgroup.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (c *mqlMuser) nullstring() (string, error) {
	c.Nullstring.State = plugin.StateIsSet | plugin.StateIsNull
	return "", nil
}

func (c *mqlMuser) groups() ([]interface{}, error) {
	one, err := CreateResource(c.MqlRuntime, "mgroup", map[string]*llx.RawData{
		"name": llx.StringData("group one"),
	})
	if err != nil {
		return nil, err
	}

	return []interface{}{
		one, nil,
	}, nil
}

func (c *mqlMgroup) id() (string, error) {
	return c.Name.Data, nil
}

func (c *mqlMuser) dict() (any, error) {
	return map[string]any{
		"listInt": []any{int64(1), int64(2), int64(3)},
		"string":  "hello world",
		"string2": "👋",
	}, nil
}

// This is an example of how we can override builtin functions today, this will have to change to provide
// a better mechanism to do so but for now, this pattern is adopted in multiple providers

// The example overrides the `length` builtin function by creating a custom list resource which
// essentially defers the loading of the actual "groups" (for this example) and provides a new function
// `length` that returns the number of "groups" but in a more performant way.

// groups() just initializes the custom list resource
func (c *mqlMos) groups() (*mqlCustomGroups, error) {
	mqlResource, err := CreateResource(c.MqlRuntime, "customGroups", map[string]*llx.RawData{})
	return mqlResource.(*mqlCustomGroups), err
}

// list() is where we actually load the real resources, which could be slow in big environments
func (c *mqlCustomGroups) list() ([]interface{}, error) {
	res := []interface{}{}
	for i := 0; i < 7; i++ {
		group, err := CreateResource(c.MqlRuntime, "mgroup", map[string]*llx.RawData{
			"name": llx.StringData(fmt.Sprintf("group%d", i+1)),
		})
		if err != nil {
			return res, err
		}
		res = append(res, group)
	}
	return res, nil
}

// length() overrides the builtin function, this should be a more performant way to count
// the "groups"
//
// NOTE this length here is different from the builtin one just for testing
func (c *mqlCustomGroups) length() (int64, error) {
	// use `c.MqlRuntime.Connection` to get the provider connection
	// make performant API call to count resources
	return 5, nil
}
