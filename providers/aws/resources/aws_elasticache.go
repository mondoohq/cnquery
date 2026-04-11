// Copyright Mondoo, Inc. 2024, 2026
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

			// Batch-fetch replication groups to cache KMS key IDs (avoids N+1 on kmsKey()).
			rgKmsKeys := map[string]*string{}
			rgPaginator := elasticache.NewDescribeReplicationGroupsPaginator(svc, &elasticache.DescribeReplicationGroupsInput{})
			for rgPaginator.HasMorePages() {
				page, err := rgPaginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) || IsServiceNotAvailableInRegionError(err) {
						break
					}
					return nil, err
				}
				for _, rg := range page.ReplicationGroups {
					if rg.ReplicationGroupId != nil {
						rgKmsKeys[*rg.ReplicationGroupId] = rg.KmsKeyId
					}
				}
			}
			for _, r := range res {
				mqlCluster := r.(*mqlAwsElasticacheCluster)
				if mqlCluster.cacheReplicationGroupId != nil {
					mqlCluster.cacheKmsKeyId = rgKmsKeys[*mqlCluster.cacheReplicationGroupId]
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
	cacheKmsKeyId           *string
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

// ── Parameter Groups ────────────────────────────────────────────────────────

func (a *mqlAwsElasticache) parameterGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getParameterGroups(conn), 5)
	poolOfJobs.Run()
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsElasticache) getParameterGroups(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Elasticache(region)
			ctx := context.Background()
			res := []any{}
			paginator := elasticache.NewDescribeCacheParameterGroupsPaginator(svc, &elasticache.DescribeCacheParameterGroupsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) || IsServiceNotAvailableInRegionError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, pg := range page.CacheParameterGroups {
					mqlPg, err := CreateResource(a.MqlRuntime, ResourceAwsElasticacheParameterGroup,
						map[string]*llx.RawData{
							"arn":         llx.StringDataPtr(pg.ARN),
							"name":        llx.StringDataPtr(pg.CacheParameterGroupName),
							"region":      llx.StringData(region),
							"family":      llx.StringDataPtr(pg.CacheParameterGroupFamily),
							"description": llx.StringDataPtr(pg.Description),
							"isGlobal":    llx.BoolDataPtr(pg.IsGlobal),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlPg)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsElasticacheParameterGroup) id() (string, error) {
	return a.Arn.Data, nil
}

// ── Subnet Groups ───────────────────────────────────────────────────────────

func (a *mqlAwsElasticache) subnetGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getSubnetGroups(conn), 5)
	poolOfJobs.Run()
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsElasticache) getSubnetGroups(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Elasticache(region)
			ctx := context.Background()
			res := []any{}
			paginator := elasticache.NewDescribeCacheSubnetGroupsPaginator(svc, &elasticache.DescribeCacheSubnetGroupsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) || IsServiceNotAvailableInRegionError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, sg := range page.CacheSubnetGroups {
					subnets, err := convert.JsonToDictSlice(sg.Subnets)
					if err != nil {
						return nil, err
					}
					netTypes := make([]any, len(sg.SupportedNetworkTypes))
					for i, t := range sg.SupportedNetworkTypes {
						netTypes[i] = string(t)
					}
					mqlSg, err := CreateResource(a.MqlRuntime, ResourceAwsElasticacheSubnetGroup,
						map[string]*llx.RawData{
							"arn":                   llx.StringDataPtr(sg.ARN),
							"name":                  llx.StringDataPtr(sg.CacheSubnetGroupName),
							"region":                llx.StringData(region),
							"description":           llx.StringDataPtr(sg.CacheSubnetGroupDescription),
							"subnets":               llx.ArrayData(subnets, types.Dict),
							"supportedNetworkTypes": llx.ArrayData(netTypes, types.String),
						})
					if err != nil {
						return nil, err
					}
					mqlSgRes := mqlSg.(*mqlAwsElasticacheSubnetGroup)
					mqlSgRes.cacheVpcId = sg.VpcId
					res = append(res, mqlSgRes)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsElasticacheSubnetGroupInternal struct {
	cacheVpcId *string
}

func (a *mqlAwsElasticacheSubnetGroup) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsElasticacheSubnetGroup) vpc() (*mqlAwsVpc, error) {
	if a.cacheVpcId == nil || *a.cacheVpcId == "" {
		a.Vpc.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	vpcArn := fmt.Sprintf(vpcArnPattern, region, conn.AccountId(), *a.cacheVpcId)
	res, err := NewResource(a.MqlRuntime, "aws.vpc", map[string]*llx.RawData{"arn": llx.StringData(vpcArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsVpc), nil
}

// ── Users ───────────────────────────────────────────────────────────────────

func (a *mqlAwsElasticache) users() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getUsers(conn), 5)
	poolOfJobs.Run()
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsElasticache) getUsers(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Elasticache(region)
			ctx := context.Background()
			res := []any{}
			paginator := elasticache.NewDescribeUsersPaginator(svc, &elasticache.DescribeUsersInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) || IsServiceNotAvailableInRegionError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, user := range page.Users {
					groupIds := make([]any, len(user.UserGroupIds))
					for i, g := range user.UserGroupIds {
						groupIds[i] = g
					}
					auth, err := convert.JsonToDict(user.Authentication)
					if err != nil {
						return nil, err
					}
					mqlUser, err := CreateResource(a.MqlRuntime, ResourceAwsElasticacheUser,
						map[string]*llx.RawData{
							"arn":                  llx.StringDataPtr(user.ARN),
							"userId":               llx.StringDataPtr(user.UserId),
							"userName":             llx.StringDataPtr(user.UserName),
							"region":               llx.StringData(region),
							"accessString":         llx.StringDataPtr(user.AccessString),
							"engine":               llx.StringDataPtr(user.Engine),
							"minimumEngineVersion": llx.StringDataPtr(user.MinimumEngineVersion),
							"status":               llx.StringDataPtr(user.Status),
							"userGroupIds":         llx.ArrayData(groupIds, types.String),
							"authentication":       llx.DictData(auth),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlUser)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsElasticacheUser) id() (string, error) {
	return a.Arn.Data, nil
}

// ── Service Updates ─────────────────────────────────────────────────────────

func (a *mqlAwsElasticache) serviceUpdates() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	regions, err := conn.Regions()
	if err != nil {
		return nil, err
	}
	if len(regions) == 0 {
		return nil, nil
	}
	// Service updates are global; fetch from the first available region.
	region := regions[0]
	svc := conn.Elasticache(region)
	ctx := context.Background()
	res := []any{}
	paginator := elasticache.NewDescribeServiceUpdatesPaginator(svc, &elasticache.DescribeServiceUpdatesInput{})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) || IsServiceNotAvailableInRegionError(err) {
				log.Warn().Str("region", region).Msg("error accessing region for AWS API")
				return res, nil
			}
			return nil, err
		}
		for _, su := range page.ServiceUpdates {
			mqlSu, err := CreateResource(a.MqlRuntime, ResourceAwsElasticacheServiceUpdate,
				map[string]*llx.RawData{
					"__id":                                  llx.StringData(fmt.Sprintf("elasticache/serviceupdate/%s", convert.ToValue(su.ServiceUpdateName))),
					"name":                                  llx.StringDataPtr(su.ServiceUpdateName),
					"region":                                llx.StringData(region),
					"description":                           llx.StringDataPtr(su.ServiceUpdateDescription),
					"engine":                                llx.StringDataPtr(su.Engine),
					"engineVersion":                         llx.StringDataPtr(su.EngineVersion),
					"severity":                              llx.StringData(string(su.ServiceUpdateSeverity)),
					"status":                                llx.StringData(string(su.ServiceUpdateStatus)),
					"updateType":                            llx.StringData(string(su.ServiceUpdateType)),
					"releaseDate":                           llx.TimeDataPtr(su.ServiceUpdateReleaseDate),
					"recommendedApplyByDate":                llx.TimeDataPtr(su.ServiceUpdateRecommendedApplyByDate),
					"endDate":                               llx.TimeDataPtr(su.ServiceUpdateEndDate),
					"estimatedUpdateTime":                   llx.StringDataPtr(su.EstimatedUpdateTime),
					"autoUpdateAfterRecommendedApplyByDate": llx.BoolDataPtr(su.AutoUpdateAfterRecommendedApplyByDate),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlSu)
		}
	}
	return res, nil
}

func (a *mqlAwsElasticacheServiceUpdate) id() (string, error) {
	return a.__id, nil
}

// ── Snapshots ───────────────────────────────────────────────────────────────

func (a *mqlAwsElasticache) snapshots() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getSnapshots(conn), 5)
	poolOfJobs.Run()
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsElasticache) getSnapshots(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Elasticache(region)
			ctx := context.Background()
			res := []any{}
			paginator := elasticache.NewDescribeSnapshotsPaginator(svc, &elasticache.DescribeSnapshotsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) || IsServiceNotAvailableInRegionError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, snap := range page.Snapshots {
					mqlSnap, err := CreateResource(a.MqlRuntime, ResourceAwsElasticacheSnapshot,
						map[string]*llx.RawData{
							"arn":                    llx.StringDataPtr(snap.ARN),
							"name":                   llx.StringDataPtr(snap.SnapshotName),
							"region":                 llx.StringData(region),
							"cacheClusterId":         llx.StringDataPtr(snap.CacheClusterId),
							"replicationGroupId":     llx.StringDataPtr(snap.ReplicationGroupId),
							"status":                 llx.StringDataPtr(snap.SnapshotStatus),
							"snapshotSource":         llx.StringDataPtr(snap.SnapshotSource),
							"engine":                 llx.StringDataPtr(snap.Engine),
							"engineVersion":          llx.StringDataPtr(snap.EngineVersion),
							"cacheNodeType":          llx.StringDataPtr(snap.CacheNodeType),
							"numCacheNodes":          llx.IntDataDefault(snap.NumCacheNodes, 0),
							"snapshotRetentionLimit": llx.IntDataDefault(snap.SnapshotRetentionLimit, 0),
							"cacheClusterCreatedAt":  llx.TimeDataPtr(snap.CacheClusterCreateTime),
						})
					if err != nil {
						return nil, err
					}
					mqlSnapRes := mqlSnap.(*mqlAwsElasticacheSnapshot)
					mqlSnapRes.cacheKmsKeyId = snap.KmsKeyId
					mqlSnapRes.cacheVpcId = snap.VpcId
					res = append(res, mqlSnapRes)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsElasticacheSnapshotInternal struct {
	cacheKmsKeyId *string
	cacheVpcId    *string
}

func (a *mqlAwsElasticacheSnapshot) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsElasticacheSnapshot) kmsKey() (*mqlAwsKmsKey, error) {
	if a.cacheKmsKeyId == nil || *a.cacheKmsKeyId == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlKey, err := NewResource(a.MqlRuntime, ResourceAwsKmsKey,
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheKmsKeyId)})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsElasticacheSnapshot) vpc() (*mqlAwsVpc, error) {
	if a.cacheVpcId == nil || *a.cacheVpcId == "" {
		a.Vpc.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	vpcArn := fmt.Sprintf(vpcArnPattern, region, conn.AccountId(), *a.cacheVpcId)
	res, err := NewResource(a.MqlRuntime, "aws.vpc", map[string]*llx.RawData{"arn": llx.StringData(vpcArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsVpc), nil
}
