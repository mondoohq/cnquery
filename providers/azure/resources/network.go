// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/azure/connection"
	"go.mondoo.com/cnquery/v11/types"
	"go.mondoo.com/cnquery/v11/utils/stringx"

	network "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
)

func (a *mqlAzureSubscriptionNetworkService) id() (string, error) {
	return "azure.subscription.network/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionNetworkService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (a *mqlAzureSubscriptionNetworkService) interfaces() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewInterfacesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(&network.InterfacesClientListAllOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, iface := range page.Value {
			if iface != nil {
				mqlAzure, err := azureInterfaceToMql(a.MqlRuntime, *iface)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlAzure)
			}
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkService) securityGroups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewSecurityGroupsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(&network.SecurityGroupsClientListAllOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, secgrp := range page.Value {
			if secgrp != nil {
				mqlAzure, err := azureSecGroupToMql(a.MqlRuntime, *secgrp)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlAzure)
			}
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkService) watchers() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewWatchersClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(&network.WatchersClientListAllOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, watcher := range page.Value {
			properties, err := convert.JsonToDict(watcher.Properties)
			if err != nil {
				return nil, err
			}

			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.watcher",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(watcher.ID),
					"name":              llx.StringDataPtr(watcher.Name),
					"location":          llx.StringDataPtr(watcher.Location),
					"tags":              llx.MapData(convert.PtrMapStrToInterface(watcher.Tags), types.String),
					"type":              llx.StringDataPtr(watcher.Type),
					"etag":              llx.StringDataPtr(watcher.Etag),
					"properties":        llx.DictData(properties),
					"provisioningState": llx.StringDataPtr((*string)(watcher.Properties.ProvisioningState)),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkService) publicIpAddresses() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewPublicIPAddressesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(&network.PublicIPAddressesClientListAllOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ip := range page.Value {
			if ip != nil {
				mqlAzure, err := azureIpToMql(a.MqlRuntime, *ip)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlAzure)
			}
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkService) bastionHosts() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewBastionHostsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(&network.BastionHostsClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, bh := range page.Value {
			properties, err := convert.JsonToDict(bh.Properties)
			if err != nil {
				return nil, err
			}
			sku, err := convert.JsonToDict(bh.SKU)
			if err != nil {
				return nil, err
			}
			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.bastionHost",
				map[string]*llx.RawData{
					"id":         llx.StringDataPtr(bh.ID),
					"name":       llx.StringDataPtr(bh.Name),
					"location":   llx.StringDataPtr(bh.Location),
					"tags":       llx.MapData(convert.PtrMapStrToInterface(bh.Tags), types.String),
					"type":       llx.StringDataPtr(bh.Type),
					"properties": llx.DictData(properties),
					"sku":        llx.DictData(sku),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceInterface) vm() (*mqlAzureSubscriptionComputeServiceVm, error) {
	return nil, errors.New("not implemented")
}

func (a *mqlAzureSubscriptionNetworkServiceWatcher) flowLogs() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	name := a.Name.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	subId := resourceID.SubscriptionID
	client, err := network.NewFlowLogsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := client.NewListPager(resourceID.ResourceGroup, name, &network.FlowLogsClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		type mqlRetentionPolicy struct {
			Enabled       bool `json:"enabled"`
			RetentionDays int  `json:"retentionDays"`
		}
		type mqlFlowLogAnalytics struct {
			Enabled             bool   `json:"allowedApplications"`
			AnalyticsInterval   int    `json:"analyticsInterval"`
			WorkspaceId         string `json:"workspaceResourceId"`
			WorkspaceResourceId string `json:"workspaceId"`
			WorkspaceRegion     string `json:"workspaceRegion"`
		}
		for _, flowLog := range page.Value {
			retentionPolicy := mqlRetentionPolicy{
				Enabled:       convert.ToBool(flowLog.Properties.RetentionPolicy.Enabled),
				RetentionDays: convert.ToIntFrom32(flowLog.Properties.RetentionPolicy.Days),
			}
			retentionPolicyDict, err := convert.JsonToDict(retentionPolicy)
			if err != nil {
				return nil, err
			}
			flowLogAnalytics := mqlFlowLogAnalytics{
				Enabled:             convert.ToBool(flowLog.Properties.FlowAnalyticsConfiguration.NetworkWatcherFlowAnalyticsConfiguration.Enabled),
				AnalyticsInterval:   convert.ToIntFrom32(flowLog.Properties.FlowAnalyticsConfiguration.NetworkWatcherFlowAnalyticsConfiguration.TrafficAnalyticsInterval),
				WorkspaceRegion:     convert.ToString(flowLog.Properties.FlowAnalyticsConfiguration.NetworkWatcherFlowAnalyticsConfiguration.WorkspaceRegion),
				WorkspaceResourceId: convert.ToString(flowLog.Properties.FlowAnalyticsConfiguration.NetworkWatcherFlowAnalyticsConfiguration.WorkspaceResourceID),
				WorkspaceId:         convert.ToString(flowLog.Properties.FlowAnalyticsConfiguration.NetworkWatcherFlowAnalyticsConfiguration.WorkspaceID),
			}
			flowLogAnalyticsDict, err := convert.JsonToDict(flowLogAnalytics)
			if err != nil {
				return nil, err
			}
			mqlFlowLog, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.watcher.flowlog",
				map[string]*llx.RawData{
					"id":                 llx.StringDataPtr(flowLog.ID),
					"name":               llx.StringDataPtr(flowLog.Name),
					"location":           llx.StringDataPtr(flowLog.Location),
					"tags":               llx.MapData(convert.PtrMapStrToInterface(flowLog.Tags), types.String),
					"type":               llx.StringDataPtr(flowLog.Type),
					"etag":               llx.StringDataPtr(flowLog.Etag),
					"retentionPolicy":    llx.DictData(retentionPolicyDict),
					"format":             llx.StringDataPtr((*string)(flowLog.Properties.Format.Type)),
					"version":            llx.IntDataDefault(flowLog.Properties.Format.Version, 0),
					"enabled":            llx.BoolDataPtr(flowLog.Properties.Enabled),
					"storageAccountId":   llx.StringDataPtr(flowLog.Properties.StorageID),
					"targetResourceId":   llx.StringDataPtr(flowLog.Properties.TargetResourceID),
					"targetResourceGuid": llx.StringDataPtr(flowLog.Properties.TargetResourceGUID),
					"provisioningState":  llx.StringDataPtr((*string)(flowLog.Properties.ProvisioningState)),
					"analytics":          llx.DictData(flowLogAnalyticsDict),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlFlowLog)
		}
	}

	return res, nil
}

func (a *mqlAzureSubscriptionNetworkService) loadBalancers() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewLoadBalancersClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(&network.LoadBalancersClientListAllOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, lb := range page.Value {
			probes := []interface{}{}
			backendPools := []interface{}{}
			frontendIConfigs := []interface{}{}
			inboundNatPools := []interface{}{}
			inboundNatRules := []interface{}{}
			outboundRules := []interface{}{}
			loadBalancerRules := []interface{}{}
			for _, p := range lb.Properties.Probes {
				props, err := convert.JsonToDict(p.Properties)
				if err != nil {
					return nil, err
				}
				mqlProbe, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.probe",
					map[string]*llx.RawData{
						"id":         llx.StringDataPtr(p.ID),
						"type":       llx.StringDataPtr(p.Type),
						"name":       llx.StringDataPtr(p.Name),
						"etag":       llx.StringDataPtr(p.Etag),
						"properties": llx.DictData(props),
					})
				if err != nil {
					return nil, err
				}
				probes = append(probes, mqlProbe)
			}
			for _, bap := range lb.Properties.BackendAddressPools {
				props, err := convert.JsonToDict(bap.Properties)
				if err != nil {
					return nil, err
				}
				mqlBap, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.backendAddressPool",
					map[string]*llx.RawData{
						"id":         llx.StringDataPtr(bap.ID),
						"type":       llx.StringDataPtr(bap.Type),
						"name":       llx.StringDataPtr(bap.Name),
						"etag":       llx.StringDataPtr(bap.Etag),
						"properties": llx.DictData(props),
					})
				if err != nil {
					return nil, err
				}
				backendPools = append(backendPools, mqlBap)
			}

			for _, ipConfig := range lb.Properties.FrontendIPConfigurations {
				props, err := convert.JsonToDict(ipConfig.Properties)
				if err != nil {
					return nil, err
				}
				mqlIpConfig, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.frontendIpConfig",
					map[string]*llx.RawData{
						"id":         llx.StringDataPtr(ipConfig.ID),
						"type":       llx.StringDataPtr(ipConfig.Type),
						"name":       llx.StringDataPtr(ipConfig.Name),
						"etag":       llx.StringDataPtr(ipConfig.Etag),
						"zones":      llx.ArrayData(convert.SliceStrPtrToInterface(ipConfig.Zones), types.String),
						"properties": llx.DictData(props),
					})
				if err != nil {
					return nil, err
				}
				frontendIConfigs = append(frontendIConfigs, mqlIpConfig)
			}

			for _, natPool := range lb.Properties.InboundNatPools {
				props, err := convert.JsonToDict(natPool.Properties)
				if err != nil {
					return nil, err
				}
				mqlNatPool, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.inboundNatPool",
					map[string]*llx.RawData{
						"id":         llx.StringDataPtr(natPool.ID),
						"type":       llx.StringDataPtr(natPool.Type),
						"name":       llx.StringDataPtr(natPool.Name),
						"etag":       llx.StringDataPtr(natPool.Etag),
						"properties": llx.DictData(props),
					})
				if err != nil {
					return nil, err
				}
				inboundNatPools = append(inboundNatPools, mqlNatPool)
			}

			for _, natRule := range lb.Properties.InboundNatRules {
				props, err := convert.JsonToDict(natRule.Properties)
				if err != nil {
					return nil, err
				}
				mqlNatRule, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.inboundNatRule",
					map[string]*llx.RawData{
						"id":         llx.StringDataPtr(natRule.ID),
						"type":       llx.StringDataPtr(natRule.Type),
						"name":       llx.StringDataPtr(natRule.Name),
						"etag":       llx.StringDataPtr(natRule.Etag),
						"properties": llx.DictData(props),
					})
				if err != nil {
					return nil, err
				}
				inboundNatRules = append(inboundNatRules, mqlNatRule)
			}

			for _, outboundRule := range lb.Properties.OutboundRules {
				props, err := convert.JsonToDict(outboundRule.Properties)
				if err != nil {
					return nil, err
				}
				mqlOutbound, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.outbundRule",
					map[string]*llx.RawData{
						"id":         llx.StringDataPtr(outboundRule.ID),
						"type":       llx.StringDataPtr(outboundRule.Type),
						"name":       llx.StringDataPtr(outboundRule.Name),
						"etag":       llx.StringDataPtr(outboundRule.Etag),
						"properties": llx.DictData(props),
					})
				if err != nil {
					return nil, err
				}
				outboundRules = append(outboundRules, mqlOutbound)
			}

			for _, lbRule := range lb.Properties.LoadBalancingRules {
				props, err := convert.JsonToDict(lbRule.Properties)
				if err != nil {
					return nil, err
				}
				mqlLbRule, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.loadBalancerRule",
					map[string]*llx.RawData{
						"id":         llx.StringDataPtr(lbRule.ID),
						"type":       llx.StringDataPtr(lbRule.Type),
						"name":       llx.StringDataPtr(lbRule.Name),
						"etag":       llx.StringDataPtr(lbRule.Etag),
						"properties": llx.DictData(props),
					})
				if err != nil {
					return nil, err
				}
				loadBalancerRules = append(loadBalancerRules, mqlLbRule)
			}

			lbProps, err := convert.JsonToDict(lb.Properties)
			if err != nil {
				return nil, err
			}
			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.loadBalancer",
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(lb.ID),
					"name":              llx.StringDataPtr(lb.Name),
					"location":          llx.StringDataPtr(lb.Location),
					"etag":              llx.StringDataPtr(lb.Etag),
					"sku":               llx.StringDataPtr((*string)(lb.SKU.Name)),
					"tags":              llx.MapData(convert.PtrMapStrToInterface(lb.Tags), types.String),
					"type":              llx.StringDataPtr(lb.Type),
					"probes":            llx.ArrayData(probes, types.ResourceLike),
					"backendPools":      llx.ArrayData(backendPools, types.ResourceLike),
					"frontendIpConfigs": llx.ArrayData(frontendIConfigs, types.ResourceLike),
					"inboundNatPools":   llx.ArrayData(inboundNatPools, types.ResourceLike),
					"inboundNatRules":   llx.ArrayData(inboundNatRules, types.ResourceLike),
					"outboundRules":     llx.ArrayData(outboundRules, types.ResourceLike),
					"loadBalancerRules": llx.ArrayData(loadBalancerRules, types.ResourceLike),
					"properties":        llx.DictData(lbProps),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkService) natGateways() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewNatGatewaysClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(&network.NatGatewaysClientListAllOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ng := range page.Value {
			if ng != nil {
				mqlNg, err := azureNatGatewayToMql(a.MqlRuntime, *ng)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlNg)
			}
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkService) firewalls() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	client, err := network.NewAzureFirewallsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(&network.AzureFirewallsClientListAllOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, fw := range page.Value {
			if fw != nil {
				mqlFw, err := azureFirewallToMql(a.MqlRuntime, *fw)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlFw)
			}
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewall) policy() (*mqlAzureSubscriptionNetworkServiceFirewallPolicy, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	props := a.Properties.Data
	propsDict := props.(map[string]interface{})
	fwp := propsDict["firewallPolicy"]
	if fwp == nil {
		return nil, errors.New("no firewall policy is associated with the ip configuration")
	}
	fwpDict := fwp.(map[string]interface{})
	id := fwpDict["id"]
	if id != nil {
		strId := id.(string)
		azureId, err := ParseResourceID(strId)
		if err != nil {
			return nil, err
		}
		client, err := network.NewFirewallPoliciesClient(azureId.SubscriptionID, token, &arm.ClientOptions{
			ClientOptions: conn.ClientOptions(),
		})
		if err != nil {
			return nil, err
		}
		policyName, err := azureId.Component("firewallPolicies")
		if err != nil {
			return nil, err
		}
		fwp, err := client.Get(ctx, azureId.ResourceGroup, policyName, &network.FirewallPoliciesClientGetOptions{})
		if err != nil {
			return nil, err
		}

		return azureFirewallPolicyToMql(a.MqlRuntime, fwp.FirewallPolicy)
	}
	return nil, errors.New("no firewall policy is associated with the ip configuration")
}

