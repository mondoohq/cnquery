package azure

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	subscriptions "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"

	azureres "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzureSubscription) init(args *resources.Args) (*resources.Args, AzureSubscription, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, nil, err
	}
	subscriptionsC, err := subscriptions.NewClient(token, &arm.ClientOptions{})
	if err != nil {
		return nil, nil, err
	}
	ctx := context.Background()
	resp, err := subscriptionsC.Get(ctx, at.SubscriptionID(), &subscriptions.ClientGetOptions{})
	if err != nil {
		return nil, nil, err
	}

	managedByTenants := []interface{}{}
	for _, t := range resp.ManagedByTenants {
		if t != nil {
			managedByTenants = append(managedByTenants, core.ToString((*string)(t.TenantID)))
		}
	}

	subPolicies, err := core.JsonToDict(resp.SubscriptionPolicies)
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = core.ToString(resp.ID)
	(*args)["name"] = core.ToString(resp.DisplayName)
	(*args)["tenantId"] = core.ToString(resp.TenantID)
	(*args)["tags"] = azureTagsToInterface(resp.Tags)
	(*args)["state"] = core.ToString((*string)(resp.State))
	(*args)["subscriptionId"] = core.ToString(resp.SubscriptionID)
	(*args)["authorizationSource"] = core.ToString(resp.AuthorizationSource)
	(*args)["managedByTenants"] = managedByTenants
	(*args)["subscriptionsPolicies"] = subPolicies

	return args, nil, nil
}

func (a *mqlAzureSubscription) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureSubscription) GetResourceGroups() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	subId, err := a.SubscriptionId()
	if err != nil {
		return nil, err
	}

	client, err := azureres.NewResourceGroupsClient(subId, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(&azureres.ResourceGroupsClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, rg := range page.Value {
			mqlAzure, err := a.MotorRuntime.CreateResource("azure.subscription.resourcegroup",
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

func (a *mqlAzureSubscriptionResourcegroup) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureSubscription) GetCompute() (interface{}, error) {
	return a.MotorRuntime.CreateResource("azure.subscription.computeService")
}

func (a *mqlAzureSubscription) GetNetwork() (interface{}, error) {
	return a.MotorRuntime.CreateResource("azure.subscription.networkService")
}

func (a *mqlAzureSubscription) GetStorage() (interface{}, error) {
	return a.MotorRuntime.CreateResource("azure.subscription.storageService")
}

func (a *mqlAzureSubscription) GetWeb() (interface{}, error) {
	return a.MotorRuntime.CreateResource("azure.subscription.webService")
}

func (a *mqlAzureSubscription) GetSql() (interface{}, error) {
	return a.MotorRuntime.CreateResource("azure.subscription.sqlService")
}

func (a *mqlAzureSubscription) GetMySql() (interface{}, error) {
	return a.MotorRuntime.CreateResource("azure.subscription.mysqlService")
}

func (a *mqlAzureSubscription) GetPostgreSql() (interface{}, error) {
	return a.MotorRuntime.CreateResource("azure.subscription.postgresqlService")
}

func (a *mqlAzureSubscription) GetMariaDb() (interface{}, error) {
	return a.MotorRuntime.CreateResource("azure.subscription.mariadbService")
}

func (a *mqlAzureSubscription) GetCosmosDb() (interface{}, error) {
	return a.MotorRuntime.CreateResource("azure.subscription.cosmosdbService")
}

func (a *mqlAzureSubscription) GetKeyVault() (interface{}, error) {
	return a.MotorRuntime.CreateResource("azure.subscription.keyvaultService")
}

func (a *mqlAzureSubscription) GetAuthorization() (interface{}, error) {
	return a.MotorRuntime.CreateResource("azure.subscription.authorizationService")
}

func (a *mqlAzureSubscription) GetMonitor() (interface{}, error) {
	return a.MotorRuntime.CreateResource("azure.subscription.monitorService")
}

func (a *mqlAzureSubscription) GetCloudDefender() (interface{}, error) {
	return a.MotorRuntime.CreateResource("azure.subscription.cloudDefenderService")
}

func (a *mqlAzureSubscription) GetAks() (interface{}, error) {
	return a.MotorRuntime.CreateResource("azure.subscription.aksService")
}

func (a *mqlAzureSubscription) GetId() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (a *mqlAzureSubscription) GetSubscriptionId() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (a *mqlAzureSubscription) GetName() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (a *mqlAzureSubscription) GetTenantId() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (a *mqlAzureSubscription) GetManagedByTenants() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (a *mqlAzureSubscription) GetTags() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (a *mqlAzureSubscription) GetAuthorizationSource() (string, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return "", errors.New("not implemented")
}

func (a *mqlAzureSubscription) GetSubscriptionsPolicies() (map[string]interface{}, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return nil, errors.New("not implemented")
}

func (a *mqlAzureSubscription) GetState() (map[string]interface{}, error) {
	// placeholder to convince MQL that this is an optional field
	// should never be called since the data is initialized in init
	return nil, errors.New("not implemented")
}

func (a *mqlAzureSubscription) GetResources() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := azureres.NewClient(at.SubscriptionID(), token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	expand := "createdTime,changedTime,provisioningState"
	pager := client.NewListPager(&azureres.ClientListOptions{Expand: &expand})
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

			mqlAzure, err := a.MotorRuntime.CreateResource("azure.subscription.resource",
				"id", core.ToString(resource.ID),
				"name", core.ToString(resource.Name),
				"kind", core.ToString(resource.Kind),
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

func (a *mqlAzureSubscriptionResource) id() (string, error) {
	return a.Id()
}
