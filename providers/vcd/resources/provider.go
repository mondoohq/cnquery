// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/vmware/go-vcloud-director/v2/types/v56"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers/vcd/connection"
)

func (v *mqlVcd) providerVDCs() ([]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VcdConnection)
	client := conn.Client()

	vdcList, err := client.QueryProviderVdcs()
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range vdcList {
		entry, err := newMqlVcdProvider(v.MqlRuntime, vdcList[i])
		if err != nil {
			return nil, err
		}
		list = append(list, entry)
	}

	return list, nil
}

func newMqlVcdProvider(runtime *plugin.Runtime, vcdProvider *types.QueryResultVMWProviderVdcRecordType) (interface{}, error) {
	return CreateResource(runtime, "vcd.vdcProvider", map[string]*llx.RawData{
		"name":                    llx.StringData(vcdProvider.Name),
		"status":                  llx.StringData(vcdProvider.Status),
		"isBusy":                  llx.BoolData(vcdProvider.IsBusy),
		"isDeleted":               llx.BoolData(vcdProvider.IsDeleted),
		"isEnabled":               llx.BoolData(vcdProvider.IsEnabled),
		"cpuAllocationMhz":        llx.IntData(int64(vcdProvider.CpuAllocationMhz)),
		"cpuLimitMhz":             llx.IntData(int64(vcdProvider.CpuLimitMhz)),
		"cpuUsedMhz":              llx.IntData(int64(vcdProvider.CpuUsedMhz)),
		"numberOfDatastores":      llx.IntData(int64(vcdProvider.NumberOfDatastores)),
		"numberOfStorageProfiles": llx.IntData(int64(vcdProvider.NumberOfStorageProfiles)),
		"numberOfVdcs":            llx.IntData(int64(vcdProvider.NumberOfVdcs)),
		"memoryAllocationMB":      llx.IntData(vcdProvider.MemoryAllocationMB),
		"memoryLimitMB":           llx.IntData(vcdProvider.MemoryLimitMB),
		"memoryUsedMB":            llx.IntData(vcdProvider.MemoryUsedMB),
		"storageAllocationMB":     llx.IntData(vcdProvider.StorageAllocationMB),
		"storageLimitMB":          llx.IntData(vcdProvider.StorageLimitMB),
		"storageUsedMB":           llx.IntData(vcdProvider.StorageUsedMB),
		"cpuOverheadMhz":          llx.IntData(vcdProvider.CpuOverheadMhz),
		"storageOverheadMB":       llx.IntData(vcdProvider.StorageOverheadMB),
		"memoryOverheadMB":        llx.IntData(vcdProvider.MemoryOverheadMB),
	})
}

func (v *mqlVcdVdcProvider) id() (string, error) {
	return "vcd.vdcProvider/" + v.Name.Data, v.Name.Error
}

func (v *mqlVcdVdcProvider) metadata() (map[string]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VcdConnection)
	client := conn.Client()

	if v.Name.Error != nil {
		return nil, v.Name.Error
	}
	name := v.Name.Data

	res := map[string]interface{}{}

	vdc, err := client.GetProviderVdcByName(name)
	if err != nil {
		return nil, err
	}
	metadata, err := vdc.GetMetadata()
	if err != nil {
		return nil, err
	}

	for i := range metadata.MetadataEntry {
		entry := metadata.MetadataEntry[i]
		key := entry.Key
		value := ""
		if entry.TypedValue != nil {
			value = entry.TypedValue.Value
		}
		res[key] = value
	}

	return res, nil
}