func (a *mqlAzureSubscriptionNetworkServiceFirewallIpConfig) publicIpAddress() (*mqlAzureSubscriptionNetworkServiceIpAddress, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	props := a.Properties.Data
	propsDict := props.(map[string]interface{})
	publicIpAddress := propsDict["publicIPAddress"]
	if publicIpAddress == nil {
		return nil, errors.New("no public ip address is associated with the ip configuration")
	}
	ipAddressDict := publicIpAddress.(map[string]interface{})
	id := ipAddressDict["id"]
	if id != nil {
		strId := id.(string)
		azureId, err := ParseResourceID(strId)
		if err != nil {
			return nil, err
		}
		client, err := network.NewPublicIPAddressesClient(azureId.SubscriptionID, token, &arm.ClientOptions{
			ClientOptions: conn.ClientOptions(),
		})
		if err != nil {
			return nil, err
		}
		ipAddressName, err := azureId.Component("publicIPAddresses")
		if err != nil {
			return nil, err
		}
		ipAddress, err := client.Get(ctx, azureId.ResourceGroup, ipAddressName, &network.PublicIPAddressesClientGetOptions{})
		if err != nil {
			return nil, err
		}

		return azureIpToMql(a.MqlRuntime, ipAddress.PublicIPAddress)
	}
	return nil, errors.New("no public ip address is associated with the ip configuration")
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualNetworkGatewayIpConfig) publicIpAddress() (*mqlAzureSubscriptionNetworkServiceIpAddress, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	props := a.Properties.Data
	propsDict := props.(map[string]interface{})
	publicIpAddress := propsDict["publicIPAddress"]
	if publicIpAddress == nil {
		return nil, errors.New("no public ip address is associated with the ip configuration")
	}
	ipAddressDict := publicIpAddress.(map[string]interface{})
	id := ipAddressDict["id"]
	if id != nil {
		strId := id.(string)
		azureId, err := ParseResourceID(strId)
		if err != nil {
			return nil, err
		}
		client, err := network.NewPublicIPAddressesClient(azureId.SubscriptionID, token, &arm.ClientOptions{
			ClientOptions: conn.ClientOptions(),
		})
		if err != nil {
			return nil, err
		}
		ipAddressName, err := azureId.Component("publicIPAddresses")
		if err != nil {
			return nil, err
		}
		ipAddress, err := client.Get(ctx, azureId.ResourceGroup, ipAddressName, &network.PublicIPAddressesClientGetOptions{})
		if err != nil {
			return nil, err
		}

		return azureIpToMql(a.MqlRuntime, ipAddress.PublicIPAddress)
	}
	return nil, errors.New("no public ip address is associated with the ip configuration")
}

