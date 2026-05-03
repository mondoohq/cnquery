// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"

	sql "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
)

type mqlAzureSubscriptionSqlServiceServerInternal struct {
	encryptionProtectorOnce sync.Once
	encryptionProtectorResp *sql.EncryptionProtectorsClientGetResponse
	encryptionProtectorErr  error
}

type mqlAzureSubscriptionSqlServiceServerFailoverGroupInternal struct {
	cacheDatabaseIds []string
}

func (a *mqlAzureSubscriptionSqlServiceServer) fetchEncryptionProtector() (*sql.EncryptionProtectorsClientGetResponse, error) {
	a.encryptionProtectorOnce.Do(func() {
		conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
		ctx := context.Background()
		token := conn.Token()
		id := a.Id.Data
		resourceID, err := ParseResourceID(id)
		if err != nil {
			a.encryptionProtectorErr = err
			return
		}

		server, err := resourceID.Component("servers")
		if err != nil {
			a.encryptionProtectorErr = err
			return
		}

		client, err := sql.NewEncryptionProtectorsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
			ClientOptions: conn.ClientOptions(),
		})
		if err != nil {
			a.encryptionProtectorErr = err
			return
		}
		resp, err := client.Get(ctx, resourceID.ResourceGroup, server, sql.EncryptionProtectorNameCurrent, &sql.EncryptionProtectorsClientGetOptions{})
		a.encryptionProtectorResp = &resp
		a.encryptionProtectorErr = err
	})
	return a.encryptionProtectorResp, a.encryptionProtectorErr
}

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
	resp, err := a.fetchEncryptionProtector()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(resp.EncryptionProtector.Properties)
}

func (a *mqlAzureSubscriptionSqlServiceServer) encryptionProtectorServerKeyType() (string, error) {
	resp, err := a.fetchEncryptionProtector()
	if err != nil {
		return "", err
	}
	if resp.Properties != nil && resp.Properties.ServerKeyType != nil {
		return string(*resp.Properties.ServerKeyType), nil
	}
	return "", nil
}

func (a *mqlAzureSubscriptionSqlServiceServer) encryptionProtectorKey() (*mqlAzureSubscriptionKeyVaultServiceKey, error) {
	resp, err := a.fetchEncryptionProtector()
	if err != nil {
		return nil, err
	}
	if resp.Properties == nil || resp.Properties.ServerKeyType == nil || string(*resp.Properties.ServerKeyType) != "AzureKeyVault" {
		a.EncryptionProtectorKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	if resp.Properties.URI != nil {
		return newKeyVaultKeyResource(a.MqlRuntime, *resp.Properties.URI)
	}
	a.EncryptionProtectorKey.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAzureSubscriptionSqlServiceServer) azureAdOnlyAuthentication() (bool, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return false, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return false, err
	}

	client, err := sql.NewServerAzureADOnlyAuthenticationsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return false, err
	}
	result, err := client.Get(ctx, resourceID.ResourceGroup, server, sql.AuthenticationNameDefault, &sql.ServerAzureADOnlyAuthenticationsClientGetOptions{})
	if err != nil {
		return false, nil
	}
	if result.Properties != nil && result.Properties.AzureADOnlyAuthentication != nil {
		return *result.Properties.AzureADOnlyAuthentication, nil
	}
	return false, nil
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

func initAzureSubscriptionSqlServiceDatabase(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	if args["id"] == nil {
		return args, nil, nil
	}
	dbId := args["id"].Value.(string)
	rid, err := ParseResourceID(dbId)
	if err != nil {
		return args, nil, nil
	}
	serverName, err := rid.Component("servers")
	if err != nil {
		return args, nil, nil
	}
	serverId := "/subscriptions/" + rid.SubscriptionID + "/resourceGroups/" + rid.ResourceGroup + "/providers/Microsoft.Sql/servers/" + serverName
	serverRes, err := NewResource(runtime, "azure.subscription.sqlService.server",
		map[string]*llx.RawData{"id": llx.StringData(serverId)})
	if err != nil {
		return args, nil, nil
	}
	server := serverRes.(*mqlAzureSubscriptionSqlServiceServer)
	dbs := server.GetDatabases()
	if dbs.Error != nil {
		return args, nil, nil
	}
	for _, entry := range dbs.Data {
		db := entry.(*mqlAzureSubscriptionSqlServiceDatabase)
		if db.Id.Data == dbId {
			return args, db, nil
		}
	}
	return args, nil, nil
}

// ---------------------------------------------------------------------------
// __id methods for new SQL security resources
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceServerBlobAuditingPolicy) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceServerEncryptionProtectorConfig) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceServerSecurityAlertPolicyConfig) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceServerAdvancedThreatProtectionSetting) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceServerDevOpsAuditingSetting) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceServerKey) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceServerOutboundFirewallRule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceServerFailoverGroup) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceServerReplicationLink) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceVulnerabilityAssessmentScan) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceDatabaseBlobAuditingPolicy) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceDatabaseSecurityAlertPolicy) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceDatabaseVulnerabilityAssessment) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceDatabaseDataMaskingPolicy) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceDatabaseDataMaskingRule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceDatabaseLedgerDigestUpload) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionSqlServiceDatabaseGeoBackupPolicy) id() (string, error) {
	return a.Id.Data, nil
}

