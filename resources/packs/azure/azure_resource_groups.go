package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	resources "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzure) GetResourceGroups() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := resources.NewResourceGroupsClient(at.SubscriptionID(), token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(&resources.ResourceGroupsClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, rg := range page.Value {
			mqlAzure, err := a.MotorRuntime.CreateResource("azure.resourcegroup",
				"id", core.ToString(rg.ID),
				"name", core.ToString(rg.Name),
				"location", core.ToString(rg.Location),
				"tags", azureTagsToInterface(rg.Tags),
				"type", core.ToString(rg.Type),
				"managedBy", core.ToString(rg.ManagedBy),
				"provisioningState", core.ToString(rg.Properties.ProvisioningState),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzureResourcegroup) id() (string, error) {
	return a.Id()
}
