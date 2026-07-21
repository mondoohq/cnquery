// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"

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
			if entry == nil || entry.Properties == nil {
				continue
			}

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
			sysData, err := convert.JsonToDict(entry.SystemData)
			if err != nil {
				return nil, err
			}
			mqlAzure.(*mqlAzureSubscriptionMonitorServiceLogprofile).cacheSystemData = sysData
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
	var creationTime *time.Time
	if entry.Properties != nil {
		creationTime = entry.Properties.CreationDate
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
			"creationTime":                    llx.TimeDataPtr(creationTime),
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
			if entry == nil || entry.Properties == nil {
				continue
			}
			actions := []mqlAlertAction{}
			conditions := []mqlAlertCondition{}

			if entry.Properties.Actions != nil {
				for _, act := range entry.Properties.Actions.ActionGroups {
					mqlAction := mqlAlertAction{
						ActionGroupId:     convert.ToValue(act.ActionGroupID),
						WebhookProperties: convert.PtrMapStrToStr(act.WebhookProperties),
					}
					actions = append(actions, mqlAction)
				}
			}
			var allOf []*monitor.AlertRuleAnyOfOrLeafCondition
			if entry.Properties.Condition != nil {
				allOf = entry.Properties.Condition.AllOf
			}
			for _, cond := range allOf {
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
			sysData, err := convert.JsonToDict(entry.SystemData)
			if err != nil {
				return nil, err
			}
			alert.(*mqlAzureSubscriptionMonitorServiceActivityLogAlert).cacheSystemData = sysData
			res = append(res, alert)
		}
	}
	return res, nil
}

type mqlAzureSubscriptionMonitorServiceActionGroupInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionMonitorServiceActionGroup) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionMonitorServiceActionGroup) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionMonitorService) actionGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	client, err := monitor.NewActionGroupsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListBySubscriptionIDPager(&monitor.ActionGroupsClientListBySubscriptionIDOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ag := range page.Value {
			if ag == nil {
				continue
			}
			var groupShortName string
			var enabled *bool
			emailReceivers := []any{}
			smsReceivers := []any{}
			voiceReceivers := []any{}
			webhookReceivers := []any{}
			armRoleReceivers := []any{}
			azureFunctionReceivers := []any{}
			logicAppReceivers := []any{}
			automationRunbookReceivers := []any{}
			if p := ag.Properties; p != nil {
				groupShortName = convert.ToValue(p.GroupShortName)
				enabled = p.Enabled
				if emailReceivers, err = convert.JsonToDictSlice(p.EmailReceivers); err != nil {
					return nil, err
				}
				if smsReceivers, err = convert.JsonToDictSlice(p.SmsReceivers); err != nil {
					return nil, err
				}
				if voiceReceivers, err = convert.JsonToDictSlice(p.VoiceReceivers); err != nil {
					return nil, err
				}
				if webhookReceivers, err = convert.JsonToDictSlice(p.WebhookReceivers); err != nil {
					return nil, err
				}
				if armRoleReceivers, err = convert.JsonToDictSlice(p.ArmRoleReceivers); err != nil {
					return nil, err
				}
				if azureFunctionReceivers, err = convert.JsonToDictSlice(p.AzureFunctionReceivers); err != nil {
					return nil, err
				}
				if logicAppReceivers, err = convert.JsonToDictSlice(p.LogicAppReceivers); err != nil {
					return nil, err
				}
				if automationRunbookReceivers, err = convert.JsonToDictSlice(p.AutomationRunbookReceivers); err != nil {
					return nil, err
				}
			}
			mqlAg, err := CreateResource(a.MqlRuntime, "azure.subscription.monitorService.actionGroup",
				map[string]*llx.RawData{
					"id":                         llx.StringDataPtr(ag.ID),
					"name":                       llx.StringDataPtr(ag.Name),
					"location":                   llx.StringDataPtr(ag.Location),
					"enabled":                    llx.BoolDataPtr(enabled),
					"groupShortName":             llx.StringData(groupShortName),
					"emailReceivers":             llx.ArrayData(emailReceivers, types.Dict),
					"smsReceivers":               llx.ArrayData(smsReceivers, types.Dict),
					"voiceReceivers":             llx.ArrayData(voiceReceivers, types.Dict),
					"webhookReceivers":           llx.ArrayData(webhookReceivers, types.Dict),
					"armRoleReceivers":           llx.ArrayData(armRoleReceivers, types.Dict),
					"azureFunctionReceivers":     llx.ArrayData(azureFunctionReceivers, types.Dict),
					"logicAppReceivers":          llx.ArrayData(logicAppReceivers, types.Dict),
					"automationRunbookReceivers": llx.ArrayData(automationRunbookReceivers, types.Dict),
				})
			if err != nil {
				return nil, err
			}
			sysData, err := convert.JsonToDict(ag.SystemData)
			if err != nil {
				return nil, err
			}
			mqlAg.(*mqlAzureSubscriptionMonitorServiceActionGroup).cacheSystemData = sysData
			res = append(res, mqlAg)
		}
	}
	return res, nil
}

