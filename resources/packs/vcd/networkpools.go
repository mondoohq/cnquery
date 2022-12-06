package vcd

import (
	"github.com/vmware/go-vcloud-director/v2/types/v56"
	"go.mondoo.com/cnquery/resources"
)

// https://developer.vmware.com/apis/1260/vmware-cloud-director/doc/doc/types/QueryResultNetworkPoolRecordType.html
func (v *mqlVcd) GetNetworkPools() ([]interface{}, error) {
	op, err := vcdProvider(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	networkPoolList, err := client.QueryNetworkPools()
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range networkPoolList {
		entry, err := newMqlVcdNetworkPool(v.MotorRuntime, networkPoolList[i])
		if err != nil {
			return nil, err
		}
		list = append(list, entry)
	}

	return list, nil
}

func newMqlVcdNetworkPool(runtime *resources.Runtime, networkPool *types.QueryResultNetworkPoolRecordType) (interface{}, error) {
	return runtime.CreateResource("vcd.networkPool",
		"name", networkPool.Name,
		"isBusy", networkPool.IsBusy,
		"networkPoolType", int64(networkPool.NetworkPoolType),
	)
}

func (o *mqlVcdNetworkPool) id() (string, error) {
	id, err := o.Name()
	if err != nil {
		return "", err
	}
	return "vcd.networkPool/" + id, nil
}