func (a *mqlAzureSubscriptionNetworkServiceFirewallIpConfig) subnet() (*mqlAzureSubscriptionNetworkServiceSubnet, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	props := a.Properties.Data
	propsDict := props.(map[string]interface{})
	subnet := propsDict["subnet"]
	if subnet == nil {
		return nil, errors.New("no subnet is associated with the ip configuration")
	}
	subnetDict := subnet.(map[string]interface{})
	id := subnetDict["id"]
	if id != nil {
		strId := id.(string)
		azureId, err := ParseResourceID(strId)
		if err != nil {
			return nil, err
		}
		client, err := network.NewSubnetsClient(azureId.SubscriptionID, token, &arm.ClientOptions{
			ClientOptions: conn.ClientOptions(),
		})
		if err != nil {
			return nil, err
		}
		vnName, err := azureId.Component("virtualNetworks")
		if err != nil {
			return nil, err
		}
		subnetName, err := azureId.Component("subnets")
		if err != nil {
			return nil, err
		}
		subnet, err := client.Get(ctx, azureId.ResourceGroup, vnName, subnetName, &network.SubnetsClientGetOptions{})
		if err != nil {
			return nil, err
		}

		return azureSubnetToMql(a.MqlRuntime, subnet.Subnet)
	}
	return nil, errors.New("no subnet is associated with the ip configuration")
}

