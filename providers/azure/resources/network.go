// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/azure/connection"
	"go.mondoo.com/cnquery/types"

	network "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
)

func (a *mqlAzureSubscriptionNetwork) id() (string, error) {
	return "azure.subscription.network/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionNetwork(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.AzureConnection)
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionNetwork) interfaces() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewInterfacesClient(subId, token, &arm.ClientOptions{})
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

func (a *mqlAzureSubscriptionNetwork) securityGroups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewSecurityGroupsClient(subId, token, &arm.ClientOptions{})
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

func (a *mqlAzureSubscriptionNetwork) watchers() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewWatchersClient(subId, token, &arm.ClientOptions{})
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

			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.network.watcher",
				map[string]*llx.RawData{
					"id":                llx.StringData(convert.ToString(watcher.ID)),
					"name":              llx.StringData(convert.ToString(watcher.Name)),
					"location":          llx.StringData(convert.ToString(watcher.Location)),
					"tags":              llx.MapData(convert.PtrMapStrToInterface(watcher.Tags), types.String),
					"type":              llx.StringData(convert.ToString(watcher.Type)),
					"etag":              llx.StringData(convert.ToString(watcher.Etag)),
					"properties":        llx.DictData(properties),
					"provisioningState": llx.StringData(convert.ToString((*string)(watcher.Properties.ProvisioningState))),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetwork) publicIpAddresses() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewPublicIPAddressesClient(subId, token, &arm.ClientOptions{})
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
			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.network.ipAddress",
				map[string]*llx.RawData{
					"id":        llx.StringData(convert.ToString(ip.ID)),
					"name":      llx.StringData(convert.ToString(ip.Name)),
					"location":  llx.StringData(convert.ToString(ip.Location)),
					"tags":      llx.MapData(convert.PtrMapStrToInterface(ip.Tags), types.String),
					"type":      llx.StringData(convert.ToString(ip.Type)),
					"ipAddress": llx.StringData(convert.ToString(ip.Properties.IPAddress)),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAzure)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetwork) bastionHosts() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewBastionHostsClient(subId, token, &arm.ClientOptions{})
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
			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.network.bastionHost",
				map[string]*llx.RawData{
					"id":         llx.StringData(convert.ToString(bh.ID)),
					"name":       llx.StringData(convert.ToString(bh.Name)),
					"location":   llx.StringData(convert.ToString(bh.Location)),
					"tags":       llx.MapData(convert.PtrMapStrToInterface(bh.Tags), types.String),
					"type":       llx.StringData(convert.ToString(bh.Type)),
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

func (a *mqlAzureSubscriptionNetworkInterface) vm() (*mqlAzureSubscriptionComputeVm, error) {
	return nil, errors.New("not implemented")
}

func (a *mqlAzureSubscriptionNetworkWatcher) flowLogs() ([]interface{}, error) {
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
	client, err := network.NewFlowLogsClient(subId, token, &arm.ClientOptions{})
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
			mqlFlowLog, err := CreateResource(a.MqlRuntime, "azure.subscription.network.watcher.flowlog",
				map[string]*llx.RawData{
					"id":                 llx.StringData(convert.ToString(flowLog.ID)),
					"name":               llx.StringData(convert.ToString(flowLog.Name)),
					"location":           llx.StringData(convert.ToString(flowLog.Location)),
					"tags":               llx.MapData(convert.PtrMapStrToInterface(flowLog.Tags), types.String),
					"type":               llx.StringData(convert.ToString(flowLog.Type)),
					"etag":               llx.StringData(convert.ToString(flowLog.Etag)),
					"retentionPolicy":    llx.DictData(retentionPolicyDict),
					"format":             llx.StringData(convert.ToString((*string)(flowLog.Properties.Format.Type))),
					"version":            llx.IntData(convert.ToInt64From32(flowLog.Properties.Format.Version)),
					"enabled":            llx.BoolData(convert.ToBool(flowLog.Properties.Enabled)),
					"storageAccountId":   llx.StringData(convert.ToString(flowLog.Properties.StorageID)),
					"targetResourceId":   llx.StringData(convert.ToString(flowLog.Properties.TargetResourceID)),
					"targetResourceGuid": llx.StringData(convert.ToString(flowLog.Properties.TargetResourceGUID)),
					"provisioningState":  llx.StringData(convert.ToString((*string)(flowLog.Properties.ProvisioningState))),
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

func (a *mqlAzureSubscriptionNetworkInterface) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkIpAddress) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkBastionHost) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkSecurityGroup) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkWatcher) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkWatcherFlowlog) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkSecurityrule) id() (string, error) {
	return a.Id.Data, nil
}

