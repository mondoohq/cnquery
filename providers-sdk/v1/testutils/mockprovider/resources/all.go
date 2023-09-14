// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
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
