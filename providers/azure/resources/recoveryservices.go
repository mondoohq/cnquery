// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservices/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/recoveryservices/armrecoveryservicesbackup/v4"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlAzureSubscriptionRecoveryServicesServiceVaultInternal struct {
	cacheSecuritySettings    *armrecoveryservices.SecuritySettings
	cacheEncryption          *armrecoveryservices.VaultPropertiesEncryption
	cacheMonitoringSettings  *armrecoveryservices.MonitoringSettings
	cacheRedundancySettings  *armrecoveryservices.VaultPropertiesRedundancySettings
	cachePrivateEndpointConn []*armrecoveryservices.PrivateEndpointConnectionVaultProperties
}

func (a *mqlAzureSubscriptionRecoveryServicesService) id() (string, error) {
	return "azure.subscription.recoveryServicesService/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionRecoveryServicesService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionRecoveryServicesServiceVault) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionRecoveryServicesService) vaults() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := armrecoveryservices.NewVaultsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListBySubscriptionIDPager(nil)
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, vault := range page.Value {
			if vault == nil {
				continue
			}
			mqlVault, err := createVaultResource(a.MqlRuntime, vault)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlVault)
		}
	}
	return res, nil
}

func createVaultResource(runtime *plugin.Runtime, vault *armrecoveryservices.Vault) (*mqlAzureSubscriptionRecoveryServicesServiceVault, error) {
	props := vault.Properties
	if props == nil {
		props = &armrecoveryservices.VaultProperties{}
	}

	identity, err := convert.JsonToDict(vault.Identity)
	if err != nil {
		return nil, err
	}

	var skuName string
	if vault.SKU != nil && vault.SKU.Name != nil {
		skuName = string(*vault.SKU.Name)
	}

	var provisioningState string
	if props.ProvisioningState != nil {
		provisioningState = *props.ProvisioningState
	}

	var publicNetworkAccess string
	if props.PublicNetworkAccess != nil {
		publicNetworkAccess = string(*props.PublicNetworkAccess)
	}

	var backupStorageVersion string
	if props.BackupStorageVersion != nil {
		backupStorageVersion = string(*props.BackupStorageVersion)
	}

	var privateEndpointStateForBackup string
	if props.PrivateEndpointStateForBackup != nil {
		privateEndpointStateForBackup = string(*props.PrivateEndpointStateForBackup)
	}

	var privateEndpointStateForSiteRecovery string
	if props.PrivateEndpointStateForSiteRecovery != nil {
		privateEndpointStateForSiteRecovery = string(*props.PrivateEndpointStateForSiteRecovery)
	}

	var secureScore string
	if props.SecureScore != nil {
		secureScore = string(*props.SecureScore)
	}

	resource, err := CreateResource(runtime, ResourceAzureSubscriptionRecoveryServicesServiceVault,
		map[string]*llx.RawData{
			"id":                                  llx.StringDataPtr(vault.ID),
			"name":                                llx.StringDataPtr(vault.Name),
			"location":                            llx.StringDataPtr(vault.Location),
			"type":                                llx.StringDataPtr(vault.Type),
			"tags":                                llx.MapData(convert.PtrMapStrToInterface(vault.Tags), types.String),
			"identity":                            llx.DictData(identity),
			"skuName":                             llx.StringData(skuName),
			"provisioningState":                   llx.StringData(provisioningState),
			"publicNetworkAccess":                 llx.StringData(publicNetworkAccess),
			"backupStorageVersion":                llx.StringData(backupStorageVersion),
			"privateEndpointStateForBackup":       llx.StringData(privateEndpointStateForBackup),
			"privateEndpointStateForSiteRecovery": llx.StringData(privateEndpointStateForSiteRecovery),
			"secureScore":                         llx.StringData(secureScore),
		})
	if err != nil {
		return nil, err
	}

	mqlVault := resource.(*mqlAzureSubscriptionRecoveryServicesServiceVault)
	mqlVault.cacheSecuritySettings = props.SecuritySettings
	mqlVault.cacheEncryption = props.Encryption
	mqlVault.cacheMonitoringSettings = props.MonitoringSettings
	mqlVault.cacheRedundancySettings = props.RedundancySettings
	mqlVault.cachePrivateEndpointConn = props.PrivateEndpointConnections

	return mqlVault, nil
}

