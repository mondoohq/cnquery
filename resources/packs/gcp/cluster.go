package gcp

import (
	"context"
	"fmt"

	container "cloud.google.com/go/container/apiv1"
	"cloud.google.com/go/container/apiv1/containerpb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/option"
)

func (g *mqlGcpCluster) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpCluster) init(args *resources.Args) (*resources.Args, GcpCluster, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	projectId := provider.ResourceID()
	(*args)["projectId"] = projectId

	return args, nil, nil
}

func (g *mqlGcpClusterNodepool) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpClusterNodepoolConfig) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpClusterNodepoolNetworkConfig) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpClusterNodepoolNetworkConfigPerformanceConfig) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpClusterNodepoolConfigAccelerator) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpClusterNodepoolConfigAcceleratorGpuSharingConfig) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpClusterNodepoolConfigNodeTaint) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpClusterNodepoolConfigSandboxConfig) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpClusterNodepoolConfigShieldedInstanceConfig) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpClusterNodepoolConfigLinuxNodeConfig) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpClusterNodepoolConfigKubeletConfig) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpClusterNodepoolConfigGcfsConfig) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpClusterNodepoolConfigAdvancedMachineFeatures) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpClusterNodepoolConfigGvnicConfig) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpClusterNodepoolConfigConfidentialNodes) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpProject) GetClusters() ([]interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	creds, err := provider.Credentials(container.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	containerSvc, err := container.NewClusterManagerClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}

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
			mqlNodePool, err := createMqlNodePool(g.MotorRuntime, np, c.Id)
			if err != nil {
				return nil, err
			}
			nodePools = append(nodePools, mqlNodePool)
		}

		mqlCluster, err := g.MotorRuntime.CreateResource("gcp.cluster",
			"projectId", projectId,
			"id", c.Id,
			"name", c.Name,
			"description", c.Description,
			"loggingService", c.LoggingService,
			"monitoringService", c.MonitoringService,
			"network", c.Network,
			"clusterIpv4Cidr", c.ClusterIpv4Cidr,
			"subnetwork", c.Subnetwork,
			"nodePools", nodePools,
			"locations", core.StrSliceToInterface(c.Locations),
			"enableKubernetesAlpha", c.EnableKubernetesAlpha,
			"autopilotEnabled", c.Autopilot.Enabled,
			"zone", c.Zone,
			"endpoint", c.Endpoint,
			"initialClusterVersion", c.InitialClusterVersion,
			"currentMasterVersion", c.CurrentMasterVersion,
			"status", c.Status.String(),
			"resourceLabels", core.StrMapToInterface(c.ResourceLabels),
			"created", parseTime(c.CreateTime),
			"expirationTime", parseTime(c.ExpireTime),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCluster)
	}

	return res, nil
}

func createMqlNodePool(runtime *resources.Runtime, np *containerpb.NodePool, clusterId string) (resources.ResourceType, error) {
	nodePoolId := fmt.Sprintf("%s/%s", clusterId, np.Name)

	mqlPoolConfig, err := createMqlNodePoolConfig(runtime, np, nodePoolId)
	if err != nil {
		return nil, err
	}

	mqlPoolNetworkConfig, err := createMqlNodePoolNetworkConfig(runtime, np, nodePoolId)
	if err != nil {
		return nil, err
	}

	return runtime.CreateResource("gcp.cluster.nodepool",
		"id", nodePoolId,
		"name", np.Name,
		"config", mqlPoolConfig,
		"initialNodeCount", int64(np.InitialNodeCount),
		"locations", core.StrSliceToInterface(np.Locations),
		"networkConfig", mqlPoolNetworkConfig,
		"version", np.Version,
		"instanceGroupUrls", core.StrSliceToInterface(np.InstanceGroupUrls),
		"status", np.Status.String(),
	)
}

