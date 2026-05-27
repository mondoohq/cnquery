// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	azruntime "github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
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

func (a *mqlAzureSubscriptionMonitorService) logProfiles() ([]any, error) {
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
	res := []any{}
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

func (a *mqlAzureSubscriptionMonitorService) diagnosticSettings() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	return getDiagnosticSettings("/subscriptions/"+a.SubscriptionId.Data, a.MqlRuntime, conn)
}

func (a *mqlAzureSubscriptionMonitorService) applicationInsights() ([]any, error) {
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
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAI, err := createApplicationInsightResource(a.MqlRuntime, entry)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAI)
		}
	}
	return res, nil
}

// createApplicationInsightResource maps an Application Insights component to its MQL resource.
func createApplicationInsightResource(runtime *plugin.Runtime, entry *appinsights.Component) (*mqlAzureSubscriptionMonitorServiceApplicationInsight, error) {
	properties, err := convert.JsonToDict(entry.Properties)
	if err != nil {
		return nil, err
	}

	var disableIpMasking bool
	var publicNetworkAccessForIngestion, publicNetworkAccessForQuery string
	var retentionInDays int64
	var workspaceResourceId string
	if entry.Properties != nil {
		if entry.Properties.DisableIPMasking != nil {
			disableIpMasking = *entry.Properties.DisableIPMasking
		}
		if entry.Properties.PublicNetworkAccessForIngestion != nil {
			publicNetworkAccessForIngestion = string(*entry.Properties.PublicNetworkAccessForIngestion)
		}
		if entry.Properties.PublicNetworkAccessForQuery != nil {
			publicNetworkAccessForQuery = string(*entry.Properties.PublicNetworkAccessForQuery)
		}
		if entry.Properties.RetentionInDays != nil {
			retentionInDays = int64(*entry.Properties.RetentionInDays)
		}
		if entry.Properties.WorkspaceResourceID != nil {
			workspaceResourceId = *entry.Properties.WorkspaceResourceID
		}
	}

	mqlAppInsight, err := CreateResource(runtime, "azure.subscription.monitorService.applicationInsight",
		map[string]*llx.RawData{
			"id":                              llx.StringDataPtr(entry.ID),
			"name":                            llx.StringDataPtr(entry.Name),
			"properties":                      llx.DictData(properties),
			"location":                        llx.StringDataPtr(entry.Location),
			"type":                            llx.StringDataPtr(entry.Type),
			"tags":                            llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
			"kind":                            llx.StringDataPtr(entry.Kind),
			"disableIpMasking":                llx.BoolData(disableIpMasking),
			"publicNetworkAccessForIngestion": llx.StringData(publicNetworkAccessForIngestion),
			"publicNetworkAccessForQuery":     llx.StringData(publicNetworkAccessForQuery),
			"retentionInDays":                 llx.IntData(retentionInDays),
			"workspaceResourceId":             llx.StringData(workspaceResourceId),
		})
	if err != nil {
		return nil, err
	}
	mqlAI := mqlAppInsight.(*mqlAzureSubscriptionMonitorServiceApplicationInsight)
	mqlAI.cacheWorkspaceResourceId = workspaceResourceId
	return mqlAI, nil
}

// initAzureSubscriptionMonitorServiceApplicationInsight fetches an Application Insights
// component by its resource ID, enabling typed cross-references from other resources.
func initAzureSubscriptionMonitorServiceApplicationInsight(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	idRaw, ok := args["id"]
	if !ok || idRaw == nil {
		return args, nil, nil
	}
	id, ok := idRaw.Value.(string)
	if !ok || id == "" {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, nil, err
	}
	componentName, err := resourceID.Component("components")
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	client, err := appinsights.NewComponentsClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, nil, err
	}

	resp, err := client.Get(ctx, resourceID.ResourceGroup, componentName, nil)
	if err != nil {
		return nil, nil, err
	}

	mqlAI, err := createApplicationInsightResource(runtime, &resp.Component)
	if err != nil {
		return nil, nil, err
	}
	return nil, mqlAI, nil
}

