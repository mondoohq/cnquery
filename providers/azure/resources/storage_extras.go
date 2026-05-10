// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	storage "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAzureSubscriptionStorageServiceAccountFileShare) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionStorageServiceAccountPrivateEndpointConnection) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionStorageServiceAccountObjectReplicationPolicy) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionStorageServiceAccountBlobInventoryPolicy) id() (string, error) {
	return a.Id.Data, nil
}

// storageAccountResourceGroup returns the {resourceGroup, accountName} pair
// parsed from the parent storage account ARM id, or an error if the id is
// missing or malformed.
func storageAccountResourceGroup(accountId string) (string, string, error) {
	parsed, err := ParseResourceID(accountId)
	if err != nil {
		return "", "", err
	}
	name, err := parsed.Component("storageAccounts")
	if err != nil {
		return "", "", err
	}
	return parsed.ResourceGroup, name, nil
}

func (a *mqlAzureSubscriptionStorageServiceAccount) fileShares() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	parsed, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	rg, accountName, err := storageAccountResourceGroup(a.Id.Data)
	if err != nil {
		return nil, err
	}
	client, err := storage.NewFileSharesClient(parsed.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListPager(rg, accountName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			if isFeatureNotSupportedForAccountError(err) {
				a.FileShares.State = plugin.StateIsNull | plugin.StateIsSet
				return nil, nil
			}
			return nil, err
		}
		for _, share := range page.Value {
			mqlShare, err := fileShareToMQL(a.MqlRuntime, share)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlShare)
		}
	}
	return res, nil
}

func fileShareToMQL(runtime *plugin.Runtime, share *storage.FileShareItem) (plugin.Resource, error) {
	var enabledProtocols, accessTier, accessTierStatus, rootSquash string
	var leaseState, leaseStatus, leaseDuration string
	var shareQuotaGiB, provisionedIops, provisionedBandwidthMibps int64
	var signedIdentifierCount int64
	var deleted bool
	var deletedTime, accessTierChangeTime, snapshotTime, lastModifiedTime *time.Time
	metadata := map[string]any{}

	if p := share.Properties; p != nil {
		if p.EnabledProtocols != nil {
			enabledProtocols = string(*p.EnabledProtocols)
		}
		if p.AccessTier != nil {
			accessTier = string(*p.AccessTier)
		}
		if p.AccessTierStatus != nil {
			accessTierStatus = *p.AccessTierStatus
		}
		if p.RootSquash != nil {
			rootSquash = string(*p.RootSquash)
		}
		if p.ShareQuota != nil {
			shareQuotaGiB = int64(*p.ShareQuota)
		}
		if p.ProvisionedIops != nil {
			provisionedIops = int64(*p.ProvisionedIops)
		}
		if p.ProvisionedBandwidthMibps != nil {
			provisionedBandwidthMibps = int64(*p.ProvisionedBandwidthMibps)
		}
		signedIdentifierCount = int64(len(p.SignedIdentifiers))
		if p.Deleted != nil {
			deleted = *p.Deleted
		}
		deletedTime = p.DeletedTime
		accessTierChangeTime = p.AccessTierChangeTime
		snapshotTime = p.SnapshotTime
		lastModifiedTime = p.LastModifiedTime
		if p.LeaseState != nil {
			leaseState = string(*p.LeaseState)
		}
		if p.LeaseStatus != nil {
			leaseStatus = string(*p.LeaseStatus)
		}
		if p.LeaseDuration != nil {
			leaseDuration = string(*p.LeaseDuration)
		}
		for k, v := range p.Metadata {
			if v != nil {
				metadata[k] = *v
			}
		}
	}

	return CreateResource(runtime, "azure.subscription.storageService.account.fileShare",
		map[string]*llx.RawData{
			"id":                        llx.StringDataPtr(share.ID),
			"name":                      llx.StringDataPtr(share.Name),
			"type":                      llx.StringDataPtr(share.Type),
			"enabledProtocols":          llx.StringData(enabledProtocols),
			"accessTier":                llx.StringData(accessTier),
			"accessTierStatus":          llx.StringData(accessTierStatus),
			"accessTierChangeTime":      llx.TimeDataPtr(accessTierChangeTime),
			"shareQuotaGiB":             llx.IntData(shareQuotaGiB),
			"provisionedIops":           llx.IntData(provisionedIops),
			"provisionedBandwidthMibps": llx.IntData(provisionedBandwidthMibps),
			"rootSquash":                llx.StringData(rootSquash),
			"signedIdentifierCount":     llx.IntData(signedIdentifierCount),
			"deleted":                   llx.BoolData(deleted),
			"deletedTime":               llx.TimeDataPtr(deletedTime),
			"snapshotTime":              llx.TimeDataPtr(snapshotTime),
			"lastModifiedTime":          llx.TimeDataPtr(lastModifiedTime),
			"leaseState":                llx.StringData(leaseState),
			"leaseStatus":               llx.StringData(leaseStatus),
			"leaseDuration":             llx.StringData(leaseDuration),
			"metadata":                  llx.MapData(metadata, types.String),
		})
}