// ---------------------------------------------------------------------------
// Server-level: connection type (replaces deprecated connectionPolicy dict)
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceServer) connectionType() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return "", err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return "", err
	}
	client, err := sql.NewServerConnectionPoliciesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return "", err
	}
	resp, err := client.Get(ctx, rid.ResourceGroup, server, sql.ConnectionPolicyNameDefault, nil)
	if err != nil {
		return "", err
	}
	if resp.Properties != nil && resp.Properties.ConnectionType != nil {
		return string(*resp.Properties.ConnectionType), nil
	}
	return "", nil
}

// ---------------------------------------------------------------------------
// Server-level: blob auditing policy (typed replacement for auditingPolicy dict)
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceServer) blobAuditingPolicy() (*mqlAzureSubscriptionSqlServiceServerBlobAuditingPolicy, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewServerBlobAuditingPoliciesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(ctx, rid.ResourceGroup, server, nil)
	if err != nil {
		return nil, err
	}

	args := serverBlobAuditingPolicyArgs(resp.ID, resp.Properties)
	res, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.server.blobAuditingPolicy", args)
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSqlServiceServerBlobAuditingPolicy), nil
}

func serverBlobAuditingPolicyArgs(id *string, props *sql.ServerBlobAuditingPolicyProperties) map[string]*llx.RawData {
	var (
		state                        string
		actions                      []any
		isAzureMonitor               bool
		isManagedIdentity            bool
		isStorageSecondary           bool
		queueDelayMs                 int64
		retentionDays                int64
		storageAccountSubscriptionId string
		storageEndpoint              string
	)
	if props != nil {
		if props.State != nil {
			state = string(*props.State)
		}
		actions = llx.TArr2Raw(convert.ToListFromPtrs(props.AuditActionsAndGroups))
		isAzureMonitor = convert.ToValue(props.IsAzureMonitorTargetEnabled)
		isManagedIdentity = convert.ToValue(props.IsManagedIdentityInUse)
		isStorageSecondary = convert.ToValue(props.IsStorageSecondaryKeyInUse)
		if props.QueueDelayMs != nil {
			queueDelayMs = int64(*props.QueueDelayMs)
		}
		if props.RetentionDays != nil {
			retentionDays = int64(*props.RetentionDays)
		}
		if props.StorageAccountSubscriptionID != nil {
			storageAccountSubscriptionId = *props.StorageAccountSubscriptionID
		}
		if props.StorageEndpoint != nil {
			storageEndpoint = *props.StorageEndpoint
		}
	}
	return map[string]*llx.RawData{
		"id":                           llx.StringDataPtr(id),
		"state":                        llx.StringData(state),
		"auditActionsAndGroups":        llx.ArrayData(actions, types.String),
		"isAzureMonitorTargetEnabled":  llx.BoolData(isAzureMonitor),
		"isManagedIdentityInUse":       llx.BoolData(isManagedIdentity),
		"isStorageSecondaryKeyInUse":   llx.BoolData(isStorageSecondary),
		"queueDelayMs":                 llx.IntData(queueDelayMs),
		"retentionDays":                llx.IntData(retentionDays),
		"storageAccountSubscriptionId": llx.StringData(storageAccountSubscriptionId),
		"storageEndpoint":              llx.StringData(storageEndpoint),
	}
}

