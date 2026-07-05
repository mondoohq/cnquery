// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	network "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v10"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

// -----------------------------------------------------------------------------
// Network virtual appliances
// -----------------------------------------------------------------------------

type mqlAzureSubscriptionNetworkServiceVirtualApplianceInternal struct {
	cacheVirtualHubId *string
}

func (a *mqlAzureSubscriptionNetworkService) virtualAppliances() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	client, err := network.NewVirtualAppliancesClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListPager(&network.VirtualAppliancesClientListOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, nva := range page.Value {
			if nva == nil {
				continue
			}
			var provisioningState, addressPrefix, deploymentType, sshPublicKey string
			var vendor, bundledScaleUnit, marketplaceVersion string
			var asn int64
			var virtualHubId *string
			if p := nva.Properties; p != nil {
				if p.ProvisioningState != nil {
					provisioningState = string(*p.ProvisioningState)
				}
				addressPrefix = convert.ToValue(p.AddressPrefix)
				deploymentType = convert.ToValue(p.DeploymentType)
				sshPublicKey = convert.ToValue(p.SSHPublicKey)
				if p.VirtualApplianceAsn != nil {
					asn = *p.VirtualApplianceAsn
				}
				if p.NvaSKU != nil {
					vendor = convert.ToValue(p.NvaSKU.Vendor)
					bundledScaleUnit = convert.ToValue(p.NvaSKU.BundledScaleUnit)
					marketplaceVersion = convert.ToValue(p.NvaSKU.MarketPlaceVersion)
				}
				if p.VirtualHub != nil {
					virtualHubId = p.VirtualHub.ID
				}
			}
			mqlNva, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.virtualAppliance",
				map[string]*llx.RawData{
					"id":                 llx.StringDataPtr(nva.ID),
					"name":               llx.StringDataPtr(nva.Name),
					"location":           llx.StringDataPtr(nva.Location),
					"tags":               llx.MapData(convert.PtrMapStrToInterface(nva.Tags), types.String),
					"type":               llx.StringDataPtr(nva.Type),
					"etag":               llx.StringDataPtr(nva.Etag),
					"provisioningState":  llx.StringData(provisioningState),
					"asn":                llx.IntData(asn),
					"vendor":             llx.StringData(vendor),
					"bundledScaleUnit":   llx.StringData(bundledScaleUnit),
					"marketplaceVersion": llx.StringData(marketplaceVersion),
					"addressPrefix":      llx.StringData(addressPrefix),
					"deploymentType":     llx.StringData(deploymentType),
					"sshPublicKey":       llx.StringData(sshPublicKey),
				})
			if err != nil {
				return nil, err
			}
			mqlNva.(*mqlAzureSubscriptionNetworkServiceVirtualAppliance).cacheVirtualHubId = virtualHubId
			res = append(res, mqlNva)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualAppliance) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualAppliance) virtualHub() (*mqlAzureSubscriptionNetworkServiceVirtualHub, error) {
	if a.cacheVirtualHubId == nil || *a.cacheVirtualHubId == "" {
		a.VirtualHub.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(a.MqlRuntime, "azure.subscription.networkService.virtualHub", map[string]*llx.RawData{
		"id": llx.StringDataPtr(a.cacheVirtualHubId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlAzureSubscriptionNetworkServiceVirtualHub), nil
}

// -----------------------------------------------------------------------------
// Virtual routers
// -----------------------------------------------------------------------------

type mqlAzureSubscriptionNetworkServiceVirtualRouterInternal struct {
	cacheHostedSubnetId *string
}

func (a *mqlAzureSubscriptionNetworkService) virtualRouters() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	client, err := network.NewVirtualRoutersClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListPager(&network.VirtualRoutersClientListOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, vr := range page.Value {
			if vr == nil {
				continue
			}
			var provisioningState string
			var asn int64
			ips := []any{}
			var hostedSubnetId *string
			if p := vr.Properties; p != nil {
				if p.ProvisioningState != nil {
					provisioningState = string(*p.ProvisioningState)
				}
				if p.VirtualRouterAsn != nil {
					asn = *p.VirtualRouterAsn
				}
				for _, ip := range p.VirtualRouterIPs {
					if ip != nil {
						ips = append(ips, *ip)
					}
				}
				if p.HostedSubnet != nil {
					hostedSubnetId = p.HostedSubnet.ID
				}
			}
			mqlVr, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.virtualRouter",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(vr.ID),
					"name":              llx.StringDataPtr(vr.Name),
					"location":          llx.StringDataPtr(vr.Location),
					"tags":              llx.MapData(convert.PtrMapStrToInterface(vr.Tags), types.String),
					"type":              llx.StringDataPtr(vr.Type),
					"etag":              llx.StringDataPtr(vr.Etag),
					"provisioningState": llx.StringData(provisioningState),
					"asn":               llx.IntData(asn),
					"ips":               llx.ArrayData(ips, types.String),
				})
			if err != nil {
				return nil, err
			}
			mqlVr.(*mqlAzureSubscriptionNetworkServiceVirtualRouter).cacheHostedSubnetId = hostedSubnetId
			res = append(res, mqlVr)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualRouter) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualRouter) hostedSubnet() (*mqlAzureSubscriptionNetworkServiceSubnet, error) {
	if a.cacheHostedSubnetId == nil || *a.cacheHostedSubnetId == "" {
		a.HostedSubnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(a.MqlRuntime, "azure.subscription.networkService.subnet", map[string]*llx.RawData{
		"id": llx.StringDataPtr(a.cacheHostedSubnetId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlAzureSubscriptionNetworkServiceSubnet), nil
}

// -----------------------------------------------------------------------------
// Public IP prefixes
// -----------------------------------------------------------------------------

type mqlAzureSubscriptionNetworkServicePublicIpPrefixInternal struct {
	cacheCustomIpPrefixId *string
}

func (a *mqlAzureSubscriptionNetworkService) publicIpPrefixes() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	client, err := network.NewPublicIPPrefixesClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListAllPager(&network.PublicIPPrefixesClientListAllOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, prefix := range page.Value {
			if prefix == nil {
				continue
			}
			var provisioningState, ipPrefix, ipVersion, resourceGuid string
			var prefixLength int64
			ipTags := []any{}
			var customIpPrefixId *string
			if p := prefix.Properties; p != nil {
				if p.ProvisioningState != nil {
					provisioningState = string(*p.ProvisioningState)
				}
				ipPrefix = convert.ToValue(p.IPPrefix)
				resourceGuid = convert.ToValue(p.ResourceGUID)
				if p.PublicIPAddressVersion != nil {
					ipVersion = string(*p.PublicIPAddressVersion)
				}
				if p.PrefixLength != nil {
					prefixLength = int64(*p.PrefixLength)
				}
				tags, err := convert.JsonToDictSlice(p.IPTags)
				if err != nil {
					return nil, err
				}
				ipTags = tags
				if p.CustomIPPrefix != nil {
					customIpPrefixId = p.CustomIPPrefix.ID
				}
			}
			var skuName, skuTier string
			if prefix.SKU != nil {
				if prefix.SKU.Name != nil {
					skuName = string(*prefix.SKU.Name)
				}
				if prefix.SKU.Tier != nil {
					skuTier = string(*prefix.SKU.Tier)
				}
			}
			mqlPrefix, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.publicIpPrefix",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(prefix.ID),
					"name":              llx.StringDataPtr(prefix.Name),
					"location":          llx.StringDataPtr(prefix.Location),
					"tags":              llx.MapData(convert.PtrMapStrToInterface(prefix.Tags), types.String),
					"type":              llx.StringDataPtr(prefix.Type),
					"etag":              llx.StringDataPtr(prefix.Etag),
					"provisioningState": llx.StringData(provisioningState),
					"zones":             llx.ArrayData(convert.SliceStrPtrToInterface(prefix.Zones), types.String),
					"prefixLength":      llx.IntData(prefixLength),
					"ipPrefix":          llx.StringData(ipPrefix),
					"ipVersion":         llx.StringData(ipVersion),
					"skuName":           llx.StringData(skuName),
					"skuTier":           llx.StringData(skuTier),
					"resourceGuid":      llx.StringData(resourceGuid),
					"ipTags":            llx.ArrayData(ipTags, types.Dict),
				})
			if err != nil {
				return nil, err
			}
			mqlPrefix.(*mqlAzureSubscriptionNetworkServicePublicIpPrefix).cacheCustomIpPrefixId = customIpPrefixId
			res = append(res, mqlPrefix)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServicePublicIpPrefix) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServicePublicIpPrefix) customIpPrefix() (*mqlAzureSubscriptionNetworkServiceCustomIpPrefix, error) {
	if a.cacheCustomIpPrefixId == nil || *a.cacheCustomIpPrefixId == "" {
		a.CustomIpPrefix.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(a.MqlRuntime, "azure.subscription.networkService.customIpPrefix", map[string]*llx.RawData{
		"id": llx.StringDataPtr(a.cacheCustomIpPrefixId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlAzureSubscriptionNetworkServiceCustomIpPrefix), nil
}

// -----------------------------------------------------------------------------
// Custom IP prefixes (BYOIP)
// -----------------------------------------------------------------------------

func (a *mqlAzureSubscriptionNetworkService) customIpPrefixes() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	client, err := network.NewCustomIPPrefixesClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListAllPager(&network.CustomIPPrefixesClientListAllOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, prefix := range page.Value {
			if prefix == nil {
				continue
			}
			mqlPrefix, err := azureCustomIpPrefixToMql(a.MqlRuntime, prefix)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlPrefix)
		}
	}
	return res, nil
}

func azureCustomIpPrefixToMql(runtime *plugin.Runtime, prefix *network.CustomIPPrefix) (*mqlAzureSubscriptionNetworkServiceCustomIpPrefix, error) {
	var provisioningState, cidr, asn, commissionedState, prefixType, geo string
	var authorizationMessage, signedMessage, failedReason, resourceGuid string
	var expressRouteAdvertise, noInternetAdvertise bool
	if p := prefix.Properties; p != nil {
		if p.ProvisioningState != nil {
			provisioningState = string(*p.ProvisioningState)
		}
		cidr = convert.ToValue(p.Cidr)
		asn = convert.ToValue(p.Asn)
		if p.CommissionedState != nil {
			commissionedState = string(*p.CommissionedState)
		}
		if p.PrefixType != nil {
			prefixType = string(*p.PrefixType)
		}
		if p.Geo != nil {
			geo = string(*p.Geo)
		}
		authorizationMessage = convert.ToValue(p.AuthorizationMessage)
		signedMessage = convert.ToValue(p.SignedMessage)
		failedReason = convert.ToValue(p.FailedReason)
		resourceGuid = convert.ToValue(p.ResourceGUID)
		expressRouteAdvertise = convert.ToValue(p.ExpressRouteAdvertise)
		noInternetAdvertise = convert.ToValue(p.NoInternetAdvertise)
	}
	mqlPrefix, err := CreateResource(runtime, "azure.subscription.networkService.customIpPrefix",
		map[string]*llx.RawData{
			"id":                    llx.StringDataPtr(prefix.ID),
			"name":                  llx.StringDataPtr(prefix.Name),
			"location":              llx.StringDataPtr(prefix.Location),
			"tags":                  llx.MapData(convert.PtrMapStrToInterface(prefix.Tags), types.String),
			"type":                  llx.StringDataPtr(prefix.Type),
			"etag":                  llx.StringDataPtr(prefix.Etag),
			"provisioningState":     llx.StringData(provisioningState),
			"zones":                 llx.ArrayData(convert.SliceStrPtrToInterface(prefix.Zones), types.String),
			"cidr":                  llx.StringData(cidr),
			"asn":                   llx.StringData(asn),
			"commissionedState":     llx.StringData(commissionedState),
			"prefixType":            llx.StringData(prefixType),
			"geo":                   llx.StringData(geo),
			"expressRouteAdvertise": llx.BoolData(expressRouteAdvertise),
			"noInternetAdvertise":   llx.BoolData(noInternetAdvertise),
			"authorizationMessage":  llx.StringData(authorizationMessage),
			"signedMessage":         llx.StringData(signedMessage),
			"failedReason":          llx.StringData(failedReason),
			"resourceGuid":          llx.StringData(resourceGuid),
		})
	if err != nil {
		return nil, err
	}
	return mqlPrefix.(*mqlAzureSubscriptionNetworkServiceCustomIpPrefix), nil
}

func (a *mqlAzureSubscriptionNetworkServiceCustomIpPrefix) id() (string, error) {
	return a.Id.Data, nil
}

func initAzureSubscriptionNetworkServiceCustomIpPrefix(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}
	if args["id"] == nil {
		return args, nil, nil
	}
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	id, ok := args["id"].Value.(string)
	if !ok {
		return nil, nil, errors.New("id must be a non-nil string value")
	}
	azureId, err := ParseResourceID(id)
	if err != nil {
		return nil, nil, err
	}
	name, err := azureId.Component("customIpPrefixes")
	if err != nil {
		return nil, nil, err
	}
	client, err := network.NewCustomIPPrefixesClient(azureId.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.Get(context.Background(), azureId.ResourceGroup, name, nil)
	if err != nil {
		return nil, nil, err
	}
	mql, err := azureCustomIpPrefixToMql(runtime, &resp.CustomIPPrefix)
	if err != nil {
		return nil, nil, err
	}
	return args, mql, nil
}

// -----------------------------------------------------------------------------
// Virtual network taps
// -----------------------------------------------------------------------------

func (a *mqlAzureSubscriptionNetworkService) virtualNetworkTaps() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	client, err := network.NewVirtualNetworkTapsClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListAllPager(&network.VirtualNetworkTapsClientListAllOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, tap := range page.Value {
			if tap == nil {
				continue
			}
			var provisioningState, resourceGuid string
			var destinationPort int64
			if p := tap.Properties; p != nil {
				if p.ProvisioningState != nil {
					provisioningState = string(*p.ProvisioningState)
				}
				resourceGuid = convert.ToValue(p.ResourceGUID)
				if p.DestinationPort != nil {
					destinationPort = int64(*p.DestinationPort)
				}
			}
			mqlTap, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.virtualNetworkTap",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(tap.ID),
					"name":              llx.StringDataPtr(tap.Name),
					"location":          llx.StringDataPtr(tap.Location),
					"tags":              llx.MapData(convert.PtrMapStrToInterface(tap.Tags), types.String),
					"type":              llx.StringDataPtr(tap.Type),
					"etag":              llx.StringDataPtr(tap.Etag),
					"provisioningState": llx.StringData(provisioningState),
					"destinationPort":   llx.IntData(destinationPort),
					"resourceGuid":      llx.StringData(resourceGuid),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlTap)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualNetworkTap) id() (string, error) {
	return a.Id.Data, nil
}

// -----------------------------------------------------------------------------
// IP allocations
// -----------------------------------------------------------------------------

type mqlAzureSubscriptionNetworkServiceIpAllocationInternal struct {
	cacheSubnetId         *string
	cacheVirtualNetworkId *string
}

func (a *mqlAzureSubscriptionNetworkService) ipAllocations() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	client, err := network.NewIPAllocationsClient(a.SubscriptionId.Data, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []any{}
	pager := client.NewListPager(&network.IPAllocationsClientListOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, alloc := range page.Value {
			if alloc == nil {
				continue
			}
			var allocType, prefix, prefixType string
			var prefixLength int64
			allocationTags := map[string]*string{}
			var subnetId, vnetId *string
			if p := alloc.Properties; p != nil {
				if p.Type != nil {
					allocType = string(*p.Type)
				}
				prefix = convert.ToValue(p.Prefix)
				if p.PrefixType != nil {
					prefixType = string(*p.PrefixType)
				}
				if p.PrefixLength != nil {
					prefixLength = int64(*p.PrefixLength)
				}
				allocationTags = p.AllocationTags
				if p.Subnet != nil {
					subnetId = p.Subnet.ID
				}
				if p.VirtualNetwork != nil {
					vnetId = p.VirtualNetwork.ID
				}
			}
			mqlAlloc, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.ipAllocation",
				map[string]*llx.RawData{
					"id":               llx.StringDataPtr(alloc.ID),
					"name":             llx.StringDataPtr(alloc.Name),
					"location":         llx.StringDataPtr(alloc.Location),
					"tags":             llx.MapData(convert.PtrMapStrToInterface(alloc.Tags), types.String),
					"type":             llx.StringDataPtr(alloc.Type),
					"etag":             llx.StringDataPtr(alloc.Etag),
					"ipAllocationType": llx.StringData(allocType),
					"prefix":           llx.StringData(prefix),
					"prefixLength":     llx.IntData(prefixLength),
					"prefixType":       llx.StringData(prefixType),
					"allocationTags":   llx.MapData(convert.PtrMapStrToInterface(allocationTags), types.String),
				})
			if err != nil {
				return nil, err
			}
			mqlIpAllocation := mqlAlloc.(*mqlAzureSubscriptionNetworkServiceIpAllocation)
			mqlIpAllocation.cacheSubnetId = subnetId
			mqlIpAllocation.cacheVirtualNetworkId = vnetId
			res = append(res, mqlAlloc)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceIpAllocation) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceIpAllocation) subnet() (*mqlAzureSubscriptionNetworkServiceSubnet, error) {
	if a.cacheSubnetId == nil || *a.cacheSubnetId == "" {
		a.Subnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(a.MqlRuntime, "azure.subscription.networkService.subnet", map[string]*llx.RawData{
		"id": llx.StringDataPtr(a.cacheSubnetId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlAzureSubscriptionNetworkServiceSubnet), nil
}

func (a *mqlAzureSubscriptionNetworkServiceIpAllocation) virtualNetwork() (*mqlAzureSubscriptionNetworkServiceVirtualNetwork, error) {
	if a.cacheVirtualNetworkId == nil || *a.cacheVirtualNetworkId == "" {
		a.VirtualNetwork.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(a.MqlRuntime, "azure.subscription.networkService.virtualNetwork", map[string]*llx.RawData{
		"id": llx.StringDataPtr(a.cacheVirtualNetworkId),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlAzureSubscriptionNetworkServiceVirtualNetwork), nil
}
