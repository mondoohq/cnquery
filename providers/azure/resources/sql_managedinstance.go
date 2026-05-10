// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	sql "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlAzureSubscriptionSqlServiceManagedInstanceInternal struct {
	cacheSubnetId                      string
	cacheKeyId                         string
	cachePrimaryUserAssignedIdentityId string
}

func (a *mqlAzureSubscriptionSqlServiceManagedInstance) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceManagedInstanceDatabase) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlService) managedInstances() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	client, err := sql.NewManagedInstancesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := client.NewListPager(&sql.ManagedInstancesClientListOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, mi := range page.Value {
			if mi == nil {
				continue
			}

			var skuName, skuTier, skuFamily string
			if mi.SKU != nil {
				if mi.SKU.Name != nil {
					skuName = *mi.SKU.Name
				}
				if mi.SKU.Tier != nil {
					skuTier = *mi.SKU.Tier
				}
				if mi.SKU.Family != nil {
					skuFamily = *mi.SKU.Family
				}
			}

			var identityType, primaryUserAssignedIdentityId string
			if mi.Identity != nil && mi.Identity.Type != nil {
				identityType = string(*mi.Identity.Type)
			}

			var (
				administratorLogin               string
				vCores                           int64
				storageSizeInGB                  int64
				provisioningState                string
				state                            string
				fullyQualifiedDomainName         string
				dnsZone                          string
				subnetId                         string
				licenseType                      string
				minimalTlsVersion                string
				proxyOverride                    string
				publicDataEndpointEnabled        bool
				zoneRedundant                    bool
				currentBackupStorageRedundancy   string
				requestedBackupStorageRedundancy string
				keyId                            string
				maintenanceConfigurationId       string
				timezoneId                       string
				collation                        string
				instancePoolId                   string
				privateEndpointConnectionCount   int64
			)
			if mi.Properties != nil {
				p := mi.Properties
				if p.AdministratorLogin != nil {
					administratorLogin = *p.AdministratorLogin
				}
				if p.VCores != nil {
					vCores = int64(*p.VCores)
				}
				if p.StorageSizeInGB != nil {
					storageSizeInGB = int64(*p.StorageSizeInGB)
				}
				if p.ProvisioningState != nil {
					provisioningState = string(*p.ProvisioningState)
				}
				if p.State != nil {
					state = *p.State
				}
				if p.FullyQualifiedDomainName != nil {
					fullyQualifiedDomainName = *p.FullyQualifiedDomainName
				}
				if p.DNSZone != nil {
					dnsZone = *p.DNSZone
				}
				if p.SubnetID != nil {
					subnetId = *p.SubnetID
				}
				if p.LicenseType != nil {
					licenseType = string(*p.LicenseType)
				}
				if p.MinimalTLSVersion != nil {
					minimalTlsVersion = *p.MinimalTLSVersion
				}
				if p.ProxyOverride != nil {
					proxyOverride = string(*p.ProxyOverride)
				}
				if p.PublicDataEndpointEnabled != nil {
					publicDataEndpointEnabled = *p.PublicDataEndpointEnabled
				}
				if p.ZoneRedundant != nil {
					zoneRedundant = *p.ZoneRedundant
				}
				if p.CurrentBackupStorageRedundancy != nil {
					currentBackupStorageRedundancy = string(*p.CurrentBackupStorageRedundancy)
				}
				if p.RequestedBackupStorageRedundancy != nil {
					requestedBackupStorageRedundancy = string(*p.RequestedBackupStorageRedundancy)
				}
				if p.KeyID != nil {
					keyId = *p.KeyID
				}
				if p.MaintenanceConfigurationID != nil {
					maintenanceConfigurationId = *p.MaintenanceConfigurationID
				}
				if p.TimezoneID != nil {
					timezoneId = *p.TimezoneID
				}
				if p.Collation != nil {
					collation = *p.Collation
				}
				if p.InstancePoolID != nil {
					instancePoolId = *p.InstancePoolID
				}
				if p.PrimaryUserAssignedIdentityID != nil {
					primaryUserAssignedIdentityId = *p.PrimaryUserAssignedIdentityID
				}
				privateEndpointConnectionCount = int64(len(p.PrivateEndpointConnections))
			}

			mqlMi, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.managedInstance",
				map[string]*llx.RawData{
					"id":                               llx.StringDataPtr(mi.ID),
					"name":                             llx.StringDataPtr(mi.Name),
					"location":                         llx.StringDataPtr(mi.Location),
					"tags":                             llx.MapData(convert.PtrMapStrToInterface(mi.Tags), types.String),
					"skuName":                          llx.StringData(skuName),
					"skuTier":                          llx.StringData(skuTier),
					"skuFamily":                        llx.StringData(skuFamily),
					"vCores":                           llx.IntData(vCores),
					"storageSizeInGB":                  llx.IntData(storageSizeInGB),
					"provisioningState":                llx.StringData(provisioningState),
					"state":                            llx.StringData(state),
					"fullyQualifiedDomainName":         llx.StringData(fullyQualifiedDomainName),
					"dnsZone":                          llx.StringData(dnsZone),
					"administratorLogin":               llx.StringData(administratorLogin),
					"licenseType":                      llx.StringData(licenseType),
					"minimalTlsVersion":                llx.StringData(minimalTlsVersion),
					"proxyOverride":                    llx.StringData(proxyOverride),
					"publicDataEndpointEnabled":        llx.BoolData(publicDataEndpointEnabled),
					"zoneRedundant":                    llx.BoolData(zoneRedundant),
					"currentBackupStorageRedundancy":   llx.StringData(currentBackupStorageRedundancy),
					"requestedBackupStorageRedundancy": llx.StringData(requestedBackupStorageRedundancy),
					"identityType":                     llx.StringData(identityType),
					"maintenanceConfigurationId":       llx.StringData(maintenanceConfigurationId),
					"timezoneId":                       llx.StringData(timezoneId),
					"collation":                        llx.StringData(collation),
					"instancePoolId":                   llx.StringData(instancePoolId),
					"privateEndpointConnectionCount":   llx.IntData(privateEndpointConnectionCount),
				})
			if err != nil {
				return nil, err
			}
			mqlInstance := mqlMi.(*mqlAzureSubscriptionSqlServiceManagedInstance)
			mqlInstance.cacheSubnetId = subnetId
			mqlInstance.cacheKeyId = keyId
			mqlInstance.cachePrimaryUserAssignedIdentityId = primaryUserAssignedIdentityId
			res = append(res, mqlInstance)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionSqlServiceManagedInstance) subnet() (*mqlAzureSubscriptionNetworkServiceSubnet, error) {
	if a.cacheSubnetId == "" {
		a.Subnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "azure.subscription.networkService.subnet",
		map[string]*llx.RawData{"id": llx.StringData(a.cacheSubnetId)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionNetworkServiceSubnet), nil
}

func (a *mqlAzureSubscriptionSqlServiceManagedInstance) encryptionKey() (*mqlAzureSubscriptionKeyVaultServiceKey, error) {
	if a.cacheKeyId == "" {
		a.EncryptionKey.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return newKeyVaultKeyResource(a.MqlRuntime, a.cacheKeyId)
}

func (a *mqlAzureSubscriptionSqlServiceManagedInstance) primaryUserAssignedIdentity() (*mqlAzureSubscriptionManagedIdentity, error) {
	if a.cachePrimaryUserAssignedIdentityId == "" {
		a.PrimaryUserAssignedIdentity.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "azure.subscription.managedIdentity",
		map[string]*llx.RawData{"__id": llx.StringData(a.cachePrimaryUserAssignedIdentityId)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionManagedIdentity), nil
}

func (a *mqlAzureSubscriptionSqlServiceManagedInstance) databases() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	resourceID, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	client, err := sql.NewManagedDatabasesClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := client.NewListByInstancePager(resourceID.ResourceGroup, a.Name.Data, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, db := range page.Value {
			if db == nil {
				continue
			}

			var (
				status                   string
				collation                string
				catalogCollation         string
				defaultSecondaryLocation string
				failoverGroupId          string
				createMode               string
			)
			var creationDate, earliestRestorePoint = (*time.Time)(nil), (*time.Time)(nil)
			if p := db.Properties; p != nil {
				if p.Status != nil {
					status = string(*p.Status)
				}
				if p.Collation != nil {
					collation = *p.Collation
				}
				if p.CatalogCollation != nil {
					catalogCollation = string(*p.CatalogCollation)
				}
				if p.DefaultSecondaryLocation != nil {
					defaultSecondaryLocation = *p.DefaultSecondaryLocation
				}
				if p.FailoverGroupID != nil {
					failoverGroupId = *p.FailoverGroupID
				}
				if p.CreateMode != nil {
					createMode = string(*p.CreateMode)
				}
				creationDate = p.CreationDate
				earliestRestorePoint = p.EarliestRestorePoint
			}

			mqlDb, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.managedInstance.database",
				map[string]*llx.RawData{
					"id":                       llx.StringDataPtr(db.ID),
					"name":                     llx.StringDataPtr(db.Name),
					"location":                 llx.StringDataPtr(db.Location),
					"tags":                     llx.MapData(convert.PtrMapStrToInterface(db.Tags), types.String),
					"status":                   llx.StringData(status),
					"collation":                llx.StringData(collation),
					"catalogCollation":         llx.StringData(catalogCollation),
					"creationDate":             llx.TimeDataPtr(creationDate),
					"earliestRestorePoint":     llx.TimeDataPtr(earliestRestorePoint),
					"defaultSecondaryLocation": llx.StringData(defaultSecondaryLocation),
					"failoverGroupId":          llx.StringData(failoverGroupId),
					"createMode":               llx.StringData(createMode),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlDb)
		}
	}
	return res, nil
}
