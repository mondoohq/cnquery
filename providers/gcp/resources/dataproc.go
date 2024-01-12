// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/gcp/connection"
	"go.mondoo.com/cnquery/v10/types"
	"google.golang.org/api/compute/v1"
	dataproc "google.golang.org/api/dataproc/v1"
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
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	return fmt.Sprintf("%s/gcp.project.dataprocService", projectId), nil
}

func (g *mqlGcpProject) dataproc() (*mqlGcpProjectDataprocService, error) {
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	projectId := g.Id.Data

	ctx := context.Background()
	client, err := conn.Client(dataproc.CloudPlatformScope)
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
	res, err := CreateResource(g.MqlRuntime, "gcp.project.dataprocService", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
		"enabled":   llx.BoolData(enabled),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectDataprocService), nil
}

func (g *mqlGcpProjectDataprocService) regions() ([]interface{}, error) {
	// no check whether DataProc service is enabled here, this uses a different service
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	client, err := conn.Client(dataproc.CloudPlatformScope)
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

func (g *mqlGcpProjectDataprocService) clusters() ([]interface{}, error) {
	if g.Enabled.Error != nil {
		return nil, g.Enabled.Error
	}
	enabled := g.Enabled.Data
	if !enabled {
		log.Warn().Msg("DataProc Cloud API is not enabled, not querying clusters")
		return []interface{}{}, nil
	}
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	regions := g.GetRegions()
	if regions.Error != nil {
		return nil, regions.Error
	}

	client, err := conn.Client(dataproc.CloudPlatformScope)
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
	wg.Add(len(regions.Data))
	mux := &sync.Mutex{}
	for _, region := range regions.Data {
		go func(projectId, regionName string) {
			defer wg.Done()
			clusters, err := dataprocSvc.Projects.Regions.Clusters.List(projectId, regionName).Do()
			if err != nil {
				log.Error().Str("region", regionName).Err(err).Send()
			} else {
				for _, c := range clusters.Clusters {
					var mqlConfig plugin.Resource
					if c.Config != nil {
						var mqlAutoscalingCfg map[string]interface{}
						if c.Config.AutoscalingConfig != nil {
							type mqlAutoscalingConfig struct {
								PolicyUri string `json:"policyUri"`
							}
							mqlAutoscalingCfg, err = convert.JsonToDict(mqlAutoscalingConfig{PolicyUri: c.Config.AutoscalingConfig.PolicyUri})
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

							mqlMetricsCfg, err = convert.JsonToDict(mqlMetricsConfig{Metrics: mqlMetricsConfigs})
							if err != nil {
								log.Error().Err(err).Send()
							}
						}

						var mqlEncryptionCfg map[string]interface{}
						if c.Config.EncryptionConfig != nil {
							type mqlEncryptionConfig struct {
								GcePdKmsKeyName string `json:"gcePdKmsKeyName"`
							}
							mqlEncryptionCfg, err = convert.JsonToDict(mqlEncryptionConfig{GcePdKmsKeyName: c.Config.EncryptionConfig.GcePdKmsKeyName})
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
							mqlEndpointCfg, err = convert.JsonToDict(mqlEndpointConfig{
								HttpPorts:            c.Config.EndpointConfig.HttpPorts,
								EnableHttpPortAccess: c.Config.EndpointConfig.EnableHttpPortAccess,
							})
							if err != nil {
								log.Error().Err(err).Send()
							}
						}

						var mqlGceClusterCfg plugin.Resource
						if c.Config.GceClusterConfig != nil {
							var mqlConfidentialInstanceCfg map[string]interface{}
							if c.Config.GceClusterConfig.ConfidentialInstanceConfig != nil {
								type mqlConfidentialInstanceConfig struct {
									EnableConfidentialCompute bool `json:"enableConfidentialCompute"`
								}
								mqlConfidentialInstanceCfg, err = convert.JsonToDict(mqlConfidentialInstanceConfig{
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
								mqlNodeGroupAffinityCfg, err = convert.JsonToDict(mqlNodeGroupAffinityConfig{
									Uri: c.Config.GceClusterConfig.NodeGroupAffinity.NodeGroupUri,
								})
								if err != nil {
									log.Error().Err(err).Send()
								}
							}

							var mqlReservationAffinity plugin.Resource
							if c.Config.GceClusterConfig.ReservationAffinity != nil {
								mqlReservationAffinity, err = CreateResource(g.MqlRuntime, "gcp.project.dataprocService.cluster.config.gceCluster.reservationAffinity", map[string]*llx.RawData{
									"id":                     llx.StringData(fmt.Sprintf("%s/dataproc/%s/config/gceCluster/reservationAffinity", projectId, c.ClusterName)),
									"consumeReservationType": llx.StringData(c.Config.GceClusterConfig.ReservationAffinity.ConsumeReservationType),
									"key":                    llx.StringData(c.Config.GceClusterConfig.ReservationAffinity.Key),
									"values":                 llx.ArrayData(convert.SliceAnyToInterface(c.Config.GceClusterConfig.ReservationAffinity.Values), types.String),
								})
								if err != nil {
									log.Error().Err(err).Send()
								}
							}

							var mqlShieldedCfg plugin.Resource
							if c.Config.GceClusterConfig.ShieldedInstanceConfig != nil {
								mqlShieldedCfg, err = CreateResource(g.MqlRuntime, "gcp.project.dataprocService.cluster.config.gceCluster.shieldedInstanceConfig", map[string]*llx.RawData{
									"id":                        llx.StringData(fmt.Sprintf("%s/dataproc/%s/config/gceCluster/shieldedInstanceConfig", projectId, c.ClusterName)),
									"enableIntegrityMonitoring": llx.BoolData(c.Config.GceClusterConfig.ShieldedInstanceConfig.EnableIntegrityMonitoring),
									"enableSecureBoot":          llx.BoolData(c.Config.GceClusterConfig.ShieldedInstanceConfig.EnableSecureBoot),
									"enableVtpm":                llx.BoolData(c.Config.GceClusterConfig.ShieldedInstanceConfig.EnableVtpm),
								})
								if err != nil {
									log.Error().Err(err).Send()
								}
							}

							mqlGceClusterCfg, err = CreateResource(g.MqlRuntime, "gcp.project.dataprocService.cluster.config.gceCluster", map[string]*llx.RawData{
								"id":                      llx.StringData(fmt.Sprintf("%s/dataproc/%s/config/gceCluster", projectId, c.ClusterName)),
								"projectId":               llx.StringData(projectId),
								"confidentialInstance":    llx.DictData(mqlConfidentialInstanceCfg),
								"internalIpOnly":          llx.BoolData(c.Config.GceClusterConfig.InternalIpOnly),
								"metadata":                llx.MapData(convert.MapToInterfaceMap(c.Config.GceClusterConfig.Metadata), types.String),
								"networkUri":              llx.StringData(c.Config.GceClusterConfig.NetworkUri),
								"nodeGroupAffinity":       llx.DictData(mqlNodeGroupAffinityCfg),
								"privateIpv6GoogleAccess": llx.StringData(c.Config.GceClusterConfig.PrivateIpv6GoogleAccess),
								"reservationAffinity":     llx.ResourceData(mqlReservationAffinity, "gcp.project.dataprocService.cluster.config.gceCluster.reservationAffinity"),
								"serviceAccountEmail":     llx.StringData(c.Config.GceClusterConfig.ServiceAccount),
								"serviceAccountScopes":    llx.ArrayData(convert.SliceAnyToInterface(c.Config.GceClusterConfig.ServiceAccountScopes), types.String),
								"shieldedInstanceConfig":  llx.ResourceData(mqlShieldedCfg, "gcp.project.dataprocService.cluster.config.gceCluster.shieldedInstanceConfig"),
								"subnetworkUri":           llx.StringData(c.Config.GceClusterConfig.SubnetworkUri),
								"tags":                    llx.ArrayData(convert.SliceAnyToInterface(c.Config.GceClusterConfig.Tags), types.String),
								"zoneUri":                 llx.StringData(c.Config.GceClusterConfig.ZoneUri),
							})
							if err != nil {
								log.Error().Err(err).Send()
							}
						}

						var mqlGkeClusterCfg plugin.Resource
						if c.Config.GkeClusterConfig != nil {
							mqlNodePools := make([]mqlGkeNodePoolTarget, 0, len(c.Config.GkeClusterConfig.NodePoolTarget))
							for _, npt := range c.Config.GkeClusterConfig.NodePoolTarget {
								mqlNodePools = append(mqlNodePools, nodePoolTargetToMql(npt))
							}
							nodePoolsDict, err := convert.JsonToDictSlice(mqlNodePools)
							if err != nil {
								log.Error().Err(err).Send()
							}
							mqlGkeClusterCfg, err = CreateResource(g.MqlRuntime, "gcp.project.dataprocService.cluster.config.gkeCluster", map[string]*llx.RawData{
								"id":               llx.StringData(fmt.Sprintf("%s/dataproc/%s/config/gkeCluster", projectId, c.ClusterName)),
								"gkeClusterTarget": llx.StringData(c.Config.GkeClusterConfig.GkeClusterTarget),
								"nodePoolTarget":   llx.ArrayData(nodePoolsDict, types.Dict),
							})
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

						dictInitActions, err := convert.JsonToDictSlice(initActions)
						if err != nil {
							log.Error().Err(err).Send()
						}

						var mqlLifecycleCfg plugin.Resource
						if c.Config.LifecycleConfig != nil {
							mqlLifecycleCfg, err = CreateResource(g.MqlRuntime, "gcp.project.dataprocService.cluster.config.lifecycle", map[string]*llx.RawData{
								"id":             llx.StringData(fmt.Sprintf("%s/dataproc/%s/config/lifecycle", projectId, c.ClusterName)),
								"autoDeleteTime": llx.StringData(c.Config.LifecycleConfig.AutoDeleteTime),
								"autoDeleteTtl":  llx.StringData(c.Config.LifecycleConfig.AutoDeleteTtl),
								"idleDeleteTtl":  llx.StringData(c.Config.LifecycleConfig.IdleDeleteTtl),
								"idleStartTime":  llx.StringData(c.Config.LifecycleConfig.IdleStartTime),
							})
							if err != nil {
								log.Error().Err(err).Send()
							}
						}

						mqlMasterCfg, err := instanceGroupConfigToMql(
							g.MqlRuntime, c.Config.MasterConfig, fmt.Sprintf("%s/dataproc/%s/config/master", projectId, c.ClusterName))
						if err != nil {
							log.Error().Err(err).Send()
						}

						type mqlMetastoreConfig struct {
							DataprocMetastoreService string `json:"dataprocMetastoreService"`
						}
						var mqlMetastoreCfg map[string]interface{}
						if c.Config.MetastoreConfig != nil {
							mqlMetastoreCfg, err = convert.JsonToDict(mqlMetastoreConfig{
								DataprocMetastoreService: c.Config.MetastoreConfig.DataprocMetastoreService,
							})
							if err != nil {
								log.Error().Err(err).Send()
							}
						}

						mqlSecondaryWorkerCfg, err := instanceGroupConfigToMql(
							g.MqlRuntime, c.Config.SecondaryWorkerConfig, fmt.Sprintf("%s/dataproc/%s/config/secondaryWorker", projectId, c.ClusterName))
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
								mqlSecurityCfg, err = convert.JsonToDict(cfg)
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
							mqlSoftwareCfg, err = convert.JsonToDict(mqlSoftwareConfig{
								ImageVersion:       c.Config.SoftwareConfig.ImageVersion,
								OptionalComponents: c.Config.SoftwareConfig.OptionalComponents,
							})
							if err != nil {
								log.Error().Err(err).Send()
							}
						}

						mqlWorkerCfg, err := instanceGroupConfigToMql(
							g.MqlRuntime, c.Config.WorkerConfig, fmt.Sprintf("%s/dataproc/%s/config/worker", projectId, c.ClusterName))
						if err != nil {
							log.Error().Err(err).Send()
						}

						mqlConfig, err = CreateResource(g.MqlRuntime, "gcp.project.dataprocService.cluster.config", map[string]*llx.RawData{
							"projectId":             llx.StringData(c.ProjectId),
							"parentResourcePath":    llx.StringData(fmt.Sprintf("%s/dataproc/%s", projectId, c.ClusterName)),
							"autoscaling":           llx.DictData(mqlAutoscalingCfg),
							"configBucket":          llx.StringData(c.Config.ConfigBucket),
							"metrics":               llx.DictData(mqlMetricsCfg),
							"encryption":            llx.DictData(mqlEncryptionCfg),
							"endpoint":              llx.DictData(mqlEndpointCfg),
							"gceCluster":            llx.ResourceData(mqlGceClusterCfg, "gcp.project.dataprocService.cluster.config.gceCluster"),
							"gkeCluster":            llx.ResourceData(mqlGkeClusterCfg, "gcp.project.dataprocService.cluster.config.gkeCluster"),
							"initializationActions": llx.ArrayData(dictInitActions, types.Dict),
							"lifecycle":             llx.ResourceData(mqlLifecycleCfg, "gcp.project.dataprocService.cluster.config.lifecycle"),
							"master":                llx.ResourceData(mqlMasterCfg, "gcp.project.dataprocService.cluster.config.instance"),
							"metastore":             llx.DictData(mqlMetastoreCfg),
							"secondaryWorker":       llx.ResourceData(mqlSecondaryWorkerCfg, "gcp.project.dataprocService.cluster.config.instance"),
							"security":              llx.DictData(mqlSecurityCfg),
							"software":              llx.DictData(mqlSoftwareCfg),
							"tempBucket":            llx.StringData(c.Config.TempBucket),
							"worker":                llx.ResourceData(mqlWorkerCfg, "gcp.project.dataprocService.cluster.config.instance"),
						})
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
						mqlMetrics, err = convert.JsonToDict(mqlClusterMetrics{HdfsMetrics: c.Metrics.HdfsMetrics, YarnMetrics: c.Metrics.YarnMetrics})
						if err != nil {
							log.Error().Err(err).Send()
						}
					}

					var mqlStatus plugin.Resource
					if c.Status != nil {
						mqlStatus, err = CreateResource(g.MqlRuntime, "gcp.project.dataprocService.cluster.status", map[string]*llx.RawData{
							"id":       llx.StringData(fmt.Sprintf("%s/dataproc/%s/status", projectId, c.ClusterName)),
							"detail":   llx.StringData(c.Status.Detail),
							"state":    llx.StringData(c.Status.State),
							"started":  llx.TimeDataPtr(parseTime(c.Status.StateStartTime)),
							"substate": llx.StringData(c.Status.Substate),
						})
						if err != nil {
							log.Error().Err(err).Send()
						}
					}

					mqlStatusHistory := make([]interface{}, 0, len(c.StatusHistory))
					for i, s := range c.StatusHistory {
						mqlStatus, err = CreateResource(g.MqlRuntime, "gcp.project.dataprocService.cluster.status", map[string]*llx.RawData{
							"id":       llx.StringData(fmt.Sprintf("%s/dataproc/%s/status/%d", projectId, c.ClusterName, i)),
							"detail":   llx.StringData(s.Detail),
							"state":    llx.StringData(s.State),
							"started":  llx.TimeDataPtr(parseTime(s.StateStartTime)),
							"substate": llx.StringData(s.Substate),
						})
						if err != nil {
							log.Error().Err(err).Send()
						}
						mqlStatusHistory = append(mqlStatusHistory, mqlStatus)
					}

					var mqlVirtualClusterCfg plugin.Resource
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

						mqlAuxServices, err := convert.JsonToDict(mqlAuxiliaryServices{
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

						mqlK8sClusterCfg, err := convert.JsonToDict(mqlKubernetesClusterConfig{
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

						mqlVirtualClusterCfg, err = CreateResource(g.MqlRuntime, "gcp.project.dataprocService.cluster.virtualClusterConfig", map[string]*llx.RawData{
							"parentResourcePath": llx.StringData(fmt.Sprintf("%s/dataproc/%s", projectId, c.ClusterName)),
							"auxiliaryServices":  llx.DictData(mqlAuxServices),
							"kubernetesCluster":  llx.DictData(mqlK8sClusterCfg),
							"stagingBucket":      llx.StringData(c.VirtualClusterConfig.StagingBucket),
						})
						if err != nil {
							log.Error().Err(err).Send()
						}
					}

					mqlCluster, err := CreateResource(g.MqlRuntime, "gcp.project.dataprocService.cluster", map[string]*llx.RawData{
						"projectId":            llx.StringData(projectId),
						"name":                 llx.StringData(c.ClusterName),
						"uuid":                 llx.StringData(c.ClusterUuid),
						"config":               llx.ResourceData(mqlConfig, "gcp.project.dataprocService.cluster.config"),
						"labels":               llx.MapData(convert.MapToInterfaceMap(c.Labels), types.String),
						"metrics":              llx.DictData(mqlMetrics),
						"status":               llx.ResourceData(mqlStatus, "gcp.project.dataprocService.cluster.status"),
						"statusHistory":        llx.ArrayData(mqlStatusHistory, types.Resource("gcp.project.dataprocService.cluster.status")),
						"virtualClusterConfig": llx.ResourceData(mqlVirtualClusterCfg, "gcp.project.dataprocService.cluster.virtualClusterConfig"),
					})
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

func (g *mqlGcpProjectDataprocServiceClusterConfigGceCluster) serviceAccount() (*mqlGcpProjectIamServiceServiceAccount, error) {
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

func (g *mqlGcpProjectDataprocServiceCluster) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	projectId := g.ProjectId.Data
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	name := g.Name.Data
	return fmt.Sprintf("%s/dataproc/%s", projectId, name), nil
}

func (g *mqlGcpProjectDataprocServiceClusterConfig) id() (string, error) {
	if g.ParentResourcePath.Error != nil {
		return "", g.ParentResourcePath.Error
	}
	parentResource := g.ParentResourcePath.Data
	return fmt.Sprintf("%s/config", parentResource), nil
}

func (g *mqlGcpProjectDataprocServiceClusterStatus) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectDataprocServiceClusterVirtualClusterConfig) id() (string, error) {
	if g.ParentResourcePath.Error != nil {
		return "", g.ParentResourcePath.Error
	}
	parentResource := g.ParentResourcePath.Data
	return fmt.Sprintf("%s/virtualClusterConfig", parentResource), nil
}

func (g *mqlGcpProjectDataprocServiceClusterConfigGceCluster) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectDataprocServiceClusterConfigGceClusterReservationAffinity) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectDataprocServiceClusterConfigGceClusterShieldedInstanceConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectDataprocServiceClusterConfigGkeCluster) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectDataprocServiceClusterConfigLifecycle) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectDataprocServiceClusterConfigInstance) id() (string, error) {
	return g.Id.Data, g.Id.Error
}

func (g *mqlGcpProjectDataprocServiceClusterConfigInstanceDiskConfig) id() (string, error) {
	return g.Id.Data, g.Id.Error
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

func instanceGroupConfigToMql(runtime *plugin.Runtime, igc *dataproc.InstanceGroupConfig, id string) (plugin.Resource, error) {
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
	mqlAccs, err := convert.JsonToDictSlice(accs)
	if err != nil {
		return nil, err
	}

	var mqlDiskCfg plugin.Resource
	if igc.DiskConfig != nil {
		mqlDiskCfg, err = CreateResource(runtime, "gcp.project.dataprocService.cluster.config.instance.diskConfig", map[string]*llx.RawData{
			"id":                llx.StringData(fmt.Sprintf("%s/diskConfig", id)),
			"bootDiskSizeGb":    llx.IntData(igc.DiskConfig.BootDiskSizeGb),
			"bootDiskType":      llx.StringData(igc.DiskConfig.BootDiskType),
			"localSsdInterface": llx.StringData(igc.DiskConfig.LocalSsdInterface),
			"numLocalSsds":      llx.IntData(igc.DiskConfig.NumLocalSsds),
		})
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
	mqlInstanceRefs, err := convert.JsonToDictSlice(instanceReferences)
	if err != nil {
		return nil, err
	}

	type mqlManagedGroupConfig struct {
		InstanceGroupManagerName string `json:"instanceGroupManagerName"`
		InstanceTemplateName     string `json:"instanceTemplateName"`
	}
	var mqlManagerGroupCfg map[string]interface{}
	if igc.ManagedGroupConfig != nil {
		mqlManagerGroupCfg, err = convert.JsonToDict(mqlManagedGroupConfig{
			InstanceGroupManagerName: igc.ManagedGroupConfig.InstanceGroupManagerName,
			InstanceTemplateName:     igc.ManagedGroupConfig.InstanceTemplateName,
		})
		if err != nil {
			return nil, err
		}
	}

	return CreateResource(runtime, "gcp.project.dataprocService.cluster.config.instance", map[string]*llx.RawData{
		"id":                 llx.StringData(id),
		"accelerators":       llx.ArrayData(mqlAccs, types.Dict),
		"diskConfig":         llx.ResourceData(mqlDiskCfg, "gcp.project.dataprocService.cluster.config.instance.diskConfig"),
		"imageUri":           llx.StringData(igc.ImageUri),
		"instanceNames":      llx.ArrayData(convert.SliceAnyToInterface(igc.InstanceNames), types.String),
		"instanceReferences": llx.ArrayData(mqlInstanceRefs, types.Dict),
		"isPreemptible":      llx.BoolData(igc.IsPreemptible),
		"machineTypeUri":     llx.StringData(igc.MachineTypeUri),
		"managedGroupConfig": llx.DictData(mqlManagerGroupCfg),
		"minCpuPlatform":     llx.StringData(igc.MinCpuPlatform),
		"numInstances":       llx.IntData(igc.NumInstances),
		"preemptibility":     llx.StringData(igc.Preemptibility),
	})
}