func createMqlNodePoolConfig(runtime *resources.Runtime, np *containerpb.NodePool, nodePoolId string) (resources.ResourceType, error) {
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
		mqlNodeTaint, err := runtime.CreateResource("gcp.cluster.nodepool.config.nodeTaint",
			"id", fmt.Sprintf("%s/taints/%d", nodePoolId, i),
			"key", taint.Key,
			"value", taint.Value,
			"effect", taint.Effect.String(),
		)
		if err != nil {
			return nil, err
		}
		nodeTaints = append(nodeTaints, mqlNodeTaint)
	}

	var mqlSandboxCfg resources.ResourceType
	if cfg.SandboxConfig != nil {
		mqlSandboxCfg, err = runtime.CreateResource("gcp.cluster.nodepool.config.sandbox",
			"id", fmt.Sprintf("%s/sandbox", nodePoolId),
			"type", cfg.SandboxConfig.Type.String(),
		)
		if err != nil {
			return nil, err
		}
	}

	var mqlShieldedInstanceCfg resources.ResourceType
	if cfg.ShieldedInstanceConfig != nil {
		mqlShieldedInstanceCfg, err = runtime.CreateResource("gcp.cluster.nodepool.config.shieldedInstanceConfig",
			"id", fmt.Sprintf("%s/shieldedInstanceConfig", nodePoolId),
			"enableSecureBoot", cfg.ShieldedInstanceConfig.EnableSecureBoot,
			"enableIntegrityMonitoring", cfg.ShieldedInstanceConfig.EnableIntegrityMonitoring,
		)
		if err != nil {
			return nil, err
		}
	}

	var mqlLinuxNodeCfg resources.ResourceType
	if cfg.LinuxNodeConfig != nil {
		mqlLinuxNodeCfg, err = runtime.CreateResource("gcp.cluster.nodepool.config.linuxNodeConfig",
			"id", fmt.Sprintf("%s/linuxNodeConfig", nodePoolId),
			"sysctls", core.StrMapToInterface(cfg.LinuxNodeConfig.Sysctls),
		)
		if err != nil {
			return nil, err
		}
	}

	var mqlKubeletCfg resources.ResourceType
	if cfg.KubeletConfig != nil {
		mqlKubeletCfg, err = runtime.CreateResource("gcp.cluster.nodepool.config.kubeletConfig",
			"id", fmt.Sprintf("%s/kubeletConfig", nodePoolId),
			"cpuManagerPolicy", cfg.KubeletConfig.CpuManagerPolicy,
			"cpuCfsQuotaPeriod", cfg.KubeletConfig.CpuCfsQuotaPeriod,
			"podPidsLimit", cfg.KubeletConfig.PodPidsLimit,
		)
		if err != nil {
			return nil, err
		}
	}

	var mqlGcfsCfg resources.ResourceType
	if cfg.GcfsConfig != nil {
		mqlGcfsCfg, err = runtime.CreateResource("gcp.cluster.nodepool.config.gcfsConfig",
			"id", fmt.Sprintf("%s/gcfsConfig", nodePoolId),
			"enabled", cfg.GcfsConfig.Enabled,
		)
		if err != nil {
			return nil, err
		}
	}

	var mqlAdvancedMachineFeatures resources.ResourceType
	if cfg.AdvancedMachineFeatures != nil {
		mqlAdvancedMachineFeatures, err = runtime.CreateResource("gcp.cluster.nodepool.config.advancedMachineFeatures",
			"id", fmt.Sprintf("%s/advancedMachineFeatures", nodePoolId),
			"threadsPerCore", cfg.Gvnic.Enabled,
		)
		if err != nil {
			return nil, err
		}
	}

	var mqlGvnicCfg resources.ResourceType
	if cfg.GcfsConfig != nil {
		mqlGvnicCfg, err = runtime.CreateResource("gcp.cluster.nodepool.config.gvnicConfig",
			"id", fmt.Sprintf("%s/gvnicConfig", nodePoolId),
			"enabled", cfg.Gvnic.Enabled,
		)
		if err != nil {
			return nil, err
		}
	}

	var mqlConfidentialNodes resources.ResourceType
	if cfg.ConfidentialNodes != nil {
		mqlConfidentialNodes, err = runtime.CreateResource("gcp.cluster.nodepool.config.confidentialNodes",
			"id", fmt.Sprintf("%s/confidentialNodes", nodePoolId),
			"enabled", cfg.ConfidentialNodes.Enabled,
		)
		if err != nil {
			return nil, err
		}
	}

	return runtime.CreateResource("gcp.cluster.nodepool.config",
		"id", fmt.Sprintf("%s/config", nodePoolId),
		"machineType", cfg.MachineType,
		"diskSizeGb", int64(cfg.DiskSizeGb),
		"oauthScopes", core.StrSliceToInterface(cfg.OauthScopes),
		"serviceAccount", cfg.ServiceAccount,
		"metadata", core.StrMapToInterface(cfg.Metadata),
		"imageType", cfg.ImageType,
		"labels", core.StrMapToInterface(cfg.Labels),
		"localSsdCount", int64(cfg.LocalSsdCount),
		"tags", core.StrSliceToInterface(cfg.Tags),
		"preemptible", cfg.Preemptible,
		"accelerators", mqlAccelerators,
		"diskType", cfg.DiskType,
		"minCpuPlatform", cfg.MinCpuPlatform,
		"workloadMetadataMode", cfg.WorkloadMetadataConfig.Mode.String(),
		"taints", nodeTaints,
		"sandboxConfig", mqlSandboxCfg,
		"shieldedInstanceConfig", mqlShieldedInstanceCfg,
		"linuxNodeConfig", mqlLinuxNodeCfg,
		"kubeletConfig", mqlKubeletCfg,
		"bootDiskKmsKey", cfg.BootDiskKmsKey,
		"gcfsConfig", mqlGcfsCfg,
		"gvnicConfig", mqlGvnicCfg,
		"advancedMachineFeatures", mqlAdvancedMachineFeatures,
		"spot", cfg.Spot,
		"confidentialNodes", mqlConfidentialNodes,
	)
}

