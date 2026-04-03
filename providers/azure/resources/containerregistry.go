// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerregistry/armcontainerregistry"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlAzureSubscriptionContainerRegistryServiceRegistryInternal struct {
	cacheNetworkRuleSet             *armcontainerregistry.NetworkRuleSet
	cachePolicies                   *armcontainerregistry.Policies
	cacheEncryption                 *armcontainerregistry.EncryptionProperty
	cachePrivateEndpointConnections []*armcontainerregistry.PrivateEndpointConnection
}

type mqlAzureSubscriptionContainerRegistryServiceRegistryTokenInternal struct {
	cacheScopeMapID string
}

func (a *mqlAzureSubscriptionContainerRegistryService) id() (string, error) {
	return "azure.subscription.containerRegistryService/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionContainerRegistryService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionContainerRegistryServiceRegistry) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionContainerRegistryService) registries() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	clientFactory, err := armcontainerregistry.NewClientFactory(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	client := clientFactory.NewRegistriesClient()
	pager := client.NewListPager(nil)

	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, reg := range page.Value {
			if reg == nil {
				continue
			}
			mqlReg, err := createRegistryResource(a.MqlRuntime, reg)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlReg)
		}
	}
	return res, nil
}

func createRegistryResource(runtime *plugin.Runtime, reg *armcontainerregistry.Registry) (*mqlAzureSubscriptionContainerRegistryServiceRegistry, error) {
	props := reg.Properties
	if props == nil {
		props = &armcontainerregistry.RegistryProperties{}
	}

	identity, err := convert.JsonToDict(reg.Identity)
	if err != nil {
		return nil, err
	}

	var skuName string
	if reg.SKU != nil && reg.SKU.Name != nil {
		skuName = string(*reg.SKU.Name)
	}

	var publicNetworkAccess string
	if props.PublicNetworkAccess != nil {
		publicNetworkAccess = string(*props.PublicNetworkAccess)
	}

	var networkRuleBypassOptions string
	if props.NetworkRuleBypassOptions != nil {
		networkRuleBypassOptions = string(*props.NetworkRuleBypassOptions)
	}

	var zoneRedundancy string
	if props.ZoneRedundancy != nil {
		zoneRedundancy = string(*props.ZoneRedundancy)
	}

	var provisioningState string
	if props.ProvisioningState != nil {
		provisioningState = string(*props.ProvisioningState)
	}

	var creationDate *llx.RawData
	if props.CreationDate != nil {
		creationDate = llx.TimeDataPtr(props.CreationDate)
	} else {
		creationDate = llx.NilData
	}

	resource, err := CreateResource(runtime, ResourceAzureSubscriptionContainerRegistryServiceRegistry,
		map[string]*llx.RawData{
			"id":                       llx.StringDataPtr(reg.ID),
			"name":                     llx.StringDataPtr(reg.Name),
			"location":                 llx.StringDataPtr(reg.Location),
			"type":                     llx.StringDataPtr(reg.Type),
			"tags":                     llx.MapData(convert.PtrMapStrToInterface(reg.Tags), types.String),
			"skuName":                  llx.StringData(skuName),
			"identity":                 llx.DictData(identity),
			"adminUserEnabled":         llx.BoolDataPtr(props.AdminUserEnabled),
			"publicNetworkAccess":      llx.StringData(publicNetworkAccess),
			"networkRuleBypassOptions": llx.StringData(networkRuleBypassOptions),
			"zoneRedundancy":           llx.StringData(zoneRedundancy),
			"dataEndpointEnabled":      llx.BoolDataPtr(props.DataEndpointEnabled),
			"loginServer":              llx.StringDataPtr(props.LoginServer),
			"creationDate":             creationDate,
			"provisioningState":        llx.StringData(provisioningState),
		})
	if err != nil {
		return nil, err
	}

	mqlReg := resource.(*mqlAzureSubscriptionContainerRegistryServiceRegistry)
	mqlReg.cacheNetworkRuleSet = props.NetworkRuleSet
	mqlReg.cachePolicies = props.Policies
	mqlReg.cacheEncryption = props.Encryption
	mqlReg.cachePrivateEndpointConnections = props.PrivateEndpointConnections

	return mqlReg, nil
}

