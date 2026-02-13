// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	redis "cloud.google.com/go/redis/apiv1"
	"cloud.google.com/go/redis/apiv1/redispb"
	rediscluster "cloud.google.com/go/redis/cluster/apiv1"
	"cloud.google.com/go/redis/cluster/apiv1/clusterpb"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers/gcp/connection"
	"go.mondoo.com/cnquery/v12/types"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProject) redis() (*mqlGcpProjectRedisService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	res, err := CreateResource(g.MqlRuntime, "gcp.project.redisService", map[string]*llx.RawData{
		"projectId": llx.StringData(g.Id.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectRedisService), nil
}

func initGcpProjectRedisServiceInstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if args == nil {
			args = make(map[string]*llx.RawData)
		}
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["projectId"] = llx.StringData(ids.project)
		} else {
			return nil, nil, errors.New("no asset identifier found")
		}
	}

	obj, err := CreateResource(runtime, "gcp.project.redisService", map[string]*llx.RawData{
		"projectId": args["projectId"],
	})
	if err != nil {
		return nil, nil, err
	}
	redisSvc := obj.(*mqlGcpProjectRedisService)
	instances := redisSvc.GetInstances()
	if instances.Error != nil {
		return nil, nil, instances.Error
	}

	for _, inst := range instances.Data {
		instance := inst.(*mqlGcpProjectRedisServiceInstance)
		// Redis instance name is a full resource path:
		// projects/{project}/locations/{location}/instances/{instance_id}
		nameParts := strings.Split(instance.Name.Data, "/")
		instanceName := nameParts[len(nameParts)-1]

		if instanceName == args["name"].Value.(string) {
			return args, instance, nil
		}
	}

	return nil, nil, errors.New("Redis instance not found")
}

func (g *mqlGcpProjectRedisService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("gcp.project/%s/redisService", g.ProjectId.Data), nil
}

func (g *mqlGcpProjectRedisServiceInstance) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf(
		"gcp.project/%s/redisService.instance/%s", g.ProjectId.Data, g.Name.Data,
	), nil
}

// Requires the following OAuth scope:
//
// https://www.googleapis.com/auth/cloud-platform
//
// Docs https://cloud.google.com/memorystore/docs/redis/reference/rest/v1/projects.locations/list#authorization-scopes
func (g *mqlGcpProjectRedisService) instances() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(redis.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	redisSvc, err := redis.NewCloudRedisClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer redisSvc.Close()

	it := redisSvc.ListInstances(ctx, &redispb.ListInstancesRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", projectId),
	})
	res := []any{}
	for {
		instance, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		persistenceConfig, err := redisConvertPersistenceConfig(instance.PersistenceConfig)
		if err != nil {
			return nil, err
		}
		maintenancePolicy, err := redisConvertMaintenancePolicy(instance.MaintenancePolicy)
		if err != nil {
			return nil, err
		}
		maintenanceSchedule, err := redisConvertMaintenanceSchedule(instance.MaintenanceSchedule)
		if err != nil {
			return nil, err
		}

		mqlRedisInstance, err := CreateResource(g.MqlRuntime, "gcp.project.redisService.instance", map[string]*llx.RawData{
			"projectId":              llx.StringData(projectId),
			"name":                   llx.StringData(instance.Name),
			"state":                  llx.StringData(instance.State.String()),
			"displayName":            llx.StringData(instance.DisplayName),
			"locationId":             llx.StringData(instance.LocationId),
			"redisVersion":           llx.StringData(instance.RedisVersion),
			"reservedIpRange":        llx.StringData(instance.ReservedIpRange),
			"secondaryIpRange":       llx.StringData(instance.SecondaryIpRange),
			"AuthorizedNetwork":      llx.StringData(instance.AuthorizedNetwork),
			"persistenceIamIdentity": llx.StringData(instance.PersistenceIamIdentity),
			"connectMode":            llx.StringData(instance.ConnectMode.String()),
			"readEndpoint":           llx.StringData(instance.ReadEndpoint),
			"customerManagedKey":     llx.StringData(instance.CustomerManagedKey),
			"maintenanceVersion":     llx.StringData(instance.MaintenanceVersion),
			"host":                   llx.StringData(instance.Host),
			"currentLocationId":      llx.StringData(instance.CurrentLocationId),
			"statusMessage":          llx.StringData(instance.StatusMessage),
			"tier":                   llx.StringData(instance.Tier.String()),
			"transitEncryptionMode":  llx.StringData(instance.TransitEncryptionMode.String()),
			"readReplicasMode":       llx.StringData(instance.ReadReplicasMode.String()),
			"alternativeLocationId":  llx.StringData(instance.AlternativeLocationId),
			"port":                   llx.IntData(instance.Port),
			"memorySizeGb":           llx.IntData(instance.MemorySizeGb),
			"replicaCount":           llx.IntData(instance.ReplicaCount),
			"readEndpointPort":       llx.IntData(instance.ReadEndpointPort),
			"authEnabled":            llx.BoolData(instance.AuthEnabled),
			"createTime":             llx.TimeData(instance.CreateTime.AsTime()),
			"labels":                 llx.MapData(convert.MapToInterfaceMap(instance.Labels), types.String),
			"redisConfigs":           llx.MapData(convert.MapToInterfaceMap(instance.RedisConfigs), types.String),
			"persistenceConfig":      llx.DictData(persistenceConfig),
			"maintenancePolicy":      llx.DictData(maintenancePolicy),
			"maintenanceSchedule":    llx.DictData(maintenanceSchedule),
			"suspensionReasons":      llx.ArrayData(redisConvertSuspensionReasons(instance.SuspensionReasons), types.String),
			"availableMaintenanceVersions": llx.ArrayData(
				convert.SliceAnyToInterface(instance.AvailableMaintenanceVersions), types.String,
			),
			"nodes": llx.ArrayData(
				redisInstanceNodesToArrayInterface(g.MqlRuntime, projectId, instance.Nodes),
				types.Resource("gcp.project.redisService.instance.nodeInfo"),
			),
			"serverCaCerts": llx.ArrayData(
				redisConvertServerCaCerts(g.MqlRuntime, projectId, instance.ServerCaCerts),
				types.Resource("gcp.project.redisService.instance.serverCaCert"),
			),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRedisInstance)
	}

	return res, nil
}

