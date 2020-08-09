package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/storage/mgmt/storage"
	"go.mondoo.io/mondoo/lumi"
)

// see https://github.com/Azure/azure-sdk-for-go/issues/8224
type AzureStorageAccountProperties storage.AccountProperties

func (a *lumiAzurerm) GetStorageAccounts() ([]interface{}, error) {
	at, err := azuretransport(a.Runtime.Motor.Transport)
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
			properties, err = jsonToDict(AzureStorageAccountProperties(*account.AccountProperties))
			if err != nil {
				return nil, err
			}
		}

		identity, err := jsonToDict(account.Identity)
		if err != nil {
			return nil, err
		}

		sku, err := jsonToDict(account.Sku)
		if err != nil {
			return nil, err
		}

		lumiAzure, err := a.Runtime.CreateResource("azurerm.storage.account",
			"id", toString(account.ID),
			"name", toString(account.Name),
			"location", toString(account.Location),
			"tags", azureTagsToInterface(account.Tags),
			"type", toString(account.Type),
			"properties", properties,
			"identity", identity,
			"sku", sku,
			"kind", string(account.Kind),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzure)
	}

	return res, nil
}

func (a *lumiAzurermStorageAccount) id() (string, error) {
	return a.Id()
}

func (a *lumiAzurermStorageAccount) init(args *lumi.Args) (*lumi.Args, AzurermStorageAccount, error) {
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

	at, err := azuretransport(a.Runtime.Motor.Transport)
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
		properties, err = jsonToDict(AzureStorageAccountProperties(*account.AccountProperties))
		if err != nil {
			return nil, nil, err
		}
	}

	identity, err := jsonToDict(account.Identity)
	if err != nil {
		return nil, nil, err
	}

	sku, err := jsonToDict(account.Sku)
	if err != nil {
		return nil, nil, err
	}

	(*args)["id"] = toString(account.ID)
	(*args)["name"] = toString(account.Name)
	(*args)["location"] = toString(account.Location)
	(*args)["tags"] = azureTagsToInterface(account.Tags)
	(*args)["type"] = toString(account.Type)
	(*args)["properties"] = properties
	(*args)["identity"] = identity
	(*args)["sku"] = sku
	(*args)["kind"] = string(account.Kind)

	return args, nil, nil
}

func (a *lumiAzurermStorageAccount) GetContainers() ([]interface{}, error) {
	at, err := azuretransport(a.Runtime.Motor.Transport)
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

		properties, err := jsonToDict(entry.ContainerProperties)
		if err != nil {
			return nil, err
		}

		lumiAzure, err := a.Runtime.CreateResource("azurerm.storage.container",
			"id", toString(entry.ID),
			"name", toString(entry.Name),
			"etag", toString(entry.Etag),
			"type", toString(entry.Type),
			"properties", properties,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAzure)
	}

	return res, nil
}

func (a *lumiAzurermStorageContainer) id() (string, error) {
	return a.Id()
}
