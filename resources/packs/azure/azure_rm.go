package azure

import (
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzurerm) id() (string, error) {
	return "azurerm", nil
}

func azureTagsToInterface(data map[string]*string) map[string]interface{} {
	labels := make(map[string]interface{})
	for key := range data {
		labels[key] = core.ToString(data[key])
	}
	return labels
}
