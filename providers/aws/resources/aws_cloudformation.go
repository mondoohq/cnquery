// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cf_types "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
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

func cfnTagsToMap(in []cf_types.Tag) map[string]any {
	tags := make(map[string]any, len(in))
	for _, t := range in {
		if t.Key == nil {
			continue
		}
		val := ""
		if t.Value != nil {
			val = *t.Value
		}
		tags[*t.Key] = val
	}
	return tags
}

// stackParameterToDict converts an SDK Parameter to a dict with lowercase
// keys. ResolvedValue and UsePreviousValue are only emitted when present
// since they're context-specific (SSM-resolved params and stack updates).
func stackParameterToDict(p cf_types.Parameter) map[string]any {
	d := map[string]any{
		"key":   aws.ToString(p.ParameterKey),
		"value": aws.ToString(p.ParameterValue),
	}
	if p.ResolvedValue != nil {
		d["resolvedValue"] = *p.ResolvedValue
	}
	if p.UsePreviousValue != nil {
		d["usePreviousValue"] = *p.UsePreviousValue
	}
	return d
}

func stackOutputToDict(o cf_types.Output) map[string]any {
	d := map[string]any{
		"key":   aws.ToString(o.OutputKey),
		"value": aws.ToString(o.OutputValue),
	}
	if o.Description != nil {
		d["description"] = *o.Description
	}
	if o.ExportName != nil {
		d["exportName"] = *o.ExportName
	}
	return d
}

