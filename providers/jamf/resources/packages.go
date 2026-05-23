// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/jamf/connection"
)

func (r *mqlJamf) packages() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.JamfConnection)
	client := conn.Client

	// GetPackages paginates through every page internally and returns the
	// full set of results, so a single call here is the complete inventory.
	inventory, err := client.GetPackages("id:asc", "")
	if err != nil {
		return nil, err
	}

	var res []interface{}
	for _, c := range inventory.Results {
		item, err := CreateResource(r.MqlRuntime, "jamf.package", map[string]*llx.RawData{
			"id":                   llx.StringData(c.ID),
			"name":                 llx.StringData(c.PackageName),
			"fileName":             llx.StringData(c.FileName),
			"osInstall":            llx.BoolDataPtr(c.OSInstall),
			"categoryId":           llx.StringData(c.CategoryID),
			"priority":             llx.IntData(c.Priority),
			"suppressUpdates":      llx.BoolDataPtr(c.SuppressUpdates),
			"suppressRegistration": llx.BoolDataPtr(c.SuppressRegistration),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, item)
	}

	return res, nil
}

func (c *mqlJamfPackage) id() (string, error) {
	return "jamf.package/" + c.Id.Data, nil
}
