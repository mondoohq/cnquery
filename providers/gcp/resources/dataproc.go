// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

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
	"google.golang.org/api/serviceusage/v1"
)

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

func (g *mqlGcpProjectDataprocService) id() (string, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/gcp.project.dataprocService", projectId), nil
}

func (g *mqlGcpProject) GetDataproc() (interface{}, error) {
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	projectId, err := g.Id()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := provider.Client(dataproc.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	serviceUsageSvc, err := serviceusage.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("projects/%s/services/dataproc.googleapis.com", projectId)
	dataProcSvc, err := serviceUsageSvc.Services.Get(url).Do()
	if err != nil {
		return nil, err
	}
	enabled := dataProcSvc.State == "ENABLED"
	return g.MotorRuntime.CreateResource("gcp.project.dataprocService",
		"projectId", projectId,
		"enabled", enabled,
	)
}

func (g *mqlGcpProjectDataprocService) GetRegions() ([]interface{}, error) {
	// no check whether DataProc service is enabled here, this uses a different service
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
	enabled, err := g.Enabled()
	if err != nil {
		return nil, err
	}
	if !enabled {
		log.Warn().Msg("DataProc Cloud API is not enabled, not querying clusters")
		return []interface{}{}, nil
	}
	provider, err := gcpProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	regions, err := g.Regions()
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
				log.Error().Str("region", regionName).Err(err).Send()
			} else {
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

						var mqlEncryptionCfg map[string]interface{}
						if c.Config.EncryptionConfig != nil {
							type mqlEncryptionConfig struct {
								GcePdKmsKeyName string `json:"gcePdKmsKeyName"`
							}
							mqlEncryptionCfg, err = core.JsonToDict(mqlEncryptionConfig{GcePdKmsKeyName: c.Config.EncryptionConfig.GcePdKmsKeyName})
							if err != nil {
								log.Error().Err(err).Send()
							}
						}

						var mqlEndpointCfg map[string]interface{}
						if c.Config.EndpointConfig != nil {
							type mqlEndpointConfig struct {
								HttpPorts            map[string]string `json:"httpPorts"`
								EnableHttpPortAccess bool              `json:"enableHttpPortAccess"`
							}
							mqlEndpointCfg, err = core.JsonToDict(mqlEndpointConfig{
								HttpPorts:            c.Config.EndpointConfig.HttpPorts,
								EnableHttpPortAccess: c.Config.EndpointConfig.EnableHttpPortAccess,
							})
							if err != nil {
								log.Error().Err(err).Send()
							}
						}

						var mqlGceClusterCfg resources.ResourceType
						if c.Config.GceClusterConfig != nil {
							var mqlConfidentialInstanceCfg map[string]interface{}
							if c.Config.GceClusterConfig.ConfidentialInstanceConfig != nil {
								type mqlConfidentialInstanceConfig struct {
									EnableConfidentialCompute bool `json:"enableConfidentialCompute"`
								}
								mqlConfidentialInstanceCfg, err = core.JsonToDict(mqlConfidentialInstanceConfig{
									EnableConfidentialCompute: c.Config.GceClusterConfig.ConfidentialInstanceConfig.EnableConfidentialCompute,
								})
								if err != nil {
									log.Error().Err(err).Send()
								}
							}

							var mqlNodeGroupAffinityCfg map[string]interface{}
							if c.Config.GceClusterConfig.NodeGroupAffinity != nil {
								type mqlNodeGroupAffinityConfig struct {
									Uri string `json:"uri"`
								}
								mqlNodeGroupAffinityCfg, err = core.JsonToDict(mqlNodeGroupAffinityConfig{
									Uri: c.Config.GceClusterConfig.NodeGroupAffinity.NodeGroupUri,
								})
								if err != nil {
									log.Error().Err(err).Send()
								}
							}

							var mqlReservationAffinity resources.ResourceType
							if c.Config.GceClusterConfig.ReservationAffinity != nil {
								mqlReservationAffinity, err = g.MotorRuntime.CreateResource("gcp.project.dataprocService.cluster.config.gceCluster.reservationAffinity",
									"id", fmt.Sprintf("%s/dataproc/%s/config/gceCluster/reservationAffinity", projectId, c.ClusterName),
									"consumeReservationType", c.Config.GceClusterConfig.ReservationAffinity.ConsumeReservationType,
									"key", c.Config.GceClusterConfig.ReservationAffinity.Key,
									"values", core.StrSliceToInterface(c.Config.GceClusterConfig.ReservationAffinity.Values),
								)
								if err != nil {
									log.Error().Err(err).Send()
								}
							}

							var mqlShieldedCfg resources.ResourceType
							if c.Config.GceClusterConfig.ShieldedInstanceConfig != nil {
								mqlShieldedCfg, err = g.MotorRuntime.CreateResource("gcp.project.dataprocService.cluster.config.gceCluster.shieldedInstanceConfig",
									"id", fmt.Sprintf("%s/dataproc/%s/config/gceCluster/shieldedInstanceConfig", projectId, c.ClusterName),
									"enableIntegrityMonitoring", c.Config.GceClusterConfig.ShieldedInstanceConfig.EnableIntegrityMonitoring,
									"enableSecureBoot", c.Config.GceClusterConfig.ShieldedInstanceConfig.EnableSecureBoot,
									"enableVtpm", c.Config.GceClusterConfig.ShieldedInstanceConfig.EnableVtpm,
								)
								if err != nil {
									log.Error().Err(err).Send()
								}
							}

							mqlGceClusterCfg, err = g.MotorRuntime.CreateResource("gcp.project.dataprocService.cluster.config.gceCluster",
								"id", fmt.Sprintf("%s/dataproc/%s/config/gceCluster", projectId, c.ClusterName),
								"projectId", projectId,
								"confidentialInstance", mqlConfidentialInstanceCfg,
								"internalIpOnly", c.Config.GceClusterConfig.InternalIpOnly,
								"metadata", core.StrMapToInterface(c.Config.GceClusterConfig.Metadata),
								"networkUri", c.Config.GceClusterConfig.NetworkUri,
								"nodeGroupAffinity", mqlNodeGroupAffinityCfg,
								"privateIpv6GoogleAccess", c.Config.GceClusterConfig.PrivateIpv6GoogleAccess,
								"reservationAffinity", mqlReservationAffinity,
								"serviceAccountEmail", c.Config.GceClusterConfig.ServiceAccount,
								"serviceAccountScopes", core.StrSliceToInterface(c.Config.GceClusterConfig.ServiceAccountScopes),
								"shieldedInstanceConfig", mqlShieldedCfg,
								"subnetworkUri", c.Config.GceClusterConfig.SubnetworkUri,
								"tags", core.StrSliceToInterface(c.Config.GceClusterConfig.Tags),
								"zoneUri", c.Config.GceClusterConfig.ZoneUri,
							)
							if err != nil {
								log.Error().Err(err).Send()
							}
						}

						var mqlGkeClusterCfg resources.ResourceType
						if c.Config.GkeClusterConfig != nil {
							mqlNodePools := make([]mqlGkeNodePoolTarget, 0, len(c.Config.GkeClusterConfig.NodePoolTarget))
							for _, npt := range c.Config.GkeClusterConfig.NodePoolTarget {
								mqlNodePools = append(mqlNodePools, nodePoolTargetToMql(npt))
							}
							nodePoolsDict, err := core.JsonToDictSlice(mqlNodePools)
							if err != nil {
								log.Error().Err(err).Send()
							}
							mqlGkeClusterCfg, err = g.MotorRuntime.CreateResource("gcp.project.dataprocService.cluster.config.gkeCluster",
								"id", fmt.Sprintf("%s/dataproc/%s/config/gkeCluster", projectId, c.ClusterName),
								"gkeClusterTarget", c.Config.GkeClusterConfig.GkeClusterTarget,
								"nodePoolTarget", nodePoolsDict,
							)
							if err != nil {
								log.Error().Err(err).Send()
							}
						}

						type mqlNodeInitializationACtion struct {
							ExecutableFile   string `json:"executableFile"`
							ExecutionTimeout string `json:"executionTimeout"`
						}
						initActions := make([]mqlNodeInitializationACtion, 0, len(c.Config.InitializationActions))
						for _, ia := range c.Config.InitializationActions {
							initActions = append(initActions, mqlNodeInitializationACtion{
								ExecutableFile:   ia.ExecutableFile,
								ExecutionTimeout: ia.ExecutionTimeout,
							})
						}

						dictInitActions, err := core.JsonToDictSlice(initActions)
						if err != nil {
							log.Error().Err(err).Send()
						}

						var mqlLifecycleCfg resources.ResourceType
						if c.Config.LifecycleConfig != nil {
							mqlLifecycleCfg, err = g.MotorRuntime.CreateResource("gcp.project.dataprocService.cluster.config.lifecycle",
								"id", fmt.Sprintf("%s/dataproc/%s/config/lifecycle", projectId, c.ClusterName),
								"autoDeleteTime", c.Config.LifecycleConfig.AutoDeleteTime,
								"autoDeleteTtl", c.Config.LifecycleConfig.AutoDeleteTtl,
								"idleDeleteTtl", c.Config.LifecycleConfig.IdleDeleteTtl,
								"idleStartTime", c.Config.LifecycleConfig.IdleStartTime,
							)
							if err != nil {
								log.Error().Err(err).Send()
							}
						}

						mqlMasterCfg, err := instaceGroupConfigToMql(
							g.MotorRuntime, c.Config.MasterConfig, fmt.Sprintf("%s/dataproc/%s/config/master", projectId, c.ClusterName))
						if err != nil {
							log.Error().Err(err).Send()
						}

						type mqlMetastoreConfig struct {
							DataprocMetastoreService string `json:"dataprocMetastoreService"`
						}
						var mqlMetastoreCfg map[string]interface{}
						if c.Config.MetastoreConfig != nil {
							mqlMetastoreCfg, err = core.JsonToDict(mqlMetastoreConfig{
								DataprocMetastoreService: c.Config.MetastoreConfig.DataprocMetastoreService,
							})
							if err != nil {
								log.Error().Err(err).Send()
							}
						}

						mqlSecondaryWorkerCfg, err := instaceGroupConfigToMql(
							g.MotorRuntime, c.Config.SecondaryWorkerConfig, fmt.Sprintf("%s/dataproc/%s/config/secondaryWorker", projectId, c.ClusterName))
						if err != nil {
							log.Error().Err(err).Send()
						}

						type mqlSecurityIdentity struct {
							UserServiceAccountMapping map[string]string `json:"userServiceAccountMapping"`
						}
						type mqlSecurityKerberos struct {
							CrossRealmTrustAdminServer       string `json:"crossRealmTrustAdminServer"`
							CrossRealmTrustKdc               string `json:"crossRealmTrustKdc"`
							CrossRealmTrustRealm             string `json:"crossRealmTrustRealm"`
							CrossRealmTrustSharedPasswordUri string `json:"crossRealmTrustSharedPasswordUri"`
							EnableKerberos                   bool   `json:"enableKerberos"`
							KdcDbKeyUri                      string `json:"kdcDbKeyUri"`
							KeyPasswordUri                   string `json:"keyPasswordUri"`
							KeystorePasswordUri              string `json:"keystorePasswordUri"`
							KeystoreUri                      string `json:"keystoreUri"`
							KmsKeyUri                        string `json:"kmsKeyUri"`
							Realm                            string `json:"realm"`
							RootPrincipalPasswordUri         string `json:"rootPrincipalPasswordUri"`
							TgtLifetimeHours                 int64  `json:"tgtLifetimeHours"`
							TruststorePasswordUri            string `json:"truststorePasswordUri"`
							TruststoreUri                    string `json:"truststoreUri"`
						}
						type mqlSecurityConfig struct {
							IdentityConfig mqlSecurityIdentity `json:"identityConfig,omitempty"`
							KerberosConfig mqlSecurityKerberos `json:"kerberosConfig,omitempty"`
						}
						var mqlSecurityCfg map[string]interface{}
						if c.Config.SecurityConfig != nil {
							cfg := mqlSecurityConfig{}
							if c.Config.SecurityConfig.IdentityConfig != nil {
								cfg.IdentityConfig = mqlSecurityIdentity{
									UserServiceAccountMapping: c.Config.SecurityConfig.IdentityConfig.UserServiceAccountMapping,
								}
							}
							if c.Config.SecurityConfig.KerberosConfig != nil {
								cfg.KerberosConfig = mqlSecurityKerberos{
									CrossRealmTrustAdminServer:       c.Config.SecurityConfig.KerberosConfig.CrossRealmTrustAdminServer,
									CrossRealmTrustKdc:               c.Config.SecurityConfig.KerberosConfig.CrossRealmTrustKdc,
									CrossRealmTrustRealm:             c.Config.SecurityConfig.KerberosConfig.CrossRealmTrustRealm,
									CrossRealmTrustSharedPasswordUri: c.Config.SecurityConfig.KerberosConfig.CrossRealmTrustSharedPasswordUri,
									EnableKerberos:                   c.Config.SecurityConfig.KerberosConfig.EnableKerberos,
									KdcDbKeyUri:                      c.Config.SecurityConfig.KerberosConfig.KdcDbKeyUri,
									KeyPasswordUri:                   c.Config.SecurityConfig.KerberosConfig.KeyPasswordUri,
									KeystorePasswordUri:              c.Config.SecurityConfig.KerberosConfig.KeystorePasswordUri,
									KeystoreUri:                      c.Config.SecurityConfig.KerberosConfig.KeystoreUri,
									KmsKeyUri:                        c.Config.SecurityConfig.KerberosConfig.KmsKeyUri,
									Realm:                            c.Config.SecurityConfig.KerberosConfig.Realm,
									RootPrincipalPasswordUri:         c.Config.SecurityConfig.KerberosConfig.RootPrincipalPasswordUri,
									TgtLifetimeHours:                 c.Config.SecurityConfig.KerberosConfig.TgtLifetimeHours,
									TruststorePasswordUri:            c.Config.SecurityConfig.KerberosConfig.TruststorePasswordUri,
									TruststoreUri:                    c.Config.SecurityConfig.KerberosConfig.TruststoreUri,
								}
								mqlSecurityCfg, err = core.JsonToDict(cfg)
								if err != nil {
									log.Error().Err(err).Send()
								}
							}
						}

						type mqlSoftwareConfig struct {
							ImageVersion       string   `json:"imageVersion"`
							OptionalComponents []string `json:"optionalComponents"`
						}
						var mqlSoftwareCfg map[string]interface{}
						if c.Config.SoftwareConfig != nil {
							mqlSoftwareCfg, err = core.JsonToDict(mqlSoftwareConfig{
								ImageVersion:       c.Config.SoftwareConfig.ImageVersion,
								OptionalComponents: c.Config.SoftwareConfig.OptionalComponents,
							})
							if err != nil {
								log.Error().Err(err).Send()
							}
						}

						mqlWorkerCfg, err := instaceGroupConfigToMql(
							g.MotorRuntime, c.Config.WorkerConfig, fmt.Sprintf("%s/dataproc/%s/config/worker", projectId, c.ClusterName))
						if err != nil {
							log.Error().Err(err).Send()
						}

						mqlConfig, err = g.MotorRuntime.CreateResource("gcp.project.dataprocService.cluster.config",
							"projectId", c.ProjectId,
							"parentResourcePath", fmt.Sprintf("%s/dataproc/%s", projectId, c.ClusterName),
							"autoscaling", mqlAutoscalingCfg,
							"configBucket", c.Config.ConfigBucket,
							"metrics", mqlMetricsCfg,
							"encryption", mqlEncryptionCfg,
							"endpoint", mqlEndpointCfg,
							"gceCluster", mqlGceClusterCfg,
							"gkeCluster", mqlGkeClusterCfg,
							"initializationActions", dictInitActions,
							"lifecycle", mqlLifecycleCfg,
							"master", mqlMasterCfg,
							"metastore", mqlMetastoreCfg,
							"secondaryWorker", mqlSecondaryWorkerCfg,
							"security", mqlSecurityCfg,
							"software", mqlSoftwareCfg,
							"tempBucket", c.Config.TempBucket,
							"worker", mqlWorkerCfg,
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
						for _, npt := range c.VirtualClusterConfig.KubernetesClusterConfig.GkeClusterConfig.NodePoolTarget {
							npTargets = append(npTargets, nodePoolTargetToMql(npt))
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
			}
		}(projectId, region.(string))
	}
	wg.Wait()
	return mqlClusters, nil
}

