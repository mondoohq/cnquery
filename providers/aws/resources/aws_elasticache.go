// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/elasticache"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"

	"go.mondoo.com/cnquery/v11/types"
)

func (a *mqlAwsElasticache) id() (string, error) {
	return "aws.elasticache", nil
}

func (a *mqlAwsElasticache) clusters() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getClusters(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.(interface{}))
		}
	}

	return res, nil
}

func (a *mqlAwsElasticache) getClusters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("elasticache>getClusters>calling aws with region %s", regionVal)

			svc := conn.Elasticache(regionVal)
			ctx := context.Background()
			var res interface{}

			var marker *string
			for {
				clusters, err := svc.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{Marker: marker})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				if len(clusters.CacheClusters) == 0 {
					return nil, nil
				}
				if clusters.Marker == nil {
					break
				}
				marker = clusters.Marker
			}
			jsonRes, err := convert.JsonToDictSlice(res)
			if err != nil {
				return nil, err
			}
			return jobpool.JobResult(jsonRes), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsElasticache) cacheClusters() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getCacheClusters(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
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
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("elasticache>getCacheClusters>calling aws with region %s", regionVal)

			svc := conn.Elasticache(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			var marker *string
			for {
				clusters, err := svc.DescribeCacheClusters(ctx, &elasticache.DescribeCacheClustersInput{Marker: marker})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				if len(clusters.CacheClusters) == 0 {
					return nil, nil
				}
				for i := range clusters.CacheClusters {
					cluster := clusters.CacheClusters[i]

					cacheNodes := []interface{}{}
					for i := range cluster.CacheNodes {
						cacheNodes = append(cacheNodes, convert.ToString(cluster.CacheNodes[i].CacheNodeId))
					}
					cacheSecurityGroups := []interface{}{}
					for i := range cluster.CacheSecurityGroups {
						cacheSecurityGroups = append(cacheSecurityGroups, convert.ToString(cluster.CacheSecurityGroups[i].CacheSecurityGroupName))
					}
					logDeliveryConfigurations, err := convert.JsonToDictSlice(cluster.LogDeliveryConfigurations)
					if err != nil {
						return nil, err
					}
					var notificationConfiguration string
					if cluster.NotificationConfiguration != nil {
						notificationConfiguration = convert.ToString(cluster.NotificationConfiguration.TopicArn)
					}

					sgs := []interface{}{}
					for i := range cluster.SecurityGroups {
						sg := cluster.SecurityGroups[i]
						mqlSg, err := NewResource(a.MqlRuntime, "aws.ec2.securitygroup",
							map[string]*llx.RawData{
								"arn": llx.StringData(fmt.Sprintf(securityGroupArnPattern, regionVal, conn.AccountId(), convert.ToString(sg.SecurityGroupId))),
							})
						if err != nil {
							return nil, err
						}
						sgs = append(sgs, mqlSg)
					}

					mqlCluster, err := CreateResource(a.MqlRuntime, "aws.elasticache.cluster",
						map[string]*llx.RawData{
							"__id":                      llx.StringDataPtr(cluster.ARN),
							"arn":                       llx.StringDataPtr(cluster.ARN),
							"atRestEncryptionEnabled":   llx.BoolDataPtr(cluster.AtRestEncryptionEnabled),
							"authTokenEnabled":          llx.BoolDataPtr(cluster.AuthTokenEnabled),
							"authTokenLastModifiedDate": llx.TimeDataPtr(cluster.AuthTokenLastModifiedDate),
							"autoMinorVersionUpgrade":   llx.BoolDataPtr(cluster.AutoMinorVersionUpgrade),
							"cacheClusterCreateTime":    llx.TimeDataPtr(cluster.CacheClusterCreateTime),
							"cacheClusterId":            llx.StringDataPtr(cluster.CacheClusterId),
							"cacheClusterStatus":        llx.StringDataPtr(cluster.CacheClusterStatus),
							"cacheNodeType":             llx.StringDataPtr(cluster.CacheNodeType),
							"cacheNodes":                llx.ArrayData(cacheNodes, types.String),
							"cacheSecurityGroups":       llx.ArrayData(cacheSecurityGroups, types.String),
							"cacheSubnetGroupName":      llx.StringDataPtr(cluster.CacheSubnetGroupName),
							"clientDownloadLandingPage": llx.StringDataPtr(cluster.ClientDownloadLandingPage),
							"nodeType":                  llx.StringDataPtr(cluster.CacheNodeType),
							"engine":                    llx.StringDataPtr(cluster.Engine),
							"engineVersion":             llx.StringDataPtr(cluster.EngineVersion),
							"ipDiscovery":               llx.StringData(string(cluster.IpDiscovery)),
							"logDeliveryConfigurations": llx.ArrayData(logDeliveryConfigurations, types.Any),
							"networkType":               llx.StringData(string(cluster.NetworkType)),
							"notificationConfiguration": llx.StringData(notificationConfiguration),
							"numCacheNodes":             llx.IntDataDefault(cluster.NumCacheNodes, 0),
							"preferredAvailabilityZone": llx.StringDataPtr(cluster.PreferredAvailabilityZone),
							"region":                    llx.StringData(regionVal),
							"securityGroups":            llx.ArrayData(sgs, types.Resource("aws.ec2.securitygroup")),
							"snapshotRetentionLimit":    llx.IntDataDefault(cluster.SnapshotRetentionLimit, 0),
							"transitEncryptionEnabled":  llx.BoolDataPtr(cluster.TransitEncryptionEnabled),
							"transitEncryptionMode":     llx.StringData(string(cluster.TransitEncryptionMode)),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlCluster)
				}
				if clusters.Marker == nil {
					break
				}
				marker = clusters.Marker
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsElasticache) serverlessCaches() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getServerlessCaches(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
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
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("elasticache>getServerlessClusters>calling aws with region %s", regionVal)

			svc := conn.Elasticache(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			var marker *string
			for {
				caches, err := svc.DescribeServerlessCaches(ctx, &elasticache.DescribeServerlessCachesInput{
					NextToken: marker,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				if len(caches.ServerlessCaches) == 0 {
					return nil, nil
				}
				for i := range caches.ServerlessCaches {
					cache := caches.ServerlessCaches[i]

					sgs := []interface{}{}
					for i := range cache.SecurityGroupIds {
						sgId := cache.SecurityGroupIds[i]
						mqlSg, err := NewResource(a.MqlRuntime, "aws.ec2.securitygroup",
							map[string]*llx.RawData{
								"arn": llx.StringData(fmt.Sprintf(securityGroupArnPattern, regionVal, conn.AccountId(), sgId)),
							})
						if err != nil {
							return nil, err
						}
						sgs = append(sgs, mqlSg)
					}

					mqlCluster, err := CreateResource(a.MqlRuntime, "aws.elasticache.serverlessCache",
						map[string]*llx.RawData{
							"__id":                   llx.StringDataPtr(cache.ARN),
							"arn":                    llx.StringDataPtr(cache.ARN),
							"name":                   llx.StringDataPtr(cache.ServerlessCacheName),
							"description":            llx.StringDataPtr(cache.Description),
							"engine":                 llx.StringDataPtr(cache.Engine),
							"engineVersion":          llx.StringDataPtr(cache.FullEngineVersion),
							"majorEngineVersion":     llx.StringDataPtr(cache.MajorEngineVersion),
							"kmsKeyId":               llx.StringDataPtr(cache.KmsKeyId),
							"region":                 llx.StringData(regionVal),
							"securityGroups":         llx.ArrayData(sgs, types.Resource("aws.ec2.securitygroup")),
							"snapshotRetentionLimit": llx.IntDataDefault(cache.SnapshotRetentionLimit, 0),
							"dailySnapshotTime":      llx.StringDataPtr(cache.DailySnapshotTime),
							"createdAt":              llx.TimeDataPtr(cache.CreateTime),
							"status":                 llx.StringDataPtr(cache.Status),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlCluster)
				}
				if caches.NextToken == nil {
					break
				}
				marker = caches.NextToken
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}