func (a *mqlAzureSubscriptionNetworkService) firewallPolicies() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	client, err := network.NewFirewallPoliciesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(&network.FirewallPoliciesClientListAllOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, fwp := range page.Value {
			if fwp != nil {
				mqlFw, err := azureFirewallPolicyToMql(a.MqlRuntime, *fwp)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlFw)
			}
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkService) virtualNetworks() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewVirtualNetworksClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(&network.VirtualNetworksClientListAllOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, vn := range page.Value {
			props, err := convert.JsonToDict(vn.Properties)
			if err != nil {
				return nil, err
			}
			subnets := []interface{}{}
			for _, s := range vn.Properties.Subnets {
				if s != nil {
					mqlSubnet, err := azureSubnetToMql(a.MqlRuntime, *s)
					if err != nil {
						return nil, err
					}
					subnets = append(subnets, mqlSubnet)
				}
			}
			args := map[string]*llx.RawData{
				"id":                   llx.StringDataPtr(vn.ID),
				"name":                 llx.StringDataPtr(vn.Name),
				"type":                 llx.StringDataPtr(vn.Type),
				"location":             llx.StringDataPtr(vn.Location),
				"tags":                 llx.MapData(convert.PtrMapStrToInterface(vn.Tags), types.String),
				"etag":                 llx.StringDataPtr(vn.Etag),
				"properties":           llx.DictData(props),
				"enableDdosProtection": llx.BoolDataPtr(vn.Properties.EnableDdosProtection),
				"enableVmProtection":   llx.BoolDataPtr(vn.Properties.EnableVMProtection),
				"subnets":              llx.ArrayData(subnets, types.ResourceLike),
			}
			if vn.Properties.DhcpOptions != nil {
				id := convert.ToString(vn.ID) + "/dhcpOptions"
				dhcpOpts, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.virtualNetwork.dhcpOptions",
					map[string]*llx.RawData{
						"id":         llx.StringData(id),
						"dnsServers": llx.ArrayData(convert.SliceStrPtrToInterface(vn.Properties.DhcpOptions.DNSServers), types.String),
					})
				if err != nil {
					return nil, err
				}
				args["dhcpOptions"] = llx.ResourceData(dhcpOpts, dhcpOpts.MqlName())
			} else {
				args["dhcpOptions"] = llx.NilData
			}

			mqlVn, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.virtualNetwork", args)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlVn)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkService) applicationSecurityGroups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewApplicationSecurityGroupsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(&network.ApplicationSecurityGroupsClientListAllOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, asg := range page.Value {
			props, err := convert.JsonToDict(asg.Properties)
			if err != nil {
				return nil, err
			}
			mqlAppSecGroup, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.appSecurityGroup",
				map[string]*llx.RawData{
					"id":         llx.StringDataPtr(asg.ID),
					"name":       llx.StringDataPtr(asg.Name),
					"type":       llx.StringDataPtr(asg.Type),
					"location":   llx.StringDataPtr(asg.Location),
					"tags":       llx.MapData(convert.PtrMapStrToInterface(asg.Tags), types.String),
					"etag":       llx.StringDataPtr(asg.Etag),
					"properties": llx.DictData(props),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAppSecGroup)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkService) virtualNetworkGateways() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewVirtualNetworkGatewaysClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	// the virtual network gateways API works on resource-group level. therefore, we need to fetch all RGs first
	sub, err := CreateResource(a.MqlRuntime, "azure.subscription", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(subId),
	})
	if err != nil {
		return nil, err
	}
	azureSub := sub.(*mqlAzureSubscription)
	rgs := azureSub.GetResourceGroups()
	if rgs.Error != nil {
		return nil, err
	}
	res := []interface{}{}
	for _, rg := range rgs.Data {
		mqlRg := rg.(*mqlAzureSubscriptionResourcegroup)
		pager := client.NewListPager(mqlRg.Name.Data, &network.VirtualNetworkGatewaysClientListOptions{})
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}
			for _, vng := range page.Value {
				props, err := convert.JsonToDict(vng.Properties)
				if err != nil {
					return nil, err
				}
				bgpPeeringAddresses := []interface{}{}
				bgpSettingsId := *vng.ID + "/bgpSettings"
				for i, bpa := range vng.Properties.BgpSettings.BgpPeeringAddresses {
					bpaId := fmt.Sprintf("%s/%s/%d", bgpSettingsId, "bgpPeeringAddresses", i)
					mqlBpa, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.bgpSettings.ipConfigurationBgpPeeringAddress",
						map[string]*llx.RawData{
							"id":                    llx.StringData(bpaId),
							"customBgpIpAddresses":  llx.ArrayData(convert.SliceStrPtrToInterface(bpa.CustomBgpIPAddresses), types.String),
							"defaultBgpIpAddresses": llx.ArrayData(convert.SliceStrPtrToInterface(bpa.DefaultBgpIPAddresses), types.String),
							"tunnelIpAddresses":     llx.ArrayData(convert.SliceStrPtrToInterface(bpa.TunnelIPAddresses), types.String),
							"ipConfigurationId":     llx.StringDataPtr(bpa.IPConfigurationID),
						})
					if err != nil {
						return nil, err
					}
					bgpPeeringAddresses = append(bgpPeeringAddresses, mqlBpa)
				}
				bgpSettings, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.bgpSettings",
					map[string]*llx.RawData{
						"id":                        llx.StringData(bgpSettingsId),
						"asn":                       llx.IntDataPtr(vng.Properties.BgpSettings.Asn),
						"bgpPeeringAddress":         llx.StringDataPtr(vng.Properties.BgpSettings.BgpPeeringAddress),
						"peerWeight":                llx.IntDataDefault(vng.Properties.BgpSettings.PeerWeight, 0),
						"bgpPeeringAddressesConfig": llx.ArrayData(bgpPeeringAddresses, types.ResourceLike),
					})
				if err != nil {
					return nil, err
				}

				ipConfigs := []interface{}{}
				natRules := []interface{}{}

				for _, nr := range vng.Properties.NatRules {
					props, err := convert.JsonToDict(nr.Properties)
					if err != nil {
						return nil, err
					}
					mqlNr, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.virtualNetworkGateway.natRule", map[string]*llx.RawData{
						"id":         llx.StringDataPtr(nr.ID),
						"name":       llx.StringDataPtr(nr.Name),
						"etag":       llx.StringDataPtr(nr.Etag),
						"properties": llx.DictData(props),
					})
					if err != nil {
						return nil, err
					}
					natRules = append(natRules, mqlNr)
				}
				for _, ipc := range vng.Properties.IPConfigurations {
					props, err := convert.JsonToDict(ipc.Properties)
					if err != nil {
						return nil, err
					}
					mqlIpc, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.virtualNetworkGateway.ipConfig", map[string]*llx.RawData{
						"id":               llx.StringDataPtr(ipc.ID),
						"name":             llx.StringDataPtr(ipc.Name),
						"etag":             llx.StringDataPtr(ipc.Etag),
						"properties":       llx.DictData(props),
						"privateIpAddress": llx.StringDataPtr(ipc.Properties.PrivateIPAddress),
					})
					if err != nil {
						return nil, err
					}
					ipConfigs = append(ipConfigs, mqlIpc)
				}
				args := map[string]*llx.RawData{
					"id":                              llx.StringDataPtr(vng.ID),
					"name":                            llx.StringDataPtr(vng.Name),
					"type":                            llx.StringDataPtr(vng.Type),
					"location":                        llx.StringDataPtr(vng.Location),
					"tags":                            llx.MapData(convert.PtrMapStrToInterface(vng.Tags), types.String),
					"etag":                            llx.StringDataPtr(vng.Etag),
					"active":                          llx.BoolDataPtr(vng.Properties.Active),
					"enableBgp":                       llx.BoolDataPtr(vng.Properties.EnableBgp),
					"enableBgpRouteTranslationForNat": llx.BoolDataPtr(vng.Properties.EnableBgpRouteTranslationForNat),
					"enableDNSForwarding":             llx.BoolDataPtr(vng.Properties.EnableDNSForwarding),
					"enablePrivateIPAddress":          llx.BoolDataPtr(vng.Properties.EnablePrivateIPAddress),
					"disableIPSecReplayProtection":    llx.BoolDataPtr(vng.Properties.DisableIPSecReplayProtection),
					"inboundDNSForwardingEndpoint":    llx.StringDataPtr(vng.Properties.InboundDNSForwardingEndpoint),
					"skuName":                         llx.StringDataPtr((*string)(vng.Properties.SKU.Name)),
					"skuCapacity":                     llx.IntDataDefault(vng.Properties.SKU.Capacity, 0),
					"provisioningState":               llx.StringDataPtr((*string)(vng.Properties.ProvisioningState)),
					"properties":                      llx.DictData(props),
					"vpnType":                         llx.StringDataPtr((*string)(vng.Properties.VPNType)),
					"vpnGatewayGeneration":            llx.StringDataPtr((*string)(vng.Properties.VPNGatewayGeneration)),
					"bgpSettings":                     llx.ResourceData(bgpSettings, "bgpSettings"),
					"ipConfigurations":                llx.ArrayData(ipConfigs, types.ResourceLike),
					"gatewayType":                     llx.StringDataPtr((*string)(vng.Properties.GatewayType)),
					"natRules":                        llx.ArrayData(natRules, types.ResourceLike),
				}
				if vng.Properties.CustomRoutes != nil {
					args["addressPrefixes"] = llx.ArrayData(convert.SliceStrPtrToInterface(vng.Properties.CustomRoutes.AddressPrefixes), types.String)
				} else {
					args["addressPrefixes"] = llx.ArrayData([]interface{}{}, types.String)
				}
				if vng.Properties.VPNClientConfiguration != nil {
					vpnClientDict, err := convert.JsonToDict(vng.Properties.VPNClientConfiguration)
					if err != nil {
						return nil, err
					}
					args["vpnClientConfiguration"] = llx.DictData(vpnClientDict)
				} else {
					args["vpnClientConfiguration"] = llx.NilData
				}
				mqlVn, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.virtualNetworkGateway", args)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlVn)
			}
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkService) applicationGateways() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewApplicationGatewaysClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	pager := client.NewListAllPager(&network.ApplicationGatewaysClientListAllOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, ag := range page.Value {
			if ag != nil {
				mqlAg, err := azureAppGatewayToMql(a.MqlRuntime, *ag)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlAg)
			}
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkService) applicationFirewallPolicies() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewWebApplicationFirewallPoliciesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	pager := client.NewListAllPager(&network.WebApplicationFirewallPoliciesClientListAllOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, waf := range page.Value {
			if waf != nil {
				mqlWaf, err := azureAppFirewallPolicyToMql(a.MqlRuntime, *waf)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlWaf)
			}
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceApplicationGateway) policy() (*mqlAzureSubscriptionNetworkServiceApplicationFirewallPolicy, error) {
	props := a.Properties
	if props.Error != nil {
		return nil, props.Error
	}
	propsDict := props.Data.(map[string]interface{})
	fwDict := propsDict["firewallPolicy"]
	if fwDict == nil {
		return nil, errors.New("no firewall policy is associated with the application gateway")
	}
	fwId := fwDict.(map[string]interface{})["id"]
	if fwId == nil {
		return nil, errors.New("no firewall policy is associated with the application gateway")
	}
	strId := fwId.(string)
	azureId, err := ParseResourceID(strId)
	if err != nil {
		return nil, err
	}
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	client, err := network.NewWebApplicationFirewallPoliciesClient(azureId.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	policyName, err := azureId.Component("ApplicationGatewayWebApplicationFirewallPolicies")
	if err != nil {
		return nil, err
	}
	policy, err := client.Get(ctx, azureId.ResourceGroup, policyName, &network.WebApplicationFirewallPoliciesClientGetOptions{})
	if err != nil {
		return nil, err
	}
	return azureAppFirewallPolicyToMql(a.MqlRuntime, policy.WebApplicationFirewallPolicy)
}

func (a *mqlAzureSubscriptionNetworkServiceApplicationFirewallPolicy) gateways() ([]interface{}, error) {
	props := a.Properties
	if props.Error != nil {
		return nil, props.Error
	}
	propsDict := props.Data.(map[string]interface{})
	gateways := propsDict["applicationGateways"]
	if gateways == nil {
		return nil, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	client, err := network.NewApplicationGatewaysClient(conn.SubId(), token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	gatewaysList := gateways.([]interface{})
	res := []interface{}{}
	for _, g := range gatewaysList {
		id := g.(map[string]interface{})["id"]
		strId := id.(string)
		azureId, err := ParseResourceID(strId)
		if err != nil {
			return nil, err
		}
		gatewayName, err := azureId.Component("applicationGateways")
		if err != nil {
			return nil, err
		}
		gateway, err := client.Get(ctx, azureId.ResourceGroup, gatewayName, &network.ApplicationGatewaysClientGetOptions{})
		if err != nil {
			return nil, err
		}
		mqlGateway, err := azureAppGatewayToMql(a.MqlRuntime, gateway.ApplicationGateway)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlGateway)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceNatGateway) publicIpAddresses() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	azureId, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	props := a.Properties.Data
	propsDict := props.(map[string]interface{})
	publicIpAddresses := propsDict["publicIpAddresses"]
	// if we have no present public ip addresses ids, we can just return nil
	if publicIpAddresses == nil {
		return nil, nil
	}

	res := []interface{}{}
	client, err := network.NewPublicIPAddressesClient(azureId.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	for _, p := range publicIpAddresses.([]interface{}) {
		pDict := p.(map[string]interface{})
		pId := pDict["id"].(string)
		resourceID, err := ParseResourceID(pId)
		if err != nil {
			return nil, err
		}
		publicIpName, err := resourceID.Component("publicIPAddresses")
		if err != nil {
			return nil, err
		}
		publicIp, err := client.Get(ctx, resourceID.ResourceGroup, publicIpName, &network.PublicIPAddressesClientGetOptions{})
		if err != nil {
			return nil, err
		}
		mqlPublicIp, err := azureIpToMql(a.MqlRuntime, publicIp.PublicIPAddress)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlPublicIp)
	}

	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualNetworkGateway) connections() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	azureId, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	client, err := network.NewVirtualNetworkGatewayConnectionsClient(azureId.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := client.NewListPager(azureId.ResourceGroup, &network.VirtualNetworkGatewayConnectionsClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, c := range page.Value {
			// the API does not let us get connections, applicable to a given gateway.
			// Therefore we filter them manually here.
			filter := []string{}
			// primary gateway
			if c.Properties.VirtualNetworkGateway1 != nil {
				filter = append(filter, *c.Properties.VirtualNetworkGateway1.ID)
			}
			// secondary, optional (only if Vnet2Vnet connection)
			if c.Properties.VirtualNetworkGateway2 != nil {
				filter = append(filter, *c.Properties.VirtualNetworkGateway2.ID)
			}
			if !stringx.Contains(filter, id) {
				continue
			}
			props, err := convert.JsonToDict(c.Properties)
			if err != nil {
				return nil, err
			}
			mqlConnection, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.virtualNetworkGateway.connection",
				map[string]*llx.RawData{
					"id":         llx.StringDataPtr(c.ID),
					"type":       llx.StringDataPtr(c.Type),
					"name":       llx.StringDataPtr(c.Name),
					"etag":       llx.StringDataPtr(c.Etag),
					"properties": llx.DictData(props),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlConnection)

		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceNatGateway) subnets() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	azureId, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	props := a.Properties.Data
	propsDict := props.(map[string]interface{})
	subnets := propsDict["subnets"]
	// if we have no present subnets in the dict, we can just return nil
	if subnets == nil {
		return nil, nil
	}
	res := []interface{}{}
	client, err := network.NewSubnetsClient(azureId.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	for _, s := range subnets.([]interface{}) {
		sDict := s.(map[string]interface{})
		sId := sDict["id"].(string)
		resourceID, err := ParseResourceID(sId)
		if err != nil {
			return nil, err
		}
		virtualNetworkName, err := resourceID.Component("virtualNetworks")
		if err != nil {
			return nil, err
		}
		subnetName, err := resourceID.Component("subnets")
		if err != nil {
			return nil, err
		}
		subnet, err := client.Get(ctx, resourceID.ResourceGroup, virtualNetworkName, subnetName, &network.SubnetsClientGetOptions{})
		if err != nil {
			return nil, err
		}
		mqlSubnet, err := azureSubnetToMql(a.MqlRuntime, subnet.Subnet)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSubnet)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceSubnet) natGateway() (*mqlAzureSubscriptionNetworkServiceNatGateway, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	azureId, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	props := a.Properties.Data
	propsDict := props.(map[string]interface{})
	natGatewayDict := propsDict["natGateway"]
	if natGatewayDict == nil {
		// TODO: Preslav: how do we define a 'nil' resource here? if i return nil, it panics
		return nil, errors.New("subnet has no NAT gateway associated with it")
	}
	natGatewayFields := natGatewayDict.(map[string]interface{})
	natGatewayId := natGatewayFields["id"].(string)
	resourceID, err := ParseResourceID(natGatewayId)
	if err != nil {
		return nil, err
	}
	natGatewayName, err := resourceID.Component("natGateways")
	if err != nil {
		return nil, err
	}
	client, err := network.NewNatGatewaysClient(azureId.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	natGatewayRes, err := client.Get(ctx, resourceID.ResourceGroup, natGatewayName, &network.NatGatewaysClientGetOptions{})
	if err != nil {
		return nil, err
	}
	mqlNatGateway, err := azureNatGatewayToMql(a.MqlRuntime, natGatewayRes.NatGateway)
	if err != nil {
		return nil, err
	}
	return mqlNatGateway, nil
}

func (a *mqlAzureSubscriptionNetworkServiceSubnet) ipConfigurations() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	subId := conn.SubId()
	props := a.Properties.Data
	propsDict := props.(map[string]interface{})
	ipConfigsDict := propsDict["ipConfigurations"]
	if ipConfigsDict == nil {
		return nil, nil
	}
	res := []interface{}{}
	ipConfigIds := []string{}
	ipConfigsList := ipConfigsDict.([]interface{})
	for _, ipc := range ipConfigsList {
		ipcDict := ipc.(map[string]interface{})
		ipcId := ipcDict["id"].(string)
		ipConfigIds = append(ipConfigIds, strings.ToLower(ipcId))
	}

	network, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(subId),
	})
	if err != nil {
		return nil, err
	}
	mqlNetwork := network.(*mqlAzureSubscriptionNetworkService)
	// the subnet ip configs are referencing the virtual network gateways ip config. There seems to be no
	// no API to fetch this so we fetch the gateaways and iterate through them
	gateways := mqlNetwork.GetVirtualNetworkGateways()
	if gateways.Error != nil {
		return nil, err
	}
	for _, gw := range gateways.Data {
		mqlGw := gw.(*mqlAzureSubscriptionNetworkServiceVirtualNetworkGateway)
		// we need to check if the gateway has the ip configuration
		for _, ipc := range mqlGw.IpConfigurations.Data {
			mqlIpc := ipc.(*mqlAzureSubscriptionNetworkServiceVirtualNetworkGatewayIpConfig)
			// Note: for some reason, the azure API returns the resource id capitalized, e.g.
			// .../ipConfigurations/MY-IP-CONFIGURATION whereas those are all lower case in the virtual network gateways
			// object. To make this work, we make sure everything's lower case
			if stringx.Contains(ipConfigIds, strings.ToLower(mqlIpc.Id.Data)) {
				res = append(res, mqlIpc)
			}
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewallPolicy) basePolicy() (*mqlAzureSubscriptionNetworkServiceFirewallPolicy, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	props := a.Properties.Data
	propsDict := props.(map[string]interface{})
	basePolicy := propsDict["basePolicy"]
	if basePolicy == nil {
		// TODO: find a way to return nil instead of err here, nil currently panics
		return nil, errors.New("no base policy found")
	}
	basePolicyDict := basePolicy.(map[string]interface{})
	basePolicyId := basePolicyDict["id"].(string)
	resourceID, err := ParseResourceID(basePolicyId)
	if err != nil {
		return nil, err
	}
	client, err := network.NewFirewallPoliciesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	basePolicyName, err := resourceID.Component("firewallPolicies")
	if err != nil {
		return nil, err
	}
	basePolicyRes, err := client.Get(ctx, resourceID.ResourceGroup, basePolicyName, &network.FirewallPoliciesClientGetOptions{})
	if err != nil {
		return nil, err
	}
	return azureFirewallPolicyToMql(a.MqlRuntime, basePolicyRes.FirewallPolicy)
}

func (a *mqlAzureSubscriptionNetworkServiceFirewallPolicy) childPolicies() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	props := a.Properties.Data
	propsDict := props.(map[string]interface{})
	childPolicies := propsDict["childPolicies"]
	if childPolicies == nil {
		return nil, nil
	}
	baseResourceId, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}

	client, err := network.NewFirewallPoliciesClient(baseResourceId.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	childPoliciesList := childPolicies.([]interface{})
	for _, cp := range childPoliciesList {
		cpDict := cp.(map[string]interface{})
		cpId := cpDict["id"].(string)
		resourceID, err := ParseResourceID(cpId)
		if err != nil {
			return nil, err
		}
		polName, err := resourceID.Component("firewallPolicies")
		if err != nil {
			return nil, err
		}
		polRes, err := client.Get(ctx, resourceID.ResourceGroup, polName, &network.FirewallPoliciesClientGetOptions{})
		if err != nil {
			return nil, err
		}
		mqlFw, err := azureFirewallPolicyToMql(a.MqlRuntime, polRes.FirewallPolicy)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlFw)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewallPolicy) firewalls() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	props := a.Properties.Data
	propsDict := props.(map[string]interface{})
	firewalls := propsDict["firewalls"]
	if firewalls == nil {
		return nil, nil
	}
	baseResourceId, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}

	client, err := network.NewAzureFirewallsClient(baseResourceId.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	firewallsList := firewalls.([]interface{})
	for _, fw := range firewallsList {
		fwDict := fw.(map[string]interface{})
		fwId := fwDict["id"].(string)
		resourceID, err := ParseResourceID(fwId)
		if err != nil {
			return nil, err
		}
		firewallName, err := resourceID.Component("azureFirewalls")
		if err != nil {
			return nil, err
		}
		fwRes, err := client.Get(ctx, resourceID.ResourceGroup, firewallName, &network.AzureFirewallsClientGetOptions{})
		if err != nil {
			return nil, err
		}
		mqlFw, err := azureFirewallToMql(a.MqlRuntime, fwRes.AzureFirewall)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlFw)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceInterface) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceIpAddress) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceBastionHost) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceSecurityGroup) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceWatcher) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceWatcherFlowlog) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceSecurityrule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceLoadBalancer) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceBackendAddressPool) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceFrontendIpConfig) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceInboundNatPool) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceInboundNatRule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceProbe) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceNatGateway) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceSubnet) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualNetwork) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualNetworkGateway) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceBgpSettings) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceBgpSettingsIpConfigurationBgpPeeringAddress) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualNetworkGatewayIpConfig) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualNetworkGatewayConnection) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewall) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewallPolicy) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewallApplicationRule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewallNetworkRule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewallNatRule) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewallIpConfig) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceAppSecurityGroup) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualNetworkDhcpOptions) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceApplicationGateway) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceApplicationFirewallPolicy) id() (string, error) {
	return a.Id.Data, nil
}

