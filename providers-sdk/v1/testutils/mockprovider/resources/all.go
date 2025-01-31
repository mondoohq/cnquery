// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
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
