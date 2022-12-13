package azure

import (
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzure) id() (string, error) {
	return "azure", nil
}

func azureTagsToInterface(data map[string]*string) map[string]interface{} {
	labels := make(map[string]interface{})
	for key := range data {
		labels[key] = core.ToString(data[key])
	}
	return labels
}
