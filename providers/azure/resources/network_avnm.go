// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	network "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v10"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

// -----------------------------------------------------------------------------
// Azure Virtual Network Manager
// -----------------------------------------------------------------------------

type mqlAzureSubscriptionNetworkServiceNetworkManagerInternal struct {
	cacheSystemData any
}

func (a *mqlAzureSubscriptionNetworkService) networkManagers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	client, err := network.NewManagersClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListBySubscriptionPager(&network.ManagersClientListBySubscriptionOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, mgr := range page.Value {
			if mgr == nil {
				continue
			}
			var provisioningState, description, resourceGuid string
			scopeAccesses := []any{}
			scopeSubscriptions := []any{}
			scopeManagementGroups := []any{}
			if p := mgr.Properties; p != nil {
				if p.ProvisioningState != nil {
					provisioningState = string(*p.ProvisioningState)
				}
				description = convert.ToValue(p.Description)
				resourceGuid = convert.ToValue(p.ResourceGUID)
				for _, sa := range p.NetworkManagerScopeAccesses {
					if sa != nil {
						scopeAccesses = append(scopeAccesses, string(*sa))
					}
				}
				if p.NetworkManagerScopes != nil {
					scopeSubscriptions = strPtrsToAny(p.NetworkManagerScopes.Subscriptions)
					scopeManagementGroups = strPtrsToAny(p.NetworkManagerScopes.ManagementGroups)
				}
			}
			mqlMgr, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.networkManager",
				map[string]*llx.RawData{
					"id":                    llx.StringDataPtr(mgr.ID),
					"name":                  llx.StringDataPtr(mgr.Name),
					"location":              llx.StringDataPtr(mgr.Location),
					"tags":                  llx.MapData(convert.PtrMapStrToInterface(mgr.Tags), types.String),
					"type":                  llx.StringDataPtr(mgr.Type),
					"etag":                  llx.StringDataPtr(mgr.Etag),
					"provisioningState":     llx.StringData(provisioningState),
					"description":           llx.StringData(description),
					"resourceGuid":          llx.StringData(resourceGuid),
					"scopeAccesses":         llx.ArrayData(scopeAccesses, types.String),
					"scopeSubscriptions":    llx.ArrayData(scopeSubscriptions, types.String),
					"scopeManagementGroups": llx.ArrayData(scopeManagementGroups, types.String),
				})
			if err != nil {
				return nil, err
			}
			sysData, err := convert.JsonToDict(mgr.SystemData)
			if err != nil {
				return nil, err
			}
			mqlMgr.(*mqlAzureSubscriptionNetworkServiceNetworkManager).cacheSystemData = sysData
			res = append(res, mqlMgr)
		}
	}
	return res, nil
}

// strPtrsToAny converts a []*string into []any, skipping nil elements (the
// convert helpers panic on nil pointers).
func strPtrsToAny(s []*string) []any {
	res := []any{}
	for _, v := range s {
		if v != nil {
			res = append(res, *v)
		}
	}
	return res
}

func (a *mqlAzureSubscriptionNetworkServiceNetworkManager) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceNetworkManager) systemMetadata() (*mqlAzureSubscriptionSystemData, error) {
	return systemMetadataFromRaw(a.MqlRuntime, a.Id.Data, a.cacheSystemData, &a.SystemMetadata)
}