func (a *mqlAzureSubscriptionMonitorServiceActivityLog) alerts() ([]any, error) {
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
	res := []any{}
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
					ActionGroupId:     convert.ToValue(act.ActionGroupID),
					WebhookProperties: convert.PtrMapStrToStr(act.WebhookProperties),
				}
				actions = append(actions, mqlAction)
			}
			for _, cond := range entry.Properties.Condition.AllOf {
				anyOf := []mqlAlertLeafCondition{}
				for _, leaf := range cond.AnyOf {
					mqlAnyOfLeaf := mqlAlertLeafCondition{
						FieldName:   convert.ToValue(leaf.Field),
						Equals:      convert.ToValue(leaf.Equals),
						ContainsAny: convert.SliceStrPtrToStr(leaf.ContainsAny),
					}
					anyOf = append(anyOf, mqlAnyOfLeaf)
				}
				mqlCondition := mqlAlertCondition{
					FieldName:   convert.ToValue(cond.Field),
					Equals:      convert.ToValue(cond.Equals),
					ContainsAny: convert.SliceStrPtrToStr(cond.ContainsAny),
					AnyOf:       anyOf,
				}
				conditions = append(conditions, mqlCondition)
			}

			actionsDict := []any{}
			for _, a := range actions {
				dict, err := convert.JsonToDict(a)
				if err != nil {
					return nil, err
				}
				actionsDict = append(actionsDict, dict)
			}
			conditionsDict := []any{}
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

// diagnosticSettingsResource mirrors the fields of the
// Microsoft.Insights/diagnosticSettings list response that we surface. armmonitor
// dropped its typed DiagnosticSettings client in v0.12.0, so we call the REST API
// directly through the ARM pipeline and decode the relevant fields here.
type diagnosticSettingsResource struct {
	ID         *string        `json:"id"`
	Name       *string        `json:"name"`
	Type       *string        `json:"type"`
	Properties map[string]any `json:"properties"`
}

func getDiagnosticSettings(id string, runtime *plugin.Runtime, conn *connection.AzureConnection) ([]any, error) {
	ctx := context.Background()
	client, err := arm.NewClient("azure.subscription.monitorService.diagnosticSettings", "v1.0.0", conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	// id is an ARM resource URI (e.g. "/subscriptions/<id>" or a full resource ID
	// returned by Azure). It is a multi-segment path, so it is joined raw rather than
	// URL-escaped — escaping would turn the "/" separators into %2F. This matches the
	// removed armmonitor DiagnosticSettingsClient, which substituted {resourceUri}
	// without escaping for the same reason.
	urlPath := azruntime.JoinPaths(client.Endpoint(), id, "/providers/Microsoft.Insights/diagnosticSettings")
	req, err := azruntime.NewRequest(ctx, http.MethodGet, urlPath)
	if err != nil {
		return nil, err
	}
	query := req.Raw().URL.Query()
	query.Set("api-version", "2021-05-01-preview")
	req.Raw().URL.RawQuery = query.Encode()

	resp, err := client.Pipeline().Do(req)
	if err != nil {
		return nil, err
	}
	if !azruntime.HasStatusCode(resp, http.StatusOK) {
		return nil, azruntime.NewResponseError(resp)
	}

	var result struct {
		Value []diagnosticSettingsResource `json:"value"`
	}
	if err := azruntime.UnmarshalAsJSON(resp, &result); err != nil {
		return nil, err
	}

	res := []any{}
	for _, entry := range result.Value {
		properties := entry.Properties
		if properties == nil {
			properties = map[string]any{}
		}

		// "storageAccountId" is the camelCase key defined by the diagnosticSettings
		// API contract (it was the json tag on the SDK's DiagnosticSettings.StorageAccountID
		// field). Azure returns response keys with stable casing, so a direct lookup is safe.
		var storageAccountId *string
		if v, ok := properties["storageAccountId"].(string); ok {
			storageAccountId = &v
		}

		mqlAzure, err := CreateResource(runtime, "azure.subscription.monitorService.diagnosticsetting",
			map[string]*llx.RawData{
				"id":               llx.StringDataPtr(entry.ID),
				"name":             llx.StringDataPtr(entry.Name),
				"type":             llx.StringDataPtr(entry.Type),
				"properties":       llx.DictData(properties),
				"storageAccountId": llx.StringDataPtr(storageAccountId),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAzure)
	}

	return res, nil
}
