// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"reflect"

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
	settingsList := make([]any, len(settings.GetValue()))
	rawSettings := settings.GetValue()
	fmt.Printf("Total settings count: %d\n", len(rawSettings))

	for i := range rawSettings {
		setting := rawSettings[i]
		fmt.Printf("\n=== Processing Setting [%d] ===\n", i)
		fmt.Printf("Setting type: %T\n", setting)

		// Use reflection to inspect the setting
		v := reflect.ValueOf(setting)
		t := reflect.TypeOf(setting)
		fmt.Printf("  Setting struct type: %s\n", t)

		// Call GetDisplayName using reflection
		displayNameMethod := v.MethodByName("GetDisplayName")
		if displayNameMethod.IsValid() {
			displayNameResult := displayNameMethod.Call(nil)
			if len(displayNameResult) > 0 {
				if !displayNameResult[0].IsNil() {
					fmt.Printf("  DisplayName: %s\n", *displayNameResult[0].Interface().(*string))
				}
			}
		}

		// Call GetTemplateId using reflection
		templateIdMethod := v.MethodByName("GetTemplateId")
		if templateIdMethod.IsValid() {
			templateIdResult := templateIdMethod.Call(nil)
			if len(templateIdResult) > 0 {
				if !templateIdResult[0].IsNil() {
					fmt.Printf("  TemplateId: %s\n", *templateIdResult[0].Interface().(*string))
				}
			}
		}

		// Call GetValues using reflection
		valuesMethod := v.MethodByName("GetValues")
		if valuesMethod.IsValid() {
			valuesResult := valuesMethod.Call(nil)
			if len(valuesResult) > 0 && !valuesResult[0].IsNil() {
				valuesArray := valuesResult[0].Interface()
				valuesLength := reflect.ValueOf(valuesArray).Len()
				fmt.Printf("  Values count: %d\n", valuesLength)
			}
		}
		// Create settingValue resources for each value
		// IMPORTANT: Create a fresh values slice for each iteration
		values := make([]any, 0, len(setting.GetValues()))
		entries := setting.GetValues()
		for j := range entries {
			name := entries[j].GetName()
			value := entries[j].GetValue()
			if name != nil && value != nil {
				settingValueResource, err := CreateResource(a.MqlRuntime, "microsoft.settingValue",
					map[string]*llx.RawData{
						"__id":  llx.StringData(*name + "|" + *value),
						"name":  llx.StringData(*name),
						"value": llx.StringData(*value),
					})
				if err != nil {
					return nil, err
				}
				values = append(values, settingValueResource)
			}
		}

		// Copy the values to avoid pointer reuse
		displayNamePtr := rawSettings[i].GetDisplayName()
		templateIdPtr := rawSettings[i].GetTemplateId()

		if displayNamePtr == nil || templateIdPtr == nil {
			continue
		}

		// Dereference and copy to avoid pointer sharing
		displayName := *displayNamePtr
		templateId := *templateIdPtr

		fmt.Printf("Creating resource [%d]: displayName=%s, values count=%d\n", i, displayName, len(values))

		// Create a unique mqlId by combining displayName and templateId
		mqlId := displayName + "|" + templateId

		settingResource, err := CreateResource(a.MqlRuntime, "microsoft.setting",
			map[string]*llx.RawData{
				"__id":        llx.StringData(mqlId),
				"displayName": llx.StringData(displayName),
				"templateId":  llx.StringData(templateId),
				"values":      llx.ArrayData(llx.TArr2Raw(values), types.Resource("microsoft.settingValue")),
			})
		if err != nil {
			return nil, err
		}

		// TODO: with reflection, println the settingResource

		settingsList[i] = settingResource
	}

	return settingsList, nil
}
