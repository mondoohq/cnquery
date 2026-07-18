// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	cosmos "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v4"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

// cosmosAccountRef resolves the subscription/resource-group/account-name triple
// from the account's ARM id, shared by every data-plane RBAC accessor.
func (a *mqlAzureSubscriptionCosmosDbServiceAccount) cosmosAccountRef() (*ResourceID, string, error) {
	rid, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, "", err
	}
	accountName, err := rid.Component("databaseAccounts")
	if err != nil {
		return nil, "", err
	}
	return rid, accountName, nil
}

// cosmosRoleDefinitionArgs maps a Cosmos data-plane role definition to MQL args.
// The Cassandra, Gremlin, Table, and MongoMI role definitions all share the
// same RoleDefinitionType enum and Permission type, so a single builder covers
// every API surface.
func cosmosRoleDefinitionArgs(id, name, typ, roleName *string, roleType *cosmos.RoleDefinitionType, assignable []*string, perms []*cosmos.Permission) (map[string]*llx.RawData, error) {
	args := map[string]*llx.RawData{
		"__id":             llx.StringDataPtr(id),
		"id":               llx.StringDataPtr(id),
		"name":             llx.StringDataPtr(name),
		"type":             llx.StringDataPtr(typ),
		"roleName":         llx.StringData(""),
		"roleType":         llx.StringData(""),
		"assignableScopes": llx.ArrayData([]any{}, types.String),
		"permissions":      llx.ArrayData([]any{}, types.Dict),
	}
	if roleName != nil {
		args["roleName"] = llx.StringDataPtr(roleName)
	}
	if roleType != nil {
		args["roleType"] = llx.StringData(string(*roleType))
	}
	scopes := []any{}
	for _, s := range assignable {
		if s != nil {
			scopes = append(scopes, *s)
		}
	}
	args["assignableScopes"] = llx.ArrayData(scopes, types.String)

	permList := []any{}
	for _, p := range perms {
		if p == nil {
			continue
		}
		m, err := convert.JsonToDict(p)
		if err != nil {
			return nil, err
		}
		permList = append(permList, m)
	}
	args["permissions"] = llx.ArrayData(permList, types.Dict)
	return args, nil
}

// cosmosRoleAssignmentArgs maps a Cosmos data-plane role assignment to MQL args.
func cosmosRoleAssignmentArgs(id, name, typ, principalID, roleDefinitionID, scope, provisioningState *string) map[string]*llx.RawData {
	args := map[string]*llx.RawData{
		"__id":              llx.StringDataPtr(id),
		"id":                llx.StringDataPtr(id),
		"name":              llx.StringDataPtr(name),
		"type":              llx.StringDataPtr(typ),
		"principalId":       llx.StringData(""),
		"roleDefinitionId":  llx.StringData(""),
		"scope":             llx.StringData(""),
		"provisioningState": llx.StringData(""),
	}
	if principalID != nil {
		args["principalId"] = llx.StringDataPtr(principalID)
	}
	if roleDefinitionID != nil {
		args["roleDefinitionId"] = llx.StringDataPtr(roleDefinitionID)
	}
	if scope != nil {
		args["scope"] = llx.StringDataPtr(scope)
	}
	if provisioningState != nil {
		args["provisioningState"] = llx.StringDataPtr(provisioningState)
	}
	return args
}

// Cassandra data-plane RBAC

type mqlAzureSubscriptionCosmosDbServiceAccountCassandraRoleDefinitionInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccountCassandraRoleDefinition) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccount) cassandraRoleDefinitions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, accountName, err := a.cosmosAccountRef()
	if err != nil {
		return nil, err
	}
	client, err := cosmos.NewCassandraResourcesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListCassandraRoleDefinitionsPager(rid.ResourceGroup, accountName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, def := range page.Value {
			if def == nil {
				continue
			}
			var roleName *string
			var roleType *cosmos.RoleDefinitionType
			var scopes []*string
			var perms []*cosmos.Permission
			if def.Properties != nil {
				roleName = def.Properties.RoleName
				roleType = def.Properties.Type
				scopes = def.Properties.AssignableScopes
				perms = def.Properties.Permissions
			}
			args, err := cosmosRoleDefinitionArgs(def.ID, def.Name, def.Type, roleName, roleType, scopes, perms)
			if err != nil {
				return nil, err
			}
			mqlDef, err := CreateResource(a.MqlRuntime, "azure.subscription.cosmosDbService.account.cassandraRoleDefinition", args)
			if err != nil {
				return nil, err
			}
			sysData, err := convert.JsonToDict(def.SystemData)
			if err != nil {
				return nil, err
			}
			mqlDef.(*mqlAzureSubscriptionCosmosDbServiceAccountCassandraRoleDefinition).cacheSystemData = sysData
			res = append(res, mqlDef)
		}
	}
	return res, nil
}

