// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	redis "cloud.google.com/go/redis/apiv1"
	"cloud.google.com/go/redis/apiv1/redispb"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers/gcp/connection"
	"go.mondoo.com/cnquery/v11/types"
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

	projectID := conn.ResourceID()
	it := redisSvc.ListInstances(ctx, &redispb.ListInstancesRequest{
		Parent: fmt.Sprintf("projects/%s/locations/-", projectID),
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
			"port":                   llx.IntData(instance.Port),
			"memorySizeGb":           llx.IntData(instance.MemorySizeGb),
			"replicaCount":           llx.IntData(instance.ReplicaCount),
			"readEndpointPort":       llx.IntData(instance.ReadEndpointPort),
			"authEnabled":            llx.BoolData(instance.AuthEnabled),
			"createTime":             llx.TimeData(instance.CreateTime.AsTime()),
			"labels":                 llx.MapData(convert.MapToInterfaceMap(instance.Labels), types.String),
			"redisConfigs":           llx.MapData(convert.MapToInterfaceMap(instance.RedisConfigs), types.String),
			"availableMaintenanceVersions": llx.ArrayData(
				convert.SliceAnyToInterface(instance.AvailableMaintenanceVersions), types.String,
			),
			"nodes": llx.ArrayData(
				redisInstanceNodesToArrayInterface(g.MqlRuntime, projectID, instance.Nodes),
				types.Resource("gcp.project.redisService.instance.nodeInfo"),
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