func createMqlNodePoolNetworkConfig(runtime *resources.Runtime, np *containerpb.NodePool, nodePoolId string) (resources.ResourceType, error) {
	netCfg := np.NetworkConfig
	if netCfg == nil {
		return nil, nil
	}

	netCfgId := fmt.Sprintf("%s/networkConfig", nodePoolId)

	var performanceConfig resources.ResourceType
	var err error
	if netCfg.NetworkPerformanceConfig != nil {
		performanceConfig, err = runtime.CreateResource("gcp.cluster.nodepool.networkConfig.performanceConfig",
			"id", fmt.Sprintf("%s/performanceConfig", netCfgId),
			"totalEgressBandwidthTier", netCfg.NetworkPerformanceConfig.TotalEgressBandwidthTier.String(),
		)
		if err != nil {
			return nil, err
		}
	}

	return runtime.CreateResource("gcp.cluster.nodepool.networkConfig",
		"id", netCfgId,
		"podRange", netCfg.PodRange,
		"podIpv4CidrBlock", netCfg.PodIpv4CidrBlock,
		"performanceConfig", performanceConfig,
	)
}

func createMqlAccelerator(runtime *resources.Runtime, acc *containerpb.AcceleratorConfig, nodePoolId string, i int) (resources.ResourceType, error) {
	accId := fmt.Sprintf("%s/accelerators/%d", nodePoolId, i)

	var gpuSharingConfig resources.ResourceType
	var err error
	if acc.GpuSharingConfig != nil {
		gpuSharingConfig, err = runtime.CreateResource("gcp.cluster.nodepool.config.gpuSharingConfig",
			"id", fmt.Sprintf("%s/gpuSharingConfig", accId),
			"maxSharedClientsPerGpu", acc.GpuSharingConfig.MaxSharedClientsPerGpu,
			"strategy", acc.GpuSharingConfig.GpuSharingStrategy.String(),
		)
		if err != nil {
			return nil, err
		}
	}

	return runtime.CreateResource("gcp.cluster.nodepool.config.accelerator",
		"id", accId,
		"count", acc.AcceleratorCount,
		"type", acc.AcceleratorType,
		"gpuPartitionSize", acc.GpuPartitionSize,
		"gpuSharingConfig", gpuSharingConfig,
	)
}
