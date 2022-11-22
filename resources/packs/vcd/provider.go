package vcd

import (
	"github.com/vmware/go-vcloud-director/v2/types/v56"
	"go.mondoo.com/cnquery/resources"
)

func (v *mqlVcd) GetProviderVDCs() ([]interface{}, error) {
	op, err := vcdProvider(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	vdcList, err := client.QueryProviderVdcs()
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range vdcList {
		entry, err := newMqlVcdProvider(v.MotorRuntime, vdcList[i])
		if err != nil {
			return nil, err
		}
		list = append(list, entry)
	}

	return list, nil
}

func newMqlVcdProvider(runtime *resources.Runtime, vcdProvider *types.QueryResultVMWProviderVdcRecordType) (interface{}, error) {
	return runtime.CreateResource("vcd.vdcProvider",
		"name", vcdProvider.Name,
		"status", vcdProvider.Status,
		"isBusy", vcdProvider.IsBusy,
		"isDeleted", vcdProvider.IsDeleted,
		"isEnabled", vcdProvider.IsEnabled,
		"cpuAllocationMhz", int64(vcdProvider.CpuAllocationMhz),
		"cpuLimitMhz", int64(vcdProvider.CpuLimitMhz),
		"cpuUsedMhz", int64(vcdProvider.CpuUsedMhz),
		"numberOfDatastores", int64(vcdProvider.NumberOfDatastores),
		"numberOfStorageProfiles", int64(vcdProvider.NumberOfStorageProfiles),
		"numberOfVdcs", int64(vcdProvider.NumberOfVdcs),
		"memoryAllocationMB", vcdProvider.MemoryAllocationMB,
		"memoryLimitMB", vcdProvider.MemoryLimitMB,
		"memoryUsedMB", vcdProvider.MemoryUsedMB,
		"storageAllocationMB", vcdProvider.StorageAllocationMB,
		"storageLimitMB", vcdProvider.StorageLimitMB,
		"storageUsedMB", vcdProvider.StorageUsedMB,
		"cpuOverheadMhz", vcdProvider.CpuOverheadMhz,
		"storageOverheadMB", vcdProvider.StorageOverheadMB,
		"memoryOverheadMB", vcdProvider.MemoryOverheadMB,
	)
}

func (v *mqlVcdVdcProvider) id() (string, error) {
	id, err := v.Name()
	if err != nil {
		return "", err
	}
	return "vcd.provider/" + id, nil
}

func (v *mqlVcdVdcProvider) GetMetadata() (map[string]interface{}, error) {
	op, err := vcdProvider(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	name, err := v.Name()
	if err != nil {
		return nil, err
	}

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
