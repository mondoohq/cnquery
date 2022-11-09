package azure

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	sql "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAzurermSql) id() (string, error) {
	return "azurerm.sql", nil
}

func (a *mqlAzurermSqlConfiguration) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermSqlFirewallrule) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermSqlServerAdministrator) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermSql) GetServers() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	dbClient, err := sql.NewServersClient(at.SubscriptionID(), token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	pager := dbClient.NewListPager(&sql.ServersClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, dbServer := range page.Value {
			properties, err := core.JsonToDict(dbServer.Properties)
			if err != nil {
				return nil, err
			}

			mqlAzureDbServer, err := a.MotorRuntime.CreateResource("azurerm.sql.server",
				"id", core.ToString(dbServer.ID),
				"name", core.ToString(dbServer.Name),
				"location", core.ToString(dbServer.Location),
				"tags", azureTagsToInterface(dbServer.Tags),
				"type", core.ToString(dbServer.Type),
				"properties", properties,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureDbServer)
		}
	}

	return res, nil
}

func (a *mqlAzurermSqlServer) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermSqlServer) GetDatabases() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	dbDatabaseClient, err := sql.NewDatabasesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	pager := dbDatabaseClient.NewListByServerPager(resourceID.ResourceGroup, server, &sql.DatabasesClientListByServerOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureDatabase, err := a.MotorRuntime.CreateResource("azurerm.sql.database",
				"id", core.ToString(entry.ID),
				"name", core.ToString(entry.Name),
				"type", core.ToString(entry.Type),
				"collation", core.ToString(entry.Properties.Collation),
				"creationDate", entry.Properties.CreationDate,
				"databaseId", core.ToString(entry.Properties.DatabaseID),
				"earliestRestoreDate", entry.Properties.EarliestRestoreDate,
				"createMode", core.ToString((*string)(entry.Properties.CreateMode)),
				"sourceDatabaseId", core.ToString(entry.Properties.SourceDatabaseID),
				"sourceDatabaseDeletionDate", entry.Properties.SourceDatabaseDeletionDate,
				"restorePointInTime", entry.Properties.RestorePointInTime,
				"recoveryServicesRecoveryPointResourceId", core.ToString(entry.Properties.RecoveryServicesRecoveryPointID),
				"edition", core.ToString(entry.SKU.Tier),
				"maxSizeBytes", core.ToInt64(entry.Properties.MaxSizeBytes),
				"requestedServiceObjectiveName", core.ToString(entry.Properties.RequestedServiceObjectiveName),
				"serviceLevelObjective", core.ToString(entry.Properties.CurrentServiceObjectiveName),
				"status", core.ToString((*string)(entry.Properties.Status)),
				"elasticPoolName", core.ToString(entry.Properties.ElasticPoolID),
				"defaultSecondaryLocation", core.ToString(entry.Properties.DefaultSecondaryLocation),
				"failoverGroupId", core.ToString(entry.Properties.FailoverGroupID),
				"readScale", core.ToString((*string)(entry.Properties.ReadScale)),
				"sampleName", core.ToString((*string)(entry.Properties.SampleName)),
				"zoneRedundant", core.ToBool(entry.Properties.ZoneRedundant),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureDatabase)
		}
	}
	return res, nil
}

func (a *mqlAzurermSqlServer) GetFirewallRules() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	dbFirewallClient, err := sql.NewFirewallRulesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	pager := dbFirewallClient.NewListByServerPager(resourceID.ResourceGroup, server, &sql.FirewallRulesClientListByServerOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {

			mqlAzureConfiguration, err := a.MotorRuntime.CreateResource("azurerm.sql.firewallrule",
				"id", core.ToString(entry.ID),
				"name", core.ToString(entry.Name),
				"type", core.ToString(entry.Type),
				"startIpAddress", core.ToString(entry.Properties.StartIPAddress),
				"endIpAddress", core.ToString(entry.Properties.EndIPAddress),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureConfiguration)
		}
	}
	return res, nil
}

func (a *mqlAzurermSqlServer) GetAzureAdAdministrators() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	administratorClient, err := sql.NewServerAzureADAdministratorsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	pager := administratorClient.NewListByServerPager(resourceID.ResourceGroup, server, &sql.ServerAzureADAdministratorsClientListByServerOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureSqlAdministrator, err := a.MotorRuntime.CreateResource("azurerm.sql.server.administrator",
				"id", core.ToString(entry.ID),
				"name", core.ToString(entry.Name),
				"type", core.ToString(entry.Type),
				"administratorType", core.ToString((*string)(entry.Properties.AdministratorType)),
				"login", core.ToString(entry.Properties.Login),
				"sid", core.ToString(entry.Properties.Sid),
				"tenantId", core.ToString(entry.Properties.TenantID),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzureSqlAdministrator)
		}
	}

	return res, nil
}

func (a *mqlAzurermSqlServer) GetConnectionPolicy() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	connectionClient, err := sql.NewServerConnectionPoliciesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	policy, err := connectionClient.Get(ctx, resourceID.ResourceGroup, server, sql.ConnectionPolicyNameDefault, &sql.ServerConnectionPoliciesClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(policy.Properties)
}

func (a *mqlAzurermSqlServer) GetAuditingPolicy() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	auditClient, err := sql.NewServerBlobAuditingPoliciesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	policy, err := auditClient.Get(ctx, resourceID.ResourceGroup, server, &sql.ServerBlobAuditingPoliciesClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(policy.ServerBlobAuditingPolicy.Properties)
}