// normalizeDriftStatus maps the empty enum default to NOT_CHECKED so the
// field always carries one of the four documented values.
func normalizeDriftStatus(s string) string {
	if s == "" {
		return "NOT_CHECKED"
	}
	return s
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
					if IsServiceNotAvailableInRegionError(err) {
						log.Warn().Str("region", region).Msg("CloudFormation is not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, stack := range resp.Stacks {
					capabilities := make([]any, len(stack.Capabilities))
					for j, c := range stack.Capabilities {
						capabilities[j] = string(c)
					}

					driftStatus := ""
					if stack.DriftInformation != nil {
						driftStatus = string(stack.DriftInformation.StackDriftStatus)
					}

					mqlStack, err := CreateResource(a.MqlRuntime, "aws.cloudformation.stack",
						map[string]*llx.RawData{
							"__id":                        llx.StringDataPtr(stack.StackId),
							"stackId":                     llx.StringDataPtr(stack.StackId),
							"name":                        llx.StringDataPtr(stack.StackName),
							"region":                      llx.StringData(region),
							"status":                      llx.StringData(string(stack.StackStatus)),
							"statusReason":                llx.StringDataPtr(stack.StackStatusReason),
							"description":                 llx.StringDataPtr(stack.Description),
							"enableTerminationProtection": llx.BoolDataPtr(stack.EnableTerminationProtection),
							"capabilities":                llx.ArrayData(capabilities, types.String),
							"driftStatus":                 llx.StringData(normalizeDriftStatus(driftStatus)),
							"tags":                        llx.MapData(cfnTagsToMap(stack.Tags), types.String),
							"createdAt":                   llx.TimeDataPtr(stack.CreationTime),
							"updatedAt":                   llx.TimeDataPtr(stack.LastUpdatedTime),
						})
					if err != nil {
						return nil, err
					}
					mqlStackRes := mqlStack.(*mqlAwsCloudformationStack)
					mqlStackRes.cacheRoleArn = stack.RoleARN
					mqlStackRes.cacheParameters = stack.Parameters
					mqlStackRes.cacheOutputs = stack.Outputs
					mqlStackRes.cacheNotificationArns = stack.NotificationARNs
					mqlStackRes.cacheRegion = region
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
	cacheRoleArn          *string
	cacheParameters       []cf_types.Parameter
	cacheOutputs          []cf_types.Output
	cacheNotificationArns []string
	cacheRegion           string
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
	res := make([]any, 0, len(a.cacheParameters))
	for _, p := range a.cacheParameters {
		res = append(res, stackParameterToDict(p))
	}
	return res, nil
}

func (a *mqlAwsCloudformationStack) outputs() ([]any, error) {
	res := make([]any, 0, len(a.cacheOutputs))
	for _, o := range a.cacheOutputs {
		res = append(res, stackOutputToDict(o))
	}
	return res, nil
}

func (a *mqlAwsCloudformationStack) notificationTopics() ([]any, error) {
	res := make([]any, 0, len(a.cacheNotificationArns))
	for _, arn := range a.cacheNotificationArns {
		mqlTopic, err := NewResource(a.MqlRuntime, "aws.sns.topic",
			map[string]*llx.RawData{
				"arn":    llx.StringData(arn),
				"region": llx.StringData(a.cacheRegion),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlTopic)
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

// stackSetStatusesToEnumerate are the statuses we sweep. ListStackSets
// defaults to ACTIVE only when Status is unset, so DELETED needs a
// second explicit call.
var stackSetStatusesToEnumerate = []cf_types.StackSetStatus{
	cf_types.StackSetStatusActive,
	cf_types.StackSetStatusDeleted,
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

			for _, status := range stackSetStatusesToEnumerate {
				var nextToken *string
				for {
					resp, err := svc.ListStackSets(ctx, &cloudformation.ListStackSetsInput{
						NextToken: nextToken,
						Status:    status,
					})
					if err != nil {
						if Is400AccessDeniedError(err) {
							log.Warn().Str("region", region).Msg("error accessing region for AWS API")
							return res, nil
						}
						if IsServiceNotAvailableInRegionError(err) {
							log.Warn().Str("region", region).Msg("CloudFormation is not available in region")
							return res, nil
						}
						return nil, err
					}

					for _, ss := range resp.Summaries {
						// Resolve tags inline by calling DescribeStackSet here
						// rather than deferring to a lazy per-resource accessor.
						// Tags are only fetchable via DescribeStackSet, and the
						// previous lazy pattern produced an N-serial-call burst
						// whenever a user queried `stackSets { tags }`. Inlining
						// keeps the calls within the parallel region sweep so
						// each region's tag lookups serialize among themselves
						// but run concurrently across regions.
						tags := map[string]any{}
						if ss.Status == cf_types.StackSetStatusActive && ss.StackSetId != nil {
							dresp, derr := svc.DescribeStackSet(ctx, &cloudformation.DescribeStackSetInput{
								StackSetName: ss.StackSetId,
							})
							if derr != nil {
								if !Is400AccessDeniedError(derr) && !IsServiceNotAvailableInRegionError(derr) {
									return nil, derr
								}
								log.Debug().Str("region", region).Str("stackSet", aws.ToString(ss.StackSetId)).Msg("could not describe stack set for tags")
							} else if dresp.StackSet != nil {
								tags = cfnTagsToMap(dresp.StackSet.Tags)
							}
						}

						mqlSs, err := CreateResource(a.MqlRuntime, "aws.cloudformation.stackSet",
							map[string]*llx.RawData{
								"__id":            llx.StringData(region + "/" + aws.ToString(ss.StackSetId)),
								"stackSetId":      llx.StringDataPtr(ss.StackSetId),
								"name":            llx.StringDataPtr(ss.StackSetName),
								"region":          llx.StringData(region),
								"status":          llx.StringData(string(ss.Status)),
								"description":     llx.StringDataPtr(ss.Description),
								"permissionModel": llx.StringData(string(ss.PermissionModel)),
								"driftStatus":     llx.StringData(normalizeDriftStatus(string(ss.DriftStatus))),
								"tags":            llx.MapData(tags, types.String),
							})
						if err != nil {
							return nil, err
						}
						mqlSsRes := mqlSs.(*mqlAwsCloudformationStackSet)
						mqlSsRes.cacheAutoDeployment = ss.AutoDeployment
						res = append(res, mqlSsRes)
					}

					if resp.NextToken == nil {
						break
					}
					nextToken = resp.NextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsCloudformationStackSetInternal struct {
	cacheAutoDeployment *cf_types.AutoDeployment
}

func (a *mqlAwsCloudformationStackSet) autoDeploymentEnabled() (bool, error) {
	if a.cacheAutoDeployment == nil || a.cacheAutoDeployment.Enabled == nil {
		a.AutoDeploymentEnabled.State = plugin.StateIsNull | plugin.StateIsSet
		return false, nil
	}
	return *a.cacheAutoDeployment.Enabled, nil
}