// ---------------------------------------------------------------------------
// Server-level: encryption protector config (typed)
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceServer) encryptionProtectorConfig() (*mqlAzureSubscriptionSqlServiceServerEncryptionProtectorConfig, error) {
	resp, err := a.fetchEncryptionProtector()
	if err != nil {
		return nil, err
	}

	var (
		serverKeyName, uri, thumbprint, subregion *string
		serverKeyType                             string
		autoRotation                              *bool
	)
	if resp.Properties != nil {
		serverKeyName = resp.Properties.ServerKeyName
		if resp.Properties.ServerKeyType != nil {
			serverKeyType = string(*resp.Properties.ServerKeyType)
		}
		uri = resp.Properties.URI
		thumbprint = resp.Properties.Thumbprint
		subregion = resp.Properties.Subregion
		autoRotation = resp.Properties.AutoRotationEnabled
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.server.encryptionProtectorConfig",
		map[string]*llx.RawData{
			"id":                  llx.StringDataPtr(resp.ID),
			"serverKeyName":       llx.StringDataPtr(serverKeyName),
			"serverKeyType":       llx.StringData(serverKeyType),
			"uri":                 llx.StringDataPtr(uri),
			"thumbprint":          llx.StringDataPtr(thumbprint),
			"subregion":           llx.StringDataPtr(subregion),
			"autoRotationEnabled": llx.BoolDataPtr(autoRotation),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSqlServiceServerEncryptionProtectorConfig), nil
}

func (a *mqlAzureSubscriptionSqlServiceServerEncryptionProtectorConfig) key() (*mqlAzureSubscriptionKeyVaultServiceKey, error) {
	if a.ServerKeyType.Data != "AzureKeyVault" || a.Uri.Data == "" {
		a.Key.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return newKeyVaultKeyResource(a.MqlRuntime, a.Uri.Data)
}

// ---------------------------------------------------------------------------
// Server-level: security alert policy (typed) and advanced threat protection
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceServer) securityAlertPolicyConfig() (*mqlAzureSubscriptionSqlServiceServerSecurityAlertPolicyConfig, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewServerSecurityAlertPoliciesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(ctx, rid.ResourceGroup, server, sql.SecurityAlertPolicyNameDefault, nil)
	if err != nil {
		return nil, err
	}

	var (
		state           string
		disabledAlerts  []any
		emailAddresses  []any
		emailAdmins     bool
		storageEndpoint string
		retentionDays   int64
		creation        *time.Time
	)
	if resp.Properties != nil {
		if resp.Properties.State != nil {
			state = string(*resp.Properties.State)
		}
		disabledAlerts = llx.TArr2Raw(convert.ToListFromPtrs(resp.Properties.DisabledAlerts))
		emailAddresses = llx.TArr2Raw(convert.ToListFromPtrs(resp.Properties.EmailAddresses))
		emailAdmins = convert.ToValue(resp.Properties.EmailAccountAdmins)
		if resp.Properties.StorageEndpoint != nil {
			storageEndpoint = *resp.Properties.StorageEndpoint
		}
		if resp.Properties.RetentionDays != nil {
			retentionDays = int64(*resp.Properties.RetentionDays)
		}
		creation = resp.Properties.CreationTime
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.server.securityAlertPolicyConfig",
		map[string]*llx.RawData{
			"id":                 llx.StringDataPtr(resp.ID),
			"state":              llx.StringData(state),
			"disabledAlerts":     llx.ArrayData(disabledAlerts, types.String),
			"emailAddresses":     llx.ArrayData(emailAddresses, types.String),
			"emailAccountAdmins": llx.BoolData(emailAdmins),
			"storageEndpoint":    llx.StringData(storageEndpoint),
			"retentionDays":      llx.IntData(retentionDays),
			"creationTime":       llx.TimeDataPtr(creation),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSqlServiceServerSecurityAlertPolicyConfig), nil
}

func (a *mqlAzureSubscriptionSqlServiceServer) advancedThreatProtectionSetting() (*mqlAzureSubscriptionSqlServiceServerAdvancedThreatProtectionSetting, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewServerAdvancedThreatProtectionSettingsClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(ctx, rid.ResourceGroup, server, sql.AdvancedThreatProtectionNameDefault, nil)
	if err != nil {
		return nil, err
	}

	var (
		state    string
		creation *time.Time
	)
	if resp.Properties != nil {
		if resp.Properties.State != nil {
			state = string(*resp.Properties.State)
		}
		creation = resp.Properties.CreationTime
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.server.advancedThreatProtectionSetting",
		map[string]*llx.RawData{
			"id":           llx.StringDataPtr(resp.ID),
			"state":        llx.StringData(state),
			"creationTime": llx.TimeDataPtr(creation),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSqlServiceServerAdvancedThreatProtectionSetting), nil
}

// ---------------------------------------------------------------------------
// Server-level: DevOps audit setting
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceServer) devOpsAuditingSetting() (*mqlAzureSubscriptionSqlServiceServerDevOpsAuditingSetting, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewServerDevOpsAuditSettingsClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(ctx, rid.ResourceGroup, server, "default", nil)
	if err != nil {
		return nil, err
	}

	var (
		state                        string
		isAzureMonitor               bool
		storageAccountSubscriptionId string
		storageEndpoint              string
	)
	if resp.Properties != nil {
		if resp.Properties.State != nil {
			state = string(*resp.Properties.State)
		}
		isAzureMonitor = convert.ToValue(resp.Properties.IsAzureMonitorTargetEnabled)
		if resp.Properties.StorageAccountSubscriptionID != nil {
			storageAccountSubscriptionId = *resp.Properties.StorageAccountSubscriptionID
		}
		if resp.Properties.StorageEndpoint != nil {
			storageEndpoint = *resp.Properties.StorageEndpoint
		}
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.server.devOpsAuditingSetting",
		map[string]*llx.RawData{
			"id":                           llx.StringDataPtr(resp.ID),
			"state":                        llx.StringData(state),
			"isAzureMonitorTargetEnabled":  llx.BoolData(isAzureMonitor),
			"storageAccountSubscriptionId": llx.StringData(storageAccountSubscriptionId),
			"storageEndpoint":              llx.StringData(storageEndpoint),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSqlServiceServerDevOpsAuditingSetting), nil
}

// ---------------------------------------------------------------------------
// Server-level: customer-managed encryption keys (BYOK)
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceServer) keys() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewServerKeysClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListByServerPager(rid.ResourceGroup, server, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, k := range page.Value {
			var (
				kind, serverKeyType, uri, thumbprint, subregion string
				creation                                        *time.Time
				autoRotation                                    bool
			)
			if k.Kind != nil {
				kind = *k.Kind
			}
			if k.Properties != nil {
				if k.Properties.ServerKeyType != nil {
					serverKeyType = string(*k.Properties.ServerKeyType)
				}
				if k.Properties.URI != nil {
					uri = *k.Properties.URI
				}
				if k.Properties.Thumbprint != nil {
					thumbprint = *k.Properties.Thumbprint
				}
				if k.Properties.Subregion != nil {
					subregion = *k.Properties.Subregion
				}
				creation = k.Properties.CreationDate
				autoRotation = convert.ToValue(k.Properties.AutoRotationEnabled)
			}

			r, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.server.key",
				map[string]*llx.RawData{
					"id":                  llx.StringDataPtr(k.ID),
					"name":                llx.StringDataPtr(k.Name),
					"type":                llx.StringDataPtr(k.Type),
					"kind":                llx.StringData(kind),
					"serverKeyType":       llx.StringData(serverKeyType),
					"uri":                 llx.StringData(uri),
					"thumbprint":          llx.StringData(thumbprint),
					"creationDate":        llx.TimeDataPtr(creation),
					"autoRotationEnabled": llx.BoolData(autoRotation),
					"subregion":           llx.StringData(subregion),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}
	}
	return res, nil
}

// ---------------------------------------------------------------------------
// Server-level: outbound firewall rules
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceServer) outboundFirewallRules() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewOutboundFirewallRulesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListByServerPager(rid.ResourceGroup, server, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, r := range page.Value {
			provisioningState := ""
			if r.Properties != nil && r.Properties.ProvisioningState != nil {
				provisioningState = string(*r.Properties.ProvisioningState)
			}
			mqlRule, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.server.outboundFirewallRule",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(r.ID),
					"name":              llx.StringDataPtr(r.Name),
					"type":              llx.StringDataPtr(r.Type),
					"provisioningState": llx.StringData(provisioningState),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRule)
		}
	}
	return res, nil
}

