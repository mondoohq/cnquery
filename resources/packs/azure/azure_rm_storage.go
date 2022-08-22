package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/storage/mgmt/storage"
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (a *mqlAzurermStorage) id() (string, error) {
	return "azurerm.storage", nil
}

// see https://github.com/Azure/azure-sdk-for-go/issues/8224
type AzureStorageAccountProperties storage.AccountProperties

func (a *mqlAzurermStorage) GetAccounts() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	subscriptionID := at.SubscriptionID()

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := storage.NewAccountsClient(subscriptionID)
	client.Authorizer = authorizer

	accounts, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	for i := range accounts.Values() {
		account := accounts.Values()[i]

		var properties map[string]interface{}
		var err error
		if account.AccountProperties != nil {
			properties, err = core.JsonToDict(AzureStorageAccountProperties(*account.AccountProperties))
			if err != nil {
				return nil, err
			}
		}

		identity, err := core.JsonToDict(account.Identity)
		if err != nil {
			return nil, err
		}

		sku, err := core.JsonToDict(account.Sku)
		if err != nil {
			return nil, err
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
			"kind", string(account.Kind),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAzure)
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

	subscriptionID := at.SubscriptionID()

	ctx := context.Background()
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	client := storage.NewAccountsClient(subscriptionID)
	client.Authorizer = authorizer

	// parse the id
	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, nil, err
	}

	accountName, err := resourceID.Component("storageAccounts")
	if err != nil {
		return nil, nil, err
	}

	account, err := client.GetProperties(ctx, resourceID.ResourceGroup, accountName, "")
	if err != nil {
		return nil, nil, err
	}

	// todo: harmonize with GetStorageAccounts
	var properties map[string]interface{}
	if account.AccountProperties != nil {
		properties, err = core.JsonToDict(AzureStorageAccountProperties(*account.AccountProperties))
		if err != nil {
			return nil, nil, err
		}
	}

	identity, err := core.JsonToDict(account.Identity)
	if err != nil {
		return nil, nil, err
	}

	sku, err := core.JsonToDict(account.Sku)
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = core.ToString(account.ID)
	(*args)["name"] = core.ToString(account.Name)
	(*args)["location"] = core.ToString(account.Location)
	(*args)["tags"] = azureTagsToInterface(account.Tags)
	(*args)["type"] = core.ToString(account.Type)
	(*args)["properties"] = properties
	(*args)["identity"] = identity
	(*args)["sku"] = sku
	(*args)["kind"] = string(account.Kind)

	return args, nil, nil
}

func (a *mqlAzurermStorageAccount) GetContainers() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
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
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := storage.NewBlobContainersClient(resourceID.SubscriptionID)
	client.Authorizer = authorizer

	container, err := client.List(ctx, resourceID.ResourceGroup, account, "", "", "")
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	for i := range container.Values() {
		entry := container.Values()[i]

		properties, err := core.JsonToDict(entry.ContainerProperties)
		if err != nil {
			return nil, err
		}

		mqlAzure, err := a.MotorRuntime.CreateResource("azurerm.storage.container",
			"id", core.ToString(entry.ID),
			"name", core.ToString(entry.Name),
			"etag", core.ToString(entry.Etag),
			"type", core.ToString(entry.Type),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAzure)
	}

	return res, nil
}

func (a *mqlAzurermStorageContainer) id() (string, error) {
	return a.Id()
}