// networkRuleSet builds the network rule set sub-resource from cached data.
func (a *mqlAzureSubscriptionContainerRegistryServiceRegistry) networkRuleSet() (*mqlAzureSubscriptionContainerRegistryServiceRegistryNetworkRuleSet, error) {
	nrs := a.cacheNetworkRuleSet
	if nrs == nil {
		a.NetworkRuleSet.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	var defaultAction string
	if nrs.DefaultAction != nil {
		defaultAction = string(*nrs.DefaultAction)
	}

	res, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionContainerRegistryServiceRegistryNetworkRuleSet,
		map[string]*llx.RawData{
			"id":            llx.StringData(a.Id.Data + "/networkRuleSet"),
			"defaultAction": llx.StringData(defaultAction),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionContainerRegistryServiceRegistryNetworkRuleSet), nil
}

func (a *mqlAzureSubscriptionContainerRegistryServiceRegistryNetworkRuleSet) id() (string, error) {
	return a.Id.Data, nil
}

// ipRules returns the IP rules from the parent registry's cached network rule set.
func (a *mqlAzureSubscriptionContainerRegistryServiceRegistryNetworkRuleSet) ipRules() ([]any, error) {
	parentID := a.Id.Data
	// parentID is like ".../networkRuleSet" — strip suffix to get registry ID
	registryID := strings.TrimSuffix(parentID, "/networkRuleSet")

	// Look up the parent registry resource to get the cached data.
	cached, ok := a.MqlRuntime.Resources.Get(ResourceAzureSubscriptionContainerRegistryServiceRegistry + "\x00" + registryID)
	if !ok {
		return []any{}, nil
	}
	registry := cached.(*mqlAzureSubscriptionContainerRegistryServiceRegistry)
	nrs := registry.cacheNetworkRuleSet
	if nrs == nil {
		return []any{}, nil
	}

	var res []any
	for i, rule := range nrs.IPRules {
		if rule == nil {
			continue
		}
		var action string
		if rule.Action != nil {
			action = string(*rule.Action)
		}
		mqlRule, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionContainerRegistryServiceRegistryNetworkRuleSetIpRule,
			map[string]*llx.RawData{
				"id":               llx.StringData(fmt.Sprintf("%s/ipRules/%d", parentID, i)),
				"ipAddressOrRange": llx.StringDataPtr(rule.IPAddressOrRange),
				"action":           llx.StringData(action),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRule)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionContainerRegistryServiceRegistryNetworkRuleSetIpRule) id() (string, error) {
	return a.Id.Data, nil
}

// policies builds the flattened policies sub-resource from cached data.
func (a *mqlAzureSubscriptionContainerRegistryServiceRegistry) policies() (*mqlAzureSubscriptionContainerRegistryServiceRegistryPolicies, error) {
	p := a.cachePolicies
	if p == nil {
		a.Policies.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	var trustEnabled bool
	var trustType string
	if p.TrustPolicy != nil {
		if p.TrustPolicy.Status != nil {
			trustEnabled = string(*p.TrustPolicy.Status) == "enabled"
		}
		if p.TrustPolicy.Type != nil {
			trustType = string(*p.TrustPolicy.Type)
		}
	}

	var retentionEnabled bool
	var retentionDays int64
	if p.RetentionPolicy != nil {
		if p.RetentionPolicy.Status != nil {
			retentionEnabled = string(*p.RetentionPolicy.Status) == "enabled"
		}
		if p.RetentionPolicy.Days != nil {
			retentionDays = int64(*p.RetentionPolicy.Days)
		}
	}

	var quarantineEnabled bool
	if p.QuarantinePolicy != nil && p.QuarantinePolicy.Status != nil {
		quarantineEnabled = string(*p.QuarantinePolicy.Status) == "enabled"
	}

	var exportEnabled bool
	if p.ExportPolicy != nil && p.ExportPolicy.Status != nil {
		exportEnabled = string(*p.ExportPolicy.Status) == "enabled"
	}

	res, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionContainerRegistryServiceRegistryPolicies,
		map[string]*llx.RawData{
			"id":                      llx.StringData(a.Id.Data + "/policies"),
			"trustPolicyEnabled":      llx.BoolData(trustEnabled),
			"trustPolicyType":         llx.StringData(trustType),
			"retentionPolicyEnabled":  llx.BoolData(retentionEnabled),
			"retentionPolicyDays":     llx.IntData(retentionDays),
			"quarantinePolicyEnabled": llx.BoolData(quarantineEnabled),
			"exportPolicyEnabled":     llx.BoolData(exportEnabled),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionContainerRegistryServiceRegistryPolicies), nil
}

func (a *mqlAzureSubscriptionContainerRegistryServiceRegistryPolicies) id() (string, error) {
	return a.Id.Data, nil
}

// encryption builds the encryption sub-resource from cached data.
func (a *mqlAzureSubscriptionContainerRegistryServiceRegistry) encryption() (*mqlAzureSubscriptionContainerRegistryServiceRegistryEncryption, error) {
	enc := a.cacheEncryption
	if enc == nil {
		a.Encryption.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	var status string
	if enc.Status != nil {
		status = string(*enc.Status)
	}

	var keyVaultKeyIdentifier string
	var keyRotationEnabled bool
	var lastKeyRotationTimestamp *llx.RawData
	if enc.KeyVaultProperties != nil {
		if enc.KeyVaultProperties.KeyIdentifier != nil {
			keyVaultKeyIdentifier = *enc.KeyVaultProperties.KeyIdentifier
		}
		if enc.KeyVaultProperties.KeyRotationEnabled != nil {
			keyRotationEnabled = *enc.KeyVaultProperties.KeyRotationEnabled
		}
		if enc.KeyVaultProperties.LastKeyRotationTimestamp != nil {
			lastKeyRotationTimestamp = llx.TimeDataPtr(enc.KeyVaultProperties.LastKeyRotationTimestamp)
		}
	}
	if lastKeyRotationTimestamp == nil {
		lastKeyRotationTimestamp = llx.NilData
	}

	res, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionContainerRegistryServiceRegistryEncryption,
		map[string]*llx.RawData{
			"id":                       llx.StringData(a.Id.Data + "/encryption"),
			"status":                   llx.StringData(status),
			"keyVaultKeyIdentifier":    llx.StringData(keyVaultKeyIdentifier),
			"keyRotationEnabled":       llx.BoolData(keyRotationEnabled),
			"lastKeyRotationTimestamp": lastKeyRotationTimestamp,
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionContainerRegistryServiceRegistryEncryption), nil
}

func (a *mqlAzureSubscriptionContainerRegistryServiceRegistryEncryption) id() (string, error) {
	return a.Id.Data, nil
}

// key returns a typed reference to the Key Vault key used for CMK encryption.
func (a *mqlAzureSubscriptionContainerRegistryServiceRegistryEncryption) key() (*mqlAzureSubscriptionKeyVaultServiceKey, error) {
	keyURI := a.KeyVaultKeyIdentifier.Data
	if keyURI == "" {
		a.Key.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return newKeyVaultKeyResource(a.MqlRuntime, keyURI)
}

// privateEndpointConnections builds the shared private endpoint connection resources.
func (a *mqlAzureSubscriptionContainerRegistryServiceRegistry) privateEndpointConnections() ([]any, error) {
	var res []any
	for _, pec := range a.cachePrivateEndpointConnections {
		if pec == nil {
			continue
		}

		var name, resType string
		if pec.ID != nil {
			connResourceID, err := ParseResourceID(*pec.ID)
			if err == nil {
				if nameComp, err := connResourceID.Component("privateEndpointConnections"); err == nil {
					name = nameComp
				}
				if connResourceID.Provider != "" {
					resType = connResourceID.Provider + "/registries/privateEndpointConnections"
				}
			}
			if name == "" {
				parts := strings.Split(*pec.ID, "/")
				if len(parts) > 0 {
					name = parts[len(parts)-1]
				}
			}
		}
		if resType == "" {
			resType = "Microsoft.ContainerRegistry/registries/privateEndpointConnections"
		}

		pecArgs := map[string]*llx.RawData{
			"__id": llx.StringDataPtr(pec.ID),
			"id":   llx.StringDataPtr(pec.ID),
			"name": llx.StringData(name),
			"type": llx.StringData(resType),
		}

		if pec.Properties != nil {
			props := pec.Properties
			propsMap, err := convert.JsonToDict(props)
			if err != nil {
				return nil, err
			}
			pecArgs["properties"] = llx.DictData(propsMap)

			if props.PrivateEndpoint != nil {
				pecArgs["privateEndpointId"] = llx.StringDataPtr(props.PrivateEndpoint.ID)
			}
			if props.PrivateLinkServiceConnectionState != nil {
				stateArgs := map[string]*llx.RawData{}
				if props.PrivateLinkServiceConnectionState.ActionsRequired != nil {
					stateArgs["actionsRequired"] = llx.StringData(string(*props.PrivateLinkServiceConnectionState.ActionsRequired))
				}
				if props.PrivateLinkServiceConnectionState.Description != nil {
					stateArgs["description"] = llx.StringDataPtr(props.PrivateLinkServiceConnectionState.Description)
				}
				if props.PrivateLinkServiceConnectionState.Status != nil {
					stateArgs["status"] = llx.StringData(string(*props.PrivateLinkServiceConnectionState.Status))
				}
				stateRes, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionPrivateEndpointConnectionConnectionState, stateArgs)
				if err != nil {
					return nil, err
				}
				pecArgs["privateLinkServiceConnectionState"] = llx.ResourceData(stateRes, ResourceAzureSubscriptionPrivateEndpointConnectionConnectionState)
			}
			if props.ProvisioningState != nil {
				pecArgs["provisioningState"] = llx.StringData(string(*props.ProvisioningState))
			}
		}

		mqlRes, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionPrivateEndpointConnection, pecArgs)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRes)
	}
	return res, nil
}

// webhooks fetches all webhooks for the registry.
func (a *mqlAzureSubscriptionContainerRegistryServiceRegistry) webhooks() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	registryName, err := resourceID.Component("registries")
	if err != nil {
		return nil, err
	}

	client, err := armcontainerregistry.NewWebhooksClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(resourceID.ResourceGroup, registryName, nil)
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, wh := range page.Value {
			if wh == nil {
				continue
			}

			var status, scope, provisioningState string
			var actions []any
			if wh.Properties != nil {
				if wh.Properties.Status != nil {
					status = string(*wh.Properties.Status)
				}
				if wh.Properties.Scope != nil {
					scope = *wh.Properties.Scope
				}
				if wh.Properties.ProvisioningState != nil {
					provisioningState = string(*wh.Properties.ProvisioningState)
				}
				for _, act := range wh.Properties.Actions {
					if act != nil {
						actions = append(actions, string(*act))
					}
				}
			}

			mqlWh, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionContainerRegistryServiceRegistryWebhook,
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(wh.ID),
					"name":              llx.StringDataPtr(wh.Name),
					"location":          llx.StringDataPtr(wh.Location),
					"type":              llx.StringDataPtr(wh.Type),
					"tags":              llx.MapData(convert.PtrMapStrToInterface(wh.Tags), types.String),
					"status":            llx.StringData(status),
					"scope":             llx.StringData(scope),
					"actions":           llx.ArrayData(actions, types.String),
					"provisioningState": llx.StringData(provisioningState),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlWh)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionContainerRegistryServiceRegistryWebhook) id() (string, error) {
	return a.Id.Data, nil
}

// replications fetches all geo-replications for the registry.
func (a *mqlAzureSubscriptionContainerRegistryServiceRegistry) replications() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	registryName, err := resourceID.Component("registries")
	if err != nil {
		return nil, err
	}

	client, err := armcontainerregistry.NewReplicationsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(resourceID.ResourceGroup, registryName, nil)
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, repl := range page.Value {
			if repl == nil {
				continue
			}

			var regionEndpointEnabled bool
			var zoneRedundancy, provisioningState string
			if repl.Properties != nil {
				if repl.Properties.RegionEndpointEnabled != nil {
					regionEndpointEnabled = *repl.Properties.RegionEndpointEnabled
				}
				if repl.Properties.ZoneRedundancy != nil {
					zoneRedundancy = string(*repl.Properties.ZoneRedundancy)
				}
				if repl.Properties.ProvisioningState != nil {
					provisioningState = string(*repl.Properties.ProvisioningState)
				}
			}

			mqlRepl, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionContainerRegistryServiceRegistryReplication,
				map[string]*llx.RawData{
					"id":                    llx.StringDataPtr(repl.ID),
					"name":                  llx.StringDataPtr(repl.Name),
					"location":              llx.StringDataPtr(repl.Location),
					"type":                  llx.StringDataPtr(repl.Type),
					"tags":                  llx.MapData(convert.PtrMapStrToInterface(repl.Tags), types.String),
					"regionEndpointEnabled": llx.BoolData(regionEndpointEnabled),
					"zoneRedundancy":        llx.StringData(zoneRedundancy),
					"provisioningState":     llx.StringData(provisioningState),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRepl)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionContainerRegistryServiceRegistryReplication) id() (string, error) {
	return a.Id.Data, nil
}

// scopeMaps fetches all scope maps for the registry.
func (a *mqlAzureSubscriptionContainerRegistryServiceRegistry) scopeMaps() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	registryName, err := resourceID.Component("registries")
	if err != nil {
		return nil, err
	}

	client, err := armcontainerregistry.NewScopeMapsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(resourceID.ResourceGroup, registryName, nil)
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, sm := range page.Value {
			if sm == nil {
				continue
			}
			mqlSm, err := createScopeMapResource(a.MqlRuntime, sm)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlSm)
		}
	}
	return res, nil
}

