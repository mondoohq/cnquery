package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/sql/mgmt/sql"
	preview_sql "github.com/Azure/azure-sdk-for-go/services/preview/sql/mgmt/2017-03-01-preview/sql"
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
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	dbClient := sql.NewServersClient(at.SubscriptionID())
	dbClient.Authorizer = authorizer

	servers, err := dbClient.List(ctx)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	if servers.Value == nil {
		return res, nil
	}

	dbServers := *servers.Value

	for i := range dbServers {
		dbServer := dbServers[i]

		properties, err := core.JsonToDict(dbServer.ServerProperties)
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

	// id is a azure resource od
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
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	dbDatabaseClient := sql.NewDatabasesClient(resourceID.SubscriptionID)
	dbDatabaseClient.Authorizer = authorizer

	databases, err := dbDatabaseClient.ListByServer(ctx, resourceID.ResourceGroup, server, "", "")
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	if databases.Value == nil {
		return res, nil
	}

	list := *databases.Value
	for i := range list {
		entry := list[i]

		recommendedIndex, err := core.JsonToDict(entry.RecommendedIndex)
		if err != nil {
			return nil, err
		}

		serviceTierAdvisors, err := core.JsonToDict(entry.ServiceTierAdvisors)
		if err != nil {
			return nil, err
		}

		mqlAzureDatabase, err := a.MotorRuntime.CreateResource("azurerm.sql.database",
			"id", core.ToString(entry.ID),
			"name", core.ToString(entry.Name),
			"type", core.ToString(entry.Type),
			"collation", core.ToString(entry.Collation),
			"creationDate", azureRmTime(entry.CreationDate),
			"containmentState", core.ToInt64(entry.ContainmentState),
			"currentServiceObjectiveId", uuidToString(entry.CurrentServiceObjectiveID),
			"databaseId", uuidToString(entry.DatabaseID),
			"earliestRestoreDate", azureRmTime(entry.EarliestRestoreDate),
			"createMode", string(entry.CreateMode),
			"sourceDatabaseId", core.ToString(entry.SourceDatabaseID),
			"sourceDatabaseDeletionDate", azureRmTime(entry.SourceDatabaseDeletionDate),
			"restorePointInTime", azureRmTime(entry.RestorePointInTime),
			"recoveryServicesRecoveryPointResourceId", core.ToString(entry.RecoveryServicesRecoveryPointResourceID),
			"edition", string(entry.Edition),
			"maxSizeBytes", core.ToString(entry.MaxSizeBytes),
			"requestedServiceObjectiveId", uuidToString(entry.RequestedServiceObjectiveID),
			"requestedServiceObjectiveName", string(entry.RequestedServiceObjectiveName),
			"serviceLevelObjective", string(entry.ServiceLevelObjective),
			"status", core.ToString(entry.Status),
			"elasticPoolName", core.ToString(entry.ElasticPoolName),
			"defaultSecondaryLocation", core.ToString(entry.DefaultSecondaryLocation),
			"serviceTierAdvisors", serviceTierAdvisors,
			"recommendedIndex", recommendedIndex,
			"failoverGroupId", core.ToString(entry.FailoverGroupID),
			"readScale", string(entry.ReadScale),
			"sampleName", string(entry.SampleName),
			"zoneRedundant", core.ToBool(entry.ZoneRedundant),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAzureDatabase)
	}

	return res, nil
}

func (a *mqlAzurermSqlServer) GetFirewallRules() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
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
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	dbFirewallClient := sql.NewFirewallRulesClient(resourceID.SubscriptionID)
	dbFirewallClient.Authorizer = authorizer

	firewallRules, err := dbFirewallClient.ListByServer(ctx, resourceID.ResourceGroup, server)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	if firewallRules.Value == nil {
		return res, nil
	}

	list := *firewallRules.Value
	for i := range list {
		entry := list[i]

		mqlAzureConfiguration, err := a.MotorRuntime.CreateResource("azurerm.sql.firewallrule",
			"id", core.ToString(entry.ID),
			"name", core.ToString(entry.Name),
			"type", core.ToString(entry.Type),
			"startIpAddress", core.ToString(entry.StartIPAddress),
			"endIpAddress", core.ToString(entry.EndIPAddress),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAzureConfiguration)
	}

	return res, nil
}

func (a *mqlAzurermSqlServer) GetAzureAdAdministrators() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
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
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	administratorClient := sql.NewServerAzureADAdministratorsClient(resourceID.SubscriptionID)
	administratorClient.Authorizer = authorizer

	administrators, err := administratorClient.ListByServer(ctx, resourceID.ResourceGroup, server)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	if administrators.Value == nil {
		return res, nil
	}

	list := *administrators.Value
	for i := range list {
		entry := list[i]

		mqlAzureSqlAdministrator, err := a.MotorRuntime.CreateResource("azurerm.sql.server.administrator",
			"id", core.ToString(entry.ID),
			"name", core.ToString(entry.Name),
			"type", core.ToString(entry.Type),
			"administratorType", core.ToString(entry.AdministratorType),
			"login", core.ToString(entry.Login),
			"sid", uuidToString(entry.Sid),
			"tenantId", uuidToString(entry.TenantID),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAzureSqlAdministrator)
	}

	return res, nil
}