func (n *mqlGcpProjectRedisServiceInstanceNodeInfo) id() (string, error) {
	if n.ProjectId.Error != nil {
		return "", n.ProjectId.Error
	}
	return fmt.Sprintf(
		"gcp.project.redisService.instance.nodeInfo/%s/%s", n.ProjectId.Data, n.Id.Data,
	), nil
}

func redisInstanceNodesToArrayInterface(runtime *plugin.Runtime, projectId string, nodes []*redispb.NodeInfo) (list []any) {
	for _, node := range nodes {
		if node == nil {
			continue
		}

		r, err := CreateResource(runtime, "gcp.project.redisService.instance.nodeInfo", map[string]*llx.RawData{
			"projectId": llx.StringData(projectId),
			"id":        llx.StringData(node.Id),
			"zone":      llx.StringData(node.Zone),
		})
		if err != nil {
			continue
		}
		list = append(list, r)
	}

	return
}

func (c *mqlGcpProjectRedisServiceInstanceServerCaCert) id() (string, error) {
	if c.ProjectId.Error != nil {
		return "", c.ProjectId.Error
	}
	return fmt.Sprintf(
		"gcp.project.redisService.instance.serverCaCert/%s/%s", c.ProjectId.Data, c.SerialNumber.Data,
	), nil
}

func redisConvertServerCaCerts(runtime *plugin.Runtime, projectId string, certs []*redispb.TlsCertificate) (list []any) {
	for _, cert := range certs {
		if cert == nil {
			continue
		}

		var createTime, expireTime time.Time
		if cert.CreateTime != nil {
			createTime = cert.CreateTime.AsTime()
		}
		if cert.ExpireTime != nil {
			expireTime = cert.ExpireTime.AsTime()
		}

		r, err := CreateResource(runtime, "gcp.project.redisService.instance.serverCaCert", map[string]*llx.RawData{
			"projectId":       llx.StringData(projectId),
			"serialNumber":    llx.StringData(cert.SerialNumber),
			"cert":            llx.StringData(cert.Cert),
			"createTime":      llx.TimeData(createTime),
			"expireTime":      llx.TimeData(expireTime),
			"sha1Fingerprint": llx.StringData(cert.Sha1Fingerprint),
		})
		if err != nil {
			continue
		}
		list = append(list, r)
	}

	return
}

func redisConvertSuspensionReasons(reasons []redispb.Instance_SuspensionReason) []any {
	res := make([]any, 0, len(reasons))
	for _, r := range reasons {
		res = append(res, r.String())
	}
	return res
}