// resolveActionGroups resolves a list of action-group ARM IDs to the typed
// actionGroup resources by filtering the subscription's action groups. ARM IDs
// are matched case-insensitively. Returns an empty list when the rule
// references no action groups.
func resolveActionGroups(runtime *plugin.Runtime, subscriptionId string, actionGroupIds []string) ([]any, error) {
	if len(actionGroupIds) == 0 {
		return []any{}, nil
	}
	msRes, err := CreateResource(runtime, "azure.subscription.monitorService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(subscriptionId),
	})
	if err != nil {
		return nil, err
	}
	agsVal := msRes.(*mqlAzureSubscriptionMonitorService).GetActionGroups()
	if agsVal.Error != nil {
		return nil, agsVal.Error
	}
	want := make(map[string]struct{}, len(actionGroupIds))
	for _, id := range actionGroupIds {
		if id != "" {
			want[strings.ToLower(id)] = struct{}{}
		}
	}
	res := []any{}
	for _, agAny := range agsVal.Data {
		ag, ok := agAny.(*mqlAzureSubscriptionMonitorServiceActionGroup)
		if !ok {
			continue
		}
		if _, ok := want[strings.ToLower(ag.Id.Data)]; ok {
			res = append(res, ag)
		}
	}
	return res, nil
}

type mqlAzureSubscriptionMonitorServiceMetricAlertInternal struct {
	cacheSystemData any
	subscriptionId  string
	actionGroupIds  []string
}

func (a *mqlAzureSubscriptionMonitorServiceMetricAlert) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionMonitorServiceMetricAlert) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionMonitorServiceMetricAlert) actionGroups() ([]any, error) {
	return resolveActionGroups(a.MqlRuntime, a.subscriptionId, a.actionGroupIds)
}

