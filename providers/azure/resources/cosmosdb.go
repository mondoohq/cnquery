// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	cosmosdb "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v3"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmosforpostgresql/armcosmosforpostgresql"
	armresources "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v3"
)

func (a *mqlAzureSubscriptionCosmosDbService) id() (string, error) {
	return "azure.subscription.cosmosdb/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionCosmosDbService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func initAzureSubscriptionCosmosDbServiceAccount(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure cosmosdb account")
	}
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	res, err := NewResource(runtime, "azure.subscription.cosmosDbService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	cosmosDbSvc := res.(*mqlAzureSubscriptionCosmosDbService)
	accountList := cosmosDbSvc.GetAccounts()
	if accountList.Error != nil {
		return nil, nil, accountList.Error
	}
	id, ok := args["id"].Value.(string)
	if !ok {
		return nil, nil, errors.New("id must be a non-nil string value")
	}
	for _, entry := range accountList.Data {
		account := entry.(*mqlAzureSubscriptionCosmosDbServiceAccount)
		if account.Id.Data == id {
			return args, account, nil
		}
	}

	return nil, nil, errors.New("azure cosmosdb account does not exist")
}

func (a *mqlAzureSubscriptionCosmosDbService) accounts() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId := a.SubscriptionId.Data

	res := []any{}

	// Fetch resources of different types - other than MongoDB and PostgreSQL
	cosmosAccounts, err := fetchCosmosDBAccounts(ctx, a.MqlRuntime, conn, subId)
	if err != nil {
		return nil, err
	}
	res = append(res, cosmosAccounts...)

	mongoAccounts, err := fetchDbAccountsByType(ctx, a.MqlRuntime, conn, subId, "Microsoft.DocumentDB/mongoClusters")
	if err != nil {
		return nil, err
	}
	res = append(res, mongoAccounts...)

	postgresAccounts, err := fetchCosmosForPostgres(ctx, a.MqlRuntime, conn, subId)
	if err != nil {
		return nil, err
	}
	res = append(res, postgresAccounts...)

	return res, nil
}

type mqlAzureSubscriptionCosmosDbServiceAccountInternal struct {
	cacheKeyVaultKeyUri        string
	cacheBackupType            string
	cacheBackupIntervalMinutes int64
	cacheBackupRetentionHours  int64
	cacheBackupRedundancy      string
}