// ---------------------------------------------------------------------------
// Server-level: private endpoint connections
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceServer) privateEndpointConnections() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewPrivateEndpointConnectionsClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListByServerPager(rid.ResourceGroup, server, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, pec := range page.Value {
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
						stateArgs["actionsRequired"] = llx.StringData(string(*pec.Properties.PrivateLinkServiceConnectionState.ActionsRequired))
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
	}
	return res, nil
}

// ---------------------------------------------------------------------------
// Server-level: failover groups
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceServer) failoverGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewFailoverGroupsClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListByServerPager(rid.ResourceGroup, server, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, fg := range page.Value {
			var (
				replicationRole, replicationState string
				partnerServers                    []any
				readWriteEndpoint                 any
				readOnlyEndpoint                  any
				databaseIds                       []string
			)
			if fg.Properties != nil {
				if fg.Properties.ReplicationRole != nil {
					replicationRole = string(*fg.Properties.ReplicationRole)
				}
				if fg.Properties.ReplicationState != nil {
					replicationState = *fg.Properties.ReplicationState
				}
				if d, derr := convert.JsonToDictSlice(fg.Properties.PartnerServers); derr == nil {
					partnerServers = d
				}
				if fg.Properties.ReadWriteEndpoint != nil {
					if d, derr := convert.JsonToDict(fg.Properties.ReadWriteEndpoint); derr == nil {
						readWriteEndpoint = d
					}
				}
				if fg.Properties.ReadOnlyEndpoint != nil {
					if d, derr := convert.JsonToDict(fg.Properties.ReadOnlyEndpoint); derr == nil {
						readOnlyEndpoint = d
					}
				}
				databaseIds = convert.ToListFromPtrs(fg.Properties.Databases)
			}

			mqlFG, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.server.failoverGroup",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(fg.ID),
					"name":              llx.StringDataPtr(fg.Name),
					"location":          llx.StringDataPtr(fg.Location),
					"tags":              llx.MapData(convert.PtrMapStrToInterface(fg.Tags), types.String),
					"replicationRole":   llx.StringData(replicationRole),
					"replicationState":  llx.StringData(replicationState),
					"partnerServers":    llx.ArrayData(partnerServers, types.Dict),
					"readWriteEndpoint": llx.DictData(readWriteEndpoint),
					"readOnlyEndpoint":  llx.DictData(readOnlyEndpoint),
				})
			if err != nil {
				return nil, err
			}
			mqlFG.(*mqlAzureSubscriptionSqlServiceServerFailoverGroup).cacheDatabaseIds = databaseIds
			res = append(res, mqlFG)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionSqlServiceServerFailoverGroup) databases() ([]any, error) {
	res := make([]any, 0, len(a.cacheDatabaseIds))
	for _, dbId := range a.cacheDatabaseIds {
		mqlDb, err := NewResource(a.MqlRuntime, "azure.subscription.sqlService.database",
			map[string]*llx.RawData{"id": llx.StringData(dbId)})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlDb)
	}
	return res, nil
}

