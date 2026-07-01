// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"

	mysql "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysql"

	flexible "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysqlflexibleservers/v2"
)

func (a *mqlAzureSubscriptionMySqlService) id() (string, error) {
	return "azure.subscription.mysql/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionMySqlService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionMySqlServiceServer) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionMySqlServiceDatabase) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionMySqlService) servers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	dbClient, err := mysql.NewServersClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbClient.NewListPager(&mysql.ServersClientListOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, dbServer := range page.Value {
			properties, err := convert.JsonToDict(dbServer.Properties)
			if err != nil {
				return nil, err
			}

			var sslEnforcement *bool
			var minimalTlsVersion *string
			var publicNetworkAccess *string
			var infrastructureEncryption *bool
			var version *string
			var backupRetentionDays int64
			var geoRedundantBackup string
			if dbServer.Properties != nil {
				if dbServer.Properties.SSLEnforcement != nil {
					v := *dbServer.Properties.SSLEnforcement == mysql.SSLEnforcementEnumEnabled
					sslEnforcement = &v
				}
				minimalTlsVersion = (*string)(dbServer.Properties.MinimalTLSVersion)
				publicNetworkAccess = (*string)(dbServer.Properties.PublicNetworkAccess)
				if dbServer.Properties.InfrastructureEncryption != nil {
					v := *dbServer.Properties.InfrastructureEncryption == mysql.InfrastructureEncryptionEnabled
					infrastructureEncryption = &v
				}
				version = (*string)(dbServer.Properties.Version)
				if dbServer.Properties.StorageProfile != nil {
					if dbServer.Properties.StorageProfile.BackupRetentionDays != nil {
						backupRetentionDays = int64(*dbServer.Properties.StorageProfile.BackupRetentionDays)
					}
					if dbServer.Properties.StorageProfile.GeoRedundantBackup != nil {
						geoRedundantBackup = string(*dbServer.Properties.StorageProfile.GeoRedundantBackup)
					}
				}
			}

			mqlAzureDbServer, err := CreateResource(a.MqlRuntime, "azure.subscription.mySqlService.server",
				map[string]*llx.RawData{
					"id":                       llx.StringDataPtr(dbServer.ID),
					"name":                     llx.StringDataPtr(dbServer.Name),
					"location":                 llx.StringDataPtr(dbServer.Location),
					"tags":                     llx.MapData(convert.PtrMapStrToInterface(dbServer.Tags), types.String),
					"type":                     llx.StringDataPtr(dbServer.Type),
					"properties":               llx.DictData(properties),
					"sslEnforcement":           llx.BoolDataPtr(sslEnforcement),
					"minimalTlsVersion":        llx.StringDataPtr(minimalTlsVersion),
					"publicNetworkAccess":      llx.StringDataPtr(publicNetworkAccess),
					"infrastructureEncryption": llx.BoolDataPtr(infrastructureEncryption),
					"version":                  llx.StringDataPtr(version),
					"backupRetentionDays":      llx.IntData(backupRetentionDays),
					"geoRedundantBackup":       llx.StringData(geoRedundantBackup),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureDbServer)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionMySqlService) flexibleServers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	dbClient, err := flexible.NewServersClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbClient.NewListPager(&flexible.ServersClientListOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, dbServer := range page.Value {
			properties, err := convert.JsonToDict(dbServer.Properties)
			if err != nil {
				return nil, err
			}

			var version string
			if dbServer.Properties != nil && dbServer.Properties.Version != nil {
				version = string(*dbServer.Properties.Version)
			}

			var publicNetworkAccess string
			if dbServer.Properties != nil {
				if dbServer.Properties.Network != nil && dbServer.Properties.Network.PublicNetworkAccess != nil {
					publicNetworkAccess = string(*dbServer.Properties.Network.PublicNetworkAccess)
				}
			}

			var dataEncryptionType string
			if dbServer.Properties != nil && dbServer.Properties.DataEncryption != nil {
				if dbServer.Properties.DataEncryption.Type != nil {
					dataEncryptionType = string(*dbServer.Properties.DataEncryption.Type)
				}
			}

			var backupRetentionDays int64
			var geoRedundantBackup string
			if dbServer.Properties != nil && dbServer.Properties.Backup != nil {
				if dbServer.Properties.Backup.BackupRetentionDays != nil {
					backupRetentionDays = int64(*dbServer.Properties.Backup.BackupRetentionDays)
				}
				if dbServer.Properties.Backup.GeoRedundantBackup != nil {
					geoRedundantBackup = string(*dbServer.Properties.Backup.GeoRedundantBackup)
				}
			}

			var haMode, haState string
			if dbServer.Properties != nil && dbServer.Properties.HighAvailability != nil {
				if dbServer.Properties.HighAvailability.Mode != nil {
					haMode = string(*dbServer.Properties.HighAvailability.Mode)
				}
				if dbServer.Properties.HighAvailability.State != nil {
					haState = string(*dbServer.Properties.HighAvailability.State)
				}
			}

			var identityType, identityPrincipalId, identityTenantId *string
			var userAssignedIdentityIds []string
			if dbServer.Identity != nil {
				identityType = cosmosEnumStrPtr(dbServer.Identity.Type)
				identityPrincipalId = dbServer.Identity.PrincipalID
				identityTenantId = dbServer.Identity.TenantID
				userAssignedIdentityIds = sortedUserAssignedIdentityKeys(dbServer.Identity.UserAssignedIdentities)
			}

			var fullVersion, storageRedundancy, maintenancePatchStrategy *string
			var databasePort, backupIntervalHours *int32
			var storageAutoIoScaling, storageLogOnDisk *bool
			if dbServer.Properties != nil {
				fullVersion = dbServer.Properties.FullVersion
				databasePort = dbServer.Properties.DatabasePort
				if dbServer.Properties.Backup != nil {
					backupIntervalHours = dbServer.Properties.Backup.BackupIntervalHours
				}
				if dbServer.Properties.Storage != nil {
					storageRedundancy = (*string)(dbServer.Properties.Storage.StorageRedundancy)
					if dbServer.Properties.Storage.AutoIoScaling != nil {
						v := *dbServer.Properties.Storage.AutoIoScaling == flexible.EnableStatusEnumEnabled
						storageAutoIoScaling = &v
					}
					if dbServer.Properties.Storage.LogOnDisk != nil {
						v := *dbServer.Properties.Storage.LogOnDisk == flexible.EnableStatusEnumEnabled
						storageLogOnDisk = &v
					}
				}
				if dbServer.Properties.MaintenancePolicy != nil {
					maintenancePatchStrategy = (*string)(dbServer.Properties.MaintenancePolicy.PatchStrategy)
				}
			}

			mqlAzureDbServer, err := CreateResource(a.MqlRuntime, "azure.subscription.mySqlService.flexibleServer",
				map[string]*llx.RawData{
					"id":                       llx.StringDataPtr(dbServer.ID),
					"name":                     llx.StringDataPtr(dbServer.Name),
					"location":                 llx.StringDataPtr(dbServer.Location),
					"tags":                     llx.MapData(convert.PtrMapStrToInterface(dbServer.Tags), types.String),
					"type":                     llx.StringDataPtr(dbServer.Type),
					"properties":               llx.DictData(properties),
					"version":                  llx.StringData(version),
					"publicNetworkAccess":      llx.StringData(publicNetworkAccess),
					"dataEncryptionType":       llx.StringData(dataEncryptionType),
					"backupRetentionDays":      llx.IntData(backupRetentionDays),
					"geoRedundantBackup":       llx.StringData(geoRedundantBackup),
					"highAvailabilityMode":     llx.StringData(haMode),
					"highAvailabilityState":    llx.StringData(haState),
					"identityType":             llx.StringDataPtr(identityType),
					"principalId":              llx.StringDataPtr(identityPrincipalId),
					"tenantId":                 llx.StringDataPtr(identityTenantId),
					"fullVersion":              llx.StringDataPtr(fullVersion),
					"databasePort":             llx.IntDataPtr(databasePort),
					"storageAutoIoScaling":     llx.BoolDataPtr(storageAutoIoScaling),
					"storageLogOnDisk":         llx.BoolDataPtr(storageLogOnDisk),
					"storageRedundancy":        llx.StringDataPtr(storageRedundancy),
					"backupIntervalHours":      llx.IntDataPtr(backupIntervalHours),
					"maintenancePatchStrategy": llx.StringDataPtr(maintenancePatchStrategy),
				})
			if err != nil {
				return nil, err
			}
			mqlServer := mqlAzureDbServer.(*mqlAzureSubscriptionMySqlServiceFlexibleServer)
			mqlServer.cacheUserAssignedIdentityIds = userAssignedIdentityIds
			if dbServer.Properties != nil && dbServer.Properties.DataEncryption != nil {
				if dbServer.Properties.DataEncryption.PrimaryKeyURI != nil {
					mqlServer.cacheDataEncryptionKeyURI = *dbServer.Properties.DataEncryption.PrimaryKeyURI
				}
				if dbServer.Properties.DataEncryption.GeoBackupKeyURI != nil {
					mqlServer.cacheGeoBackupKeyURI = *dbServer.Properties.DataEncryption.GeoBackupKeyURI
				}
			}
			sysData, err := convert.JsonToDict(dbServer.SystemData)
			if err != nil {
				return nil, err
			}
			mqlServer.cacheSystemData = sysData
			res = append(res, mqlAzureDbServer)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) sslEnforcement() (bool, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return false, err
	}

	server, err := resourceID.Component("flexibleServers")
	if err != nil {
		return false, err
	}

	dbConfClient, err := flexible.NewConfigurationsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return false, err
	}

	resp, err := dbConfClient.Get(ctx, resourceID.ResourceGroup, server, "require_secure_transport", nil)
	if err != nil {
		// Default to true as MySQL flexible servers enforce SSL by default
		return true, nil
	}

	if resp.Properties != nil && resp.Properties.Value != nil {
		return strings.EqualFold(*resp.Properties.Value, "ON"), nil
	}

	return true, nil
}

