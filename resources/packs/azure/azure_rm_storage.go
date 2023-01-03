package azure

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	table "github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	storage "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	azure "go.mondoo.com/cnquery/motor/providers/microsoft/azure"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzureStorage) id() (string, error) {
	return "azure.storage", nil
}

// see https://github.com/Azure/azure-sdk-for-go/issues/8224
type (
	AzureStorageAccountProperties storage.AccountProperties
	Kind                          storage.Kind
)

func (a *mqlAzureStorage) GetAccounts() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
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
			mqlAzure, err := a.MotorRuntime.CreateResource("azure.storage.account",
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

func (a *mqlAzureStorageAccount) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureStorageAccount) init(args *resources.Args) (*resources.Args, AzureStorageAccount, error) {
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

	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
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
	resourceID, err := azure.ParseResourceID(id)
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

func (a *mqlAzureStorageAccount) GetContainers() ([]interface{}, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := azure.ParseResourceID(id)
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

			mqlAzure, err := a.MotorRuntime.CreateResource("azure.storage.container",
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

func (a *mqlAzureStorageAccount) GetQueueProperties() (interface{}, error) {
	props, err := a.getServiceStorageProperties("queue")
	if err != nil {
		return nil, err
	}
	parentId, err := a.Id()
	if err != nil {
		return nil, err
	}
	return toMqlServiceStorageProperties(a.MotorRuntime, props.ServiceProperties, "queue", parentId)
}

func (a *mqlAzureStorageAccount) GetTableProperties() (interface{}, error) {
	props, err := a.getServiceStorageProperties("table")
	if err != nil {
		return nil, err
	}
	parentId, err := a.Id()
	if err != nil {
		return nil, err
	}
	return toMqlServiceStorageProperties(a.MotorRuntime, props.ServiceProperties, "table", parentId)
}

func (a *mqlAzureStorageAccount) GetBlobProperties() (interface{}, error) {
	props, err := a.getServiceStorageProperties("blob")
	if err != nil {
		return nil, err
	}
	parentId, err := a.Id()
	if err != nil {
		return nil, err
	}
	return toMqlServiceStorageProperties(a.MotorRuntime, props.ServiceProperties, "blob", parentId)
}

func (a *mqlAzureStorageContainer) id() (string, error) {
	return a.Id()
}

// there seems to be no queue sdk out there, we can reuse the table sdk here as the table/queue properties are identical.
func (a *mqlAzureStorageAccount) getServiceStorageProperties(serviceType string) (table.GetPropertiesResponse, error) {
	at, err := azureTransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return table.GetPropertiesResponse{}, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return table.GetPropertiesResponse{}, err
	}

	resourceID, err := azure.ParseResourceID(id)
	if err != nil {
		return table.GetPropertiesResponse{}, err
	}

	account, err := resourceID.Component("storageAccounts")
	if err != nil {
		return table.GetPropertiesResponse{}, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return table.GetPropertiesResponse{}, err
	}
	urlPath := "https://{accountName}.{serviceType}.core.windows.net/"
	urlPath = strings.ReplaceAll(urlPath, "{accountName}", url.PathEscape(account))
	urlPath = strings.ReplaceAll(urlPath, "{serviceType}", url.PathEscape(serviceType))

	client, err := table.NewServiceClient(urlPath, token, &table.ClientOptions{})
	if err != nil {
		return table.GetPropertiesResponse{}, err
	}
	props, err := client.GetProperties(ctx, &table.GetPropertiesOptions{})
	if err != nil {
		return table.GetPropertiesResponse{}, err
	}
	return props, nil
}

func (a *mqlAzureStorageBlobServiceProperties) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureStorageTableServiceProperties) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureStorageQueueServiceProperties) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureStorageServicePropertiesLogging) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureStorageServicePropertiesMetrics) id() (string, error) {
	return a.Id()
}

func (a *mqlAzureStorageServicePropertiesRetentionPolicy) id() (string, error) {
	return a.Id()
}

func toMqlServiceStorageProperties(runtime *resources.Runtime, props table.ServiceProperties, serviceType, parentId string) (interface{}, error) {
	loggingRetentionPolicy, err := runtime.CreateResource("azure.storage.service.properties.retentionPolicy",
		"id", fmt.Sprintf("%s/%s/properties/logging/retentionPolicy", parentId, serviceType),
		"retentionDays", core.ToInt64From32(props.Logging.RetentionPolicy.Days),
		"enabled", core.ToBool(props.Logging.RetentionPolicy.Enabled))
	if err != nil {
		return nil, err
	}
	logging, err := runtime.CreateResource("azure.storage.service.properties.logging",
		"retentionPolicy", loggingRetentionPolicy,
		"id", fmt.Sprintf("%s/%s/properties/logging", parentId, serviceType),
		"delete", core.ToBool(props.Logging.Delete),
		"write", core.ToBool(props.Logging.Write),
		"read", core.ToBool(props.Logging.Read),
		"version", core.ToString(props.Logging.Version),
	)
	if err != nil {
		return nil, err
	}
	minuteMetricsRetentionPolicy, err := runtime.CreateResource("azure.storage.service.properties.retentionPolicy",
		"id", fmt.Sprintf("%s/%s/properties/minuteMetrics/retentionPolicy", parentId, serviceType),
		"retentionDays", core.ToInt64From32(props.MinuteMetrics.RetentionPolicy.Days),
		"enabled", core.ToBool(props.MinuteMetrics.Enabled),
	)
	if err != nil {
		return nil, err
	}
	minuteMetrics, err := runtime.CreateResource("azure.storage.service.properties.metrics",
		"id", fmt.Sprintf("%s/%s/properties/minuteMetrics/", parentId, serviceType),
		"retentionPolicy", minuteMetricsRetentionPolicy,
		"enabled", core.ToBool(props.MinuteMetrics.Enabled),
		"includeAPIs", core.ToBool(props.MinuteMetrics.IncludeAPIs),
		"version", core.ToString(props.MinuteMetrics.Version),
	)
	if err != nil {
		return nil, err
	}
	hourMetricsRetentionPolicy, err := runtime.CreateResource("azure.storage.service.properties.retentionPolicy",
		"id", fmt.Sprintf("%s/%s/properties/hourMetrics/retentionPolicy", parentId, serviceType),
		"retentionDays", core.ToInt64From32(props.HourMetrics.RetentionPolicy.Days),
		"enabled", core.ToBool(props.HourMetrics.Enabled),
	)
	if err != nil {
		return nil, err
	}
	hourMetrics, err := runtime.CreateResource("azure.storage.service.properties.metrics",
		"id", fmt.Sprintf("%s/%s/properties/hourMetrics", parentId, serviceType),
		"retentionPolicy", hourMetricsRetentionPolicy,
		"enabled", core.ToBool(props.HourMetrics.Enabled),
		"includeAPIs", core.ToBool(props.HourMetrics.IncludeAPIs),
		"version", core.ToString(props.HourMetrics.Version),
	)
	if err != nil {
		return nil, err
	}
	settings, err := runtime.CreateResource(fmt.Sprintf("azure.storage.%sService.properties", serviceType),
		"id", fmt.Sprintf("%s/%s/properties", parentId, serviceType),
		"minuteMetrics", minuteMetrics,
		"hourMetrics", hourMetrics,
		"logging", logging,
	)
	if err != nil {
		return nil, err
	}
	return settings, nil
}
