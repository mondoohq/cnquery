package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"

	cosmosdb "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzureSubscriptionCosmosdbService) init(args *resources.Args) (*resources.Args, AzureSubscriptionCosmosdbService, error) {
	if len(*args) > 0 {
		return args, nil, nil
	}

	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	(*args)["subscriptionId"] = at.SubscriptionID()

	return args, nil, nil
}

func (a *mqlAzureSubscriptionCosmosdbService) id() (string, error) {
	subId, err := a.SubscriptionId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/subscriptions/%s/cosmosDbService", subId), nil
}

func (a *mqlAzureSubscriptionCosmosdbServiceAccount) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureSubscriptionCosmosdbService) GetAccounts() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	accClient, err := cosmosdb.NewDatabaseAccountsClient(at.SubscriptionID(), token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	pager := accClient.NewListPager(&cosmosdb.DatabaseAccountsClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, account := range page.Value {
			properties, err := core.JsonToDict(account.Properties)
			if err != nil {
				return nil, err
			}

			mqlCosmosDbAccount, err := a.MotorRuntime.CreateResource("azure.subscription.cosmosdbService.account",
				"id", core.ToString(account.ID),
				"name", core.ToString(account.Name),
				"tags", azureTagsToInterface(account.Tags),
				"location", core.ToString(account.Location),
				"kind", core.ToString((*string)(account.Kind)),
				"type", core.ToString(account.Type),
				"properties", properties,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlCosmosDbAccount)
		}
	}
	return res, nil
}