func (a *mqlAzureSubscriptionMySqlServiceServer) databases() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}

	dbDatabaseClient, err := mysql.NewDatabasesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbDatabaseClient.NewListByServerPager(resourceID.ResourceGroup, server, &mysql.DatabasesClientListByServerOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureDatabase, err := CreateResource(a.MqlRuntime, "azure.subscription.mySqlService.database",
				map[string]*llx.RawData{
					"id":        llx.StringDataPtr(entry.ID),
					"name":      llx.StringDataPtr(entry.Name),
					"type":      llx.StringDataPtr(entry.Type),
					"charset":   llx.StringDataPtr(entry.Properties.Charset),
					"collation": llx.StringDataPtr(entry.Properties.Collation),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureDatabase)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionMySqlServiceServer) firewallRules() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}

	dbFirewallClient, err := mysql.NewFirewallRulesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbFirewallClient.NewListByServerPager(resourceID.ResourceGroup, server, &mysql.FirewallRulesClientListByServerOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlFireWallRule, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.firewallrule",
				map[string]*llx.RawData{
					"id":             llx.StringDataPtr(entry.ID),
					"name":           llx.StringDataPtr(entry.Name),
					"type":           llx.StringDataPtr(entry.Type),
					"startIpAddress": llx.StringDataPtr(entry.Properties.StartIPAddress),
					"endIpAddress":   llx.StringDataPtr(entry.Properties.EndIPAddress),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlFireWallRule)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionMySqlServiceServer) configuration() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}

	dbConfClient, err := mysql.NewConfigurationsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbConfClient.NewListByServerPager(resourceID.ResourceGroup, server, &mysql.ConfigurationsClientListByServerOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureConfiguration, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.configuration",
				map[string]*llx.RawData{
					"id":            llx.StringDataPtr(entry.ID),
					"name":          llx.StringDataPtr(entry.Name),
					"type":          llx.StringDataPtr(entry.Type),
					"value":         llx.StringDataPtr(entry.Properties.Value),
					"description":   llx.StringDataPtr(entry.Properties.Description),
					"defaultValue":  llx.StringDataPtr(entry.Properties.DefaultValue),
					"dataType":      llx.StringDataPtr(entry.Properties.DataType),
					"allowedValues": llx.StringDataPtr(entry.Properties.AllowedValues),
					"source":        llx.StringDataPtr(entry.Properties.Source),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureConfiguration)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) databases() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("flexibleServers")
	if err != nil {
		return nil, err
	}

	dbDatabaseClient, err := flexible.NewDatabasesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbDatabaseClient.NewListByServerPager(resourceID.ResourceGroup, server, &flexible.DatabasesClientListByServerOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureDatabase, err := CreateResource(a.MqlRuntime, "azure.subscription.mySqlService.database",
				map[string]*llx.RawData{
					"id":        llx.StringDataPtr(entry.ID),
					"name":      llx.StringDataPtr(entry.Name),
					"type":      llx.StringDataPtr(entry.Type),
					"charset":   llx.StringDataPtr(entry.Properties.Charset),
					"collation": llx.StringDataPtr(entry.Properties.Collation),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureDatabase)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) firewallRules() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("flexibleServers")
	if err != nil {
		return nil, err
	}
	dbFirewallClient, err := flexible.NewFirewallRulesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbFirewallClient.NewListByServerPager(resourceID.ResourceGroup, server, &flexible.FirewallRulesClientListByServerOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlFireWallRule, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.firewallrule",
				map[string]*llx.RawData{
					"id":             llx.StringDataPtr(entry.ID),
					"name":           llx.StringDataPtr(entry.Name),
					"type":           llx.StringDataPtr(entry.Type),
					"startIpAddress": llx.StringDataPtr(entry.Properties.StartIPAddress),
					"endIpAddress":   llx.StringDataPtr(entry.Properties.EndIPAddress),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlFireWallRule)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) configuration() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("flexibleServers")
	if err != nil {
		return nil, err
	}

	dbConfClient, err := flexible.NewConfigurationsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbConfClient.NewListByServerPager(resourceID.ResourceGroup, server, &flexible.ConfigurationsClientListByServerOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureConfiguration, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.configuration",
				map[string]*llx.RawData{
					"id":            llx.StringDataPtr(entry.ID),
					"name":          llx.StringDataPtr(entry.Name),
					"type":          llx.StringDataPtr(entry.Type),
					"value":         llx.StringDataPtr(entry.Properties.Value),
					"description":   llx.StringDataPtr(entry.Properties.Description),
					"defaultValue":  llx.StringDataPtr(entry.Properties.DefaultValue),
					"dataType":      llx.StringDataPtr(entry.Properties.DataType),
					"allowedValues": llx.StringDataPtr(entry.Properties.AllowedValues),
					"source":        llx.StringDataPtr((*string)(entry.Properties.Source)),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureConfiguration)
		}
	}
	return res, nil
}