type mqlRedisPersistenceConfig struct {
	PersistenceMode      string  `json:"persistenceMode"`
	RdbSnapshotPeriod    string  `json:"rdbSnapshotPeriod"`
	RdbNextSnapshotTime  *string `json:"rdbNextSnapshotTime"`
	RdbSnapshotStartTime *string `json:"rdbSnapshotStartTime"`
}

func redisConvertPersistenceConfig(pc *redispb.PersistenceConfig) (map[string]any, error) {
	if pc == nil {
		return nil, nil
	}

	cfg := mqlRedisPersistenceConfig{
		PersistenceMode:   pc.PersistenceMode.String(),
		RdbSnapshotPeriod: pc.RdbSnapshotPeriod.String(),
	}
	if pc.RdbNextSnapshotTime != nil {
		s := pc.RdbNextSnapshotTime.AsTime().Format(time.RFC3339)
		cfg.RdbNextSnapshotTime = &s
	}
	if pc.RdbSnapshotStartTime != nil {
		s := pc.RdbSnapshotStartTime.AsTime().Format(time.RFC3339)
		cfg.RdbSnapshotStartTime = &s
	}

	return convert.JsonToDict(cfg)
}

type mqlRedisMaintenanceWindow struct {
	Day       string `json:"day"`
	StartTime string `json:"startTime"`
	Duration  string `json:"duration"`
}

type mqlRedisMaintenancePolicy struct {
	CreateTime              *string                     `json:"createTime"`
	UpdateTime              *string                     `json:"updateTime"`
	Description             string                      `json:"description"`
	WeeklyMaintenanceWindow []mqlRedisMaintenanceWindow `json:"weeklyMaintenanceWindow"`
}

func redisConvertMaintenancePolicy(mp *redispb.MaintenancePolicy) (map[string]any, error) {
	if mp == nil {
		return nil, nil
	}

	policy := mqlRedisMaintenancePolicy{
		Description: mp.Description,
	}
	if mp.CreateTime != nil {
		s := mp.CreateTime.AsTime().Format(time.RFC3339)
		policy.CreateTime = &s
	}
	if mp.UpdateTime != nil {
		s := mp.UpdateTime.AsTime().Format(time.RFC3339)
		policy.UpdateTime = &s
	}

	for _, w := range mp.WeeklyMaintenanceWindow {
		if w == nil {
			continue
		}
		window := mqlRedisMaintenanceWindow{
			Day: w.Day.String(),
		}
		if w.StartTime != nil {
			window.StartTime = fmt.Sprintf("%02d:%02d:%02d", w.StartTime.Hours, w.StartTime.Minutes, w.StartTime.Seconds)
		}
		if w.Duration != nil {
			window.Duration = w.Duration.AsDuration().String()
		}
		policy.WeeklyMaintenanceWindow = append(policy.WeeklyMaintenanceWindow, window)
	}

	return convert.JsonToDict(policy)
}

type mqlRedisMaintenanceSchedule struct {
	StartTime            *string `json:"startTime"`
	EndTime              *string `json:"endTime"`
	ScheduleDeadlineTime *string `json:"scheduleDeadlineTime"`
}

func redisConvertMaintenanceSchedule(ms *redispb.MaintenanceSchedule) (map[string]any, error) {
	if ms == nil {
		return nil, nil
	}

	schedule := mqlRedisMaintenanceSchedule{}
	if ms.StartTime != nil {
		s := ms.StartTime.AsTime().Format(time.RFC3339)
		schedule.StartTime = &s
	}
	if ms.EndTime != nil {
		s := ms.EndTime.AsTime().Format(time.RFC3339)
		schedule.EndTime = &s
	}
	if ms.ScheduleDeadlineTime != nil {
		s := ms.ScheduleDeadlineTime.AsTime().Format(time.RFC3339)
		schedule.ScheduleDeadlineTime = &s
	}

	return convert.JsonToDict(schedule)
}

// ===== Redis Cluster resources =====

func (g *mqlGcpProjectRedisServiceCluster) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf(
		"gcp.project/%s/redisService.cluster/%s", g.ProjectId.Data, g.Name.Data,
	), nil
}

