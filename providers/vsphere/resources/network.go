// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"reflect"

	"go.mondoo.com/mql/v13/providers/vsphere/connection"
	"go.mondoo.com/mql/v13/providers/vsphere/resources/resourceclient"
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

// failoverPolicy is the legacy ESXCli-sourced dict view of the vSwitch's NIC
// teaming policy. Superseded by failoverSettings() which returns a typed
// resource sourced directly from the cached host config.
func (v *mqlVsphereVswitchStandard) failoverPolicy() (map[string]any, error) {
	if v.Name.Error != nil {
		return nil, v.Name.Error
	}
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	return esxiClient.VswitchStandardFailoverPolicy(v.Name.Data)
}

// securityPolicy is the legacy ESXCli-sourced dict view of the vSwitch's
// layer-2 security policy. Superseded by securitySettings().
func (v *mqlVsphereVswitchStandard) securityPolicy() (map[string]any, error) {
	if v.Name.Error != nil {
		return nil, v.Name.Error
	}
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	return esxiClient.VswitchStandardSecurityPolicy(v.Name.Data)
}

// shapingPolicy is the legacy ESXCli-sourced dict view of the vSwitch's
// traffic shaping policy. Superseded by shapingSettings().
func (v *mqlVsphereVswitchStandard) shapingPolicy() (map[string]any, error) {
	if v.Name.Error != nil {
		return nil, v.Name.Error
	}
	esxiClient, err := v.esxiClient()
	if err != nil {
		return nil, err
	}
	return esxiClient.VswitchStandardShapingPolicy(v.Name.Data)
}

func (v *mqlVsphereVswitchStandard) uplinks() ([]any, error) {
	props := v.GetProperties()
	if props.Error != nil {
		return nil, props.Error
	}

	properties, ok := props.Data.(map[string]any)
	if !ok {
		return nil, errors.New("unexpected properties structure for vsphere switch")
	}

	// if no properties are set, we have no uplinks for the switch
	if properties == nil {
		return []any{}, nil
	}

	uplinksRaw := properties["Uplinks"]
	if uplinksRaw == nil {
		return []any{}, nil
	}

	uplinkNames, ok := uplinksRaw.([]any)
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

func findHostAdapter(host *mqlVsphereHost, uplinkNames []any) ([]any, error) {
	adapters := host.GetAdapters()
	if adapters.Error != nil {
		return nil, errors.New("cannot retrieve esxi host adapters")
	}

	// gather all adapters on that host so that we can find the adapter by name
	mqlUplinks := []any{}
	for i := range adapters.Data {
		adapter := adapters.Data[i].(*mqlVsphereVmnic)

		if adapter.Name.Error != nil {
			return nil, errors.New("cannot retrieve esxi adapter name")
		}
		name := adapter.Name.Data

		for i := range uplinkNames {
			uplinkName, ok := uplinkNames[i].(string)
			if !ok {
				continue
			}
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

func (v *mqlVsphereVswitchDvs) uplinks() ([]any, error) {
	props := v.GetProperties()
	if props.Error != nil {
		return nil, props.Error
	}

	properties, ok := props.Data.(map[string]any)
	if !ok {
		return nil, errors.New("unexpected properties structure for vsphere switch")
	}

	// if no properties are set, we have no uplinks for the dvs
	if properties == nil {
		return []any{}, nil
	}

	uplinksRaw, ok := properties["Uplinks"]
	if !ok || uplinksRaw == nil {
		return []any{}, nil
	}

	uplinkNames, ok := uplinksRaw.([]any)
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

func (v *mqlVsphereVswitchPortgroup) id() (string, error) {
	return v.Name.Data, v.Name.Error
}
