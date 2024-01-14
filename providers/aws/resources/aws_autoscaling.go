// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	ec2types "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v10/providers/aws/connection"
	"go.mondoo.com/cnquery/v10/types"
)

func (a *mqlAwsAutoscaling) id() (string, error) {
	return "aws.autoscaling", nil
}

func (a *mqlAwsAutoscalingGroup) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsAutoscaling) groups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getGroups(conn), 5)
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

func (a *mqlAwsAutoscaling) getGroups(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Autoscaling(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &autoscaling.DescribeAutoScalingGroupsInput{}
			for nextToken != nil {
				groups, err := svc.DescribeAutoScalingGroups(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, group := range groups.AutoScalingGroups {
					lbNames := []interface{}{}
					for _, name := range group.LoadBalancerNames {
						lbNames = append(lbNames, name)
					}
					mqlGroup, err := CreateResource(a.MqlRuntime, "aws.autoscaling.group",
						map[string]*llx.RawData{
							"arn":                     llx.StringDataPtr(group.AutoScalingGroupARN),
							"name":                    llx.StringDataPtr(group.AutoScalingGroupName),
							"loadBalancerNames":       llx.ArrayData(lbNames, types.String),
							"healthCheckType":         llx.StringDataPtr(group.HealthCheckType),
							"tags":                    llx.MapData(autoscalingTagsToMap(group.Tags), types.String),
							"region":                  llx.StringData(regionVal),
							"minSize":                 llx.IntData(convert.ToInt64From32(group.MinSize)),
							"maxSize":                 llx.IntData(convert.ToInt64From32(group.MaxSize)),
							"defaultCooldown":         llx.IntData(convert.ToInt64From32(group.DefaultCooldown)),
							"launchConfigurationName": llx.StringDataPtr(group.LaunchConfigurationName),
							"healthCheckGracePeriod":  llx.IntData(convert.ToInt64From32(group.HealthCheckGracePeriod)),
							"createdAt":               llx.TimeDataPtr(group.CreatedTime),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlGroup)
				}
				nextToken = groups.NextToken
				if groups.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func autoscalingTagsToMap(tags []ec2types.TagDescription) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[convert.ToString(tag.Key)] = convert.ToString(tag.Value)
		}
	}

	return tagsMap
}
