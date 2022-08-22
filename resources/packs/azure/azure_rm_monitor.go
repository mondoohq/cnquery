package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/2019-03-01/resources/mgmt/insights"
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (a *mqlAzurermMonitor) id() (string, error) {
	return "azurerm.monitor", nil
}

func (a *mqlAzurermMonitor) GetLogProfiles() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := insights.NewLogProfilesClient(at.SubscriptionID())
	client.Authorizer = authorizer

	logProfiles, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	if logProfiles.Value == nil {
		return res, nil
	}

	list := *logProfiles.Value

	for i := range list {
		entry := list[i]

		properties, err := core.JsonToDict(entry.LogProfileProperties)
		if err != nil {
			return nil, err
		}

		var mqlAzureStorageAccount interface{}
		if entry.LogProfileProperties != nil && entry.LogProfileProperties.StorageAccountID != nil {
			// the resource fetches the data itself
			mqlAzureStorageAccount, err = a.MotorRuntime.CreateResource("azurerm.storage.account",
				"id", core.ToString(entry.LogProfileProperties.StorageAccountID),
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

	return res, nil
}

func (a *mqlAzurermMonitorLogprofile) id() (string, error) {
	return a.Id()
}

func diagnosticsSettings(runtime *resources.Runtime, id string) ([]interface{}, error) {
	// fetch the details
	at, err := azuretransport(runtime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := insights.NewDiagnosticSettingsClient(at.SubscriptionID())
	client.Authorizer = authorizer
	diagnosticSettings, err := client.List(ctx, id)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	if diagnosticSettings.Value == nil {
		return res, nil
	}

	list := *diagnosticSettings.Value

	for i := range list {
		entry := list[i]

		properties, err := core.JsonToDict(entry.DiagnosticSettings)
		if err != nil {
			return nil, err
		}

		var mqlAzureStorageAccount interface{}
		if entry.DiagnosticSettings != nil && entry.DiagnosticSettings.StorageAccountID != nil {
			// the resource fetches the data itself
			mqlAzureStorageAccount, err = runtime.CreateResource("azurerm.storage.account",
				"id", core.ToString(entry.DiagnosticSettings.StorageAccountID),
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

	return res, nil
}

func (a *mqlAzurermMonitorDiagnosticsetting) id() (string, error) {
	return a.Id()
}
