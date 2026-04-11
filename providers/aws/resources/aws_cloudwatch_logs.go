// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsCloudwatch) logDestinations() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getLogDestinations(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}
	return res, nil
}

func (a *mqlAwsCloudwatch) getLogDestinations(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.CloudwatchLogs(region)
			ctx := context.Background()
			res := []any{}

			paginator := cloudwatchlogs.NewDescribeDestinationsPaginator(svc, &cloudwatchlogs.DescribeDestinationsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, errors.Wrap(err, "could not gather cloudwatch log destinations")
				}
				for _, dest := range page.Destinations {
					mqlDest, err := CreateResource(a.MqlRuntime, "aws.cloudwatch.logDestination",
						map[string]*llx.RawData{
							"name":         llx.StringDataPtr(dest.DestinationName),
							"arn":          llx.StringDataPtr(dest.Arn),
							"region":       llx.StringData(region),
							"targetArn":    llx.StringDataPtr(dest.TargetArn),
							"roleArn":      llx.StringDataPtr(dest.RoleArn),
							"accessPolicy": llx.StringDataPtr(dest.AccessPolicy),
							"createdAt":    llx.TimeDataPtr(int64MillisToTime(dest.CreationTime)),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlDest)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsCloudwatchLogDestination) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsCloudwatchLogDestination) iamRole() (*mqlAwsIamRole, error) {
	arn := a.RoleArn.Data
	if arn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlRole, err := NewResource(a.MqlRuntime, ResourceAwsIamRole,
		map[string]*llx.RawData{"arn": llx.StringData(arn)})
	if err != nil {
		return nil, err
	}
	return mqlRole.(*mqlAwsIamRole), nil
}

func (a *mqlAwsCloudwatch) logInsightQueries() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getLogInsightQueries(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}
	return res, nil
}

func (a *mqlAwsCloudwatch) getLogInsightQueries(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.CloudwatchLogs(region)
			ctx := context.Background()
			res := []any{}

			var nextToken *string
			for {
				resp, err := svc.DescribeQueryDefinitions(ctx, &cloudwatchlogs.DescribeQueryDefinitionsInput{
					NextToken: nextToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, errors.Wrap(err, "could not gather cloudwatch log insight queries")
				}

				for _, qd := range resp.QueryDefinitions {
					logGroupNames := make([]any, 0, len(qd.LogGroupNames))
					for _, lgn := range qd.LogGroupNames {
						logGroupNames = append(logGroupNames, lgn)
					}

					mqlQuery, err := CreateResource(a.MqlRuntime, "aws.cloudwatch.logInsightQuery",
						map[string]*llx.RawData{
							"id":            llx.StringDataPtr(qd.QueryDefinitionId),
							"name":          llx.StringDataPtr(qd.Name),
							"region":        llx.StringData(region),
							"queryString":   llx.StringDataPtr(qd.QueryString),
							"logGroupNames": llx.ArrayData(logGroupNames, types.String),
							"modifiedAt":    llx.TimeDataPtr(int64MillisToTime(qd.LastModified)),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlQuery)
				}

				if resp.NextToken == nil {
					break
				}
				nextToken = resp.NextToken
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsCloudwatchLogInsightQuery) id() (string, error) {
	return a.Region.Data + "/" + a.Id.Data, nil
}
