package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/2019-03-01/resources/mgmt/insights"
	"go.mondoo.io/mondoo/lumi"
)

func (a *lumiAzurermMonitor) id() (string, error) {
	return "azurerm.monitor", nil
}

func (a *lumiAzurermMonitor) GetLogProfiles() ([]interface{}, error) {
	at, err := azuretransport(a.Runtime.Motor.Transport)
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

		properties, err := jsonToDict(entry.LogProfileProperties)
		if err != nil {
			return nil, err
		}

		var lumiAzureStorageAccount interface{}
		if entry.LogProfileProperties != nil && entry.LogProfileProperties.StorageAccountID != nil {
			// the resource fetches the data itself
			lumiAzureStorageAccount, err = a.Runtime.CreateResource("azurerm.storage.account",
				"id", toString(entry.LogProfileProperties.StorageAccountID),
			)
			if err != nil {
				return nil, err
			}
		}

		lumiAzure, err := a.Runtime.CreateResource("azurerm.monitor.logprofile",
			"id", toString(entry.ID),
			"name", toString(entry.Name),
			"location", toString(entry.Location),
			"type", toString(entry.Type),
			"tags", azureTagsToInterface(entry.Tags),
			"properties", properties,
			"storageAccount", lumiAzureStorageAccount,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzure)
	}

	return res, nil
}

func (a *lumiAzurermMonitorLogprofile) id() (string, error) {
	return a.Id()
}

func diagnosticsSettings(runtime *lumi.Runtime, id string) ([]interface{}, error) {
	// fetch the details
	at, err := azuretransport(runtime.Motor.Transport)
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

		properties, err := jsonToDict(entry.DiagnosticSettings)
		if err != nil {
			return nil, err
		}

		var lumiAzureStorageAccount interface{}
		if entry.DiagnosticSettings != nil && entry.DiagnosticSettings.StorageAccountID != nil {
			// the resource fetches the data itself
			lumiAzureStorageAccount, err = runtime.CreateResource("azurerm.storage.account",
				"id", toString(entry.DiagnosticSettings.StorageAccountID),
			)
			if err != nil {
				return nil, err
			}
		}

		lumiAzure, err := runtime.CreateResource("azurerm.monitor.diagnosticsetting",
			"id", toString(entry.ID),
			"name", toString(entry.Name),
			"type", toString(entry.Type),
			"properties", properties,
			"storageAccount", lumiAzureStorageAccount,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzure)
	}

	return res, nil
}

func (a *lumiAzurermMonitorDiagnosticsetting) id() (string, error) {
	return a.Id()
}