func initGcpProjectRedisServiceCluster(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if args == nil {
			args = make(map[string]*llx.RawData)
		}
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["projectId"] = llx.StringData(ids.project)
		} else {
			return nil, nil, errors.New("no asset identifier found")
		}
	}

	obj, err := CreateResource(runtime, "gcp.project.redisService", map[string]*llx.RawData{
		"projectId": args["projectId"],
	})
	if err != nil {
		return nil, nil, err
	}
	redisSvc := obj.(*mqlGcpProjectRedisService)
	clusters := redisSvc.GetClusters()
	if clusters.Error != nil {
		return nil, nil, clusters.Error
	}

	for _, c := range clusters.Data {
		cluster := c.(*mqlGcpProjectRedisServiceCluster)
		nameParts := strings.Split(cluster.Name.Data, "/")
		clusterName := nameParts[len(nameParts)-1]

		if clusterName == args["name"].Value.(string) {
			return args, cluster, nil
		}
	}

	return nil, nil, errors.New("Redis cluster not found")
}

func (g *mqlGcpProjectRedisService) clusters() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)

	creds, err := conn.Credentials(rediscluster.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	clusterSvc, err := rediscluster.NewCloudRedisClusterClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer clusterSvc.Close()

	it := clusterSvc.ListClusters(ctx, &clusterpb.ListClustersRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", projectId),
	})
	res := []any{}
	for {
		cluster, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		persistenceConfig, err := clusterConvertPersistenceConfig(cluster.PersistenceConfig)
		if err != nil {
			return nil, err
		}
		zoneDistConfig, err := clusterConvertZoneDistributionConfig(cluster.ZoneDistributionConfig)
		if err != nil {
			return nil, err
		}
		maintenancePolicy, err := clusterConvertMaintenancePolicy(cluster.MaintenancePolicy)
		if err != nil {
			return nil, err
		}
		maintenanceSchedule, err := clusterConvertMaintenanceSchedule(cluster.MaintenanceSchedule)
		if err != nil {
			return nil, err
		}
		encryptionInfo, err := clusterConvertEncryptionInfo(cluster.EncryptionInfo)
		if err != nil {
			return nil, err
		}
		automatedBackupConfig, err := clusterConvertAutomatedBackupConfig(cluster.AutomatedBackupConfig)
		if err != nil {
			return nil, err
		}
		crossClusterConfig, err := clusterConvertCrossClusterReplicationConfig(cluster.CrossClusterReplicationConfig)
		if err != nil {
			return nil, err
		}

		var replicaCount int64
		if cluster.ReplicaCount != nil {
			replicaCount = int64(*cluster.ReplicaCount)
		}
		var shardCount int64
		if cluster.ShardCount != nil {
			shardCount = int64(*cluster.ShardCount)
		}
		var sizeGb int64
		if cluster.SizeGb != nil {
			sizeGb = int64(*cluster.SizeGb)
		}
		var preciseSizeGb float64
		if cluster.PreciseSizeGb != nil {
			preciseSizeGb = *cluster.PreciseSizeGb
		}
		var deletionProtection bool
		if cluster.DeletionProtectionEnabled != nil {
			deletionProtection = *cluster.DeletionProtectionEnabled
		}
		var kmsKey string
		if cluster.KmsKey != nil {
			kmsKey = *cluster.KmsKey
		}
		var backupCollection string
		if cluster.BackupCollection != nil {
			backupCollection = *cluster.BackupCollection
		}

		mqlCluster, err := CreateResource(g.MqlRuntime, "gcp.project.redisService.cluster", map[string]*llx.RawData{
			"projectId":                     llx.StringData(projectId),
			"name":                          llx.StringData(cluster.Name),
			"uid":                           llx.StringData(cluster.Uid),
			"state":                         llx.StringData(cluster.State.String()),
			"createTime":                    llx.TimeData(cluster.CreateTime.AsTime()),
			"authorizationMode":             llx.StringData(cluster.AuthorizationMode.String()),
			"transitEncryptionMode":         llx.StringData(cluster.TransitEncryptionMode.String()),
			"nodeType":                      llx.StringData(cluster.NodeType.String()),
			"shardCount":                    llx.IntData(shardCount),
			"replicaCount":                  llx.IntData(replicaCount),
			"sizeGb":                        llx.IntData(sizeGb),
			"preciseSizeGb":                 llx.FloatData(preciseSizeGb),
			"deletionProtectionEnabled":     llx.BoolData(deletionProtection),
			"kmsKey":                        llx.StringData(kmsKey),
			"backupCollection":              llx.StringData(backupCollection),
			"redisConfigs":                  llx.MapData(convert.MapToInterfaceMap(cluster.RedisConfigs), types.String),
			"persistenceConfig":             llx.DictData(persistenceConfig),
			"zoneDistributionConfig":        llx.DictData(zoneDistConfig),
			"maintenancePolicy":             llx.DictData(maintenancePolicy),
			"maintenanceSchedule":           llx.DictData(maintenanceSchedule),
			"encryptionInfo":                llx.DictData(encryptionInfo),
			"automatedBackupConfig":         llx.DictData(automatedBackupConfig),
			"crossClusterReplicationConfig": llx.DictData(crossClusterConfig),
			"pscConfigs": llx.ArrayData(
				clusterConvertPscConfigs(g.MqlRuntime, projectId, cluster.Name, cluster.PscConfigs),
				types.Resource("gcp.project.redisService.cluster.pscConfig"),
			),
			"discoveryEndpoints": llx.ArrayData(
				clusterConvertDiscoveryEndpoints(g.MqlRuntime, projectId, cluster.Name, cluster.DiscoveryEndpoints),
				types.Resource("gcp.project.redisService.cluster.discoveryEndpoint"),
			),
			"pscConnections": llx.ArrayData(
				clusterConvertPscConnections(g.MqlRuntime, projectId, cluster.Name, cluster.PscConnections),
				types.Resource("gcp.project.redisService.cluster.pscConnection"),
			),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCluster)
	}

	return res, nil
}

