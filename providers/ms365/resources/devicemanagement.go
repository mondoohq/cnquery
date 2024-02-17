// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/microsoftgraph/msgraph-sdk-go/devicemanagement"
	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/ms365/connection"
	"go.mondoo.com/cnquery/v10/types"
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
		properties := getConfigurationProperties(configuration)
		mqlResource, err := CreateResource(a.MqlRuntime, "microsoft.devicemanagement.deviceconfiguration",
			map[string]*llx.RawData{
				"id":                   llx.StringDataPtr(configuration.GetId()),
				"lastModifiedDateTime": llx.TimeDataPtr(configuration.GetLastModifiedDateTime()),
				"createdDateTime":      llx.TimeDataPtr(configuration.GetCreatedDateTime()),
				"description":          llx.StringDataPtr(configuration.GetDescription()),
				"displayName":          llx.StringDataPtr(configuration.GetDisplayName()),
				"version":              llx.IntData(convert.ToInt64From32(configuration.GetVersion())),
				"properties":           llx.DictData(properties),
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
		properties := getComplianceProperties(compliancePolicy)
		mqlResource, err := CreateResource(a.MqlRuntime, "microsoft.devicemanagement.devicecompliancepolicy",
			map[string]*llx.RawData{
				"id":                   llx.StringDataPtr(compliancePolicy.GetId()),
				"createdDateTime":      llx.TimeDataPtr(compliancePolicy.GetCreatedDateTime()),
				"description":          llx.StringDataPtr(compliancePolicy.GetDescription()),
				"displayName":          llx.StringDataPtr(compliancePolicy.GetDisplayName()),
				"lastModifiedDateTime": llx.TimeDataPtr(compliancePolicy.GetLastModifiedDateTime()),
				"version":              llx.IntData(convert.ToInt64From32(compliancePolicy.GetVersion())),
				"assignments":          llx.ArrayData(assignments, types.Any),
				"properties":           llx.DictData(properties),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlResource)
	}
	return res, nil
}

// TODO: androidDeviceOwnerGeneralDeviceConfiguration missing
func getConfigurationProperties(config models.DeviceConfigurationable) map[string]interface{} {
	props := map[string]interface{}{}
	if config.GetOdataType() != nil {
		props["@odata.type"] = *config.GetOdataType()
	}

	agdc, ok := config.(*models.AndroidGeneralDeviceConfiguration)
	if ok {
		if agdc.GetPasswordRequired() != nil {
			props["passwordRequired"] = *agdc.GetPasswordRequired()
		}
		if agdc.GetPasswordSignInFailureCountBeforeFactoryReset() != nil {
			props["passwordSignInFailureCountBeforeFactoryReset"] = int64(*agdc.GetPasswordSignInFailureCountBeforeFactoryReset())
		}
		if agdc.GetPasswordMinimumLength() != nil {
			props["passwordMinimumLength"] = int64(*agdc.GetPasswordMinimumLength())
		}
		if agdc.GetStorageRequireDeviceEncryption() != nil {
			props["storageRequireDeviceEncryption"] = *agdc.GetStorageRequireDeviceEncryption()
		}
		if agdc.GetPasswordRequiredType() != nil {
			props["passwordRequiredType"] = agdc.GetPasswordRequiredType().String()
		}
		if agdc.GetPasswordExpirationDays() != nil {
			props["passwordExpirationDays"] = int64(*agdc.GetPasswordExpirationDays())
		}
		if agdc.GetPasswordMinutesOfInactivityBeforeScreenTimeout() != nil {
			props["passwordMinutesOfInactivityBeforeScreenTimeout"] = int64(*agdc.GetPasswordMinutesOfInactivityBeforeScreenTimeout())
		}
	}
	w10gc, ok := config.(*models.Windows10GeneralConfiguration)
	if ok {
		if w10gc.GetPasswordRequired() != nil {
			props["passwordRequired"] = *w10gc.GetPasswordRequired()
		}
		if w10gc.GetPasswordBlockSimple() != nil {
			props["passwordBlockSimple"] = *w10gc.GetPasswordBlockSimple()
		}
		if w10gc.GetPasswordMinutesOfInactivityBeforeScreenTimeout() != nil {
			props["passwordMinutesOfInactivityBeforeScreenTimeout"] = int64(*w10gc.GetPasswordMinutesOfInactivityBeforeScreenTimeout())
		}
		if w10gc.GetPasswordSignInFailureCountBeforeFactoryReset() != nil {
			props["passwordSignInFailureCountBeforeFactoryReset"] = int64(*w10gc.GetPasswordSignInFailureCountBeforeFactoryReset())
		}
		if w10gc.GetPasswordMinimumLength() != nil {
			props["passwordMinimumLength"] = int64(*w10gc.GetPasswordMinimumLength())
		}
		if w10gc.GetPasswordRequiredType() != nil {
			props["passwordRequiredType"] = w10gc.GetPasswordRequiredType().String()
		}
		if w10gc.GetPasswordExpirationDays() != nil {
			props["passwordExpirationDays"] = int64(*w10gc.GetPasswordExpirationDays())
		}
		if w10gc.GetPasswordExpirationDays() != nil {
			props["passwordExpirationDays"] = int64(*w10gc.GetPasswordExpirationDays())
		}
	}
	macdc, ok := config.(*models.MacOSGeneralDeviceConfiguration)
	if ok {
		if macdc.GetPasswordMinutesOfInactivityBeforeScreenTimeout() != nil {
			props["passwordMinutesOfInactivityBeforeScreenTimeout"] = int64(*macdc.GetPasswordMinutesOfInactivityBeforeScreenTimeout())
		}
		if macdc.GetPasswordMinimumLength() != nil {
			props["passwordMinimumLength"] = int64(*macdc.GetPasswordMinimumLength())
		}
		if macdc.GetPasswordMinutesOfInactivityBeforeLock() != nil {
			props["passwordMinutesOfInactivityBeforeLock"] = int64(*macdc.GetPasswordMinutesOfInactivityBeforeLock())
		}
		if macdc.GetPasswordRequiredType() != nil {
			props["passwordRequiredType"] = macdc.GetPasswordRequiredType().String()
		}
		if macdc.GetPasswordBlockSimple() != nil {
			props["passwordBlockSimple"] = *macdc.GetPasswordBlockSimple()
		}
		if macdc.GetPasswordRequired() != nil {
			props["passwordRequired"] = *macdc.GetPasswordRequired()
		}
		if macdc.GetPasswordExpirationDays() != nil {
			props["passwordExpirationDays"] = int64(*macdc.GetPasswordExpirationDays())
		}
	}

	iosdc, ok := config.(*models.IosGeneralDeviceConfiguration)
	if ok {
		if iosdc.GetPasscodeSignInFailureCountBeforeWipe() != nil {
			props["passcodeSignInFailureCountBeforeWipe"] = int64(*iosdc.GetPasscodeSignInFailureCountBeforeWipe())
		}
		if iosdc.GetPasscodeMinimumLength() != nil {
			props["passcodeMinimumLength"] = int64(*iosdc.GetPasscodeMinimumLength())
		}
		if iosdc.GetPasscodeMinutesOfInactivityBeforeLock() != nil {
			props["passcodeMinutesOfInactivityBeforeLock"] = int64(*iosdc.GetPasscodeMinutesOfInactivityBeforeLock())
		}
		if iosdc.GetPasscodeMinutesOfInactivityBeforeScreenTimeout() != nil {
			props["passcodeMinutesOfInactivityBeforeScreenTimeout"] = int64(*iosdc.GetPasscodeMinutesOfInactivityBeforeScreenTimeout())
		}
		if iosdc.GetPasscodeRequiredType() != nil {
			props["passcodeRequiredType"] = iosdc.GetPasscodeRequiredType().String()
		}
		if iosdc.GetPasscodeBlockSimple() != nil {
			props["passcodeBlockSimple"] = *iosdc.GetPasscodeBlockSimple()
		}
		if iosdc.GetPasscodeRequired() != nil {
			props["passcodeRequired"] = *iosdc.GetPasscodeRequired()
		}
		if iosdc.GetPasscodeExpirationDays() != nil {
			props["passcodeExpirationDays"] = int64(*iosdc.GetPasscodeExpirationDays())
		}
	}
	awpgdc, ok := config.(*models.AndroidWorkProfileGeneralDeviceConfiguration)
	if ok {
		if awpgdc.GetPasswordSignInFailureCountBeforeFactoryReset() != nil {
			props["passwordSignInFailureCountBeforeFactoryReset"] = int64(*awpgdc.GetPasswordSignInFailureCountBeforeFactoryReset())
		}
		if awpgdc.GetPasswordMinimumLength() != nil {
			props["passwordMinimumLength"] = int64(*awpgdc.GetPasswordMinimumLength())
		}
		if awpgdc.GetPasswordMinutesOfInactivityBeforeScreenTimeout() != nil {
			props["passwordMinutesOfInactivityBeforeScreenTimeout"] = int64(*awpgdc.GetPasswordMinutesOfInactivityBeforeScreenTimeout())
		}
		if awpgdc.GetWorkProfilePasswordMinutesOfInactivityBeforeScreenTimeout() != nil {
			props["workProfilePasswordMinutesOfInactivityBeforeScreenTimeout"] = int64(*awpgdc.GetWorkProfilePasswordMinutesOfInactivityBeforeScreenTimeout())
		}
		if awpgdc.GetPasswordRequiredType() != nil {
			props["passwordRequiredType"] = awpgdc.GetPasswordRequiredType().String()
		}
		if awpgdc.GetWorkProfilePasswordRequiredType() != nil {
			props["workProfilePasswordRequiredType"] = awpgdc.GetWorkProfilePasswordRequiredType().String()
		}
		if awpgdc.GetPasswordExpirationDays() != nil {
			props["passwordExpirationDays"] = int64(*awpgdc.GetPasswordExpirationDays())
		}
	}
	return props
}

// TODO: windows 10 props missing.
func getComplianceProperties(compliance models.DeviceCompliancePolicyable) map[string]interface{} {
	props := map[string]interface{}{}
	if compliance.GetOdataType() != nil {
		props["@odata.type"] = *compliance.GetOdataType()
	}

	ioscp, ok := compliance.(*models.IosCompliancePolicy)
	if ok {
		if ioscp.GetSecurityBlockJailbrokenDevices() != nil {
			props["securityBlockJailbrokenDevices"] = *ioscp.GetSecurityBlockJailbrokenDevices()
		}
		if ioscp.GetManagedEmailProfileRequired() != nil {
			props["managedEmailProfileRequired"] = *ioscp.GetManagedEmailProfileRequired()
		}
	}
	androidcp, ok := compliance.(*models.AndroidCompliancePolicy)
	if ok {
		if androidcp.GetSecurityBlockJailbrokenDevices() != nil {
			props["securityBlockJailbrokenDevices"] = *androidcp.GetSecurityBlockJailbrokenDevices()
		}
	}
	androidworkcp, ok := compliance.(*models.AndroidWorkProfileCompliancePolicy)
	if ok {
		if androidworkcp.GetSecurityBlockJailbrokenDevices() != nil {
			props["securityBlockJailbrokenDevices"] = *androidworkcp.GetSecurityBlockJailbrokenDevices()
		}
	}
	return props
}