func (a *mqlAzureSubscriptionMonitorService) metricAlerts() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	// The armmonitor SDK pins the metricAlerts client to an api-version
	// (2026-01-01) that the Microsoft.Insights/metricAlerts resource type does
	// not serve, so ListBySubscription 404s with InvalidResourceType. Override
	// it to the newest api-version the service actually supports for this
	// resource type. Scoped to this client only, so sibling Monitor clients
	// (action groups, scheduled query rules) keep their own versions.
	metricAlertOpts := conn.ClientOptions()
	metricAlertOpts.APIVersion = "2024-03-01-preview"
	client, err := monitor.NewMetricAlertsClient(subId, token, &arm.ClientOptions{
		ClientOptions: metricAlertOpts,
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListBySubscriptionPager(&monitor.MetricAlertsClientListBySubscriptionOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ma := range page.Value {
			if ma == nil {
				continue
			}
			var description, evaluationFrequency, windowSize, targetResourceType, targetResourceRegion string
			var enabled, autoMitigate *bool
			var severity int64
			var lastUpdated *time.Time
			scopes := []any{}
			var criteria any
			actionGroupIds := []string{}
			if p := ma.Properties; p != nil {
				description = convert.ToValue(p.Description)
				enabled = p.Enabled
				autoMitigate = p.AutoMitigate
				if p.Severity != nil {
					severity = int64(*p.Severity)
				}
				evaluationFrequency = convert.ToValue(p.EvaluationFrequency)
				windowSize = convert.ToValue(p.WindowSize)
				targetResourceType = convert.ToValue(p.TargetResourceType)
				targetResourceRegion = convert.ToValue(p.TargetResourceRegion)
				lastUpdated = p.LastUpdatedTime
				for _, s := range p.Scopes {
					if s != nil {
						scopes = append(scopes, *s)
					}
				}
				if p.Criteria != nil {
					criteria, err = convert.JsonToDict(p.Criteria)
					if err != nil {
						return nil, err
					}
				}
				for _, act := range p.Actions {
					if act != nil && act.ActionGroupID != nil {
						actionGroupIds = append(actionGroupIds, *act.ActionGroupID)
					}
				}
			}
			mqlMa, err := CreateResource(a.MqlRuntime, "azure.subscription.monitorService.metricAlert", map[string]*llx.RawData{
				"id":                   llx.StringDataPtr(ma.ID),
				"name":                 llx.StringDataPtr(ma.Name),
				"description":          llx.StringData(description),
				"enabled":              llx.BoolDataPtr(enabled),
				"severity":             llx.IntData(severity),
				"scopes":               llx.ArrayData(scopes, types.String),
				"evaluationFrequency":  llx.StringData(evaluationFrequency),
				"windowSize":           llx.StringData(windowSize),
				"autoMitigate":         llx.BoolDataPtr(autoMitigate),
				"targetResourceType":   llx.StringData(targetResourceType),
				"targetResourceRegion": llx.StringData(targetResourceRegion),
				"criteria":             llx.DictData(criteria),
				"lastUpdatedTime":      llx.TimeDataPtr(lastUpdated),
			})
			if err != nil {
				return nil, err
			}
			m := mqlMa.(*mqlAzureSubscriptionMonitorServiceMetricAlert)
			m.subscriptionId = subId
			m.actionGroupIds = actionGroupIds
			sysData, err := convert.JsonToDict(ma.SystemData)
			if err != nil {
				return nil, err
			}
			m.cacheSystemData = sysData
			res = append(res, mqlMa)
		}
	}
	return res, nil
}

type mqlAzureSubscriptionMonitorServiceScheduledQueryRuleInternal struct {
	cacheSystemData any
	subscriptionId  string
	actionGroupIds  []string
}

func (a *mqlAzureSubscriptionMonitorServiceScheduledQueryRule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionMonitorServiceScheduledQueryRule) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionMonitorServiceScheduledQueryRule) actionGroups() ([]any, error) {
	return resolveActionGroups(a.MqlRuntime, a.subscriptionId, a.actionGroupIds)
}

