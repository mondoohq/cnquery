// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strings"

	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/gcp/connection"
	"go.mondoo.com/cnquery/v10/types"

	container "cloud.google.com/go/container/apiv1"
	"cloud.google.com/go/container/apiv1/containerpb"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectGkeService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	return fmt.Sprintf("%s/gcp.project.gkeService", projectId), nil
}

func (g *mqlGcpProject) gke() (*mqlGcpProjectGkeService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.gkeService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectGkeService), nil
}

func (g *mqlGcpProjectGkeServiceCluster) id() (string, error) {
	if g.Id.Error != nil {
		return "", g.Id.Error
	}
	id := g.Id.Data
	return fmt.Sprintf("gcp.project.gkeService.cluster/%s", id), nil
}

func initGcpProjectGkeServiceCluster(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 3 {
		return args, nil, nil
	}

	// If no args are set, try reading them from the platform ID
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["location"] = llx.StringData(ids.region)
			args["projectId"] = llx.StringData(ids.project)
		}
	}

	obj, err := CreateResource(runtime, "gcp.project.gkeService", map[string]*llx.RawData{
		"projectId": llx.StringData(args["projectId"].Value.(string)),
	})
	if err != nil {
		return nil, nil, err
	}
	gkeSvc := obj.(*mqlGcpProjectGkeService)
	clusters := gkeSvc.GetClusters()
	if clusters.Error != nil {
		return nil, nil, clusters.Error
	}

	for _, c := range clusters.Data {
		cluster := c.(*mqlGcpProjectGkeServiceCluster)
		name := cluster.GetName()
		if name.Error != nil {
			return nil, nil, name.Error
		}
		projectId := cluster.GetProjectId()
		if projectId.Error != nil {
			return nil, nil, projectId.Error
		}
		location := cluster.GetLocation()
		if location.Error != nil {
			return nil, nil, location.Error
		}

		if name.Data == args["name"].Value && projectId.Data == args["projectId"].Value && location.Data == args["location"].Value {
			return args, cluster, nil
		}
	}
	return nil, nil, nil
}

