package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
)

func (a *lumiAzurerm) GetResources() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := resources.NewClient(at.SubscriptionID())
	client.Authorizer = authorizer

	resources, err := client.List(ctx, "", "", nil)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range resources.Values() {
		resource := resources.Values()[i]

		// NOTE: properties not not properly filled, therefore you would need to ask each individual resource:
		// https://docs.microsoft.com/en-us/rest/api/resources/resources/getbyid
		// In order to make it happen you need to support each individual type and their api version. Therefore
		// we should not support that via the resource api but instead make sure those properties are properly
		// exposed by the typed resources

		sku, err := jsonToDict(resource.Sku)
		if err != nil {
			return nil, err
		}

		plan, err := jsonToDict(resource.Plan)
		if err != nil {
			return nil, err
		}

		identity, err := jsonToDict(resource.Identity)
		if err != nil {
			return nil, err
		}

		lumiAzure, err := a.MotorRuntime.CreateResource("azurerm.resource",
			"id", toString(resource.ID),
			"name", toString(resource.Name),
			"kind", toString(resource.Location),
			"location", toString(resource.Location),
			"tags", azureTagsToInterface(resource.Tags),
			"type", toString(resource.Type),
			"managedBy", toString(resource.ManagedBy),
			"sku", sku,
			"plan", plan,
			"identity", identity,
			"provisioningState", toString(resource.ProvisioningState),
			"createdTime", azureRmTime(resource.CreatedTime),
			"changedTime", azureRmTime(resource.ChangedTime),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzure)
	}

	return res, nil
}

func (a *lumiAzurermResource) id() (string, error) {
	return a.Id()
}