func createScopeMapResource(runtime *plugin.Runtime, sm *armcontainerregistry.ScopeMap) (*mqlAzureSubscriptionContainerRegistryServiceRegistryScopeMap, error) {
	var description, provisioningState, resType string
	var actions []any
	var creationDate *llx.RawData

	if sm.Properties != nil {
		if sm.Properties.Description != nil {
			description = *sm.Properties.Description
		}
		if sm.Properties.ProvisioningState != nil {
			provisioningState = string(*sm.Properties.ProvisioningState)
		}
		for _, act := range sm.Properties.Actions {
			if act != nil {
				actions = append(actions, *act)
			}
		}
		if sm.Properties.CreationDate != nil {
			creationDate = llx.TimeDataPtr(sm.Properties.CreationDate)
		}
	}
	if creationDate == nil {
		creationDate = llx.NilData
	}
	if sm.Type != nil {
		resType = *sm.Type
	}

	res, err := CreateResource(runtime, ResourceAzureSubscriptionContainerRegistryServiceRegistryScopeMap,
		map[string]*llx.RawData{
			"id":                llx.StringDataPtr(sm.ID),
			"name":              llx.StringDataPtr(sm.Name),
			"type":              llx.StringData(resType),
			"description":       llx.StringData(description),
			"actions":           llx.ArrayData(actions, types.String),
			"creationDate":      creationDate,
			"provisioningState": llx.StringData(provisioningState),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionContainerRegistryServiceRegistryScopeMap), nil
}

func (a *mqlAzureSubscriptionContainerRegistryServiceRegistryScopeMap) id() (string, error) {
	return a.Id.Data, nil
}

// initAzureSubscriptionContainerRegistryServiceRegistryScopeMap fetches a scope map by ID
// when it hasn't been pre-cached via the parent registry's scopeMaps() call.
func initAzureSubscriptionContainerRegistryServiceRegistryScopeMap(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	idRaw, ok := args["id"]
	if !ok || idRaw == nil {
		return nil, nil, errors.New("id required to fetch scope map")
	}
	id, ok := idRaw.Value.(string)
	if !ok || id == "" {
		return nil, nil, errors.New("id must be a non-empty string")
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, nil, err
	}
	registryName, err := resourceID.Component("registries")
	if err != nil {
		return nil, nil, err
	}
	scopeMapName, err := resourceID.Component("scopeMaps")
	if err != nil {
		return nil, nil, err
	}

	ctx := context.Background()
	client, err := armcontainerregistry.NewScopeMapsClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, nil, err
	}

	resp, err := client.Get(ctx, resourceID.ResourceGroup, registryName, scopeMapName, nil)
	if err != nil {
		return nil, nil, err
	}

	mqlSm, err := createScopeMapResource(runtime, &resp.ScopeMap)
	if err != nil {
		return nil, nil, err
	}
	return nil, mqlSm, nil
}