func azureAppFirewallPolicyToMql(runtime *plugin.Runtime, waf network.WebApplicationFirewallPolicy) (*mqlAzureSubscriptionNetworkServiceApplicationFirewallPolicy, error) {
	props, err := convert.JsonToDict(waf.Properties)
	if err != nil {
		return nil, err
	}
	args := map[string]*llx.RawData{
		"id":         llx.StringDataPtr(waf.ID),
		"name":       llx.StringDataPtr(waf.Name),
		"type":       llx.StringDataPtr(waf.Type),
		"location":   llx.StringDataPtr(waf.Location),
		"tags":       llx.MapData(convert.PtrMapStrToInterface(waf.Tags), types.String),
		"etag":       llx.StringDataPtr(waf.Etag),
		"properties": llx.DictData(props),
	}

	mqlWaf, err := CreateResource(runtime, "azure.subscription.networkService.applicationFirewallPolicy", args)
	if err != nil {
		return nil, err
	}

	return mqlWaf.(*mqlAzureSubscriptionNetworkServiceApplicationFirewallPolicy), nil
}

func azureAppGatewayToMql(runtime *plugin.Runtime, ag network.ApplicationGateway) (*mqlAzureSubscriptionNetworkServiceApplicationGateway, error) {
	props, err := convert.JsonToDict(ag.Properties)
	if err != nil {
		return nil, err
	}
	args := map[string]*llx.RawData{
		"id":         llx.StringDataPtr(ag.ID),
		"name":       llx.StringDataPtr(ag.Name),
		"type":       llx.StringDataPtr(ag.Type),
		"location":   llx.StringDataPtr(ag.Location),
		"tags":       llx.MapData(convert.PtrMapStrToInterface(ag.Tags), types.String),
		"etag":       llx.StringDataPtr(ag.Etag),
		"properties": llx.DictData(props),
	}

	mqlAg, err := CreateResource(runtime, "azure.subscription.networkService.applicationGateway", args)
	if err != nil {
		return nil, err
	}

	return mqlAg.(*mqlAzureSubscriptionNetworkServiceApplicationGateway), nil
}

