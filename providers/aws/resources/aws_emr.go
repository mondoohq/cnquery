// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/emr"
	emrtypes "github.com/aws/aws-sdk-go-v2/service/emr/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"

	"go.mondoo.com/cnquery/v11/types"
)

func (a *mqlAwsEmr) id() (string, error) {
	return "aws.emr", nil
}

func (a *mqlAwsEmr) clusters() ([]interface{}, error) {
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
		res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
	}
	return res, nil
}

func (a *mqlAwsEmrCluster) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEmr) getClusters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Emr(region)
			ctx := context.Background()

			res := []interface{}{}

			params := &emr.ListClustersInput{}
			paginator := emr.NewListClustersPaginator(svc, params)
			for paginator.HasMorePages() {
				clusters, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, cluster := range clusters.Clusters {
					jsonStatus, err := convert.JsonToDict(cluster.Status)
					if err != nil {
						return nil, err
					}
					mqlCluster, err := CreateResource(a.MqlRuntime, "aws.emr.cluster",
						map[string]*llx.RawData{
							"arn":                     llx.StringDataPtr(cluster.ClusterArn),
							"name":                    llx.StringDataPtr(cluster.Name),
							"normalizedInstanceHours": llx.IntDataDefault(cluster.NormalizedInstanceHours, 0),
							"outpostArn":              llx.StringDataPtr(cluster.OutpostArn),
							"status":                  llx.MapData(jsonStatus, types.String),
							"id":                      llx.StringDataPtr(cluster.Id),
						})
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

func (a *mqlAwsEmrCluster) masterInstances() ([]interface{}, error) {
	arn := a.Arn.Data
	id := a.Id.Data
	region, err := GetRegionFromArn(arn)
	if err != nil {
		return nil, err
	}
	res := []emrtypes.Instance{}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Emr(region)
	ctx := context.Background()

	params := &emr.ListInstancesInput{
		ClusterId:          &id,
		InstanceGroupTypes: []emrtypes.InstanceGroupType{"MASTER"},
	}
	paginator := emr.NewListInstancesPaginator(svc, params)
	for paginator.HasMorePages() {
		instances, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		res = append(res, instances.Instances...)
	}
	return convert.JsonToDictSlice(res)
}
