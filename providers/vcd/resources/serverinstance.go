// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/vcd/connection"
)

// https://developer.vmware.com/apis/1260/vmware-cloud-director/doc/doc/types/VimServerType.html
func (v *mqlVcd) serverInstances() ([]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VcdConnection)
	client := conn.Client()

	vdcList, err := govcd.QueryVirtualCenters(client, "")
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range vdcList {
		entry, err := newMqlVcdServerInstance(v.MqlRuntime, vdcList[i])
		if err != nil {
			return nil, err
		}
		list = append(list, entry)
	}

	return list, nil
}

func newMqlVcdServerInstance(runtime *plugin.Runtime, vcdProvider *types.QueryResultVirtualCenterRecordType) (interface{}, error) {
	return CreateResource(runtime, "vcd.serverInstance", map[string]*llx.RawData{
		"name":          llx.StringData(vcdProvider.Name),
		"isBusy":        llx.BoolData(vcdProvider.IsBusy),
		"isEnabled":     llx.BoolData(vcdProvider.IsEnabled),
		"isSupported":   llx.BoolData(vcdProvider.IsEnabled),
		"listenerState": llx.StringData(vcdProvider.ListenerState),
		"status":        llx.StringData(vcdProvider.Status),
		"userName":      llx.StringData(vcdProvider.UserName),
		"vcVersion":     llx.StringData(vcdProvider.VcVersion),
		"uuid":          llx.StringData(vcdProvider.UUID),
		"vsmIP":         llx.StringData(vcdProvider.VsmIP),
	})
}

func (o *mqlVcdServerInstance) id() (string, error) {
	return "vcd.serverInstance/" + o.Name.Data, o.Name.Error
}