func (a *mqlAzureSubscriptionNetworkServiceNetworkManager) networkGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	resourceID, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	managerName, err := resourceID.Component("networkManagers")
	if err != nil {
		return nil, err
	}
	client, err := network.NewGroupsClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListPager(resourceID.ResourceGroup, managerName, &network.GroupsClientListOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, g := range page.Value {
			if g == nil {
				continue
			}
			var provisioningState, description, memberType, resourceGuid string
			if p := g.Properties; p != nil {
				if p.ProvisioningState != nil {
					provisioningState = string(*p.ProvisioningState)
				}
				description = convert.ToValue(p.Description)
				resourceGuid = convert.ToValue(p.ResourceGUID)
				if p.MemberType != nil {
					memberType = string(*p.MemberType)
				}
			}
			mqlGroup, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.networkManager.networkGroup",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(g.ID),
					"name":              llx.StringDataPtr(g.Name),
					"type":              llx.StringDataPtr(g.Type),
					"etag":              llx.StringDataPtr(g.Etag),
					"provisioningState": llx.StringData(provisioningState),
					"description":       llx.StringData(description),
					"memberType":        llx.StringData(memberType),
					"resourceGuid":      llx.StringData(resourceGuid),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlGroup)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceNetworkManagerNetworkGroup) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceNetworkManager) securityAdminConfigurations() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	resourceID, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	managerName, err := resourceID.Component("networkManagers")
	if err != nil {
		return nil, err
	}
	client, err := network.NewSecurityAdminConfigurationsClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListPager(resourceID.ResourceGroup, managerName, &network.SecurityAdminConfigurationsClientListOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, cfg := range page.Value {
			if cfg == nil {
				continue
			}
			var provisioningState, description string
			intentServices := []any{}
			if p := cfg.Properties; p != nil {
				if p.ProvisioningState != nil {
					provisioningState = string(*p.ProvisioningState)
				}
				description = convert.ToValue(p.Description)
				for _, svc := range p.ApplyOnNetworkIntentPolicyBasedServices {
					if svc != nil {
						intentServices = append(intentServices, string(*svc))
					}
				}
			}
			mqlCfg, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.networkManager.securityAdminConfiguration",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(cfg.ID),
					"name":              llx.StringDataPtr(cfg.Name),
					"type":              llx.StringDataPtr(cfg.Type),
					"etag":              llx.StringDataPtr(cfg.Etag),
					"provisioningState": llx.StringData(provisioningState),
					"description":       llx.StringData(description),
					"applyOnNetworkIntentPolicyBasedServices": llx.ArrayData(intentServices, types.String),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlCfg)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceNetworkManagerSecurityAdminConfiguration) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceNetworkManagerSecurityAdminConfiguration) ruleCollections() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	resourceID, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	managerName, err := resourceID.Component("networkManagers")
	if err != nil {
		return nil, err
	}
	configName, err := resourceID.Component("securityAdminConfigurations")
	if err != nil {
		return nil, err
	}
	client, err := network.NewAdminRuleCollectionsClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListPager(resourceID.ResourceGroup, managerName, configName, &network.AdminRuleCollectionsClientListOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, rc := range page.Value {
			if rc == nil {
				continue
			}
			var provisioningState, description string
			appliesTo := []any{}
			if p := rc.Properties; p != nil {
				if p.ProvisioningState != nil {
					provisioningState = string(*p.ProvisioningState)
				}
				description = convert.ToValue(p.Description)
				for _, grp := range p.AppliesToGroups {
					if grp != nil && grp.NetworkGroupID != nil {
						appliesTo = append(appliesTo, *grp.NetworkGroupID)
					}
				}
			}
			mqlRc, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.networkManager.securityAdminConfiguration.ruleCollection",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(rc.ID),
					"name":              llx.StringDataPtr(rc.Name),
					"type":              llx.StringDataPtr(rc.Type),
					"etag":              llx.StringDataPtr(rc.Etag),
					"provisioningState": llx.StringData(provisioningState),
					"description":       llx.StringData(description),
					"appliesToGroupIds": llx.ArrayData(appliesTo, types.String),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRc)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceNetworkManagerSecurityAdminConfigurationRuleCollection) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceNetworkManagerSecurityAdminConfigurationRuleCollection) rules() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	resourceID, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	managerName, err := resourceID.Component("networkManagers")
	if err != nil {
		return nil, err
	}
	configName, err := resourceID.Component("securityAdminConfigurations")
	if err != nil {
		return nil, err
	}
	collectionName, err := resourceID.Component("ruleCollections")
	if err != nil {
		return nil, err
	}
	client, err := network.NewAdminRulesClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListPager(resourceID.ResourceGroup, managerName, configName, collectionName, &network.AdminRulesClientListOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, rule := range page.Value {
			mqlRule, err := azureAdminRuleToMql(a.MqlRuntime, rule)
			if err != nil {
				return nil, err
			}
			if mqlRule != nil {
				res = append(res, mqlRule)
			}
		}
	}
	return res, nil
}

