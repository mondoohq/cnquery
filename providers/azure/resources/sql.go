// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"

	sql "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
)

func (a *mqlAzureSubscriptionSqlService) id() (string, error) {
	return "azure.subscription.sql/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionSqlService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionSqlServiceDatabase) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceDatabaseusage) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceServer) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceServerAdministrator) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceConfiguration) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceServerVulnerabilityassessmentsettings) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceFirewallrule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlService) servers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	dbClient, err := sql.NewServersClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbClient.NewListPager(&sql.ServersClientListOptions{})
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

			var minimalTlsVersion *string
			var publicNetworkAccess *string
			var restrictOutboundNetworkAccess *string
			var version *string
			var state *string
			var fullyQualifiedDomainName *string
			var administratorLogin *string
			if dbServer.Properties != nil {
				minimalTlsVersion = dbServer.Properties.MinimalTLSVersion
				publicNetworkAccess = (*string)(dbServer.Properties.PublicNetworkAccess)
				restrictOutboundNetworkAccess = (*string)(dbServer.Properties.RestrictOutboundNetworkAccess)
				version = dbServer.Properties.Version
				state = dbServer.Properties.State
				fullyQualifiedDomainName = dbServer.Properties.FullyQualifiedDomainName
				administratorLogin = dbServer.Properties.AdministratorLogin
			}

			mqlAzureDbServer, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.server",
				map[string]*llx.RawData{
					"id":                            llx.StringDataPtr(dbServer.ID),
					"name":                          llx.StringDataPtr(dbServer.Name),
					"location":                      llx.StringDataPtr(dbServer.Location),
					"tags":                          llx.MapData(convert.PtrMapStrToInterface(dbServer.Tags), types.String),
					"type":                          llx.StringDataPtr(dbServer.Type),
					"properties":                    llx.DictData(properties),
					"minimalTlsVersion":             llx.StringDataPtr(minimalTlsVersion),
					"publicNetworkAccess":           llx.StringDataPtr(publicNetworkAccess),
					"restrictOutboundNetworkAccess": llx.StringDataPtr(restrictOutboundNetworkAccess),
					"version":                       llx.StringDataPtr(version),
					"state":                         llx.StringDataPtr(state),
					"fullyQualifiedDomainName":      llx.StringDataPtr(fullyQualifiedDomainName),
					"administratorLogin":            llx.StringDataPtr(administratorLogin),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureDbServer)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionSqlServiceServer) databases() ([]any, error) {
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
	dbDatabaseClient, err := sql.NewDatabasesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := dbDatabaseClient.NewListByServerPager(resourceID.ResourceGroup, server, &sql.DatabasesClientListByServerOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			args := map[string]*llx.RawData{
				"id":               llx.StringDataPtr(entry.ID),
				"name":             llx.StringDataPtr(entry.Name),
				"type":             llx.StringDataPtr(entry.Type),
				"collation":        llx.StringDataPtr(entry.Properties.Collation),
				"creationDate":     llx.TimeDataPtr(entry.Properties.CreationDate),
				"databaseId":       llx.StringDataPtr(entry.Properties.DatabaseID),
				"createMode":       llx.StringDataPtr((*string)(entry.Properties.CreateMode)),
				"sourceDatabaseId": llx.StringDataPtr(entry.Properties.SourceDatabaseID),
				"recoveryServicesRecoveryPointResourceId": llx.StringDataPtr(entry.Properties.RecoveryServicesRecoveryPointID),
				"edition":                       llx.StringDataPtr(entry.SKU.Tier),
				"maxSizeBytes":                  llx.IntDataDefault(entry.Properties.MaxSizeBytes, 0),
				"requestedServiceObjectiveName": llx.StringDataPtr(entry.Properties.RequestedServiceObjectiveName),
				"serviceLevelObjective":         llx.StringDataPtr(entry.Properties.CurrentServiceObjectiveName),
				"status":                        llx.StringDataPtr((*string)(entry.Properties.Status)),
				"elasticPoolName":               llx.StringDataPtr(entry.Properties.ElasticPoolID),
				"defaultSecondaryLocation":      llx.StringDataPtr(entry.Properties.DefaultSecondaryLocation),
				"failoverGroupId":               llx.StringDataPtr(entry.Properties.FailoverGroupID),
				"readScale":                     llx.StringDataPtr((*string)(entry.Properties.ReadScale)),
				"sampleName":                    llx.StringDataPtr((*string)(entry.Properties.SampleName)),
				"zoneRedundant":                 llx.BoolDataPtr(entry.Properties.ZoneRedundant),
				"earliestRestoreDate":           llx.TimeDataPtr(entry.Properties.EarliestRestoreDate),
				"sourceDatabaseDeletionDate":    llx.TimeDataPtr(entry.Properties.SourceDatabaseDeletionDate),
				"restorePointInTime":            llx.TimeDataPtr(entry.Properties.RestorePointInTime),
			}

			mqlAzureDatabase, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.database", args)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureDatabase)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionSqlServiceServer) firewallRules() ([]any, error) {
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

	dbFirewallClient, err := sql.NewFirewallRulesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := dbFirewallClient.NewListByServerPager(resourceID.ResourceGroup, server, &sql.FirewallRulesClientListByServerOptions{})
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

func (a *mqlAzureSubscriptionSqlServiceServer) virtualNetworkRules() ([]any, error) {
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

	client, err := sql.NewVirtualNetworkRulesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListByServerPager(resourceID.ResourceGroup, server, &sql.VirtualNetworkRulesClientListByServerOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			properties, err := convert.JsonToDict(entry)
			if err != nil {
				return nil, err
			}

			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.virtualNetworkRule",
				map[string]*llx.RawData{
					"id":                     llx.StringDataPtr(entry.ID),
					"name":                   llx.StringDataPtr(entry.Name),
					"type":                   llx.StringDataPtr(entry.Type),
					"properties":             llx.DictData(properties),
					"virtualNetworkSubnetId": llx.StringDataPtr(entry.Properties.VirtualNetworkSubnetID),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionSqlServiceServer) azureAdAdministrators() ([]any, error) {
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
	administratorClient, err := sql.NewServerAzureADAdministratorsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := administratorClient.NewListByServerPager(resourceID.ResourceGroup, server, &sql.ServerAzureADAdministratorsClientListByServerOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureSqlAdministrator, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.server.administrator",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(entry.ID),
					"name":              llx.StringDataPtr(entry.Name),
					"type":              llx.StringDataPtr(entry.Type),
					"administratorType": llx.StringDataPtr((*string)(entry.Properties.AdministratorType)),
					"login":             llx.StringDataPtr(entry.Properties.Login),
					"sid":               llx.StringDataPtr(entry.Properties.Sid),
					"tenantId":          llx.StringDataPtr(entry.Properties.TenantID),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureSqlAdministrator)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionSqlServiceServer) connectionPolicy() (any, error) {
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

	connectionClient, err := sql.NewServerConnectionPoliciesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	policy, err := connectionClient.Get(ctx, resourceID.ResourceGroup, server, sql.ConnectionPolicyNameDefault, &sql.ServerConnectionPoliciesClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(policy)
}

func (a *mqlAzureSubscriptionSqlServiceServer) securityAlertPolicy() (any, error) {
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

	secAlertClient, err := sql.NewServerSecurityAlertPoliciesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	policy, err := secAlertClient.Get(ctx, resourceID.ResourceGroup, server, sql.SecurityAlertPolicyNameDefault, &sql.ServerSecurityAlertPoliciesClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(policy.ServerSecurityAlertPolicy.Properties)
}

func (a *mqlAzureSubscriptionSqlServiceServer) auditingPolicy() (any, error) {
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
	auditClient, err := sql.NewServerBlobAuditingPoliciesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	policy, err := auditClient.Get(ctx, resourceID.ResourceGroup, server, &sql.ServerBlobAuditingPoliciesClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(policy.ServerBlobAuditingPolicy.Properties)
}

func (a *mqlAzureSubscriptionSqlServiceServer) threatDetectionPolicy() (any, error) {
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

	serverClient, err := sql.NewServerAdvancedThreatProtectionSettingsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	threatPolicy, err := serverClient.Get(ctx, resourceID.ResourceGroup, server, sql.AdvancedThreatProtectionNameDefault, &sql.ServerAdvancedThreatProtectionSettingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(threatPolicy.Properties)
}

func (a *mqlAzureSubscriptionSqlServiceServer) encryptionProtector() (any, error) {
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

	client, err := sql.NewEncryptionProtectorsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	policy, err := client.Get(ctx, resourceID.ResourceGroup, server, sql.EncryptionProtectorNameCurrent, &sql.EncryptionProtectorsClientGetOptions{})
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(policy.EncryptionProtector.Properties)
}

func (a *mqlAzureSubscriptionSqlServiceServer) vulnerabilityAssessmentSettings() (*mqlAzureSubscriptionSqlServiceServerVulnerabilityassessmentsettings, error) {
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

	serverClient, err := sql.NewServerVulnerabilityAssessmentsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	vaSettings, err := serverClient.Get(ctx, resourceID.ResourceGroup, server, sql.VulnerabilityAssessmentNameDefault, &sql.ServerVulnerabilityAssessmentsClientGetOptions{})
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.server.vulnerabilityassessmentsettings",
		map[string]*llx.RawData{
			"id":                      llx.StringDataPtr(vaSettings.ID),
			"name":                    llx.StringDataPtr(vaSettings.Name),
			"type":                    llx.StringDataPtr(vaSettings.Type),
			"storageContainerPath":    llx.StringDataPtr(vaSettings.Properties.StorageContainerPath),
			"storageAccountAccessKey": llx.StringDataPtr(vaSettings.Properties.StorageAccountAccessKey),
			"storageContainerSasKey":  llx.StringDataPtr(vaSettings.Properties.StorageContainerSasKey),
			"recurringScanEnabled":    llx.BoolDataPtr(vaSettings.Properties.RecurringScans.IsEnabled),
			"recurringScanEmails":     llx.ArrayData(llx.TArr2Raw(convert.ToListFromPtrs(vaSettings.Properties.RecurringScans.Emails)), types.String),
			"mailSubscriptionAdmins":  llx.BoolDataPtr(vaSettings.Properties.RecurringScans.EmailSubscriptionAdmins),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSqlServiceServerVulnerabilityassessmentsettings), err
}

func (a *mqlAzureSubscriptionSqlServiceDatabase) transparentDataEncryption() (any, error) {
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

	database, err := resourceID.Component("databases")
	if err != nil {
		return nil, err
	}

	client, err := sql.NewTransparentDataEncryptionsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	policy, err := client.Get(ctx, resourceID.ResourceGroup, server, database, sql.TransparentDataEncryptionNameCurrent, &sql.TransparentDataEncryptionsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(policy.LogicalDatabaseTransparentDataEncryption.Properties)
}

func (a *mqlAzureSubscriptionSqlServiceDatabase) advisor() ([]any, error) {
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

	database, err := resourceID.Component("databases")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewDatabaseAdvisorsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	// it's an OData API, supports $expand. We can get the recommendedActions for all advisors here.
	expandRecommendedActions := "recommendedActions"
	advisors, err := client.ListByDatabase(ctx, resourceID.ResourceGroup, server, database, &sql.DatabaseAdvisorsClientListByDatabaseOptions{Expand: &expandRecommendedActions})
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, entry := range advisors.AdvisorArray {
		dict, err := convert.JsonToDict(entry)
		if err != nil {
			return nil, err
		}

		res = append(res, dict)
	}

	return res, nil
}

func (a *mqlAzureSubscriptionSqlServiceDatabase) threatDetectionPolicy() (any, error) {
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

	database, err := resourceID.Component("databases")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewDatabaseSecurityAlertPoliciesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	policy, err := client.Get(ctx, resourceID.ResourceGroup, server, database, sql.SecurityAlertPolicyNameDefault, &sql.DatabaseSecurityAlertPoliciesClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(policy.DatabaseSecurityAlertPolicy.Properties)
}

func (a *mqlAzureSubscriptionSqlServiceDatabase) connectionPolicy() (any, error) {
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

	connectionClient, err := sql.NewServerConnectionPoliciesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	policy, err := connectionClient.Get(ctx, resourceID.ResourceGroup, server, sql.ConnectionPolicyNameDefault, &sql.ServerConnectionPoliciesClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(policy.ServerConnectionPolicy.Properties)
}

func (a *mqlAzureSubscriptionSqlServiceDatabase) auditingPolicy() (any, error) {
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

	database, err := resourceID.Component("databases")
	if err != nil {
		return nil, err
	}

	auditClient, err := sql.NewDatabaseBlobAuditingPoliciesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	policy, err := auditClient.Get(ctx, resourceID.ResourceGroup, server, database, &sql.DatabaseBlobAuditingPoliciesClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(policy.DatabaseBlobAuditingPolicy.Properties)
}

func (a *mqlAzureSubscriptionSqlServiceDatabase) advancedThreatProtection() (*mqlAzureSubscriptionSqlServiceDatabaseAdvancedthreatprotection, error) {
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

	database, err := resourceID.Component("databases")
	if err != nil {
		return nil, err
	}

	client, err := sql.NewDatabaseAdvancedThreatProtectionSettingsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	policy, err := client.Get(ctx, resourceID.ResourceGroup, server, database, sql.AdvancedThreatProtectionNameDefault, &sql.DatabaseAdvancedThreatProtectionSettingsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	var state string
	var creationTime *time.Time
	if policy.Properties != nil {
		if policy.Properties.State != nil {
			state = string(*policy.Properties.State)
		}
		creationTime = policy.Properties.CreationTime
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.database.advancedthreatprotection",
		map[string]*llx.RawData{
			"id":           llx.StringDataPtr(policy.ID),
			"state":        llx.StringData(state),
			"creationTime": llx.TimeDataPtr(creationTime),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSqlServiceDatabaseAdvancedthreatprotection), nil
}

func (a *mqlAzureSubscriptionSqlServiceDatabase) backupShortTermRetentionPolicy() (*mqlAzureSubscriptionSqlServiceDatabaseBackupshorttermretentionpolicy, error) {
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

	database, err := resourceID.Component("databases")
	if err != nil {
		return nil, err
	}

	client, err := sql.NewBackupShortTermRetentionPoliciesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	policy, err := client.Get(ctx, resourceID.ResourceGroup, server, database, sql.ShortTermRetentionPolicyNameDefault, &sql.BackupShortTermRetentionPoliciesClientGetOptions{})
	if err != nil {
		return nil, err
	}

	var retentionDays, diffBackupInterval int64
	if policy.Properties != nil {
		if policy.Properties.RetentionDays != nil {
			retentionDays = int64(*policy.Properties.RetentionDays)
		}
		if policy.Properties.DiffBackupIntervalInHours != nil {
			diffBackupInterval = int64(*policy.Properties.DiffBackupIntervalInHours)
		}
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.database.backupshorttermretentionpolicy",
		map[string]*llx.RawData{
			"id":                        llx.StringDataPtr(policy.ID),
			"retentionDays":             llx.IntData(retentionDays),
			"diffBackupIntervalInHours": llx.IntData(diffBackupInterval),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSqlServiceDatabaseBackupshorttermretentionpolicy), nil
}

func (a *mqlAzureSubscriptionSqlServiceDatabase) longTermRetentionPolicy() (*mqlAzureSubscriptionSqlServiceDatabaseLongtermretentionpolicy, error) {
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

	database, err := resourceID.Component("databases")
	if err != nil {
		return nil, err
	}

	client, err := sql.NewLongTermRetentionPoliciesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	policy, err := client.Get(ctx, resourceID.ResourceGroup, server, database, sql.LongTermRetentionPolicyNameDefault, &sql.LongTermRetentionPoliciesClientGetOptions{})
	if err != nil {
		return nil, err
	}

	var weekOfYear int64
	var weeklyRetention, monthlyRetention, yearlyRetention *string
	if policy.Properties != nil {
		weeklyRetention = policy.Properties.WeeklyRetention
		monthlyRetention = policy.Properties.MonthlyRetention
		yearlyRetention = policy.Properties.YearlyRetention
		if policy.Properties.WeekOfYear != nil {
			weekOfYear = int64(*policy.Properties.WeekOfYear)
		}
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.database.longtermretentionpolicy",
		map[string]*llx.RawData{
			"id":               llx.StringDataPtr(policy.ID),
			"weeklyRetention":  llx.StringDataPtr(weeklyRetention),
			"monthlyRetention": llx.StringDataPtr(monthlyRetention),
			"yearlyRetention":  llx.StringDataPtr(yearlyRetention),
			"weekOfYear":       llx.IntData(weekOfYear),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSqlServiceDatabaseLongtermretentionpolicy), nil
}

func (a *mqlAzureSubscriptionSqlServiceDatabase) usage() ([]any, error) {
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

	database, err := resourceID.Component("databases")
	if err != nil {
		return nil, err
	}

	client, err := sql.NewDatabaseUsagesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := client.NewListByDatabasePager(resourceID.ResourceGroup, server, database, &sql.DatabaseUsagesClientListByDatabaseOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureSqlUsage, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.databaseusage",
				map[string]*llx.RawData{
					"id":           llx.StringDataPtr(entry.ID),
					"name":         llx.StringDataPtr(entry.Name),
					"resourceName": llx.StringDataPtr(entry.Name),
					"displayName":  llx.StringDataPtr(entry.Properties.DisplayName),
					"currentValue": llx.FloatData(convert.ToValue(entry.Properties.CurrentValue)),
					"limit":        llx.FloatData(convert.ToValue(entry.Properties.Limit)),
					"unit":         llx.StringDataPtr(entry.Properties.Unit),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureSqlUsage)
		}
	}

	return res, nil
}

func initAzureSubscriptionSqlServiceServer(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure sql database server")
	}
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	res, err := NewResource(runtime, "azure.subscription.sqlService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	sqlSvc := res.(*mqlAzureSubscriptionSqlService)
	servers := sqlSvc.GetServers()
	if servers.Error != nil {
		return nil, nil, servers.Error
	}
	id := args["id"].Value.(string)
	for _, entry := range servers.Data {
		vm := entry.(*mqlAzureSubscriptionSqlServiceServer)
		if vm.Id.Data == id {
			return args, vm, nil
		}
	}

	return nil, nil, errors.New("azure sql database server does not exist")
}
