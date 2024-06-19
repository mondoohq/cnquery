// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/azure/connection"
	"go.mondoo.com/cnquery/v11/types"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	appinsights "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/applicationinsights/armapplicationinsights"
	monitor "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
)

func (a *mqlAzureSubscriptionMonitorService) id() (string, error) {
	return "azure.subscription.monitor/" + a.SubscriptionId.Data, nil
}

func (a *mqlAzureSubscriptionMonitorServiceActivityLog) id() (string, error) {
	return "azure.subscription.monitorService.activityLog/" + a.SubscriptionId.Data, nil
}

func (a *mqlAzureSubscriptionMonitorServiceActivityLogAlert) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionMonitorServiceApplicationInsight) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionMonitorServiceDiagnosticsetting) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionMonitorServiceLogprofile) id() (string, error) {
	return a.Id.Data, nil
}

func initAzureSubscriptionMonitorService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionMonitorService) activityLog() (*mqlAzureSubscriptionMonitorServiceActivityLog, error) {
	res, err := CreateResource(a.MqlRuntime, "azure.subscription.monitorService.activityLog", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionMonitorServiceActivityLog), nil
}

func initAzureSubscriptionMonitorServiceActivityLog(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionMonitorService) logProfiles() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	client, err := monitor.NewLogProfilesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(&monitor.LogProfilesClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {

			properties, err := convert.JsonToDict(entry.Properties)
			if err != nil {
				return nil, err
			}

			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.monitorService.logprofile",
				map[string]*llx.RawData{
					"id":               llx.StringDataPtr(entry.ID),
					"name":             llx.StringDataPtr(entry.Name),
					"location":         llx.StringDataPtr(entry.Location),
					"type":             llx.StringDataPtr(entry.Type),
					"tags":             llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
					"properties":       llx.DictData(properties),
					"storageAccountId": llx.StringDataPtr(entry.Properties.StorageAccountID),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionMonitorService) diagnosticSettings() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	return getDiagnosticSettings("/subscriptions/"+a.SubscriptionId.Data, a.MqlRuntime, conn)
}

func (a *mqlAzureSubscriptionMonitorService) applicationInsights() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := appinsights.NewComponentsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(&appinsights.ComponentsClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			properties, err := convert.JsonToDict(entry.Properties)
			if err != nil {
				return nil, err
			}

			mqlAppInsight, err := CreateResource(a.MqlRuntime, "azure.subscription.monitorService.applicationInsight",
				map[string]*llx.RawData{
					"id":         llx.StringDataPtr(entry.ID),
					"name":       llx.StringDataPtr(entry.Name),
					"properties": llx.DictData(properties),
					"location":   llx.StringDataPtr(entry.Location),
					"type":       llx.StringDataPtr(entry.Type),
					"tags":       llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
					"kind":       llx.StringDataPtr(entry.Kind),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAppInsight)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionMonitorServiceActivityLog) alerts() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	client, err := monitor.NewActivityLogAlertsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := client.NewListBySubscriptionIDPager(&monitor.ActivityLogAlertsClientListBySubscriptionIDOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		type mqlAlertAction struct {
			ActionGroupId     string            `json:"actionGroupId"`
			WebhookProperties map[string]string `json:"webhookProperties"`
		}

		type mqlAlertLeafCondition struct {
			FieldName   string   `json:"fieldName"`
			Equals      string   `json:"equals"`
			ContainsAny []string `json:"containsAny"`
		}

		type mqlAlertCondition struct {
			FieldName   string                  `json:"fieldName"`
			Equals      string                  `json:"equals"`
			ContainsAny []string                `json:"containsAny"`
			AnyOf       []mqlAlertLeafCondition `json:"anyOf"`
		}

		for _, entry := range page.Value {
			actions := []mqlAlertAction{}
			conditions := []mqlAlertCondition{}

			for _, act := range entry.Properties.Actions.ActionGroups {
				mqlAction := mqlAlertAction{
					ActionGroupId:     convert.ToString(act.ActionGroupID),
					WebhookProperties: convert.PtrMapStrToStr(act.WebhookProperties),
				}
				actions = append(actions, mqlAction)
			}
			for _, cond := range entry.Properties.Condition.AllOf {
				anyOf := []mqlAlertLeafCondition{}
				for _, leaf := range cond.AnyOf {
					mqlAnyOfLeaf := mqlAlertLeafCondition{
						FieldName:   convert.ToString(leaf.Field),
						Equals:      convert.ToString(leaf.Equals),
						ContainsAny: convert.SliceStrPtrToStr(leaf.ContainsAny),
					}
					anyOf = append(anyOf, mqlAnyOfLeaf)
				}
				mqlCondition := mqlAlertCondition{
					FieldName:   convert.ToString(cond.Field),
					Equals:      convert.ToString(cond.Equals),
					ContainsAny: convert.SliceStrPtrToStr(cond.ContainsAny),
					AnyOf:       anyOf,
				}
				conditions = append(conditions, mqlCondition)
			}

			actionsDict := []interface{}{}
			for _, a := range actions {
				dict, err := convert.JsonToDict(a)
				if err != nil {
					return nil, err
				}
				actionsDict = append(actionsDict, dict)
			}
			conditionsDict := []interface{}{}
			for _, c := range conditions {
				dict, err := convert.JsonToDict(c)
				if err != nil {
					return nil, err
				}
				conditionsDict = append(conditionsDict, dict)
			}
			alert, err := CreateResource(a.MqlRuntime, "azure.subscription.monitorService.activityLog.alert",
				map[string]*llx.RawData{
					"id":          llx.StringDataPtr(entry.ID),
					"name":        llx.StringDataPtr(entry.Name),
					"actions":     llx.DictData(actionsDict),
					"conditions":  llx.DictData(conditionsDict),
					"description": llx.StringDataPtr(entry.Properties.Description),
					"scopes":      llx.ArrayData(convert.SliceStrPtrToInterface(entry.Properties.Scopes), types.String),
					"type":        llx.StringDataPtr(entry.Type),
					"tags":        llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
					"location":    llx.StringDataPtr(entry.Location),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, alert)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionMonitorServiceLogprofile) storageAccount() (*mqlAzureSubscriptionStorageServiceAccount, error) {
	if a.StorageAccountId.IsNull() {
		return nil, errors.New("diagnostic settings has no storage account")
	}
	if a.StorageAccountId.Error != nil {
		return nil, a.StorageAccountId.Error
	}
	storageAccId := a.StorageAccountId.Data
	if storageAccId == "" {
		return nil, errors.New("diagnostic settings has no storage account")
	}
	return getStorageAccount(storageAccId, a.MqlRuntime, a.MqlRuntime.Connection.(*connection.AzureConnection))
}

func (a *mqlAzureSubscriptionMonitorServiceDiagnosticsetting) storageAccount() (*mqlAzureSubscriptionStorageServiceAccount, error) {
	if a.StorageAccountId.IsNull() {
		return nil, errors.New("diagnostic settings has no storage account")
	}
	if a.StorageAccountId.Error != nil {
		return nil, a.StorageAccountId.Error
	}
	storageAccId := a.StorageAccountId.Data
	if storageAccId == "" {
		return nil, errors.New("diagnostic settings has no storage account")
	}
	return getStorageAccount(storageAccId, a.MqlRuntime, a.MqlRuntime.Connection.(*connection.AzureConnection))
}

func getDiagnosticSettings(id string, runtime *plugin.Runtime, conn *connection.AzureConnection) ([]interface{}, error) {
	ctx := context.Background()
	token := conn.Token()
	client, err := monitor.NewDiagnosticSettingsClient(token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := client.NewListPager(id, &monitor.DiagnosticSettingsClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			properties, err := convert.JsonToDict(entry.Properties)
			if err != nil {
				return nil, err
			}

			mqlAzure, err := CreateResource(runtime, "azure.subscription.monitorService.diagnosticsetting",
				map[string]*llx.RawData{
					"id":               llx.StringDataPtr(entry.ID),
					"name":             llx.StringDataPtr(entry.Name),
					"type":             llx.StringDataPtr(entry.Type),
					"properties":       llx.DictData(properties),
					"storageAccountId": llx.StringDataPtr(entry.Properties.StorageAccountID),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}