func (a *mqlAzurermSqlServer) GetConnectionPolicy() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
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
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	connectionClient := sql.NewServerConnectionPoliciesClient(resourceID.SubscriptionID)
	connectionClient.Authorizer = authorizer

	policy, err := connectionClient.Get(ctx, resourceID.ResourceGroup, server)
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(policy)
}

func (a *mqlAzurermSqlServer) GetAuditingPolicy() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
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
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	auditClient := preview_sql.NewServerBlobAuditingPoliciesClient(resourceID.SubscriptionID)
	auditClient.Authorizer = authorizer

	policy, err := auditClient.Get(ctx, resourceID.ResourceGroup, server)
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(policy.ServerBlobAuditingPolicyProperties)
}

func (a *mqlAzurermSqlServer) GetSecurityAlertPolicy() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
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
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	auditClient := preview_sql.NewServerSecurityAlertPoliciesClient(resourceID.SubscriptionID)
	auditClient.Authorizer = authorizer

	policy, err := auditClient.Get(ctx, resourceID.ResourceGroup, server)
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(policy.SecurityAlertPolicyProperties)
}

func (a *mqlAzurermSqlServer) GetEncryptionProtector() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
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
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := preview_sql.NewEncryptionProtectorsClient(resourceID.SubscriptionID)
	client.Authorizer = authorizer

	policy, err := client.Get(ctx, resourceID.ResourceGroup, server)
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(policy.EncryptionProtectorProperties)
}

func (a *mqlAzurermSqlDatabase) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermSqlDatabase) GetUsage() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
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
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := sql.NewDatabaseUsagesClient(resourceID.SubscriptionID)
	client.Authorizer = authorizer

	usage, err := client.ListByDatabase(ctx, resourceID.ResourceGroup, server, database)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	if usage.Value == nil {
		return res, nil
	}

	list := *usage.Value

	for i := range list {
		entry := list[i]

		mqlAzureSqlUsage, err := a.MotorRuntime.CreateResource("azurerm.sql.databaseusage",
			"id", id+"/metrics/"+core.ToString(entry.Name),
			"name", core.ToString(entry.Name),
			"resourceName", core.ToString(entry.ResourceName),
			"displayName", core.ToString(entry.DisplayName),
			"currentValue", core.ToFloat64(entry.CurrentValue),
			"limit", core.ToFloat64(entry.Limit),
			"unit", core.ToString(entry.Unit),
			"nextResetTime", azureRmTime(entry.NextResetTime),
		)
		if err != nil {
			log.Error().Err(err).Msg("could not create MQL resource")
			return nil, err
		}
		res = append(res, mqlAzureSqlUsage)
	}

	return res, nil
}

func (a *mqlAzurermSqlDatabaseusage) id() (string, error) {
	return a.Id()
}

func (a *mqlAzurermSqlDatabase) GetAdvisor() ([]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
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
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := sql.NewDatabaseAdvisorsClient(resourceID.SubscriptionID)
	client.Authorizer = authorizer

	advisors, err := client.ListByDatabase(ctx, resourceID.ResourceGroup, server, database)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	if advisors.Value == nil {
		return res, nil
	}

	list := *advisors.Value

	for i := range list {
		entry := list[i]

		dict, err := core.JsonToDict(entry)
		if err != nil {
			return nil, err
		}

		res = append(res, dict)
	}

	return res, nil
}

func (a *mqlAzurermSqlDatabase) GetThreadDetectionPolicy() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
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
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := sql.NewDatabaseThreatDetectionPoliciesClient(resourceID.SubscriptionID)
	client.Authorizer = authorizer

	policy, err := client.Get(ctx, resourceID.ResourceGroup, server, database)
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(policy)
}

func (a *mqlAzurermSqlDatabase) GetConnectionPolicy() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
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
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	connectionClient := sql.NewDatabaseConnectionPoliciesClient(resourceID.SubscriptionID)
	connectionClient.Authorizer = authorizer

	policy, err := connectionClient.Get(ctx, resourceID.ResourceGroup, server, database)
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(policy)
}

func (a *mqlAzurermSqlDatabase) GetAuditingPolicy() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
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
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	auditClient := preview_sql.NewDatabaseBlobAuditingPoliciesClient(resourceID.SubscriptionID)
	auditClient.Authorizer = authorizer

	policy, err := auditClient.Get(ctx, resourceID.ResourceGroup, server, database)
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(policy.DatabaseBlobAuditingPolicyProperties)
}

func (a *mqlAzurermSqlDatabase) GetTransparentDataEncryption() (map[string]interface{}, error) {
	at, err := azuretransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	// id is a azure resource od
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
	authorizer, err := at.Authorizer()
	if err != nil {
		return nil, err
	}

	client := sql.NewTransparentDataEncryptionsClient(resourceID.SubscriptionID)
	client.Authorizer = authorizer

	policy, err := client.Get(ctx, resourceID.ResourceGroup, server, database)
	if err != nil {
		return nil, err
	}

	return core.JsonToDict(policy.TransparentDataEncryptionProperties)
}
