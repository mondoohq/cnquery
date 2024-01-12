// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"reflect"

	"go.mondoo.com/cnquery/v10/providers/vsphere/connection"
	"go.mondoo.com/cnquery/v10/providers/vsphere/resources/resourceclient"
)

type mqlVsphereVswitchStandardInternal struct {
	hostInventoryPath string
	parentResource    *mqlVsphereHost
}

func (v *mqlVsphereVswitchStandard) id() (string, error) {
	return v.Name.Data, v.Name.Error
}

func (v *mqlVsphereVswitchStandard) esxiClient() (*resourceclient.Esxi, error) {
	conn := v.MqlRuntime.Connection.(*connection.VsphereConnection)
	return esxiClient(conn, v.hostInventoryPath)
}

func (v *mqlVsphereVswitchStandard) failoverPolicy() (map[string]interface{}, error) {
	if v.Name.Error != nil {
		return nil, v.Name.Error
	}
	name := v.Name.Data

	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}

	return esxiClient.VswitchStandardFailoverPolicy(name)
}

func (v *mqlVsphereVswitchStandard) securityPolicy() (map[string]interface{}, error) {
	if v.Name.Error != nil {
		return nil, v.Name.Error
	}
	name := v.Name.Data

	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}

	return esxiClient.VswitchStandardSecurityPolicy(name)
}

func (v *mqlVsphereVswitchStandard) shapingPolicy() (map[string]interface{}, error) {
	if v.Name.Error != nil {
		return nil, v.Name.Error
	}
	name := v.Name.Data

	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}

	return esxiClient.VswitchStandardShapingPolicy(name)
}

func (v *mqlVsphereVswitchStandard) uplinks() ([]interface{}, error) {
	props := v.GetProperties()
	if props.Error != nil {
		return nil, props.Error
	}

	properties, ok := props.Data.(map[string]interface{})
	if !ok {
		return nil, errors.New("unexpected properties structure for vsphere switch")
	}

	// if no properties are set, we have no uplinks for dvs
	if properties == nil {
		return nil, nil
	}

	uplinksRaw := properties["Uplinks"]

	// no uplinks for dvs
	if properties == nil {
		return nil, nil
	}

	uplinkNames, ok := uplinksRaw.([]interface{})
	if !ok {
		return nil, errors.New("unexpected type for vsphere switch uplinks " + reflect.ValueOf(uplinksRaw).Type().Name())
	}

	// get the esxi.host parent resource
	if v.parentResource == nil {
		return nil, errors.New("cannot get esxi host inventory path")
	}

	// get all host adapter
	return findHostAdapter(v.parentResource, uplinkNames)
}

func findHostAdapter(host *mqlVsphereHost, uplinkNames []interface{}) ([]interface{}, error) {
	adapters := host.GetAdapters()
	if adapters.Error != nil {
		return nil, errors.New("cannot retrieve esxi host adapters")
	}

	// gather all adapters on that host so that we can find the adapter by name
	mqlUplinks := []interface{}{}
	for i := range adapters.Data {
		adapter := adapters.Data[i].(*mqlVsphereVmnic)

		if adapter.Name.Error != nil {
			return nil, errors.New("cannot retrieve esxi adapter name")
		}
		name := adapter.Name.Data

		for i := range uplinkNames {
			uplinkName := uplinkNames[i].(string)

			if name == uplinkName {
				mqlUplinks = append(mqlUplinks, adapter)
			}
		}
	}

	return mqlUplinks, nil
}

type mqlVsphereVswitchDvsInternal struct {
	hostInventoryPath string
	parentResource    *mqlVsphereHost
}

func (v *mqlVsphereVswitchDvs) id() (string, error) {
	return v.Name.Data, v.Name.Error
}

func (v *mqlVsphereVswitchDvs) uplinks() ([]interface{}, error) {
	props := v.GetProperties()
	if props.Error != nil {
		return nil, props.Error
	}

	properties, ok := props.Data.(map[string]interface{})
	if !ok {
		return nil, errors.New("unexpected properties structure for vsphere switch")
	}

	// if no properties are set, we have no uplinks for dvs
	if properties == nil {
		return nil, nil
	}

	uplinksRaw := properties["Uplinks"]

	// no uplinks for dvs
	if properties == nil {
		return nil, nil
	}

	uplinkNames, ok := uplinksRaw.([]interface{})
	if !ok {
		return nil, errors.New("unexpected type for vsphere switch uplinks " + reflect.ValueOf(uplinksRaw).Type().Name())
	}

	// get the esxi.host parent resource
	if v.parentResource == nil {
		return nil, errors.New("cannot get esxi host inventory path")
	}

	// get all host adapter
	return findHostAdapter(v.parentResource, uplinkNames)
}
