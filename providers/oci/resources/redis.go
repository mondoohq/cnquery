// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/redis"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciRedis) id() (string, error) {
	return "oci.redis", nil
}

func (o *mqlOciRedis) clusters() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	ociResource, err := CreateResource(o.MqlRuntime, "oci", nil)
	if err != nil {
		return nil, err
	}
	oci := ociResource.(*mqlOci)
	regions := oci.GetRegions()
	if regions.Error != nil {
		return nil, regions.Error
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(o.getClusters(conn, regions.Data), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (o *mqlOciRedis) getClusters(conn *connection.OciConnection, regions []any) []*jobpool.Job {
	ctx := context.Background()
	tasks := make([]*jobpool.Job, 0)
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return jobErr(errors.New("invalid region type"))
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling oci redis cluster with region %s", regionResource.Id.Data)

			svc, err := conn.RedisClusterClient(regionResource.Id.Data)
			if err != nil {
				return nil, err
			}

			clusters := []redis.RedisClusterSummary{}
			var page *string
			for {
				response, err := svc.ListRedisClusters(ctx, redis.ListRedisClustersRequest{
					CompartmentId: common.String(conn.TenantID()),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				clusters = append(clusters, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			res := make([]any, 0, len(clusters))
			for i := range clusters {
				c := clusters[i]

				var created, updated *time.Time
				if c.TimeCreated != nil {
					created = &c.TimeCreated.Time
				}
				if c.TimeUpdated != nil {
					updated = &c.TimeUpdated.Time
				}

				var nodeMemory float64
				if c.NodeMemoryInGBs != nil {
					nodeMemory = float64(*c.NodeMemoryInGBs)
				}

				freeformTags := make(map[string]interface{}, len(c.FreeformTags))
				for k, v := range c.FreeformTags {
					freeformTags[k] = v
				}
				definedTags := make(map[string]interface{}, len(c.DefinedTags))
				for k, v := range c.DefinedTags {
					definedTags[k] = v
				}

				mqlCluster, err := CreateResource(o.MqlRuntime, "oci.redis.cluster", map[string]*llx.RawData{
					"id":                         llx.StringDataPtr(c.Id),
					"name":                       llx.StringDataPtr(c.DisplayName),
					"compartmentID":              llx.StringDataPtr(c.CompartmentId),
					"softwareVersion":            llx.StringData(string(c.SoftwareVersion)),
					"clusterMode":                llx.StringData(string(c.ClusterMode)),
					"nodeCount":                  llx.IntDataPtr(c.NodeCount),
					"nodeMemoryInGBs":            llx.FloatData(nodeMemory),
					"shardCount":                 llx.IntDataPtr(c.ShardCount),
					"primaryFqdn":                llx.StringDataPtr(c.PrimaryFqdn),
					"primaryEndpointIpAddress":   llx.StringDataPtr(c.PrimaryEndpointIpAddress),
					"replicasFqdn":               llx.StringDataPtr(c.ReplicasFqdn),
					"replicasEndpointIpAddress":  llx.StringDataPtr(c.ReplicasEndpointIpAddress),
					"discoveryFqdn":              llx.StringDataPtr(c.DiscoveryFqdn),
					"discoveryEndpointIpAddress": llx.StringDataPtr(c.DiscoveryEndpointIpAddress),
					"state":                      llx.StringData(string(c.LifecycleState)),
					"created":                    llx.TimeDataPtr(created),
					"timeUpdated":                llx.TimeDataPtr(updated),
					"freeformTags":               llx.MapData(freeformTags, types.String),
					"definedTags":                llx.MapData(definedTags, types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlClusterTyped := mqlCluster.(*mqlOciRedisCluster)
				mqlClusterTyped.cacheSubnetId = stringValue(c.SubnetId)
				mqlClusterTyped.cacheNsgIds = c.NsgIds
				mqlClusterTyped.cacheRegion = regionResource.Id.Data
				res = append(res, mqlClusterTyped)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlOciRedisClusterInternal struct {
	cacheSubnetId string
	cacheNsgIds   []string
	cacheRegion   string
}

func initOciRedisCluster(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	idVal := ociArgString(args, "id")
	if idVal == "" {
		conn := runtime.Connection.(*connection.OciConnection)
		if conn.Conf == nil || conn.Conf.PlatformId == "" {
			return args, nil, nil
		}
		parsed, ok := parseOciObjectPlatformID(conn.Conf.PlatformId)
		if !ok || parsed.service != "redis" || parsed.objectType != "cluster" {
			return args, nil, nil
		}
		idVal = parsed.id
	}

	obj, err := CreateResource(runtime, "oci.redis", nil)
	if err != nil {
		return nil, nil, err
	}
	svc := obj.(*mqlOciRedis)

	rawClusters := svc.GetClusters()
	if rawClusters.Error != nil {
		return nil, nil, rawClusters.Error
	}

	for _, raw := range rawClusters.Data {
		c := raw.(*mqlOciRedisCluster)
		if c.Id.Data == idVal {
			return args, c, nil
		}
	}

	return args, nil, nil
}

func (o *mqlOciRedisCluster) id() (string, error) {
	return "oci.redis.cluster/" + o.Id.Data, nil
}

func (o *mqlOciRedisCluster) subnet() (*mqlOciNetworkSubnet, error) {
	if o.cacheSubnetId == "" {
		o.Subnet.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	mqlSubnet, err := NewResource(o.MqlRuntime, "oci.network.subnet", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheSubnetId),
	})
	if err != nil {
		return nil, err
	}
	return mqlSubnet.(*mqlOciNetworkSubnet), nil
}

func (o *mqlOciRedisCluster) networkSecurityGroups() ([]any, error) {
	if len(o.cacheNsgIds) == 0 {
		return []any{}, nil
	}
	res := make([]any, 0, len(o.cacheNsgIds))
	for _, nsgId := range o.cacheNsgIds {
		mqlNsg, err := NewResource(o.MqlRuntime, "oci.network.networkSecurityGroup", map[string]*llx.RawData{
			"id": llx.StringData(nsgId),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlNsg)
	}
	return res, nil
}
