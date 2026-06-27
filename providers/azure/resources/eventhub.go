// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/eventhub/armeventhub"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlAzureSubscriptionEventHubServiceNamespaceInternal struct {
	networkRuleSetFetched bool
	networkRuleSetProps   *armeventhub.NetworkRuleSetProperties
	networkRuleSetLock    sync.Mutex
	cacheSystemData       any
}

type mqlAzureSubscriptionEventHubServiceNamespaceEventHubInternal struct {
	cacheSystemData any
}

type mqlAzureSubscriptionEventHubServiceNamespaceEventHubConsumerGroupInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionEventHubServiceNamespaceEventHubConsumerGroup) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

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

			var status, cmkKeySource string
			var isAutoInflateEnabled, kafkaEnabled, disableLocalAuth, zoneRedundant bool
			var requireInfraEnc *bool
			var maximumThroughputUnits int64
			var minimumTlsVersion, publicNetworkAccess string
			var cmkKeys []any
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
				if enc := ns.Properties.Encryption; enc != nil {
					if enc.KeySource != nil {
						cmkKeySource = string(*enc.KeySource)
					}
					requireInfraEnc = enc.RequireInfrastructureEncryption
					for _, kvp := range enc.KeyVaultProperties {
						if kvp == nil {
							continue
						}
						if d, err := convert.JsonToDict(kvp); err == nil {
							cmkKeys = append(cmkKeys, d)
						}
					}
				}
			}

			mqlNs, err := CreateResource(a.MqlRuntime, "azure.subscription.eventHubService.namespace", map[string]*llx.RawData{
				"id":                              llx.StringDataPtr(ns.ID),
				"name":                            llx.StringDataPtr(ns.Name),
				"location":                        llx.StringDataPtr(ns.Location),
				"tags":                            llx.MapData(convert.PtrMapStrToInterface(ns.Tags), types.String),
				"sku":                             llx.DictData(sku),
				"status":                          llx.StringData(status),
				"isAutoInflateEnabled":            llx.BoolData(isAutoInflateEnabled),
				"maximumThroughputUnits":          llx.IntData(maximumThroughputUnits),
				"kafkaEnabled":                    llx.BoolData(kafkaEnabled),
				"disableLocalAuth":                llx.BoolData(disableLocalAuth),
				"minimumTlsVersion":               llx.StringData(minimumTlsVersion),
				"publicNetworkAccess":             llx.StringData(publicNetworkAccess),
				"zoneRedundant":                   llx.BoolData(zoneRedundant),
				"cmkKeySource":                    llx.StringData(cmkKeySource),
				"requireInfrastructureEncryption": llx.BoolDataPtr(requireInfraEnc),
				"cmkKeys":                         llx.ArrayData(cmkKeys, types.Dict),
			})
			if err != nil {
				return nil, err
			}
			sysData, err := convert.JsonToDict(ns.SystemData)
			if err != nil {
				return nil, err
			}
			mqlNs.(*mqlAzureSubscriptionEventHubServiceNamespace).cacheSystemData = sysData
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
			sysData, err := convert.JsonToDict(eh.SystemData)
			if err != nil {
				return nil, err
			}
			mqlEh.(*mqlAzureSubscriptionEventHubServiceNamespaceEventHub).cacheSystemData = sysData
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
			sysData, err := convert.JsonToDict(cg.SystemData)
			if err != nil {
				return nil, err
			}
			mqlCg.(*mqlAzureSubscriptionEventHubServiceNamespaceEventHubConsumerGroup).cacheSystemData = sysData
			res = append(res, mqlCg)
		}
	}

	return res, nil
}

// networkRuleSet fetches the namespace-level network rule set.
func (a *mqlAzureSubscriptionEventHubServiceNamespace) networkRuleSet() (any, error) {
	props, err := a.fetchNetworkRuleSetProperties()
	if err != nil {
		return nil, err
	}
	if props == nil {
		return nil, nil
	}
	return convert.JsonToDict(props)
}

