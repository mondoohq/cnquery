// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (e *mqlAwsElasticache) id() (string, error) {
	return "aws.elasticache", nil
}

func (e *mqlAwsElasticache) GetClusters() ([]interface{}, error) {
	provider, err := awsProvider(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getClusters(provider), 5)
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

func (e *mqlAwsElasticache) getClusters(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.Elasticache(regionVal)
			ctx := context.Background()
			res := []types.CacheCluster{}

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
				res = append(res, clusters.CacheClusters...)
				if clusters.Marker == nil {
					break
				}
				marker = clusters.Marker
			}
			jsonRes, err := core.JsonToDictSlice(res)
			if err != nil {
				return nil, err
			}
			return jobpool.JobResult(jsonRes), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (e *mqlAwsElasticache) GetCacheClusters() ([]interface{}, error) {
	provider, err := awsProvider(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getCacheClusters(provider), 5)
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

func (e *mqlAwsElasticacheCluster) id() (string, error) {
	return e.Arn()
}

func (e *mqlAwsElasticache) getCacheClusters(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	account, err := provider.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.Elasticache(regionVal)
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
						cacheNodes = append(cacheNodes, core.ToString(cluster.CacheNodes[i].CacheNodeId))
					}
					cacheSecurityGroups := []interface{}{}
					for i := range cluster.CacheSecurityGroups {
						cacheSecurityGroups = append(cacheSecurityGroups, core.ToString(cluster.CacheSecurityGroups[i].CacheSecurityGroupName))
					}
					logDeliveryConfigurations, err := core.JsonToDictSlice(cluster.LogDeliveryConfigurations)
					if err != nil {
						return nil, err
					}
					var notificationConfiguration string
					if cluster.NotificationConfiguration != nil {
						notificationConfiguration = core.ToString(cluster.NotificationConfiguration.TopicArn)
					}

					sgs := []interface{}{}
					for i := range cluster.SecurityGroups {
						sg := cluster.SecurityGroups[i]
						mqlSg, err := e.MotorRuntime.CreateResource("aws.ec2.securitygroup",
							"arn", fmt.Sprintf(securityGroupArnPattern, regionVal, account.ID, core.ToString(sg.SecurityGroupId)),
						)
						if err != nil {
							return nil, err
						}
						sgs = append(sgs, mqlSg)
					}

					mqlCluster, err := e.MotorRuntime.CreateResource("aws.elasticache.cluster",
						"arn", core.ToString(cluster.ARN),
						"atRestEncryptionEnabled", core.ToBool(cluster.AtRestEncryptionEnabled),
						"authTokenEnabled", core.ToBool(cluster.AuthTokenEnabled),
						"authTokenLastModifiedDate", cluster.AuthTokenLastModifiedDate,
						"autoMinorVersionUpgrade", cluster.AutoMinorVersionUpgrade,
						"cacheClusterCreateTime", cluster.CacheClusterCreateTime,
						"cacheClusterId", core.ToString(cluster.CacheClusterId),
						"cacheClusterStatus", core.ToString(cluster.CacheClusterStatus),
						"cacheNodeType", core.ToString(cluster.CacheNodeType),
						"cacheNodes", cacheNodes,
						"cacheSecurityGroups", cacheSecurityGroups,
						"cacheSubnetGroupName", core.ToString(cluster.CacheSubnetGroupName),
						"clientDownloadLandingPage", core.ToString(cluster.ClientDownloadLandingPage),
						"nodeType", core.ToString(cluster.CacheNodeType),
						"engine", core.ToString(cluster.Engine),
						"engineVersion", core.ToString(cluster.EngineVersion),
						"ipDiscovery", string(cluster.IpDiscovery),
						"logDeliveryConfigurations", logDeliveryConfigurations,
						"networkType", string(cluster.NetworkType),
						"notificationConfiguration", notificationConfiguration,
						"numCacheNodes", core.ToInt64From32(cluster.NumCacheNodes),
						"preferredAvailabilityZone", core.ToString(cluster.PreferredAvailabilityZone),
						"region", regionVal,
						"securityGroups", sgs,
						"snapshotRetentionLimit", core.ToInt64From32(cluster.SnapshotRetentionLimit),
						"transitEncryptionEnabled", core.ToBool(cluster.TransitEncryptionEnabled),
						"transitEncryptionMode", string(cluster.TransitEncryptionMode),
					)
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