func fetchCosmosDBAccounts(ctx context.Context, runtime *plugin.Runtime, conn *connection.AzureConnection, subId string) ([]any, error) {
	accClient, err := cosmosdb.NewDatabaseAccountsClient(subId, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := accClient.NewListPager(&cosmosdb.DatabaseAccountsClientListOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, account := range page.Value {
			properties, err := convert.JsonToDict(account.Properties)
			if err != nil {
				return nil, err
			}

			var publicNetworkAccess *string
			var disableLocalAuth *bool
			var isVirtualNetworkFilterEnabled *bool
			var disableKeyBasedMetadataWriteAccess *bool
			var enableAutomaticFailover *bool
			var enableMultipleWriteLocations *bool
			var minimalTlsVersion *string
			var defaultIdentity *string
			if account.Properties != nil {
				publicNetworkAccess = (*string)(account.Properties.PublicNetworkAccess)
				disableLocalAuth = account.Properties.DisableLocalAuth
				isVirtualNetworkFilterEnabled = account.Properties.IsVirtualNetworkFilterEnabled
				disableKeyBasedMetadataWriteAccess = account.Properties.DisableKeyBasedMetadataWriteAccess
				enableAutomaticFailover = account.Properties.EnableAutomaticFailover
				enableMultipleWriteLocations = account.Properties.EnableMultipleWriteLocations
				minimalTlsVersion = (*string)(account.Properties.MinimalTLSVersion)
				defaultIdentity = account.Properties.DefaultIdentity
			}

			ipRangeFilter := []any{}
			if account.Properties != nil && account.Properties.IPRules != nil {
				for _, rule := range account.Properties.IPRules {
					if rule != nil && rule.IPAddressOrRange != nil {
						ipRangeFilter = append(ipRangeFilter, *rule.IPAddressOrRange)
					}
				}
			}

			var backupType string
			var backupIntervalMinutes, backupRetentionHours int64
			var backupRedundancy string
			if account.Properties != nil && account.Properties.BackupPolicy != nil {
				bp := account.Properties.BackupPolicy
				if bp.GetBackupPolicy().Type != nil {
					backupType = string(*bp.GetBackupPolicy().Type)
				}
				if periodic, ok := bp.(*cosmosdb.PeriodicModeBackupPolicy); ok && periodic.PeriodicModeProperties != nil {
					if periodic.PeriodicModeProperties.BackupIntervalInMinutes != nil {
						backupIntervalMinutes = int64(*periodic.PeriodicModeProperties.BackupIntervalInMinutes)
					}
					if periodic.PeriodicModeProperties.BackupRetentionIntervalInHours != nil {
						backupRetentionHours = int64(*periodic.PeriodicModeProperties.BackupRetentionIntervalInHours)
					}
					if periodic.PeriodicModeProperties.BackupStorageRedundancy != nil {
						backupRedundancy = string(*periodic.PeriodicModeProperties.BackupStorageRedundancy)
					}
				}
			}

			virtualNetworkRules := []any{}
			if account.Properties != nil && account.Properties.VirtualNetworkRules != nil {
				for _, rule := range account.Properties.VirtualNetworkRules {
					if rule == nil {
						continue
					}
					var subnetId string
					var ignoreMissing bool
					if rule.ID != nil {
						subnetId = *rule.ID
					}
					if rule.IgnoreMissingVNetServiceEndpoint != nil {
						ignoreMissing = *rule.IgnoreMissingVNetServiceEndpoint
					}
					ruleId := *account.ID + "/virtualNetworkRules/" + subnetId
					mqlRule, err := CreateResource(runtime, "azure.subscription.cosmosDbService.account.virtualNetworkRule",
						map[string]*llx.RawData{
							"id":                               llx.StringData(ruleId),
							"subnetId":                         llx.StringData(subnetId),
							"ignoreMissingVNetServiceEndpoint": llx.BoolData(ignoreMissing),
						})
					if err != nil {
						return nil, err
					}
					virtualNetworkRules = append(virtualNetworkRules, mqlRule)
				}
			}

			mqlCosmosDbAccount, err := CreateResource(runtime, "azure.subscription.cosmosDbService.account",
				map[string]*llx.RawData{
					"__id":                               llx.StringDataPtr(account.ID),
					"id":                                 llx.StringDataPtr(account.ID),
					"name":                               llx.StringDataPtr(account.Name),
					"tags":                               llx.MapData(convert.PtrMapStrToInterface(account.Tags), types.String),
					"location":                           llx.StringDataPtr(account.Location),
					"kind":                               llx.StringDataPtr((*string)(account.Kind)),
					"type":                               llx.StringDataPtr(account.Type),
					"properties":                         llx.DictData(properties),
					"publicNetworkAccess":                llx.StringDataPtr(publicNetworkAccess),
					"disableLocalAuth":                   llx.BoolDataPtr(disableLocalAuth),
					"isVirtualNetworkFilterEnabled":      llx.BoolDataPtr(isVirtualNetworkFilterEnabled),
					"disableKeyBasedMetadataWriteAccess": llx.BoolDataPtr(disableKeyBasedMetadataWriteAccess),
					"enableAutomaticFailover":            llx.BoolDataPtr(enableAutomaticFailover),
					"enableMultipleWriteLocations":       llx.BoolDataPtr(enableMultipleWriteLocations),
					"ipRangeFilter":                      llx.ArrayData(ipRangeFilter, types.String),
					"minimalTlsVersion":                  llx.StringDataPtr(minimalTlsVersion),
					"defaultIdentity":                    llx.StringDataPtr(defaultIdentity),
					"backupType":                         llx.StringData(backupType),
					"backupIntervalInMinutes":            llx.IntData(backupIntervalMinutes),
					"backupRetentionIntervalInHours":     llx.IntData(backupRetentionHours),
					"backupStorageRedundancy":            llx.StringData(backupRedundancy),
					"virtualNetworkRules":                llx.ArrayData(virtualNetworkRules, types.Resource("azure.subscription.cosmosDbService.account.virtualNetworkRule")),
				})
			if err != nil {
				return nil, err
			}
			mqlAccount := mqlCosmosDbAccount.(*mqlAzureSubscriptionCosmosDbServiceAccount)
			if account.Properties != nil && account.Properties.KeyVaultKeyURI != nil {
				mqlAccount.cacheKeyVaultKeyUri = *account.Properties.KeyVaultKeyURI
			}
			res = append(res, mqlCosmosDbAccount)
		}
	}
	return res, nil
}

func fetchDbAccountsByType(ctx context.Context, runtime *plugin.Runtime, conn *connection.AzureConnection, subId string, resourceType string) ([]any, error) {
	resClient, err := armresources.NewClient(subId, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	filter := fmt.Sprintf("resourceType eq '%s'", resourceType)
	pager := resClient.NewListPager(&armresources.ClientListOptions{
		Filter: &filter,
	})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, account := range page.Value {
			properties, err := convert.JsonToDict(account.Properties)
			if err != nil {
				return nil, err
			}

			mqlResource, err := CreateResource(runtime, "azure.subscription.cosmosDbService.account",
				map[string]*llx.RawData{
					"__id":                               llx.StringDataPtr(account.ID),
					"id":                                 llx.StringDataPtr(account.ID),
					"name":                               llx.StringDataPtr(account.Name),
					"tags":                               llx.MapData(convert.PtrMapStrToInterface(account.Tags), types.String),
					"location":                           llx.StringDataPtr(account.Location),
					"kind":                               llx.StringDataPtr(account.Kind),
					"type":                               llx.StringDataPtr(account.Type),
					"properties":                         llx.DictData(properties),
					"publicNetworkAccess":                llx.StringData(""),
					"disableLocalAuth":                   llx.BoolData(false),
					"isVirtualNetworkFilterEnabled":      llx.BoolData(false),
					"disableKeyBasedMetadataWriteAccess": llx.BoolData(false),
					"enableAutomaticFailover":            llx.BoolData(false),
					"enableMultipleWriteLocations":       llx.BoolData(false),
					"ipRangeFilter":                      llx.ArrayData([]any{}, types.String),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlResource)
		}
	}
	return res, nil
}

// fetches resources of type "Microsoft.DBforPostgreSQL/serverGroupsv2"
func fetchCosmosForPostgres(ctx context.Context, runtime *plugin.Runtime, conn *connection.AzureConnection, subId string) ([]any, error) {
	resClient, err := armcosmosforpostgresql.NewClustersClient(subId, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := resClient.NewListPager(&armcosmosforpostgresql.ClustersClientListOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, account := range page.Value {
			properties, err := convert.JsonToDict(account.Properties)
			if err != nil {
				return nil, err
			}

			mqlResource, err := CreateResource(runtime, "azure.subscription.cosmosDbService.account",
				map[string]*llx.RawData{
					"__id":                               llx.StringDataPtr(account.ID),
					"id":                                 llx.StringDataPtr(account.ID),
					"name":                               llx.StringDataPtr(account.Name),
					"tags":                               llx.MapData(convert.PtrMapStrToInterface(account.Tags), types.String),
					"location":                           llx.StringDataPtr(account.Location),
					"kind":                               llx.StringDataPtr(nil),
					"type":                               llx.StringDataPtr(account.Type),
					"properties":                         llx.DictData(properties),
					"publicNetworkAccess":                llx.StringData(""),
					"disableLocalAuth":                   llx.BoolData(false),
					"isVirtualNetworkFilterEnabled":      llx.BoolData(false),
					"disableKeyBasedMetadataWriteAccess": llx.BoolData(false),
					"enableAutomaticFailover":            llx.BoolData(false),
					"enableMultipleWriteLocations":       llx.BoolData(false),
					"ipRangeFilter":                      llx.ArrayData([]any{}, types.String),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlResource)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccount) encryptionKey() (*mqlAzureSubscriptionKeyVaultServiceKey, error) {
	if a.cacheKeyVaultKeyUri == "" {
		a.EncryptionKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return newKeyVaultKeyResource(a.MqlRuntime, a.cacheKeyVaultKeyUri)
}