// tokens fetches all tokens for the registry.
func (a *mqlAzureSubscriptionContainerRegistryServiceRegistry) tokens() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data

	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	registryName, err := resourceID.Component("registries")
	if err != nil {
		return nil, err
	}

	client, err := armcontainerregistry.NewTokensClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(resourceID.ResourceGroup, registryName, nil)
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, tk := range page.Value {
			if tk == nil {
				continue
			}
			mqlTk, err := createTokenResource(a.MqlRuntime, tk)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlTk)
		}
	}
	return res, nil
}

func createTokenResource(runtime *plugin.Runtime, tk *armcontainerregistry.Token) (*mqlAzureSubscriptionContainerRegistryServiceRegistryToken, error) {
	var status, provisioningState, resType, scopeMapID string
	var creationDate *llx.RawData
	var certificates []any

	if tk.Properties != nil {
		if tk.Properties.Status != nil {
			status = string(*tk.Properties.Status)
		}
		if tk.Properties.ProvisioningState != nil {
			provisioningState = string(*tk.Properties.ProvisioningState)
		}
		if tk.Properties.ScopeMapID != nil {
			scopeMapID = *tk.Properties.ScopeMapID
		}
		if tk.Properties.CreationDate != nil {
			creationDate = llx.TimeDataPtr(tk.Properties.CreationDate)
		}
		if tk.Properties.Credentials != nil {
			for _, cert := range tk.Properties.Credentials.Certificates {
				if cert == nil {
					continue
				}
				certDict, err := convert.JsonToDict(cert)
				if err != nil {
					return nil, err
				}
				certificates = append(certificates, certDict)
			}
		}
	}
	if creationDate == nil {
		creationDate = llx.NilData
	}
	if tk.Type != nil {
		resType = *tk.Type
	}

	res, err := CreateResource(runtime, ResourceAzureSubscriptionContainerRegistryServiceRegistryToken,
		map[string]*llx.RawData{
			"id":                llx.StringDataPtr(tk.ID),
			"name":              llx.StringDataPtr(tk.Name),
			"type":              llx.StringData(resType),
			"status":            llx.StringData(status),
			"creationDate":      creationDate,
			"provisioningState": llx.StringData(provisioningState),
			"certificates":      llx.ArrayData(certificates, types.Dict),
		})
	if err != nil {
		return nil, err
	}

	mqlToken := res.(*mqlAzureSubscriptionContainerRegistryServiceRegistryToken)
	mqlToken.cacheScopeMapID = scopeMapID

	return mqlToken, nil
}

func (a *mqlAzureSubscriptionContainerRegistryServiceRegistryToken) id() (string, error) {
	return a.Id.Data, nil
}

// scopeMap returns a typed reference to the token's scope map.
func (a *mqlAzureSubscriptionContainerRegistryServiceRegistryToken) scopeMap() (*mqlAzureSubscriptionContainerRegistryServiceRegistryScopeMap, error) {
	if a.cacheScopeMapID == "" {
		a.ScopeMap.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	res, err := NewResource(a.MqlRuntime, ResourceAzureSubscriptionContainerRegistryServiceRegistryScopeMap,
		map[string]*llx.RawData{
			"id": llx.StringData(a.cacheScopeMapID),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionContainerRegistryServiceRegistryScopeMap), nil
}
