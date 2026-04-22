// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/eventhub/armeventhub"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAzureSubscriptionEventHubService) id() (string, error) {
	return "azure.subscription.eventHub/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionEventHubService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionEventHubServiceNamespace) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionEventHubServiceNamespaceEventHub) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionEventHubServiceNamespaceEventHubConsumerGroup) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionEventHubService) namespaces() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := armeventhub.NewNamespacesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	var res []any

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ns := range page.Value {
			if ns == nil {
				continue
			}

			sku, err := convert.JsonToDict(ns.SKU)
			if err != nil {
				return nil, err
			}

			var status string
			var isAutoInflateEnabled, kafkaEnabled, disableLocalAuth, zoneRedundant bool
			var maximumThroughputUnits int64
			var minimumTlsVersion, publicNetworkAccess string
			if ns.Properties != nil {
				if ns.Properties.Status != nil {
					status = *ns.Properties.Status
				}
				if ns.Properties.IsAutoInflateEnabled != nil {
					isAutoInflateEnabled = *ns.Properties.IsAutoInflateEnabled
				}
				if ns.Properties.MaximumThroughputUnits != nil {
					maximumThroughputUnits = int64(*ns.Properties.MaximumThroughputUnits)
				}
				if ns.Properties.KafkaEnabled != nil {
					kafkaEnabled = *ns.Properties.KafkaEnabled
				}
				if ns.Properties.DisableLocalAuth != nil {
					disableLocalAuth = *ns.Properties.DisableLocalAuth
				}
				if ns.Properties.MinimumTLSVersion != nil {
					minimumTlsVersion = string(*ns.Properties.MinimumTLSVersion)
				}
				if ns.Properties.PublicNetworkAccess != nil {
					publicNetworkAccess = string(*ns.Properties.PublicNetworkAccess)
				}
				if ns.Properties.ZoneRedundant != nil {
					zoneRedundant = *ns.Properties.ZoneRedundant
				}
			}

			mqlNs, err := CreateResource(a.MqlRuntime, "azure.subscription.eventHubService.namespace", map[string]*llx.RawData{
				"id":                     llx.StringDataPtr(ns.ID),
				"name":                   llx.StringDataPtr(ns.Name),
				"location":               llx.StringDataPtr(ns.Location),
				"tags":                   llx.MapData(convert.PtrMapStrToInterface(ns.Tags), types.String),
				"sku":                    llx.DictData(sku),
				"status":                 llx.StringData(status),
				"isAutoInflateEnabled":   llx.BoolData(isAutoInflateEnabled),
				"maximumThroughputUnits": llx.IntData(maximumThroughputUnits),
				"kafkaEnabled":           llx.BoolData(kafkaEnabled),
				"disableLocalAuth":       llx.BoolData(disableLocalAuth),
				"minimumTlsVersion":      llx.StringData(minimumTlsVersion),
				"publicNetworkAccess":    llx.StringData(publicNetworkAccess),
				"zoneRedundant":          llx.BoolData(zoneRedundant),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlNs)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionEventHubServiceNamespace) eventHubs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	nsName := a.Name.Data

	client, err := armeventhub.NewEventHubsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListByNamespacePager(resourceID.ResourceGroup, nsName, nil)
	var res []any

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, eh := range page.Value {
			if eh == nil {
				continue
			}

			var partitionCount, messageRetentionInDays int64
			var status string
			var partitionIds []any
			if eh.Properties != nil {
				if eh.Properties.PartitionCount != nil {
					partitionCount = *eh.Properties.PartitionCount
				}
				if eh.Properties.MessageRetentionInDays != nil {
					messageRetentionInDays = *eh.Properties.MessageRetentionInDays
				}
				if eh.Properties.Status != nil {
					status = string(*eh.Properties.Status)
				}
				if eh.Properties.PartitionIDs != nil {
					for _, pid := range eh.Properties.PartitionIDs {
						if pid != nil {
							partitionIds = append(partitionIds, *pid)
						}
					}
				}
			}

			mqlEh, err := CreateResource(a.MqlRuntime, "azure.subscription.eventHubService.namespace.eventHub", map[string]*llx.RawData{
				"id":                     llx.StringDataPtr(eh.ID),
				"name":                   llx.StringDataPtr(eh.Name),
				"partitionCount":         llx.IntData(partitionCount),
				"messageRetentionInDays": llx.IntData(messageRetentionInDays),
				"status":                 llx.StringData(status),
				"partitionIds":           llx.ArrayData(partitionIds, types.String),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlEh)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionEventHubServiceNamespaceEventHub) consumerGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	nsName, err := resourceID.Component("namespaces")
	if err != nil {
		return nil, err
	}

	ehName := a.Name.Data

	client, err := armeventhub.NewConsumerGroupsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListByEventHubPager(resourceID.ResourceGroup, nsName, ehName, nil)
	var res []any

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, cg := range page.Value {
			if cg == nil {
				continue
			}

			var userMetadata string
			if cg.Properties != nil && cg.Properties.UserMetadata != nil {
				userMetadata = *cg.Properties.UserMetadata
			}

			mqlCg, err := CreateResource(a.MqlRuntime, "azure.subscription.eventHubService.namespace.eventHub.consumerGroup", map[string]*llx.RawData{
				"id":           llx.StringDataPtr(cg.ID),
				"name":         llx.StringDataPtr(cg.Name),
				"userMetadata": llx.StringData(userMetadata),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlCg)
		}
	}

	return res, nil
}