func azureFirewallToMql(runtime *plugin.Runtime, fw network.AzureFirewall) (*mqlAzureSubscriptionNetworkServiceFirewall, error) {
	applicationRules := []interface{}{}
	natRules := []interface{}{}
	networkRules := []interface{}{}
	ipConfigs := []interface{}{}
	props, err := convert.JsonToDict(fw.Properties)
	if err != nil {
		return nil, err
	}
	for _, ipConfig := range fw.Properties.IPConfigurations {
		props, err := convert.JsonToDict(ipConfig.Properties)
		if err != nil {
			return nil, err
		}
		mqlIpConfig, err := CreateResource(runtime, "azure.subscription.networkService.firewall.ipConfig",
			map[string]*llx.RawData{
				"id":               llx.StringDataPtr(ipConfig.ID),
				"name":             llx.StringDataPtr(ipConfig.Name),
				"etag":             llx.StringDataPtr(ipConfig.Etag),
				"privateIpAddress": llx.StringDataPtr(ipConfig.Properties.PrivateIPAddress),
				"properties":       llx.DictData(props),
			})
		if err != nil {
			return nil, err
		}
		ipConfigs = append(ipConfigs, mqlIpConfig)
	}
	for _, natRule := range fw.Properties.NatRuleCollections {
		props, err := convert.JsonToDict(natRule.Properties)
		if err != nil {
			return nil, err
		}
		mqlNatRule, err := CreateResource(runtime, "azure.subscription.networkService.firewall.natRule",
			map[string]*llx.RawData{
				"id":         llx.StringDataPtr(natRule.ID),
				"name":       llx.StringDataPtr(natRule.Name),
				"etag":       llx.StringDataPtr(natRule.Etag),
				"properties": llx.DictData(props),
			})
		if err != nil {
			return nil, err
		}
		natRules = append(natRules, mqlNatRule)
	}
	for _, networkRule := range fw.Properties.NetworkRuleCollections {
		props, err := convert.JsonToDict(networkRule.Properties)
		if err != nil {
			return nil, err
		}
		mqlNetworkRule, err := CreateResource(runtime, "azure.subscription.networkService.firewall.networkRule",
			map[string]*llx.RawData{
				"id":         llx.StringDataPtr(networkRule.ID),
				"name":       llx.StringDataPtr(networkRule.Name),
				"etag":       llx.StringDataPtr(networkRule.Etag),
				"properties": llx.DictData(props),
			})
		if err != nil {
			return nil, err
		}
		networkRules = append(networkRules, mqlNetworkRule)
	}
	for _, appRule := range fw.Properties.ApplicationRuleCollections {
		props, err := convert.JsonToDict(appRule.Properties)
		if err != nil {
			return nil, err
		}
		mqlAppRule, err := CreateResource(runtime, "azure.subscription.networkService.firewall.applicationRule",
			map[string]*llx.RawData{
				"id":         llx.StringDataPtr(appRule.ID),
				"name":       llx.StringDataPtr(appRule.Name),
				"etag":       llx.StringDataPtr(appRule.Etag),
				"properties": llx.DictData(props),
			})
		if err != nil {
			return nil, err
		}
		applicationRules = append(applicationRules, mqlAppRule)
	}
	args := map[string]*llx.RawData{
		"id":                llx.StringDataPtr(fw.ID),
		"name":              llx.StringDataPtr(fw.Name),
		"type":              llx.StringDataPtr(fw.Type),
		"location":          llx.StringDataPtr(fw.Location),
		"tags":              llx.MapData(convert.PtrMapStrToInterface(fw.Tags), types.String),
		"etag":              llx.StringDataPtr(fw.Etag),
		"properties":        llx.DictData(props),
		"skuTier":           llx.StringDataPtr((*string)(fw.Properties.SKU.Tier)),
		"skuName":           llx.StringDataPtr((*string)(fw.Properties.SKU.Name)),
		"provisioningState": llx.StringDataPtr((*string)(fw.Properties.ProvisioningState)),
		"threatIntelMode":   llx.StringDataPtr((*string)(fw.Properties.ThreatIntelMode)),
		"natRules":          llx.ArrayData(natRules, types.ResourceLike),
		"applicationRules":  llx.ArrayData(applicationRules, types.ResourceLike),
		"networkRules":      llx.ArrayData(networkRules, types.ResourceLike),
		"ipConfigurations":  llx.ArrayData(ipConfigs, types.ResourceLike),
	}
	if fw.Properties.ManagementIPConfiguration != nil {
		ipConfig := fw.Properties.ManagementIPConfiguration
		props, err := convert.JsonToDict(ipConfig.Properties)
		if err != nil {
			return nil, err
		}
		mqlIpConfig, err := CreateResource(runtime, "azure.subscription.networkService.firewall.ipConfig",
			map[string]*llx.RawData{
				"id":               llx.StringDataPtr(ipConfig.ID),
				"name":             llx.StringDataPtr(ipConfig.Name),
				"etag":             llx.StringDataPtr(ipConfig.Etag),
				"privateIpAddress": llx.StringDataPtr(ipConfig.Properties.PrivateIPAddress),
				"properties":       llx.DictData(props),
			})
		if err != nil {
			return nil, err
		}
		args["managementIpConfiguration"] = llx.ResourceData(mqlIpConfig, "managementIpConfiguration")
	} else {
		args["managementIpConfiguration"] = llx.NilData
	}
	mqlFw, err := CreateResource(runtime, "azure.subscription.networkService.firewall", args)
	if err != nil {
		return nil, err
	}
	return mqlFw.(*mqlAzureSubscriptionNetworkServiceFirewall), nil
}

