package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	monitor "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzureMonitor) id() (string, error) {
	return "azure.monitor", nil
}

func (a *mqlAzureMonitor) GetLogProfiles() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := monitor.NewLogProfilesClient(at.SubscriptionID(), token, &arm.ClientOptions{})
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

			properties, err := core.JsonToDict(entry.Properties)
			if err != nil {
				return nil, err
			}

			var mqlAzureStorageAccount interface{}
			if entry.Properties != nil && entry.Properties.StorageAccountID != nil {
				// the resource fetches the data itself
				mqlAzureStorageAccount, err = a.MotorRuntime.CreateResource("azure.storage.account",
					"id", core.ToString(entry.Properties.StorageAccountID),
				)
				if err != nil {
					return nil, err
				}
			}

			mqlAzure, err := a.MotorRuntime.CreateResource("azure.monitor.logprofile",
				"id", core.ToString(entry.ID),
				"name", core.ToString(entry.Name),
				"location", core.ToString(entry.Location),
				"type", core.ToString(entry.Type),
				"tags", azureTagsToInterface(entry.Tags),
				"properties", properties,
				"storageAccount", mqlAzureStorageAccount,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}
	return res, nil
}

func (a *mqlAzureMonitor) GetDiagnosticSettings() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	return diagnosticsSettings(a.MotorRuntime, "/subscriptions/"+at.SubscriptionID())
}

func (a *mqlAzureMonitor) GetActivityLog() (interface{}, error) {
	return a.MotorRuntime.CreateResource("azure.monitor.activitylog")
}

func (a *mqlAzureMonitorActivitylog) id() (string, error) {
	return "azure.monitor.activitylog", nil
}

func (a *mqlAzureMonitorActivitylogAlert) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureMonitorActivitylog) GetAlerts() ([]interface{}, error) {
	// fetch the details
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := monitor.NewActivityLogAlertsClient(at.SubscriptionID(), token, &arm.ClientOptions{})
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
		for _, entry := range page.Value {
			actions := []interface{}{}
			conditions := []interface{}{}

			for _, act := range entry.Properties.Actions.ActionGroups {
				action, err := a.MotorRuntime.CreateResource("azure.monitor.activitylog.alert.action",
					"actionGroupId", core.ToString(act.ActionGroupID),
					"webhookProperties", azureTagsToInterface(act.WebhookProperties),
				)
				if err != nil {
					return nil, err
				}
				actions = append(actions, action)
			}
			for idx, cond := range entry.Properties.Condition.AllOf {
				anyOf := []interface{}{}
				for childIdx, leaf := range cond.AnyOf {
					cond, err := a.MotorRuntime.CreateResource("azure.monitor.activitylog.alert.condition",
						"id", fmt.Sprintf("%s/condition/%d/anyOf/%d", *entry.ID, idx, childIdx),
						"fieldName", core.ToString(leaf.Field),
						"equals", core.ToString(leaf.Equals),
						"containsAny", core.PtrSliceToInterface(leaf.ContainsAny),
					)
					if err != nil {
						return nil, err
					}
					anyOf = append(anyOf, cond)
				}
				cond, err := a.MotorRuntime.CreateResource("azure.monitor.activitylog.alert.condition",
					"id", fmt.Sprintf("%s/condition/%d", *entry.ID, idx),
					"fieldName", core.ToString(cond.Field),
					"equals", core.ToString(cond.Equals),
					"containsAny", core.PtrSliceToInterface(cond.ContainsAny),
					"anyOf", anyOf,
				)
				if err != nil {
					return nil, err
				}
				conditions = append(conditions, cond)
			}
			alert, err := a.MotorRuntime.CreateResource("azure.monitor.activitylog.alert",
				"conditions", conditions,
				"id", core.ToString(entry.ID),
				"name", core.ToString(entry.Name),
				"actions", actions,
				"description", core.ToString(entry.Properties.Description),
				"scopes", core.PtrSliceToInterface(entry.Properties.Scopes),
				"type", core.ToString(entry.Type),
				"tags", azureTagsToInterface(entry.Tags),
				"location", core.ToString(entry.Location),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, alert)
		}
	}
	return res, nil
}

func (a *mqlAzureMonitorLogprofile) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureMonitorActivitylogAlertCondition) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureMonitorActivitylogAlertAction) id() (string, error) {
	return a.ActionGroupId()
}

func diagnosticsSettings(runtime *resources.Runtime, id string) ([]interface{}, error) {
	// fetch the details
	at, err := azureTransport(runtime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := monitor.NewDiagnosticSettingsClient(token, &arm.ClientOptions{})
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
			properties, err := core.JsonToDict(entry.Properties)
			if err != nil {
				return nil, err
			}

			var mqlAzureStorageAccount interface{}
			if entry.Properties != nil && entry.Properties.StorageAccountID != nil {
				// the resource fetches the data itself
				mqlAzureStorageAccount, err = runtime.CreateResource("azure.storage.account",
					"id", core.ToString(entry.Properties.StorageAccountID),
				)
				if err != nil {
					return nil, err
				}
			}

			mqlAzure, err := runtime.CreateResource("azure.monitor.diagnosticsetting",
				"id", core.ToString(entry.ID),
				"name", core.ToString(entry.Name),
				"type", core.ToString(entry.Type),
				"properties", properties,
				"storageAccount", mqlAzureStorageAccount,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzureMonitorDiagnosticsetting) id() (string, error) {
	return a.Id()
}
