// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cf_types "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsCloudformation) id() (string, error) {
	return "aws.cloudformation", nil
}

func (a *mqlAwsCloudformation) stacks() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getStacks(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsCloudformation) getStacks(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("cloudformation>getStacks>calling aws with region %s", region)

			svc := conn.CloudFormation(region)
			ctx := context.Background()
			res := []any{}

			var nextToken *string
			for {
				resp, err := svc.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
					NextToken: nextToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, stack := range resp.Stacks {
					tags := make(map[string]any)
					for _, t := range stack.Tags {
						if t.Key != nil && t.Value != nil {
							tags[*t.Key] = *t.Value
						}
					}

					capabilities := make([]any, len(stack.Capabilities))
					for j, c := range stack.Capabilities {
						capabilities[j] = string(c)
					}

					notificationArns := make([]any, len(stack.NotificationARNs))
					for j, n := range stack.NotificationARNs {
						notificationArns[j] = n
					}

					driftStatus := ""
					if stack.DriftInformation != nil {
						driftStatus = string(stack.DriftInformation.StackDriftStatus)
					}

					mqlStack, err := CreateResource(a.MqlRuntime, "aws.cloudformation.stack",
						map[string]*llx.RawData{
							"__id":                          llx.StringDataPtr(stack.StackId),
							"stackId":                       llx.StringDataPtr(stack.StackId),
							"name":                          llx.StringDataPtr(stack.StackName),
							"region":                        llx.StringData(region),
							"status":                        llx.StringData(string(stack.StackStatus)),
							"statusReason":                  llx.StringDataPtr(stack.StackStatusReason),
							"description":                   llx.StringDataPtr(stack.Description),
							"enableTerminationProtection":   llx.BoolDataPtr(stack.EnableTerminationProtection),
							"capabilities":                  llx.ArrayData(capabilities, types.String),
							"notificationArns":              llx.ArrayData(notificationArns, types.String),
							"driftStatus":                   llx.StringData(driftStatus),
							"tags":                          llx.MapData(tags, types.String),
							"createdAt":                  llx.TimeDataPtr(stack.CreationTime),
							"updatedAt":               llx.TimeDataPtr(stack.LastUpdatedTime),
						})
					if err != nil {
						return nil, err
					}
					mqlStackRes := mqlStack.(*mqlAwsCloudformationStack)
					mqlStackRes.cacheRoleArn = stack.RoleARN
					mqlStackRes.cacheParameters = stack.Parameters
					mqlStackRes.cacheOutputs = stack.Outputs
					res = append(res, mqlStackRes)
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

type mqlAwsCloudformationStackInternal struct {
	cacheRoleArn    *string
	cacheParameters []cf_types.Parameter
	cacheOutputs    []cf_types.Output
}

func (a *mqlAwsCloudformationStack) iamRole() (*mqlAwsIamRole, error) {
	if a.cacheRoleArn == nil || *a.cacheRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsCloudformationStack) parameters() ([]any, error) {
	res, err := convert.JsonToDictSlice(a.cacheParameters)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (a *mqlAwsCloudformationStack) outputs() ([]any, error) {
	res, err := convert.JsonToDictSlice(a.cacheOutputs)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (a *mqlAwsCloudformation) stackSets() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getStackSets(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsCloudformation) getStackSets(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("cloudformation>getStackSets>calling aws with region %s", region)

			svc := conn.CloudFormation(region)
			ctx := context.Background()
			res := []any{}

			var nextToken *string
			for {
				resp, err := svc.ListStackSets(ctx, &cloudformation.ListStackSetsInput{
					NextToken: nextToken,
					Status:    cf_types.StackSetStatusActive,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, ss := range resp.Summaries {
					driftStatus := string(ss.DriftStatus)

					mqlSs, err := CreateResource(a.MqlRuntime, "aws.cloudformation.stackSet",
						map[string]*llx.RawData{
							"__id":            llx.StringDataPtr(ss.StackSetId),
							"stackSetId":      llx.StringDataPtr(ss.StackSetId),
							"name":            llx.StringDataPtr(ss.StackSetName),
							"region":          llx.StringData(region),
							"status":          llx.StringData(string(ss.Status)),
							"description":     llx.StringDataPtr(ss.Description),
							"permissionModel": llx.StringData(string(ss.PermissionModel)),
							"driftStatus":     llx.StringData(driftStatus),
						})
					if err != nil {
						return nil, err
					}
					mqlSsRes := mqlSs.(*mqlAwsCloudformationStackSet)
					mqlSsRes.cacheAutoDeployment = ss.AutoDeployment
					mqlSsRes.cacheStackSetId = ss.StackSetId
					mqlSsRes.cacheRegion = region
					res = append(res, mqlSsRes)
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

type mqlAwsCloudformationStackSetInternal struct {
	cacheAutoDeployment *cf_types.AutoDeployment
	cacheStackSetId     *string
	cacheRegion         string
}

func (a *mqlAwsCloudformationStackSet) autoDeploymentEnabled() (bool, error) {
	if a.cacheAutoDeployment == nil {
		return false, nil
	}
	if a.cacheAutoDeployment.Enabled == nil {
		return false, nil
	}
	return *a.cacheAutoDeployment.Enabled, nil
}

func (a *mqlAwsCloudformationStackSet) tags() (map[string]any, error) {
	if a.cacheStackSetId == nil {
		return map[string]any{}, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.CloudFormation(a.cacheRegion)
	ctx := context.Background()

	resp, err := svc.DescribeStackSet(ctx, &cloudformation.DescribeStackSetInput{
		StackSetName: a.cacheStackSetId,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			return nil, nil
		}
		return nil, err
	}

	tags := make(map[string]any)
	if resp.StackSet != nil {
		for _, t := range resp.StackSet.Tags {
			if t.Key != nil && t.Value != nil {
				tags[*t.Key] = *t.Value
			}
		}
	}
	return tags, nil
}