type mqlAzureSubscriptionCosmosDbServiceAccountCassandraRoleAssignmentInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccountCassandraRoleAssignment) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccount) cassandraRoleAssignments() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, accountName, err := a.cosmosAccountRef()
	if err != nil {
		return nil, err
	}
	client, err := cosmos.NewCassandraResourcesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListCassandraRoleAssignmentsPager(rid.ResourceGroup, accountName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ra := range page.Value {
			if ra == nil {
				continue
			}
			var principalID, roleDefinitionID, scope, provisioningState *string
			if ra.Properties != nil {
				principalID = ra.Properties.PrincipalID
				roleDefinitionID = ra.Properties.RoleDefinitionID
				scope = ra.Properties.Scope
				provisioningState = ra.Properties.ProvisioningState
			}
			args := cosmosRoleAssignmentArgs(ra.ID, ra.Name, ra.Type, principalID, roleDefinitionID, scope, provisioningState)
			mqlRA, err := CreateResource(a.MqlRuntime, "azure.subscription.cosmosDbService.account.cassandraRoleAssignment", args)
			if err != nil {
				return nil, err
			}
			sysData, err := convert.JsonToDict(ra.SystemData)
			if err != nil {
				return nil, err
			}
			mqlRA.(*mqlAzureSubscriptionCosmosDbServiceAccountCassandraRoleAssignment).cacheSystemData = sysData
			res = append(res, mqlRA)
		}
	}
	return res, nil
}

// Gremlin data-plane RBAC

type mqlAzureSubscriptionCosmosDbServiceAccountGremlinRoleDefinitionInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccountGremlinRoleDefinition) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccount) gremlinRoleDefinitions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, accountName, err := a.cosmosAccountRef()
	if err != nil {
		return nil, err
	}
	client, err := cosmos.NewGremlinResourcesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListGremlinRoleDefinitionsPager(rid.ResourceGroup, accountName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, def := range page.Value {
			if def == nil {
				continue
			}
			var roleName *string
			var roleType *cosmos.RoleDefinitionType
			var scopes []*string
			var perms []*cosmos.Permission
			if def.Properties != nil {
				roleName = def.Properties.RoleName
				roleType = def.Properties.Type
				scopes = def.Properties.AssignableScopes
				perms = def.Properties.Permissions
			}
			args, err := cosmosRoleDefinitionArgs(def.ID, def.Name, def.Type, roleName, roleType, scopes, perms)
			if err != nil {
				return nil, err
			}
			mqlDef, err := CreateResource(a.MqlRuntime, "azure.subscription.cosmosDbService.account.gremlinRoleDefinition", args)
			if err != nil {
				return nil, err
			}
			sysData, err := convert.JsonToDict(def.SystemData)
			if err != nil {
				return nil, err
			}
			mqlDef.(*mqlAzureSubscriptionCosmosDbServiceAccountGremlinRoleDefinition).cacheSystemData = sysData
			res = append(res, mqlDef)
		}
	}
	return res, nil
}

type mqlAzureSubscriptionCosmosDbServiceAccountGremlinRoleAssignmentInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccountGremlinRoleAssignment) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccount) gremlinRoleAssignments() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, accountName, err := a.cosmosAccountRef()
	if err != nil {
		return nil, err
	}
	client, err := cosmos.NewGremlinResourcesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListGremlinRoleAssignmentsPager(rid.ResourceGroup, accountName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ra := range page.Value {
			if ra == nil {
				continue
			}
			var principalID, roleDefinitionID, scope, provisioningState *string
			if ra.Properties != nil {
				principalID = ra.Properties.PrincipalID
				roleDefinitionID = ra.Properties.RoleDefinitionID
				scope = ra.Properties.Scope
				provisioningState = ra.Properties.ProvisioningState
			}
			args := cosmosRoleAssignmentArgs(ra.ID, ra.Name, ra.Type, principalID, roleDefinitionID, scope, provisioningState)
			mqlRA, err := CreateResource(a.MqlRuntime, "azure.subscription.cosmosDbService.account.gremlinRoleAssignment", args)
			if err != nil {
				return nil, err
			}
			sysData, err := convert.JsonToDict(ra.SystemData)
			if err != nil {
				return nil, err
			}
			mqlRA.(*mqlAzureSubscriptionCosmosDbServiceAccountGremlinRoleAssignment).cacheSystemData = sysData
			res = append(res, mqlRA)
		}
	}
	return res, nil
}

// Table data-plane RBAC

type mqlAzureSubscriptionCosmosDbServiceAccountTableRoleDefinitionInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccountTableRoleDefinition) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccount) tableRoleDefinitions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, accountName, err := a.cosmosAccountRef()
	if err != nil {
		return nil, err
	}
	client, err := cosmos.NewTableResourcesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListTableRoleDefinitionsPager(rid.ResourceGroup, accountName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, def := range page.Value {
			if def == nil {
				continue
			}
			var roleName *string
			var roleType *cosmos.RoleDefinitionType
			var scopes []*string
			var perms []*cosmos.Permission
			if def.Properties != nil {
				roleName = def.Properties.RoleName
				roleType = def.Properties.Type
				scopes = def.Properties.AssignableScopes
				perms = def.Properties.Permissions
			}
			args, err := cosmosRoleDefinitionArgs(def.ID, def.Name, def.Type, roleName, roleType, scopes, perms)
			if err != nil {
				return nil, err
			}
			mqlDef, err := CreateResource(a.MqlRuntime, "azure.subscription.cosmosDbService.account.tableRoleDefinition", args)
			if err != nil {
				return nil, err
			}
			sysData, err := convert.JsonToDict(def.SystemData)
			if err != nil {
				return nil, err
			}
			mqlDef.(*mqlAzureSubscriptionCosmosDbServiceAccountTableRoleDefinition).cacheSystemData = sysData
			res = append(res, mqlDef)
		}
	}
	return res, nil
}

