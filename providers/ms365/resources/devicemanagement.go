// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/microsoftgraph/msgraph-sdk-go/devicemanagement"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/ms365/connection"
	"go.mondoo.com/cnquery/v12/types"
)

func (m *mqlMicrosoftDevicemanagementDeviceconfiguration) id() (string, error) {
	return m.Id.Data, nil
}

func (m *mqlMicrosoftDevicemanagementDevicecompliancepolicy) id() (string, error) {
	return m.Id.Data, nil
}

func (a *mqlMicrosoftDevicemanagement) deviceConfigurations() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.DeviceManagement().DeviceConfigurations().Get(ctx, &devicemanagement.DeviceConfigurationsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	res := []interface{}{}
	configurations := resp.GetValue()
	for _, configuration := range configurations {
		mqlResource, err := CreateResource(a.MqlRuntime, "microsoft.devicemanagement.deviceconfiguration",
			map[string]*llx.RawData{
				"id":                   llx.StringDataPtr(configuration.GetId()),
				"lastModifiedDateTime": llx.TimeDataPtr(configuration.GetLastModifiedDateTime()),
				"createdDateTime":      llx.TimeDataPtr(configuration.GetCreatedDateTime()),
				"description":          llx.StringDataPtr(configuration.GetDescription()),
				"displayName":          llx.StringDataPtr(configuration.GetDisplayName()),
				"version":              llx.IntDataDefault(configuration.GetVersion(), 0),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}
	return res, nil
}

func (a *mqlMicrosoftDevicemanagement) deviceCompliancePolicies() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	requestConfig := &devicemanagement.DeviceCompliancePoliciesRequestBuilderGetRequestConfiguration{
		QueryParameters: &devicemanagement.DeviceCompliancePoliciesRequestBuilderGetQueryParameters{
			Expand: []string{"assignments"},
		},
	}
	resp, err := graphClient.DeviceManagement().DeviceCompliancePolicies().Get(ctx, requestConfig)
	if err != nil {
		return nil, transformError(err)
	}

	compliancePolicies := resp.GetValue()
	res := []interface{}{}
	for _, compliancePolicy := range compliancePolicies {
		assignments, err := convert.JsonToDictSlice(newDeviceCompliancePolicyAssignments(compliancePolicy.GetAssignments()))
		if err != nil {
			return nil, err
		}
		mqlResource, err := CreateResource(a.MqlRuntime, "microsoft.devicemanagement.devicecompliancepolicy",
			map[string]*llx.RawData{
				"id":                   llx.StringDataPtr(compliancePolicy.GetId()),
				"createdDateTime":      llx.TimeDataPtr(compliancePolicy.GetCreatedDateTime()),
				"description":          llx.StringDataPtr(compliancePolicy.GetDescription()),
				"displayName":          llx.StringDataPtr(compliancePolicy.GetDisplayName()),
				"lastModifiedDateTime": llx.TimeDataPtr(compliancePolicy.GetLastModifiedDateTime()),
				"version":              llx.IntDataDefault(compliancePolicy.GetVersion(), 0),
				"assignments":          llx.ArrayData(assignments, types.Any),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}
	return res, nil
}