func (a *mqlAzureSubscriptionEventHubServiceNamespace) networkRules() (*mqlAzureSubscriptionEventHubServiceNamespaceNetworkRules, error) {
	props, err := a.fetchNetworkRuleSetProperties()
	if err != nil {
		return nil, err
	}
	if props == nil {
		a.NetworkRules.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	defaultAction := ""
	if props.DefaultAction != nil {
		defaultAction = string(*props.DefaultAction)
	}
	publicNetworkAccess := ""
	if props.PublicNetworkAccess != nil {
		publicNetworkAccess = string(*props.PublicNetworkAccess)
	}
	trustedServiceAccess := false
	if props.TrustedServiceAccessEnabled != nil {
		trustedServiceAccess = *props.TrustedServiceAccessEnabled
	}

	ipRules := []any{}
	for _, r := range props.IPRules {
		if r == nil {
			continue
		}
		entry := map[string]any{}
		if r.IPMask != nil {
			entry["ipMask"] = *r.IPMask
		}
		if r.Action != nil {
			entry["action"] = string(*r.Action)
		}
		ipRules = append(ipRules, entry)
	}

	vnetRules := []any{}
	for i, r := range props.VirtualNetworkRules {
		if r == nil {
			continue
		}
		ignore := false
		if r.IgnoreMissingVnetServiceEndpoint != nil {
			ignore = *r.IgnoreMissingVnetServiceEndpoint
		}
		subnetID := ""
		if r.Subnet != nil && r.Subnet.ID != nil {
			subnetID = *r.Subnet.ID
		}
		id := fmt.Sprintf("%s/networkRules/virtualNetworkRules/%d", a.Id.Data, i)
		mqlRule, err := CreateResource(a.MqlRuntime, "azure.subscription.eventHubService.namespace.networkRules.virtualNetworkRule",
			map[string]*llx.RawData{
				"__id":                             llx.StringData(id),
				"ignoreMissingVnetServiceEndpoint": llx.BoolData(ignore),
			})
		if err != nil {
			return nil, err
		}
		mqlRule.(*mqlAzureSubscriptionEventHubServiceNamespaceNetworkRulesVirtualNetworkRule).cacheSubnetID = subnetID
		vnetRules = append(vnetRules, mqlRule)
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.eventHubService.namespace.networkRules", map[string]*llx.RawData{
		"__id":                        llx.StringData(a.Id.Data + "/networkRules"),
		"defaultAction":               llx.StringData(defaultAction),
		"publicNetworkAccess":         llx.StringData(publicNetworkAccess),
		"trustedServiceAccessEnabled": llx.BoolData(trustedServiceAccess),
		"ipRules":                     llx.ArrayData(ipRules, types.Dict),
		"virtualNetworkRules":         llx.ArrayData(vnetRules, types.Resource("azure.subscription.eventHubService.namespace.networkRules.virtualNetworkRule")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionEventHubServiceNamespaceNetworkRules), nil
}

func (a *mqlAzureSubscriptionEventHubServiceNamespace) fetchNetworkRuleSetProperties() (*armeventhub.NetworkRuleSetProperties, error) {
	if a.networkRuleSetFetched {
		return a.networkRuleSetProps, nil
	}
	a.networkRuleSetLock.Lock()
	defer a.networkRuleSetLock.Unlock()
	if a.networkRuleSetFetched {
		return a.networkRuleSetProps, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	resourceID, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	namespace, err := resourceID.Component("namespaces")
	if err != nil {
		return nil, err
	}
	client, err := armeventhub.NewNamespacesClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	resp, err := client.GetNetworkRuleSet(ctx, resourceID.ResourceGroup, namespace, nil)
	if err != nil {
		return nil, err
	}
	a.networkRuleSetProps = resp.NetworkRuleSet.Properties
	a.networkRuleSetFetched = true
	return a.networkRuleSetProps, nil
}

type mqlAzureSubscriptionEventHubServiceNamespaceNetworkRulesVirtualNetworkRuleInternal struct {
	cacheSubnetID string
}

func (a *mqlAzureSubscriptionEventHubServiceNamespaceNetworkRulesVirtualNetworkRule) subnet() (*mqlAzureSubscriptionNetworkServiceSubnet, error) {
	if a.cacheSubnetID == "" {
		a.Subnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "azure.subscription.networkService.subnet",
		map[string]*llx.RawData{"id": llx.StringData(a.cacheSubnetID)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionNetworkServiceSubnet), nil
}