// ===== Cluster sub-resource id() methods =====

func (c *mqlGcpProjectRedisServiceClusterPscConfig) id() (string, error) {
	return fmt.Sprintf(
		"gcp.project.redisService.cluster.pscConfig/%s/%s/%s", c.ProjectId.Data, c.ClusterName.Data, c.Network.Data,
	), nil
}

func (c *mqlGcpProjectRedisServiceClusterDiscoveryEndpoint) id() (string, error) {
	return fmt.Sprintf(
		"gcp.project.redisService.cluster.discoveryEndpoint/%s/%s/%s", c.ProjectId.Data, c.ClusterName.Data, c.Address.Data,
	), nil
}

func (c *mqlGcpProjectRedisServiceClusterPscConnection) id() (string, error) {
	return fmt.Sprintf(
		"gcp.project.redisService.cluster.pscConnection/%s/%s/%s", c.ProjectId.Data, c.ClusterName.Data, c.PscConnectionId.Data,
	), nil
}

// ===== Cluster sub-resource converters =====

func clusterConvertPscConfigs(runtime *plugin.Runtime, projectId, clusterName string, configs []*clusterpb.PscConfig) (list []any) {
	for _, cfg := range configs {
		if cfg == nil {
			continue
		}
		r, err := CreateResource(runtime, "gcp.project.redisService.cluster.pscConfig", map[string]*llx.RawData{
			"projectId":   llx.StringData(projectId),
			"clusterName": llx.StringData(clusterName),
			"network":     llx.StringData(cfg.Network),
		})
		if err != nil {
			continue
		}
		list = append(list, r)
	}
	return
}

func clusterConvertDiscoveryEndpoints(runtime *plugin.Runtime, projectId, clusterName string, endpoints []*clusterpb.DiscoveryEndpoint) (list []any) {
	for _, ep := range endpoints {
		if ep == nil {
			continue
		}
		var network string
		if ep.PscConfig != nil {
			network = ep.PscConfig.Network
		}
		r, err := CreateResource(runtime, "gcp.project.redisService.cluster.discoveryEndpoint", map[string]*llx.RawData{
			"projectId":   llx.StringData(projectId),
			"clusterName": llx.StringData(clusterName),
			"address":     llx.StringData(ep.Address),
			"port":        llx.IntData(int64(ep.Port)),
			"network":     llx.StringData(network),
		})
		if err != nil {
			continue
		}
		list = append(list, r)
	}
	return
}

