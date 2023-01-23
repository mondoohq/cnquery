package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	monitor "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzureSubscriptionMonitorService) id() (string, error) {
	return "azure.monitor", nil
}

func (a *mqlAzureSubscriptionMonitorService) GetLogProfiles() ([]interface{}, error) {
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

func (a *mqlAzureSubscriptionMonitorService) GetDiagnosticSettings() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	return diagnosticsSettings(a.MotorRuntime, "/subscriptions/"+at.SubscriptionID())
}

func (a *mqlAzureSubscriptionMonitorService) GetActivityLog() (interface{}, error) {
	return a.MotorRuntime.CreateResource("azure.monitor.activitylog")
}

func (a *mqlAzureSubscriptionMonitorServiceActivitylog) id() (string, error) {
	return "azure.monitor.activitylog", nil
}

func (a *mqlAzureSubscriptionMonitorServiceActivitylogAlert) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureSubscriptionMonitorServiceActivitylog) GetAlerts() ([]interface{}, error) {
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
					ActionGroupId:     core.ToString(act.ActionGroupID),
					WebhookProperties: core.PtrMapSliceToStr(act.WebhookProperties),
				}
				actions = append(actions, mqlAction)
			}
			for _, cond := range entry.Properties.Condition.AllOf {
				anyOf := []mqlAlertLeafCondition{}
				for _, leaf := range cond.AnyOf {
					mqlAnyOfLeaf := mqlAlertLeafCondition{
						FieldName:   core.ToString(leaf.Field),
						Equals:      core.ToString(leaf.Equals),
						ContainsAny: core.PtrStrSliceToStr(leaf.ContainsAny),
					}
					anyOf = append(anyOf, mqlAnyOfLeaf)
				}
				mqlCondition := mqlAlertCondition{
					FieldName:   core.ToString(cond.Field),
					Equals:      core.ToString(cond.Equals),
					ContainsAny: core.PtrStrSliceToStr(cond.ContainsAny),
					AnyOf:       anyOf,
				}
				conditions = append(conditions, mqlCondition)
			}

			actionsDict := []interface{}{}
			for _, a := range actions {
				dict, err := core.JsonToDict(a)
				if err != nil {
					return nil, err
				}
				actionsDict = append(actionsDict, dict)
			}
			conditionsDict := []interface{}{}
			for _, c := range conditions {
				dict, err := core.JsonToDict(c)
				if err != nil {
					return nil, err
				}
				conditionsDict = append(conditionsDict, dict)
			}
			alert, err := a.MotorRuntime.CreateResource("azure.monitor.activitylog.alert",
				"conditions", conditionsDict,
				"id", core.ToString(entry.ID),
				"name", core.ToString(entry.Name),
				"actions", actionsDict,
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

func (a *mqlAzureSubscriptionMonitorServiceLogprofile) id() (string, error) {
	return a.Id()
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

func (a *mqlAzureSubscriptionMonitorServiceDiagnosticsetting) id() (string, error) {
	return a.Id()
}
