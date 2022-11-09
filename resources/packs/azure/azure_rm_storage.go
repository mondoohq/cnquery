package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	storage "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzurermStorage) id() (string, error) {
	return "azurerm.storage", nil
}

// see https://github.com/Azure/azure-sdk-for-go/issues/8224
type AzureStorageAccountProperties storage.AccountProperties
type Kind storage.Kind

func (a *mqlAzurermStorage) GetAccounts() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := storage.NewAccountsClient(at.SubscriptionID(), token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(&storage.AccountsClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, account := range page.Value {
			var properties map[string]interface{}
			var err error
			if account.Properties != nil {
				properties, err = core.JsonToDict(AzureStorageAccountProperties(*account.Properties))
				if err != nil {
					return nil, err
				}
			}

			identity, err := core.JsonToDict(account.Identity)
			if err != nil {
				return nil, err
			}

			sku, err := core.JsonToDict(account.SKU)
			if err != nil {
				return nil, err
			}

			kind := ""
			if account.Kind != nil {
				kind = string(*account.Kind)
			}
			mqlAzure, err := a.MotorRuntime.CreateResource("azurerm.storage.account",
				"id", core.ToString(account.ID),
				"name", core.ToString(account.Name),
				"location", core.ToString(account.Location),
				"tags", azureTagsToInterface(account.Tags),
				"type", core.ToString(account.Type),
				"properties", properties,
				"identity", identity,
				"sku", sku,
				"kind", kind,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzurermStorageAccount) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermStorageAccount) init(args *resources.Args) (*resources.Args, AzurermStorageAccount, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	idRaw := (*args)["id"]
	if idRaw == nil {
		return args, nil, nil
	}

	id, ok := idRaw.(string)
	if !ok {
		return args, nil, nil
	}

	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, nil, err
	}

	client, err := storage.NewAccountsClient(at.SubscriptionID(), token, &arm.ClientOptions{})
	if err != nil {
		return nil, nil, err
	}

	// parse the id
	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, nil, err
	}

	accountName, err := resourceID.Component("storageAccounts")
	if err != nil {
		return nil, nil, err
	}

	account, err := client.GetProperties(ctx, resourceID.ResourceGroup, accountName, &storage.AccountsClientGetPropertiesOptions{})
	if err != nil {
		return nil, nil, err
	}

	// todo: harmonize with GetStorageAccounts
	var properties map[string]interface{}
	if account.Properties != nil {
		properties, err = core.JsonToDict(AzureStorageAccountProperties(*account.Properties))
		if err != nil {
			return nil, nil, err
		}
	}

	identity, err := core.JsonToDict(account.Identity)
	if err != nil {
		return nil, nil, err
	}

	sku, err := core.JsonToDict(account.SKU)
	if err != nil {
		return nil, nil, err
	}
	kind := ""
	if account.Kind != nil {
		kind = string(*account.Kind)
	}
	(*args)["id"] = core.ToString(account.ID)
	(*args)["name"] = core.ToString(account.Name)
	(*args)["location"] = core.ToString(account.Location)
	(*args)["tags"] = azureTagsToInterface(account.Tags)
	(*args)["type"] = core.ToString(account.Type)
	(*args)["properties"] = properties
	(*args)["identity"] = identity
	(*args)["sku"] = sku
	(*args)["kind"] = kind

	return args, nil, nil
}

func (a *mqlAzurermStorageAccount) GetContainers() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	account, err := resourceID.Component("storageAccounts")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := storage.NewBlobContainersClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(resourceID.ResourceGroup, account, &storage.BlobContainersClientListOptions{})
	res := []interface{}{}

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, container := range page.Value {

			properties, err := core.JsonToDict(container.Properties)
			if err != nil {
				return nil, err
			}

			mqlAzure, err := a.MotorRuntime.CreateResource("azurerm.storage.container",
				"id", core.ToString(container.ID),
				"name", core.ToString(container.Name),
				"etag", core.ToString(container.Etag),
				"type", core.ToString(container.Type),
				"properties", properties,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzurermStorageContainer) id() (string, error) {
	return a.Id()
}