func (a *mqlAzureSubscriptionStorageServiceAccount) privateEndpointConnections() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	parsed, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	rg, accountName, err := storageAccountResourceGroup(a.Id.Data)
	if err != nil {
		return nil, err
	}
	client, err := storage.NewPrivateEndpointConnectionsClient(parsed.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListPager(rg, accountName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			// Private endpoint connections are supported on virtually all storage
			// account kinds, so a FeatureNotSupportedForAccount here would be
			// surprising — surface it rather than swallowing.
			return nil, err
		}
		for _, c := range page.Value {
			var privateEndpointId, status, description, actionsRequired, provisioningState string
			if p := c.Properties; p != nil {
				if p.PrivateEndpoint != nil && p.PrivateEndpoint.ID != nil {
					privateEndpointId = *p.PrivateEndpoint.ID
				}
				if pls := p.PrivateLinkServiceConnectionState; pls != nil {
					if pls.Status != nil {
						status = string(*pls.Status)
					}
					if pls.Description != nil {
						description = *pls.Description
					}
					if pls.ActionRequired != nil {
						actionsRequired = *pls.ActionRequired
					}
				}
				if p.ProvisioningState != nil {
					provisioningState = string(*p.ProvisioningState)
				}
			}
			mqlPe, err := CreateResource(a.MqlRuntime, "azure.subscription.storageService.account.privateEndpointConnection",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(c.ID),
					"name":              llx.StringDataPtr(c.Name),
					"type":              llx.StringDataPtr(c.Type),
					"privateEndpointId": llx.StringData(privateEndpointId),
					"status":            llx.StringData(status),
					"description":       llx.StringData(description),
					"actionsRequired":   llx.StringData(actionsRequired),
					"provisioningState": llx.StringData(provisioningState),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlPe)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionStorageServiceAccount) objectReplicationPolicies() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	parsed, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	rg, accountName, err := storageAccountResourceGroup(a.Id.Data)
	if err != nil {
		return nil, err
	}
	client, err := storage.NewObjectReplicationPoliciesClient(parsed.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListPager(rg, accountName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			if isFeatureNotSupportedForAccountError(err) {
				a.ObjectReplicationPolicies.State = plugin.StateIsNull | plugin.StateIsSet
				return nil, nil
			}
			return nil, err
		}
		for _, pol := range page.Value {
			var policyId, sourceAccount, destinationAccount string
			var enabledTime *time.Time
			rules := []any{}
			if p := pol.Properties; p != nil {
				if p.PolicyID != nil {
					policyId = *p.PolicyID
				}
				if p.SourceAccount != nil {
					sourceAccount = *p.SourceAccount
				}
				if p.DestinationAccount != nil {
					destinationAccount = *p.DestinationAccount
				}
				enabledTime = p.EnabledTime
				rules = objectReplicationRulesToDicts(p.Rules)
			}
			mqlPol, err := CreateResource(a.MqlRuntime, "azure.subscription.storageService.account.objectReplicationPolicy",
				map[string]*llx.RawData{
					"id":                 llx.StringDataPtr(pol.ID),
					"name":               llx.StringDataPtr(pol.Name),
					"type":               llx.StringDataPtr(pol.Type),
					"policyId":           llx.StringData(policyId),
					"sourceAccount":      llx.StringData(sourceAccount),
					"destinationAccount": llx.StringData(destinationAccount),
					"enabledTime":        llx.TimeDataPtr(enabledTime),
					"rules":              llx.ArrayData(rules, types.Dict),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlPol)
		}
	}
	return res, nil
}

// objectReplicationRulesToDicts flattens object-replication rules into the
// stable dict shape advertised in the .lr comment so query output is
// deterministic across runs.
func objectReplicationRulesToDicts(rules []*storage.ObjectReplicationPolicyRule) []any {
	out := []any{}
	for _, r := range rules {
		if r == nil {
			continue
		}
		entry := map[string]any{
			"ruleId":               "",
			"sourceContainer":      "",
			"destinationContainer": "",
			"prefixMatch":          []any{},
			"minCreationTime":      "",
		}
		if r.RuleID != nil {
			entry["ruleId"] = *r.RuleID
		}
		if r.SourceContainer != nil {
			entry["sourceContainer"] = *r.SourceContainer
		}
		if r.DestinationContainer != nil {
			entry["destinationContainer"] = *r.DestinationContainer
		}
		if f := r.Filters; f != nil {
			prefixes := []any{}
			for _, p := range f.PrefixMatch {
				if p != nil {
					prefixes = append(prefixes, *p)
				}
			}
			entry["prefixMatch"] = prefixes
			if f.MinCreationTime != nil {
				entry["minCreationTime"] = *f.MinCreationTime
			}
		}
		out = append(out, entry)
	}
	return out
}

func (a *mqlAzureSubscriptionStorageServiceAccount) blobInventoryPolicy() (*mqlAzureSubscriptionStorageServiceAccountBlobInventoryPolicy, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	parsed, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	rg, accountName, err := storageAccountResourceGroup(a.Id.Data)
	if err != nil {
		return nil, err
	}
	client, err := storage.NewBlobInventoryPoliciesClient(parsed.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(ctx, rg, accountName, storage.BlobInventoryPolicyNameDefault, nil)
	if err != nil {
		// Storage accounts without a blob-inventory policy return 404; surface that as null
		// rather than as an error so .blobInventoryPolicy reads as "no policy" cleanly.
		if isInventoryPolicyNotFoundError(err) {
			a.BlobInventoryPolicy.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
		return nil, err
	}
	pol := resp.BlobInventoryPolicy

	var enabled bool
	var lastModified *time.Time
	rules := []any{}
	if p := pol.Properties; p != nil {
		lastModified = p.LastModifiedTime
		if p.Policy != nil {
			if p.Policy.Enabled != nil {
				enabled = *p.Policy.Enabled
			}
			rules = inventoryRulesToDicts(p.Policy.Rules)
		}
	}

	mqlPol, err := CreateResource(a.MqlRuntime, "azure.subscription.storageService.account.blobInventoryPolicy",
		map[string]*llx.RawData{
			"id":               llx.StringDataPtr(pol.ID),
			"name":             llx.StringDataPtr(pol.Name),
			"type":             llx.StringDataPtr(pol.Type),
			"enabled":          llx.BoolData(enabled),
			"lastModifiedTime": llx.TimeDataPtr(lastModified),
			"rules":            llx.ArrayData(rules, types.Dict),
		})
	if err != nil {
		return nil, err
	}
	return mqlPol.(*mqlAzureSubscriptionStorageServiceAccountBlobInventoryPolicy), nil
}

func inventoryRulesToDicts(rules []*storage.BlobInventoryPolicyRule) []any {
	out := []any{}
	for _, r := range rules {
		if r == nil {
			continue
		}
		entry := map[string]any{
			"name":        "",
			"enabled":     false,
			"destination": "",
			"definition":  map[string]any{},
		}
		if r.Name != nil {
			entry["name"] = *r.Name
		}
		if r.Enabled != nil {
			entry["enabled"] = *r.Enabled
		}
		if r.Destination != nil {
			entry["destination"] = *r.Destination
		}
		if r.Definition != nil {
			d, err := convert.JsonToDict(r.Definition)
			if err == nil {
				entry["definition"] = d
			}
		}
		out = append(out, entry)
	}
	return out
}

// isInventoryPolicyNotFoundError reports whether the error returned from
// the BlobInventoryPolicies Get call is the "no policy configured" 404 case
// versus a permission or transient error.
func isInventoryPolicyNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var rerr *azcore.ResponseError
	if errors.As(err, &rerr) {
		return rerr.StatusCode == 404
	}
	return false
}

func (a *mqlAzureSubscriptionStorageServiceAccountQueue) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionStorageServiceAccountTable) id() (string, error) {
	return a.Id.Data, nil
}

type mqlAzureSubscriptionStorageServiceAccountQueueInternal struct {
	cacheAccountId string
	countFetched   bool
	count          int64
	countLock      sync.Mutex
}

func (a *mqlAzureSubscriptionStorageServiceAccount) queues() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	parsed, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	rg, accountName, err := storageAccountResourceGroup(a.Id.Data)
	if err != nil {
		return nil, err
	}
	client, err := storage.NewQueueClient(parsed.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListPager(rg, accountName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			if isFeatureNotSupportedForAccountError(err) {
				a.Queues.State = plugin.StateIsNull | plugin.StateIsSet
				return nil, nil
			}
			return nil, err
		}
		for _, q := range page.Value {
			if q == nil {
				continue
			}
			metadata := map[string]any{}
			if q.QueueProperties != nil {
				for k, v := range q.QueueProperties.Metadata {
					if v != nil {
						metadata[k] = *v
					}
				}
			}
			mqlQueue, err := CreateResource(a.MqlRuntime, "azure.subscription.storageService.account.queue",
				map[string]*llx.RawData{
					"id":       llx.StringDataPtr(q.ID),
					"name":     llx.StringDataPtr(q.Name),
					"metadata": llx.MapData(metadata, types.String),
				})
			if err != nil {
				return nil, err
			}
			mqlQ := mqlQueue.(*mqlAzureSubscriptionStorageServiceAccountQueue)
			mqlQ.cacheAccountId = a.Id.Data
			res = append(res, mqlQ)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionStorageServiceAccountQueue) approximateMessageCount() (int64, error) {
	if a.countFetched {
		return a.count, nil
	}
	a.countLock.Lock()
	defer a.countLock.Unlock()
	if a.countFetched {
		return a.count, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	parsed, err := ParseResourceID(a.cacheAccountId)
	if err != nil {
		return 0, err
	}
	rg, accountName, err := storageAccountResourceGroup(a.cacheAccountId)
	if err != nil {
		return 0, err
	}
	client, err := storage.NewQueueClient(parsed.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return 0, err
	}
	resp, err := client.Get(ctx, rg, accountName, a.Name.Data, nil)
	if err != nil {
		return 0, err
	}
	if resp.QueueProperties != nil && resp.QueueProperties.ApproximateMessageCount != nil {
		a.count = int64(*resp.QueueProperties.ApproximateMessageCount)
	}
	a.countFetched = true
	return a.count, nil
}

func (a *mqlAzureSubscriptionStorageServiceAccount) tables() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	parsed, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	rg, accountName, err := storageAccountResourceGroup(a.Id.Data)
	if err != nil {
		return nil, err
	}
	client, err := storage.NewTableClient(parsed.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListPager(rg, accountName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			if isFeatureNotSupportedForAccountError(err) {
				a.Tables.State = plugin.StateIsNull | plugin.StateIsSet
				return nil, nil
			}
			return nil, err
		}
		for _, t := range page.Value {
			if t == nil {
				continue
			}
			signedIdentifiers := []any{}
			if t.TableProperties != nil {
				for _, si := range t.TableProperties.SignedIdentifiers {
					if si == nil {
						continue
					}
					entry := map[string]any{}
					if si.ID != nil {
						entry["id"] = *si.ID
					}
					if si.AccessPolicy != nil {
						if si.AccessPolicy.Permission != nil {
							entry["permission"] = *si.AccessPolicy.Permission
						}
						if si.AccessPolicy.StartTime != nil {
							entry["startTime"] = si.AccessPolicy.StartTime.Format(time.RFC3339)
						}
						if si.AccessPolicy.ExpiryTime != nil {
							entry["expiryTime"] = si.AccessPolicy.ExpiryTime.Format(time.RFC3339)
						}
					}
					signedIdentifiers = append(signedIdentifiers, entry)
				}
			}
			mqlTable, err := CreateResource(a.MqlRuntime, "azure.subscription.storageService.account.table",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(t.ID),
					"name":              llx.StringDataPtr(t.Name),
					"signedIdentifiers": llx.ArrayData(signedIdentifiers, types.Dict),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlTable)
		}
	}
	return res, nil
}