func (a *mqlAzurermSqlServer) GetSecurityAlertPolicy() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	auditClient, err := sql.NewServerSecurityAlertPoliciesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	policy, err := auditClient.Get(ctx, resourceID.ResourceGroup, server, sql.SecurityAlertPolicyNameDefault, &sql.ServerSecurityAlertPoliciesClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(policy.ServerSecurityAlertPolicy.Properties)
}

func (a *mqlAzurermSqlServer) GetEncryptionProtector() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := sql.NewEncryptionProtectorsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	policy, err := client.Get(ctx, resourceID.ResourceGroup, server, sql.EncryptionProtectorNameCurrent, &sql.EncryptionProtectorsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(policy.EncryptionProtector.Properties)
}

func (a *mqlAzurermSqlDatabase) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermSqlDatabase) GetUsage() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
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

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := sql.NewDatabaseUsagesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	pager := client.NewListByDatabasePager(resourceID.ResourceGroup, server, database, &sql.DatabaseUsagesClientListByDatabaseOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			mqlAzureSqlUsage, err := a.MotorRuntime.CreateResource("azurerm.sql.databaseusage",
				"id", id+"/metrics/"+core.ToString(entry.Name),
				"name", core.ToString(entry.Name),
				"resourceName", core.ToString(entry.Name),
				"displayName", core.ToString(entry.Properties.DisplayName),
				"currentValue", core.ToFloat64(entry.Properties.CurrentValue),
				"limit", core.ToFloat64(entry.Properties.Limit),
				"unit", core.ToString(entry.Properties.Unit),
			)
			if err != nil {
				log.Error().Err(err).Msg("could not create MQL resource")
				return nil, err
			}
			res = append(res, mqlAzureSqlUsage)
		}
	}

	return res, nil
}

func (a *mqlAzurermSqlDatabaseusage) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermSqlDatabaseusage) GetNextResetTime() (interface{}, error) {
	return nil, errors.New("deprecated, no longer supported")
}

func (a *mqlAzurermSqlDatabase) GetAdvisor() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	status, err := a.Status()
	if err != nil {
		return nil, err
	}

	// If the database is in a paused or resuming state, advisors are not available.
	if status == "Paused" || status == "Resuming" {
		return []interface{}{}, nil
	}

	resourceID, err := at.ParseResourceID(id)
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

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := sql.NewDatabaseAdvisorsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	// it's an OData API, supports $expand. We can get the recommendedActions for all advisors here.
	expandRecommendedActions := "recommendedActions"
	advisors, err := client.ListByDatabase(ctx, resourceID.ResourceGroup, server, database, &sql.DatabaseAdvisorsClientListByDatabaseOptions{Expand: &expandRecommendedActions})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for _, entry := range advisors.AdvisorArray {
		dict, err := core.JsonToDict(entry)
		if err != nil {
			return nil, err
		}

		res = append(res, dict)
	}

	return res, nil
}

func (a *mqlAzurermSqlDatabase) GetThreatDetectionPolicy() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
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

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := sql.NewDatabaseSecurityAlertPoliciesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	policy, err := client.Get(ctx, resourceID.ResourceGroup, server, database, sql.SecurityAlertPolicyNameDefault, &sql.DatabaseSecurityAlertPoliciesClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(policy.DatabaseSecurityAlertPolicy.Properties)
}

func (a *mqlAzurermSqlDatabase) GetConnectionPolicy() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	server, err := resourceID.Component("servers")
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	connectionClient, err := sql.NewServerConnectionPoliciesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}
	policy, err := connectionClient.Get(ctx, resourceID.ResourceGroup, server, sql.ConnectionPolicyNameDefault, &sql.ServerConnectionPoliciesClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(policy.ServerConnectionPolicy.Properties)
}

func (a *mqlAzurermSqlDatabase) GetAuditingPolicy() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
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

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	auditClient, err := sql.NewDatabaseBlobAuditingPoliciesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	policy, err := auditClient.Get(ctx, resourceID.ResourceGroup, server, database, &sql.DatabaseBlobAuditingPoliciesClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(policy.DatabaseBlobAuditingPolicy.Properties)
}

func (a *mqlAzurermSqlDatabase) GetTransparentDataEncryption() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource id
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	resourceID, err := at.ParseResourceID(id)
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

	ctx := context.Background()
	token, err := at.GetTokenCredential()
	if err != nil {
		return nil, err
	}

	client, err := sql.NewTransparentDataEncryptionsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{})
	if err != nil {
		return nil, err
	}

	policy, err := client.Get(ctx, resourceID.ResourceGroup, server, database, sql.TransparentDataEncryptionNameCurrent, &sql.TransparentDataEncryptionsClientGetOptions{})
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(policy.LogicalDatabaseTransparentDataEncryption.Properties)
}

func (a *mqlAzurermSqlDatabase) GetCurrentServiceObjectiveId() (interface{}, error) {
	return nil, errors.New("deprecated, use 'serviceLevelObjective'")
}

func (a *mqlAzurermSqlDatabase) GetContainmentState() (interface{}, error) {
	return nil, errors.New("deprecated, no longer supported")
}

func (a *mqlAzurermSqlDatabase) GetRequestedServiceObjectiveId() (interface{}, error) {
	return nil, errors.New("deprecated, use 'requestedServiceObjectiveName'")
}

func (a *mqlAzurermSqlDatabase) GetRecommendedIndex() (interface{}, error) {
	return nil, errors.New("deprecated, use 'advisor.recommendedActions'")
}

func (a *mqlAzurermSqlDatabase) GetServiceTierAdvisors() (interface{}, error) {
	return nil, errors.New("deprecated, no longer supported")
}