// ---------------------------------------------------------------------------
// Server-level: replication links
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceServer) replicationLinks() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewReplicationLinksClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListByServerPager(rid.ResourceGroup, server, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, link := range page.Value {
			var (
				partnerLocation, partnerServer, partnerDatabase, partnerRole string
				replicationMode, replicationState, role, linkType            string
				isTerminationAllowed                                         bool
				startTime                                                    *time.Time
				percentComplete                                              int64
			)
			if link.Properties != nil {
				if link.Properties.PartnerLocation != nil {
					partnerLocation = *link.Properties.PartnerLocation
				}
				if link.Properties.PartnerServer != nil {
					partnerServer = *link.Properties.PartnerServer
				}
				if link.Properties.PartnerDatabase != nil {
					partnerDatabase = *link.Properties.PartnerDatabase
				}
				if link.Properties.PartnerRole != nil {
					partnerRole = string(*link.Properties.PartnerRole)
				}
				if link.Properties.ReplicationMode != nil {
					replicationMode = *link.Properties.ReplicationMode
				}
				if link.Properties.ReplicationState != nil {
					replicationState = string(*link.Properties.ReplicationState)
				}
				if link.Properties.Role != nil {
					role = string(*link.Properties.Role)
				}
				if link.Properties.LinkType != nil {
					linkType = string(*link.Properties.LinkType)
				}
				isTerminationAllowed = convert.ToValue(link.Properties.IsTerminationAllowed)
				startTime = link.Properties.StartTime
				if link.Properties.PercentComplete != nil {
					percentComplete = int64(*link.Properties.PercentComplete)
				}
			}

			mqlLink, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.server.replicationLink",
				map[string]*llx.RawData{
					"id":                   llx.StringDataPtr(link.ID),
					"name":                 llx.StringDataPtr(link.Name),
					"partnerLocation":      llx.StringData(partnerLocation),
					"partnerServer":        llx.StringData(partnerServer),
					"partnerDatabase":      llx.StringData(partnerDatabase),
					"partnerRole":          llx.StringData(partnerRole),
					"replicationMode":      llx.StringData(replicationMode),
					"replicationState":     llx.StringData(replicationState),
					"role":                 llx.StringData(role),
					"isTerminationAllowed": llx.BoolData(isTerminationAllowed),
					"linkType":             llx.StringData(linkType),
					"startTime":            llx.TimeDataPtr(startTime),
					"percentComplete":      llx.IntData(percentComplete),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlLink)
		}
	}
	return res, nil
}

// ---------------------------------------------------------------------------
// Database-level: TDE flat field
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceDatabase) transparentDataEncryptionEnabled() (bool, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return false, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return false, err
	}
	database, err := rid.Component("databases")
	if err != nil {
		return false, err
	}
	client, err := sql.NewTransparentDataEncryptionsClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return false, err
	}
	resp, err := client.Get(ctx, rid.ResourceGroup, server, database, sql.TransparentDataEncryptionNameCurrent, nil)
	if err != nil {
		return false, err
	}
	if resp.Properties != nil && resp.Properties.State != nil {
		return *resp.Properties.State == sql.TransparentDataEncryptionStateEnabled, nil
	}
	return false, nil
}

// ---------------------------------------------------------------------------
// Database-level: blob auditing policy (typed)
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceDatabase) blobAuditingPolicy() (*mqlAzureSubscriptionSqlServiceDatabaseBlobAuditingPolicy, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	database, err := rid.Component("databases")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewDatabaseBlobAuditingPoliciesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(ctx, rid.ResourceGroup, server, database, nil)
	if err != nil {
		return nil, err
	}

	args := databaseBlobAuditingPolicyArgs(resp.ID, resp.Properties)
	res, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.database.blobAuditingPolicy", args)
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSqlServiceDatabaseBlobAuditingPolicy), nil
}