func clusterConvertPscConnections(runtime *plugin.Runtime, projectId, clusterName string, connections []*clusterpb.PscConnection) (list []any) {
	for _, conn := range connections {
		if conn == nil {
			continue
		}
		r, err := CreateResource(runtime, "gcp.project.redisService.cluster.pscConnection", map[string]*llx.RawData{
			"projectId":           llx.StringData(projectId),
			"clusterName":         llx.StringData(clusterName),
			"pscConnectionId":     llx.StringData(conn.PscConnectionId),
			"address":             llx.StringData(conn.Address),
			"forwardingRule":      llx.StringData(conn.ForwardingRule),
			"connectionProjectId": llx.StringData(conn.ProjectId),
			"network":             llx.StringData(conn.Network),
			"serviceAttachment":   llx.StringData(conn.ServiceAttachment),
			"pscConnectionStatus": llx.StringData(conn.PscConnectionStatus.String()),
			"connectionType":      llx.StringData(conn.ConnectionType.String()),
		})
		if err != nil {
			continue
		}
		list = append(list, r)
	}
	return
}

// ===== Cluster dict converters =====

type mqlClusterPersistenceConfig struct {
	Mode      string `json:"mode"`
	RdbConfig *struct {
		RdbSnapshotPeriod    string  `json:"rdbSnapshotPeriod"`
		RdbSnapshotStartTime *string `json:"rdbSnapshotStartTime"`
	} `json:"rdbConfig"`
	AofConfig *struct {
		AppendFsync string `json:"appendFsync"`
	} `json:"aofConfig"`
}

func clusterConvertPersistenceConfig(pc *clusterpb.ClusterPersistenceConfig) (map[string]any, error) {
	if pc == nil {
		return nil, nil
	}
	cfg := mqlClusterPersistenceConfig{
		Mode: pc.Mode.String(),
	}
	if pc.RdbConfig != nil {
		rdb := &struct {
			RdbSnapshotPeriod    string  `json:"rdbSnapshotPeriod"`
			RdbSnapshotStartTime *string `json:"rdbSnapshotStartTime"`
		}{
			RdbSnapshotPeriod: pc.RdbConfig.RdbSnapshotPeriod.String(),
		}
		if pc.RdbConfig.RdbSnapshotStartTime != nil {
			s := pc.RdbConfig.RdbSnapshotStartTime.AsTime().Format(time.RFC3339)
			rdb.RdbSnapshotStartTime = &s
		}
		cfg.RdbConfig = rdb
	}
	if pc.AofConfig != nil {
		cfg.AofConfig = &struct {
			AppendFsync string `json:"appendFsync"`
		}{
			AppendFsync: pc.AofConfig.AppendFsync.String(),
		}
	}
	return convert.JsonToDict(cfg)
}

type mqlClusterZoneDistConfig struct {
	Mode string `json:"mode"`
	Zone string `json:"zone"`
}

func clusterConvertZoneDistributionConfig(zdc *clusterpb.ZoneDistributionConfig) (map[string]any, error) {
	if zdc == nil {
		return nil, nil
	}
	return convert.JsonToDict(mqlClusterZoneDistConfig{
		Mode: zdc.Mode.String(),
		Zone: zdc.Zone,
	})
}

type mqlClusterMaintenanceWindow struct {
	Day       string `json:"day"`
	StartTime string `json:"startTime"`
}

type mqlClusterMaintenancePolicy struct {
	CreateTime              *string                       `json:"createTime"`
	UpdateTime              *string                       `json:"updateTime"`
	WeeklyMaintenanceWindow []mqlClusterMaintenanceWindow `json:"weeklyMaintenanceWindow"`
}

func clusterConvertMaintenancePolicy(mp *clusterpb.ClusterMaintenancePolicy) (map[string]any, error) {
	if mp == nil {
		return nil, nil
	}
	policy := mqlClusterMaintenancePolicy{}
	if mp.CreateTime != nil {
		s := mp.CreateTime.AsTime().Format(time.RFC3339)
		policy.CreateTime = &s
	}
	if mp.UpdateTime != nil {
		s := mp.UpdateTime.AsTime().Format(time.RFC3339)
		policy.UpdateTime = &s
	}
	for _, w := range mp.WeeklyMaintenanceWindow {
		if w == nil {
			continue
		}
		window := mqlClusterMaintenanceWindow{
			Day: w.Day.String(),
		}
		if w.StartTime != nil {
			window.StartTime = fmt.Sprintf("%02d:%02d:%02d", w.StartTime.Hours, w.StartTime.Minutes, w.StartTime.Seconds)
		}
		policy.WeeklyMaintenanceWindow = append(policy.WeeklyMaintenanceWindow, window)
	}
	return convert.JsonToDict(policy)
}

type mqlClusterMaintenanceSchedule struct {
	StartTime *string `json:"startTime"`
	EndTime   *string `json:"endTime"`
}