func (g *mqlGcpProjectDataprocServiceClusterConfigGceCluster) GetServiceAccount() (interface{}, error) {
	projectId, err := g.ProjectId()
	if err != nil {
		return nil, err
	}

	email, err := g.ServiceAccountEmail()
	if err != nil {
		return nil, err
	}

	return g.MotorRuntime.CreateResource("gcp.project.iamService.serviceAccount",
		"projectId", projectId,
		"email", email,
	)
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

func nodePoolTargetToMql(npt *dataproc.GkeNodePoolTarget) mqlGkeNodePoolTarget {
	accs := make([]mqlGkeNodePoolAccelerator, 0, len(npt.NodePoolConfig.Config.Accelerators))
	for _, acc := range npt.NodePoolConfig.Config.Accelerators {
		accs = append(accs, mqlGkeNodePoolAccelerator{
			AcceleratorCount: acc.AcceleratorCount,
			AcceleratorType:  acc.AcceleratorType,
			GpuPartitionSize: acc.GpuPartitionSize,
		})
	}

	return mqlGkeNodePoolTarget{
		NodePool: npt.NodePool,
		NodePoolConfig: mqlGkeNodePoolConfig{
			Autoscaling: mqlGkeNodePoolAutoscalingConfig{
				MaxNodeCount: npt.NodePoolConfig.Autoscaling.MaxNodeCount,
				MinNodeCount: npt.NodePoolConfig.Autoscaling.MinNodeCount,
			},
			Config: mqlGkeNodeConfig{
				Accelerators:   accs,
				BootDiskKmsKey: npt.NodePoolConfig.Config.BootDiskKmsKey,
				LocalSsdCount:  npt.NodePoolConfig.Config.LocalSsdCount,
				MachineType:    npt.NodePoolConfig.Config.MachineType,
				MinCpuPlatform: npt.NodePoolConfig.Config.MinCpuPlatform,
				Preemptible:    npt.NodePoolConfig.Config.Preemptible,
				Spot:           npt.NodePoolConfig.Config.Spot,
			},
			Locations: npt.NodePoolConfig.Locations,
		},
		Roles: npt.Roles,
	}
}

func instaceGroupConfigToMql(runtime *resources.Runtime, igc *dataproc.InstanceGroupConfig, id string) (resources.ResourceType, error) {
	if igc == nil {
		return nil, nil
	}
	type mqlAccelerator struct {
		AcceleratorCount   int64  `json:"acceleratorCount"`
		AcceleratorTypeUri string `json:"acceleratorTypeUri"`
	}
	accs := make([]mqlAccelerator, 0, len(igc.Accelerators))
	for _, acc := range igc.Accelerators {
		accs = append(accs, mqlAccelerator{
			AcceleratorCount:   acc.AcceleratorCount,
			AcceleratorTypeUri: acc.AcceleratorTypeUri,
		})
	}
	mqlAccs, err := core.JsonToDictSlice(accs)
	if err != nil {
		return nil, err
	}

	var mqlDiskCfg resources.ResourceType
	if igc.DiskConfig != nil {
		mqlDiskCfg, err = runtime.CreateResource("gcp.project.dataprocService.cluster.config.instance.diskConfig",
			"id", fmt.Sprintf("%s/diskConfig", id),
			"bootDiskSizeGb", igc.DiskConfig.BootDiskSizeGb,
			"bootDiskType", igc.DiskConfig.BootDiskType,
			"localSsdInterface", igc.DiskConfig.LocalSsdInterface,
			"numLocalSsds", igc.DiskConfig.NumLocalSsds,
		)
		if err != nil {
			return nil, err
		}
	}

	type mqlInstanceReference struct {
		InstanceId     string `json:"instanceId"`
		InstanceName   string `json:"instanceName"`
		PublicEciesKey string `json:"publicEciesKey"`
		PublicKey      string `json:"publicKey"`
	}
	instanceReferences := make([]mqlInstanceReference, 0, len(igc.InstanceReferences))
	for _, ref := range igc.InstanceReferences {
		instanceReferences = append(instanceReferences, mqlInstanceReference{
			InstanceId:     ref.InstanceId,
			InstanceName:   ref.InstanceName,
			PublicEciesKey: ref.PublicEciesKey,
			PublicKey:      ref.PublicKey,
		})
	}
	mqlInstanceRefs, err := core.JsonToDictSlice(instanceReferences)
	if err != nil {
		return nil, err
	}

	type mqlManagedGroupConfig struct {
		InstanceGroupManagerName string `json:"instanceGroupManagerName"`
		InstanceTemplateName     string `json:"instanceTemplateName"`
	}
	var mqlManagerGroupCfg map[string]interface{}
	if igc.ManagedGroupConfig != nil {
		mqlManagerGroupCfg, err = core.JsonToDict(mqlManagedGroupConfig{
			InstanceGroupManagerName: igc.ManagedGroupConfig.InstanceGroupManagerName,
			InstanceTemplateName:     igc.ManagedGroupConfig.InstanceTemplateName,
		})
		if err != nil {
			return nil, err
		}
	}

	return runtime.CreateResource("gcp.project.dataprocService.cluster.config.instance",
		"id", id,
		"accelerators", mqlAccs,
		"diskConfig", mqlDiskCfg,
		"imageUri", igc.ImageUri,
		"instanceNames", core.StrSliceToInterface(igc.InstanceNames),
		"instanceReferences", mqlInstanceRefs,
		"isPreemptible", igc.IsPreemptible,
		"machineTypeUri", igc.MachineTypeUri,
		"managedGroupConfig", mqlManagerGroupCfg,
		"minCpuPlatform", igc.MinCpuPlatform,
		"numInstances", igc.NumInstances,
		"preemptibility", igc.Preemptibility,
	)
}
