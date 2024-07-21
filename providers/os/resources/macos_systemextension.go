// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strings"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/types"
)

func (m *mqlMacos) systemExtensions() ([]interface{}, error) {
	conn := m.MqlRuntime.Connection.(shared.Connection)

	f, err := conn.FileSystem().Open("/Library/SystemExtensions/db.plist")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	systemExtensionDb, err := Decode(f)
	if err != nil {
		return nil, err
	}

	extensions := systemExtensionDb["extensions"].([]interface{})
	extensionPolicies := systemExtensionDb["extensionPolicies"].([]interface{})

	list := []interface{}{}
	for i := range extensions {
		ex, err := newMacosSystemExtension(m.MqlRuntime, extensions[i].(map[string]interface{}), extensionPolicies)
		if err != nil {
			return nil, err
		}
		list = append(list, ex)
	}

	return list, nil
}

func newMacosSystemExtension(runtime *plugin.Runtime, extension plistData, extensionPolicies []interface{}) (*mqlMacosSystemExtension, error) {
	uuid := extension.GetString("uniqueID")
	identifier := extension.GetString("identifier")
	teamID := extension.GetString("teamID")
	isMdmManaged := false
	for i := range extensionPolicies {
		policy, ok := extensionPolicies[i].(map[string]interface{})
		if !ok {
			continue
		}
		plistPolicy := plistData(policy)

		// check if the team id is in allowedTeamIDs list
		allowedTeams := plistPolicy.GetPlistData("allowedTeamIDs")
		for k := range allowedTeams {
			list := allowedTeams[k].([]interface{})
			for j := range list {
				if list[j].(string) == teamID {
					isMdmManaged = true
					break
				}
			}
		}

		// if it is not in the team id list, check allowedExtensions list
		allowedExtensions := plistPolicy.GetPlistData("allowedExtensions")
		for k := range allowedExtensions {
			list := allowedExtensions[k].([]interface{})
			for j := range list {
				if list[j].(string) == identifier {
					isMdmManaged = true
					break
				}
			}
		}
	}

	pkg, err := CreateResource(runtime, "macos.systemExtension", map[string]*llx.RawData{
		"__id":       llx.StringData(uuid),
		"identifier": llx.StringData(identifier),
		"uuid":       llx.StringData(uuid),
		"version":    llx.StringData(extension.GetString("bundleVersion", "CFBundleShortVersionString")),
		"categories": llx.ArrayData(convert.SliceAnyToInterface(extension.GetList("categories")), types.String),
		"state":      llx.StringData(extension.GetString("state")),
		"teamID":     llx.StringData(teamID),
		"bundlePath": llx.StringData(extension.GetString("container", "bundlePath")),
		"mdmManaged": llx.BoolData(isMdmManaged),
	})
	if err != nil {
		return nil, err
	}

	s := pkg.(*mqlMacosSystemExtension)
	return s, nil
}

func (m *mqlMacosSystemExtension) enabled() (bool, error) {
	state := m.GetState()
	return strings.Contains(state.Data, "enabled"), nil
}

func (m *mqlMacosSystemExtension) active() (bool, error) {
	state := m.GetState()
	return strings.Contains(state.Data, "activated"), nil
}
