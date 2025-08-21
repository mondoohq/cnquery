// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"github.com/aws/aws-sdk-go-v2/service/securityhub/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"
)

func (a *mqlAwsSecurityhub) id() (string, error) {
	return "aws.securityhub", nil
}

func (a *mqlAwsSecurityhub) hubs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getHubs(conn), 5)
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

func (a *mqlAwsSecurityhub) getHubs(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Securityhub(region)
			ctx := context.Background()
			res := []any{}
			secHub, err := svc.DescribeHub(ctx, &securityhub.DescribeHubInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS API")
					return res, nil
				}
				var notFoundErr *types.InvalidAccessException
				if errors.As(err, &notFoundErr) {
					return nil, nil
				}
				return nil, err
			}
			mqlHub, err := CreateResource(a.MqlRuntime, "aws.securityhub.hub",
				map[string]*llx.RawData{
					"arn":          llx.StringDataPtr(secHub.HubArn),
					"subscribedAt": llx.StringDataPtr(secHub.SubscribedAt),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlHub)
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsSecurityhubHub) id() (string, error) {
	return a.Arn.Data, nil
}