func clusterConvertMaintenanceSchedule(ms *clusterpb.ClusterMaintenanceSchedule) (map[string]any, error) {
	if ms == nil {
		return nil, nil
	}
	schedule := mqlClusterMaintenanceSchedule{}
	if ms.StartTime != nil {
		s := ms.StartTime.AsTime().Format(time.RFC3339)
		schedule.StartTime = &s
	}
	if ms.EndTime != nil {
		s := ms.EndTime.AsTime().Format(time.RFC3339)
		schedule.EndTime = &s
	}
	return convert.JsonToDict(schedule)
}

type mqlClusterEncryptionInfo struct {
	EncryptionType     string   `json:"encryptionType"`
	KmsKeyVersions     []string `json:"kmsKeyVersions"`
	KmsKeyPrimaryState string   `json:"kmsKeyPrimaryState"`
	LastUpdateTime     *string  `json:"lastUpdateTime"`
}

func clusterConvertEncryptionInfo(ei *clusterpb.EncryptionInfo) (map[string]any, error) {
	if ei == nil {
		return nil, nil
	}
	info := mqlClusterEncryptionInfo{
		EncryptionType:     ei.EncryptionType.String(),
		KmsKeyVersions:     ei.KmsKeyVersions,
		KmsKeyPrimaryState: ei.KmsKeyPrimaryState.String(),
	}
	if ei.LastUpdateTime != nil {
		s := ei.LastUpdateTime.AsTime().Format(time.RFC3339)
		info.LastUpdateTime = &s
	}
	return convert.JsonToDict(info)
}

type mqlClusterAutomatedBackupConfig struct {
	AutomatedBackupMode    string  `json:"automatedBackupMode"`
	Retention              *string `json:"retention"`
	FixedFrequencySchedule *struct {
		StartTime string `json:"startTime"`
	} `json:"fixedFrequencySchedule"`
}

func clusterConvertAutomatedBackupConfig(abc *clusterpb.AutomatedBackupConfig) (map[string]any, error) {
	if abc == nil {
		return nil, nil
	}
	cfg := mqlClusterAutomatedBackupConfig{
		AutomatedBackupMode: abc.AutomatedBackupMode.String(),
	}
	if abc.Retention != nil {
		s := abc.Retention.AsDuration().String()
		cfg.Retention = &s
	}
	if ffs := abc.GetFixedFrequencySchedule(); ffs != nil {
		sched := &struct {
			StartTime string `json:"startTime"`
		}{}
		if ffs.StartTime != nil {
			sched.StartTime = fmt.Sprintf("%02d:%02d:%02d", ffs.StartTime.Hours, ffs.StartTime.Minutes, ffs.StartTime.Seconds)
		}
		cfg.FixedFrequencySchedule = sched
	}
	return convert.JsonToDict(cfg)
}

type mqlClusterRemoteCluster struct {
	Cluster string `json:"cluster"`
	Uid     string `json:"uid"`
}

type mqlClusterCrossClusterReplicationConfig struct {
	ClusterRole       string                    `json:"clusterRole"`
	PrimaryCluster    *mqlClusterRemoteCluster  `json:"primaryCluster"`
	SecondaryClusters []mqlClusterRemoteCluster `json:"secondaryClusters"`
	UpdateTime        *string                   `json:"updateTime"`
}

func clusterConvertCrossClusterReplicationConfig(ccrc *clusterpb.CrossClusterReplicationConfig) (map[string]any, error) {
	if ccrc == nil {
		return nil, nil
	}
	cfg := mqlClusterCrossClusterReplicationConfig{
		ClusterRole: ccrc.ClusterRole.String(),
	}
	if ccrc.PrimaryCluster != nil {
		cfg.PrimaryCluster = &mqlClusterRemoteCluster{
			Cluster: ccrc.PrimaryCluster.Cluster,
			Uid:     ccrc.PrimaryCluster.Uid,
		}
	}
	for _, sc := range ccrc.SecondaryClusters {
		if sc == nil {
			continue
		}
		cfg.SecondaryClusters = append(cfg.SecondaryClusters, mqlClusterRemoteCluster{
			Cluster: sc.Cluster,
			Uid:     sc.Uid,
		})
	}
	if ccrc.UpdateTime != nil {
		s := ccrc.UpdateTime.AsTime().Format(time.RFC3339)
		cfg.UpdateTime = &s
	}
	return convert.JsonToDict(cfg)
}