// securitySettings builds the security settings sub-resource from cached data.
func (a *mqlAzureSubscriptionRecoveryServicesServiceVault) securitySettings() (*mqlAzureSubscriptionRecoveryServicesServiceVaultSecuritySettings, error) {
	ss := a.cacheSecuritySettings
	if ss == nil {
		a.SecuritySettings.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	var softDeleteState string
	var softDeleteRetentionPeriodInDays int64
	var enhancedSecurityState string
	if ss.SoftDeleteSettings != nil {
		if ss.SoftDeleteSettings.SoftDeleteState != nil {
			softDeleteState = string(*ss.SoftDeleteSettings.SoftDeleteState)
		}
		if ss.SoftDeleteSettings.SoftDeleteRetentionPeriodInDays != nil {
			softDeleteRetentionPeriodInDays = int64(*ss.SoftDeleteSettings.SoftDeleteRetentionPeriodInDays)
		}
		if ss.SoftDeleteSettings.EnhancedSecurityState != nil {
			enhancedSecurityState = string(*ss.SoftDeleteSettings.EnhancedSecurityState)
		}
	}

	var immutabilityState string
	if ss.ImmutabilitySettings != nil && ss.ImmutabilitySettings.State != nil {
		immutabilityState = string(*ss.ImmutabilitySettings.State)
	}

	res, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionRecoveryServicesServiceVaultSecuritySettings,
		map[string]*llx.RawData{
			"id":                              llx.StringData(a.Id.Data + "/securitySettings"),
			"softDeleteState":                 llx.StringData(softDeleteState),
			"softDeleteRetentionPeriodInDays": llx.IntData(softDeleteRetentionPeriodInDays),
			"enhancedSecurityState":           llx.StringData(enhancedSecurityState),
			"immutabilityState":               llx.StringData(immutabilityState),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionRecoveryServicesServiceVaultSecuritySettings), nil
}

func (a *mqlAzureSubscriptionRecoveryServicesServiceVaultSecuritySettings) id() (string, error) {
	return a.Id.Data, nil
}

// encryption builds the encryption sub-resource from cached data.
func (a *mqlAzureSubscriptionRecoveryServicesServiceVault) encryption() (*mqlAzureSubscriptionRecoveryServicesServiceVaultEncryption, error) {
	enc := a.cacheEncryption
	if enc == nil {
		a.Encryption.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	var infrastructureEncryption string
	if enc.InfrastructureEncryption != nil {
		infrastructureEncryption = string(*enc.InfrastructureEncryption)
	}

	var keyVaultKeyUri string
	if enc.KeyVaultProperties != nil && enc.KeyVaultProperties.KeyURI != nil {
		keyVaultKeyUri = *enc.KeyVaultProperties.KeyURI
	}

	res, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionRecoveryServicesServiceVaultEncryption,
		map[string]*llx.RawData{
			"id":                       llx.StringData(a.Id.Data + "/encryption"),
			"infrastructureEncryption": llx.StringData(infrastructureEncryption),
			"keyVaultKeyUri":           llx.StringData(keyVaultKeyUri),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionRecoveryServicesServiceVaultEncryption), nil
}

func (a *mqlAzureSubscriptionRecoveryServicesServiceVaultEncryption) id() (string, error) {
	return a.Id.Data, nil
}

// key returns a typed reference to the Key Vault key used for CMK encryption.
func (a *mqlAzureSubscriptionRecoveryServicesServiceVaultEncryption) key() (*mqlAzureSubscriptionKeyVaultServiceKey, error) {
	keyURI := a.KeyVaultKeyUri.Data
	if keyURI == "" {
		a.Key.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return newKeyVaultKeyResource(a.MqlRuntime, keyURI)
}

// monitoringSettings builds the monitoring settings sub-resource from cached data.
func (a *mqlAzureSubscriptionRecoveryServicesServiceVault) monitoringSettings() (*mqlAzureSubscriptionRecoveryServicesServiceVaultMonitoringSettings, error) {
	ms := a.cacheMonitoringSettings
	if ms == nil {
		a.MonitoringSettings.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	var alertsForAllFailoverIssues, alertsForAllJobFailures, alertsForAllReplicationIssues string
	if ms.AzureMonitorAlertSettings != nil {
		if ms.AzureMonitorAlertSettings.AlertsForAllFailoverIssues != nil {
			alertsForAllFailoverIssues = string(*ms.AzureMonitorAlertSettings.AlertsForAllFailoverIssues)
		}
		if ms.AzureMonitorAlertSettings.AlertsForAllJobFailures != nil {
			alertsForAllJobFailures = string(*ms.AzureMonitorAlertSettings.AlertsForAllJobFailures)
		}
		if ms.AzureMonitorAlertSettings.AlertsForAllReplicationIssues != nil {
			alertsForAllReplicationIssues = string(*ms.AzureMonitorAlertSettings.AlertsForAllReplicationIssues)
		}
	}

	var alertsForCriticalOperations, emailNotificationsForSiteRecovery string
	if ms.ClassicAlertSettings != nil {
		if ms.ClassicAlertSettings.AlertsForCriticalOperations != nil {
			alertsForCriticalOperations = string(*ms.ClassicAlertSettings.AlertsForCriticalOperations)
		}
		if ms.ClassicAlertSettings.EmailNotificationsForSiteRecovery != nil {
			emailNotificationsForSiteRecovery = string(*ms.ClassicAlertSettings.EmailNotificationsForSiteRecovery)
		}
	}

	res, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionRecoveryServicesServiceVaultMonitoringSettings,
		map[string]*llx.RawData{
			"id":                                llx.StringData(a.Id.Data + "/monitoringSettings"),
			"alertsForAllFailoverIssues":        llx.StringData(alertsForAllFailoverIssues),
			"alertsForAllJobFailures":           llx.StringData(alertsForAllJobFailures),
			"alertsForAllReplicationIssues":     llx.StringData(alertsForAllReplicationIssues),
			"alertsForCriticalOperations":       llx.StringData(alertsForCriticalOperations),
			"emailNotificationsForSiteRecovery": llx.StringData(emailNotificationsForSiteRecovery),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionRecoveryServicesServiceVaultMonitoringSettings), nil
}

func (a *mqlAzureSubscriptionRecoveryServicesServiceVaultMonitoringSettings) id() (string, error) {
	return a.Id.Data, nil
}

// redundancySettings builds the redundancy settings sub-resource from cached data.
func (a *mqlAzureSubscriptionRecoveryServicesServiceVault) redundancySettings() (*mqlAzureSubscriptionRecoveryServicesServiceVaultRedundancySettings, error) {
	rs := a.cacheRedundancySettings
	if rs == nil {
		a.RedundancySettings.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	var storageRedundancy string
	if rs.StandardTierStorageRedundancy != nil {
		storageRedundancy = string(*rs.StandardTierStorageRedundancy)
	}

	var crossRegionRestore string
	if rs.CrossRegionRestore != nil {
		crossRegionRestore = string(*rs.CrossRegionRestore)
	}

	res, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionRecoveryServicesServiceVaultRedundancySettings,
		map[string]*llx.RawData{
			"id":                 llx.StringData(a.Id.Data + "/redundancySettings"),
			"storageRedundancy":  llx.StringData(storageRedundancy),
			"crossRegionRestore": llx.StringData(crossRegionRestore),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionRecoveryServicesServiceVaultRedundancySettings), nil
}

func (a *mqlAzureSubscriptionRecoveryServicesServiceVaultRedundancySettings) id() (string, error) {
	return a.Id.Data, nil
}

// privateEndpointConnections builds the shared private endpoint connection resources.
func (a *mqlAzureSubscriptionRecoveryServicesServiceVault) privateEndpointConnections() ([]any, error) {
	var res []any
	for _, pec := range a.cachePrivateEndpointConn {
		if pec == nil {
			continue
		}

		var name, resType string
		if pec.ID != nil {
			connResourceID, err := ParseResourceID(*pec.ID)
			if err == nil {
				if nameComp, err := connResourceID.Component("privateEndpointConnections"); err == nil {
					name = nameComp
				}
				if connResourceID.Provider != "" {
					resType = connResourceID.Provider + "/vaults/privateEndpointConnections"
				}
			}
			if name == "" {
				parts := strings.Split(*pec.ID, "/")
				if len(parts) > 0 {
					name = parts[len(parts)-1]
				}
			}
		}
		if resType == "" {
			resType = "Microsoft.RecoveryServices/vaults/privateEndpointConnections"
		}

		pecArgs := map[string]*llx.RawData{
			"__id": llx.StringDataPtr(pec.ID),
			"id":   llx.StringDataPtr(pec.ID),
			"name": llx.StringData(name),
			"type": llx.StringData(resType),
		}

		if pec.Properties != nil {
			props := pec.Properties
			propsMap, err := convert.JsonToDict(props)
			if err != nil {
				return nil, err
			}
			pecArgs["properties"] = llx.DictData(propsMap)

			if props.PrivateEndpoint != nil {
				pecArgs["privateEndpointId"] = llx.StringDataPtr(props.PrivateEndpoint.ID)
			}
			if props.PrivateLinkServiceConnectionState != nil {
				stateArgs := map[string]*llx.RawData{}
				if props.PrivateLinkServiceConnectionState.ActionsRequired != nil {
					stateArgs["actionsRequired"] = llx.StringData(*props.PrivateLinkServiceConnectionState.ActionsRequired)
				}
				if props.PrivateLinkServiceConnectionState.Description != nil {
					stateArgs["description"] = llx.StringDataPtr(props.PrivateLinkServiceConnectionState.Description)
				}
				if props.PrivateLinkServiceConnectionState.Status != nil {
					stateArgs["status"] = llx.StringData(string(*props.PrivateLinkServiceConnectionState.Status))
				}
				stateRes, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionPrivateEndpointConnectionConnectionState, stateArgs)
				if err != nil {
					return nil, err
				}
				pecArgs["privateLinkServiceConnectionState"] = llx.ResourceData(stateRes, ResourceAzureSubscriptionPrivateEndpointConnectionConnectionState)
			}
			if props.ProvisioningState != nil {
				pecArgs["provisioningState"] = llx.StringData(string(*props.ProvisioningState))
			}
		}

		mqlRes, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionPrivateEndpointConnection, pecArgs)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRes)
	}
	return res, nil
}

// backupConfig fetches the backup vault configuration (soft delete, storage type).
func (a *mqlAzureSubscriptionRecoveryServicesServiceVault) backupConfig() (*mqlAzureSubscriptionRecoveryServicesServiceVaultBackupConfig, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	vaultName, err := resourceID.Component("vaults")
	if err != nil {
		return nil, err
	}

	client, err := armrecoveryservicesbackup.NewBackupResourceVaultConfigsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	resp, err := client.Get(ctx, vaultName, resourceID.ResourceGroup, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && (respErr.StatusCode == http.StatusForbidden || respErr.StatusCode == http.StatusNotFound) {
			a.BackupConfig.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	props := resp.Properties
	if props == nil {
		a.BackupConfig.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	var softDeleteFeatureState string
	if props.SoftDeleteFeatureState != nil {
		softDeleteFeatureState = string(*props.SoftDeleteFeatureState)
	}

	var softDeleteRetentionDays int64
	if props.SoftDeleteRetentionPeriodInDays != nil {
		softDeleteRetentionDays = int64(*props.SoftDeleteRetentionPeriodInDays)
	}

	var enhancedSecurityState string
	if props.EnhancedSecurityState != nil {
		enhancedSecurityState = string(*props.EnhancedSecurityState)
	}

	var storageType string
	if props.StorageType != nil {
		storageType = string(*props.StorageType)
	}

	var storageTypeState string
	if props.StorageTypeState != nil {
		storageTypeState = string(*props.StorageTypeState)
	}

	res, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionRecoveryServicesServiceVaultBackupConfig,
		map[string]*llx.RawData{
			"id":                              llx.StringData(id + "/backupConfig"),
			"softDeleteFeatureState":          llx.StringData(softDeleteFeatureState),
			"softDeleteRetentionPeriodInDays": llx.IntData(softDeleteRetentionDays),
			"enhancedSecurityState":           llx.StringData(enhancedSecurityState),
			"storageType":                     llx.StringData(storageType),
			"storageTypeState":                llx.StringData(storageTypeState),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionRecoveryServicesServiceVaultBackupConfig), nil
}

func (a *mqlAzureSubscriptionRecoveryServicesServiceVaultBackupConfig) id() (string, error) {
	return a.Id.Data, nil
}

// backupPolicies fetches all backup policies in the vault.
func (a *mqlAzureSubscriptionRecoveryServicesServiceVault) backupPolicies() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	vaultName, err := resourceID.Component("vaults")
	if err != nil {
		return nil, err
	}

	client, err := armrecoveryservicesbackup.NewBackupPoliciesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(vaultName, resourceID.ResourceGroup, nil)
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, policy := range page.Value {
			if policy == nil {
				continue
			}

			properties, err := convert.JsonToDict(policy.Properties)
			if err != nil {
				return nil, err
			}

			mqlPolicy, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionRecoveryServicesServiceVaultBackupPolicy,
				map[string]*llx.RawData{
					"id":         llx.StringDataPtr(policy.ID),
					"name":       llx.StringDataPtr(policy.Name),
					"type":       llx.StringDataPtr(policy.Type),
					"properties": llx.DictData(properties),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlPolicy)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionRecoveryServicesServiceVaultBackupPolicy) id() (string, error) {
	return a.Id.Data, nil
}

// protectedItems fetches all protected (backed-up) items in the vault.
func (a *mqlAzureSubscriptionRecoveryServicesServiceVault) protectedItems() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	vaultName, err := resourceID.Component("vaults")
	if err != nil {
		return nil, err
	}

	client, err := armrecoveryservicesbackup.NewBackupProtectedItemsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(vaultName, resourceID.ResourceGroup, nil)
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range page.Value {
			if item == nil {
				continue
			}

			properties, err := convert.JsonToDict(item.Properties)
			if err != nil {
				return nil, err
			}

			mqlItem, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionRecoveryServicesServiceVaultProtectedItem,
				map[string]*llx.RawData{
					"id":         llx.StringDataPtr(item.ID),
					"name":       llx.StringDataPtr(item.Name),
					"type":       llx.StringDataPtr(item.Type),
					"properties": llx.DictData(properties),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlItem)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionRecoveryServicesServiceVaultProtectedItem) id() (string, error) {
	return a.Id.Data, nil
}