func databaseBlobAuditingPolicyArgs(id *string, props *sql.DatabaseBlobAuditingPolicyProperties) map[string]*llx.RawData {
	var (
		state                        string
		actions                      []any
		isAzureMonitor               bool
		isManagedIdentity            bool
		isStorageSecondary           bool
		queueDelayMs                 int64
		retentionDays                int64
		storageAccountSubscriptionId string
		storageEndpoint              string
	)
	if props != nil {
		if props.State != nil {
			state = string(*props.State)
		}
		actions = llx.TArr2Raw(convert.ToListFromPtrs(props.AuditActionsAndGroups))
		isAzureMonitor = convert.ToValue(props.IsAzureMonitorTargetEnabled)
		isManagedIdentity = convert.ToValue(props.IsManagedIdentityInUse)
		isStorageSecondary = convert.ToValue(props.IsStorageSecondaryKeyInUse)
		if props.QueueDelayMs != nil {
			queueDelayMs = int64(*props.QueueDelayMs)
		}
		if props.RetentionDays != nil {
			retentionDays = int64(*props.RetentionDays)
		}
		if props.StorageAccountSubscriptionID != nil {
			storageAccountSubscriptionId = *props.StorageAccountSubscriptionID
		}
		if props.StorageEndpoint != nil {
			storageEndpoint = *props.StorageEndpoint
		}
	}
	return map[string]*llx.RawData{
		"id":                           llx.StringDataPtr(id),
		"state":                        llx.StringData(state),
		"auditActionsAndGroups":        llx.ArrayData(actions, types.String),
		"isAzureMonitorTargetEnabled":  llx.BoolData(isAzureMonitor),
		"isManagedIdentityInUse":       llx.BoolData(isManagedIdentity),
		"isStorageSecondaryKeyInUse":   llx.BoolData(isStorageSecondary),
		"queueDelayMs":                 llx.IntData(queueDelayMs),
		"retentionDays":                llx.IntData(retentionDays),
		"storageAccountSubscriptionId": llx.StringData(storageAccountSubscriptionId),
		"storageEndpoint":              llx.StringData(storageEndpoint),
	}
}

// ---------------------------------------------------------------------------
// Database-level: security alert policy (typed)
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceDatabase) securityAlertPolicy() (*mqlAzureSubscriptionSqlServiceDatabaseSecurityAlertPolicy, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	database, err := rid.Component("databases")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewDatabaseSecurityAlertPoliciesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(ctx, rid.ResourceGroup, server, database, sql.SecurityAlertPolicyNameDefault, nil)
	if err != nil {
		return nil, err
	}

	var (
		state           string
		disabledAlerts  []any
		emailAddresses  []any
		emailAdmins     bool
		storageEndpoint string
		retentionDays   int64
		creation        *time.Time
	)
	if resp.Properties != nil {
		if resp.Properties.State != nil {
			state = string(*resp.Properties.State)
		}
		disabledAlerts = llx.TArr2Raw(convert.ToListFromPtrs(resp.Properties.DisabledAlerts))
		emailAddresses = llx.TArr2Raw(convert.ToListFromPtrs(resp.Properties.EmailAddresses))
		emailAdmins = convert.ToValue(resp.Properties.EmailAccountAdmins)
		if resp.Properties.StorageEndpoint != nil {
			storageEndpoint = *resp.Properties.StorageEndpoint
		}
		if resp.Properties.RetentionDays != nil {
			retentionDays = int64(*resp.Properties.RetentionDays)
		}
		creation = resp.Properties.CreationTime
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.database.securityAlertPolicy",
		map[string]*llx.RawData{
			"id":                 llx.StringDataPtr(resp.ID),
			"state":              llx.StringData(state),
			"disabledAlerts":     llx.ArrayData(disabledAlerts, types.String),
			"emailAddresses":     llx.ArrayData(emailAddresses, types.String),
			"emailAccountAdmins": llx.BoolData(emailAdmins),
			"storageEndpoint":    llx.StringData(storageEndpoint),
			"retentionDays":      llx.IntData(retentionDays),
			"creationTime":       llx.TimeDataPtr(creation),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSqlServiceDatabaseSecurityAlertPolicy), nil
}

// ---------------------------------------------------------------------------
// Database-level: vulnerability assessment + scans
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceDatabase) vulnerabilityAssessment() (*mqlAzureSubscriptionSqlServiceDatabaseVulnerabilityAssessment, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	database, err := rid.Component("databases")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewDatabaseVulnerabilityAssessmentsClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(ctx, rid.ResourceGroup, server, database, sql.VulnerabilityAssessmentNameDefault, nil)
	if err != nil {
		return nil, err
	}

	var (
		recurringEnabled bool
		emails           []any
		mailAdmins       bool
		containerPath    string
	)
	if resp.Properties != nil {
		if resp.Properties.RecurringScans != nil {
			recurringEnabled = convert.ToValue(resp.Properties.RecurringScans.IsEnabled)
			emails = llx.TArr2Raw(convert.ToListFromPtrs(resp.Properties.RecurringScans.Emails))
			mailAdmins = convert.ToValue(resp.Properties.RecurringScans.EmailSubscriptionAdmins)
		}
		if resp.Properties.StorageContainerPath != nil {
			containerPath = *resp.Properties.StorageContainerPath
		}
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.database.vulnerabilityAssessment",
		map[string]*llx.RawData{
			"id":                     llx.StringDataPtr(resp.ID),
			"name":                   llx.StringDataPtr(resp.Name),
			"type":                   llx.StringDataPtr(resp.Type),
			"recurringScansEnabled":  llx.BoolData(recurringEnabled),
			"recurringScanEmails":    llx.ArrayData(emails, types.String),
			"mailSubscriptionAdmins": llx.BoolData(mailAdmins),
			"storageContainerPath":   llx.StringData(containerPath),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSqlServiceDatabaseVulnerabilityAssessment), nil
}

func (a *mqlAzureSubscriptionSqlServiceDatabaseVulnerabilityAssessment) scans() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	database, err := rid.Component("databases")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewDatabaseVulnerabilityAssessmentScansClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListByDatabasePager(rid.ResourceGroup, server, database, sql.VulnerabilityAssessmentNameDefault, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, scan := range page.Value {
			args, err := vulnerabilityAssessmentScanArgs(scan)
			if err != nil {
				return nil, err
			}
			r, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.vulnerabilityAssessmentScan", args)
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}
	}
	return res, nil
}

