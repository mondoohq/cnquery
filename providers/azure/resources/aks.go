// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	clusters "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v2"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/azure/connection"
	"go.mondoo.com/cnquery/v10/types"
)

func (a *mqlAzureSubscriptionAksService) id() (string, error) {
	return "azure.subscription.aks/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionAks(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}

	conn := runtime.Connection.(*connection.AzureConnection)
	args["subscriptionId"] = llx.StringData(conn.SubId())

	return args, nil, nil
}

func (a *mqlAzureSubscriptionAksServiceCluster) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionAksService) clusters() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()
	subId := a.SubscriptionId.Data
	client, err := clusters.NewManagedClustersClient(subId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}
	pager := client.NewListPager(&clusters.ManagedClustersClientListOptions{})
	res := []interface{}{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			storageProfile, err := convert.JsonToDict(entry.Properties.StorageProfile)
			if err != nil {
				return nil, err
			}
			workloadAutoScalerProfile, err := convert.JsonToDict(entry.Properties.WorkloadAutoScalerProfile)
			if err != nil {
				return nil, err
			}
			securityProfile, err := convert.JsonToDict(entry.Properties.SecurityProfile)
			if err != nil {
				return nil, err
			}
			podIdentityProfile, err := convert.JsonToDict(entry.Properties.PodIdentityProfile)
			if err != nil {
				return nil, err
			}
			networkProfile, err := convert.JsonToDict(entry.Properties.NetworkProfile)
			if err != nil {
				return nil, err
			}
			httpProxyConfig, err := convert.JsonToDict(entry.Properties.HTTPProxyConfig)
			if err != nil {
				return nil, err
			}
			addonProfiles := []interface{}{}
			for k, a := range entry.Properties.AddonProfiles {
				dict, err := convert.JsonToDict(a)
				if err != nil {
					return nil, err
				}
				m := map[string]interface{}{}
				m[k] = dict
				addonProfiles = append(addonProfiles, m)
			}
			if err != nil {
				return nil, err
			}
			agentPoolProfiles, err := convert.JsonToDictSlice(entry.Properties.AgentPoolProfiles)
			if err != nil {
				return nil, err
			}

			var createdAt *time.Time
			if entry.SystemData != nil {
				createdAt = entry.SystemData.CreatedAt
			}

			mqlAksCluster, err := CreateResource(a.MqlRuntime, "azure.subscription.aksService.cluster",
				map[string]*llx.RawData{
					"id":                        llx.StringData(convert.ToString(entry.ID)),
					"name":                      llx.StringData(convert.ToString(entry.Name)),
					"location":                  llx.StringData(convert.ToString(entry.Location)),
					"kubernetesVersion":         llx.StringData(convert.ToString(entry.Properties.KubernetesVersion)),
					"provisioningState":         llx.StringData(convert.ToString(entry.Properties.ProvisioningState)),
					"createdAt":                 llx.TimeDataPtr(createdAt),
					"nodeResourceGroup":         llx.StringData(convert.ToString(entry.Properties.NodeResourceGroup)),
					"powerState":                llx.StringData(convert.ToString((*string)(entry.Properties.PowerState.Code))),
					"tags":                      llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
					"rbacEnabled":               llx.BoolData(convert.ToBool(entry.Properties.EnableRBAC)),
					"dnsPrefix":                 llx.StringData(convert.ToString(entry.Properties.DNSPrefix)),
					"fqdn":                      llx.StringData(convert.ToString(entry.Properties.Fqdn)),
					"agentPoolProfiles":         llx.DictData(agentPoolProfiles),
					"addonProfiles":             llx.DictData(addonProfiles),
					"httpProxyConfig":           llx.DictData(httpProxyConfig),
					"networkProfile":            llx.DictData(networkProfile),
					"podIdentityProfile":        llx.DictData(podIdentityProfile),
					"securityProfile":           llx.DictData(securityProfile),
					"storageProfile":            llx.DictData(storageProfile),
					"workloadAutoScalerProfile": llx.DictData(workloadAutoScalerProfile),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAksCluster)
		}
	}
	return res, nil
}
