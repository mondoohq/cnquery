// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/google-workspace/connection"
	directory "google.golang.org/api/admin/directory/v1"
)

func (g *mqlGoogleworkspace) orgUnits() ([]interface{}, error) {
	conn := g.MqlRuntime.Connection.(*connection.GoogleWorkspaceConnection)
	directoryService, err := directoryService(conn, directory.AdminDirectoryOrgunitReadonlyScope)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	orgUnits, err := directoryService.Orgunits.List(conn.CustomerID()).Do()
	if err != nil {
		return nil, err
	}

	for i := range orgUnits.OrganizationUnits {
		r, err := newMqlGoogleWorkspaceOrgUnit(g.MqlRuntime, orgUnits.OrganizationUnits[i])
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}

func newMqlGoogleWorkspaceOrgUnit(runtime *plugin.Runtime, entry *directory.OrgUnit) (interface{}, error) {
	return CreateResource(runtime, "googleworkspace.orgUnit", map[string]*llx.RawData{
		"id":          llx.StringData(entry.OrgUnitId),
		"name":        llx.StringData(entry.Name),
		"description": llx.StringData(entry.Description),
	})
}

func (g *mqlGoogleworkspaceOrgUnit) id() (string, error) {
	return "googleworkspace.orgUnit/" + g.Id.Data, g.Id.Error
}