func azureFirewallPolicyToMql(runtime *plugin.Runtime, fwp network.FirewallPolicy) (*mqlAzureSubscriptionNetworkServiceFirewallPolicy, error) {
	props, err := convert.JsonToDict(fwp.Properties)
	if err != nil {
		return nil, err
	}
	mqlFw, err := CreateResource(runtime, "azure.subscription.networkService.firewallPolicy",
		map[string]*llx.RawData{
			"id":                llx.StringDataPtr(fwp.ID),
			"name":              llx.StringDataPtr(fwp.Name),
			"type":              llx.StringDataPtr(fwp.Type),
			"location":          llx.StringDataPtr(fwp.Location),
			"tags":              llx.MapData(convert.PtrMapStrToInterface(fwp.Tags), types.String),
			"etag":              llx.StringDataPtr(fwp.Etag),
			"properties":        llx.DictData(props),
			"provisioningState": llx.StringDataPtr((*string)(fwp.Properties.ProvisioningState)),
		})
	if err != nil {
		return nil, err
	}

	return mqlFw.(*mqlAzureSubscriptionNetworkServiceFirewallPolicy), nil
}

func azureIpToMql(runtime *plugin.Runtime, ip network.PublicIPAddress) (*mqlAzureSubscriptionNetworkServiceIpAddress, error) {
	mqlAzure, err := CreateResource(runtime, "azure.subscription.networkService.ipAddress",
		map[string]*llx.RawData{
			"id":        llx.StringDataPtr(ip.ID),
			"name":      llx.StringDataPtr(ip.Name),
			"location":  llx.StringDataPtr(ip.Location),
			"tags":      llx.MapData(convert.PtrMapStrToInterface(ip.Tags), types.String),
			"type":      llx.StringDataPtr(ip.Type),
			"ipAddress": llx.StringDataPtr(ip.Properties.IPAddress),
		})
	if err != nil {
		return nil, err
	}
	return mqlAzure.(*mqlAzureSubscriptionNetworkServiceIpAddress), nil
}

func azureNatGatewayToMql(runtime *plugin.Runtime, ng network.NatGateway) (*mqlAzureSubscriptionNetworkServiceNatGateway, error) {
	props, err := convert.JsonToDict(ng.Properties)
	if err != nil {
		return nil, err
	}
	mqlNg, err := CreateResource(runtime, "azure.subscription.networkService.natGateway",
		map[string]*llx.RawData{
			"id":         llx.StringDataPtr(ng.ID),
			"name":       llx.StringDataPtr(ng.Name),
			"type":       llx.StringDataPtr(ng.Type),
			"location":   llx.StringDataPtr(ng.Location),
			"tags":       llx.MapData(convert.PtrMapStrToInterface(ng.Tags), types.String),
			"etag":       llx.StringDataPtr(ng.Etag),
			"zones":      llx.ArrayData(convert.SliceStrPtrToInterface(ng.Zones), types.String),
			"properties": llx.DictData(props),
		})
	if err != nil {
		return nil, err
	}
	return mqlNg.(*mqlAzureSubscriptionNetworkServiceNatGateway), nil
}

