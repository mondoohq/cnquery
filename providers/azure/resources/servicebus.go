// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/servicebus/armservicebus"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlAzureSubscriptionServiceBusServiceNamespaceInternal struct {
	networkRuleSetFetched bool
	networkRuleSetProps   *armservicebus.NetworkRuleSetProperties
	networkRuleSetLock    sync.Mutex
}

func (a *mqlAzureSubscriptionServiceBusService) id() (string, error) {
	return "azure.subscription.serviceBus/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionServiceBusService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionServiceBusServiceNamespace) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionServiceBusServiceNamespaceQueue) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionServiceBusServiceNamespaceTopic) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionServiceBusServiceNamespaceTopicSubscription) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionServiceBusService) namespaces() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := armservicebus.NewNamespacesClient(subId, token, &arm.ClientOptions{
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

			var status, serviceBusEndpoint, provisioningState, cmkKeySource string
			var disableLocalAuth, zoneRedundant bool
			var requireInfraEnc *bool
			var cmkKeys []any
			if ns.Properties != nil {
				if ns.Properties.Status != nil {
					status = *ns.Properties.Status
				}
				if ns.Properties.ServiceBusEndpoint != nil {
					serviceBusEndpoint = *ns.Properties.ServiceBusEndpoint
				}
				if ns.Properties.DisableLocalAuth != nil {
					disableLocalAuth = *ns.Properties.DisableLocalAuth
				}
				if ns.Properties.ZoneRedundant != nil {
					zoneRedundant = *ns.Properties.ZoneRedundant
				}
				if ns.Properties.ProvisioningState != nil {
					provisioningState = *ns.Properties.ProvisioningState
				}
				if enc := ns.Properties.Encryption; enc != nil {
					if enc.KeySource != nil {
						cmkKeySource = *enc.KeySource
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

			mqlNs, err := CreateResource(a.MqlRuntime, "azure.subscription.serviceBusService.namespace", map[string]*llx.RawData{
				"id":                              llx.StringDataPtr(ns.ID),
				"name":                            llx.StringDataPtr(ns.Name),
				"location":                        llx.StringDataPtr(ns.Location),
				"tags":                            llx.MapData(convert.PtrMapStrToInterface(ns.Tags), types.String),
				"sku":                             llx.DictData(sku),
				"status":                          llx.StringData(status),
				"serviceBusEndpoint":              llx.StringData(serviceBusEndpoint),
				"disableLocalAuth":                llx.BoolData(disableLocalAuth),
				"zoneRedundant":                   llx.BoolData(zoneRedundant),
				"provisioningState":               llx.StringData(provisioningState),
				"cmkKeySource":                    llx.StringData(cmkKeySource),
				"requireInfrastructureEncryption": llx.BoolDataPtr(requireInfraEnc),
				"cmkKeys":                         llx.ArrayData(cmkKeys, types.Dict),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlNs)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionServiceBusServiceNamespace) queues() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	nsName := a.Name.Data

	client, err := armservicebus.NewQueuesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
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
		for _, q := range page.Value {
			if q == nil {
				continue
			}

			var status string
			var maxSizeInMegabytes, maxDeliveryCount int64
			var messageCount, deadLetterMessageCount int64
			var lockDuration, defaultMessageTimeToLive string
			var requiresDuplicateDetection, requiresSession, enablePartitioning bool
			if q.Properties != nil {
				if q.Properties.Status != nil {
					status = string(*q.Properties.Status)
				}
				if q.Properties.MaxSizeInMegabytes != nil {
					maxSizeInMegabytes = int64(*q.Properties.MaxSizeInMegabytes)
				}
				if q.Properties.MessageCount != nil {
					messageCount = *q.Properties.MessageCount
				}
				if q.Properties.CountDetails != nil && q.Properties.CountDetails.DeadLetterMessageCount != nil {
					deadLetterMessageCount = *q.Properties.CountDetails.DeadLetterMessageCount
				}
				if q.Properties.MaxDeliveryCount != nil {
					maxDeliveryCount = int64(*q.Properties.MaxDeliveryCount)
				}
				if q.Properties.LockDuration != nil {
					lockDuration = *q.Properties.LockDuration
				}
				if q.Properties.DefaultMessageTimeToLive != nil {
					defaultMessageTimeToLive = *q.Properties.DefaultMessageTimeToLive
				}
				if q.Properties.RequiresDuplicateDetection != nil {
					requiresDuplicateDetection = *q.Properties.RequiresDuplicateDetection
				}
				if q.Properties.RequiresSession != nil {
					requiresSession = *q.Properties.RequiresSession
				}
				if q.Properties.EnablePartitioning != nil {
					enablePartitioning = *q.Properties.EnablePartitioning
				}
			}

			mqlQueue, err := CreateResource(a.MqlRuntime, "azure.subscription.serviceBusService.namespace.queue", map[string]*llx.RawData{
				"id":                         llx.StringDataPtr(q.ID),
				"name":                       llx.StringDataPtr(q.Name),
				"status":                     llx.StringData(status),
				"maxSizeInMegabytes":         llx.IntData(maxSizeInMegabytes),
				"messageCount":               llx.IntData(messageCount),
				"deadLetterMessageCount":     llx.IntData(deadLetterMessageCount),
				"maxDeliveryCount":           llx.IntData(maxDeliveryCount),
				"lockDuration":               llx.StringData(lockDuration),
				"defaultMessageTimeToLive":   llx.StringData(defaultMessageTimeToLive),
				"requiresDuplicateDetection": llx.BoolData(requiresDuplicateDetection),
				"requiresSession":            llx.BoolData(requiresSession),
				"enablePartitioning":         llx.BoolData(enablePartitioning),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlQueue)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionServiceBusServiceNamespace) topics() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}

	nsName := a.Name.Data

	client, err := armservicebus.NewTopicsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
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
		for _, t := range page.Value {
			if t == nil {
				continue
			}

			var status string
			var maxSizeInMegabytes int64
			var subscriptionCount int64
			var enablePartitioning, supportOrdering, requiresDuplicateDetection bool
			var defaultMessageTimeToLive string
			if t.Properties != nil {
				if t.Properties.Status != nil {
					status = string(*t.Properties.Status)
				}
				if t.Properties.MaxSizeInMegabytes != nil {
					maxSizeInMegabytes = int64(*t.Properties.MaxSizeInMegabytes)
				}
				if t.Properties.SubscriptionCount != nil {
					subscriptionCount = int64(*t.Properties.SubscriptionCount)
				}
				if t.Properties.EnablePartitioning != nil {
					enablePartitioning = *t.Properties.EnablePartitioning
				}
				if t.Properties.SupportOrdering != nil {
					supportOrdering = *t.Properties.SupportOrdering
				}
				if t.Properties.RequiresDuplicateDetection != nil {
					requiresDuplicateDetection = *t.Properties.RequiresDuplicateDetection
				}
				if t.Properties.DefaultMessageTimeToLive != nil {
					defaultMessageTimeToLive = *t.Properties.DefaultMessageTimeToLive
				}
			}

			mqlTopic, err := CreateResource(a.MqlRuntime, "azure.subscription.serviceBusService.namespace.topic", map[string]*llx.RawData{
				"id":                         llx.StringDataPtr(t.ID),
				"name":                       llx.StringDataPtr(t.Name),
				"status":                     llx.StringData(status),
				"maxSizeInMegabytes":         llx.IntData(maxSizeInMegabytes),
				"subscriptionCount":          llx.IntData(subscriptionCount),
				"enablePartitioning":         llx.BoolData(enablePartitioning),
				"supportOrdering":            llx.BoolData(supportOrdering),
				"requiresDuplicateDetection": llx.BoolData(requiresDuplicateDetection),
				"defaultMessageTimeToLive":   llx.StringData(defaultMessageTimeToLive),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlTopic)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionServiceBusServiceNamespaceTopic) subscriptions() ([]any, error) {
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

	topicName := a.Name.Data

	client, err := armservicebus.NewSubscriptionsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListByTopicPager(resourceID.ResourceGroup, nsName, topicName, nil)
	var res []any

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, sub := range page.Value {
			if sub == nil {
				continue
			}

			var status string
			var messageCount, deadLetterMessageCount int64
			var maxDeliveryCount int64
			var lockDuration, defaultMessageTimeToLive string
			var requiresSession bool
			if sub.Properties != nil {
				if sub.Properties.Status != nil {
					status = string(*sub.Properties.Status)
				}
				if sub.Properties.MessageCount != nil {
					messageCount = *sub.Properties.MessageCount
				}
				if sub.Properties.CountDetails != nil && sub.Properties.CountDetails.DeadLetterMessageCount != nil {
					deadLetterMessageCount = *sub.Properties.CountDetails.DeadLetterMessageCount
				}
				if sub.Properties.MaxDeliveryCount != nil {
					maxDeliveryCount = int64(*sub.Properties.MaxDeliveryCount)
				}
				if sub.Properties.LockDuration != nil {
					lockDuration = *sub.Properties.LockDuration
				}
				if sub.Properties.DefaultMessageTimeToLive != nil {
					defaultMessageTimeToLive = *sub.Properties.DefaultMessageTimeToLive
				}
				if sub.Properties.RequiresSession != nil {
					requiresSession = *sub.Properties.RequiresSession
				}
			}

			mqlSub, err := CreateResource(a.MqlRuntime, "azure.subscription.serviceBusService.namespace.topic.subscription", map[string]*llx.RawData{
				"id":                       llx.StringDataPtr(sub.ID),
				"name":                     llx.StringDataPtr(sub.Name),
				"status":                   llx.StringData(status),
				"messageCount":             llx.IntData(messageCount),
				"deadLetterMessageCount":   llx.IntData(deadLetterMessageCount),
				"maxDeliveryCount":         llx.IntData(maxDeliveryCount),
				"lockDuration":             llx.StringData(lockDuration),
				"defaultMessageTimeToLive": llx.StringData(defaultMessageTimeToLive),
				"requiresSession":          llx.BoolData(requiresSession),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlSub)
		}
	}

	return res, nil
}

// networkRuleSet fetches the namespace-level network rule set (default action, public network access,
// IP rules, virtual-network rules, trusted-service-access).
func (a *mqlAzureSubscriptionServiceBusServiceNamespace) networkRuleSet() (any, error) {
	props, err := a.fetchNetworkRuleSetProperties()
	if err != nil {
		return nil, err
	}
	if props == nil {
		return nil, nil
	}
	return convert.JsonToDict(props)
}

func (a *mqlAzureSubscriptionServiceBusServiceNamespace) networkRules() (*mqlAzureSubscriptionServiceBusServiceNamespaceNetworkRules, error) {
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
		mqlRule, err := CreateResource(a.MqlRuntime, "azure.subscription.serviceBusService.namespace.networkRules.virtualNetworkRule",
			map[string]*llx.RawData{
				"__id":                             llx.StringData(id),
				"ignoreMissingVnetServiceEndpoint": llx.BoolData(ignore),
			})
		if err != nil {
			return nil, err
		}
		mqlRule.(*mqlAzureSubscriptionServiceBusServiceNamespaceNetworkRulesVirtualNetworkRule).cacheSubnetID = subnetID
		vnetRules = append(vnetRules, mqlRule)
	}

	res, err := CreateResource(a.MqlRuntime, "azure.subscription.serviceBusService.namespace.networkRules", map[string]*llx.RawData{
		"__id":                        llx.StringData(a.Id.Data + "/networkRules"),
		"defaultAction":               llx.StringData(defaultAction),
		"publicNetworkAccess":         llx.StringData(publicNetworkAccess),
		"trustedServiceAccessEnabled": llx.BoolData(trustedServiceAccess),
		"ipRules":                     llx.ArrayData(ipRules, types.Dict),
		"virtualNetworkRules":         llx.ArrayData(vnetRules, types.Resource("azure.subscription.serviceBusService.namespace.networkRules.virtualNetworkRule")),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionServiceBusServiceNamespaceNetworkRules), nil
}

func (a *mqlAzureSubscriptionServiceBusServiceNamespace) fetchNetworkRuleSetProperties() (*armservicebus.NetworkRuleSetProperties, error) {
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
	client, err := armservicebus.NewNamespacesClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
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

type mqlAzureSubscriptionServiceBusServiceNamespaceNetworkRulesVirtualNetworkRuleInternal struct {
	cacheSubnetID string
}

func (a *mqlAzureSubscriptionServiceBusServiceNamespaceNetworkRulesVirtualNetworkRule) subnet() (*mqlAzureSubscriptionNetworkServiceSubnet, error) {
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
