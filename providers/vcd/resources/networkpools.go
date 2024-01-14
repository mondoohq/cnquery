// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/vmware/go-vcloud-director/v2/types/v56"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/vcd/connection"
)

// https://developer.vmware.com/apis/1260/vmware-cloud-director/doc/doc/types/QueryResultNetworkPoolRecordType.html
func (v *mqlVcd) networkPools() ([]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VcdConnection)
	client := conn.Client()

	networkPoolList, err := client.QueryNetworkPools()
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range networkPoolList {
		entry, err := newMqlVcdNetworkPool(v.MqlRuntime, networkPoolList[i])
		if err != nil {
			return nil, err
		}
		list = append(list, entry)
	}

	return list, nil
}

func newMqlVcdNetworkPool(runtime *plugin.Runtime, networkPool *types.QueryResultNetworkPoolRecordType) (interface{}, error) {
	return CreateResource(runtime, "vcd.networkPool", map[string]*llx.RawData{
		"name":            llx.StringData(networkPool.Name),
		"isBusy":          llx.BoolData(networkPool.IsBusy),
		"networkPoolType": llx.IntData(int64(networkPool.NetworkPoolType)),
	})
}

func (o *mqlVcdNetworkPool) id() (string, error) {
	return "vcd.networkPool/" + o.Name.Data, o.Name.Error
}
