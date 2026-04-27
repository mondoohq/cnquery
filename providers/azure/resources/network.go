// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
	"go.mondoo.com/mql/v13/utils/stringx"
	"golang.org/x/sync/errgroup"

	network "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
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

func (a *mqlAzureSubscriptionNetworkService) interfaces() ([]any, error) {
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
	res := []any{}
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

func (a *mqlAzureSubscriptionNetworkService) securityGroups() ([]any, error) {
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
	res := []any{}
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

func (a *mqlAzureSubscriptionNetworkService) watchers() ([]any, error) {
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
	res := []any{}
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

func (a *mqlAzureSubscriptionNetworkService) publicIpAddresses() ([]any, error) {
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
	res := []any{}
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

func (a *mqlAzureSubscriptionNetworkService) bastionHosts() ([]any, error) {
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
	res := []any{}
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
	props := a.Properties.Data
	if props == nil {
		a.Vm.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	propsMap, ok := props.(map[string]any)
	if !ok {
		a.Vm.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	vmRef, ok := propsMap["virtualMachine"]
	if !ok || vmRef == nil {
		a.Vm.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	vmMap, ok := vmRef.(map[string]any)
	if !ok {
		a.Vm.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	vmID, ok := vmMap["id"].(string)
	if !ok || vmID == "" {
		a.Vm.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "azure.subscription.computeService.vm", map[string]*llx.RawData{
		"id": llx.StringData(strings.ToLower(vmID)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionComputeServiceVm), nil
}

// effectiveSecurityRules computes the merged NSG rules effective on this NIC
// (NSG attached to NIC + ASG + NSG attached to subnet). Lazily called per NIC.
//
// Azure only computes effective rules for NICs attached to a running VM; for
// detached or stopped NICs the API returns NicNotAssociatedWithVm or similar
// 400/404 errors. We treat those as "no effective rules" rather than failing
// the whole interfaces query.
func (a *mqlAzureSubscriptionNetworkServiceInterface) effectiveSecurityRules() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	// Bound the long-poll so a stuck operation doesn't hang the interfaces query.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	nicName, err := resourceID.Component("networkInterfaces")
	if err != nil {
		return nil, err
	}

	// armnetwork v9 misshapes EffectiveNetworkSecurityGroup.TagMap (declared *string,
	// API returns object), so SDK unmarshalling fails. Use REST directly and pluck
	// the effectiveSecurityRules out as raw JSON.
	tok, err := conn.Token().GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf(
		"https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/%s/effectiveNetworkSecurityGroups?api-version=2024-05-01",
		resourceID.SubscriptionID, resourceID.ResourceGroup, nicName,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok.Token)
	req.Header.Set("Accept", "application/json")

	httpClient := &http.Client{Timeout: 60 * time.Second}
	httpResp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	// 202 Accepted → poll the Location header until the result is ready. We don't
	// fall back to Azure-AsyncOperation: that endpoint returns a status envelope
	// (`{"status": "InProgress"|"Succeeded"|"Failed"}`), not the effective-rules
	// payload, so a 200 from it would just be the loop exiting onto the wrong body.
	for httpResp.StatusCode == http.StatusAccepted {
		loc := httpResp.Header.Get("Location")
		if loc == "" {
			io.Copy(io.Discard, httpResp.Body)
			return nil, fmt.Errorf("effective NSG list returned 202 without a Location header")
		}
		// Sleep a beat then poll.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
		pollReq, perr := http.NewRequestWithContext(ctx, http.MethodGet, loc, nil)
		if perr != nil {
			return nil, perr
		}
		pollReq.Header.Set("Authorization", "Bearer "+tok.Token)
		pollReq.Header.Set("Accept", "application/json")
		newResp, perr := httpClient.Do(pollReq)
		if perr != nil {
			return nil, perr
		}
		httpResp.Body.Close()
		httpResp = newResp
	}

	if httpResp.StatusCode == http.StatusBadRequest ||
		httpResp.StatusCode == http.StatusNotFound ||
		httpResp.StatusCode == http.StatusForbidden {
		log.Warn().Str("nic", nicName).Int("status", httpResp.StatusCode).Msg("effective security rules unavailable for NIC")
		return []any{}, nil
	}
	if httpResp.StatusCode >= 400 {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("effective NSG list returned %d: %s", httpResp.StatusCode, string(body))
	}

	var payload struct {
		Value []struct {
			EffectiveSecurityRules []any `json:"effectiveSecurityRules"`
		} `json:"value"`
	}
	if err := json.NewDecoder(httpResp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	var res []any
	for _, ensg := range payload.Value {
		for _, rule := range ensg.EffectiveSecurityRules {
			if rule == nil {
				continue
			}
			res = append(res, rule)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceWatcher) flowLogs() ([]any, error) {
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
	res := []any{}
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
			var retentionPolicy mqlRetentionPolicy
			if rp := flowLog.Properties.RetentionPolicy; rp != nil {
				retentionPolicy = mqlRetentionPolicy{
					Enabled:       convert.ToValue(rp.Enabled),
					RetentionDays: int(convert.ToValue(rp.Days)),
				}
			}
			retentionPolicyDict, err := convert.JsonToDict(retentionPolicy)
			if err != nil {
				return nil, err
			}
			var flowLogAnalytics mqlFlowLogAnalytics
			if fac := flowLog.Properties.FlowAnalyticsConfiguration; fac != nil && fac.NetworkWatcherFlowAnalyticsConfiguration != nil {
				nwfac := fac.NetworkWatcherFlowAnalyticsConfiguration
				flowLogAnalytics = mqlFlowLogAnalytics{
					Enabled:             convert.ToValue(nwfac.Enabled),
					AnalyticsInterval:   int(convert.ToValue(nwfac.TrafficAnalyticsInterval)),
					WorkspaceRegion:     convert.ToValue(nwfac.WorkspaceRegion),
					WorkspaceResourceId: convert.ToValue(nwfac.WorkspaceResourceID),
					WorkspaceId:         convert.ToValue(nwfac.WorkspaceID),
				}
			}
			flowLogAnalyticsDict, err := convert.JsonToDict(flowLogAnalytics)
			if err != nil {
				return nil, err
			}
			var formatType *string
			var formatVersion *int32
			if f := flowLog.Properties.Format; f != nil {
				formatType = (*string)(f.Type)
				formatVersion = f.Version
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
					"format":             llx.StringDataPtr(formatType),
					"version":            llx.IntDataDefault(formatVersion, 0),
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

func (a *mqlAzureSubscriptionNetworkService) loadBalancers() ([]any, error) {
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
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, lb := range page.Value {
			lbProps, err := convert.JsonToDict(lb.Properties)
			if err != nil {
				return nil, err
			}
			var lbSkuName, lbSkuTier *string
			if lb.SKU != nil {
				lbSkuName = (*string)(lb.SKU.Name)
				lbSkuTier = (*string)(lb.SKU.Tier)
			}
			mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.loadBalancer",
				map[string]*llx.RawData{
					"id":         llx.StringDataPtr(lb.ID),
					"name":       llx.StringDataPtr(lb.Name),
					"location":   llx.StringDataPtr(lb.Location),
					"etag":       llx.StringDataPtr(lb.Etag),
					"sku":        llx.StringDataPtr(lbSkuName),
					"skuTier":    llx.StringDataPtr(lbSkuTier),
					"tags":       llx.MapData(convert.PtrMapStrToInterface(lb.Tags), types.String),
					"type":       llx.StringDataPtr(lb.Type),
					"properties": llx.DictData(lbProps),
				})
			if err != nil {
				return nil, err
			}
			mqlLb := mqlAzure.(*mqlAzureSubscriptionNetworkServiceLoadBalancer)
			mqlLb.cacheProperties = lb.Properties
			res = append(res, mqlLb)
		}
	}
	return res, nil
}

type mqlAzureSubscriptionNetworkServiceLoadBalancerInternal struct {
	cacheProperties *network.LoadBalancerPropertiesFormat
}

func (a *mqlAzureSubscriptionNetworkServiceLoadBalancer) probes() ([]any, error) {
	if a.cacheProperties == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, p := range a.cacheProperties.Probes {
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
		res = append(res, mqlProbe)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceLoadBalancer) backendPools() ([]any, error) {
	if a.cacheProperties == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, bap := range a.cacheProperties.BackendAddressPools {
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
		res = append(res, mqlBap)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceLoadBalancer) frontendIpConfigs() ([]any, error) {
	if a.cacheProperties == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, ipConfig := range a.cacheProperties.FrontendIPConfigurations {
		props, err := convert.JsonToDict(ipConfig.Properties)
		if err != nil {
			return nil, err
		}
		isPublic := false
		var publicIpAddressId string
		var privateIpAddress string
		if ipConfig.Properties != nil {
			if ipConfig.Properties.PublicIPAddress != nil && ipConfig.Properties.PublicIPAddress.ID != nil {
				isPublic = true
				publicIpAddressId = *ipConfig.Properties.PublicIPAddress.ID
			}
			if ipConfig.Properties.PrivateIPAddress != nil {
				privateIpAddress = *ipConfig.Properties.PrivateIPAddress
			}
		}

		mqlIpConfig, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.frontendIpConfig",
			map[string]*llx.RawData{
				"id":                llx.StringDataPtr(ipConfig.ID),
				"type":              llx.StringDataPtr(ipConfig.Type),
				"name":              llx.StringDataPtr(ipConfig.Name),
				"etag":              llx.StringDataPtr(ipConfig.Etag),
				"zones":             llx.ArrayData(convert.SliceStrPtrToInterface(ipConfig.Zones), types.String),
				"properties":        llx.DictData(props),
				"isPublic":          llx.BoolData(isPublic),
				"publicIpAddressId": llx.StringData(publicIpAddressId),
				"privateIpAddress":  llx.StringData(privateIpAddress),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlIpConfig)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceLoadBalancer) inboundNatPools() ([]any, error) {
	if a.cacheProperties == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, natPool := range a.cacheProperties.InboundNatPools {
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
		res = append(res, mqlNatPool)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceLoadBalancer) inboundNatRules() ([]any, error) {
	if a.cacheProperties == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, natRule := range a.cacheProperties.InboundNatRules {
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
		res = append(res, mqlNatRule)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceLoadBalancer) outboundRules() ([]any, error) {
	if a.cacheProperties == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, outboundRule := range a.cacheProperties.OutboundRules {
		props, err := convert.JsonToDict(outboundRule.Properties)
		if err != nil {
			return nil, err
		}
		mqlOutbound, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.outboundRule",
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
		res = append(res, mqlOutbound)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceLoadBalancer) loadBalancerRules() ([]any, error) {
	if a.cacheProperties == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, lbRule := range a.cacheProperties.LoadBalancingRules {
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
		res = append(res, mqlLbRule)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkService) natGateways() ([]any, error) {
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
	res := []any{}
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

func (a *mqlAzureSubscriptionNetworkService) firewalls() ([]any, error) {
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
	res := []any{}
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
	propsDict := props.(map[string]any)
	fwp := propsDict["firewallPolicy"]
	if fwp == nil {
		return nil, errors.New("no firewall policy is associated with the ip configuration")
	}
	fwpDict := fwp.(map[string]any)
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
	propsDict := props.(map[string]any)
	publicIpAddress := propsDict["publicIPAddress"]
	if publicIpAddress == nil {
		return nil, errors.New("no public ip address is associated with the ip configuration")
	}
	ipAddressDict := publicIpAddress.(map[string]any)
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
	propsDict := props.(map[string]any)
	publicIpAddress := propsDict["publicIPAddress"]
	if publicIpAddress == nil {
		return nil, errors.New("no public ip address is associated with the ip configuration")
	}
	ipAddressDict := publicIpAddress.(map[string]any)
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
	propsDict := props.(map[string]any)
	subnet := propsDict["subnet"]
	if subnet == nil {
		return nil, errors.New("no subnet is associated with the ip configuration")
	}
	subnetDict := subnet.(map[string]any)
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

func (a *mqlAzureSubscriptionNetworkService) firewallPolicies() ([]any, error) {
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
	res := []any{}
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

func azureVirtualNetworkToMql(runtime *plugin.Runtime, vn network.VirtualNetwork) (*mqlAzureSubscriptionNetworkServiceVirtualNetwork, error) {
	props, err := convert.JsonToDict(vn.Properties)
	if err != nil {
		return nil, err
	}
	subnets := []any{}
	if vn.Properties != nil {
		for _, s := range vn.Properties.Subnets {
			if s != nil {
				mqlSubnet, err := azureSubnetToMql(runtime, *s)
				if err != nil {
					return nil, err
				}
				subnets = append(subnets, mqlSubnet)
			}
		}
	}
	args := map[string]*llx.RawData{
		"id":         llx.StringDataPtr(vn.ID),
		"name":       llx.StringDataPtr(vn.Name),
		"type":       llx.StringDataPtr(vn.Type),
		"location":   llx.StringDataPtr(vn.Location),
		"tags":       llx.MapData(convert.PtrMapStrToInterface(vn.Tags), types.String),
		"etag":       llx.StringDataPtr(vn.Etag),
		"properties": llx.DictData(props),
		"subnets":    llx.ArrayData(subnets, types.ResourceLike),
	}
	if vn.Properties != nil {
		args["enableDdosProtection"] = llx.BoolDataPtr(vn.Properties.EnableDdosProtection)
		args["enableVmProtection"] = llx.BoolDataPtr(vn.Properties.EnableVMProtection)
		args["provisioningState"] = llx.StringDataPtr((*string)(vn.Properties.ProvisioningState))
		args["flowTimeoutInMinutes"] = llx.IntDataPtr(vn.Properties.FlowTimeoutInMinutes)
		if vn.Properties.AddressSpace != nil {
			args["addressPrefixes"] = llx.ArrayData(convert.SliceStrPtrToInterface(vn.Properties.AddressSpace.AddressPrefixes), types.String)
		} else {
			args["addressPrefixes"] = llx.ArrayData([]any{}, types.String)
		}
		if vn.Properties.Encryption != nil {
			args["encryptionEnabled"] = llx.BoolDataPtr(vn.Properties.Encryption.Enabled)
			args["encryptionEnforcement"] = llx.StringDataPtr((*string)(vn.Properties.Encryption.Enforcement))
		} else {
			args["encryptionEnabled"] = llx.BoolData(false)
			args["encryptionEnforcement"] = llx.StringData("")
		}
		if vn.Properties.DhcpOptions != nil {
			id := convert.ToValue(vn.ID) + "/dhcpOptions"
			dhcpOpts, err := CreateResource(runtime, "azure.subscription.networkService.virtualNetwork.dhcpOptions",
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
	} else {
		args["enableDdosProtection"] = llx.BoolData(false)
		args["enableVmProtection"] = llx.BoolData(false)
		args["provisioningState"] = llx.StringData("")
		args["flowTimeoutInMinutes"] = llx.IntData(0)
		args["addressPrefixes"] = llx.ArrayData([]any{}, types.String)
		args["encryptionEnabled"] = llx.BoolData(false)
		args["encryptionEnforcement"] = llx.StringData("")
		args["dhcpOptions"] = llx.NilData
	}

	mqlVn, err := CreateResource(runtime, ResourceAzureSubscriptionNetworkServiceVirtualNetwork, args)
	if err != nil {
		return nil, err
	}
	return mqlVn.(*mqlAzureSubscriptionNetworkServiceVirtualNetwork), nil
}

func (a *mqlAzureSubscriptionNetworkService) virtualNetworks() ([]any, error) {
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
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, vn := range page.Value {
			mqlVn, err := azureVirtualNetworkToMql(a.MqlRuntime, *vn)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlVn)
		}
	}
	return res, nil
}

func initAzureSubscriptionNetworkServiceVirtualNetwork(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure virtual network")
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
	networkSvc := res.(*mqlAzureSubscriptionNetworkService)
	vnets := networkSvc.GetVirtualNetworks()
	if vnets.Error != nil {
		return nil, nil, vnets.Error
	}
	id := args["id"].Value.(string)
	for _, entry := range vnets.Data {
		vnet := entry.(*mqlAzureSubscriptionNetworkServiceVirtualNetwork)
		if vnet.Id.Data == id {
			return args, vnet, nil
		}
	}

	return nil, nil, errors.New("azure virtual network does not exist")
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualNetwork) peerings() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := conn.SubId()

	client, err := network.NewVirtualNetworkPeeringsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	// Extract resource group and vnet name from the ID
	// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{name}
	id := a.Id.Data
	parts := strings.Split(id, "/")
	var rgName, vnetName string
	for i, p := range parts {
		if strings.EqualFold(p, "resourceGroups") && i+1 < len(parts) {
			rgName = parts[i+1]
		}
		if strings.EqualFold(p, "virtualNetworks") && i+1 < len(parts) {
			vnetName = parts[i+1]
		}
	}
	if rgName == "" || vnetName == "" {
		return nil, fmt.Errorf("could not parse resource group and vnet name from id: %s", id)
	}

	pager := client.NewListPager(rgName, vnetName, &network.VirtualNetworkPeeringsClientListOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, p := range page.Value {
			if p.Properties == nil {
				continue
			}
			var remoteVnetId string
			if p.Properties.RemoteVirtualNetwork != nil && p.Properties.RemoteVirtualNetwork.ID != nil {
				remoteVnetId = *p.Properties.RemoteVirtualNetwork.ID
			}
			var remoteEncEnabled bool
			var remoteEncEnforcement string
			if p.Properties.RemoteVirtualNetworkEncryption != nil {
				if p.Properties.RemoteVirtualNetworkEncryption.Enabled != nil {
					remoteEncEnabled = *p.Properties.RemoteVirtualNetworkEncryption.Enabled
				}
				if p.Properties.RemoteVirtualNetworkEncryption.Enforcement != nil {
					remoteEncEnforcement = string(*p.Properties.RemoteVirtualNetworkEncryption.Enforcement)
				}
			}
			mqlPeering, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.virtualNetwork.peering",
				map[string]*llx.RawData{
					"id":                                    llx.StringDataPtr(p.ID),
					"name":                                  llx.StringDataPtr(p.Name),
					"allowForwardedTraffic":                 llx.BoolDataPtr(p.Properties.AllowForwardedTraffic),
					"allowGatewayTransit":                   llx.BoolDataPtr(p.Properties.AllowGatewayTransit),
					"allowVirtualNetworkAccess":             llx.BoolDataPtr(p.Properties.AllowVirtualNetworkAccess),
					"useRemoteGateways":                     llx.BoolDataPtr(p.Properties.UseRemoteGateways),
					"peeringState":                          llx.StringDataPtr((*string)(p.Properties.PeeringState)),
					"peeringSyncLevel":                      llx.StringDataPtr((*string)(p.Properties.PeeringSyncLevel)),
					"provisioningState":                     llx.StringDataPtr((*string)(p.Properties.ProvisioningState)),
					"remoteVirtualNetworkId":                llx.StringData(remoteVnetId),
					"remoteVirtualNetworkEncryptionEnabled": llx.BoolData(remoteEncEnabled),
					"remoteVirtualNetworkEncryptionEnforcement": llx.StringData(remoteEncEnforcement),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlPeering)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualNetworkPeering) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkService) applicationSecurityGroups() ([]any, error) {
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
	res := []any{}
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

func (a *mqlAzureSubscriptionNetworkService) virtualNetworkGateways() ([]any, error) {
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
	res := []any{}
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
					"gatewayType":                     llx.StringDataPtr((*string)(vng.Properties.GatewayType)),
				}
				if vng.Properties.CustomRoutes != nil {
					args["addressPrefixes"] = llx.ArrayData(convert.SliceStrPtrToInterface(vng.Properties.CustomRoutes.AddressPrefixes), types.String)
				} else {
					args["addressPrefixes"] = llx.ArrayData([]any{}, types.String)
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
				mqlGw := mqlVn.(*mqlAzureSubscriptionNetworkServiceVirtualNetworkGateway)
				mqlGw.cacheProperties = vng.Properties
				res = append(res, mqlGw)
			}
		}
	}
	return res, nil
}

type mqlAzureSubscriptionNetworkServiceVirtualNetworkGatewayInternal struct {
	cacheProperties *network.VirtualNetworkGatewayPropertiesFormat
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualNetworkGateway) bgpSettings() (*mqlAzureSubscriptionNetworkServiceBgpSettings, error) {
	if a.cacheProperties == nil || a.cacheProperties.BgpSettings == nil {
		a.BgpSettings.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	bgpSettingsId := a.Id.Data + "/bgpSettings"
	bgpPeeringAddresses := []any{}
	for i, bpa := range a.cacheProperties.BgpSettings.BgpPeeringAddresses {
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
	mqlBgp, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.bgpSettings",
		map[string]*llx.RawData{
			"id":                        llx.StringData(bgpSettingsId),
			"asn":                       llx.IntDataPtr(a.cacheProperties.BgpSettings.Asn),
			"bgpPeeringAddress":         llx.StringDataPtr(a.cacheProperties.BgpSettings.BgpPeeringAddress),
			"peerWeight":                llx.IntDataDefault(a.cacheProperties.BgpSettings.PeerWeight, 0),
			"bgpPeeringAddressesConfig": llx.ArrayData(bgpPeeringAddresses, types.ResourceLike),
		})
	if err != nil {
		return nil, err
	}
	return mqlBgp.(*mqlAzureSubscriptionNetworkServiceBgpSettings), nil
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualNetworkGateway) ipConfigurations() ([]any, error) {
	if a.cacheProperties == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, ipc := range a.cacheProperties.IPConfigurations {
		props, err := convert.JsonToDict(ipc.Properties)
		if err != nil {
			return nil, err
		}
		var privateIP *string
		if ipc.Properties != nil {
			privateIP = ipc.Properties.PrivateIPAddress
		}
		mqlIpc, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.virtualNetworkGateway.ipConfig", map[string]*llx.RawData{
			"id":               llx.StringDataPtr(ipc.ID),
			"name":             llx.StringDataPtr(ipc.Name),
			"etag":             llx.StringDataPtr(ipc.Etag),
			"properties":       llx.DictData(props),
			"privateIpAddress": llx.StringDataPtr(privateIP),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlIpc)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceVirtualNetworkGateway) natRules() ([]any, error) {
	if a.cacheProperties == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, nr := range a.cacheProperties.NatRules {
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
		res = append(res, mqlNr)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkService) applicationGateways() ([]any, error) {
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

	res := []any{}
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

func (a *mqlAzureSubscriptionNetworkServiceWafConfig) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceApplicationGateway) wafConfiguration() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	client, err := network.NewClientFactory(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	c := client.NewApplicationGatewayWafDynamicManifestsClient()

	res := []any{}
	pager := c.NewGetPager(a.Location.Data, &network.ApplicationGatewayWafDynamicManifestsClientGetOptions{})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			if entry != nil {
				props, err := convert.JsonToDict(entry.Properties)
				if err != nil {
					return nil, err
				}
				mqlAzure, err := CreateResource(a.MqlRuntime, "azure.subscription.applicationGateway.wafconfig",
					map[string]*llx.RawData{
						"id":         llx.StringDataPtr(entry.ID),
						"name":       llx.StringDataPtr(entry.Name),
						"type":       llx.StringDataPtr(entry.Type),
						"properties": llx.AnyData(props),
					})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlAzure)
			}
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkService) applicationFirewallPolicies() ([]any, error) {
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

	res := []any{}
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

func (a *mqlAzureSubscriptionNetworkService) privateEndpoints() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewPrivateEndpointsClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListBySubscriptionPager(&network.PrivateEndpointsClientListBySubscriptionOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, pe := range page.Value {
			if pe == nil {
				continue
			}

			var provisioningState, subnetId, customNicName string
			var plsConns, manualPlsConns []any

			if pe.Properties != nil {
				if pe.Properties.ProvisioningState != nil {
					provisioningState = string(*pe.Properties.ProvisioningState)
				}
				if pe.Properties.Subnet != nil {
					subnetId = convert.ToValue(pe.Properties.Subnet.ID)
				}
				customNicName = convert.ToValue(pe.Properties.CustomNetworkInterfaceName)

				for _, c := range pe.Properties.PrivateLinkServiceConnections {
					mqlConn, err := privateLinkServiceConnectionToMql(a.MqlRuntime, c)
					if err != nil {
						return nil, err
					}
					plsConns = append(plsConns, mqlConn)
				}
				for _, c := range pe.Properties.ManualPrivateLinkServiceConnections {
					mqlConn, err := privateLinkServiceConnectionToMql(a.MqlRuntime, c)
					if err != nil {
						return nil, err
					}
					manualPlsConns = append(manualPlsConns, mqlConn)
				}
			}

			mqlPe, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.privateEndpoint",
				map[string]*llx.RawData{
					"id":                                  llx.StringDataPtr(pe.ID),
					"name":                                llx.StringDataPtr(pe.Name),
					"location":                            llx.StringDataPtr(pe.Location),
					"tags":                                llx.MapData(convert.PtrMapStrToInterface(pe.Tags), types.String),
					"type":                                llx.StringDataPtr(pe.Type),
					"provisioningState":                   llx.StringData(provisioningState),
					"subnetId":                            llx.StringData(subnetId),
					"customNetworkInterfaceName":          llx.StringData(customNicName),
					"privateLinkServiceConnections":       llx.ArrayData(plsConns, types.ResourceLike),
					"manualPrivateLinkServiceConnections": llx.ArrayData(manualPlsConns, types.ResourceLike),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlPe)
		}
	}
	return res, nil
}

// privateDnsZoneGroups fetches the Private DNS Zone Groups attached to this PE.
// Each group lists which Private DNS zones records will be auto-registered into.
func (a *mqlAzureSubscriptionNetworkServicePrivateEndpoint) privateDnsZoneGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	id := a.Id.Data
	resourceID, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	peName, err := resourceID.Component("privateEndpoints")
	if err != nil {
		return nil, err
	}

	client, err := network.NewPrivateDNSZoneGroupsClient(resourceID.SubscriptionID, conn.Token(), &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(peName, resourceID.ResourceGroup, nil)
	var res []any
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, g := range page.Value {
			if g == nil {
				continue
			}
			entry := map[string]any{
				"id":   convert.ToValue(g.ID),
				"name": convert.ToValue(g.Name),
			}
			if g.Properties != nil {
				var zoneIds []any
				var configs []any
				for _, c := range g.Properties.PrivateDNSZoneConfigs {
					if c == nil {
						continue
					}
					ce := map[string]any{
						"name": convert.ToValue(c.Name),
					}
					if c.Properties != nil && c.Properties.PrivateDNSZoneID != nil {
						ce["privateDnsZoneId"] = *c.Properties.PrivateDNSZoneID
						zoneIds = append(zoneIds, *c.Properties.PrivateDNSZoneID)
					}
					configs = append(configs, ce)
				}
				entry["privateDnsZoneIds"] = zoneIds
				entry["configs"] = configs
				if g.Properties.ProvisioningState != nil {
					entry["provisioningState"] = string(*g.Properties.ProvisioningState)
				}
			}
			res = append(res, entry)
		}
	}
	return res, nil
}

func privateLinkServiceConnectionToMql(runtime *plugin.Runtime, c *network.PrivateLinkServiceConnection) (*mqlAzureSubscriptionNetworkServicePrivateEndpointServiceconnection, error) {
	if c == nil {
		return nil, errors.New("private link service connection is nil")
	}

	var plsId, connectionStatus, requestMessage string
	var groupIds []any

	if c.Properties != nil {
		plsId = convert.ToValue(c.Properties.PrivateLinkServiceID)
		requestMessage = convert.ToValue(c.Properties.RequestMessage)
		if c.Properties.PrivateLinkServiceConnectionState != nil {
			connectionStatus = convert.ToValue(c.Properties.PrivateLinkServiceConnectionState.Status)
		}
		for _, gid := range c.Properties.GroupIDs {
			if gid != nil {
				groupIds = append(groupIds, *gid)
			}
		}
	}

	res, err := CreateResource(runtime, "azure.subscription.networkService.privateEndpoint.serviceconnection",
		map[string]*llx.RawData{
			"id":                   llx.StringDataPtr(c.ID),
			"name":                 llx.StringDataPtr(c.Name),
			"privateLinkServiceId": llx.StringData(plsId),
			"groupIds":             llx.ArrayData(groupIds, types.String),
			"connectionStatus":     llx.StringData(connectionStatus),
			"requestMessage":       llx.StringData(requestMessage),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionNetworkServicePrivateEndpointServiceconnection), nil
}

func (a *mqlAzureSubscriptionNetworkService) routeTables() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewRouteTablesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListAllPager(&network.RouteTablesClientListAllOptions{})
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, rt := range page.Value {
			if rt == nil {
				continue
			}

			var disableBgp bool
			var provisioningState string
			var routes []any

			if rt.Properties != nil {
				disableBgp = convert.ToValue(rt.Properties.DisableBgpRoutePropagation)
				if rt.Properties.ProvisioningState != nil {
					provisioningState = string(*rt.Properties.ProvisioningState)
				}
				for _, route := range rt.Properties.Routes {
					if route == nil {
						continue
					}
					mqlRoute, err := azureRouteToMql(a.MqlRuntime, route)
					if err != nil {
						return nil, err
					}
					routes = append(routes, mqlRoute)
				}
			}

			mqlRt, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.routeTable",
				map[string]*llx.RawData{
					"id":                         llx.StringDataPtr(rt.ID),
					"name":                       llx.StringDataPtr(rt.Name),
					"location":                   llx.StringDataPtr(rt.Location),
					"tags":                       llx.MapData(convert.PtrMapStrToInterface(rt.Tags), types.String),
					"type":                       llx.StringDataPtr(rt.Type),
					"etag":                       llx.StringDataPtr(rt.Etag),
					"disableBgpRoutePropagation": llx.BoolData(disableBgp),
					"provisioningState":          llx.StringData(provisioningState),
					"routes":                     llx.ArrayData(routes, types.ResourceLike),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRt)
		}
	}
	return res, nil
}

func azureRouteToMql(runtime *plugin.Runtime, route *network.Route) (*mqlAzureSubscriptionNetworkServiceRoute, error) {
	var addressPrefix, nextHopType, nextHopIpAddress, provisioningState string
	var hasBgpOverride bool

	if route.Properties != nil {
		addressPrefix = convert.ToValue(route.Properties.AddressPrefix)
		nextHopIpAddress = convert.ToValue(route.Properties.NextHopIPAddress)
		hasBgpOverride = convert.ToValue(route.Properties.HasBgpOverride)
		if route.Properties.NextHopType != nil {
			nextHopType = string(*route.Properties.NextHopType)
		}
		if route.Properties.ProvisioningState != nil {
			provisioningState = string(*route.Properties.ProvisioningState)
		}
	}

	res, err := CreateResource(runtime, "azure.subscription.networkService.route",
		map[string]*llx.RawData{
			"id":                llx.StringDataPtr(route.ID),
			"name":              llx.StringDataPtr(route.Name),
			"addressPrefix":     llx.StringData(addressPrefix),
			"nextHopType":       llx.StringData(nextHopType),
			"nextHopIpAddress":  llx.StringData(nextHopIpAddress),
			"hasBgpOverride":    llx.BoolData(hasBgpOverride),
			"provisioningState": llx.StringData(provisioningState),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionNetworkServiceRoute), nil
}

func (a *mqlAzureSubscriptionNetworkServiceApplicationGateway) policy() (*mqlAzureSubscriptionNetworkServiceApplicationFirewallPolicy, error) {
	props := a.Properties
	if props.Error != nil {
		return nil, props.Error
	}
	propsDict := props.Data.(map[string]any)
	fwDict := propsDict["firewallPolicy"]
	if fwDict == nil {
		return nil, errors.New("no firewall policy is associated with the application gateway")
	}
	fwId := fwDict.(map[string]any)["id"]
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

func (a *mqlAzureSubscriptionNetworkServiceApplicationFirewallPolicy) gateways() ([]any, error) {
	props := a.Properties
	if props.Error != nil {
		return nil, props.Error
	}
	propsDict := props.Data.(map[string]any)
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

	gatewaysList := gateways.([]any)

	// Pre-validate all gateway IDs before launching any goroutines so that an
	// early parse error can't leak in-flight workers.
	type gwFetch struct {
		rg    string
		name  string
		index int
	}
	fetches := make([]gwFetch, 0, len(gatewaysList))
	for i, gw := range gatewaysList {
		idVal, ok := gw.(map[string]any)["id"]
		if !ok {
			continue
		}
		strId, ok := idVal.(string)
		if !ok {
			continue
		}
		azureId, err := ParseResourceID(strId)
		if err != nil {
			return nil, err
		}
		gatewayName, err := azureId.Component("applicationGateways")
		if err != nil {
			return nil, err
		}
		fetches = append(fetches, gwFetch{rg: azureId.ResourceGroup, name: gatewayName, index: i})
	}

	// Fetch the referenced application gateways in parallel; there is no
	// batch endpoint, so a bounded errgroup is the cheapest fix.
	results := make([]network.ApplicationGateway, len(gatewaysList))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(10)
	for _, f := range fetches {
		g.Go(func() error {
			resp, err := client.Get(gctx, f.rg, f.name, &network.ApplicationGatewaysClientGetOptions{})
			if err != nil {
				return err
			}
			results[f.index] = resp.ApplicationGateway
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	res := []any{}
	for _, gw := range results {
		if gw.ID == nil {
			continue
		}
		mqlGateway, err := azureAppGatewayToMql(a.MqlRuntime, gw)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlGateway)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceNatGateway) publicIpAddresses() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	azureId, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	props := a.Properties.Data
	propsDict := props.(map[string]any)
	publicIpAddresses := propsDict["publicIpAddresses"]
	// if we have no present public ip addresses ids, we can just return nil
	if publicIpAddresses == nil {
		return nil, nil
	}

	res := []any{}
	client, err := network.NewPublicIPAddressesClient(azureId.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	for _, p := range publicIpAddresses.([]any) {
		pDict := p.(map[string]any)
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

func (a *mqlAzureSubscriptionNetworkServiceVirtualNetworkGateway) connections() ([]any, error) {
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
	res := []any{}
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

func (a *mqlAzureSubscriptionNetworkServiceNatGateway) subnets() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	id := a.Id.Data
	azureId, err := ParseResourceID(id)
	if err != nil {
		return nil, err
	}
	props := a.Properties.Data
	propsDict := props.(map[string]any)
	subnets := propsDict["subnets"]
	// if we have no present subnets in the dict, we can just return nil
	if subnets == nil {
		return nil, nil
	}
	res := []any{}
	client, err := network.NewSubnetsClient(azureId.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	for _, s := range subnets.([]any) {
		sDict := s.(map[string]any)
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
	propsDict := props.(map[string]any)
	natGatewayDict := propsDict["natGateway"]
	if natGatewayDict == nil {
		// TODO: Preslav: how do we define a 'nil' resource here? if i return nil, it panics
		return nil, errors.New("subnet has no NAT gateway associated with it")
	}
	natGatewayFields := natGatewayDict.(map[string]any)
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

func (a *mqlAzureSubscriptionNetworkServiceSubnet) ipConfigurations() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	subId := conn.SubId()
	props := a.Properties.Data
	propsDict := props.(map[string]any)
	ipConfigsDict := propsDict["ipConfigurations"]
	if ipConfigsDict == nil {
		return nil, nil
	}
	res := []any{}
	ipConfigIds := []string{}
	ipConfigsList := ipConfigsDict.([]any)
	for _, ipc := range ipConfigsList {
		ipcDict := ipc.(map[string]any)
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
	// no API to fetch this so we fetch the gateways and iterate through them
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
	propsDict := props.(map[string]any)
	basePolicy := propsDict["basePolicy"]
	if basePolicy == nil {
		// TODO: find a way to return nil instead of err here, nil currently panics
		return nil, errors.New("no base policy found")
	}
	basePolicyDict := basePolicy.(map[string]any)
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

func (a *mqlAzureSubscriptionNetworkServiceFirewallPolicy) childPolicies() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	props := a.Properties.Data
	propsDict := props.(map[string]any)
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
	res := []any{}
	childPoliciesList := childPolicies.([]any)
	for _, cp := range childPoliciesList {
		cpDict := cp.(map[string]any)
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

func (a *mqlAzureSubscriptionNetworkServiceFirewallPolicy) firewalls() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	props := a.Properties.Data
	propsDict := props.(map[string]any)
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
	res := []any{}
	firewallsList := firewalls.([]any)
	for _, fw := range firewallsList {
		fwDict := fw.(map[string]any)
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
	var sslPolicyType, sslMinProtocolVersion string
	sslCipherSuites := []any{}
	if ag.Properties != nil && ag.Properties.SSLPolicy != nil {
		sp := ag.Properties.SSLPolicy
		if sp.PolicyType != nil {
			sslPolicyType = string(*sp.PolicyType)
		}
		if sp.MinProtocolVersion != nil {
			sslMinProtocolVersion = string(*sp.MinProtocolVersion)
		}
		for _, cs := range sp.CipherSuites {
			if cs != nil {
				sslCipherSuites = append(sslCipherSuites, string(*cs))
			}
		}
	}

	args := map[string]*llx.RawData{
		"id":                    llx.StringDataPtr(ag.ID),
		"name":                  llx.StringDataPtr(ag.Name),
		"type":                  llx.StringDataPtr(ag.Type),
		"location":              llx.StringDataPtr(ag.Location),
		"tags":                  llx.MapData(convert.PtrMapStrToInterface(ag.Tags), types.String),
		"etag":                  llx.StringDataPtr(ag.Etag),
		"properties":            llx.DictData(props),
		"sslPolicyType":         llx.StringData(sslPolicyType),
		"sslMinProtocolVersion": llx.StringData(sslMinProtocolVersion),
		"sslCipherSuites":       llx.ArrayData(sslCipherSuites, types.String),
	}

	mqlAg, err := CreateResource(runtime, "azure.subscription.networkService.applicationGateway", args)
	if err != nil {
		return nil, err
	}

	return mqlAg.(*mqlAzureSubscriptionNetworkServiceApplicationGateway), nil
}

type mqlAzureSubscriptionNetworkServiceFirewallInternal struct {
	cacheProperties *network.AzureFirewallPropertiesFormat
}

func azureFirewallToMql(runtime *plugin.Runtime, fw network.AzureFirewall) (*mqlAzureSubscriptionNetworkServiceFirewall, error) {
	props, err := convert.JsonToDict(fw.Properties)
	if err != nil {
		return nil, err
	}
	var fwSkuTier, fwSkuName, fwProvisioningState, fwThreatIntelMode *string
	if fw.Properties != nil {
		fwProvisioningState = (*string)(fw.Properties.ProvisioningState)
		fwThreatIntelMode = (*string)(fw.Properties.ThreatIntelMode)
		if fw.Properties.SKU != nil {
			fwSkuTier = (*string)(fw.Properties.SKU.Tier)
			fwSkuName = (*string)(fw.Properties.SKU.Name)
		}
	}
	args := map[string]*llx.RawData{
		"id":                llx.StringDataPtr(fw.ID),
		"name":              llx.StringDataPtr(fw.Name),
		"type":              llx.StringDataPtr(fw.Type),
		"location":          llx.StringDataPtr(fw.Location),
		"tags":              llx.MapData(convert.PtrMapStrToInterface(fw.Tags), types.String),
		"etag":              llx.StringDataPtr(fw.Etag),
		"properties":        llx.DictData(props),
		"skuTier":           llx.StringDataPtr(fwSkuTier),
		"skuName":           llx.StringDataPtr(fwSkuName),
		"provisioningState": llx.StringDataPtr(fwProvisioningState),
		"threatIntelMode":   llx.StringDataPtr(fwThreatIntelMode),
	}
	mqlFw, err := CreateResource(runtime, "azure.subscription.networkService.firewall", args)
	if err != nil {
		return nil, err
	}
	fwRes := mqlFw.(*mqlAzureSubscriptionNetworkServiceFirewall)
	fwRes.cacheProperties = fw.Properties
	return fwRes, nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewall) ipConfigurations() ([]any, error) {
	if a.cacheProperties == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, ipConfig := range a.cacheProperties.IPConfigurations {
		props, err := convert.JsonToDict(ipConfig.Properties)
		if err != nil {
			return nil, err
		}
		var privateIP *string
		if ipConfig.Properties != nil {
			privateIP = ipConfig.Properties.PrivateIPAddress
		}
		mqlIpConfig, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.firewall.ipConfig",
			map[string]*llx.RawData{
				"id":               llx.StringDataPtr(ipConfig.ID),
				"name":             llx.StringDataPtr(ipConfig.Name),
				"etag":             llx.StringDataPtr(ipConfig.Etag),
				"privateIpAddress": llx.StringDataPtr(privateIP),
				"properties":       llx.DictData(props),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlIpConfig)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewall) managementIpConfiguration() (*mqlAzureSubscriptionNetworkServiceFirewallIpConfig, error) {
	if a.cacheProperties == nil || a.cacheProperties.ManagementIPConfiguration == nil {
		a.ManagementIpConfiguration.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	ipConfig := a.cacheProperties.ManagementIPConfiguration
	props, err := convert.JsonToDict(ipConfig.Properties)
	if err != nil {
		return nil, err
	}
	var privateIP *string
	if ipConfig.Properties != nil {
		privateIP = ipConfig.Properties.PrivateIPAddress
	}
	mqlIpConfig, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.firewall.ipConfig",
		map[string]*llx.RawData{
			"id":               llx.StringDataPtr(ipConfig.ID),
			"name":             llx.StringDataPtr(ipConfig.Name),
			"etag":             llx.StringDataPtr(ipConfig.Etag),
			"privateIpAddress": llx.StringDataPtr(privateIP),
			"properties":       llx.DictData(props),
		})
	if err != nil {
		return nil, err
	}
	return mqlIpConfig.(*mqlAzureSubscriptionNetworkServiceFirewallIpConfig), nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewall) natRules() ([]any, error) {
	if a.cacheProperties == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, natRule := range a.cacheProperties.NatRuleCollections {
		props, err := convert.JsonToDict(natRule.Properties)
		if err != nil {
			return nil, err
		}
		mqlNatRule, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.firewall.natRule",
			map[string]*llx.RawData{
				"id":         llx.StringDataPtr(natRule.ID),
				"name":       llx.StringDataPtr(natRule.Name),
				"etag":       llx.StringDataPtr(natRule.Etag),
				"properties": llx.DictData(props),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlNatRule)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewall) networkRules() ([]any, error) {
	if a.cacheProperties == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, networkRule := range a.cacheProperties.NetworkRuleCollections {
		props, err := convert.JsonToDict(networkRule.Properties)
		if err != nil {
			return nil, err
		}
		mqlNetworkRule, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.firewall.networkRule",
			map[string]*llx.RawData{
				"id":         llx.StringDataPtr(networkRule.ID),
				"name":       llx.StringDataPtr(networkRule.Name),
				"etag":       llx.StringDataPtr(networkRule.Etag),
				"properties": llx.DictData(props),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlNetworkRule)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewall) applicationRules() ([]any, error) {
	if a.cacheProperties == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, appRule := range a.cacheProperties.ApplicationRuleCollections {
		props, err := convert.JsonToDict(appRule.Properties)
		if err != nil {
			return nil, err
		}
		mqlAppRule, err := CreateResource(a.MqlRuntime, "azure.subscription.networkService.firewall.applicationRule",
			map[string]*llx.RawData{
				"id":         llx.StringDataPtr(appRule.ID),
				"name":       llx.StringDataPtr(appRule.Name),
				"etag":       llx.StringDataPtr(appRule.Etag),
				"properties": llx.DictData(props),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAppRule)
	}
	return res, nil
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

	mqlFwPolicy := mqlFw.(*mqlAzureSubscriptionNetworkServiceFirewallPolicy)
	if fwp.Properties != nil {
		mqlFwPolicy.cacheIntrusionDetection = fwp.Properties.IntrusionDetection
	}

	return mqlFwPolicy, nil
}

func azureIpToMql(runtime *plugin.Runtime, ip network.PublicIPAddress) (*mqlAzureSubscriptionNetworkServiceIpAddress, error) {
	var ipAllocationMethod, ipVersion, ddosProtectionMode string
	var ipAddr *string
	if ip.Properties != nil {
		ipAddr = ip.Properties.IPAddress
		if ip.Properties.PublicIPAllocationMethod != nil {
			ipAllocationMethod = string(*ip.Properties.PublicIPAllocationMethod)
		}
		if ip.Properties.PublicIPAddressVersion != nil {
			ipVersion = string(*ip.Properties.PublicIPAddressVersion)
		}
		if ip.Properties.DdosSettings != nil && ip.Properties.DdosSettings.ProtectionMode != nil {
			ddosProtectionMode = string(*ip.Properties.DdosSettings.ProtectionMode)
		}
	}
	zones := []any{}
	for _, z := range ip.Zones {
		if z != nil {
			zones = append(zones, *z)
		}
	}
	mqlAzure, err := CreateResource(runtime, "azure.subscription.networkService.ipAddress",
		map[string]*llx.RawData{
			"id":                 llx.StringDataPtr(ip.ID),
			"name":               llx.StringDataPtr(ip.Name),
			"location":           llx.StringDataPtr(ip.Location),
			"tags":               llx.MapData(convert.PtrMapStrToInterface(ip.Tags), types.String),
			"type":               llx.StringDataPtr(ip.Type),
			"ipAddress":          llx.StringDataPtr(ipAddr),
			"ipAllocationMethod": llx.StringData(ipAllocationMethod),
			"ipVersion":          llx.StringData(ipVersion),
			"zones":              llx.ArrayData(zones, types.String),
			"ddosProtectionMode": llx.StringData(ddosProtectionMode),
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

	var addressPrefix *llx.RawData
	var privateEndpointNetworkPolicies, privateLinkServiceNetworkPolicies *llx.RawData
	var defaultOutboundAccess *llx.RawData
	if subnet.Properties != nil {
		addressPrefix = llx.StringDataPtr(subnet.Properties.AddressPrefix)
		if subnet.Properties.PrivateEndpointNetworkPolicies != nil {
			privateEndpointNetworkPolicies = llx.StringData(string(*subnet.Properties.PrivateEndpointNetworkPolicies))
		} else {
			privateEndpointNetworkPolicies = llx.StringData("")
		}
		if subnet.Properties.PrivateLinkServiceNetworkPolicies != nil {
			privateLinkServiceNetworkPolicies = llx.StringData(string(*subnet.Properties.PrivateLinkServiceNetworkPolicies))
		} else {
			privateLinkServiceNetworkPolicies = llx.StringData("")
		}
		defaultOutboundAccess = llx.BoolDataPtr(subnet.Properties.DefaultOutboundAccess)
	} else {
		addressPrefix = llx.StringData("")
		privateEndpointNetworkPolicies = llx.StringData("")
		privateLinkServiceNetworkPolicies = llx.StringData("")
		defaultOutboundAccess = llx.BoolData(false)
	}

	mqlAzure, err := CreateResource(runtime, "azure.subscription.networkService.subnet",
		map[string]*llx.RawData{
			"id":                                llx.StringDataPtr(subnet.ID),
			"name":                              llx.StringDataPtr(subnet.Name),
			"type":                              llx.StringDataPtr(subnet.Type),
			"etag":                              llx.StringDataPtr(subnet.Etag),
			"addressPrefix":                     addressPrefix,
			"properties":                        llx.DictData(props),
			"privateEndpointNetworkPolicies":    privateEndpointNetworkPolicies,
			"privateLinkServiceNetworkPolicies": privateLinkServiceNetworkPolicies,
			"defaultOutboundAccess":             defaultOutboundAccess,
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

	var enableIPForwarding, enableAcceleratedNetworking, primary *llx.RawData
	var networkSecurityGroupId string
	ipConfigs := []any{}
	if iface.Properties != nil {
		enableIPForwarding = llx.BoolDataPtr(iface.Properties.EnableIPForwarding)
		enableAcceleratedNetworking = llx.BoolDataPtr(iface.Properties.EnableAcceleratedNetworking)
		primary = llx.BoolDataPtr(iface.Properties.Primary)
		if iface.Properties.NetworkSecurityGroup != nil && iface.Properties.NetworkSecurityGroup.ID != nil {
			networkSecurityGroupId = *iface.Properties.NetworkSecurityGroup.ID
		}
		for _, ipConfig := range iface.Properties.IPConfigurations {
			if ipConfig == nil {
				continue
			}
			configDict := map[string]any{}
			if ipConfig.Name != nil {
				configDict["name"] = *ipConfig.Name
			}
			if ipConfig.ID != nil {
				configDict["id"] = *ipConfig.ID
			}
			if ipConfig.Properties != nil {
				if ipConfig.Properties.PrivateIPAddress != nil {
					configDict["privateIpAddress"] = *ipConfig.Properties.PrivateIPAddress
				}
				if ipConfig.Properties.PrivateIPAllocationMethod != nil {
					configDict["privateIpAllocationMethod"] = string(*ipConfig.Properties.PrivateIPAllocationMethod)
				}
				if ipConfig.Properties.Primary != nil {
					configDict["primary"] = *ipConfig.Properties.Primary
				}
				if ipConfig.Properties.PublicIPAddress != nil && ipConfig.Properties.PublicIPAddress.ID != nil {
					configDict["publicIpAddressId"] = *ipConfig.Properties.PublicIPAddress.ID
				}
				if ipConfig.Properties.Subnet != nil && ipConfig.Properties.Subnet.ID != nil {
					configDict["subnetId"] = *ipConfig.Properties.Subnet.ID
				}
			}
			ipConfigs = append(ipConfigs, configDict)
		}
	} else {
		enableIPForwarding = llx.BoolData(false)
		enableAcceleratedNetworking = llx.BoolData(false)
		primary = llx.BoolData(false)
	}

	res, err := CreateResource(runtime, "azure.subscription.networkService.interface",
		map[string]*llx.RawData{
			"id":                          llx.StringDataPtr(iface.ID),
			"name":                        llx.StringDataPtr(iface.Name),
			"location":                    llx.StringDataPtr(iface.Location),
			"tags":                        llx.MapData(convert.PtrMapStrToInterface(iface.Tags), types.String),
			"type":                        llx.StringDataPtr(iface.Type),
			"etag":                        llx.StringDataPtr(iface.Etag),
			"properties":                  llx.DictData(properties),
			"enableIPForwarding":          enableIPForwarding,
			"enableAcceleratedNetworking": enableAcceleratedNetworking,
			"primary":                     primary,
			"networkSecurityGroupId":      llx.StringData(networkSecurityGroupId),
			"ipConfigurations":            llx.ArrayData(ipConfigs, types.Dict),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionNetworkServiceInterface), nil
}

// see https://github.com/Azure/azure-sdk-for-go/issues/8224
type AzureSecurityGroupPropertiesFormat network.SecurityGroupPropertiesFormat

type mqlAzureSubscriptionNetworkServiceSecurityGroupInternal struct {
	cacheProperties *network.SecurityGroupPropertiesFormat
}

func azureSecGroupToMql(runtime *plugin.Runtime, secGroup network.SecurityGroup) (*mqlAzureSubscriptionNetworkServiceSecurityGroup, error) {
	var properties map[string]any
	var err error
	if secGroup.Properties != nil {
		// avoid using the azure sdk SecurityGroupPropertiesFormat MarshalJSON
		var j AzureSecurityGroupPropertiesFormat
		j = AzureSecurityGroupPropertiesFormat(*secGroup.Properties)

		properties, err = convert.JsonToDict(j)
		if err != nil {
			return nil, err
		}
	}
	res, err := CreateResource(runtime, "azure.subscription.networkService.securityGroup",
		map[string]*llx.RawData{
			"id":         llx.StringDataPtr(secGroup.ID),
			"name":       llx.StringDataPtr(secGroup.Name),
			"location":   llx.StringDataPtr(secGroup.Location),
			"tags":       llx.MapData(convert.PtrMapStrToInterface(secGroup.Tags), types.String),
			"type":       llx.StringDataPtr(secGroup.Type),
			"etag":       llx.StringDataPtr(secGroup.Etag),
			"properties": llx.DictData(properties),
		})
	if err != nil {
		return nil, err
	}
	mqlSecGroup := res.(*mqlAzureSubscriptionNetworkServiceSecurityGroup)
	mqlSecGroup.cacheProperties = secGroup.Properties
	return mqlSecGroup, nil
}

func (a *mqlAzureSubscriptionNetworkServiceSecurityGroup) interfaces() ([]any, error) {
	if a.cacheProperties == nil || a.cacheProperties.NetworkInterfaces == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, iface := range a.cacheProperties.NetworkInterfaces {
		if iface != nil {
			mqlIface, err := azureInterfaceToMql(a.MqlRuntime, *iface)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlIface)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceSecurityGroup) securityRules() ([]any, error) {
	if a.cacheProperties == nil || a.cacheProperties.SecurityRules == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, secRule := range a.cacheProperties.SecurityRules {
		if secRule != nil {
			mqlRule, err := azureSecurityRuleToMql(a.MqlRuntime, *secRule)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRule)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceSecurityGroup) defaultSecurityRules() ([]any, error) {
	if a.cacheProperties == nil || a.cacheProperties.DefaultSecurityRules == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, secRule := range a.cacheProperties.DefaultSecurityRules {
		if secRule != nil {
			mqlRule, err := azureSecurityRuleToMql(a.MqlRuntime, *secRule)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRule)
		}
	}
	return res, nil
}

func azureSecurityRuleToMql(runtime *plugin.Runtime, secRule network.SecurityRule) (*mqlAzureSubscriptionNetworkServiceSecurityrule, error) {
	properties, err := convert.JsonToDict(secRule.Properties)
	if err != nil {
		return nil, err
	}

	destinationPortRange := []any{}

	if secRule.Properties != nil && secRule.Properties.DestinationPortRange != nil {
		dPortRange := parseAzureSecurityRulePortRange(*secRule.Properties.DestinationPortRange)
		for i := range dPortRange {
			destinationPortRange = append(destinationPortRange, map[string]any{
				"fromPort": dPortRange[i].FromPort,
				"toPort":   dPortRange[i].ToPort,
			})
		}
	}

	if secRule.Properties != nil && secRule.Properties.DestinationPortRanges != nil {
		for _, r := range secRule.Properties.DestinationPortRanges {
			dPortRange := parseAzureSecurityRulePortRange(*r)
			for i := range dPortRange {
				destinationPortRange = append(destinationPortRange, map[string]any{
					"fromPort": dPortRange[i].FromPort,
					"toPort":   dPortRange[i].ToPort,
				})
			}
		}
	}

	var direction, protocol, access, sourcePortRange, sourceAddressPrefix, destinationAddressPrefix, description *llx.RawData
	var priority *llx.RawData
	if secRule.Properties != nil {
		direction = llx.StringDataPtr((*string)(secRule.Properties.Direction))
		if secRule.Properties.Protocol != nil {
			protocol = llx.StringData(string(*secRule.Properties.Protocol))
		} else {
			protocol = llx.StringData("")
		}
		if secRule.Properties.Access != nil {
			access = llx.StringData(string(*secRule.Properties.Access))
		} else {
			access = llx.StringData("")
		}
		priority = llx.IntDataDefault(secRule.Properties.Priority, 0)
		sourcePortRange = llx.StringDataPtr(secRule.Properties.SourcePortRange)
		sourceAddressPrefix = llx.StringDataPtr(secRule.Properties.SourceAddressPrefix)
		destinationAddressPrefix = llx.StringDataPtr(secRule.Properties.DestinationAddressPrefix)
		description = llx.StringDataPtr(secRule.Properties.Description)
	} else {
		direction = llx.StringData("")
		protocol = llx.StringData("")
		access = llx.StringData("")
		priority = llx.IntData(0)
		sourcePortRange = llx.StringData("")
		sourceAddressPrefix = llx.StringData("")
		destinationAddressPrefix = llx.StringData("")
		description = llx.StringData("")
	}

	res, err := CreateResource(runtime, "azure.subscription.networkService.securityrule",
		map[string]*llx.RawData{
			"id":                       llx.StringDataPtr(secRule.ID),
			"name":                     llx.StringDataPtr(secRule.Name),
			"etag":                     llx.StringDataPtr(secRule.Etag),
			"direction":                direction,
			"properties":               llx.DictData(properties),
			"destinationPortRange":     llx.ArrayData(destinationPortRange, types.String),
			"protocol":                 protocol,
			"access":                   access,
			"priority":                 priority,
			"sourcePortRange":          sourcePortRange,
			"sourceAddressPrefix":      sourceAddressPrefix,
			"destinationAddressPrefix": destinationAddressPrefix,
			"description":              description,
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

// --- DDoS Protection Plans ---

type mqlAzureSubscriptionNetworkServiceDdosProtectionPlanInternal struct {
	cacheVnetIds     []string
	cachePublicIpIds []string
}

func (a *mqlAzureSubscriptionNetworkService) ddosProtectionPlans() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewDdosProtectionPlansClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list DDoS protection plans due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, plan := range page.Value {
			if plan == nil {
				continue
			}

			var provisioningState string
			var vnetIds []string
			var publicIpIds []string
			if plan.Properties != nil {
				if plan.Properties.ProvisioningState != nil {
					provisioningState = string(*plan.Properties.ProvisioningState)
				}
				for _, vn := range plan.Properties.VirtualNetworks {
					if vn != nil && vn.ID != nil {
						vnetIds = append(vnetIds, *vn.ID)
					}
				}
				for _, pip := range plan.Properties.PublicIPAddresses {
					if pip != nil && pip.ID != nil {
						publicIpIds = append(publicIpIds, *pip.ID)
					}
				}
			}

			mqlPlan, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionNetworkServiceDdosProtectionPlan,
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(plan.ID),
					"name":              llx.StringDataPtr(plan.Name),
					"location":          llx.StringDataPtr(plan.Location),
					"tags":              llx.MapData(convert.PtrMapStrToInterface(plan.Tags), types.String),
					"type":              llx.StringDataPtr(plan.Type),
					"etag":              llx.StringDataPtr(plan.Etag),
					"provisioningState": llx.StringData(provisioningState),
				})
			if err != nil {
				return nil, err
			}
			mqlDdos := mqlPlan.(*mqlAzureSubscriptionNetworkServiceDdosProtectionPlan)
			mqlDdos.cacheVnetIds = vnetIds
			mqlDdos.cachePublicIpIds = publicIpIds
			res = append(res, mqlDdos)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceDdosProtectionPlan) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceDdosProtectionPlan) virtualNetworks() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()

	// Group VNet IDs by subscription to reuse clients.
	type vnetRef struct {
		rgName   string
		vnetName string
		id       string
	}
	bySubscription := map[string][]vnetRef{}
	for _, vnetId := range a.cacheVnetIds {
		resourceID, err := ParseResourceID(vnetId)
		if err != nil {
			log.Warn().Err(err).Str("id", vnetId).Msg("could not parse virtual network resource ID")
			continue
		}
		bySubscription[resourceID.SubscriptionID] = append(bySubscription[resourceID.SubscriptionID], vnetRef{
			rgName:   resourceID.ResourceGroup,
			vnetName: resourceID.Path["virtualnetworks"],
			id:       vnetId,
		})
	}

	res := []any{}
	for subId, refs := range bySubscription {
		client, err := network.NewVirtualNetworksClient(subId, token, &arm.ClientOptions{
			ClientOptions: conn.ClientOptions(),
		})
		if err != nil {
			return nil, err
		}

		for _, ref := range refs {
			resp, err := client.Get(ctx, ref.rgName, ref.vnetName, nil)
			if err != nil {
				log.Warn().Err(err).Str("id", ref.id).Msg("could not get virtual network for DDoS protection plan")
				continue
			}

			mqlVn, err := azureVirtualNetworkToMql(a.MqlRuntime, resp.VirtualNetwork)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlVn)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceDdosProtectionPlan) publicIpAddresses() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()

	res := []any{}
	for _, pipId := range a.cachePublicIpIds {
		resourceID, err := ParseResourceID(pipId)
		if err != nil {
			log.Warn().Err(err).Str("id", pipId).Msg("could not parse public IP address resource ID")
			continue
		}
		rgName := resourceID.ResourceGroup
		ipName := resourceID.Path["publicipaddresses"]

		client, err := network.NewPublicIPAddressesClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
			ClientOptions: conn.ClientOptions(),
		})
		if err != nil {
			return nil, err
		}

		resp, err := client.Get(ctx, rgName, ipName, nil)
		if err != nil {
			log.Warn().Err(err).Str("id", pipId).Msg("could not get public IP address for DDoS protection plan")
			continue
		}

		mqlIp, err := azureIpToMql(a.MqlRuntime, resp.PublicIPAddress)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlIp)
	}
	return res, nil
}

// --- Service Endpoint Policies ---

type mqlAzureSubscriptionNetworkServiceServiceEndpointPolicyInternal struct {
	cacheSubnets []network.Subnet
}

func (a *mqlAzureSubscriptionNetworkService) serviceEndpointPolicies() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data

	client, err := network.NewServiceEndpointPoliciesClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden {
				log.Warn().Err(err).Msg("could not list service endpoint policies due to access denied")
				return res, nil
			}
			return nil, err
		}
		for _, policy := range page.Value {
			if policy == nil {
				continue
			}

			var provisioningState, serviceAlias string
			definitions := []any{}
			var cachedSubnets []network.Subnet
			if policy.Properties != nil {
				if policy.Properties.ProvisioningState != nil {
					provisioningState = string(*policy.Properties.ProvisioningState)
				}
				if policy.Properties.ServiceAlias != nil {
					serviceAlias = *policy.Properties.ServiceAlias
				}
				for _, def := range policy.Properties.ServiceEndpointPolicyDefinitions {
					if def == nil {
						continue
					}
					var defDescription, defService, defProvisioningState string
					var defServiceResources []any
					if def.Properties != nil {
						if def.Properties.Description != nil {
							defDescription = *def.Properties.Description
						}
						if def.Properties.Service != nil {
							defService = *def.Properties.Service
						}
						if def.Properties.ProvisioningState != nil {
							defProvisioningState = string(*def.Properties.ProvisioningState)
						}
						defServiceResources = convert.SliceStrPtrToInterface(def.Properties.ServiceResources)
					}
					mqlDef, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionNetworkServiceServiceEndpointPolicyDefinition,
						map[string]*llx.RawData{
							"id":                llx.StringDataPtr(def.ID),
							"name":              llx.StringDataPtr(def.Name),
							"type":              llx.StringDataPtr(def.Type),
							"etag":              llx.StringDataPtr(def.Etag),
							"description":       llx.StringData(defDescription),
							"service":           llx.StringData(defService),
							"serviceResources":  llx.ArrayData(defServiceResources, types.String),
							"provisioningState": llx.StringData(defProvisioningState),
						})
					if err != nil {
						return nil, err
					}
					definitions = append(definitions, mqlDef)
				}
				for _, subnet := range policy.Properties.Subnets {
					if subnet != nil {
						cachedSubnets = append(cachedSubnets, *subnet)
					}
				}
			}

			mqlPolicy, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionNetworkServiceServiceEndpointPolicy,
				map[string]*llx.RawData{
					"id":                llx.StringDataPtr(policy.ID),
					"name":              llx.StringDataPtr(policy.Name),
					"location":          llx.StringDataPtr(policy.Location),
					"tags":              llx.MapData(convert.PtrMapStrToInterface(policy.Tags), types.String),
					"type":              llx.StringDataPtr(policy.Type),
					"etag":              llx.StringDataPtr(policy.Etag),
					"kind":              llx.StringDataPtr(policy.Kind),
					"provisioningState": llx.StringData(provisioningState),
					"serviceAlias":      llx.StringData(serviceAlias),
					"definitions":       llx.ArrayData(definitions, types.ResourceLike),
				})
			if err != nil {
				return nil, err
			}
			mqlSep := mqlPolicy.(*mqlAzureSubscriptionNetworkServiceServiceEndpointPolicy)
			mqlSep.cacheSubnets = cachedSubnets
			res = append(res, mqlSep)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceServiceEndpointPolicy) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceServiceEndpointPolicyDefinition) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionNetworkServiceServiceEndpointPolicy) subnets() ([]any, error) {
	res := []any{}
	for _, subnet := range a.cacheSubnets {
		mqlSubnet, err := azureSubnetToMql(a.MqlRuntime, subnet)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSubnet)
	}
	return res, nil
}

// --- Firewall Policy IDPS ---

type mqlAzureSubscriptionNetworkServiceFirewallPolicyInternal struct {
	cacheIntrusionDetection *network.FirewallPolicyIntrusionDetection
}

func (a *mqlAzureSubscriptionNetworkServiceFirewallPolicy) intrusionDetectionMode() (string, error) {
	if a.cacheIntrusionDetection == nil || a.cacheIntrusionDetection.Mode == nil {
		return "", nil
	}
	return string(*a.cacheIntrusionDetection.Mode), nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewallPolicy) intrusionDetectionProfile() (string, error) {
	if a.cacheIntrusionDetection == nil || a.cacheIntrusionDetection.Profile == nil {
		return "", nil
	}
	return string(*a.cacheIntrusionDetection.Profile), nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewallPolicy) intrusionDetectionSignatureOverrides() ([]any, error) {
	if a.cacheIntrusionDetection == nil || a.cacheIntrusionDetection.Configuration == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, sig := range a.cacheIntrusionDetection.Configuration.SignatureOverrides {
		if sig == nil {
			continue
		}
		entry := map[string]any{}
		if sig.ID != nil {
			entry["id"] = *sig.ID
		}
		if sig.Mode != nil {
			entry["mode"] = string(*sig.Mode)
		}
		res = append(res, entry)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewallPolicy) intrusionDetectionBypassRules() ([]any, error) {
	if a.cacheIntrusionDetection == nil || a.cacheIntrusionDetection.Configuration == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, rule := range a.cacheIntrusionDetection.Configuration.BypassTrafficSettings {
		if rule == nil {
			continue
		}
		var protocol string
		if rule.Protocol != nil {
			protocol = string(*rule.Protocol)
		}

		id := a.Id.Data + "/intrusionDetection/bypassRules/" + convert.ToValue(rule.Name)
		mqlRule, err := CreateResource(a.MqlRuntime, ResourceAzureSubscriptionNetworkServiceFirewallPolicyIdpsBypassRule,
			map[string]*llx.RawData{
				"id":                   llx.StringData(id),
				"name":                 llx.StringDataPtr(rule.Name),
				"description":          llx.StringDataPtr(rule.Description),
				"protocol":             llx.StringData(protocol),
				"sourceAddresses":      llx.ArrayData(convert.SliceStrPtrToInterface(rule.SourceAddresses), types.String),
				"sourceIpGroups":       llx.ArrayData(convert.SliceStrPtrToInterface(rule.SourceIPGroups), types.String),
				"destinationAddresses": llx.ArrayData(convert.SliceStrPtrToInterface(rule.DestinationAddresses), types.String),
				"destinationIpGroups":  llx.ArrayData(convert.SliceStrPtrToInterface(rule.DestinationIPGroups), types.String),
				"destinationPorts":     llx.ArrayData(convert.SliceStrPtrToInterface(rule.DestinationPorts), types.String),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRule)
	}
	return res, nil
}

func (a *mqlAzureSubscriptionNetworkServiceFirewallPolicyIdpsBypassRule) id() (string, error) {
	return a.Id.Data, nil
}