func (a *mqlAzureSubscriptionMonitorService) scheduledQueryRules() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	client, err := monitor.NewScheduledQueryRulesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListBySubscriptionPager(&monitor.ScheduledQueryRulesClientListBySubscriptionOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, sqr := range page.Value {
			if sqr == nil {
				continue
			}
			var displayName, description, evaluationFrequency, windowSize, muteActionsDuration string
			var enabled, autoMitigate *bool
			var severity int64
			scopes := []any{}
			targetResourceTypes := []any{}
			var criteria any
			actionGroupIds := []string{}
			if p := sqr.Properties; p != nil {
				displayName = convert.ToValue(p.DisplayName)
				description = convert.ToValue(p.Description)
				enabled = p.Enabled
				autoMitigate = p.AutoMitigate
				if p.Severity != nil {
					severity = int64(*p.Severity)
				}
				evaluationFrequency = convert.ToValue(p.EvaluationFrequency)
				windowSize = convert.ToValue(p.WindowSize)
				muteActionsDuration = convert.ToValue(p.MuteActionsDuration)
				for _, s := range p.Scopes {
					if s != nil {
						scopes = append(scopes, *s)
					}
				}
				for _, t := range p.TargetResourceTypes {
					if t != nil {
						targetResourceTypes = append(targetResourceTypes, *t)
					}
				}
				if p.Criteria != nil {
					criteria, err = convert.JsonToDict(p.Criteria)
					if err != nil {
						return nil, err
					}
				}
				if p.Actions != nil {
					for _, ag := range p.Actions.ActionGroups {
						if ag != nil {
							actionGroupIds = append(actionGroupIds, *ag)
						}
					}
				}
			}
			mqlSqr, err := CreateResource(a.MqlRuntime, "azure.subscription.monitorService.scheduledQueryRule", map[string]*llx.RawData{
				"id":                  llx.StringDataPtr(sqr.ID),
				"name":                llx.StringDataPtr(sqr.Name),
				"displayName":         llx.StringData(displayName),
				"description":         llx.StringData(description),
				"enabled":             llx.BoolDataPtr(enabled),
				"severity":            llx.IntData(severity),
				"scopes":              llx.ArrayData(scopes, types.String),
				"evaluationFrequency": llx.StringData(evaluationFrequency),
				"windowSize":          llx.StringData(windowSize),
				"muteActionsDuration": llx.StringData(muteActionsDuration),
				"autoMitigate":        llx.BoolDataPtr(autoMitigate),
				"targetResourceTypes": llx.ArrayData(targetResourceTypes, types.String),
				"criteria":            llx.DictData(criteria),
			})
			if err != nil {
				return nil, err
			}
			s := mqlSqr.(*mqlAzureSubscriptionMonitorServiceScheduledQueryRule)
			s.subscriptionId = subId
			s.actionGroupIds = actionGroupIds
			sysData, err := convert.JsonToDict(sqr.SystemData)
			if err != nil {
				return nil, err
			}
			s.cacheSystemData = sysData
			res = append(res, mqlSqr)
		}
	}
	return res, nil
}

type mqlAzureSubscriptionMonitorServiceLogprofileInternal struct {
	cacheSystemData any
}

type mqlAzureSubscriptionMonitorServiceActivityLogAlertInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionMonitorServiceLogprofile) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionMonitorServiceActivityLogAlert) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionMonitorServiceActivityLogEntry) id() (string, error) {
	return a.Id.Data, nil
}

// localizableString returns the human-readable form of an activity log
// LocalizableString, preferring the localized value and falling back to the
// invariant value.
func localizableString(s *monitor.LocalizableString) string {
	if s == nil {
		return ""
	}
	if s.LocalizedValue != nil && *s.LocalizedValue != "" {
		return *s.LocalizedValue
	}
	return convert.ToValue(s.Value)
}

// entries lists the subscription's activity log management events from the last
// seven days. The activity log API requires an eventTimestamp filter, so the
// window is fixed; seven days is the practical horizon for change attribution
// and keeps the response bounded.
func (a *mqlAzureSubscriptionMonitorServiceActivityLog) entries() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	client, err := monitor.NewActivityLogsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	end := time.Now().UTC()
	start := end.AddDate(0, 0, -7)
	filter := fmt.Sprintf("eventTimestamp ge '%s' and eventTimestamp le '%s'",
		start.Format(time.RFC3339), end.Format(time.RFC3339))

	pager := client.NewListPager(filter, &monitor.ActivityLogsClientListOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlEntry, err := newMqlActivityLogEntry(a.MqlRuntime, entry)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlEntry)
		}
	}
	return res, nil
}