func (g *mqlGcpProjectGkeServiceClusterNodepool) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolNetworkConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolNetworkConfigPerformanceConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigAccelerator) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigAcceleratorGpuSharingConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigNodeTaint) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigSandboxConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigShieldedInstanceConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigLinuxNodeConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigKubeletConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigGcfsConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigAdvancedMachineFeatures) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigGvnicConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigConfidentialNodes) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterAddonsConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterIpAllocationPolicy) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeServiceClusterNetworkConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectGkeService) clusters() ([]interface{}, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(container.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	containerSvc, err := container.NewClusterManagerClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer containerSvc.Close()

	// List the clusters in the current projects for all locations
	resp, err := containerSvc.ListClusters(ctx, &containerpb.ListClustersRequest{Parent: fmt.Sprintf("projects/%s/locations/-", projectId)})
	if err != nil {
		log.Error().Err(err).Msg("failed to list clusters")
		return nil, err
	}
	res := []interface{}{}

	for i := range resp.Clusters {
		c := resp.Clusters[i]

		nodePools := make([]interface{}, 0, len(c.NodePools))
		for _, np := range c.NodePools {
			mqlNodePool, err := createMqlNodePool(g.MqlRuntime, np, c.Id, projectId)
			if err != nil {
				return nil, err
			}
			nodePools = append(nodePools, mqlNodePool)
		}

		autopilotEnabled := false
		if c.Autopilot != nil {
			autopilotEnabled = c.Autopilot.Enabled
		}

		var addonsConfig plugin.Resource
		if c.AddonsConfig != nil {
			var httpLoadBalancing map[string]interface{}
			if c.AddonsConfig.HttpLoadBalancing != nil {
				httpLoadBalancing = map[string]interface{}{
					"disabled": c.AddonsConfig.HttpLoadBalancing.Disabled,
				}
			}

			var horizontalPodAutoscaling map[string]interface{}
			if c.AddonsConfig.HorizontalPodAutoscaling != nil {
				horizontalPodAutoscaling = map[string]interface{}{
					"disabled": c.AddonsConfig.HorizontalPodAutoscaling.Disabled,
				}
			}

			var kubernetesDashboard map[string]interface{}
			if c.AddonsConfig.KubernetesDashboard != nil {
				kubernetesDashboard = map[string]interface{}{
					"disabled": c.AddonsConfig.KubernetesDashboard.Disabled,
				}
			}

			var networkPolicyConfig map[string]interface{}
			if c.AddonsConfig.NetworkPolicyConfig != nil {
				networkPolicyConfig = map[string]interface{}{
					"disabled": c.AddonsConfig.NetworkPolicyConfig.Disabled,
				}
			}

			var cloudRunConfig map[string]interface{}
			if c.AddonsConfig.CloudRunConfig != nil {
				cloudRunConfig = map[string]interface{}{
					"disabled":         c.AddonsConfig.CloudRunConfig.Disabled,
					"loadBalancerType": c.AddonsConfig.CloudRunConfig.LoadBalancerType.String(),
				}
			}

			var dnsCacheConfig map[string]interface{}
			if c.AddonsConfig.DnsCacheConfig != nil {
				dnsCacheConfig = map[string]interface{}{
					"enabled": c.AddonsConfig.DnsCacheConfig.Enabled,
				}
			}

			var configConnectorConfig map[string]interface{}
			if c.AddonsConfig.ConfigConnectorConfig != nil {
				configConnectorConfig = map[string]interface{}{
					"enabled": c.AddonsConfig.ConfigConnectorConfig.Enabled,
				}
			}

			var gcePersistentDiskCsiDriverConfig map[string]interface{}
			if c.AddonsConfig.GcePersistentDiskCsiDriverConfig != nil {
				gcePersistentDiskCsiDriverConfig = map[string]interface{}{
					"enabled": c.AddonsConfig.GcePersistentDiskCsiDriverConfig.Enabled,
				}
			}

			var gcpFilestoreCsiDriverConfig map[string]interface{}
			if c.AddonsConfig.GcpFilestoreCsiDriverConfig != nil {
				gcpFilestoreCsiDriverConfig = map[string]interface{}{
					"enabled": c.AddonsConfig.GcpFilestoreCsiDriverConfig.Enabled,
				}
			}

			var gkeBackupAgentConfig map[string]interface{}
			if c.AddonsConfig.GkeBackupAgentConfig != nil {
				gkeBackupAgentConfig = map[string]interface{}{
					"enabled": c.AddonsConfig.GkeBackupAgentConfig.Enabled,
				}
			}

			addonsConfig, err = CreateResource(g.MqlRuntime, "gcp.project.gkeService.cluster.addonsConfig", map[string]*llx.RawData{
				"id":                               llx.StringData(fmt.Sprintf("gcp.project.gkeService.cluster/%s/addonsConfig", c.Id)),
				"httpLoadBalancing":                llx.DictData(httpLoadBalancing),
				"horizontalPodAutoscaling":         llx.DictData(horizontalPodAutoscaling),
				"kubernetesDashboard":              llx.DictData(kubernetesDashboard),
				"networkPolicyConfig":              llx.DictData(networkPolicyConfig),
				"cloudRunConfig":                   llx.DictData(cloudRunConfig),
				"dnsCacheConfig":                   llx.DictData(dnsCacheConfig),
				"configConnectorConfig":            llx.DictData(configConnectorConfig),
				"gcePersistentDiskCsiDriverConfig": llx.DictData(gcePersistentDiskCsiDriverConfig),
				"gcpFilestoreCsiDriverConfig":      llx.DictData(gcpFilestoreCsiDriverConfig),
				"gkeBackupAgentConfig":             llx.DictData(gkeBackupAgentConfig),
			})
			if err != nil {
				return nil, err
			}
		}

		var workloadIdCfg map[string]interface{}
		if c.WorkloadIdentityConfig != nil {
			workloadIdCfg = map[string]interface{}{
				"workloadPool": c.WorkloadIdentityConfig.WorkloadPool,
			}
		}

		var ipAllocPolicy plugin.Resource
		if c.IpAllocationPolicy != nil {
			ipAllocPolicy, err = CreateResource(g.MqlRuntime, "gcp.project.gkeService.cluster.ipAllocationPolicy", map[string]*llx.RawData{
				"id":                         llx.StringData(fmt.Sprintf("gcp.project.gkeService.cluster/%s/ipAllocationPolicy", c.Id)),
				"useIpAliases":               llx.BoolData(c.IpAllocationPolicy.UseIpAliases),
				"createSubnetwork":           llx.BoolData(c.IpAllocationPolicy.CreateSubnetwork),
				"subnetworkName":             llx.StringData(c.IpAllocationPolicy.SubnetworkName),
				"clusterSecondaryRangeName":  llx.StringData(c.IpAllocationPolicy.ClusterSecondaryRangeName),
				"servicesSecondaryRangeName": llx.StringData(c.IpAllocationPolicy.ServicesSecondaryRangeName),
				"clusterIpv4CidrBlock":       llx.StringData(c.IpAllocationPolicy.ClusterIpv4CidrBlock),
				"nodeIpv4CidrBlock":          llx.StringData(c.IpAllocationPolicy.NodeIpv4CidrBlock),
				"servicesIpv4CidrBlock":      llx.StringData(c.IpAllocationPolicy.ServicesIpv4CidrBlock),
				"tpuIpv4CidrBlock":           llx.StringData(c.IpAllocationPolicy.TpuIpv4CidrBlock),
				"useRoutes":                  llx.BoolData(c.IpAllocationPolicy.UseRoutes),
				"stackType":                  llx.StringData(c.IpAllocationPolicy.StackType.String()),
				"ipv6AccessType":             llx.StringData(c.IpAllocationPolicy.Ipv6AccessType.String()),
			})
			if err != nil {
				return nil, err
			}
		}
		var networkConfig plugin.Resource
		if c.NetworkConfig != nil {
			var defaultSnatStatus map[string]interface{}
			if c.NetworkConfig.DefaultSnatStatus != nil {
				defaultSnatStatus = map[string]interface{}{
					"disabled": c.NetworkConfig.DefaultSnatStatus.Disabled,
				}
			}

			var dnsConfig map[string]interface{}
			if c.NetworkConfig.DnsConfig != nil {
				dnsConfig = map[string]interface{}{
					"clusterDns":       c.NetworkConfig.DnsConfig.ClusterDns.String(),
					"clusterDnsScope":  c.NetworkConfig.DnsConfig.ClusterDnsScope.String(),
					"clusterDnsDomain": c.NetworkConfig.DnsConfig.ClusterDnsDomain,
				}
			}

			var serviceExternalIpsConfig map[string]interface{}
			if c.NetworkConfig.ServiceExternalIpsConfig != nil {
				serviceExternalIpsConfig = map[string]interface{}{
					"enabled": c.NetworkConfig.ServiceExternalIpsConfig.Enabled,
				}
			}
			networkConfig, err = CreateResource(g.MqlRuntime, "gcp.project.gkeService.cluster.networkConfig", map[string]*llx.RawData{
				"id":                        llx.StringData(fmt.Sprintf("gcp.project.gkeService.cluster/%s/networkConfig", c.Id)),
				"networkPath":               llx.StringData(c.NetworkConfig.Network),
				"subnetworkPath":            llx.StringData(c.NetworkConfig.Subnetwork),
				"enableIntraNodeVisibility": llx.BoolData(c.NetworkConfig.EnableIntraNodeVisibility),
				"defaultSnatStatus":         llx.DictData(defaultSnatStatus),
				"enableL4IlbSubsetting":     llx.BoolData(c.NetworkConfig.EnableL4IlbSubsetting),
				"datapathProvider":          llx.StringData(c.NetworkConfig.DatapathProvider.String()),
				"privateIpv6GoogleAccess":   llx.StringData(c.NetworkConfig.PrivateIpv6GoogleAccess.String()),
				"dnsConfig":                 llx.DictData(dnsConfig),
				"serviceExternalIpsConfig":  llx.DictData(serviceExternalIpsConfig),
			})
			if err != nil {
				return nil, err
			}
		}

		var binAuth map[string]interface{}
		if c.BinaryAuthorization != nil {
			binAuth = map[string]interface{}{
				"enabled":        c.BinaryAuthorization.Enabled,
				"evaluationMode": c.BinaryAuthorization.EvaluationMode.String(),
			}
		}

		var legacyAbac map[string]interface{}
		if c.LegacyAbac != nil {
			legacyAbac = map[string]interface{}{
				"enabled": c.LegacyAbac.Enabled,
			}
		}

		var masterAuth map[string]interface{}
		if c.MasterAuth != nil {
			var clientCertCfg map[string]interface{}
			if c.MasterAuth.ClientCertificateConfig != nil {
				clientCertCfg = map[string]interface{}{
					"issueClientCertificate": c.MasterAuth.ClientCertificateConfig.IssueClientCertificate,
				}
			}
			masterAuth = map[string]interface{}{
				"username":                c.MasterAuth.Username,
				"password":                c.MasterAuth.Password,
				"clientCertificateConfig": clientCertCfg,
				"clusterCaCertificate":    c.MasterAuth.ClusterCaCertificate,
				"clientCertificate":       c.MasterAuth.ClientCertificate,
				"clientKey":               c.MasterAuth.ClientKey,
			}
		}

		var masterAuthorizedNetworksCfg map[string]interface{}
		if c.MasterAuthorizedNetworksConfig != nil {
			cidrBlocks := make([]interface{}, 0, len(c.MasterAuthorizedNetworksConfig.CidrBlocks))
			for _, cidrBlock := range c.MasterAuthorizedNetworksConfig.CidrBlocks {
				cidrBlocks = append(cidrBlocks, map[string]interface{}{
					"displayName": cidrBlock.DisplayName,
					"cidrBlock":   cidrBlock.CidrBlock,
				})
			}
			masterAuthorizedNetworksCfg = map[string]interface{}{
				"enabled":    c.MasterAuthorizedNetworksConfig.Enabled,
				"cidrBlocks": cidrBlocks,
			}
		}

		var privateClusterCfg map[string]interface{}
		if c.PrivateClusterConfig != nil {
			var masterGlobalAccessCfg map[string]interface{}
			if c.PrivateClusterConfig.MasterGlobalAccessConfig != nil {
				masterGlobalAccessCfg = map[string]interface{}{
					"enabled": c.PrivateClusterConfig.MasterGlobalAccessConfig.Enabled,
				}
			}

			privateClusterCfg = map[string]interface{}{
				"enablePrivateNodes":       c.PrivateClusterConfig.EnablePrivateNodes,
				"enablePrivateEndpoint":    c.PrivateClusterConfig.EnablePrivateEndpoint,
				"masterIpv4CidrBlock":      c.PrivateClusterConfig.MasterIpv4CidrBlock,
				"privateEndpoint":          c.PrivateClusterConfig.PrivateEndpoint,
				"publicEndpoint":           c.PrivateClusterConfig.PublicEndpoint,
				"peeringName":              c.PrivateClusterConfig.PeeringName,
				"masterGlobalAccessConfig": masterGlobalAccessCfg,
			}
		}

		var databaseEncryption map[string]interface{}
		if c.DatabaseEncryption != nil {
			databaseEncryption = map[string]interface{}{
				"state":   c.DatabaseEncryption.State.String(),
				"keyName": c.DatabaseEncryption.KeyName,
			}
		}

		mqlCluster, err := CreateResource(g.MqlRuntime, "gcp.project.gkeService.cluster", map[string]*llx.RawData{
			"projectId":                      llx.StringData(projectId),
			"id":                             llx.StringData(c.Id),
			"name":                           llx.StringData(c.Name),
			"description":                    llx.StringData(c.Description),
			"loggingService":                 llx.StringData(c.LoggingService),
			"monitoringService":              llx.StringData(c.MonitoringService),
			"network":                        llx.StringData(c.Network),
			"clusterIpv4Cidr":                llx.StringData(c.ClusterIpv4Cidr),
			"subnetwork":                     llx.StringData(c.Subnetwork),
			"nodePools":                      llx.ArrayData(nodePools, types.Resource("gcp.project.gkeService.cluster.nodepool")),
			"locations":                      llx.ArrayData(convert.SliceAnyToInterface(c.Locations), types.String),
			"enableKubernetesAlpha":          llx.BoolData(c.EnableKubernetesAlpha),
			"autopilotEnabled":               llx.BoolData(autopilotEnabled),
			"zone":                           llx.StringData(c.Zone),
			"location":                       llx.StringData(c.Location),
			"endpoint":                       llx.StringData(c.Endpoint),
			"initialClusterVersion":          llx.StringData(c.InitialClusterVersion),
			"currentMasterVersion":           llx.StringData(c.CurrentMasterVersion),
			"status":                         llx.StringData(c.Status.String()),
			"resourceLabels":                 llx.MapData(convert.MapToInterfaceMap(c.ResourceLabels), types.String),
			"created":                        llx.TimeDataPtr(parseTime(c.CreateTime)),
			"expirationTime":                 llx.TimeDataPtr(parseTime(c.ExpireTime)),
			"addonsConfig":                   llx.ResourceData(addonsConfig, "gcp.project.gkeService.cluster.addonsConfig"),
			"workloadIdentityConfig":         llx.DictData(workloadIdCfg),
			"ipAllocationPolicy":             llx.ResourceData(ipAllocPolicy, "gcp.project.gkeService.cluster.ipAllocationPolicy"),
			"networkConfig":                  llx.ResourceData(networkConfig, "gcp.project.gkeService.cluster.networkConfig"),
			"binaryAuthorization":            llx.DictData(binAuth),
			"legacyAbac":                     llx.DictData(legacyAbac),
			"masterAuth":                     llx.DictData(masterAuth),
			"masterAuthorizedNetworksConfig": llx.DictData(masterAuthorizedNetworksCfg),
			"privateClusterConfig":           llx.DictData(privateClusterCfg),
			"databaseEncryption":             llx.DictData(databaseEncryption),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCluster)
	}

	return res, nil
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfig) serviceAccount() (*mqlGcpProjectIamServiceServiceAccount, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	if g.ServiceAccountEmail.Error != nil {
		return nil, g.ServiceAccountEmail.Error
	}
	email := g.ServiceAccountEmail.Data

	res, err := CreateResource(g.MqlRuntime, "gcp.project.iamService.serviceAccount", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
		"email":     llx.StringData(email),
	})
	if err != nil {
		return nil, err
	}

	return res.(*mqlGcpProjectIamServiceServiceAccount), nil
}

func createMqlNodePool(runtime *plugin.Runtime, np *containerpb.NodePool, clusterId, projectId string) (plugin.Resource, error) {
	nodePoolId := fmt.Sprintf("%s/%s", clusterId, np.Name)

	mqlPoolConfig, err := createMqlNodePoolConfig(runtime, np, nodePoolId, projectId)
	if err != nil {
		return nil, err
	}

	mqlPoolNetworkConfig, err := createMqlNodePoolNetworkConfig(runtime, np, nodePoolId)
	if err != nil {
		return nil, err
	}

	var management map[string]interface{}
	if np.Management != nil {
		var upgradeOpts map[string]interface{}
		if np.Management.UpgradeOptions != nil {
			upgradeOpts = map[string]interface{}{
				"autoUpgradeStartTime": np.Management.UpgradeOptions.AutoUpgradeStartTime,
				"description":          np.Management.UpgradeOptions.Description,
			}
		}
		management = map[string]interface{}{
			"autoRepair":     np.Management.AutoRepair,
			"autoUpgrade":    np.Management.AutoUpgrade,
			"upgradeOptions": upgradeOpts,
		}
	}

	return CreateResource(runtime, "gcp.project.gkeService.cluster.nodepool", map[string]*llx.RawData{
		"id":                llx.StringData(nodePoolId),
		"name":              llx.StringData(np.Name),
		"config":            llx.ResourceData(mqlPoolConfig, "gcp.project.gkeService.cluster.nodepool.config"),
		"initialNodeCount":  llx.IntData(int64(np.InitialNodeCount)),
		"locations":         llx.ArrayData(convert.SliceAnyToInterface(np.Locations), types.String),
		"networkConfig":     llx.ResourceData(mqlPoolNetworkConfig, "gcp.project.gkeService.cluster.nodepool.networkConfig"),
		"version":           llx.StringData(np.Version),
		"instanceGroupUrls": llx.ArrayData(convert.SliceAnyToInterface(np.InstanceGroupUrls), types.String),
		"status":            llx.StringData(np.Status.String()),
		"management":        llx.DictData(management),
	})
}

func createMqlNodePoolConfig(runtime *plugin.Runtime, np *containerpb.NodePool, nodePoolId, projectId string) (plugin.Resource, error) {
	cfg := np.Config
	var err error
	mqlAccelerators := make([]interface{}, 0, len(cfg.Accelerators))
	for i, acc := range cfg.Accelerators {
		mqlAcc, err := createMqlAccelerator(runtime, acc, nodePoolId, i)
		if err != nil {
			return nil, err
		}
		mqlAccelerators = append(mqlAccelerators, mqlAcc)
	}

	nodeTaints := make([]interface{}, 0, len(cfg.Taints))
	for i, taint := range cfg.Taints {
		mqlNodeTaint, err := CreateResource(runtime, "gcp.project.gkeService.cluster.nodepool.config.nodeTaint", map[string]*llx.RawData{
			"id":     llx.StringData(fmt.Sprintf("%s/taints/%d", nodePoolId, i)),
			"key":    llx.StringData(taint.Key),
			"value":  llx.StringData(taint.Value),
			"effect": llx.StringData(taint.Effect.String()),
		})
		if err != nil {
			return nil, err
		}
		nodeTaints = append(nodeTaints, mqlNodeTaint)
	}

	var mqlSandboxCfg plugin.Resource
	if cfg.SandboxConfig != nil {
		mqlSandboxCfg, err = CreateResource(runtime, "gcp.project.gkeService.cluster.nodepool.config.sandboxConfig", map[string]*llx.RawData{
			"id":   llx.StringData(fmt.Sprintf("%s/sandbox", nodePoolId)),
			"type": llx.StringData(cfg.SandboxConfig.Type.String()),
		})
		if err != nil {
			return nil, err
		}
	}

	var mqlShieldedInstanceCfg plugin.Resource
	if cfg.ShieldedInstanceConfig != nil {
		mqlShieldedInstanceCfg, err = CreateResource(runtime, "gcp.project.gkeService.cluster.nodepool.config.shieldedInstanceConfig", map[string]*llx.RawData{
			"id":                        llx.StringData(fmt.Sprintf("%s/shieldedInstanceConfig", nodePoolId)),
			"enableSecureBoot":          llx.BoolData(cfg.ShieldedInstanceConfig.EnableSecureBoot),
			"enableIntegrityMonitoring": llx.BoolData(cfg.ShieldedInstanceConfig.EnableIntegrityMonitoring),
		})
		if err != nil {
			return nil, err
		}
	}

	var mqlLinuxNodeCfg plugin.Resource
	if cfg.LinuxNodeConfig != nil {
		mqlLinuxNodeCfg, err = CreateResource(runtime, "gcp.project.gkeService.cluster.nodepool.config.linuxNodeConfig", map[string]*llx.RawData{
			"id":      llx.StringData(fmt.Sprintf("%s/linuxNodeConfig", nodePoolId)),
			"sysctls": llx.MapData(convert.MapToInterfaceMap(cfg.LinuxNodeConfig.Sysctls), types.String),
		})
		if err != nil {
			return nil, err
		}
	}

	var mqlKubeletCfg plugin.Resource
	if cfg.KubeletConfig != nil {
		mqlKubeletCfg, err = CreateResource(runtime, "gcp.project.gkeService.cluster.nodepool.config.kubeletConfig", map[string]*llx.RawData{
			"id":                llx.StringData(fmt.Sprintf("%s/kubeletConfig", nodePoolId)),
			"cpuManagerPolicy":  llx.StringData(cfg.KubeletConfig.CpuManagerPolicy),
			"cpuCfsQuotaPeriod": llx.StringData(cfg.KubeletConfig.CpuCfsQuotaPeriod),
			"podPidsLimit":      llx.IntData(cfg.KubeletConfig.PodPidsLimit),
		})
		if err != nil {
			return nil, err
		}
	}

	var mqlGcfsCfg plugin.Resource
	if cfg.GcfsConfig != nil {
		mqlGcfsCfg, err = CreateResource(runtime, "gcp.project.gkeService.cluster.nodepool.config.gcfsConfig", map[string]*llx.RawData{
			"id":      llx.StringData(fmt.Sprintf("%s/gcfsConfig", nodePoolId)),
			"enabled": llx.BoolData(cfg.GcfsConfig.Enabled),
		})
		if err != nil {
			return nil, err
		}
	}

	var mqlAdvancedMachineFeatures plugin.Resource
	if cfg.AdvancedMachineFeatures != nil {
		mqlAdvancedMachineFeatures, err = CreateResource(runtime, "gcp.project.gkeService.cluster.nodepool.config.advancedMachineFeatures", map[string]*llx.RawData{
			"id":             llx.StringData(fmt.Sprintf("%s/advancedMachineFeatures", nodePoolId)),
			"threadsPerCore": llx.BoolData(cfg.Gvnic.Enabled),
		})
		if err != nil {
			return nil, err
		}
	}

	var mqlGvnicCfg plugin.Resource
	if cfg.GcfsConfig != nil {
		mqlGvnicCfg, err = CreateResource(runtime, "gcp.project.gkeService.cluster.nodepool.config.gvnicConfig", map[string]*llx.RawData{
			"id":      llx.StringData(fmt.Sprintf("%s/gvnicConfig", nodePoolId)),
			"enabled": llx.BoolData(cfg.Gvnic.Enabled),
		})
		if err != nil {
			return nil, err
		}
	}

	var mqlConfidentialNodes plugin.Resource
	if cfg.ConfidentialNodes != nil {
		mqlConfidentialNodes, err = CreateResource(runtime, "gcp.project.gkeService.cluster.nodepool.config.confidentialNodes", map[string]*llx.RawData{
			"id":      llx.StringData(fmt.Sprintf("%s/confidentialNodes", nodePoolId)),
			"enabled": llx.BoolData(cfg.ConfidentialNodes.Enabled),
		})
		if err != nil {
			return nil, err
		}
	}

	workloadMetadataMode := ""
	if cfg.WorkloadMetadataConfig != nil {
		workloadMetadataMode = cfg.WorkloadMetadataConfig.Mode.String()
	}

	return CreateResource(runtime, "gcp.project.gkeService.cluster.nodepool.config", map[string]*llx.RawData{
		"id":                      llx.StringData(fmt.Sprintf("%s/config", nodePoolId)),
		"projectId":               llx.StringData(projectId),
		"machineType":             llx.StringData(cfg.MachineType),
		"diskSizeGb":              llx.IntData(int64(cfg.DiskSizeGb)),
		"oauthScopes":             llx.ArrayData(convert.SliceAnyToInterface(cfg.OauthScopes), types.String),
		"serviceAccountEmail":     llx.StringData(cfg.ServiceAccount),
		"metadata":                llx.MapData(convert.MapToInterfaceMap(cfg.Metadata), types.String),
		"imageType":               llx.StringData(cfg.ImageType),
		"labels":                  llx.MapData(convert.MapToInterfaceMap(cfg.Labels), types.String),
		"localSsdCount":           llx.IntData(int64(cfg.LocalSsdCount)),
		"tags":                    llx.ArrayData(convert.SliceAnyToInterface(cfg.Tags), types.String),
		"preemptible":             llx.BoolData(cfg.Preemptible),
		"accelerators":            llx.ArrayData(mqlAccelerators, types.Resource("gcp.project.gkeService.cluster.nodepool.config.accelerator")),
		"diskType":                llx.StringData(cfg.DiskType),
		"minCpuPlatform":          llx.StringData(cfg.MinCpuPlatform),
		"workloadMetadataMode":    llx.StringData(workloadMetadataMode),
		"taints":                  llx.ArrayData(nodeTaints, types.Resource("gcp.project.gkeService.cluster.nodepool.config.nodeTaint")),
		"sandboxConfig":           llx.ResourceData(mqlSandboxCfg, "gcp.project.gkeService.cluster.nodepool.config.sandboxConfig"),
		"shieldedInstanceConfig":  llx.ResourceData(mqlShieldedInstanceCfg, "gcp.project.gkeService.cluster.nodepool.config.shieldedInstanceConfig"),
		"linuxNodeConfig":         llx.ResourceData(mqlLinuxNodeCfg, " gcp.project.gkeService.cluster.nodepool.config.linuxNodeConfig"),
		"kubeletConfig":           llx.ResourceData(mqlKubeletCfg, "gcp.project.gkeService.cluster.nodepool.config.kubeletConfig"),
		"bootDiskKmsKey":          llx.StringData(cfg.BootDiskKmsKey),
		"gcfsConfig":              llx.ResourceData(mqlGcfsCfg, "gcp.project.gkeService.cluster.nodepool.config.gcfsConfig"),
		"gvnicConfig":             llx.ResourceData(mqlGvnicCfg, "gcp.project.gkeService.cluster.nodepool.config.gvnicConfig"),
		"advancedMachineFeatures": llx.ResourceData(mqlAdvancedMachineFeatures, "gcp.project.gkeService.cluster.nodepool.config.advancedMachineFeatures"),
		"spot":                    llx.BoolData(cfg.Spot),
		"confidentialNodes":       llx.ResourceData(mqlConfidentialNodes, "gcp.project.gkeService.cluster.nodepool.config.confidentialNodes"),
	})
}

func createMqlNodePoolNetworkConfig(runtime *plugin.Runtime, np *containerpb.NodePool, nodePoolId string) (plugin.Resource, error) {
	netCfg := np.NetworkConfig
	if netCfg == nil {
		return nil, nil
	}

	netCfgId := fmt.Sprintf("%s/networkConfig", nodePoolId)

	var performanceConfig plugin.Resource
	var err error
	if netCfg.NetworkPerformanceConfig != nil {
		performanceConfig, err = CreateResource(runtime, "gcp.project.gkeService.cluster.nodepool.networkConfig.performanceConfig", map[string]*llx.RawData{
			"id":                       llx.StringData(fmt.Sprintf("%s/performanceConfig", netCfgId)),
			"totalEgressBandwidthTier": llx.StringData(netCfg.NetworkPerformanceConfig.TotalEgressBandwidthTier.String()),
		})
		if err != nil {
			return nil, err
		}
	}

	return CreateResource(runtime, "gcp.project.gkeService.cluster.nodepool.networkConfig", map[string]*llx.RawData{
		"id":                llx.StringData(netCfgId),
		"podRange":          llx.StringData(netCfg.PodRange),
		"podIpv4CidrBlock":  llx.StringData(netCfg.PodIpv4CidrBlock),
		"performanceConfig": llx.ResourceData(performanceConfig, "gcp.project.gkeService.cluster.nodepool.networkConfig.performanceConfig"),
	})
}

func createMqlAccelerator(runtime *plugin.Runtime, acc *containerpb.AcceleratorConfig, nodePoolId string, i int) (plugin.Resource, error) {
	accId := fmt.Sprintf("%s/accelerators/%d", nodePoolId, i)

	var gpuSharingConfig plugin.Resource
	var err error
	if acc.GpuSharingConfig != nil {
		gpuSharingConfig, err = CreateResource(runtime, "gcp.project.gkeService.cluster.nodepool.config.accelerator.gpuSharingConfig", map[string]*llx.RawData{
			"id":                     llx.StringData(fmt.Sprintf("%s/gpuSharingConfig", accId)),
			"maxSharedClientsPerGpu": llx.IntData(acc.GpuSharingConfig.MaxSharedClientsPerGpu),
			"strategy":               llx.StringData(acc.GpuSharingConfig.GpuSharingStrategy.String()),
		})
		if err != nil {
			return nil, err
		}
	}

	return CreateResource(runtime, "gcp.project.gkeService.cluster.nodepool.config.accelerator", map[string]*llx.RawData{
		"id":               llx.StringData(accId),
		"count":            llx.IntData(acc.AcceleratorCount),
		"type":             llx.StringData(acc.AcceleratorType),
		"gpuPartitionSize": llx.StringData(acc.GpuPartitionSize),
		"gpuSharingConfig": llx.ResourceData(gpuSharingConfig, "gcp.project.gkeService.cluster.nodepool.config.accelerator.gpuSharingConfig"),
	})
}

func (g *mqlGcpProjectGkeServiceClusterNetworkConfig) network() (*mqlGcpProjectComputeServiceNetwork, error) {
	if g.NetworkPath.Error != nil {
		return nil, g.NetworkPath.Error
	}
	networkPath := g.NetworkPath.Data

	// Format is projects/project-1/global/networks/net-1
	params := strings.Split(networkPath, "/")
	res, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.network", map[string]*llx.RawData{
		"name":      llx.StringData(params[len(params)-1]),
		"projectId": llx.StringData(params[1]),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectComputeServiceNetwork), nil
}

func (g *mqlGcpProjectGkeServiceClusterNetworkConfig) subnetwork() (*mqlGcpProjectComputeServiceSubnetwork, error) {
	if g.SubnetworkPath.Error != nil {
		return nil, g.SubnetworkPath.Error
	}
	subnetPath := g.SubnetworkPath.Data

	// Format is projects/project-1/regions/us-central1/subnetworks/subnet-1
	params := strings.Split(subnetPath, "/")
	res, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.subnetwork", map[string]*llx.RawData{
		"name":      llx.StringData(params[len(params)-1]),
		"projectId": llx.StringData(params[1]),
		"region":    llx.StringData(params[3]),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectComputeServiceSubnetwork), nil
}
