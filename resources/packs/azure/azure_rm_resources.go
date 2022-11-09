package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	resources "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzurerm) GetResources() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := resources.NewClient(at.SubscriptionID(), token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(&resources.ClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, resource := range page.Value {

			// NOTE: properties not not properly filled, therefore you would need to ask each individual resource:
			// https://docs.microsoft.com/en-us/rest/api/resources/resources/getbyid
			// In order to make it happen you need to support each individual type and their api version. Therefore
			// we should not support that via the resource api but instead make sure those properties are properly
			// exposed by the typed resources
			sku, err := core.JsonToDict(resource.SKU)
			if err != nil {
				return nil, err
			}

			plan, err := core.JsonToDict(resource.Plan)
			if err != nil {
				return nil, err
			}

			identity, err := core.JsonToDict(resource.Identity)
			if err != nil {
				return nil, err
			}

			mqlAzure, err := a.MotorRuntime.CreateResource("azurerm.resource",
				"id", core.ToString(resource.ID),
				"name", core.ToString(resource.Name),
				"kind", core.ToString(resource.Location),
				"location", core.ToString(resource.Location),
				"tags", azureTagsToInterface(resource.Tags),
				"type", core.ToString(resource.Type),
				"managedBy", core.ToString(resource.ManagedBy),
				"sku", sku,
				"plan", plan,
				"identity", identity,
				"provisioningState", core.ToString(resource.ProvisioningState),
				"createdTime", resource.CreatedTime,
				"changedTime", resource.ChangedTime,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzurermResource) id() (string, error) {
	return a.Id()
}
