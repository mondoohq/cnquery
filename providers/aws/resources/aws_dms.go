// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/databasemigrationservice"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsDms) id() (string, error) {
	return "aws.dms", nil
}

func (a *mqlAwsDms) replicationInstances() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getReplicationInstances(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	var errs []error
	// get all the results
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Err != nil {
			errs = append(errs, poolOfJobs.Jobs[i].Err)
		}
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	converted, err := convert.JsonToDictSlice(res)
	if err != nil {
		return nil, err
	}
	return converted, errors.Join(errs...)
}

func (a *mqlAwsDms) getReplicationInstances(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("dms>getReplicationInstances>calling aws with region %s", region)

			svc := conn.Dms(region)
			ctx := context.Background()
			res := []any{}

			params := &databasemigrationservice.DescribeReplicationInstancesInput{}
			paginator := databasemigrationservice.NewDescribeReplicationInstancesPaginator(svc, params)
			for paginator.HasMorePages() {
				replicationInstances, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return nil, nil
					}
					return nil, err
				}

				mqlRep, err := convert.JsonToDictSlice(replicationInstances.ReplicationInstances)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlRep...)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}