func vulnerabilityAssessmentScanArgs(scan *sql.VulnerabilityAssessmentScanRecord) (map[string]*llx.RawData, error) {
	var (
		scanID, triggerType, state, containerPath string
		startTime, endTime                        *time.Time
		failedChecks                              int64
		errorList                                 []any
	)
	if scan.Properties != nil {
		if scan.Properties.ScanID != nil {
			scanID = *scan.Properties.ScanID
		}
		if scan.Properties.TriggerType != nil {
			triggerType = string(*scan.Properties.TriggerType)
		}
		if scan.Properties.State != nil {
			state = string(*scan.Properties.State)
		}
		if scan.Properties.StorageContainerPath != nil {
			containerPath = *scan.Properties.StorageContainerPath
		}
		startTime = scan.Properties.StartTime
		endTime = scan.Properties.EndTime
		if scan.Properties.NumberOfFailedSecurityChecks != nil {
			failedChecks = int64(*scan.Properties.NumberOfFailedSecurityChecks)
		}
		if errs, derr := convert.JsonToDictSlice(scan.Properties.Errors); derr == nil {
			errorList = errs
		}
	}
	return map[string]*llx.RawData{
		"id":                           llx.StringDataPtr(scan.ID),
		"name":                         llx.StringDataPtr(scan.Name),
		"scanId":                       llx.StringData(scanID),
		"triggerType":                  llx.StringData(triggerType),
		"state":                        llx.StringData(state),
		"startTime":                    llx.TimeDataPtr(startTime),
		"endTime":                      llx.TimeDataPtr(endTime),
		"storageContainerPath":         llx.StringData(containerPath),
		"numberOfFailedSecurityChecks": llx.IntData(failedChecks),
		"errors":                       llx.ArrayData(errorList, types.Dict),
	}, nil
}

