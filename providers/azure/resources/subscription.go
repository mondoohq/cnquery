// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	subscriptions "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func initAzureSubscription(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	azure, err := CreateResource(runtime, "azure", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	az := azure.(*mqlAzure)
	if az.sub != nil {
		return nil, az.sub, nil
	}

	subscriptionsC, err := subscriptions.NewClient(conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, nil, err
	}
	ctx := context.Background()
	resp, err := subscriptionsC.Get(ctx, conn.SubId(), &subscriptions.ClientGetOptions{})
	if err != nil {
		return nil, nil, err
	}

	managedByTenants := []any{}
	for _, t := range resp.ManagedByTenants {
		if t != nil {
			managedByTenants = append(managedByTenants, *t.TenantID)
		}
	}
	subPolicies, err := convert.JsonToDict(resp.SubscriptionPolicies)
	if err != nil {
		return nil, nil, err
	}
	args["id"] = llx.StringDataPtr(resp.ID)
	args["name"] = llx.StringDataPtr(resp.DisplayName)
	args["tenantId"] = llx.StringDataPtr(resp.TenantID)
	args["tags"] = llx.MapData(convert.PtrMapStrToInterface(resp.Tags), types.String)
	args["state"] = llx.StringDataPtr((*string)(resp.State))
	args["subscriptionId"] = llx.StringDataPtr(resp.SubscriptionID)
	args["authorizationSource"] = llx.StringDataPtr((*string)(resp.AuthorizationSource))
	args["managedByTenants"] = llx.ArrayData(managedByTenants, types.String)
	args["subscriptionsPolicies"] = llx.DictData(subPolicies)
	sub, err := CreateResource(runtime, "azure.subscription", args)
	if err != nil {
		return nil, nil, err
	}
	az.sub = sub.(*mqlAzureSubscription)
	return nil, az.sub, nil
}

func (a *mqlAzureSubscription) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscription) compute() (*mqlAzureSubscriptionComputeService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionComputeService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	computeSvc := svc.(*mqlAzureSubscriptionComputeService)
	return computeSvc, nil
}

func (a *mqlAzureSubscription) batch() (*mqlAzureSubscriptionBatchService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionBatchService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	batchSvc := svc.(*mqlAzureSubscriptionBatchService)
	return batchSvc, nil
}

func (a *mqlAzureSubscription) databricks() (*mqlAzureSubscriptionDatabricksService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionDatabricksService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	databricksSvc := svc.(*mqlAzureSubscriptionDatabricksService)
	return databricksSvc, nil
}

func (a *mqlAzureSubscription) network() (*mqlAzureSubscriptionNetworkService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionNetworkService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	networkSvc := svc.(*mqlAzureSubscriptionNetworkService)
	return networkSvc, nil
}

func (a *mqlAzureSubscription) storage() (*mqlAzureSubscriptionStorageService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionStorageService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	storageSvc := svc.(*mqlAzureSubscriptionStorageService)
	return storageSvc, nil
}

func (a *mqlAzureSubscription) web() (*mqlAzureSubscriptionWebService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionWebService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	webSvc := svc.(*mqlAzureSubscriptionWebService)
	return webSvc, nil
}

func (a *mqlAzureSubscription) sql() (*mqlAzureSubscriptionSqlService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionSqlService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	sqlSvc := svc.(*mqlAzureSubscriptionSqlService)
	return sqlSvc, nil
}

func (a *mqlAzureSubscription) mySql() (*mqlAzureSubscriptionMySqlService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionMySqlService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	mySqlSvc := svc.(*mqlAzureSubscriptionMySqlService)
	return mySqlSvc, nil
}

func (a *mqlAzureSubscription) postgreSql() (*mqlAzureSubscriptionPostgreSqlService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionPostgreSqlService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	postgreSqlSvc := svc.(*mqlAzureSubscriptionPostgreSqlService)
	return postgreSqlSvc, nil
}

func (a *mqlAzureSubscription) mariaDb() (*mqlAzureSubscriptionMariaDbService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionMariaDbService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	mariadbSvc := svc.(*mqlAzureSubscriptionMariaDbService)
	return mariadbSvc, nil
}

func (a *mqlAzureSubscription) cosmosDb() (*mqlAzureSubscriptionCosmosDbService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionCosmosDbService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	cosmosDbSvc := svc.(*mqlAzureSubscriptionCosmosDbService)
	return cosmosDbSvc, nil
}

func (a *mqlAzureSubscription) keyVault() (*mqlAzureSubscriptionKeyVaultService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionKeyVaultService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	kvSvc := svc.(*mqlAzureSubscriptionKeyVaultService)
	return kvSvc, nil
}

func (a *mqlAzureSubscription) cloudDefender() (*mqlAzureSubscriptionCloudDefenderService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionCloudDefenderService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	cdSvc := svc.(*mqlAzureSubscriptionCloudDefenderService)
	return cdSvc, nil
}

func (a *mqlAzureSubscription) aks() (*mqlAzureSubscriptionAksService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionAksService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	aksSvc := svc.(*mqlAzureSubscriptionAksService)
	return aksSvc, nil
}

func (a *mqlAzureSubscription) monitor() (*mqlAzureSubscriptionMonitorService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionMonitorService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	monitorSvc := svc.(*mqlAzureSubscriptionMonitorService)
	return monitorSvc, nil
}

func (a *mqlAzureSubscription) advisor() (*mqlAzureSubscriptionAdvisorService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionAdvisorService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	advisorSvc := svc.(*mqlAzureSubscriptionAdvisorService)
	return advisorSvc, nil
}

func (a *mqlAzureSubscription) iot() (*mqlAzureSubscriptionIotService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionIotService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	iotSvc := svc.(*mqlAzureSubscriptionIotService)
	return iotSvc, nil
}

func (a *mqlAzureSubscription) cache() (*mqlAzureSubscriptionCacheService, error) {
	svc, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionCacheService, map[string]*llx.RawData{
		"subscriptionId": llx.StringData(a.SubscriptionId.Data),
	})
	if err != nil {
		return nil, err
	}
	cacheSvc := svc.(*mqlAzureSubscriptionCacheService)
	return cacheSvc, nil
}
