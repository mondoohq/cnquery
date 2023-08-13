// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/azure/connection"
	"go.mondoo.com/cnquery/types"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	table "github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	storage "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
)

// see https://github.com/Azure/azure-sdk-for-go/issues/8224
type (
	AzureStorageAccountProperties storage.AccountProperties
	Kind                          storage.Kind
)

func (a *mqlAzureSubscriptionStorage) id() (string, error) {
	return "azure.subscription.storage/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionStorage(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.AzureConnection)
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionStorageAccount) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionStorageAccountContainer) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionStorageAccountDataProtection) id() (string, error) {
	return a.StorageAccountId.Data + "/dataProtection", nil
}

func (a *mqlAzureSubscriptionStorageAccountServiceProperties) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionStorageAccountServicePropertiesRetentionPolicy) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionStorageAccountServicePropertiesLogging) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionStorageAccountServicePropertiesMetrics) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionStorage) accounts() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := storage.NewAccountsClient(subId, token, &arm.ClientOptions{})
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
			acc, err := storageAccountToMql(a.MqlRuntime, account)
			if err != nil {
				return nil, err
			}
			res = append(res, acc)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionStorageAccount) containers() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	account, err := resourceID.Component("storageAccounts")
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

			properties, err := convert.JsonToDict(container.Properties)
			if err != nil {
				return nil, err
			}

			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.storage.account.container",
				map[string]*llx.RawData{
					"id":         llx.StringData(convert.ToString(container.ID)),
					"name":       llx.StringData(convert.ToString(container.Name)),
					"etag":       llx.StringData(convert.ToString(container.Etag)),
					"type":       llx.StringData(convert.ToString(container.Type)),
					"properties": llx.DictData(properties),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionStorageAccount) queueProperties() (*mqlAzureSubscriptionStorageAccountServiceProperties, error) {
	props, err := a.getServiceStorageProperties("queue")
	if err != nil {
		return nil, err
	}
	id := a.Id.Data
	return toMqlServiceStorageProperties(a.MqlRuntime, props.ServiceProperties, "queue", id)
}

func (a *mqlAzureSubscriptionStorageAccount) tableProperties() (*mqlAzureSubscriptionStorageAccountServiceProperties, error) {
	props, err := a.getServiceStorageProperties("table")
	if err != nil {
		return nil, err
	}
	id := a.Id.Data
	return toMqlServiceStorageProperties(a.MqlRuntime, props.ServiceProperties, "table", id)
}

func (a *mqlAzureSubscriptionStorageAccount) blobProperties() (*mqlAzureSubscriptionStorageAccountServiceProperties, error) {
	props, err := a.getServiceStorageProperties("blob")
	if err != nil {
		return nil, err
	}
	id := a.Id.Data
	return toMqlServiceStorageProperties(a.MqlRuntime, props.ServiceProperties, "blob", id)
}

