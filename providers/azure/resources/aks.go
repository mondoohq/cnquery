// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	clusters "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v8"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAzureSubscriptionAksService) id() (string, error) {
	return "azure.subscription.aks/" + a.SubscriptionId.Data, nil
}

func initAzureSubscriptionAksService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func initAzureSubscriptionAksServiceCluster(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["id"] = llx.StringData(ids.id)
		}
	}

	if args["id"] == nil {
		return nil, nil, errors.New("id required to fetch azure aks cluster")
	}
	conn, ok := runtime.Connection.(*connection.AzureConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not an Azure connection")
	}
	res, err := NewResource(runtime, "azure.subscription.aksService", map[string]*llx.RawData{
		"subscriptionId": llx.StringData(conn.SubId()),
	})
	if err != nil {
		return nil, nil, err
	}
	aksSvc := res.(*mqlAzureSubscriptionAksService)
	clusterList := aksSvc.GetClusters()
	if clusterList.Error != nil {
		return nil, nil, clusterList.Error
	}
	id, ok := args["id"].Value.(string)
	if !ok {
		return nil, nil, errors.New("id must be a non-nil string value")
	}
	for _, entry := range clusterList.Data {
		cluster := entry.(*mqlAzureSubscriptionAksServiceCluster)
		if cluster.Id.Data == id {
			return args, cluster, nil
		}
	}

	return nil, nil, errors.New("azure aks cluster does not exist")
}