// ---------------------------------------------------------------------------
// Database-level: Dynamic Data Masking policy + rules
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceDatabase) dataMaskingPolicy() (*mqlAzureSubscriptionSqlServiceDatabaseDataMaskingPolicy, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	database, err := rid.Component("databases")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewDataMaskingPoliciesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(ctx, rid.ResourceGroup, server, database, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusInternalServerError {
			// Azure returns 500 from DataMaskingPolicies/Default on databases where DDM does not apply
			// (e.g. the master system database, certain SKUs). Treat as not-configured.
			log.Debug().Str("database", database).Msg("data masking policy unavailable for this database")
			a.DataMaskingPolicy.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	var (
		location, kind, dataMaskingState, exemptPrincipals, maskingLevel string
	)
	if resp.Location != nil {
		location = *resp.Location
	}
	if resp.Kind != nil {
		kind = *resp.Kind
	}
	if resp.Properties != nil {
		if resp.Properties.DataMaskingState != nil {
			dataMaskingState = string(*resp.Properties.DataMaskingState)
		}
		if resp.Properties.ExemptPrincipals != nil {
			exemptPrincipals = *resp.Properties.ExemptPrincipals
		}
		if resp.Properties.MaskingLevel != nil {
			maskingLevel = *resp.Properties.MaskingLevel
		}
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.database.dataMaskingPolicy",
		map[string]*llx.RawData{
			"id":               llx.StringDataPtr(resp.ID),
			"name":             llx.StringDataPtr(resp.Name),
			"type":             llx.StringDataPtr(resp.Type),
			"location":         llx.StringData(location),
			"kind":             llx.StringData(kind),
			"dataMaskingState": llx.StringData(dataMaskingState),
			"exemptPrincipals": llx.StringData(exemptPrincipals),
			"maskingLevel":     llx.StringData(maskingLevel),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSqlServiceDatabaseDataMaskingPolicy), nil
}

func (a *mqlAzureSubscriptionSqlServiceDatabaseDataMaskingPolicy) rules() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	database, err := rid.Component("databases")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewDataMaskingRulesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListByDatabasePager(rid.ResourceGroup, server, database, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, rule := range page.Value {
			var (
				ruleID, aliasName, ruleState, schemaName, tableName, columnName string
				maskingFunction                                                 string
				numberFrom, numberTo, prefixSize, suffixSize, replacementString string
			)
			if rule.Properties != nil {
				if rule.Properties.ID != nil {
					ruleID = *rule.Properties.ID
				}
				if rule.Properties.AliasName != nil {
					aliasName = *rule.Properties.AliasName
				}
				if rule.Properties.RuleState != nil {
					ruleState = string(*rule.Properties.RuleState)
				}
				if rule.Properties.SchemaName != nil {
					schemaName = *rule.Properties.SchemaName
				}
				if rule.Properties.TableName != nil {
					tableName = *rule.Properties.TableName
				}
				if rule.Properties.ColumnName != nil {
					columnName = *rule.Properties.ColumnName
				}
				if rule.Properties.MaskingFunction != nil {
					maskingFunction = string(*rule.Properties.MaskingFunction)
				}
				if rule.Properties.NumberFrom != nil {
					numberFrom = *rule.Properties.NumberFrom
				}
				if rule.Properties.NumberTo != nil {
					numberTo = *rule.Properties.NumberTo
				}
				if rule.Properties.PrefixSize != nil {
					prefixSize = *rule.Properties.PrefixSize
				}
				if rule.Properties.SuffixSize != nil {
					suffixSize = *rule.Properties.SuffixSize
				}
				if rule.Properties.ReplacementString != nil {
					replacementString = *rule.Properties.ReplacementString
				}
			}

			r, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.database.dataMaskingRule",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(rule.ID),
					"name":              llx.StringDataPtr(rule.Name),
					"type":              llx.StringDataPtr(rule.Type),
					"ruleId":            llx.StringData(ruleID),
					"aliasName":         llx.StringData(aliasName),
					"ruleState":         llx.StringData(ruleState),
					"schemaName":        llx.StringData(schemaName),
					"tableName":         llx.StringData(tableName),
					"columnName":        llx.StringData(columnName),
					"maskingFunction":   llx.StringData(maskingFunction),
					"numberFrom":        llx.StringData(numberFrom),
					"numberTo":          llx.StringData(numberTo),
					"prefixSize":        llx.StringData(prefixSize),
					"suffixSize":        llx.StringData(suffixSize),
					"replacementString": llx.StringData(replacementString),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}
	}
	return res, nil
}

// ---------------------------------------------------------------------------
// Database-level: ledger digest upload
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceDatabase) ledgerDigestUpload() (*mqlAzureSubscriptionSqlServiceDatabaseLedgerDigestUpload, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	database, err := rid.Component("databases")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewLedgerDigestUploadsClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(ctx, rid.ResourceGroup, server, database, sql.LedgerDigestUploadsNameCurrent, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			// Azure returns 404 from LedgerDigestUploads/Current on databases without ledger configured (the common case).
			log.Debug().Str("database", database).Msg("ledger digest upload not configured for this database")
			a.LedgerDigestUpload.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	var (
		state, digestEndpoint string
	)
	if resp.Properties != nil {
		if resp.Properties.State != nil {
			state = string(*resp.Properties.State)
		}
		if resp.Properties.DigestStorageEndpoint != nil {
			digestEndpoint = *resp.Properties.DigestStorageEndpoint
		}
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.database.ledgerDigestUpload",
		map[string]*llx.RawData{
			"id":                    llx.StringDataPtr(resp.ID),
			"name":                  llx.StringDataPtr(resp.Name),
			"state":                 llx.StringData(state),
			"digestStorageEndpoint": llx.StringData(digestEndpoint),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSqlServiceDatabaseLedgerDigestUpload), nil
}

// ---------------------------------------------------------------------------
// Database-level: geo backup policy
// ---------------------------------------------------------------------------

func (a *mqlAzureSubscriptionSqlServiceDatabase) geoBackupPolicy() (*mqlAzureSubscriptionSqlServiceDatabaseGeoBackupPolicy, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	server, err := rid.Component("servers")
	if err != nil {
		return nil, err
	}
	database, err := rid.Component("databases")
	if err != nil {
		return nil, err
	}
	client, err := sql.NewGeoBackupPoliciesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(ctx, rid.ResourceGroup, server, database, sql.GeoBackupPolicyNameDefault, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			// Azure returns 404 from GeoBackupPolicies/Default on databases where geo-backup does not apply
			// (e.g. Hyperscale, Serverless, certain SKUs).
			log.Debug().Str("database", database).Msg("geo backup policy unavailable for this database")
			a.GeoBackupPolicy.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}

	var (
		state, storageType string
	)
	if resp.Properties != nil {
		if resp.Properties.State != nil {
			state = string(*resp.Properties.State)
		}
		if resp.Properties.StorageType != nil {
			storageType = *resp.Properties.StorageType
		}
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.sqlService.database.geoBackupPolicy",
		map[string]*llx.RawData{
			"id":          llx.StringDataPtr(resp.ID),
			"name":        llx.StringDataPtr(resp.Name),
			"state":       llx.StringData(state),
			"storageType": llx.StringData(storageType),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionSqlServiceDatabaseGeoBackupPolicy), nil
}
