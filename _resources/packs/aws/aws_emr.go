// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/emr"
	"github.com/aws/aws-sdk-go-v2/service/emr/types"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (e *mqlAwsEmr) id() (string, error) {
	return "aws.emr", nil
}

func (e *mqlAwsEmr) GetClusters() ([]interface{}, error) {
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
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
	}
	return res, nil
}

func (e *mqlAwsEmrCluster) id() (string, error) {
	return e.Arn()
}

func (e *mqlAwsEmr) getClusters(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Emr(regionVal)
			ctx := context.Background()

			res := []interface{}{}

			var marker *string
			for {
				clusters, err := svc.ListClusters(ctx, &emr.ListClustersInput{Marker: marker})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, cluster := range clusters.Clusters {
					jsonStatus, err := core.JsonToDict(cluster.Status)
					if err != nil {
						return nil, err
					}
					mqlCluster, err := e.MotorRuntime.CreateResource("aws.emr.cluster",
						"arn", core.ToString(cluster.ClusterArn),
						"name", core.ToString(cluster.Name),
						"normalizedInstanceHours", core.ToInt64From32(cluster.NormalizedInstanceHours),
						"outpostArn", core.ToString(cluster.OutpostArn),
						"status", jsonStatus,
						"id", core.ToString(cluster.Id),
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

func (e *mqlAwsEmrCluster) GetMasterInstances() ([]interface{}, error) {
	arn, err := e.Arn()
	if err != nil {
		return nil, err
	}
	id, err := e.Id()
	if err != nil {
		return nil, err
	}
	region, err := GetRegionFromArn(arn)
	if err != nil {
		return nil, err
	}
	res := []types.Instance{}
	at, err := awsProvider(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := at.Emr(region)
	ctx := context.Background()
	var marker *string
	for {
		instances, err := svc.ListInstances(ctx, &emr.ListInstancesInput{
			Marker:             marker,
			ClusterId:          &id,
			InstanceGroupTypes: []types.InstanceGroupType{"MASTER"},
		})
		if err != nil {
			return nil, err
		}
		res = append(res, instances.Instances...)
		if instances.Marker == nil {
			break
		}
		marker = instances.Marker
	}
	return core.JsonToDictSlice(res)
}
