package gcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/dataproc/v1"
	"google.golang.org/api/option"
)

func (g *mqlGcpProjectDataprocService) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gcp.project.dataprocService", projectId), nil
}

func (g *mqlGcpProject) GetDataproc() (interface{}, error) {
	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.dataprocService",
		"projectId", projectId,
	)
}

func (g *mqlGcpProjectDataprocService) GetRegions() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(dataproc.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	computeSvc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	regions, err := computeSvc.Regions.List(projectId).Do()
	if err != nil {
		return nil, err
	}

	regionNames := make([]interface{}, 0, len(regions.Items))
	for _, region := range regions.Items {
		regionNames = append(regionNames, region.Name)
	}
	return regionNames, nil
}

func (g *mqlGcpProjectDataprocService) GetClusters() ([]interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	regions, err := g.GetRegions()
	if err != nil {
		return nil, err
	}

	client, err := provider.Client(dataproc.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	dataprocSvc, err := dataproc.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var mqlClusters []interface{}
	wg.Add(len(regions))
	mux := &sync.Mutex{}
	for _, region := range regions {
		go func(projectId, regionName string) {
			defer wg.Done()
			clusters, err := dataprocSvc.Projects.Regions.Clusters.List(projectId, regionName).Do()
			if err != nil {
				log.Error().Err(err).Send()
			}

			for _, c := range clusters.Clusters {
				var mqlConfig resources.ResourceType
				if c.Config != nil {
					var mqlAutoscalingCfg map[string]interface{}
					if c.Config.AutoscalingConfig != nil {
						type mqlAutoscalingConfig struct {
							PolicyUri string `json:"policyUri"`
						}
						mqlAutoscalingCfg, err = core.JsonToDict(mqlAutoscalingConfig{PolicyUri: c.Config.AutoscalingConfig.PolicyUri})
						if err != nil {
							log.Error().Err(err).Send()
						}
					}

					var mqlMetricsCfg map[string]interface{}
					if c.Config.DataprocMetricConfig != nil {
						type mqlMetricsConfigMetric struct {
							MetricOverrides []string `json:"metricOverrides"`
							MetricSource    string   `json:"metricSource"`
						}
						type mqlMetricsConfig struct {
							Metrics []mqlMetricsConfigMetric `json:"metrics"`
						}

						mqlMetricsConfigs := make([]mqlMetricsConfigMetric, 0, len(c.Config.DataprocMetricConfig.Metrics))
						for _, m := range c.Config.DataprocMetricConfig.Metrics {
							mqlMetricsConfigs = append(mqlMetricsConfigs, mqlMetricsConfigMetric{
								MetricOverrides: m.MetricOverrides,
								MetricSource:    m.MetricSource,
							})
						}

						mqlMetricsCfg, err = core.JsonToDict(mqlMetricsConfig{Metrics: mqlMetricsConfigs})
						if err != nil {
							log.Error().Err(err).Send()
						}
					}

					mqlConfig, err = g.MotorRuntime.CreateResource("gcp.project.dataprocService.cluster.config",
						"parentResourcePath", fmt.Sprintf("%s/dataproc/%s", projectId, c.ClusterName),
						"autoscaling", mqlAutoscalingCfg,
						"configBucket", c.Config.ConfigBucket,
						"metrics", mqlMetricsCfg,
						"encryption", nil, // TODO
						"endpoint", nil, // TODO
						"gceCluster", nil, // TODO
						"gkeCluster", nil, // TODO
						"initializationActions", nil, // TODO
						"lifecycle", nil, // TODO
						"master", nil, // TODO
						"metastore", nil, // TODO
						"secondaryWorker", nil, // TODO
						"security", nil, // TODO
						"software", nil, // TODO
						"tempBucket", c.Config.TempBucket,
						"worker", nil, // TODO
					)
					if err != nil {
						log.Error().Err(err).Send()
					}
				}

				var mqlMetrics map[string]interface{}
				if c.Metrics != nil {
					type mqlClusterMetrics struct {
						HdfsMetrics map[string]string `json:"hdfsMetrics"`
						YarnMetrics map[string]string `json:"yarnMetrics"`
					}
					mqlMetrics, err = core.JsonToDict(mqlClusterMetrics{HdfsMetrics: c.Metrics.HdfsMetrics, YarnMetrics: c.Metrics.YarnMetrics})
					if err != nil {
						log.Error().Err(err).Send()
					}
				}

				var mqlStatus resources.ResourceType
				if c.Status != nil {
					mqlStatus, err = g.MotorRuntime.CreateResource("gcp.project.dataprocService.cluster.status",
						"id", fmt.Sprintf("%s/dataproc/%s/status", projectId, c.ClusterName),
						"detail", c.Status.Detail,
						"state", c.Status.State,
						"started", parseTime(c.Status.StateStartTime),
						"substate", c.Status.Substate,
					)
					if err != nil {
						log.Error().Err(err).Send()
					}
				}

				mqlStatusHistory := make([]interface{}, 0, len(c.StatusHistory))
				for i, s := range c.StatusHistory {
					mqlStatus, err = g.MotorRuntime.CreateResource("gcp.project.dataprocService.cluster.status",
						"id", fmt.Sprintf("%s/dataproc/%s/status/%d", projectId, c.ClusterName, i),
						"detail", s.Detail,
						"state", s.State,
						"started", parseTime(s.StateStartTime),
						"substate", s.Substate,
					)
					if err != nil {
						log.Error().Err(err).Send()
					}
					mqlStatusHistory = append(mqlStatusHistory, mqlStatus)
				}

				var mqlVirtualClusterCfg resources.ResourceType
				if c.VirtualClusterConfig != nil {
					type mqlMetastoreConfig struct {
						DataprocMetastoreService string `json:"dataprocMetastoreService"`
					}
					type mqlSparkHistoryServerConfig struct {
						DataprocCluster string `json:"dataprocCluster"`
					}
					type mqlAuxiliaryServices struct {
						MetastoreConfig          mqlMetastoreConfig          `json:"metastoreConfig"`
						SparkHistoryServerConfig mqlSparkHistoryServerConfig `json:"sparkHistoryServerConfig"`
					}

					mqlAuxServices, err := core.JsonToDict(mqlAuxiliaryServices{
						MetastoreConfig:          mqlMetastoreConfig{DataprocMetastoreService: c.VirtualClusterConfig.AuxiliaryServicesConfig.MetastoreConfig.DataprocMetastoreService},
						SparkHistoryServerConfig: mqlSparkHistoryServerConfig{DataprocCluster: c.VirtualClusterConfig.AuxiliaryServicesConfig.SparkHistoryServerConfig.DataprocCluster},
					})
					if err != nil {
						log.Error().Err(err).Send()
					}

					type mqlNamespacedGkeDeploymentTarget struct {
						ClusterNamespace string `json:"clusterNamespace"`
						TargetGkeCluster string `json:"targetGkeCluster"`
					}
					type mqlGkeNodePoolAccelerator struct {
						AcceleratorCount int64  `json:"acceleratorCount"`
						AcceleratorType  string `json:"acceleratorType"`
						GpuPartitionSize string `json:"gpuPartitionSize"`
					}
					type mqlGkeNodePoolAutoscalingConfig struct {
						MaxNodeCount int64 `json:"maxNodeCount"`
						MinNodeCount int64 `json:"minNodeCount"`
					}
					type mqlGkeNodeConfig struct {
						Accelerators   []mqlGkeNodePoolAccelerator `json:"accelerators"`
						BootDiskKmsKey string                      `json:"bootDiskKmsKey"`
						LocalSsdCount  int64                       `json:"localSsdCount"`
						MachineType    string                      `json:"machineType"`
						MinCpuPlatform string                      `json:"minCpuPlatform"`
						Preemptible    bool                        `json:"preemptible"`
						Spot           bool                        `json:"spot"`
					}
					type mqlGkeNodePoolConfig struct {
						Autoscaling mqlGkeNodePoolAutoscalingConfig `json:"autoscaling"`
						Config      mqlGkeNodeConfig                `json:"config"`
						Locations   []string                        `json:"locations"`
					}
					type mqlGkeNodePoolTarget struct {
						NodePool       string               `json:"nodePool"`
						NodePoolConfig mqlGkeNodePoolConfig `json:"nodePoolConfig"`
						Roles          []string             `json:"roles"`
					}
					type mqlGkeClusterConfig struct {
						TargetCluster                 string                           `json:"targetCluster"`
						NamespacedGkeDeploymentTarget mqlNamespacedGkeDeploymentTarget `json:"namespacedGkeDeploymentTarget"`
						NodePoolTarget                []mqlGkeNodePoolTarget           `json:"nodePoolTarget"`
					}
					type mqlKubernetesSoftwareConfig struct {
						ComponentVersion map[string]string `json:"componentVersion"`
						Properties       map[string]string `json:"properties"`
					}
					type mqlKubernetesClusterConfig struct {
						GkeClusterConfig    mqlGkeClusterConfig `json:"gkeClusterConfig"`
						KubernetesNamespace string              `json:"kubernetesNamespace"`
					}

					npTargets := make([]mqlGkeNodePoolTarget, 0, len(c.VirtualClusterConfig.KubernetesClusterConfig.GkeClusterConfig.NodePoolTarget))
					for _, np := range c.VirtualClusterConfig.KubernetesClusterConfig.GkeClusterConfig.NodePoolTarget {
						accs := make([]mqlGkeNodePoolAccelerator, 0, len(np.NodePoolConfig.Config.Accelerators))
						for _, acc := range np.NodePoolConfig.Config.Accelerators {
							accs = append(accs, mqlGkeNodePoolAccelerator{
								AcceleratorCount: acc.AcceleratorCount,
								AcceleratorType:  acc.AcceleratorType,
								GpuPartitionSize: acc.GpuPartitionSize,
							})
						}

						npTargets = append(npTargets, mqlGkeNodePoolTarget{
							NodePool: np.NodePool,
							NodePoolConfig: mqlGkeNodePoolConfig{
								Autoscaling: mqlGkeNodePoolAutoscalingConfig{
									MaxNodeCount: np.NodePoolConfig.Autoscaling.MaxNodeCount,
									MinNodeCount: np.NodePoolConfig.Autoscaling.MinNodeCount,
								},
								Config: mqlGkeNodeConfig{
									Accelerators:   accs,
									BootDiskKmsKey: np.NodePoolConfig.Config.BootDiskKmsKey,
									LocalSsdCount:  np.NodePoolConfig.Config.LocalSsdCount,
									MachineType:    np.NodePoolConfig.Config.MachineType,
									MinCpuPlatform: np.NodePoolConfig.Config.MinCpuPlatform,
									Preemptible:    np.NodePoolConfig.Config.Preemptible,
									Spot:           np.NodePoolConfig.Config.Spot,
								},
								Locations: np.NodePoolConfig.Locations,
							},
							Roles: np.Roles,
						})
					}

					mqlK8sClusterCfg, err := core.JsonToDict(mqlKubernetesClusterConfig{
						GkeClusterConfig: mqlGkeClusterConfig{
							TargetCluster: c.VirtualClusterConfig.KubernetesClusterConfig.GkeClusterConfig.GkeClusterTarget,
							NamespacedGkeDeploymentTarget: mqlNamespacedGkeDeploymentTarget{
								ClusterNamespace: c.VirtualClusterConfig.KubernetesClusterConfig.GkeClusterConfig.NamespacedGkeDeploymentTarget.ClusterNamespace,
								TargetGkeCluster: c.VirtualClusterConfig.KubernetesClusterConfig.GkeClusterConfig.NamespacedGkeDeploymentTarget.TargetGkeCluster,
							},
							NodePoolTarget: npTargets,
						},
					})
					if err != nil {
						log.Error().Err(err).Send()
					}

					mqlVirtualClusterCfg, err = g.MotorRuntime.CreateResource("gcp.project.dataprocService.cluster.virtualClusterConfig",
						"parentResourcePath", fmt.Sprintf("%s/dataproc/%s", projectId, c.ClusterName),
						"auxiliaryServices", mqlAuxServices,
						"kubernetesCluster", mqlK8sClusterCfg,
						"stagingBucket", c.VirtualClusterConfig.StagingBucket,
					)
					if err != nil {
						log.Error().Err(err).Send()
					}
				}

				mqlCluster, err := g.MotorRuntime.CreateResource("gcp.project.dataprocService.cluster",
					"projectId", projectId,
					"name", c.ClusterName,
					"uuid", c.ClusterUuid,
					"config", mqlConfig,
					"labels", core.StrMapToInterface(c.Labels),
					"metrics", mqlMetrics,
					"status", mqlStatus,
					"statusHistory", mqlStatusHistory,
					"virtualClusterConfig", mqlVirtualClusterCfg,
				)
				if err != nil {
					log.Error().Err(err).Send()
				}
				mux.Lock()
				mqlClusters = append(mqlClusters, mqlCluster)
				mux.Unlock()
			}
		}(projectId, region.(string))
	}
	wg.Wait()
	return mqlClusters, nil
}

func (g *mqlGcpProjectDataprocServiceCluster) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	name, err := g.Name()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/dataproc/%s", projectId, name), nil
}

func (g *mqlGcpProjectDataprocServiceClusterConfig) id() (string, error) {
	parentResource, err := g.ParentResourcePath()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/config", parentResource), nil
}

func (g *mqlGcpProjectDataprocServiceClusterStatus) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectDataprocServiceClusterVirtualClusterConfig) id() (string, error) {
	parentResource, err := g.ParentResourcePath()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/virtualClusterConfig", parentResource), nil
}

func (g *mqlGcpProjectDataprocServiceClusterConfigGceCluster) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectDataprocServiceClusterConfigGceClusterReservationAffinity) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectDataprocServiceClusterConfigGceClusterShieldedInstanceConfig) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectDataprocServiceClusterConfigGkeCluster) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectDataprocServiceClusterConfigLifecycle) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectDataprocServiceClusterConfigInstance) id() (string, error) {
	return g.Id()
}

func (g *mqlGcpProjectDataprocServiceClusterConfigInstanceDiskConfig) id() (string, error) {
	return g.Id()
}
