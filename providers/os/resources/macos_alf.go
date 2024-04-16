// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/types"
)

func initMacosAlf(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	conn := runtime.Connection.(shared.Connection)

	if args == nil {
		args = map[string]*llx.RawData{}
	}

	f, err := conn.FileSystem().Open("/Library/Preferences/com.apple.alf.plist")
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	alfConfig, err := Decode(f)
	if err != nil {
		return nil, nil, err
	}

	explicitAuthsRaw := alfConfig["explicitauths"].([]interface{})
	explicitAuths := []interface{}{}
	for i := range explicitAuthsRaw {
		entry := explicitAuthsRaw[i].(map[string]interface{})
		explicitAuths = append(explicitAuths, entry["id"])
	}

	args["allowDownloadSignedEnabled"] = llx.IntData(int64(alfConfig["allowdownloadsignedenabled"].(float64)))
	args["allowSignedEnabled"] = llx.IntData(int64(alfConfig["allowsignedenabled"].(float64)))
	args["firewallUnload"] = llx.IntData(int64(alfConfig["firewallunload"].(float64)))
	args["globalState"] = llx.IntData(int64(alfConfig["globalstate"].(float64)))
	args["loggingEnabled"] = llx.IntData(int64(alfConfig["loggingenabled"].(float64)))
	args["loggingOption"] = llx.IntData(int64(alfConfig["loggingoption"].(float64)))
	args["stealthEnabled"] = llx.IntData(int64(alfConfig["stealthenabled"].(float64)))
	args["version"] = llx.StringData(alfConfig["version"].(string))
	args["exceptions"] = llx.ArrayData(alfConfig["exceptions"].([]interface{}), types.Dict)
	args["explicitAuths"] = llx.ArrayData(explicitAuths, types.String)
	args["applications"] = llx.ArrayData(alfConfig["applications"].([]interface{}), types.Dict)

	return args, nil, nil
}