// azureAdminRuleToMql maps a network manager admin rule into its MQL resource.
// Admin rules are polymorphic: custom rules (*AdminRule) carry a full set of
// match fields, while default rules (*DefaultAdminRule) expose the same fields
// as read-only values derived from a built-in flag. Both are flattened to the
// same typed fields so a query does not have to care which kind it is.
func azureAdminRuleToMql(runtime *plugin.Runtime, rule network.BaseAdminRuleClassification) (*mqlAzureSubscriptionNetworkServiceNetworkManagerSecurityAdminConfigurationRuleCollectionRule, error) {
	var id, name, typ, etag, kind, provisioningState string
	var access, direction, protocol, description string
	var priority int64
	sourcePortRanges := []any{}
	destinationPortRanges := []any{}
	sources := []any{}
	destinations := []any{}

	switch r := rule.(type) {
	case *network.AdminRule:
		id = convert.ToValue(r.ID)
		name = convert.ToValue(r.Name)
		typ = convert.ToValue(r.Type)
		etag = convert.ToValue(r.Etag)
		kind = string(network.AdminRuleKindCustom)
		if p := r.Properties; p != nil {
			if p.Access != nil {
				access = string(*p.Access)
			}
			if p.Direction != nil {
				direction = string(*p.Direction)
			}
			if p.Protocol != nil {
				protocol = string(*p.Protocol)
			}
			if p.Priority != nil {
				priority = int64(*p.Priority)
			}
			if p.ProvisioningState != nil {
				provisioningState = string(*p.ProvisioningState)
			}
			description = convert.ToValue(p.Description)
			sourcePortRanges = strPtrsToAny(p.SourcePortRanges)
			destinationPortRanges = strPtrsToAny(p.DestinationPortRanges)
			sources = addressPrefixItemsToAny(p.Sources)
			destinations = addressPrefixItemsToAny(p.Destinations)
		}
	case *network.DefaultAdminRule:
		id = convert.ToValue(r.ID)
		name = convert.ToValue(r.Name)
		typ = convert.ToValue(r.Type)
		etag = convert.ToValue(r.Etag)
		kind = string(network.AdminRuleKindDefault)
		if p := r.Properties; p != nil {
			if p.Access != nil {
				access = string(*p.Access)
			}
			if p.Direction != nil {
				direction = string(*p.Direction)
			}
			if p.Protocol != nil {
				protocol = string(*p.Protocol)
			}
			if p.Priority != nil {
				priority = int64(*p.Priority)
			}
			if p.ProvisioningState != nil {
				provisioningState = string(*p.ProvisioningState)
			}
			description = convert.ToValue(p.Description)
			sourcePortRanges = strPtrsToAny(p.SourcePortRanges)
			destinationPortRanges = strPtrsToAny(p.DestinationPortRanges)
			sources = addressPrefixItemsToAny(p.Sources)
			destinations = addressPrefixItemsToAny(p.Destinations)
		}
	default:
		// Unknown rule kind; skip rather than emit a malformed resource.
		return nil, nil
	}

	mqlRule, err := CreateResource(runtime, "azure.subscription.networkService.networkManager.securityAdminConfiguration.ruleCollection.rule",
		map[string]*llx.RawData{
			"id":                    llx.StringData(id),
			"name":                  llx.StringData(name),
			"type":                  llx.StringData(typ),
			"etag":                  llx.StringData(etag),
			"provisioningState":     llx.StringData(provisioningState),
			"kind":                  llx.StringData(kind),
			"description":           llx.StringData(description),
			"access":                llx.StringData(access),
			"direction":             llx.StringData(direction),
			"priority":              llx.IntData(priority),
			"protocol":              llx.StringData(protocol),
			"sourcePortRanges":      llx.ArrayData(sourcePortRanges, types.String),
			"destinationPortRanges": llx.ArrayData(destinationPortRanges, types.String),
			"sources":               llx.ArrayData(sources, types.String),
			"destinations":          llx.ArrayData(destinations, types.String),
		})
	if err != nil {
		return nil, err
	}
	return mqlRule.(*mqlAzureSubscriptionNetworkServiceNetworkManagerSecurityAdminConfigurationRuleCollectionRule), nil
}

// addressPrefixItemsToAny flattens a slice of *AddressPrefixItem into their
// address prefixes, skipping nil entries and nil prefixes.
func addressPrefixItemsToAny(items []*network.AddressPrefixItem) []any {
	res := []any{}
	for _, item := range items {
		if item != nil && item.AddressPrefix != nil {
			res = append(res, *item.AddressPrefix)
		}
	}
	return res
}

func (a *mqlAzureSubscriptionNetworkServiceNetworkManagerSecurityAdminConfigurationRuleCollectionRule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceNetworkManager) connectivityConfigurations() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	resourceID, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}
	managerName, err := resourceID.Component("networkManagers")
	if err != nil {
		return nil, err
	}
	client, err := network.NewConnectivityConfigurationsClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListPager(resourceID.ResourceGroup, managerName, &network.ConnectivityConfigurationsClientListOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, cfg := range page.Value {
			if cfg == nil {
				continue
			}
			var provisioningState, description, topology string
			var isGlobal, deleteExistingPeering bool
			appliesTo := []any{}
			hubs := []any{}
			if p := cfg.Properties; p != nil {
				if p.ProvisioningState != nil {
					provisioningState = string(*p.ProvisioningState)
				}
				description = convert.ToValue(p.Description)
				if p.ConnectivityTopology != nil {
					topology = string(*p.ConnectivityTopology)
				}
				if p.IsGlobal != nil {
					isGlobal = *p.IsGlobal == network.IsGlobalTrue
				}
				if p.DeleteExistingPeering != nil {
					deleteExistingPeering = *p.DeleteExistingPeering == network.DeleteExistingPeeringTrue
				}
				for _, grp := range p.AppliesToGroups {
					if grp != nil && grp.NetworkGroupID != nil {
						appliesTo = append(appliesTo, *grp.NetworkGroupID)
					}
				}
				hubDicts, err := convert.JsonToDictSlice(p.Hubs)
				if err != nil {
					return nil, err
				}
				hubs = hubDicts
			}
			mqlCfg, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.networkManager.connectivityConfiguration",
				map[string]*llx.RawData{
					"id":                    llx.StringDataPtr(cfg.ID),
					"name":                  llx.StringDataPtr(cfg.Name),
					"type":                  llx.StringDataPtr(cfg.Type),
					"etag":                  llx.StringDataPtr(cfg.Etag),
					"provisioningState":     llx.StringData(provisioningState),
					"description":           llx.StringData(description),
					"connectivityTopology":  llx.StringData(topology),
					"isGlobal":              llx.BoolData(isGlobal),
					"deleteExistingPeering": llx.BoolData(deleteExistingPeering),
					"appliesToGroupIds":     llx.ArrayData(appliesTo, types.String),
					"hubs":                  llx.ArrayData(hubs, types.Dict),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlCfg)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceNetworkManagerConnectivityConfiguration) id() (string, error) {
	return a.Id.Data, nil
}
