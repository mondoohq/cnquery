// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/vmware/go-vcloud-director/v2/govcd"
	"github.com/vmware/go-vcloud-director/v2/types/v56"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/vcd/connection"
)

func (v *mqlVcd) externalNetworks() ([]interface{}, error) {
	conn := v.MqlRuntime.Connection.(*connection.VcdConnection)
	client := conn.Client()

	externalNetworkList, err := client.GetExternalNetworks()
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range externalNetworkList.ExternalNetworkReference {
		entry, err := newMqlVcdExternalNetwork(v.MqlRuntime, externalNetworkList.ExternalNetworkReference[i])
		if err != nil {
			return nil, err
		}
		list = append(list, entry)
	}

	return list, nil
}

func newMqlVcdExternalNetwork(runtime *plugin.Runtime, networkPool *types.ExternalNetworkReference) (interface{}, error) {
	return CreateResource(runtime, "vcd.externalNetwork", map[string]*llx.RawData{
		"name": llx.StringData(networkPool.Name),
	})
}

type mqlVcdExternalNetworkInternal struct {
	externalNetwork *govcd.ExternalNetwork
}

func (v *mqlVcdExternalNetwork) id() (string, error) {
	if v.Name.Error != nil {
		return "", v.Name.Error
	}

	// FIXME: DEPRECATED, remove in v10.0. The ID field will be removed and
	// this request won't be necessary anymore. vv
	urn := v.GetUrn()
	if urn == nil {
		v.Id = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	} else {
		v.Id = *urn
	}
	// ^^

	return "vcd.externalNetwork/" + v.Name.Data, nil
}

func (v *mqlVcdExternalNetwork) getData() (*types.ExternalNetwork, error) {
	// check if the data is cached
	// TODO: probably we need locking here to make sure concurrent access is covered
	if v.externalNetwork != nil {
		return v.externalNetwork.ExternalNetwork, nil
	}

	conn := v.MqlRuntime.Connection.(*connection.VcdConnection)
	client := conn.Client()

	if v.Name.Error != nil {
		return nil, v.Name.Error
	}
	name := v.Name.Data

	externalNetwork, err := client.GetExternalNetworkByName(name)
	if err != nil {
		return nil, err
	}
	v.externalNetwork = externalNetwork

	return externalNetwork.ExternalNetwork, nil
}

func (v *mqlVcdExternalNetwork) urn() (string, error) {
	externalNetwork, err := v.getData()
	if err != nil {
		return "", err
	}
	return externalNetwork.ID, nil
}

func (v *mqlVcdExternalNetwork) description() (string, error) {
	externalNetwork, err := v.getData()
	if err != nil {
		return "", err
	}
	return externalNetwork.Description, nil
}

func (v *mqlVcdExternalNetwork) configuration() (interface{}, error) {
	externalNetwork, err := v.getData()
	if err != nil {
		return "", err
	}

	// TODO: json has inconsistent naming
	return convert.JsonToDict(externalNetwork.Configuration)
}