func azureSubnetToMql(runtime *plugin.Runtime, subnet network.Subnet) (*mqlAzureSubscriptionNetworkServiceSubnet, error) {
	props, err := convert.JsonToDict(subnet.Properties)
	if err != nil {
		return nil, err
	}

	mqlAzure, err := CreateResource(runtime, "azure.subscription.networkService.subnet",
		map[string]*llx.RawData{
			"id":            llx.StringDataPtr(subnet.ID),
			"name":          llx.StringDataPtr(subnet.Name),
			"type":          llx.StringDataPtr(subnet.Type),
			"etag":          llx.StringDataPtr(subnet.Etag),
			"addressPrefix": llx.StringDataPtr(subnet.Properties.AddressPrefix),
			"properties":    llx.DictData(props),
		})
	if err != nil {
		return nil, err
	}
	return mqlAzure.(*mqlAzureSubscriptionNetworkServiceSubnet), nil
}

func azureInterfaceToMql(runtime *plugin.Runtime, iface network.Interface) (*mqlAzureSubscriptionNetworkServiceInterface, error) {
	properties, err := convert.JsonToDict(iface.Properties)
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(runtime, "azure.subscription.networkService.interface",
		map[string]*llx.RawData{
			"id":         llx.StringDataPtr(iface.ID),
			"name":       llx.StringDataPtr(iface.Name),
			"location":   llx.StringDataPtr(iface.Location),
			"tags":       llx.MapData(convert.PtrMapStrToInterface(iface.Tags), types.String),
			"type":       llx.StringDataPtr(iface.Type),
			"etag":       llx.StringDataPtr(iface.Etag),
			"properties": llx.DictData(properties),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionNetworkServiceInterface), nil
}

// see https://github.com/Azure/azure-sdk-for-go/issues/8224
type AzureSecurityGroupPropertiesFormat network.SecurityGroupPropertiesFormat

func azureSecGroupToMql(runtime *plugin.Runtime, secGroup network.SecurityGroup) (*mqlAzureSubscriptionNetworkServiceSecurityGroup, error) {
	var properties map[string]interface{}
	ifaces := []interface{}{}
	securityRules := []interface{}{}
	defaultSecurityRules := []interface{}{}
	var err error
	if secGroup.Properties != nil {
		// avoid using the azure sdk SecurityGroupPropertiesFormat MarshalJSON
		var j AzureSecurityGroupPropertiesFormat
		j = AzureSecurityGroupPropertiesFormat(*secGroup.Properties)

		properties, err = convert.JsonToDict(j)
		if err != nil {
			return nil, err
		}

		if secGroup.Properties.NetworkInterfaces != nil {
			list := secGroup.Properties.NetworkInterfaces
			for _, iface := range list {
				if iface != nil {
					mqlAzure, err := azureInterfaceToMql(runtime, *iface)
					if err != nil {
						return nil, err
					}
					ifaces = append(ifaces, mqlAzure)
				}
			}
		}

		if secGroup.Properties.SecurityRules != nil {
			list := secGroup.Properties.SecurityRules
			for _, secRule := range list {
				if secRule != nil {
					mqlAzure, err := azureSecurityRuleToMql(runtime, *secRule)
					if err != nil {
						return nil, err
					}
					securityRules = append(securityRules, mqlAzure)
				}
			}
		}

		if secGroup.Properties.DefaultSecurityRules != nil {
			list := secGroup.Properties.DefaultSecurityRules
			for _, secRule := range list {
				if secRule != nil {
					mqlAzure, err := azureSecurityRuleToMql(runtime, *secRule)
					if err != nil {
						return nil, err
					}

					defaultSecurityRules = append(defaultSecurityRules, mqlAzure)
				}
			}
		}
	}
	res, err := CreateResource(runtime, "azure.subscription.networkService.securityGroup",
		map[string]*llx.RawData{
			"id":                   llx.StringDataPtr(secGroup.ID),
			"name":                 llx.StringDataPtr(secGroup.Name),
			"location":             llx.StringDataPtr(secGroup.Location),
			"tags":                 llx.MapData(convert.PtrMapStrToInterface(secGroup.Tags), types.String),
			"type":                 llx.StringDataPtr(secGroup.Type),
			"etag":                 llx.StringDataPtr(secGroup.Etag),
			"properties":           llx.DictData(properties),
			"interfaces":           llx.ArrayData(ifaces, types.ResourceLike),
			"securityRules":        llx.ArrayData(securityRules, types.ResourceLike),
			"defaultSecurityRules": llx.ArrayData(defaultSecurityRules, types.ResourceLike),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionNetworkServiceSecurityGroup), nil
}

func azureSecurityRuleToMql(runtime *plugin.Runtime, secRule network.SecurityRule) (*mqlAzureSubscriptionNetworkServiceSecurityrule, error) {
	properties, err := convert.JsonToDict(secRule.Properties)
	if err != nil {
		return nil, err
	}

	destinationPortRange := []interface{}{}

	if secRule.Properties != nil && secRule.Properties.DestinationPortRange != nil {
		dPortRange := parseAzureSecurityRulePortRange(*secRule.Properties.DestinationPortRange)
		for i := range dPortRange {
			destinationPortRange = append(destinationPortRange, map[string]interface{}{
				"fromPort": dPortRange[i].FromPort,
				"toPort":   dPortRange[i].ToPort,
			})
		}
	}

	if secRule.Properties != nil && secRule.Properties.DestinationPortRanges != nil {
		for _, r := range secRule.Properties.DestinationPortRanges {
			dPortRange := parseAzureSecurityRulePortRange(*r)
			for i := range dPortRange {
				destinationPortRange = append(destinationPortRange, map[string]interface{}{
					"fromPort": dPortRange[i].FromPort,
					"toPort":   dPortRange[i].ToPort,
				})
			}
		}
	}

	res, err := CreateResource(runtime, "azure.subscription.networkService.securityrule",
		map[string]*llx.RawData{
			"id":                   llx.StringDataPtr(secRule.ID),
			"name":                 llx.StringDataPtr(secRule.Name),
			"etag":                 llx.StringDataPtr(secRule.Etag),
			"direction":            llx.StringDataPtr((*string)(secRule.Properties.Direction)),
			"properties":           llx.DictData(properties),
			"destinationPortRange": llx.ArrayData(destinationPortRange, types.String),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionNetworkServiceSecurityrule), nil
}

type AzureSecurityRulePortRange struct {
	FromPort string
	ToPort   string
}

func parseAzureSecurityRulePortRange(portRange string) []AzureSecurityRulePortRange {
	res := []AzureSecurityRulePortRange{}
	entries := strings.Split(portRange, ",")
	for i := range entries {
		entry := strings.TrimSpace(entries[i])
		if strings.Contains(entry, "-") {
			entryRange := strings.Split(entry, "-")
			res = append(res, AzureSecurityRulePortRange{FromPort: entryRange[0], ToPort: entryRange[1]})
		} else {
			res = append(res, AzureSecurityRulePortRange{FromPort: entry, ToPort: entry})
		}
	}
	return res
}

func initAzureSubscriptionNetworkServiceSecurityGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure network security group")
	}

	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	res, err := NewResource(runtime, "azure.subscription.networkService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	network := res.(*mqlAzureSubscriptionNetworkService)
	secGrps := network.GetSecurityGroups()
	if secGrps.Error != nil {
		return nil, nil, secGrps.Error
	}
	id := args["id"].Value.(string)
	for _, entry := range secGrps.Data {
		secGrp := entry.(*mqlAzureSubscriptionNetworkServiceSecurityGroup)
		if secGrp.Id.Data == id {
			return args, secGrp, nil
		}
	}

	return nil, nil, errors.New("azure network security group does not exist")
}