type mqlAzureSubscriptionCosmosDbServiceAccountTableRoleAssignmentInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccountTableRoleAssignment) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccount) tableRoleAssignments() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, accountName, err := a.cosmosAccountRef()
	if err != nil {
		return nil, err
	}
	client, err := cosmos.NewTableResourcesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListTableRoleAssignmentsPager(rid.ResourceGroup, accountName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ra := range page.Value {
			if ra == nil {
				continue
			}
			var principalID, roleDefinitionID, scope, provisioningState *string
			if ra.Properties != nil {
				principalID = ra.Properties.PrincipalID
				roleDefinitionID = ra.Properties.RoleDefinitionID
				scope = ra.Properties.Scope
				provisioningState = ra.Properties.ProvisioningState
			}
			args := cosmosRoleAssignmentArgs(ra.ID, ra.Name, ra.Type, principalID, roleDefinitionID, scope, provisioningState)
			mqlRA, err := CreateResource(a.MqlRuntime, "azure.subscription.cosmosDbService.account.tableRoleAssignment", args)
			if err != nil {
				return nil, err
			}
			sysData, err := convert.JsonToDict(ra.SystemData)
			if err != nil {
				return nil, err
			}
			mqlRA.(*mqlAzureSubscriptionCosmosDbServiceAccountTableRoleAssignment).cacheSystemData = sysData
			res = append(res, mqlRA)
		}
	}
	return res, nil
}

// MongoDB managed-instance data-plane RBAC

type mqlAzureSubscriptionCosmosDbServiceAccountMongoMIRoleDefinitionInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccountMongoMIRoleDefinition) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccount) mongoMIRoleDefinitions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, accountName, err := a.cosmosAccountRef()
	if err != nil {
		return nil, err
	}
	client, err := cosmos.NewMongoMIResourcesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListMongoMIRoleDefinitionsPager(rid.ResourceGroup, accountName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, def := range page.Value {
			if def == nil {
				continue
			}
			var roleName *string
			var roleType *cosmos.RoleDefinitionType
			var scopes []*string
			var perms []*cosmos.Permission
			if def.Properties != nil {
				roleName = def.Properties.RoleName
				roleType = def.Properties.Type
				scopes = def.Properties.AssignableScopes
				perms = def.Properties.Permissions
			}
			args, err := cosmosRoleDefinitionArgs(def.ID, def.Name, def.Type, roleName, roleType, scopes, perms)
			if err != nil {
				return nil, err
			}
			mqlDef, err := CreateResource(a.MqlRuntime, "azure.subscription.cosmosDbService.account.mongoMIRoleDefinition", args)
			if err != nil {
				return nil, err
			}
			sysData, err := convert.JsonToDict(def.SystemData)
			if err != nil {
				return nil, err
			}
			mqlDef.(*mqlAzureSubscriptionCosmosDbServiceAccountMongoMIRoleDefinition).cacheSystemData = sysData
			res = append(res, mqlDef)
		}
	}
	return res, nil
}

type mqlAzureSubscriptionCosmosDbServiceAccountMongoMIRoleAssignmentInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccountMongoMIRoleAssignment) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccount) mongoMIRoleAssignments() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	rid, accountName, err := a.cosmosAccountRef()
	if err != nil {
		return nil, err
	}
	client, err := cosmos.NewMongoMIResourcesClient(rid.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := client.NewListMongoMIRoleAssignmentsPager(rid.ResourceGroup, accountName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ra := range page.Value {
			if ra == nil {
				continue
			}
			var principalID, roleDefinitionID, scope, provisioningState *string
			if ra.Properties != nil {
				principalID = ra.Properties.PrincipalID
				roleDefinitionID = ra.Properties.RoleDefinitionID
				scope = ra.Properties.Scope
				provisioningState = ra.Properties.ProvisioningState
			}
			args := cosmosRoleAssignmentArgs(ra.ID, ra.Name, ra.Type, principalID, roleDefinitionID, scope, provisioningState)
			mqlRA, err := CreateResource(a.MqlRuntime, "azure.subscription.cosmosDbService.account.mongoMIRoleAssignment", args)
			if err != nil {
				return nil, err
			}
			sysData, err := convert.JsonToDict(ra.SystemData)
			if err != nil {
				return nil, err
			}
			mqlRA.(*mqlAzureSubscriptionCosmosDbServiceAccountMongoMIRoleAssignment).cacheSystemData = sysData
			res = append(res, mqlRA)
		}
	}
	return res, nil
}
