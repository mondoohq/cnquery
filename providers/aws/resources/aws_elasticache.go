// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	elasticache_types "github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsElasticache) id() (string, error) {
	return "aws.elasticache", nil
}

func (a *mqlAwsElasticache) cacheClusters() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getCacheClusters(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for _, job := range poolOfJobs.Jobs {
		if job.Result != nil {
			res = append(res, job.Result.([]any)...)
		}
	}

	return res, nil
}

func (a *mqlAwsElasticache) getCacheClusters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("elasticache>getCacheClusters>calling aws with region %s", region)

			svc := conn.Elasticache(region)
			ctx := context.Background()
			res := []any{}

			params := &elasticache.DescribeCacheClustersInput{}
			paginator := elasticache.NewDescribeCacheClustersPaginator(svc, params)
			for paginator.HasMorePages() {
				clusters, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("elasticache service not available in region")
						return res, nil
					}
					return nil, err
				}
				for _, cluster := range clusters.CacheClusters {
					mqlCluster, err := newMqlAwsElasticacheCluster(a.MqlRuntime, region, conn.AccountId(), cluster)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlCluster)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsElasticacheClusterInternal struct {
	securityGroupIdHandler
	cacheReplicationGroupId *string
	region                  string
}

func newMqlAwsElasticacheCluster(runtime *plugin.Runtime, region string, accountID string, cluster elasticache_types.CacheCluster) (*mqlAwsElasticacheCluster, error) {
	cacheNodes := []any{}
	for i := range cluster.CacheNodes {
		cacheNodes = append(cacheNodes, convert.ToValue(cluster.CacheNodes[i].CacheNodeId))
	}
	cacheSecurityGroups := []any{}
	for _, sg := range cluster.CacheSecurityGroups {
		cacheSecurityGroups = append(cacheSecurityGroups, convert.ToValue(sg.CacheSecurityGroupName))
	}
	logDeliveryConfigurations, err := convert.JsonToDictSlice(cluster.LogDeliveryConfigurations)
	if err != nil {
		return nil, err
	}
	var notificationConfiguration string
	if cluster.NotificationConfiguration != nil {
		notificationConfiguration = convert.ToValue(cluster.NotificationConfiguration.TopicArn)
	}

	sgs := []string{}
	for _, sg := range cluster.SecurityGroups {
		if sg.SecurityGroupId == nil {
			log.Debug().Msgf("elasticache>newMqlAwsElasticacheCluster>missing security group id for cluster %s", *cluster.CacheClusterId)
			continue
		}
		sgs = append(sgs, NewSecurityGroupArn(region, accountID, convert.ToValue(sg.SecurityGroupId)))
	}

	resource, err := CreateResource(runtime, "aws.elasticache.cluster",
		map[string]*llx.RawData{
			"__id":                               llx.StringDataPtr(cluster.ARN),
			"arn":                                llx.StringDataPtr(cluster.ARN),
			"atRestEncryptionEnabled":            llx.BoolDataPtr(cluster.AtRestEncryptionEnabled),
			"authTokenEnabled":                   llx.BoolDataPtr(cluster.AuthTokenEnabled),
			"authTokenLastModifiedDate":          llx.TimeDataPtr(cluster.AuthTokenLastModifiedDate),
			"autoMinorVersionUpgrade":            llx.BoolDataPtr(cluster.AutoMinorVersionUpgrade),
			"cacheClusterCreateTime":             llx.TimeDataPtr(cluster.CacheClusterCreateTime),
			"cacheClusterId":                     llx.StringDataPtr(cluster.CacheClusterId),
			"cacheClusterStatus":                 llx.StringDataPtr(cluster.CacheClusterStatus),
			"cacheNodeType":                      llx.StringDataPtr(cluster.CacheNodeType),
			"cacheNodes":                         llx.ArrayData(cacheNodes, types.String),
			"cacheSecurityGroups":                llx.ArrayData(cacheSecurityGroups, types.String),
			"cacheSubnetGroupName":               llx.StringDataPtr(cluster.CacheSubnetGroupName),
			"clientDownloadLandingPage":          llx.StringDataPtr(cluster.ClientDownloadLandingPage),
			"nodeType":                           llx.StringDataPtr(cluster.CacheNodeType),
			"engine":                             llx.StringDataPtr(cluster.Engine),
			"engineVersion":                      llx.StringDataPtr(cluster.EngineVersion),
			"ipDiscovery":                        llx.StringData(string(cluster.IpDiscovery)),
			"logDeliveryConfigurations":          llx.ArrayData(logDeliveryConfigurations, types.Any),
			"networkType":                        llx.StringData(string(cluster.NetworkType)),
			"notificationConfiguration":          llx.StringData(notificationConfiguration),
			"numCacheNodes":                      llx.IntDataDefault(cluster.NumCacheNodes, 0),
			"preferredAvailabilityZone":          llx.StringDataPtr(cluster.PreferredAvailabilityZone),
			"region":                             llx.StringData(region),
			"snapshotRetentionLimit":             llx.IntDataDefault(cluster.SnapshotRetentionLimit, 0),
			"snapshotWindow":                     llx.StringDataPtr(cluster.SnapshotWindow),
			"transitEncryptionEnabled":           llx.BoolDataPtr(cluster.TransitEncryptionEnabled),
			"transitEncryptionMode":              llx.StringData(string(cluster.TransitEncryptionMode)),
			"preferredMaintenanceWindow":         llx.StringDataPtr(cluster.PreferredMaintenanceWindow),
			"replicationGroupLogDeliveryEnabled": llx.BoolDataPtr(cluster.ReplicationGroupLogDeliveryEnabled),
		})
	if err != nil {
		return nil, err
	}

	mqlCluster := resource.(*mqlAwsElasticacheCluster)
	mqlCluster.setSecurityGroupArns(sgs)
	mqlCluster.cacheReplicationGroupId = cluster.ReplicationGroupId
	mqlCluster.region = region
	return mqlCluster, nil
}

