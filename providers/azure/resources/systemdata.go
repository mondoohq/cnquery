// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// azureSystemData builds the typed azure.subscription.systemData resource from
// the raw systemData dict already attached to a resource. The dict is produced
// by convert.JsonToDict on the SDK's SystemData struct, so timestamps arrive as
// RFC 3339 strings. Returns nil when the resource carries no system metadata.
func azureSystemData(runtime *plugin.Runtime, parentID string, raw any) (*mqlAzureSubscriptionSystemData, error) {
	m, ok := raw.(map[string]any)
	if !ok || len(m) == 0 {
		return nil, nil
	}

	getStr := func(key string) string {
		if v, ok := m[key].(string); ok {
			return v
		}
		return ""
	}
	getTime := func(key string) *time.Time {
		s, ok := m[key].(string)
		if !ok || s == "" {
			return nil
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil
		}
		return &t
	}

	res, err := CreateResource(runtime, "azure.subscription.systemData", map[string]*llx.RawData{
		"__id":               llx.StringData(parentID + "/systemData"),
		"createdBy":          llx.StringData(getStr("createdBy")),
		"createdByType":      llx.StringData(getStr("createdByType")),
		"createdAt":          llx.TimeDataPtr(getTime("createdAt")),
		"lastModifiedBy":     llx.StringData(getStr("lastModifiedBy")),
		"lastModifiedByType": llx.StringData(getStr("lastModifiedByType")),
		"lastModifiedAt":     llx.TimeDataPtr(getTime("lastModifiedAt")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSystemData), nil
}

func (a *mqlAzureSubscriptionComputeServiceVm) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	sd, err := azureSystemData(a.MqlRuntime, a.Id.Data, a.GetSystemData().Data)
	if err != nil {
		return nil, err
	}
	if sd == nil {
		a.SystemMetadata.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return sd, nil
}

func (a *mqlAzureSubscriptionComputeServiceDisk) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	sd, err := azureSystemData(a.MqlRuntime, a.Id.Data, a.GetSystemData().Data)
	if err != nil {
		return nil, err
	}
	if sd == nil {
		a.SystemMetadata.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return sd, nil
}

func (a *mqlAzureSubscriptionComputeServiceSnapshot) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	sd, err := azureSystemData(a.MqlRuntime, a.Id.Data, a.GetSystemData().Data)
	if err != nil {
		return nil, err
	}
	if sd == nil {
		a.SystemMetadata.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return sd, nil
}

func (a *mqlAzureSubscriptionComputeServiceVmScaleSet) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	sd, err := azureSystemData(a.MqlRuntime, a.Id.Data, a.GetSystemData().Data)
	if err != nil {
		return nil, err
	}
	if sd == nil {
		a.SystemMetadata.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return sd, nil
}

func (a *mqlAzureSubscriptionComputeServiceHybridMachine) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	sd, err := azureSystemData(a.MqlRuntime, a.Id.Data, a.GetSystemData().Data)
	if err != nil {
		return nil, err
	}
	if sd == nil {
		a.SystemMetadata.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return sd, nil
}

func (a *mqlAzureSubscriptionComputeServiceHybridMachineExtension) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	sd, err := azureSystemData(a.MqlRuntime, a.Id.Data, a.GetSystemData().Data)
	if err != nil {
		return nil, err
	}
	if sd == nil {
		a.SystemMetadata.State = plugin.StateIsSet | plugin.StateIsNull
	}
	return sd, nil
}