// activityLogEntryID returns a stable, unique cache key for an activity log
// event. EventDataID is the natural unique id, but it (and ID) can be nil for
// some platform-emitted events; fall back to ID, then to a composite of the
// operation/correlation ids and timestamp so distinct events never collide.
func activityLogEntryID(entry *monitor.EventData) string {
	if entry.EventDataID != nil && *entry.EventDataID != "" {
		return *entry.EventDataID
	}
	if entry.ID != nil && *entry.ID != "" {
		return *entry.ID
	}
	var ts string
	if entry.EventTimestamp != nil {
		ts = entry.EventTimestamp.UTC().Format(time.RFC3339Nano)
	}
	return fmt.Sprintf("%s/%s/%s", convert.ToValue(entry.OperationID), convert.ToValue(entry.CorrelationID), ts)
}

func newMqlActivityLogEntry(runtime *plugin.Runtime, entry *monitor.EventData) (*mqlAzureSubscriptionMonitorServiceActivityLogEntry, error) {
	authorization, err := convert.JsonToDict(entry.Authorization)
	if err != nil {
		return nil, err
	}
	httpRequest, err := convert.JsonToDict(entry.HTTPRequest)
	if err != nil {
		return nil, err
	}

	var level string
	if entry.Level != nil {
		level = string(*entry.Level)
	}

	id := activityLogEntryID(entry)
	res, err := CreateResource(runtime, "azure.subscription.monitorService.activityLog.entry",
		map[string]*llx.RawData{
			"__id":                 llx.StringData(id),
			"id":                   llx.StringData(id),
			"eventTimestamp":       llx.TimeDataPtr(entry.EventTimestamp),
			"operationName":        llx.StringData(localizableString(entry.OperationName)),
			"caller":               llx.StringDataPtr(entry.Caller),
			"level":                llx.StringData(level),
			"status":               llx.StringData(localizableString(entry.Status)),
			"subStatus":            llx.StringData(localizableString(entry.SubStatus)),
			"resourceId":           llx.StringDataPtr(entry.ResourceID),
			"resourceGroupName":    llx.StringDataPtr(entry.ResourceGroupName),
			"resourceType":         llx.StringData(localizableString(entry.ResourceType)),
			"resourceProviderName": llx.StringData(localizableString(entry.ResourceProviderName)),
			"correlationId":        llx.StringDataPtr(entry.CorrelationID),
			"operationId":          llx.StringDataPtr(entry.OperationID),
			"category":             llx.StringData(localizableString(entry.Category)),
			"description":          llx.StringDataPtr(entry.Description),
			"claims":               llx.MapData(convert.PtrMapStrToInterface(entry.Claims), types.String),
			"authorization":        llx.DictData(authorization),
			"httpRequest":          llx.DictData(httpRequest),
			"properties":           llx.MapData(convert.PtrMapStrToInterface(entry.Properties), types.String),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionMonitorServiceActivityLogEntry), nil
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

// diagnosticLogSettings normalizes a diagnostic setting's log categories into
// the raw dict slice plus the list of categories (or category groups) that are
// enabled, which is what most logging-coverage audits actually check.
func diagnosticLogSettings(props *monitor.DiagnosticSettings) ([]any, []any) {
	entries := []any{}
	enabled := []any{}
	if props == nil {
		return entries, enabled
	}
	for _, l := range props.Logs {
		if l == nil {
			continue
		}
		if d, err := convert.JsonToDict(l); err == nil && d != nil {
			entries = append(entries, d)
		}
		if l.Enabled == nil || !*l.Enabled {
			continue
		}
		if l.Category != nil && *l.Category != "" {
			enabled = append(enabled, *l.Category)
		} else if l.CategoryGroup != nil && *l.CategoryGroup != "" {
			enabled = append(enabled, *l.CategoryGroup)
		}
	}
	return entries, enabled
}

// diagnosticMetricSettings mirrors diagnosticLogSettings for metric categories
// (metric settings carry no category group).
func diagnosticMetricSettings(props *monitor.DiagnosticSettings) ([]any, []any) {
	entries := []any{}
	enabled := []any{}
	if props == nil {
		return entries, enabled
	}
	for _, m := range props.Metrics {
		if m == nil {
			continue
		}
		if d, err := convert.JsonToDict(m); err == nil && d != nil {
			entries = append(entries, d)
		}
		if m.Enabled == nil || !*m.Enabled {
			continue
		}
		if m.Category != nil && *m.Category != "" {
			enabled = append(enabled, *m.Category)
		}
	}
	return entries, enabled
}

// getDiagnosticSettingsCategories lists the log and metric categories a resource
// is capable of emitting, independent of which are actively collected. Comparing
// this against a resource's enabled diagnostic-setting categories reveals
// telemetry that could be collected but isn't.
func getDiagnosticSettingsCategories(id string, runtime *plugin.Runtime, conn *connection.AzureConnection) ([]any, error) {
	ctx := context.Background()
	client, err := monitor.NewDiagnosticSettingsCategoryClient(conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(id, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			if entry == nil {
				continue
			}

			categoryType := ""
			groups := []any{}
			if p := entry.Properties; p != nil {
				if p.CategoryType != nil {
					categoryType = string(*p.CategoryType)
				}
				for _, g := range p.CategoryGroups {
					if g != nil {
						groups = append(groups, *g)
					}
				}
			}

			mqlCategory, err := CreateResource(runtime, "azure.subscription.monitorService.diagnosticSettingsCategory",
				map[string]*llx.RawData{
					"__id":           llx.StringDataPtr(entry.ID),
					"name":           llx.StringDataPtr(entry.Name),
					"categoryType":   llx.StringData(categoryType),
					"categoryGroups": llx.ArrayData(groups, types.String),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlCategory)
		}
	}

	return res, nil
}

func getDiagnosticSettings(id string, runtime *plugin.Runtime, conn *connection.AzureConnection) ([]any, error) {
	ctx := context.Background()
	client, err := monitor.NewDiagnosticSettingsClient(conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	// id is an ARM resource URI (e.g. "/subscriptions/<id>" or a full resource ID).
	// The client substitutes it into the {resourceUri} path segment.
	pager := client.NewListPager(id, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			if entry == nil {
				continue
			}

			properties, err := convert.JsonToDict(entry.Properties)
			if err != nil {
				return nil, err
			}

			var storageAccountId, workspaceId, eventHubName, eventHubAuthorizationRuleId, logAnalyticsDestinationType *string
			if p := entry.Properties; p != nil {
				storageAccountId = p.StorageAccountID
				workspaceId = p.WorkspaceID
				eventHubName = p.EventHubName
				eventHubAuthorizationRuleId = p.EventHubAuthorizationRuleID
				logAnalyticsDestinationType = p.LogAnalyticsDestinationType
			}

			// logs / metrics are arrays of per-category settings. Surface the raw
			// entries as dicts and also derive the set of enabled categories, which
			// is what most logging-coverage audits actually check.
			logs, enabledLogCategories := diagnosticLogSettings(entry.Properties)
			metrics, enabledMetricCategories := diagnosticMetricSettings(entry.Properties)

			mqlAzure, err := CreateResource(runtime, "azure.subscription.monitorService.diagnosticsetting",
				map[string]*llx.RawData{
					"id":                          llx.StringDataPtr(entry.ID),
					"name":                        llx.StringDataPtr(entry.Name),
					"type":                        llx.StringDataPtr(entry.Type),
					"properties":                  llx.DictData(properties),
					"storageAccountId":            llx.StringDataPtr(storageAccountId),
					"workspaceId":                 llx.StringData(convert.ToValue(workspaceId)),
					"eventHubName":                llx.StringData(convert.ToValue(eventHubName)),
					"eventHubAuthorizationRuleId": llx.StringData(convert.ToValue(eventHubAuthorizationRuleId)),
					"logAnalyticsDestinationType": llx.StringData(convert.ToValue(logAnalyticsDestinationType)),
					"enabledLogCategories":        llx.ArrayData(enabledLogCategories, types.String),
					"enabledMetricCategories":     llx.ArrayData(enabledMetricCategories, types.String),
					"logs":                        llx.ArrayData(logs, types.Dict),
					"metrics":                     llx.ArrayData(metrics, types.Dict),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}
