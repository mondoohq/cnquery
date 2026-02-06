// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	ec2types "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"
	"go.mondoo.com/cnquery/v12/types"
)

func (a *mqlAwsAutoscaling) id() (string, error) {
	return "aws.autoscaling", nil
}

func (a *mqlAwsAutoscalingGroup) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsAutoscaling) groups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getGroups(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsAutoscalingGroup) instances() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	groupInstances := []any{}
	for _, instance := range a.groupInstances {
		mqlInstance, err := NewResource(a.MqlRuntime, "aws.ec2.instance",
			map[string]*llx.RawData{
				"arn": llx.StringData(fmt.Sprintf(ec2InstanceArnPattern, a.region, conn.AccountId(), convert.ToValue(instance.InstanceId))),
			})
		if err != nil {
			return nil, err
		}
		groupInstances = append(groupInstances, mqlInstance)
	}
	return groupInstances, nil
}

func (a *mqlAwsAutoscalingGroup) targetGroups() ([]any, error) {
	res := []any{}
	for _, tgArn := range a.targetGroupArns {
		mqlTg, err := NewResource(a.MqlRuntime, "aws.elb.targetgroup",
			map[string]*llx.RawData{
				"arn": llx.StringData(tgArn),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlTg)
	}
	return res, nil
}

type mqlAwsAutoscalingGroupInternal struct {
	groupInstances  []ec2types.Instance
	targetGroupArns []string
	region          string
}

func initAwsAutoscalingGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	if args["region"] == nil || args["name"] == nil {
		return nil, nil, errors.New("region and name required to fetch aws autoscaling group")
	}
	region := args["region"].Value.(string)
	name := args["name"].Value.(string)
	conn := runtime.Connection.(*connection.AwsConnection)
	svc := conn.Autoscaling(region)
	ctx := context.Background()
	ags, err := svc.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: []string{name}})
	if err != nil {
		return nil, nil, err
	}

	if len(ags.AutoScalingGroups) == 1 {
		group := ags.AutoScalingGroups[0]
		lbNames := []any{}
		for _, name := range group.LoadBalancerNames {
			lbNames = append(lbNames, name)
		}
		availabilityZones := []any{}
		for _, zone := range group.AvailabilityZones {
			availabilityZones = append(availabilityZones, zone)
		}

		groupArn := convert.ToValue(group.AutoScalingGroupARN)
		tagSpecs, err := createTagSpecifications(runtime, group.Tags, groupArn)
		if err != nil {
			return nil, nil, err
		}

		args["arn"] = llx.StringData(groupArn)
		args["availabilityZones"] = llx.ArrayData(availabilityZones, types.String)
		args["capacityRebalance"] = llx.BoolDataPtr(group.CapacityRebalance)
		args["createdAt"] = llx.TimeDataPtr(group.CreatedTime)
		args["defaultCooldown"] = llx.IntDataDefault(group.DefaultCooldown, 0)
		args["defaultInstanceWarmup"] = llx.IntDataDefault(group.DefaultInstanceWarmup, 0)
		args["desiredCapacity"] = llx.IntDataDefault(group.DesiredCapacity, 0)
		args["healthCheckGracePeriod"] = llx.IntDataDefault(group.HealthCheckGracePeriod, 0)
		args["healthCheckType"] = llx.StringDataPtr(group.HealthCheckType)
		args["launchConfigurationName"] = llx.StringDataPtr(group.LaunchConfigurationName)
		args["loadBalancerNames"] = llx.ArrayData(lbNames, types.String)
		args["maxInstanceLifetime"] = llx.IntDataDefault(group.MaxInstanceLifetime, 0)
		args["maxSize"] = llx.IntDataDefault(group.MaxSize, 0)
		args["minSize"] = llx.IntDataDefault(group.MinSize, 0)
		args["name"] = llx.StringDataPtr(group.AutoScalingGroupName)
		args["region"] = llx.StringData(region)
		args["tags"] = llx.MapData(autoscalingTagsToMap(group.Tags), types.String)
		args["tagSpecifications"] = llx.ArrayData(tagSpecs, types.Resource(ResourceAwsAutoscalingGroupTag))
		args["desiredCapacityType"] = llx.StringDataPtr(group.DesiredCapacityType)
		args["warmPoolSize"] = llx.IntDataDefault(group.WarmPoolSize, 0)
		args["predictedCapacity"] = llx.IntDataDefault(group.PredictedCapacity, 0)
		args["placementGroup"] = llx.StringDataPtr(group.PlacementGroup)
		args["newInstancesProtectedFromScaleIn"] = llx.BoolDataPtr(group.NewInstancesProtectedFromScaleIn)
		mqlGroup, err := CreateResource(runtime, ResourceAwsAutoscalingGroup, args)
		if err != nil {
			return args, nil, err
		}
		mqlGroup.(*mqlAwsAutoscalingGroup).groupInstances = group.Instances
		mqlGroup.(*mqlAwsAutoscalingGroup).targetGroupArns = group.TargetGroupARNs
		mqlGroup.(*mqlAwsAutoscalingGroup).region = region
		return args, mqlGroup, nil
	}
	return args, nil, nil
}

