// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/databasemigrationservice"
	"github.com/aws/aws-sdk-go-v2/service/databasemigrationservice/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v10/providers/aws/connection"
)

func (a *mqlAwsDms) id() (string, error) {
	return "aws.dms", nil
}

func (a *mqlAwsDms) replicationInstances() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []types.ReplicationInstance{}
	poolOfJobs := jobpool.CreatePool(a.getReplicationInstances(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]types.ReplicationInstance)...)
	}
	return convert.JsonToDictSlice(res)
}

func (a *mqlAwsDms) getReplicationInstances(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("dms>getReplicationInstances>calling aws with region %s", regionVal)

			svc := conn.Dms(regionVal)
			ctx := context.Background()
			replicationInstancesAggregated := []types.ReplicationInstance{}

			var marker *string
			for {
				replicationInstances, err := svc.DescribeReplicationInstances(ctx, &databasemigrationservice.DescribeReplicationInstancesInput{Marker: marker})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return tasks, nil
					}
					return nil, err
				}
				replicationInstancesAggregated = append(replicationInstancesAggregated, replicationInstances.ReplicationInstances...)

				if replicationInstances.Marker == nil {
					break
				}
				marker = replicationInstances.Marker
			}
			return jobpool.JobResult(replicationInstancesAggregated), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}