func (a *mqlAzureSubscriptionStorageAccount) dataProtection() (*mqlAzureSubscriptionStorageAccountDataProtection, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	account, err := resourceID.Component("storageAccounts")
	if err != nil {
		return nil, err
	}
	client, err := storage.NewBlobServicesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	properties, err := client.GetServiceProperties(ctx, resourceID.ResourceGroup, account, &storage.BlobServicesClientGetServicePropertiesOptions{})
	if err != nil {
		return nil, err
	}

	var blobSoftDeletionEnabled bool
	var blobRetentionDays int64
	var containerSoftDeletionEnabled bool
	var containerRetentionDays int64
	if properties.BlobServiceProperties.BlobServiceProperties.DeleteRetentionPolicy != nil {
		blobSoftDeletionEnabled = convert.ToBool(properties.BlobServiceProperties.BlobServiceProperties.DeleteRetentionPolicy.Enabled)
	}
	if properties.BlobServiceProperties.BlobServiceProperties.DeleteRetentionPolicy != nil {
		blobRetentionDays = convert.ToInt64From32(properties.BlobServiceProperties.BlobServiceProperties.DeleteRetentionPolicy.Days)
	}
	if properties.BlobServiceProperties.BlobServiceProperties.ContainerDeleteRetentionPolicy != nil {
		containerSoftDeletionEnabled = convert.ToBool(properties.BlobServiceProperties.BlobServiceProperties.ContainerDeleteRetentionPolicy.Enabled)
	}
	if properties.BlobServiceProperties.BlobServiceProperties.ContainerDeleteRetentionPolicy != nil {
		containerRetentionDays = convert.ToInt64From32(properties.BlobServiceProperties.BlobServiceProperties.ContainerDeleteRetentionPolicy.Days)
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.storage.account.dataProtection",
		map[string]*llx.RawData{
			"storageAccountId":             llx.StringData(id),
			"blobSoftDeletionEnabled":      llx.BoolData(blobSoftDeletionEnabled),
			"blobRetentionDays":            llx.IntData(blobRetentionDays),
			"containerSoftDeletionEnabled": llx.BoolData(containerSoftDeletionEnabled),
			"containerRetentionDays":       llx.IntData(containerRetentionDays),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionStorageAccountDataProtection), nil
}

func (a *mqlAzureSubscriptionStorageAccount) getServiceStorageProperties(serviceType string) (table.GetPropertiesResponse, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return table.GetPropertiesResponse{}, err
	}

	account, err := resourceID.Component("storageAccounts")
	if err != nil {
		return table.GetPropertiesResponse{}, err
	}

	ctx := context.Background()
	token := conn.Token()
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

func toMqlServiceStorageProperties(runtime *plugin.Runtime, props table.ServiceProperties, serviceType, parentId string) (*mqlAzureSubscriptionStorageAccountServiceProperties, error) {
	loggingRetentionPolicy, err := CreateResource(runtime, "azure.subscription.storage.account.service.properties.retentionPolicy",
		map[string]*llx.RawData{
			"id":            llx.StringData(fmt.Sprintf("%s/%s/properties/logging/retentionPolicy", parentId, serviceType)),
			"retentionDays": llx.IntData(convert.ToInt64From32(props.Logging.RetentionPolicy.Days)),
			"enabled":       llx.BoolData(convert.ToBool(props.Logging.RetentionPolicy.Enabled)),
		})
	if err != nil {
		return nil, err
	}
	logging, err := CreateResource(runtime, "azure.subscription.storage.account.service.properties.logging",
		map[string]*llx.RawData{
			"id":              llx.StringData(fmt.Sprintf("%s/%s/properties/logging", parentId, serviceType)),
			"retentionPolicy": llx.ResourceData(loggingRetentionPolicy, "retentionPolicy"),
			"delete":          llx.BoolData(convert.ToBool(props.Logging.Delete)),
			"write":           llx.BoolData(convert.ToBool(props.Logging.Write)),
			"read":            llx.BoolData(convert.ToBool(props.Logging.Read)),
			"version":         llx.StringData(convert.ToString(props.Logging.Version)),
		})
	if err != nil {
		return nil, err
	}
	minuteMetricsRetentionPolicy, err := CreateResource(runtime, "azure.subscription.storage.account.service.properties.retentionPolicy",
		map[string]*llx.RawData{
			"id":            llx.StringData(fmt.Sprintf("%s/%s/properties/minuteMetrics/retentionPolicy", parentId, serviceType)),
			"retentionDays": llx.IntData(convert.ToInt64From32(props.MinuteMetrics.RetentionPolicy.Days)),
			"enabled":       llx.BoolData(convert.ToBool(props.MinuteMetrics.RetentionPolicy.Enabled)),
		})
	if err != nil {
		return nil, err
	}
	minuteMetrics, err := CreateResource(runtime, "azure.subscription.storage.account.service.properties.metrics",
		map[string]*llx.RawData{
			"id":              llx.StringData(fmt.Sprintf("%s/%s/properties/minuteMetrics/", parentId, serviceType)),
			"retentionPolicy": llx.ResourceData(minuteMetricsRetentionPolicy, "retentionPolicy"),
			"enabled":         llx.BoolData(convert.ToBool(props.MinuteMetrics.Enabled)),
			"includeAPIs":     llx.BoolData(convert.ToBool(props.MinuteMetrics.IncludeAPIs)),
			"version":         llx.StringData(convert.ToString(props.MinuteMetrics.Version)),
		})
	if err != nil {
		return nil, err
	}
	hourMetricsRetentionPolicy, err := CreateResource(runtime, "azure.subscription.storage.account.service.properties.retentionPolicy",
		map[string]*llx.RawData{
			"id":            llx.StringData(fmt.Sprintf("%s/%s/properties/hourMetrics/retentionPolicy", parentId, serviceType)),
			"retentionDays": llx.IntData(convert.ToInt64From32(props.HourMetrics.RetentionPolicy.Days)),
			"enabled":       llx.BoolData(convert.ToBool(props.HourMetrics.RetentionPolicy.Enabled)),
		})
	if err != nil {
		return nil, err
	}
	hourMetrics, err := CreateResource(runtime, "azure.subscription.storage.account.service.properties.metrics",
		map[string]*llx.RawData{
			"id":              llx.StringData(fmt.Sprintf("%s/%s/properties/hourMetrics", parentId, serviceType)),
			"retentionPolicy": llx.ResourceData(hourMetricsRetentionPolicy, "retentionPolicy"),
			"enabled":         llx.BoolData(convert.ToBool(props.HourMetrics.Enabled)),
			"includeAPIs":     llx.BoolData(convert.ToBool(props.HourMetrics.IncludeAPIs)),
			"version":         llx.StringData(convert.ToString(props.HourMetrics.Version)),
		})
	if err != nil {
		return nil, err
	}
	settings, err := CreateResource(runtime, "azure.subscription.storage.account.service.properties",
		map[string]*llx.RawData{
			"id":            llx.StringData(fmt.Sprintf("%s/%s/properties", parentId, serviceType)),
			"minuteMetrics": llx.ResourceData(minuteMetrics, "minuteMetrics"),
			"hourMetrics":   llx.ResourceData(hourMetrics, "hourMetrics"),
			"logging":       llx.ResourceData(logging, "logging"),
		})
	if err != nil {
		return nil, err
	}
	return settings.(*mqlAzureSubscriptionStorageAccountServiceProperties), nil
}

func storageAccountToMql(runtime *plugin.Runtime, account *storage.Account) (*mqlAzureSubscriptionStorageAccount, error) {
	var properties map[string]interface{}
	var err error
	if account.Properties != nil {
		properties, err = convert.JsonToDict(AzureStorageAccountProperties(*account.Properties))
		if err != nil {
			return nil, err
		}
	}

	identity, err := convert.JsonToDict(account.Identity)
	if err != nil {
		return nil, err
	}

	sku, err := convert.JsonToDict(account.SKU)
	if err != nil {
		return nil, err
	}

	kind := ""
	if account.Kind != nil {
		kind = string(*account.Kind)
	}
	res, err := CreateResource(runtime, "azure.subscription.storage.account",
		map[string]*llx.RawData{
			"id":         llx.StringData(convert.ToString(account.ID)),
			"name":       llx.StringData(convert.ToString(account.Name)),
			"location":   llx.StringData(convert.ToString(account.Location)),
			"tags":       llx.MapData(convert.PtrMapStrToInterface(account.Tags), types.String),
			"type":       llx.StringData(convert.ToString(account.Type)),
			"properties": llx.DictData(properties),
			"identity":   llx.DictData(identity),
			"sku":        llx.DictData(sku),
			"kind":       llx.StringData(kind),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionStorageAccount), nil
}

func getStorageAccount(id string, runtime *plugin.Runtime, azureConnection *connection.AzureConnection) (*mqlAzureSubscriptionStorageAccount, error) {
	client, err := storage.NewAccountsClient(azureConnection.SubId(), azureConnection.Token(), &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	// parse the id
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	accountName, err := resourceID.Component("storageAccounts")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	account, err := client.GetProperties(ctx, resourceID.ResourceGroup, accountName, &storage.AccountsClientGetPropertiesOptions{})
	if err != nil {
		return nil, err
	}

	return storageAccountToMql(runtime, &account.Account)
}
