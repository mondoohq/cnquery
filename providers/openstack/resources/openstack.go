// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

func (r *mqlOpenstack) id() (string, error) {
	c := conn(r.MqlRuntime)
	return "openstack/" + c.ProjectID(), nil
}

// initOpenstack populates the top-level resource's scalar fields from the
// connection. Without this, MQL tries to use the explicit "create" path and
// the runtime fails because no field setter is registered for these inputs.
func initOpenstack(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	c := conn(runtime)
	if _, ok := args["authUrl"]; !ok {
		args["authUrl"] = llx.StringData(c.AuthURL())
	}
	if _, ok := args["projectId"]; !ok {
		args["projectId"] = llx.StringData(c.ProjectID())
	}
	if _, ok := args["region"]; !ok {
		args["region"] = llx.StringData(c.Region())
	}
	return args, nil, nil
}
