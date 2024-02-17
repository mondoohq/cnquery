// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/azure/connection"
	"go.mondoo.com/cnquery/v10/types"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	table "github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	storage "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
)

// see https://github.com/Azure/azure-sdk-for-go/issues/8224
type (
	AzureStorageAccountProperties storage.AccountProperties
	Kind                          storage.Kind
)

func (a *mqlAzureSubscriptionStorageService) id() (string, error) {
	return "azure.subscription.storage/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionStorageService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.AzureConnection)
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionStorageServiceAccount) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionStorageServiceAccountContainer) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionStorageServiceAccountDataProtection) id() (string, error) {
	return a.StorageAccountId.Data + "/dataProtection", nil
}

func (a *mqlAzureSubscriptionStorageServiceAccountServiceProperties) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionStorageServiceAccountServicePropertiesRetentionPolicy) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionStorageServiceAccountServicePropertiesLogging) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionStorageServiceAccountServicePropertiesMetrics) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionStorageService) accounts() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := storage.NewAccountsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
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

func (a *mqlAzureSubscriptionStorageServiceAccount) containers() ([]interface{}, error) {
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
	client, err := storage.NewBlobContainersClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
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

			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.storageService.account.container",
				map[string]*llx.RawData{
					"id":         llx.StringDataPtr(container.ID),
					"name":       llx.StringDataPtr(container.Name),
					"etag":       llx.StringDataPtr(container.Etag),
					"type":       llx.StringDataPtr(container.Type),
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

func (a *mqlAzureSubscriptionStorageServiceAccount) queueProperties() (*mqlAzureSubscriptionStorageServiceAccountServiceProperties, error) {
	props, err := a.getServiceStorageProperties("queue")
	if err != nil {
		return nil, err
	}
	id := a.Id.Data
	return toMqlServiceStorageProperties(a.MqlRuntime, props.ServiceProperties, "queue", id)
}

func (a *mqlAzureSubscriptionStorageServiceAccount) tableProperties() (*mqlAzureSubscriptionStorageServiceAccountServiceProperties, error) {
	props, err := a.getServiceStorageProperties("table")
	if err != nil {
		return nil, err
	}
	id := a.Id.Data
	return toMqlServiceStorageProperties(a.MqlRuntime, props.ServiceProperties, "table", id)
}

func (a *mqlAzureSubscriptionStorageServiceAccount) blobProperties() (*mqlAzureSubscriptionStorageServiceAccountServiceProperties, error) {
	props, err := a.getServiceStorageProperties("blob")
	if err != nil {
		return nil, err
	}
	id := a.Id.Data
	return toMqlServiceStorageProperties(a.MqlRuntime, props.ServiceProperties, "blob", id)
}

func (a *mqlAzureSubscriptionStorageServiceAccount) dataProtection() (*mqlAzureSubscriptionStorageServiceAccountDataProtection, error) {
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
	client, err := storage.NewBlobServicesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
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

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.storageService.account.dataProtection",
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
	return res.(*mqlAzureSubscriptionStorageServiceAccountDataProtection), nil
}

func (a *mqlAzureSubscriptionStorageServiceAccount) getServiceStorageProperties(serviceType string) (table.GetPropertiesResponse, error) {
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

func toMqlServiceStorageProperties(runtime *plugin.Runtime, props table.ServiceProperties, serviceType, parentId string) (*mqlAzureSubscriptionStorageServiceAccountServiceProperties, error) {
	loggingRetentionPolicy, err := CreateResource(runtime, "azure.subscription.storageService.account.service.properties.retentionPolicy",
		map[string]*llx.RawData{
			"id":            llx.StringData(fmt.Sprintf("%s/%s/properties/logging/retentionPolicy", parentId, serviceType)),
			"retentionDays": llx.IntData(convert.ToInt64From32(props.Logging.RetentionPolicy.Days)),
			"enabled":       llx.BoolDataPtr(props.Logging.RetentionPolicy.Enabled),
		})
	if err != nil {
		return nil, err
	}
	logging, err := CreateResource(runtime, "azure.subscription.storageService.account.service.properties.logging",
		map[string]*llx.RawData{
			"id":              llx.StringData(fmt.Sprintf("%s/%s/properties/logging", parentId, serviceType)),
			"retentionPolicy": llx.ResourceData(loggingRetentionPolicy, "retentionPolicy"),
			"delete":          llx.BoolDataPtr(props.Logging.Delete),
			"write":           llx.BoolDataPtr(props.Logging.Write),
			"read":            llx.BoolDataPtr(props.Logging.Read),
			"version":         llx.StringDataPtr(props.Logging.Version),
		})
	if err != nil {
		return nil, err
	}
	minuteMetricsRetentionPolicy, err := CreateResource(runtime, "azure.subscription.storageService.account.service.properties.retentionPolicy",
		map[string]*llx.RawData{
			"id":            llx.StringData(fmt.Sprintf("%s/%s/properties/minuteMetrics/retentionPolicy", parentId, serviceType)),
			"retentionDays": llx.IntData(convert.ToInt64From32(props.MinuteMetrics.RetentionPolicy.Days)),
			"enabled":       llx.BoolDataPtr(props.MinuteMetrics.RetentionPolicy.Enabled),
		})
	if err != nil {
		return nil, err
	}
	minuteMetrics, err := CreateResource(runtime, "azure.subscription.storageService.account.service.properties.metrics",
		map[string]*llx.RawData{
			"id":              llx.StringData(fmt.Sprintf("%s/%s/properties/minuteMetrics/", parentId, serviceType)),
			"retentionPolicy": llx.ResourceData(minuteMetricsRetentionPolicy, "retentionPolicy"),
			"enabled":         llx.BoolDataPtr(props.MinuteMetrics.Enabled),
			"includeAPIs":     llx.BoolDataPtr(props.MinuteMetrics.IncludeAPIs),
			"version":         llx.StringDataPtr(props.MinuteMetrics.Version),
		})
	if err != nil {
		return nil, err
	}
	hourMetricsRetentionPolicy, err := CreateResource(runtime, "azure.subscription.storageService.account.service.properties.retentionPolicy",
		map[string]*llx.RawData{
			"id":            llx.StringData(fmt.Sprintf("%s/%s/properties/hourMetrics/retentionPolicy", parentId, serviceType)),
			"retentionDays": llx.IntData(convert.ToInt64From32(props.HourMetrics.RetentionPolicy.Days)),
			"enabled":       llx.BoolDataPtr(props.HourMetrics.RetentionPolicy.Enabled),
		})
	if err != nil {
		return nil, err
	}
	hourMetrics, err := CreateResource(runtime, "azure.subscription.storageService.account.service.properties.metrics",
		map[string]*llx.RawData{
			"id":              llx.StringData(fmt.Sprintf("%s/%s/properties/hourMetrics", parentId, serviceType)),
			"retentionPolicy": llx.ResourceData(hourMetricsRetentionPolicy, "retentionPolicy"),
			"enabled":         llx.BoolDataPtr(props.HourMetrics.Enabled),
			"includeAPIs":     llx.BoolDataPtr(props.HourMetrics.IncludeAPIs),
			"version":         llx.StringDataPtr(props.HourMetrics.Version),
		})
	if err != nil {
		return nil, err
	}
	settings, err := CreateResource(runtime, "azure.subscription.storageService.account.service.properties",
		map[string]*llx.RawData{
			"id":            llx.StringData(fmt.Sprintf("%s/%s/properties", parentId, serviceType)),
			"minuteMetrics": llx.ResourceData(minuteMetrics, "minuteMetrics"),
			"hourMetrics":   llx.ResourceData(hourMetrics, "hourMetrics"),
			"logging":       llx.ResourceData(logging, "logging"),
		})
	if err != nil {
		return nil, err
	}
	return settings.(*mqlAzureSubscriptionStorageServiceAccountServiceProperties), nil
}

func storageAccountToMql(runtime *plugin.Runtime, account *storage.Account) (*mqlAzureSubscriptionStorageServiceAccount, error) {
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
	res, err := CreateResource(runtime, "azure.subscription.storageService.account",
		map[string]*llx.RawData{
			"id":         llx.StringDataPtr(account.ID),
			"name":       llx.StringDataPtr(account.Name),
			"location":   llx.StringDataPtr(account.Location),
			"tags":       llx.MapData(convert.PtrMapStrToInterface(account.Tags), types.String),
			"type":       llx.StringDataPtr(account.Type),
			"properties": llx.DictData(properties),
			"identity":   llx.DictData(identity),
			"sku":        llx.DictData(sku),
			"kind":       llx.StringData(kind),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionStorageServiceAccount), nil
}

func getStorageAccount(id string, runtime *plugin.Runtime, azureConnection *connection.AzureConnection) (*mqlAzureSubscriptionStorageServiceAccount, error) {
	client, err := storage.NewAccountsClient(azureConnection.SubId(), azureConnection.Token(), &arm.ClientOptions{
		ClientOptions: azureConnection.ClientOptions(),
	})
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

func initAzureSubscriptionStorageServiceAccount(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure storage account")
	}

	conn := runtime.Connection.(*connection.AzureConnection)
	res, err := NewResource(runtime, "azure.subscription.storageService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	storage := res.(*mqlAzureSubscriptionStorageService)
	accs := storage.GetAccounts()
	if accs.Error != nil {
		return nil, nil, accs.Error
	}
	id := args["id"].Value.(string)
	for _, entry := range accs.Data {
		storageAcc := entry.(*mqlAzureSubscriptionStorageServiceAccount)
		if storageAcc.Id.Data == id {
			return args, storageAcc, nil
		}
	}

	return nil, nil, errors.New("azure storage account does not exist")
}

func initAzureSubscriptionStorageServiceAccountContainer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure storage account")
	}

	conn := runtime.Connection.(*connection.AzureConnection)
	res, err := NewResource(runtime, "azure.subscription.storageService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	storage := res.(*mqlAzureSubscriptionStorageService)
	accs := storage.GetAccounts()
	if accs.Error != nil {
		return nil, nil, accs.Error
	}
	id := args["id"].Value.(string)
	for _, entry := range accs.Data {
		storageAcc := entry.(*mqlAzureSubscriptionStorageServiceAccount)
		containers := storageAcc.GetContainers()
		if containers.Error != nil {
			return nil, nil, containers.Error
		}
		for _, c := range containers.Data {
			container := c.(*mqlAzureSubscriptionStorageServiceAccountContainer)
			if container.Id.Data == id {
				return args, storageAcc, nil
			}

		}
	}

	return nil, nil, errors.New("azure storage account does not exist")
}