func (a *mqlAwsAutoscaling) getGroups(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Autoscaling(region)
			ctx := context.Background()
			res := []any{}

			params := &autoscaling.DescribeAutoScalingGroupsInput{}
			paginator := autoscaling.NewDescribeAutoScalingGroupsPaginator(svc, params)
			for paginator.HasMorePages() {
				groups, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, group := range groups.AutoScalingGroups {
					lbNames := []any{}
					for _, name := range group.LoadBalancerNames {
						lbNames = append(lbNames, name)
					}
					availabilityZones := []any{}
					for _, zone := range group.AvailabilityZones {
						availabilityZones = append(availabilityZones, zone)
					}

					groupArn := convert.ToValue(group.AutoScalingGroupARN)
					tagSpecs, err := createTagSpecifications(a.MqlRuntime, group.Tags, groupArn)
					if err != nil {
						return nil, err
					}

					mqlGroup, err := CreateResource(a.MqlRuntime, ResourceAwsAutoscalingGroup,
						map[string]*llx.RawData{
							"arn":                              llx.StringData(groupArn),
							"availabilityZones":                llx.ArrayData(availabilityZones, types.String),
							"capacityRebalance":                llx.BoolDataPtr(group.CapacityRebalance),
							"createdAt":                        llx.TimeDataPtr(group.CreatedTime),
							"defaultCooldown":                  llx.IntDataDefault(group.DefaultCooldown, 0),
							"defaultInstanceWarmup":            llx.IntDataDefault(group.DefaultInstanceWarmup, 0),
							"desiredCapacity":                  llx.IntDataDefault(group.DesiredCapacity, 0),
							"healthCheckGracePeriod":           llx.IntDataDefault(group.HealthCheckGracePeriod, 0),
							"healthCheckType":                  llx.StringDataPtr(group.HealthCheckType),
							"launchConfigurationName":          llx.StringDataPtr(group.LaunchConfigurationName),
							"loadBalancerNames":                llx.ArrayData(lbNames, types.String),
							"maxInstanceLifetime":              llx.IntDataDefault(group.MaxInstanceLifetime, 0),
							"maxSize":                          llx.IntDataDefault(group.MaxSize, 0),
							"minSize":                          llx.IntDataDefault(group.MinSize, 0),
							"name":                             llx.StringDataPtr(group.AutoScalingGroupName),
							"region":                           llx.StringData(region),
							"tags":                             llx.MapData(autoscalingTagsToMap(group.Tags), types.String),
							"tagSpecifications":                llx.ArrayData(tagSpecs, types.Resource(ResourceAwsAutoscalingGroupTag)),
							"desiredCapacityType":              llx.StringDataPtr(group.DesiredCapacityType),
							"warmPoolSize":                     llx.IntDataDefault(group.WarmPoolSize, 0),
							"predictedCapacity":                llx.IntDataDefault(group.PredictedCapacity, 0),
							"placementGroup":                   llx.StringDataPtr(group.PlacementGroup),
							"newInstancesProtectedFromScaleIn": llx.BoolDataPtr(group.NewInstancesProtectedFromScaleIn),
						})
					if err != nil {
						return nil, err
					}
					mqlGroup.(*mqlAwsAutoscalingGroup).groupInstances = group.Instances
					mqlGroup.(*mqlAwsAutoscalingGroup).targetGroupArns = group.TargetGroupARNs
					mqlGroup.(*mqlAwsAutoscalingGroup).region = region
					res = append(res, mqlGroup)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func autoscalingTagsToMap(tags []ec2types.TagDescription) map[string]any {
	tagsMap := make(map[string]any)

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
		}
	}

	return tagsMap
}

func createTagSpecifications(runtime *plugin.Runtime, tags []ec2types.TagDescription, groupArn string) ([]any, error) {
	tagSpecs := make([]any, 0, len(tags))

	for _, tag := range tags {
		key := convert.ToValue(tag.Key)
		tagId := fmt.Sprintf("%s/tag/%s", groupArn, key)

		mqlTag, err := CreateResource(runtime, ResourceAwsAutoscalingGroupTag,
			map[string]*llx.RawData{
				"__id":              llx.StringData(tagId),
				"key":               llx.StringData(key),
				"value":             llx.StringData(convert.ToValue(tag.Value)),
				"propagateAtLaunch": llx.BoolDataPtr(tag.PropagateAtLaunch),
				"resourceId":        llx.StringData(convert.ToValue(tag.ResourceId)),
				"resourceType":      llx.StringData(convert.ToValue(tag.ResourceType)),
			})
		if err != nil {
			return nil, err
		}
		tagSpecs = append(tagSpecs, mqlTag)
	}

	return tagSpecs, nil
}
