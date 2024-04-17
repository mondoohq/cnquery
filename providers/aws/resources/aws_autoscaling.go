// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	ec2types "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
	"go.mondoo.com/cnquery/v11/types"
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
					availabilityZones := []interface{}{}
					for _, zone := range group.AvailabilityZones {
						availabilityZones = append(availabilityZones, zone)
					}
					groupInstances := []interface{}{}
					for _, instance := range group.Instances {
						mqlInstance, err := NewResource(a.MqlRuntime, "aws.ec2.instance",
							map[string]*llx.RawData{
								"arn": llx.StringData(fmt.Sprintf(ec2InstanceArnPattern, regionVal, conn.AccountId(), convert.ToString(instance.InstanceId))),
							})
						if err != nil {
							return nil, err
						}
						groupInstances = append(groupInstances, mqlInstance)
					}

					mqlGroup, err := CreateResource(a.MqlRuntime, "aws.autoscaling.group",
						map[string]*llx.RawData{
							"arn":                     llx.StringDataPtr(group.AutoScalingGroupARN),
							"availabilityZones":       llx.ArrayData(availabilityZones, types.String),
							"capacityRebalance":       llx.BoolDataPtr(group.CapacityRebalance),
							"createdAt":               llx.TimeDataPtr(group.CreatedTime),
							"defaultCooldown":         llx.IntDataDefault(group.DefaultCooldown, 0),
							"defaultInstanceWarmup":   llx.IntDataDefault(group.DefaultInstanceWarmup, 0),
							"desiredCapacity":         llx.IntDataDefault(group.DesiredCapacity, 0),
							"healthCheckGracePeriod":  llx.IntDataDefault(group.HealthCheckGracePeriod, 0),
							"healthCheckType":         llx.StringDataPtr(group.HealthCheckType),
							"instances":               llx.ArrayData(groupInstances, types.Resource("aws.ec2.instance")),
							"launchConfigurationName": llx.StringDataPtr(group.LaunchConfigurationName),
							"loadBalancerNames":       llx.ArrayData(lbNames, types.String),
							"maxInstanceLifetime":     llx.IntDataDefault(group.MaxInstanceLifetime, 0),
							"maxSize":                 llx.IntDataDefault(group.MaxSize, 0),
							"minSize":                 llx.IntDataDefault(group.MinSize, 0),
							"name":                    llx.StringDataPtr(group.AutoScalingGroupName),
							"region":                  llx.StringData(regionVal),
							"tags":                    llx.MapData(autoscalingTagsToMap(group.Tags), types.String),
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