type mqlAzureSubscriptionMySqlServiceFlexibleServerInternal struct {
	cacheDataEncryptionKeyURI    string
	cacheGeoBackupKeyURI         string
	cacheSystemData              any
	cacheUserAssignedIdentityIds []string
}

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServerAdministrator) id() (string, error) {
	return a.Id.Data, nil
}

// threatProtectionState fetches the Microsoft Defender for Cloud advanced threat protection state.
func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) threatProtectionState() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	resourceID, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return "", err
	}
	server, err := resourceID.Component("flexibleServers")
	if err != nil {
		return "", err
	}
	client, err := flexible.NewAdvancedThreatProtectionSettingsClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return "", err
	}
	resp, err := client.Get(ctx, resourceID.ResourceGroup, server, flexible.AdvancedThreatProtectionNameDefault, nil)
	if err != nil {
		return "", err
	}
	if resp.Properties == nil || resp.Properties.State == nil {
		return "", nil
	}
	return string(*resp.Properties.State), nil
}

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) azureAdAdministrators() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	resourceID, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := resourceID.Component("flexibleServers")
	if err != nil {
		return nil, err
	}
	adminClient, err := flexible.NewAzureADAdministratorsClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := adminClient.NewListByServerPager(resourceID.ResourceGroup, server, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			if entry == nil {
				continue
			}
			var administratorType, login, sid, tenantId *string
			if entry.Properties != nil {
				administratorType = (*string)(entry.Properties.AdministratorType)
				login = entry.Properties.Login
				sid = entry.Properties.Sid
				tenantId = entry.Properties.TenantID
			}
			mqlAdmin, err := CreateResource(a.MqlRuntime, "azure.subscription.mySqlService.flexibleServer.administrator",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(entry.ID),
					"name":              llx.StringDataPtr(entry.Name),
					"type":              llx.StringDataPtr(entry.Type),
					"administratorType": llx.StringDataPtr(administratorType),
					"login":             llx.StringDataPtr(login),
					"sid":               llx.StringDataPtr(sid),
					"tenantId":          llx.StringDataPtr(tenantId),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAdmin)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) privateEndpointConnections() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	resourceID, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := resourceID.Component("flexibleServers")
	if err != nil {
		return nil, err
	}
	client, err := flexible.NewPrivateEndpointConnectionsClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	resp, err := client.ListByServer(ctx, resourceID.ResourceGroup, server, nil)
	if err != nil {
		return nil, err
	}
	for _, pec := range resp.Value {
		if pec == nil {
			continue
		}
		args := map[string]*llx.RawData{
			"__id": llx.StringDataPtr(pec.ID),
			"id":   llx.StringDataPtr(pec.ID),
			"name": llx.StringDataPtr(pec.Name),
			"type": llx.StringDataPtr(pec.Type),
		}
		if pec.Properties != nil {
			propsMap, err := convert.JsonToDict(pec.Properties)
			if err != nil {
				return nil, err
			}
			args["properties"] = llx.DictData(propsMap)
			if pec.Properties.PrivateEndpoint != nil {
				args["privateEndpointId"] = llx.StringDataPtr(pec.Properties.PrivateEndpoint.ID)
			}
			if pec.Properties.ProvisioningState != nil {
				args["provisioningState"] = llx.StringData(string(*pec.Properties.ProvisioningState))
			}
			if pec.Properties.PrivateLinkServiceConnectionState != nil {
				stateArgs := map[string]*llx.RawData{}
				if pec.Properties.PrivateLinkServiceConnectionState.ActionsRequired != nil {
					stateArgs["actionsRequired"] = llx.StringDataPtr(pec.Properties.PrivateLinkServiceConnectionState.ActionsRequired)
				}
				if pec.Properties.PrivateLinkServiceConnectionState.Description != nil {
					stateArgs["description"] = llx.StringDataPtr(pec.Properties.PrivateLinkServiceConnectionState.Description)
				}
				if pec.Properties.PrivateLinkServiceConnectionState.Status != nil {
					stateArgs["status"] = llx.StringData(string(*pec.Properties.PrivateLinkServiceConnectionState.Status))
				}
				stateRes, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionPrivateEndpointConnectionConnectionState, stateArgs)
				if err != nil {
					return nil, err
				}
				args["privateLinkServiceConnectionState"] = llx.ResourceData(stateRes, ResourceAzureSubscriptionPrivateEndpointConnectionConnectionState)
			}
		}
		mqlConn, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionPrivateEndpointConnection, args)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlConn)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) userAssignedIdentities() ([]any, error) {
	return resolveUserAssignedIdentities(a.MqlRuntime, a.cacheUserAssignedIdentityIds)
}

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) dataEncryptionKey() (*mqlAzureSubscriptionKeyVaultServiceKey, error) {
	if a.cacheDataEncryptionKeyURI == "" {
		a.DataEncryptionKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return newKeyVaultKeyResource(a.MqlRuntime, a.cacheDataEncryptionKeyURI)
}

func (a *mqlAzureSubscriptionMySqlServiceFlexibleServer) geoBackupEncryptionKey() (*mqlAzureSubscriptionKeyVaultServiceKey, error) {
	if a.cacheGeoBackupKeyURI == "" {
		a.GeoBackupEncryptionKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return newKeyVaultKeyResource(a.MqlRuntime, a.cacheGeoBackupKeyURI)
}

func initAzureSubscriptionMySqlServiceServer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure mysql server")
	}
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	res, err := NewResource(runtime, "azure.subscription.mySqlService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	mysql := res.(*mqlAzureSubscriptionMySqlService)
	servers := mysql.GetServers()
	if servers.Error != nil {
		return nil, nil, servers.Error
	}
	id := args["id"].Value.(string)
	for _, entry := range servers.Data {
		vm := entry.(*mqlAzureSubscriptionMySqlServiceServer)
		if vm.Id.Data == id {
			return args, vm, nil
		}
	}

	return nil, nil, errors.New("azure mysql server does not exist")
}

func initAzureSubscriptionMySqlServiceFlexibleServer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure mysql flexible server")
	}
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	res, err := NewResource(runtime, "azure.subscription.mySqlService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	mysql := res.(*mqlAzureSubscriptionMySqlService)
	servers := mysql.GetFlexibleServers()
	if servers.Error != nil {
		return nil, nil, servers.Error
	}
	id := args["id"].Value.(string)
	for _, entry := range servers.Data {
		vm := entry.(*mqlAzureSubscriptionMySqlServiceFlexibleServer)
		if vm.Id.Data == id {
			return args, vm, nil
		}
	}

	return nil, nil, errors.New("azure mysql flexible server does not exist")
}
