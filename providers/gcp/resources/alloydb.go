// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	alloydb "cloud.google.com/go/alloydb/apiv1"
	"cloud.google.com/go/alloydb/apiv1/alloydbpb"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func (g *mqlGcpProject) alloydb() (*mqlGcpProjectAlloydbService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	res, err := CreateResource(g.MqlRuntime, "gcp.project.alloydbService", map[string]*llx.RawData{
		"projectId": llx.StringData(g.Id.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectAlloydbService), nil
}

func initGcpProjectAlloydbService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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

func (g *mqlGcpProjectAlloydbService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("gcp.project/%s/alloydbService", g.ProjectId.Data), nil
}

func (g *mqlGcpProjectAlloydbService) clusters() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(alloydb.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := alloydb.NewAlloyDBAdminClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListClusters(ctx, &alloydbpb.ListClustersRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", projectId),
	})

	var res []any
	for {
		cluster, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		networkConfig, err := protoToDict(cluster.NetworkConfig)
		if err != nil {
			return nil, err
		}
		encryptionConfig, err := protoToDict(cluster.EncryptionConfig)
		if err != nil {
			return nil, err
		}
		encryptionInfo, err := protoToDict(cluster.EncryptionInfo)
		if err != nil {
			return nil, err
		}
		automatedBackupPolicy, err := protoToDict(cluster.AutomatedBackupPolicy)
		if err != nil {
			return nil, err
		}
		continuousBackupConfig, err := protoToDict(cluster.ContinuousBackupConfig)
		if err != nil {
			return nil, err
		}
		continuousBackupInfo, err := protoToDict(cluster.ContinuousBackupInfo)
		if err != nil {
			return nil, err
		}
		sslConfig, err := protoToDict(cluster.SslConfig)
		if err != nil {
			return nil, err
		}
		primaryConfig, err := protoToDict(cluster.PrimaryConfig)
		if err != nil {
			return nil, err
		}
		secondaryConfig, err := protoToDict(cluster.SecondaryConfig)
		if err != nil {
			return nil, err
		}
		maintenanceUpdatePolicy, err := protoToDict(cluster.MaintenanceUpdatePolicy)
		if err != nil {
			return nil, err
		}
		maintenanceSchedule, err := protoToDict(cluster.MaintenanceSchedule)
		if err != nil {
			return nil, err
		}

		var createdAt *llx.RawData
		if cluster.CreateTime != nil {
			createdAt = llx.TimeData(cluster.CreateTime.AsTime())
		} else {
			createdAt = llx.NilData
		}

		var updatedAt *llx.RawData
		if cluster.UpdateTime != nil {
			updatedAt = llx.TimeData(cluster.UpdateTime.AsTime())
		} else {
			updatedAt = llx.NilData
		}

		// Extract location from the cluster name: projects/{project}/locations/{location}/clusters/{cluster}
		location := ""
		if parts := parseAlloyDBClusterName(cluster.Name); parts != nil {
			location = parts.location
		}

		mqlCluster, err := CreateResource(g.MqlRuntime, "gcp.project.alloydbService.cluster", map[string]*llx.RawData{
			"projectId":               llx.StringData(projectId),
			"name":                    llx.StringData(cluster.Name),
			"displayName":             llx.StringData(cluster.DisplayName),
			"uid":                     llx.StringData(cluster.Uid),
			"state":                   llx.StringData(cluster.State.String()),
			"clusterType":             llx.StringData(cluster.ClusterType.String()),
			"databaseVersion":         llx.StringData(cluster.DatabaseVersion.String()),
			"networkConfig":           llx.DictData(networkConfig),
			"encryptionConfig":        llx.DictData(encryptionConfig),
			"encryptionInfo":          llx.DictData(encryptionInfo),
			"automatedBackupPolicy":   llx.DictData(automatedBackupPolicy),
			"continuousBackupConfig":  llx.DictData(continuousBackupConfig),
			"continuousBackupInfo":    llx.DictData(continuousBackupInfo),
			"sslConfig":               llx.DictData(sslConfig),
			"labels":                  llx.MapData(convert.MapToInterfaceMap(cluster.Labels), types.String),
			"annotations":             llx.MapData(convert.MapToInterfaceMap(cluster.Annotations), types.String),
			"primaryConfig":           llx.DictData(primaryConfig),
			"secondaryConfig":         llx.DictData(secondaryConfig),
			"maintenanceUpdatePolicy": llx.DictData(maintenanceUpdatePolicy),
			"maintenanceSchedule":     llx.DictData(maintenanceSchedule),
			"location":                llx.StringData(location),
			"reconciling":             llx.BoolData(cluster.Reconciling),
			"etag":                    llx.StringData(cluster.Etag),
			"createdAt":               createdAt,
			"updatedAt":               updatedAt,
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCluster)
	}

	return res, nil
}

type alloyDBClusterParts struct {
	project  string
	location string
	cluster  string
}

func parseAlloyDBClusterName(name string) *alloyDBClusterParts {
	// Format: projects/{project}/locations/{location}/clusters/{cluster}
	parts := strings.Split(name, "/")
	if len(parts) < 6 {
		return nil
	}
	return &alloyDBClusterParts{
		project:  parts[1],
		location: parts[3],
		cluster:  parts[5],
	}
}

func (g *mqlGcpProjectAlloydbServiceCluster) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return fmt.Sprintf("gcp.project/%s/alloydbService/%s", g.ProjectId.Data, g.Name.Data), nil
}

func (g *mqlGcpProjectAlloydbServiceCluster) instances() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	projectId := g.ProjectId.Data
	clusterName := g.Name.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(alloydb.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := alloydb.NewAlloyDBAdminClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	it := client.ListInstances(ctx, &alloydbpb.ListInstancesRequest{
		Parent: clusterName,
	})

	var res []any
	for {
		inst, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		machineConfig, err := protoToDict(inst.MachineConfig)
		if err != nil {
			return nil, err
		}
		queryInsightsConfig, err := protoToDict(inst.QueryInsightsConfig)
		if err != nil {
			return nil, err
		}
		readPoolConfig, err := protoToDict(inst.ReadPoolConfig)
		if err != nil {
			return nil, err
		}
		clientConnectionConfig, err := protoToDict(inst.ClientConnectionConfig)
		if err != nil {
			return nil, err
		}
		pscInstanceConfig, err := protoToDict(inst.PscInstanceConfig)
		if err != nil {
			return nil, err
		}
		writableNode, err := protoToDict(inst.WritableNode)
		if err != nil {
			return nil, err
		}

		nodes := make([]any, 0, len(inst.Nodes))
		for _, node := range inst.Nodes {
			nodeDict, err := protoToDict(node)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, nodeDict)
		}

		var createdAt *llx.RawData
		if inst.CreateTime != nil {
			createdAt = llx.TimeData(inst.CreateTime.AsTime())
		} else {
			createdAt = llx.NilData
		}

		var updatedAt *llx.RawData
		if inst.UpdateTime != nil {
			updatedAt = llx.TimeData(inst.UpdateTime.AsTime())
		} else {
			updatedAt = llx.NilData
		}

		mqlInst, err := CreateResource(g.MqlRuntime, "gcp.project.alloydbService.instance", map[string]*llx.RawData{
			"projectId":              llx.StringData(projectId),
			"clusterName":            llx.StringData(clusterName),
			"name":                   llx.StringData(inst.Name),
			"displayName":            llx.StringData(inst.DisplayName),
			"uid":                    llx.StringData(inst.Uid),
			"state":                  llx.StringData(inst.State.String()),
			"instanceType":           llx.StringData(inst.InstanceType.String()),
			"machineConfig":          llx.DictData(machineConfig),
			"availabilityType":       llx.StringData(inst.AvailabilityType.String()),
			"gceZone":                llx.StringData(inst.GceZone),
			"ipAddress":              llx.StringData(inst.IpAddress),
			"publicIpAddress":        llx.StringData(inst.PublicIpAddress),
			"databaseFlags":          llx.MapData(convert.MapToInterfaceMap(inst.DatabaseFlags), types.String),
			"labels":                 llx.MapData(convert.MapToInterfaceMap(inst.Labels), types.String),
			"queryInsightsConfig":    llx.DictData(queryInsightsConfig),
			"readPoolConfig":         llx.DictData(readPoolConfig),
			"clientConnectionConfig": llx.DictData(clientConnectionConfig),
			"pscInstanceConfig":      llx.DictData(pscInstanceConfig),
			"nodes":                  llx.ArrayData(nodes, types.Dict),
			"writableNode":           llx.DictData(writableNode),
			"reconciling":            llx.BoolData(inst.Reconciling),
			"etag":                   llx.StringData(inst.Etag),
			"createdAt":              createdAt,
			"updatedAt":              updatedAt,
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlInst)
	}

	return res, nil
}

func (g *mqlGcpProjectAlloydbServiceInstance) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return fmt.Sprintf("gcp.project/%s/alloydbService/%s", g.ProjectId.Data, g.Name.Data), nil
}