func (a *mqlAzureSubscriptionAksServiceCluster) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionAksServiceClusterAadProfile) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionAksServiceClusterAutoUpgradeProfile) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionAksService) clusters() ([]any, error) {
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
	res := []any{}
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
			apiServerAccessProfile, err := convert.JsonToDict(entry.Properties.APIServerAccessProfile)
			if err != nil {
				return nil, err
			}
			addonProfiles := []any{}
			for k, a := range entry.Properties.AddonProfiles {
				dict, err := convert.JsonToDict(a)
				if err != nil {
					return nil, err
				}
				m := map[string]any{}
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

			var enablePrivateCluster *bool
			var enablePrivateClusterPublicFQDN *bool
			var disableRunCommand *bool
			var privateDnsZone *string
			apiServerAuthorizedIPRanges := []any{}
			if entry.Properties.APIServerAccessProfile != nil {
				asp := entry.Properties.APIServerAccessProfile
				enablePrivateCluster = asp.EnablePrivateCluster
				enablePrivateClusterPublicFQDN = asp.EnablePrivateClusterPublicFQDN
				disableRunCommand = asp.DisableRunCommand
				privateDnsZone = asp.PrivateDNSZone
				for _, r := range asp.AuthorizedIPRanges {
					if r != nil {
						apiServerAuthorizedIPRanges = append(apiServerAuthorizedIPRanges, *r)
					}
				}
			}

			var defenderEnabled, imageCleanerEnabled, workloadIdentityEnabled, azureKeyVaultKmsEnabled *bool
			var imageCleanerIntervalHours *int32
			var azureKeyVaultKmsNetworkAccess *string
			if entry.Properties.SecurityProfile != nil {
				sp := entry.Properties.SecurityProfile
				if sp.Defender != nil && sp.Defender.SecurityMonitoring != nil {
					defenderEnabled = sp.Defender.SecurityMonitoring.Enabled
				}
				if sp.ImageCleaner != nil {
					imageCleanerEnabled = sp.ImageCleaner.Enabled
					imageCleanerIntervalHours = sp.ImageCleaner.IntervalHours
				}
				if sp.WorkloadIdentity != nil {
					workloadIdentityEnabled = sp.WorkloadIdentity.Enabled
				}
				if sp.AzureKeyVaultKms != nil {
					azureKeyVaultKmsEnabled = sp.AzureKeyVaultKms.Enabled
					azureKeyVaultKmsNetworkAccess = (*string)(sp.AzureKeyVaultKms.KeyVaultNetworkAccess)
				}
			}

			// Create AAD Profile sub-resource
			var aadProfileData *llx.RawData = llx.NilData
			if entry.Properties.AADProfile != nil {
				aadP := entry.Properties.AADProfile
				adminGroupObjectIDs := []any{}
				for _, gid := range aadP.AdminGroupObjectIDs {
					if gid != nil {
						adminGroupObjectIDs = append(adminGroupObjectIDs, *gid)
					}
				}
				aadRes, err := CreateResource(a.MqlRuntime, "azure.subscription.aksService.cluster.aadProfile",
					map[string]*llx.RawData{
						"id":                  llx.StringData(*entry.ID + "/aadProfile"),
						"managed":             llx.BoolDataPtr(aadP.Managed),
						"enableAzureRBAC":     llx.BoolDataPtr(aadP.EnableAzureRBAC),
						"adminGroupObjectIDs": llx.ArrayData(adminGroupObjectIDs, types.String),
					})
				if err != nil {
					return nil, err
				}
				aadProfileData = llx.ResourceData(aadRes, "azure.subscription.aksService.cluster.aadProfile")
			}

			// Create Auto-Upgrade Profile sub-resource
			var autoUpgradeProfileData *llx.RawData = llx.NilData
			if entry.Properties.AutoUpgradeProfile != nil {
				aup := entry.Properties.AutoUpgradeProfile
				autoUpgradeRes, err := CreateResource(a.MqlRuntime, "azure.subscription.aksService.cluster.autoUpgradeProfile",
					map[string]*llx.RawData{
						"id":                   llx.StringData(*entry.ID + "/autoUpgradeProfile"),
						"upgradeChannel":       llx.StringDataPtr((*string)(aup.UpgradeChannel)),
						"nodeOSUpgradeChannel": llx.StringDataPtr((*string)(aup.NodeOSUpgradeChannel)),
					})
				if err != nil {
					return nil, err
				}
				autoUpgradeProfileData = llx.ResourceData(autoUpgradeRes, "azure.subscription.aksService.cluster.autoUpgradeProfile")
			}

			mqlAksCluster, err := CreateResource(a.MqlRuntime, "azure.subscription.aksService.cluster",
				map[string]*llx.RawData{
					"id":                             llx.StringDataPtr(entry.ID),
					"name":                           llx.StringDataPtr(entry.Name),
					"location":                       llx.StringDataPtr(entry.Location),
					"kubernetesVersion":              llx.StringDataPtr(entry.Properties.KubernetesVersion),
					"provisioningState":              llx.StringDataPtr(entry.Properties.ProvisioningState),
					"createdAt":                      llx.TimeDataPtr(createdAt),
					"nodeResourceGroup":              llx.StringDataPtr(entry.Properties.NodeResourceGroup),
					"powerState":                     llx.StringDataPtr((*string)(entry.Properties.PowerState.Code)),
					"tags":                           llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
					"rbacEnabled":                    llx.BoolDataPtr(entry.Properties.EnableRBAC),
					"dnsPrefix":                      llx.StringDataPtr(entry.Properties.DNSPrefix),
					"fqdn":                           llx.StringDataPtr(entry.Properties.Fqdn),
					"agentPoolProfiles":              llx.DictData(agentPoolProfiles),
					"addonProfiles":                  llx.DictData(addonProfiles),
					"httpProxyConfig":                llx.DictData(httpProxyConfig),
					"networkProfile":                 llx.DictData(networkProfile),
					"podIdentityProfile":             llx.DictData(podIdentityProfile),
					"securityProfile":                llx.DictData(securityProfile),
					"storageProfile":                 llx.DictData(storageProfile),
					"workloadAutoScalerProfile":      llx.DictData(workloadAutoScalerProfile),
					"apiServerAccessProfile":         llx.DictData(apiServerAccessProfile),
					"enablePrivateCluster":           llx.BoolDataPtr(enablePrivateCluster),
					"enablePrivateClusterPublicFQDN": llx.BoolDataPtr(enablePrivateClusterPublicFQDN),
					"disableRunCommand":              llx.BoolDataPtr(disableRunCommand),
					"apiServerAuthorizedIPRanges":    llx.ArrayData(apiServerAuthorizedIPRanges, types.String),
					"privateDnsZone":                 llx.StringDataPtr(privateDnsZone),
					"defenderEnabled":                llx.BoolDataPtr(defenderEnabled),
					"imageCleanerEnabled":            llx.BoolDataPtr(imageCleanerEnabled),
					"imageCleanerIntervalHours":      llx.IntDataDefault(imageCleanerIntervalHours, 0),
					"workloadIdentityEnabled":        llx.BoolDataPtr(workloadIdentityEnabled),
					"azureKeyVaultKmsEnabled":        llx.BoolDataPtr(azureKeyVaultKmsEnabled),
					"azureKeyVaultKmsNetworkAccess":  llx.StringDataPtr(azureKeyVaultKmsNetworkAccess),
					"disableLocalAccounts":           llx.BoolDataPtr(entry.Properties.DisableLocalAccounts),
					"publicNetworkAccess":            llx.StringDataPtr((*string)(entry.Properties.PublicNetworkAccess)),
					"aadProfile":                     aadProfileData,
					"autoUpgradeProfile":             autoUpgradeProfileData,
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAksCluster)
		}
	}
	return res, nil
}