func azureInterfaceToMql(runtime *plugin.Runtime, iface network.Interface) (*mqlAzureSubscriptionNetworkInterface, error) {
	properties, err := convert.JsonToDict(iface.Properties)
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(runtime, "azure.subscription.network.interface",
		map[string]*llx.RawData{
			"id":         llx.StringData(convert.ToString(iface.ID)),
			"name":       llx.StringData(convert.ToString(iface.Name)),
			"location":   llx.StringData(convert.ToString(iface.Location)),
			"tags":       llx.MapData(convert.PtrMapStrToInterface(iface.Tags), types.String),
			"type":       llx.StringData(convert.ToString(iface.Type)),
			"etag":       llx.StringData(convert.ToString(iface.Etag)),
			"properties": llx.DictData(properties),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionNetworkInterface), nil
}

// see https://github.com/Azure/azure-sdk-for-go/issues/8224
type AzureSecurityGroupPropertiesFormat network.SecurityGroupPropertiesFormat

func azureSecGroupToMql(runtime *plugin.Runtime, secGroup network.SecurityGroup) (*mqlAzureSubscriptionNetworkSecurityGroup, error) {
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
	res, err := CreateResource(runtime, "azure.subscription.network.securityGroup",
		map[string]*llx.RawData{
			"id":                   llx.StringData(convert.ToString(secGroup.ID)),
			"name":                 llx.StringData(convert.ToString(secGroup.Name)),
			"location":             llx.StringData(convert.ToString(secGroup.Location)),
			"tags":                 llx.MapData(convert.PtrMapStrToInterface(secGroup.Tags), types.String),
			"type":                 llx.StringData(convert.ToString(secGroup.Type)),
			"etag":                 llx.StringData(convert.ToString(secGroup.Etag)),
			"properties":           llx.DictData(properties),
			"interfaces":           llx.ArrayData(ifaces, types.ResourceLike),
			"securityRules":        llx.ArrayData(securityRules, types.ResourceLike),
			"defaultSecurityRules": llx.ArrayData(defaultSecurityRules, types.ResourceLike),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionNetworkSecurityGroup), nil
}

func azureSecurityRuleToMql(runtime *plugin.Runtime, secRule network.SecurityRule) (*mqlAzureSubscriptionNetworkSecurityrule, error) {
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

	res, err := CreateResource(runtime, "azure.subscription.network.securityrule",
		map[string]*llx.RawData{
			"id":                   llx.StringData(convert.ToString(secRule.ID)),
			"name":                 llx.StringData(convert.ToString(secRule.Name)),
			"etag":                 llx.StringData(convert.ToString(secRule.Etag)),
			"properties":           llx.DictData(properties),
			"destinationPortRange": llx.ArrayData(destinationPortRange, types.String),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionNetworkSecurityrule), nil
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

func initAzureSubscriptionNetworkSecurityGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

	conn := runtime.Connection.(*connection.AzureConnection)
	res, err := NewResource(runtime, "azure.subscription.network", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	network := res.(*mqlAzureSubscriptionNetwork)
	secGrps := network.GetSecurityGroups()
	if secGrps.Error != nil {
		return nil, nil, secGrps.Error
	}
	id := args["id"].Value.(string)
	for _, entry := range secGrps.Data {
		secGrp := entry.(*mqlAzureSubscriptionNetworkSecurityGroup)
		if secGrp.Id.Data == id {
			return args, secGrp, nil
		}
	}

	return nil, nil, errors.New("azure network security group does not exist")
}