func (g *mqlGcpProjectAlloydbServiceCluster) backups() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	projectId := g.ProjectId.Data
	clusterName := g.Name.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(alloydb.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := alloydb.NewAlloyDBAdminClient(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// Extract location from cluster name for the backup list parent
	location := ""
	if parts := parseAlloyDBClusterName(clusterName); parts != nil {
		location = parts.location
	}
	if location == "" {
		location = "-"
	}

	it := client.ListBackups(ctx, &alloydbpb.ListBackupsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, location),
		Filter: fmt.Sprintf("cluster_name=%q", clusterName),
	})

	var res []any
	for {
		backup, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		encryptionConfig, err := protoToDict(backup.EncryptionConfig)
		if err != nil {
			return nil, err
		}
		encryptionInfo, err := protoToDict(backup.EncryptionInfo)
		if err != nil {
			return nil, err
		}
		expiryQuantity, err := protoToDict(backup.ExpiryQuantity)
		if err != nil {
			return nil, err
		}

		var createdAt *llx.RawData
		if backup.CreateTime != nil {
			createdAt = llx.TimeData(backup.CreateTime.AsTime())
		} else {
			createdAt = llx.NilData
		}

		var updatedAt *llx.RawData
		if backup.UpdateTime != nil {
			updatedAt = llx.TimeData(backup.UpdateTime.AsTime())
		} else {
			updatedAt = llx.NilData
		}

		var expiryTime *llx.RawData
		if backup.ExpiryTime != nil {
			expiryTime = llx.TimeData(backup.ExpiryTime.AsTime())
		} else {
			expiryTime = llx.NilData
		}

		mqlBackup, err := CreateResource(g.MqlRuntime, "gcp.project.alloydbService.backup", map[string]*llx.RawData{
			"projectId":        llx.StringData(projectId),
			"name":             llx.StringData(backup.Name),
			"displayName":      llx.StringData(backup.DisplayName),
			"uid":              llx.StringData(backup.Uid),
			"state":            llx.StringData(backup.State.String()),
			"type":             llx.StringData(backup.Type.String()),
			"description":      llx.StringData(backup.Description),
			"clusterName":      llx.StringData(backup.ClusterName),
			"databaseVersion":  llx.StringData(backup.DatabaseVersion.String()),
			"encryptionConfig": llx.DictData(encryptionConfig),
			"encryptionInfo":   llx.DictData(encryptionInfo),
			"sizeBytes":        llx.IntData(backup.SizeBytes),
			"expiryTime":       expiryTime,
			"expiryQuantity":   llx.DictData(expiryQuantity),
			"labels":           llx.MapData(convert.MapToInterfaceMap(backup.Labels), types.String),
			"etag":             llx.StringData(backup.Etag),
			"reconciling":      llx.BoolData(backup.Reconciling),
			"createdAt":        createdAt,
			"updatedAt":        updatedAt,
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBackup)
	}

	return res, nil
}

func (g *mqlGcpProjectAlloydbServiceBackup) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	if g.Name.Error != nil {
		return "", g.Name.Error
	}
	return fmt.Sprintf("gcp.project/%s/alloydbService/%s", g.ProjectId.Data, g.Name.Data), nil
}
