// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/bigtable"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// bigtableAdminScopes are the OAuth scopes needed for Bigtable admin operations.
var bigtableAdminScopes = []string{
	bigtable.AdminScope,
	bigtable.InstanceAdminScope,
	"https://www.googleapis.com/auth/cloud-platform",
}

func (g *mqlGcpProject) bigtable() (*mqlGcpProjectBigtableService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	res, err := CreateResource(g.MqlRuntime, "gcp.project.bigtableService", map[string]*llx.RawData{
		"projectId": llx.StringData(g.Id.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectBigtableService), nil
}

func initGcpProjectBigtableService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}
	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	args["projectId"] = llx.StringData(conn.ResourceID())
	return args, nil, nil
}

func (g *mqlGcpProjectBigtableService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("gcp.project/%s/bigtableService", g.ProjectId.Data), nil
}

func (g *mqlGcpProjectBigtableService) instances() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(bigtableAdminScopes...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	iac, err := bigtable.NewInstanceAdminClient(ctx, projectId, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer iac.Close()

	instances, err := iac.Instances(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(instances))
	for _, inst := range instances {
		mqlInst, err := CreateResource(g.MqlRuntime, "gcp.project.bigtableService.instance", map[string]*llx.RawData{
			"projectId":    llx.StringData(projectId),
			"name":         llx.StringData(inst.Name),
			"displayName":  llx.StringData(inst.DisplayName),
			"state":        llx.StringData(bigtableInstanceStateToString(inst.InstanceState)),
			"instanceType": llx.StringData(bigtableInstanceTypeToString(inst.InstanceType)),
			"labels":       llx.MapData(convert.MapToInterfaceMap(inst.Labels), types.String),
			"createdAt":    llx.NilData, // Bigtable SDK doesn't expose creation time in InstanceInfo
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInst)
	}

	return res, nil
}

func bigtableInstanceStateToString(s bigtable.InstanceState) string {
	switch s {
	case bigtable.Ready:
		return "READY"
	case bigtable.Creating:
		return "CREATING"
	default:
		return "STATE_NOT_KNOWN"
	}
}

func bigtableInstanceTypeToString(t bigtable.InstanceType) string {
	switch t {
	case bigtable.PRODUCTION:
		return "PRODUCTION"
	case bigtable.DEVELOPMENT:
		return "DEVELOPMENT"
	default:
		return "UNSPECIFIED"
	}
}

func (g *mqlGcpProjectBigtableServiceInstance) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return fmt.Sprintf("gcp.project/%s/bigtableService/%s", g.ProjectId.Data, g.Name.Data), nil
}

func (g *mqlGcpProjectBigtableServiceInstance) clusters() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	projectId := g.ProjectId.Data
	instanceName := g.Name.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(bigtableAdminScopes...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	iac, err := bigtable.NewInstanceAdminClient(ctx, projectId, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer iac.Close()

	clusters, err := iac.Clusters(ctx, instanceName)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(clusters))
	for _, c := range clusters {
		var autoscalingConfig map[string]any
		if c.AutoscalingConfig != nil {
			autoscalingConfig = map[string]any{
				"minNodes":                  c.AutoscalingConfig.MinNodes,
				"maxNodes":                  c.AutoscalingConfig.MaxNodes,
				"cpuTargetPercent":          c.AutoscalingConfig.CPUTargetPercent,
				"storageUtilizationPerNode": c.AutoscalingConfig.StorageUtilizationPerNode,
			}
		}

		var encryptionConfig map[string]any
		if c.KMSKeyName != "" {
			encryptionConfig = map[string]any{
				"kmsKeyName": c.KMSKeyName,
			}
		}

		mqlCluster, err := CreateResource(g.MqlRuntime, "gcp.project.bigtableService.cluster", map[string]*llx.RawData{
			"projectId":          llx.StringData(projectId),
			"instanceName":       llx.StringData(instanceName),
			"name":               llx.StringData(c.Name),
			"location":           llx.StringData(c.Zone),
			"state":              llx.StringData(c.State),
			"serveNodes":         llx.IntData(int64(c.ServeNodes)),
			"defaultStorageType": llx.StringData(bigtableStorageTypeToString(c.StorageType)),
			"encryptionConfig":   llx.DictData(encryptionConfig),
			"nodeScalingFactor":  llx.StringData(bigtableNodeScalingFactorToString(c.NodeScalingFactor)),
			"autoscalingConfig":  llx.DictData(autoscalingConfig),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCluster)
	}

	return res, nil
}

func bigtableStorageTypeToString(t bigtable.StorageType) string {
	switch t {
	case bigtable.SSD:
		return "SSD"
	case bigtable.HDD:
		return "HDD"
	default:
		return "UNSPECIFIED"
	}
}

func bigtableNodeScalingFactorToString(f bigtable.NodeScalingFactor) string {
	switch f {
	case bigtable.NodeScalingFactor1X:
		return "NODE_SCALING_FACTOR_1X"
	case bigtable.NodeScalingFactor2X:
		return "NODE_SCALING_FACTOR_2X"
	default:
		return "NODE_SCALING_FACTOR_UNSPECIFIED"
	}
}

func (g *mqlGcpProjectBigtableServiceCluster) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return fmt.Sprintf("gcp.project/%s/bigtableService/cluster/%s", g.ProjectId.Data, g.Name.Data), nil
}

func (g *mqlGcpProjectBigtableServiceInstance) tables() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	projectId := g.ProjectId.Data
	instanceName := g.Name.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(bigtableAdminScopes...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	ac, err := bigtable.NewAdminClient(ctx, projectId, instanceName, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer ac.Close()

	tableNames, err := ac.Tables(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(tableNames))
	for _, tableName := range tableNames {
		tableInfo, err := ac.TableInfo(ctx, tableName)
		if err != nil {
			return nil, err
		}

		columnFamilies := map[string]any{}
		for _, fi := range tableInfo.FamilyInfos {
			columnFamilies[fi.Name] = map[string]any{
				"name":         fi.Name,
				"gcPolicy":     fi.GCPolicy,
				"fullGCPolicy": fi.FullGCPolicy.String(),
			}
		}

		var automatedBackupPolicy map[string]any
		if abp, ok := tableInfo.AutomatedBackupConfig.(*bigtable.TableAutomatedBackupPolicy); ok && abp != nil {
			automatedBackupPolicy = map[string]any{
				"retentionPeriod": fmt.Sprintf("%v", abp.RetentionPeriod),
				"frequency":       fmt.Sprintf("%v", abp.Frequency),
			}
		}

		var changeStreamConfig map[string]any
		if tableInfo.ChangeStreamRetention != 0 {
			changeStreamConfig = map[string]any{
				"retentionPeriod": fmt.Sprintf("%v", tableInfo.ChangeStreamRetention),
			}
		}

		deletionProtection := false
		if tableInfo.DeletionProtection == bigtable.Protected {
			deletionProtection = true
		}

		mqlTable, err := CreateResource(g.MqlRuntime, "gcp.project.bigtableService.table", map[string]*llx.RawData{
			"projectId":             llx.StringData(projectId),
			"instanceName":          llx.StringData(instanceName),
			"name":                  llx.StringData(tableName),
			"columnFamilies":        llx.DictData(columnFamilies),
			"granularity":           llx.StringData("MILLIS"),
			"deletionProtection":    llx.BoolData(deletionProtection),
			"automatedBackupPolicy": llx.DictData(automatedBackupPolicy),
			"changeStreamConfig":    llx.DictData(changeStreamConfig),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlTable)
	}

	return res, nil
}

func (g *mqlGcpProjectBigtableServiceTable) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	if g.InstanceName.Error != nil {
		return "", g.InstanceName.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return fmt.Sprintf("gcp.project/%s/bigtableService/%s/table/%s", g.ProjectId.Data, g.InstanceName.Data, g.Name.Data), nil
}

func (g *mqlGcpProjectBigtableServiceInstance) appProfiles() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	projectId := g.ProjectId.Data
	instanceName := g.Name.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(bigtableAdminScopes...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	iac, err := bigtable.NewInstanceAdminClient(ctx, projectId, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer iac.Close()

	it := iac.ListAppProfiles(ctx, instanceName)

	var res []any
	for {
		p, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		routingPolicy := map[string]any{}
		if mcr := p.GetMultiClusterRoutingUseAny(); mcr != nil {
			clusterIds := make([]any, 0, len(mcr.GetClusterIds()))
			for _, cid := range mcr.GetClusterIds() {
				clusterIds = append(clusterIds, cid)
			}
			routingPolicy = map[string]any{
				"type":       "MULTI_CLUSTER_ROUTING_USE_ANY",
				"clusterIds": clusterIds,
			}
		} else if scr := p.GetSingleClusterRouting(); scr != nil {
			routingPolicy = map[string]any{
				"type":                     "SINGLE_CLUSTER_ROUTING",
				"clusterId":                scr.GetClusterId(),
				"allowTransactionalWrites": scr.GetAllowTransactionalWrites(),
			}
		}

		mqlProfile, err := CreateResource(g.MqlRuntime, "gcp.project.bigtableService.appProfile", map[string]*llx.RawData{
			"projectId":     llx.StringData(projectId),
			"instanceName":  llx.StringData(instanceName),
			"name":          llx.StringData(p.GetName()),
			"description":   llx.StringData(p.GetDescription()),
			"routingPolicy": llx.DictData(routingPolicy),
			"etag":          llx.StringData(p.GetEtag()),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlProfile)
	}

	return res, nil
}

func (g *mqlGcpProjectBigtableServiceAppProfile) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	if g.InstanceName.Error != nil {
		return "", g.InstanceName.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return fmt.Sprintf("gcp.project/%s/bigtableService/%s/appProfile/%s", g.ProjectId.Data, g.InstanceName.Data, g.Name.Data), nil
}

func (g *mqlGcpProjectBigtableServiceInstance) backups() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	projectId := g.ProjectId.Data
	instanceName := g.Name.Data

	// First get all clusters for this instance
	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(bigtableAdminScopes...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	iac, err := bigtable.NewInstanceAdminClient(ctx, projectId, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer iac.Close()

	clusters, err := iac.Clusters(ctx, instanceName)
	if err != nil {
		return nil, err
	}

	// For each cluster, list backups
	ac, err := bigtable.NewAdminClient(ctx, projectId, instanceName, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer ac.Close()

	var res []any
	for _, c := range clusters {
		// Extract just the cluster ID from the full name
		clusterID := c.Name
		if idx := strings.LastIndex(c.Name, "/"); idx >= 0 {
			clusterID = c.Name[idx+1:]
		}

		it := ac.Backups(ctx, clusterID)
		for {
			backup, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}

			var encryptionInfo map[string]any
			if backup.EncryptionInfo != nil {
				encryptionInfo = map[string]any{
					"encryptionType":   fmt.Sprintf("%d", backup.EncryptionInfo.Type),
					"encryptionStatus": fmt.Sprintf("%v", backup.EncryptionInfo.Status),
					"kmsKeyVersion":    backup.EncryptionInfo.KMSKeyVersion,
				}
			}

			mqlBackup, err := CreateResource(g.MqlRuntime, "gcp.project.bigtableService.backup", map[string]*llx.RawData{
				"projectId":      llx.StringData(projectId),
				"clusterName":    llx.StringData(c.Name),
				"name":           llx.StringData(backup.Name),
				"sourceTable":    llx.StringData(backup.SourceTable),
				"expireTime":     llx.TimeData(backup.ExpireTime),
				"startTime":      llx.TimeData(backup.StartTime),
				"endTime":        llx.TimeData(backup.EndTime),
				"sizeBytes":      llx.IntData(backup.SizeBytes),
				"state":          llx.StringData(backup.State),
				"encryptionInfo": llx.DictData(encryptionInfo),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlBackup)
		}
	}

	return res, nil
}

func (g *mqlGcpProjectBigtableServiceBackup) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	if g.ClusterName.Error != nil {
		return "", g.ClusterName.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return fmt.Sprintf("gcp.project/%s/bigtableService/%s/backup/%s", g.ProjectId.Data, g.ClusterName.Data, g.Name.Data), nil
}
