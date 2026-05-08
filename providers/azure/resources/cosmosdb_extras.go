// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	cosmos "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v3"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAzureSubscriptionCosmosDbServiceAccountSqlDatabase) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccountSqlDatabaseContainer) id() (string, error) {
	return a.Id.Data, nil
}

// cosmosAccountResourceGroup parses {subscriptionID, resourceGroup, accountName}
// from a Cosmos account ARM id in a single ParseResourceID call.
func cosmosAccountResourceGroup(accountId string) (string, string, string, error) {
	parsed, err := ParseResourceID(accountId)
	if err != nil {
		return "", "", "", err
	}
	name, err := parsed.Component("databaseAccounts")
	if err != nil {
		return "", "", "", err
	}
	return parsed.SubscriptionID, parsed.ResourceGroup, name, nil
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccount) sqlDatabases() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	subId, rg, accountName, err := cosmosAccountResourceGroup(a.Id.Data)
	if err != nil {
		return nil, err
	}
	dbClient, err := cosmos.NewSQLResourcesClient(subId, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := dbClient.NewListSQLDatabasesPager(rg, accountName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			if isCosmosForbiddenError(err) {
				return res, nil
			}
			return nil, err
		}
		for _, db := range page.Value {
			mqlDb, err := sqlDatabaseToMQL(ctx, a.MqlRuntime, dbClient, rg, accountName, db)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlDb)
		}
	}
	return res, nil
}

func sqlDatabaseToMQL(ctx context.Context, runtime *plugin.Runtime, dbClient *cosmos.SQLResourcesClient, rg, accountName string, db *cosmos.SQLDatabaseGetResults) (plugin.Resource, error) {
	var dbName, etag string
	if db.Properties != nil && db.Properties.Resource != nil {
		if db.Properties.Resource.ID != nil {
			dbName = *db.Properties.Resource.ID
		}
		if db.Properties.Resource.Etag != nil {
			etag = *db.Properties.Resource.Etag
		}
	}
	if dbName == "" && db.Name != nil {
		dbName = *db.Name
	}

	manualTP, autoscaleMax, autoscaleEnabled, shared := fetchSqlDatabaseThroughput(ctx, dbClient, rg, accountName, dbName)

	return CreateResource(runtime, "azure.subscription.cosmosDbService.account.sqlDatabase",
		map[string]*llx.RawData{
			"id":                     llx.StringDataPtr(db.ID),
			"name":                   llx.StringData(dbName),
			"type":                   llx.StringDataPtr(db.Type),
			"etag":                   llx.StringData(etag),
			"throughputShared":       llx.BoolData(shared),
			"manualThroughput":       llx.IntData(int64(manualTP)),
			"autoscaleMaxThroughput": llx.IntData(int64(autoscaleMax)),
			"autoscaleEnabled":       llx.BoolData(autoscaleEnabled),
		})
}

// fetchSqlDatabaseThroughput resolves the database-level throughput offer.
// Returns (manualTP, autoscaleMax, autoscaleEnabled, sharedAcrossContainers).
// Cosmos returns 404 when the offer doesn't exist at this scope (i.e., throughput
// is configured per-container instead) — that case is reported as `shared = true`.
func fetchSqlDatabaseThroughput(ctx context.Context, dbClient *cosmos.SQLResourcesClient, rg, accountName, dbName string) (int32, int32, bool, bool) {
	resp, err := dbClient.GetSQLDatabaseThroughput(ctx, rg, accountName, dbName, nil)
	if err != nil {
		if isCosmosNotFoundError(err) {
			return 0, 0, false, true
		}
		// Real failures (rate limits, network timeouts, 5xx) shouldn't read as
		// "no offer" — log so operators can correlate empty-throughput rows
		// with API errors. The default zero-value return is preserved so the
		// rest of the database row still renders.
		log.Warn().Err(err).Str("account", accountName).Str("database", dbName).
			Msg("failed to fetch Cosmos DB SQL database throughput")
		return 0, 0, false, false
	}
	return throughputFromResource(resp.Properties)
}

func fetchSqlContainerThroughput(ctx context.Context, dbClient *cosmos.SQLResourcesClient, rg, accountName, dbName, containerName string) (int32, int32, bool, bool) {
	resp, err := dbClient.GetSQLContainerThroughput(ctx, rg, accountName, dbName, containerName, nil)
	if err != nil {
		if isCosmosNotFoundError(err) {
			return 0, 0, false, true
		}
		log.Warn().Err(err).Str("account", accountName).Str("database", dbName).Str("container", containerName).
			Msg("failed to fetch Cosmos DB SQL container throughput")
		return 0, 0, false, false
	}
	return throughputFromResource(resp.Properties)
}

// throughputFromResource picks apart a Cosmos throughput offer into the four
// fields we surface: manual provisioned throughput, autoscale max throughput,
// autoscale-enabled flag, and a shared flag (always false when an offer exists
// at this scope; only the 404 path sets shared = true).
func throughputFromResource(props *cosmos.ThroughputSettingsGetProperties) (int32, int32, bool, bool) {
	if props == nil || props.Resource == nil {
		return 0, 0, false, false
	}
	r := props.Resource
	var manual, autoscale int32
	autoEnabled := false
	if r.Throughput != nil {
		manual = *r.Throughput
	}
	if r.AutoscaleSettings != nil && r.AutoscaleSettings.MaxThroughput != nil {
		autoscale = *r.AutoscaleSettings.MaxThroughput
		autoEnabled = true
	}
	return manual, autoscale, autoEnabled, false
}

