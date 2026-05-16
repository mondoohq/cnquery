// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	betamodels "github.com/microsoftgraph/msgraph-beta-sdk-go/models"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/ms365/connection"
)

func (m *mqlMicrosoftDevicemanagementWindowsAutopilotDeploymentProfile) id() (string, error) {
	return m.Id.Data, nil
}

func (a *mqlMicrosoftDevicemanagement) windowsAutopilotDeploymentProfiles() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.BetaGraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	resp, err := graphClient.DeviceManagement().WindowsAutopilotDeploymentProfiles().Get(ctx, nil)
	if err != nil {
		return nil, transformError(err)
	}
	profiles, err := iterate[betamodels.WindowsAutopilotDeploymentProfileable](ctx, resp, graphClient.GetAdapter(), betamodels.CreateWindowsAutopilotDeploymentProfileCollectionResponseFromDiscriminatorValue)
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, profile := range profiles {
		oobeSettings := autopilotOobeToDict(profile.GetOutOfBoxExperienceSettings())
		profileType := ""
		if v := profile.GetOdataType(); v != nil {
			profileType = trimOdataType(*v)
		}
		assignedDeviceCount := int64(0)
		if devices := profile.GetAssignedDevices(); devices != nil {
			assignedDeviceCount = int64(len(devices))
		}
		r, err := CreateResource(a.MqlRuntime, "microsoft.devicemanagement.windowsAutopilotDeploymentProfile",
			map[string]*llx.RawData{
				"__id":                       llx.StringDataPtr(profile.GetId()),
				"id":                         llx.StringDataPtr(profile.GetId()),
				"displayName":                llx.StringDataPtr(profile.GetDisplayName()),
				"description":                llx.StringDataPtr(profile.GetDescription()),
				"language":                   llx.StringDataPtr(profile.GetLanguage()),
				"locale":                     llx.StringDataPtr(profile.GetLocale()),
				"deviceNameTemplate":         llx.StringDataPtr(profile.GetDeviceNameTemplate()),
				"deploymentProfileType":      llx.StringData(profileType),
				"extractHardwareHash":        llx.BoolDataPtr(profile.GetExtractHardwareHash()),
				"enableWhiteGlove":           llx.BoolDataPtr(profile.GetEnableWhiteGlove()),
				"outOfBoxExperienceSettings": llx.DictData(oobeSettings),
				"assignedDeviceCount":        llx.IntData(assignedDeviceCount),
				"createdDateTime":            llx.TimeDataPtr(profile.GetCreatedDateTime()),
				"lastModifiedDateTime":       llx.TimeDataPtr(profile.GetLastModifiedDateTime()),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func autopilotOobeToDict(s betamodels.OutOfBoxExperienceSettingsable) map[string]any {
	if s == nil {
		return nil
	}
	out := map[string]any{}
	if v := s.GetHidePrivacySettings(); v != nil {
		out["hidePrivacySettings"] = *v
	}
	if v := s.GetHideEULA(); v != nil {
		out["hideEULA"] = *v
	}
	if v := s.GetSkipKeyboardSelectionPage(); v != nil {
		out["skipKeyboardSelectionPage"] = *v
	}
	if v := s.GetHideEscapeLink(); v != nil {
		out["hideEscapeLink"] = *v
	}
	if v := s.GetUserType(); v != nil {
		out["userType"] = v.String()
	}
	if v := s.GetDeviceUsageType(); v != nil {
		out["deviceUsageType"] = v.String()
	}
	return out
}
