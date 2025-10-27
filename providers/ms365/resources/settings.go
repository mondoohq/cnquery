// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/microsoftgraph/msgraph-sdk-go/groupsettings"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers/ms365/connection"
	"go.mondoo.com/cnquery/v12/types"
)

func (a *mqlMicrosoft) settings() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.Ms365Connection)
	graphClient, err := conn.GraphClient()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	settings, err := graphClient.GroupSettings().Get(ctx, &groupsettings.GroupSettingsRequestBuilderGetRequestConfiguration{})
	if err != nil {
		return nil, transformError(err)
	}

	// Use microsoft.setting resource to create the settings as we loop over settings.getValue()
	settingsList := []any{}
	for _, setting := range settings.GetValue() {
		settingValues := make([]any, 0, len(setting.GetValues()))

		for _, val := range setting.GetValues() {
			settingValueResource, err := CreateResource(a.MqlRuntime, ResourceMicrosoftSettingValue,
				map[string]*llx.RawData{
					"__id":  llx.StringDataPtr(val.GetName()),
					"name":  llx.StringDataPtr(val.GetName()),
					"value": llx.StringDataPtr(val.GetValue()),
				})
			if err != nil {
				return nil, err
			}
			settingValues = append(settingValues, settingValueResource)
		}

		// Copy the values to avoid pointer reuse
		displayName := setting.GetDisplayName()
		templateId := setting.GetTemplateId()

		if displayName == nil || templateId == nil {
			continue
		}

		// Create a unique mqlId, otherwise first resource is returned everytime we call CreateResource
		mqlId := *displayName + "|" + *templateId

		settingResource, err := CreateResource(a.MqlRuntime, ResourceMicrosoftSetting,
			map[string]*llx.RawData{
				"__id":        llx.StringData(mqlId),
				"displayName": llx.StringDataPtr(displayName),
				"templateId":  llx.StringDataPtr(templateId),
				"values":      llx.ArrayData(llx.TArr2Raw(settingValues), types.Resource(ResourceMicrosoftSettingValue)),
			})
		if err != nil {
			return nil, err
		}

		settingsList = append(settingsList, settingResource)
	}

	return settingsList, nil
}
