// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	clusters "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v9"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/azure/connection"
	"go.mondoo.com/mql/v13/types"
)

type mqlAzureSubscriptionAksServiceClusterInternal struct {
	cacheKmsKeyId   string
	cacheProperties *clusters.ManagedClusterProperties
}

type mqlAzureSubscriptionAksServiceClusterIdentityBindingInternal struct {
	cacheManagedIdentityId string
}

type mqlAzureSubscriptionAksServiceClusterNodePoolInternal struct {
	subscriptionId string
	resourceGroup  string
	clusterName    string
	poolName       string
}

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

func (a *mqlAzureSubscriptionAksServiceClusterAdvancedNetworking) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionAksServiceClusterNodePool) id() (string, error) {
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
			var azureKeyVaultKmsKeyId string
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
					if sp.AzureKeyVaultKms.KeyID != nil {
						azureKeyVaultKmsKeyId = *sp.AzureKeyVaultKms.KeyID
					}
				}
			}

			var networkPlugin, networkPolicy *string
			if entry.Properties.NetworkProfile != nil {
				np := entry.Properties.NetworkProfile
				networkPlugin = (*string)(np.NetworkPlugin)
				networkPolicy = (*string)(np.NetworkPolicy)
			}

			var oidcIssuerEnabled *bool
			if entry.Properties.OidcIssuerProfile != nil {
				oidcIssuerEnabled = entry.Properties.OidcIssuerProfile.Enabled
			}

			var nodeResourceGroupRestrictionLevel *string
			if entry.Properties.NodeResourceGroupProfile != nil {
				nodeResourceGroupRestrictionLevel = (*string)(entry.Properties.NodeResourceGroupProfile.RestrictionLevel)
			}

			var serviceMeshMode *string
			if entry.Properties.ServiceMeshProfile != nil {
				serviceMeshMode = (*string)(entry.Properties.ServiceMeshProfile.Mode)
			}

			var skuTier string
			if entry.SKU != nil && entry.SKU.Tier != nil {
				skuTier = string(*entry.SKU.Tier)
			}

			var powerState *string
			if entry.Properties.PowerState != nil {
				powerState = (*string)(entry.Properties.PowerState.Code)
			}

			var controlPlaneMetricsEnabled *bool
			if amp := entry.Properties.AzureMonitorProfile; amp != nil && amp.Metrics != nil && amp.Metrics.ControlPlane != nil {
				controlPlaneMetricsEnabled = amp.Metrics.ControlPlane.Enabled
			}

			mqlAksCluster, err := CreateResource(a.MqlRuntime, "azure.subscription.aksService.cluster",
				map[string]*llx.RawData{
					"id":                                llx.StringDataPtr(entry.ID),
					"name":                              llx.StringDataPtr(entry.Name),
					"location":                          llx.StringDataPtr(entry.Location),
					"kubernetesVersion":                 llx.StringDataPtr(entry.Properties.KubernetesVersion),
					"provisioningState":                 llx.StringDataPtr(entry.Properties.ProvisioningState),
					"createdAt":                         llx.TimeDataPtr(createdAt),
					"nodeResourceGroup":                 llx.StringDataPtr(entry.Properties.NodeResourceGroup),
					"powerState":                        llx.StringDataPtr(powerState),
					"tags":                              llx.MapData(convert.PtrMapStrToInterface(entry.Tags), types.String),
					"rbacEnabled":                       llx.BoolDataPtr(entry.Properties.EnableRBAC),
					"dnsPrefix":                         llx.StringDataPtr(entry.Properties.DNSPrefix),
					"fqdn":                              llx.StringDataPtr(entry.Properties.Fqdn),
					"fqdnSubdomain":                     llx.StringDataPtr(entry.Properties.FqdnSubdomain),
					"privateFqdn":                       llx.StringDataPtr(entry.Properties.PrivateFQDN),
					"agentPoolProfiles":                 llx.DictData(agentPoolProfiles),
					"addonProfiles":                     llx.DictData(addonProfiles),
					"httpProxyConfig":                   llx.DictData(httpProxyConfig),
					"networkProfile":                    llx.DictData(networkProfile),
					"podIdentityProfile":                llx.DictData(podIdentityProfile),
					"securityProfile":                   llx.DictData(securityProfile),
					"storageProfile":                    llx.DictData(storageProfile),
					"workloadAutoScalerProfile":         llx.DictData(workloadAutoScalerProfile),
					"apiServerAccessProfile":            llx.DictData(apiServerAccessProfile),
					"enablePrivateCluster":              llx.BoolDataPtr(enablePrivateCluster),
					"enablePrivateClusterPublicFQDN":    llx.BoolDataPtr(enablePrivateClusterPublicFQDN),
					"disableRunCommand":                 llx.BoolDataPtr(disableRunCommand),
					"apiServerAuthorizedIPRanges":       llx.ArrayData(apiServerAuthorizedIPRanges, types.String),
					"privateDnsZone":                    llx.StringDataPtr(privateDnsZone),
					"defenderEnabled":                   llx.BoolDataPtr(defenderEnabled),
					"imageCleanerEnabled":               llx.BoolDataPtr(imageCleanerEnabled),
					"imageCleanerIntervalHours":         llx.IntDataDefault(imageCleanerIntervalHours, 0),
					"workloadIdentityEnabled":           llx.BoolDataPtr(workloadIdentityEnabled),
					"azureKeyVaultKmsEnabled":           llx.BoolDataPtr(azureKeyVaultKmsEnabled),
					"azureKeyVaultKmsNetworkAccess":     llx.StringDataPtr(azureKeyVaultKmsNetworkAccess),
					"disableLocalAccounts":              llx.BoolDataPtr(entry.Properties.DisableLocalAccounts),
					"publicNetworkAccess":               llx.StringDataPtr((*string)(entry.Properties.PublicNetworkAccess)),
					"skuTier":                           llx.StringData(skuTier),
					"networkPlugin":                     llx.StringDataPtr(networkPlugin),
					"networkPolicy":                     llx.StringDataPtr(networkPolicy),
					"oidcIssuerEnabled":                 llx.BoolDataPtr(oidcIssuerEnabled),
					"nodeResourceGroupRestrictionLevel": llx.StringDataPtr(nodeResourceGroupRestrictionLevel),
					"serviceMeshMode":                   llx.StringDataPtr(serviceMeshMode),
					"supportPlan":                       llx.StringDataPtr((*string)(entry.Properties.SupportPlan)),
					"controlPlaneMetricsEnabled":        llx.BoolDataPtr(controlPlaneMetricsEnabled),
				})
			if err != nil {
				return nil, err
			}
			mqlCluster := mqlAksCluster.(*mqlAzureSubscriptionAksServiceCluster)
			mqlCluster.cacheKmsKeyId = azureKeyVaultKmsKeyId
			mqlCluster.cacheProperties = entry.Properties
			res = append(res, mqlCluster)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionAksServiceCluster) azureKeyVaultKmsKey() (*mqlAzureSubscriptionKeyVaultServiceKey, error) {
	if a.cacheKmsKeyId == "" {
		a.AzureKeyVaultKmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return newKeyVaultKeyResource(a.MqlRuntime, a.cacheKmsKeyId)
}

func (a *mqlAzureSubscriptionAksServiceCluster) aadProfile() (*mqlAzureSubscriptionAksServiceClusterAadProfile, error) {
	if a.cacheProperties == nil || a.cacheProperties.AADProfile == nil {
		a.AadProfile.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	aadP := a.cacheProperties.AADProfile
	adminGroupObjectIDs := []any{}
	for _, gid := range aadP.AdminGroupObjectIDs {
		if gid != nil {
			adminGroupObjectIDs = append(adminGroupObjectIDs, *gid)
		}
	}
	aadRes, err := CreateResource(a.MqlRuntime, "azure.subscription.aksService.cluster.aadProfile",
		map[string]*llx.RawData{
			"id":                  llx.StringData(a.Id.Data + "/aadProfile"),
			"managed":             llx.BoolDataPtr(aadP.Managed),
			"enableAzureRBAC":     llx.BoolDataPtr(aadP.EnableAzureRBAC),
			"adminGroupObjectIDs": llx.ArrayData(adminGroupObjectIDs, types.String),
		})
	if err != nil {
		return nil, err
	}
	return aadRes.(*mqlAzureSubscriptionAksServiceClusterAadProfile), nil
}

func (a *mqlAzureSubscriptionAksServiceCluster) autoUpgradeProfile() (*mqlAzureSubscriptionAksServiceClusterAutoUpgradeProfile, error) {
	if a.cacheProperties == nil || a.cacheProperties.AutoUpgradeProfile == nil {
		a.AutoUpgradeProfile.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	aup := a.cacheProperties.AutoUpgradeProfile
	autoUpgradeRes, err := CreateResource(a.MqlRuntime, "azure.subscription.aksService.cluster.autoUpgradeProfile",
		map[string]*llx.RawData{
			"id":                   llx.StringData(a.Id.Data + "/autoUpgradeProfile"),
			"upgradeChannel":       llx.StringDataPtr((*string)(aup.UpgradeChannel)),
			"nodeOSUpgradeChannel": llx.StringDataPtr((*string)(aup.NodeOSUpgradeChannel)),
		})
	if err != nil {
		return nil, err
	}
	return autoUpgradeRes.(*mqlAzureSubscriptionAksServiceClusterAutoUpgradeProfile), nil
}

func (a *mqlAzureSubscriptionAksServiceCluster) nodePools() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()

	resourceID, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}

	client, err := clusters.NewAgentPoolsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListPager(resourceID.ResourceGroup, a.Name.Data, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range page.Value {
			if entry == nil || entry.Properties == nil {
				continue
			}
			props := entry.Properties

			zones := []any{}
			for _, z := range props.AvailabilityZones {
				if z != nil {
					zones = append(zones, *z)
				}
			}

			taints := []any{}
			for _, t := range props.NodeTaints {
				if t != nil {
					taints = append(taints, *t)
				}
			}

			var spotMaxPrice float64
			if props.SpotMaxPrice != nil {
				spotMaxPrice = float64(*props.SpotMaxPrice)
			}

			var powerState *string
			if props.PowerState != nil {
				powerState = (*string)(props.PowerState.Code)
			}

			var enableSecureBoot, enableVTPM *bool
			var sshAccess *string
			if props.SecurityProfile != nil {
				enableSecureBoot = props.SecurityProfile.EnableSecureBoot
				enableVTPM = props.SecurityProfile.EnableVTPM
				sshAccess = (*string)(props.SecurityProfile.SSHAccess)
			}

			var upgradeMaxSurge, upgradeMaxUnavailable *string
			var upgradeDrainTimeout, upgradeNodeSoak *int32
			if props.UpgradeSettings != nil {
				upgradeMaxSurge = props.UpgradeSettings.MaxSurge
				upgradeMaxUnavailable = props.UpgradeSettings.MaxUnavailable
				upgradeDrainTimeout = props.UpgradeSettings.DrainTimeoutInMinutes
				upgradeNodeSoak = props.UpgradeSettings.NodeSoakDurationInMinutes
			}

			mqlPool, err := CreateResource(a.MqlRuntime, "azure.subscription.aksService.cluster.nodePool",
				map[string]*llx.RawData{
					"id":                               llx.StringDataPtr(entry.ID),
					"name":                             llx.StringDataPtr(entry.Name),
					"mode":                             llx.StringDataPtr((*string)(props.Mode)),
					"vmType":                           llx.StringDataPtr((*string)(props.Type)),
					"count":                            llx.IntDataDefault(props.Count, 0),
					"minCount":                         llx.IntDataDefault(props.MinCount, 0),
					"maxCount":                         llx.IntDataDefault(props.MaxCount, 0),
					"enableAutoScaling":                llx.BoolDataPtr(props.EnableAutoScaling),
					"maxPods":                          llx.IntDataDefault(props.MaxPods, 0),
					"vmSize":                           llx.StringDataPtr(props.VMSize),
					"osType":                           llx.StringDataPtr((*string)(props.OSType)),
					"osSku":                            llx.StringDataPtr((*string)(props.OSSKU)),
					"osDiskType":                       llx.StringDataPtr((*string)(props.OSDiskType)),
					"osDiskSizeGB":                     llx.IntDataDefault(props.OSDiskSizeGB, 0),
					"scaleSetPriority":                 llx.StringDataPtr((*string)(props.ScaleSetPriority)),
					"scaleSetEvictionPolicy":           llx.StringDataPtr((*string)(props.ScaleSetEvictionPolicy)),
					"spotMaxPrice":                     llx.FloatData(spotMaxPrice),
					"availabilityZones":                llx.ArrayData(zones, types.String),
					"enableNodePublicIP":               llx.BoolDataPtr(props.EnableNodePublicIP),
					"nodePublicIPPrefixId":             llx.StringDataPtr(props.NodePublicIPPrefixID),
					"enableEncryptionAtHost":           llx.BoolDataPtr(props.EnableEncryptionAtHost),
					"enableUltraSSD":                   llx.BoolDataPtr(props.EnableUltraSSD),
					"enableFIPS":                       llx.BoolDataPtr(props.EnableFIPS),
					"orchestratorVersion":              llx.StringDataPtr(props.OrchestratorVersion),
					"currentOrchestratorVersion":       llx.StringDataPtr(props.CurrentOrchestratorVersion),
					"nodeImageVersion":                 llx.StringDataPtr(props.NodeImageVersion),
					"vnetSubnetId":                     llx.StringDataPtr(props.VnetSubnetID),
					"podSubnetId":                      llx.StringDataPtr(props.PodSubnetID),
					"proximityPlacementGroupId":        llx.StringDataPtr(props.ProximityPlacementGroupID),
					"hostGroupId":                      llx.StringDataPtr(props.HostGroupID),
					"provisioningState":                llx.StringDataPtr(props.ProvisioningState),
					"powerState":                       llx.StringDataPtr(powerState),
					"workloadRuntime":                  llx.StringDataPtr((*string)(props.WorkloadRuntime)),
					"nodeLabels":                       llx.MapData(convert.PtrMapStrToInterface(props.NodeLabels), types.String),
					"nodeTaints":                       llx.ArrayData(taints, types.String),
					"tags":                             llx.MapData(convert.PtrMapStrToInterface(props.Tags), types.String),
					"upgradeMaxSurge":                  llx.StringDataPtr(upgradeMaxSurge),
					"upgradeMaxUnavailable":            llx.StringDataPtr(upgradeMaxUnavailable),
					"upgradeDrainTimeoutInMinutes":     llx.IntDataDefault(upgradeDrainTimeout, 0),
					"upgradeNodeSoakDurationInMinutes": llx.IntDataDefault(upgradeNodeSoak, 0),
					"sshAccess":                        llx.StringDataPtr(sshAccess),
					"enableVTPM":                       llx.BoolDataPtr(enableVTPM),
					"enableSecureBoot":                 llx.BoolDataPtr(enableSecureBoot),
				})
			if err != nil {
				return nil, err
			}
			pool := mqlPool.(*mqlAzureSubscriptionAksServiceClusterNodePool)
			pool.subscriptionId = resourceID.SubscriptionID
			pool.resourceGroup = resourceID.ResourceGroup
			pool.clusterName = a.Name.Data
			pool.poolName = convert.ToValue(entry.Name)
			res = append(res, pool)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionAksServiceClusterIdentityBinding) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAzureSubscriptionAksServiceCluster) identityBindings() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()

	resourceID, err := ParseResourceID(a.Id.Data)
	if err != nil {
		return nil, err
	}

	client, err := clusters.NewIdentityBindingsClient(resourceID.SubscriptionID, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	pager := client.NewListByManagedClusterPager(resourceID.ResourceGroup, a.Name.Data, nil)
	res := []any{}
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			// Identity bindings are a preview capability: the endpoint
			// returns 404 on clusters/subscriptions where it is not
			// available, and 403 when the caller lacks access. Treat
			// both as "no bindings" rather than failing the query.
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && (respErr.StatusCode == http.StatusNotFound || respErr.StatusCode == http.StatusForbidden) {
				return res, nil
			}
			return nil, err
		}
		for _, entry := range page.Value {
			if entry == nil {
				continue
			}

			var provisioningState, oidcIssuerUrl string
			var managedIdentityId, managedIdentityClientId, managedIdentityObjectId, managedIdentityTenantId string
			if props := entry.Properties; props != nil {
				provisioningState = convert.ToValue((*string)(props.ProvisioningState))
				if props.OidcIssuer != nil {
					oidcIssuerUrl = convert.ToValue(props.OidcIssuer.OidcIssuerURL)
				}
				if mi := props.ManagedIdentity; mi != nil {
					managedIdentityId = convert.ToValue(mi.ResourceID)
					managedIdentityClientId = convert.ToValue(mi.ClientID)
					managedIdentityObjectId = convert.ToValue(mi.ObjectID)
					managedIdentityTenantId = convert.ToValue(mi.TenantID)
				}
			}

			mqlBinding, err := CreateResource(a.MqlRuntime, "azure.subscription.aksService.cluster.identityBinding",
				map[string]*llx.RawData{
					"id":                      llx.StringDataPtr(entry.ID),
					"name":                    llx.StringDataPtr(entry.Name),
					"provisioningState":       llx.StringData(provisioningState),
					"managedIdentityClientId": llx.StringData(managedIdentityClientId),
					"managedIdentityObjectId": llx.StringData(managedIdentityObjectId),
					"managedIdentityTenantId": llx.StringData(managedIdentityTenantId),
					"oidcIssuerUrl":           llx.StringData(oidcIssuerUrl),
				})
			if err != nil {
				return nil, err
			}
			binding := mqlBinding.(*mqlAzureSubscriptionAksServiceClusterIdentityBinding)
			binding.cacheManagedIdentityId = managedIdentityId
			res = append(res, binding)
		}
	}
	return res, nil
}

func (a *mqlAzureSubscriptionAksServiceClusterIdentityBinding) managedIdentity() (*mqlAzureSubscriptionManagedIdentity, error) {
	if a.cacheManagedIdentityId == "" {
		a.ManagedIdentity.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "azure.subscription.managedIdentity",
		map[string]*llx.RawData{"__id": llx.StringData(a.cacheManagedIdentityId)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAzureSubscriptionManagedIdentity), nil
}

func (a *mqlAzureSubscriptionAksServiceClusterNodePool) recentlyUsedVersions() ([]any, error) {
	if a.poolName == "" || a.clusterName == "" {
		return []any{}, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AzureConnection)
	ctx := context.Background()
	token := conn.Token()

	client, err := clusters.NewAgentPoolsClient(a.subscriptionId, token, &arm.ClientOptions{
		ClientOptions: conn.ClientOptions(),
	})
	if err != nil {
		return nil, err
	}

	profile, err := client.GetUpgradeProfile(ctx, a.resourceGroup, a.clusterName, a.poolName, nil)
	if err != nil {
		return nil, err
	}
	if profile.Properties == nil {
		return []any{}, nil
	}
	return convert.JsonToDictSlice(profile.Properties.RecentlyUsedVersions)
}