func (a *mqlAwsElasticacheCluster) securityGroups() ([]any, error) {
	return a.newSecurityGroupResources(a.MqlRuntime)
}

func (a *mqlAwsElasticacheCluster) kmsKey() (*mqlAwsKmsKey, error) {
	if a.cacheReplicationGroupId == nil || *a.cacheReplicationGroupId == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Elasticache(a.region)
	ctx := context.Background()
	resp, err := svc.DescribeReplicationGroups(ctx, &elasticache.DescribeReplicationGroupsInput{
		ReplicationGroupId: a.cacheReplicationGroupId,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.ReplicationGroups) == 0 || resp.ReplicationGroups[0].KmsKeyId == nil || *resp.ReplicationGroups[0].KmsKeyId == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlKey, err := NewResource(a.MqlRuntime, ResourceAwsKmsKey,
		map[string]*llx.RawData{
			"arn": llx.StringDataPtr(resp.ReplicationGroups[0].KmsKeyId),
		})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsElasticache) serverlessCaches() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getServerlessCaches(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}

	return res, nil
}

func (a *mqlAwsElasticache) getServerlessCaches(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("elasticache>getServerlessClusters>calling aws with region %s", region)

			svc := conn.Elasticache(region)
			ctx := context.Background()
			res := []any{}

			params := &elasticache.DescribeServerlessCachesInput{}
			paginator := elasticache.NewDescribeServerlessCachesPaginator(svc, params)
			for paginator.HasMorePages() {
				caches, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("elasticache service not available in region")
						return res, nil
					}
					return nil, err
				}
				for _, cache := range caches.ServerlessCaches {
					mqlCache, err := newMqlAwsElasticacheServerlessCache(a.MqlRuntime, region, conn.AccountId(), cache)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlCache)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsElasticacheServerlessCacheInternal struct {
	securityGroupIdHandler
	region        string
	accountID     string
	subnetIds     []string
	cacheKmsKeyId *string
}

func newMqlAwsElasticacheServerlessCache(runtime *plugin.Runtime, region string, accountID string, cache elasticache_types.ServerlessCache) (*mqlAwsElasticacheServerlessCache, error) {
	sgArgs := []string{}
	for i := range cache.SecurityGroupIds {
		sgId := cache.SecurityGroupIds[i]
		sgArgs = append(sgArgs, NewSecurityGroupArn(region, accountID, sgId))
	}

	resource, err := CreateResource(runtime, "aws.elasticache.serverlessCache",
		map[string]*llx.RawData{
			"__id":                   llx.StringDataPtr(cache.ARN),
			"arn":                    llx.StringDataPtr(cache.ARN),
			"name":                   llx.StringDataPtr(cache.ServerlessCacheName),
			"description":            llx.StringDataPtr(cache.Description),
			"engine":                 llx.StringDataPtr(cache.Engine),
			"engineVersion":          llx.StringDataPtr(cache.FullEngineVersion),
			"majorEngineVersion":     llx.StringDataPtr(cache.MajorEngineVersion),
			"kmsKeyId":               llx.StringDataPtr(cache.KmsKeyId),
			"region":                 llx.StringData(region),
			"snapshotRetentionLimit": llx.IntDataDefault(cache.SnapshotRetentionLimit, 0),
			"dailySnapshotTime":      llx.StringDataPtr(cache.DailySnapshotTime),
			"createdAt":              llx.TimeDataPtr(cache.CreateTime),
			"status":                 llx.StringDataPtr(cache.Status),
		})
	if err != nil {
		return nil, err
	}

	mqlCache := resource.(*mqlAwsElasticacheServerlessCache)
	mqlCache.setSecurityGroupArns(sgArgs)
	mqlCache.region = region
	mqlCache.accountID = accountID
	mqlCache.subnetIds = cache.SubnetIds
	mqlCache.cacheKmsKeyId = cache.KmsKeyId
	return mqlCache, nil
}

func (a *mqlAwsElasticacheServerlessCache) kmsKey() (*mqlAwsKmsKey, error) {
	if a.cacheKmsKeyId == nil || *a.cacheKmsKeyId == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlKey, err := NewResource(a.MqlRuntime, ResourceAwsKmsKey,
		map[string]*llx.RawData{
			"arn": llx.StringDataPtr(a.cacheKmsKeyId),
		})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsElasticacheServerlessCache) securityGroups() ([]any, error) {
	return a.newSecurityGroupResources(a.MqlRuntime)
}

func (a *mqlAwsElasticacheServerlessCache) subnets() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	for _, subnetId := range a.subnetIds {
		mqlSubnet, err := NewResource(a.MqlRuntime, ResourceAwsVpcSubnet,
			map[string]*llx.RawData{
				"arn": llx.StringData(fmt.Sprintf(subnetArnPattern, a.region, conn.AccountId(), subnetId)),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSubnet)
	}
	return res, nil
}

func initAwsElasticacheCluster(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["cacheClusterId"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch elasticache cluster")
	}

	obj, err := CreateResource(runtime, "aws.elasticache", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}

	ec := obj.(*mqlAwsElasticache)
	rawResources := ec.GetCacheClusters()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	arnVal, ok := args["arn"].Value.(string)
	if !ok {
		return nil, nil, errors.New("arn must be a string")
	}
	for _, rawResource := range rawResources.Data {
		cluster := rawResource.(*mqlAwsElasticacheCluster)
		if cluster.Arn.Data == arnVal {
			return args, cluster, nil
		}
	}
	return nil, nil, errors.New("elasticache cluster does not exist")
}
