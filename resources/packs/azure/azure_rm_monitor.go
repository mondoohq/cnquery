package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	monitor "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzurermMonitor) id() (string, error) {
	return "azurerm.monitor", nil
}

func (a *mqlAzurermMonitor) GetLogProfiles() ([]interface{}, error) {
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
				mqlAzureStorageAccount, err = a.MotorRuntime.CreateResource("azurerm.storage.account",
					"id", core.ToString(entry.Properties.StorageAccountID),
				)
				if err != nil {
					return nil, err
				}
			}

			mqlAzure, err := a.MotorRuntime.CreateResource("azurerm.monitor.logprofile",
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

func (a *mqlAzurermMonitorLogprofile) id() (string, error) {
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
				mqlAzureStorageAccount, err = runtime.CreateResource("azurerm.storage.account",
					"id", core.ToString(entry.Properties.StorageAccountID),
				)
				if err != nil {
					return nil, err
				}
			}

			mqlAzure, err := runtime.CreateResource("azurerm.monitor.diagnosticsetting",
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

func (a *mqlAzurermMonitorDiagnosticsetting) id() (string, error) {
	return a.Id()
}
