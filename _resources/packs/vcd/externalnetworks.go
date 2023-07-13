package vcd

import (
	"github.com/vmware/go-vcloud-director/v2/types/v56"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (v *mqlVcd) GetExternalNetworks() ([]interface{}, error) {
	op, err := vcdProvider(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	externalNetworkList, err := client.GetExternalNetworks()
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range externalNetworkList.ExternalNetworkReference {
		entry, err := newMqlVcdExternalNetwork(v.MotorRuntime, externalNetworkList.ExternalNetworkReference[i])
		if err != nil {
			return nil, err
		}
		list = append(list, entry)
	}

	return list, nil
}

func newMqlVcdExternalNetwork(runtime *resources.Runtime, networkPool *types.ExternalNetworkReference) (interface{}, error) {
	return runtime.CreateResource("vcd.externalNetwork",
		"name", networkPool.Name,
	)
}

func (v *mqlVcdExternalNetwork) id() (string, error) {
	id, err := v.Name()
	if err != nil {
		return "", err
	}
	return "vcd.externalNetwork/" + id, nil
}

func (v *mqlVcdExternalNetwork) getData() (*types.ExternalNetwork, error) {
	// check if the data is cached
	// TODO: probably we need locking here to make sure concurrent access is covered
	entry, ok := v.Cache.Load("_externalNetwork")
	if ok {
		extN, ok := entry.Data.(*types.ExternalNetwork)
		if ok {
			return extN, nil
		}
	}

	op, err := vcdProvider(v.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	client := op.Client()

	name, err := v.Name()
	if err != nil {
		return nil, err
	}

	data, err := client.GetExternalNetworkByName(name)
	if err != nil {
		return nil, err
	}

	v.Cache.Store("_externalNetwork", &resources.CacheEntry{Data: data})

	return data.ExternalNetwork, nil
}

func (v *mqlVcdExternalNetwork) GetId() (string, error) {
	externalNetwork, err := v.getData()
	if err != nil {
		return "", err
	}
	return externalNetwork.ID, nil
}

func (v *mqlVcdExternalNetwork) GetDescription() (string, error) {
	externalNetwork, err := v.getData()
	if err != nil {
		return "", err
	}
	return externalNetwork.Description, nil
}

func (v *mqlVcdExternalNetwork) GetConfiguration() (interface{}, error) {
	externalNetwork, err := v.getData()
	if err != nil {
		return "", err
	}

	// TODO: json has inconsistent naming
	return core.JsonToDict(externalNetwork.Configuration)
}
