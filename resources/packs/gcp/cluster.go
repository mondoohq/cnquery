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

func (g *mqlGcpProjectGkeService) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gcp.project.gkeService", projectId), nil
}

func (g *mqlGcpProject) GetGke() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.gkeService",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectGkeServiceCluster) id() (string, error) {
	id, err := g.Id()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (g *mqlGcpProjectGkeServiceCluster) init(args *resources.Args) (*resources.Args, GcpProjectGkeServiceCluster, error) {
	if len(*args) > 3 {
		return args, nil, nil
	}

	if ids := getAssetIdentifier(g.MotorRuntime); ids != nil {
		(*args)["name"] = ids.name
		(*args)["location"] = ids.region
		(*args)["projectId"] = ids.project
	}

	obj, err := g.MotorRuntime.CreateResource("gcp.project.gkeService", "projectId", (*args)["projectId"])
	if err != nil {
		return nil, nil, err
	}
	gkeSvc := obj.(GcpProjectGkeService)
	clusters, err := gkeSvc.Clusters()
	if err != nil {
		return nil, nil, err
	}

	for _, c := range clusters {
		cluster := c.(GcpProjectGkeServiceCluster)
		name, err := cluster.Name()
		if err != nil {
			return nil, nil, err
		}
		projectId, err := cluster.ProjectId()
		if err != nil {
			return nil, nil, err
		}
		location, err := cluster.Location()
		if err != nil {
			return nil, nil, err
		}

		if name == (*args)["name"] && projectId == (*args)["projectId"] && location == (*args)["location"] {
			return args, cluster, nil
		}
	}
	return nil, nil, &resources.ResourceNotFound{}
}

func (g *mqlGcpProjectGkeServiceClusterNodepool) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfig) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolNetworkConfig) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolNetworkConfigPerformanceConfig) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigAccelerator) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigAcceleratorGpuSharingConfig) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigNodeTaint) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigSandboxConfig) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigShieldedInstanceConfig) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigLinuxNodeConfig) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigKubeletConfig) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigGcfsConfig) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigAdvancedMachineFeatures) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigGvnicConfig) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectGkeServiceClusterNodepoolConfigConfidentialNodes) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectGkeService) GetClusters() ([]interface{}, error) {
	projectId, err := g.ProjectId()
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
			mqlNodePool, err := createMqlNodePool(g.MotorRuntime, np, c.Id)
			if err != nil {
				return nil, err
			}
			nodePools = append(nodePools, mqlNodePool)
		}

		autopilotEnabled := false
		if c.Autopilot != nil {
			autopilotEnabled = c.Autopilot.Enabled
		}

		mqlCluster, err := g.MotorRuntime.CreateResource("gcp.project.gkeService.cluster",
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
			"autopilotEnabled", autopilotEnabled,
			"zone", c.Zone,
			"location", c.Location,
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

	return runtime.CreateResource("gcp.project.gkeService.cluster.nodepool",
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
		mqlNodeTaint, err := runtime.CreateResource("gcp.project.gkeService.cluster.nodepool.config.nodeTaint",
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
		mqlSandboxCfg, err = runtime.CreateResource("gcp.project.gkeService.cluster.nodepool.config.sandbox",
			"id", fmt.Sprintf("%s/sandbox", nodePoolId),
			"type", cfg.SandboxConfig.Type.String(),
		)
		if err != nil {
			return nil, err
		}
	}

	var mqlShieldedInstanceCfg resources.ResourceType
	if cfg.ShieldedInstanceConfig != nil {
		mqlShieldedInstanceCfg, err = runtime.CreateResource("gcp.project.gkeService.cluster.nodepool.config.shieldedInstanceConfig",
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
		mqlLinuxNodeCfg, err = runtime.CreateResource("gcp.project.gkeService.cluster.nodepool.config.linuxNodeConfig",
			"id", fmt.Sprintf("%s/linuxNodeConfig", nodePoolId),
			"sysctls", core.StrMapToInterface(cfg.LinuxNodeConfig.Sysctls),
		)
		if err != nil {
			return nil, err
		}
	}

	var mqlKubeletCfg resources.ResourceType
	if cfg.KubeletConfig != nil {
		mqlKubeletCfg, err = runtime.CreateResource("gcp.project.gkeService.cluster.nodepool.config.kubeletConfig",
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
		mqlGcfsCfg, err = runtime.CreateResource("gcp.project.gkeService.cluster.nodepool.config.gcfsConfig",
			"id", fmt.Sprintf("%s/gcfsConfig", nodePoolId),
			"enabled", cfg.GcfsConfig.Enabled,
		)
		if err != nil {
			return nil, err
		}
	}

	var mqlAdvancedMachineFeatures resources.ResourceType
	if cfg.AdvancedMachineFeatures != nil {
		mqlAdvancedMachineFeatures, err = runtime.CreateResource("gcp.project.gkeService.cluster.nodepool.config.advancedMachineFeatures",
			"id", fmt.Sprintf("%s/advancedMachineFeatures", nodePoolId),
			"threadsPerCore", cfg.Gvnic.Enabled,
		)
		if err != nil {
			return nil, err
		}
	}

	var mqlGvnicCfg resources.ResourceType
	if cfg.GcfsConfig != nil {
		mqlGvnicCfg, err = runtime.CreateResource("gcp.project.gkeService.cluster.nodepool.config.gvnicConfig",
			"id", fmt.Sprintf("%s/gvnicConfig", nodePoolId),
			"enabled", cfg.Gvnic.Enabled,
		)
		if err != nil {
			return nil, err
		}
	}

	var mqlConfidentialNodes resources.ResourceType
	if cfg.ConfidentialNodes != nil {
		mqlConfidentialNodes, err = runtime.CreateResource("gcp.project.gkeService.cluster.nodepool.config.confidentialNodes",
			"id", fmt.Sprintf("%s/confidentialNodes", nodePoolId),
			"enabled", cfg.ConfidentialNodes.Enabled,
		)
		if err != nil {
			return nil, err
		}
	}

	workloadMetadataMode := ""
	if cfg.WorkloadMetadataConfig != nil {
		workloadMetadataMode = cfg.WorkloadMetadataConfig.Mode.String()
	}

	return runtime.CreateResource("gcp.project.gkeService.cluster.nodepool.config",
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
		"workloadMetadataMode", workloadMetadataMode,
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
		performanceConfig, err = runtime.CreateResource("gcp.project.gkeService.cluster.nodepool.networkConfig.performanceConfig",
			"id", fmt.Sprintf("%s/performanceConfig", netCfgId),
			"totalEgressBandwidthTier", netCfg.NetworkPerformanceConfig.TotalEgressBandwidthTier.String(),
		)
		if err != nil {
			return nil, err
		}
	}

	return runtime.CreateResource("gcp.project.gkeService.cluster.nodepool.networkConfig",
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
		gpuSharingConfig, err = runtime.CreateResource("gcp.project.gkeService.cluster.nodepool.config.gpuSharingConfig",
			"id", fmt.Sprintf("%s/gpuSharingConfig", accId),
			"maxSharedClientsPerGpu", acc.GpuSharingConfig.MaxSharedClientsPerGpu,
			"strategy", acc.GpuSharingConfig.GpuSharingStrategy.String(),
		)
		if err != nil {
			return nil, err
		}
	}

	return runtime.CreateResource("gcp.project.gkeService.cluster.nodepool.config.accelerator",
		"id", accId,
		"count", acc.AcceleratorCount,
		"type", acc.AcceleratorType,
		"gpuPartitionSize", acc.GpuPartitionSize,
		"gpuSharingConfig", gpuSharingConfig,
	)
}