func (a *mqlAzureSubscriptionCosmosDbServiceAccountSqlDatabase) containers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	parsed, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	accountName, err := parsed.Component("databaseAccounts")
	if err != nil {
		return nil, err
	}
	rg := parsed.ResourceGroup
	dbName := a.Name.Data

	dbClient, err := cosmos.NewSQLResourcesClient(parsed.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []any{}
	pager := dbClient.NewListSQLContainersPager(rg, accountName, dbName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			if isCosmosForbiddenError(err) {
				return res, nil
			}
			return nil, err
		}
		for _, c := range page.Value {
			mqlC, err := sqlContainerToMQL(ctx, a.MqlRuntime, dbClient, rg, accountName, dbName, c)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlC)
		}
	}
	return res, nil
}

func sqlContainerToMQL(ctx context.Context, runtime *plugin.Runtime, dbClient *cosmos.SQLResourcesClient, rg, accountName, dbName string, c *cosmos.SQLContainerGetResults) (plugin.Resource, error) {
	var name, etag, partitionKeyKind, indexingMode, conflictMode, conflictPath string
	var defaultTtl, analyticalTtl int64
	autoIndex := false
	partitionKeyPaths := []any{}
	uniqueKeys := []any{}

	if c.Properties != nil && c.Properties.Resource != nil {
		r := c.Properties.Resource
		if r.ID != nil {
			name = *r.ID
		}
		if r.Etag != nil {
			etag = *r.Etag
		}
		if r.DefaultTTL != nil {
			defaultTtl = int64(*r.DefaultTTL)
		}
		if r.AnalyticalStorageTTL != nil {
			analyticalTtl = *r.AnalyticalStorageTTL
		}
		if r.PartitionKey != nil {
			if r.PartitionKey.Kind != nil {
				partitionKeyKind = string(*r.PartitionKey.Kind)
			}
			for _, p := range r.PartitionKey.Paths {
				if p != nil {
					partitionKeyPaths = append(partitionKeyPaths, *p)
				}
			}
		}
		if r.IndexingPolicy != nil {
			if r.IndexingPolicy.IndexingMode != nil {
				indexingMode = string(*r.IndexingPolicy.IndexingMode)
			}
			if r.IndexingPolicy.Automatic != nil {
				autoIndex = *r.IndexingPolicy.Automatic
			}
		}
		if r.ConflictResolutionPolicy != nil {
			if r.ConflictResolutionPolicy.Mode != nil {
				conflictMode = string(*r.ConflictResolutionPolicy.Mode)
			}
			if r.ConflictResolutionPolicy.ConflictResolutionPath != nil {
				conflictPath = *r.ConflictResolutionPolicy.ConflictResolutionPath
			}
		}
		if r.UniqueKeyPolicy != nil {
			for _, uk := range r.UniqueKeyPolicy.UniqueKeys {
				if uk == nil {
					continue
				}
				paths := []any{}
				for _, p := range uk.Paths {
					if p != nil {
						paths = append(paths, *p)
					}
				}
				uniqueKeys = append(uniqueKeys, map[string]any{"paths": paths})
			}
		}
	}
	if name == "" && c.Name != nil {
		name = *c.Name
	}

	manualTP, autoscaleMax, autoscaleEnabled, shared := fetchSqlContainerThroughput(ctx, dbClient, rg, accountName, dbName, name)

	return CreateResource(runtime, "azure.subscription.cosmosDbService.account.sqlDatabase.container",
		map[string]*llx.RawData{
			"id":                     llx.StringDataPtr(c.ID),
			"name":                   llx.StringData(name),
			"type":                   llx.StringDataPtr(c.Type),
			"etag":                   llx.StringData(etag),
			"partitionKeyPaths":      llx.ArrayData(partitionKeyPaths, types.String),
			"partitionKeyKind":       llx.StringData(partitionKeyKind),
			"defaultTtl":             llx.IntData(defaultTtl),
			"analyticalStorageTtl":   llx.IntData(analyticalTtl),
			"indexingMode":           llx.StringData(indexingMode),
			"automaticIndexing":      llx.BoolData(autoIndex),
			"uniqueKeys":             llx.ArrayData(uniqueKeys, types.Dict),
			"conflictResolutionMode": llx.StringData(conflictMode),
			"conflictResolutionPath": llx.StringData(conflictPath),
			"throughputShared":       llx.BoolData(shared),
			"manualThroughput":       llx.IntData(int64(manualTP)),
			"autoscaleMaxThroughput": llx.IntData(int64(autoscaleMax)),
			"autoscaleEnabled":       llx.BoolData(autoscaleEnabled),
		})
}

// isCosmosNotFoundError reports whether err is a Cosmos throughput 404. Used
// to distinguish "no offer at this scope" (database/container has shared
// throughput) from real errors.
func isCosmosNotFoundError(err error) bool {
	var rerr *azcore.ResponseError
	if errors.As(err, &rerr) {
		return rerr.StatusCode == http.StatusNotFound
	}
	return false
}

// isCosmosForbiddenError mirrors the convention used elsewhere in this
// provider — 403 from a sub-list is treated as "no entries visible" rather
// than fatal so an over-scoped audit role still works on the rest of the
// schema.
func isCosmosForbiddenError(err error) bool {
	var rerr *azcore.ResponseError
	if errors.As(err, &rerr) {
		return rerr.StatusCode == http.StatusForbidden
	}
	return false
}
