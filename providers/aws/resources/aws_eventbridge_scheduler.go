// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsEventbridge) schedules() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getSchedules(conn), 5)
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

func (a *mqlAwsEventbridge) getSchedules(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("eventbridge>getSchedules>calling aws with region %s", region)

			svc := conn.Scheduler(region)
			ctx := context.Background()
			res := []any{}

			var nextToken *string
			for {
				resp, err := svc.ListSchedules(ctx, &scheduler.ListSchedulesInput{
					NextToken: nextToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("scheduler not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, sched := range resp.Schedules {
					mqlSched, err := CreateResource(a.MqlRuntime, "aws.eventbridge.schedule",
						map[string]*llx.RawData{
							"__id":      llx.StringDataPtr(sched.Arn),
							"arn":       llx.StringDataPtr(sched.Arn),
							"name":      llx.StringDataPtr(sched.Name),
							"region":    llx.StringData(region),
							"groupName": llx.StringDataPtr(sched.GroupName),
							"state":     llx.StringData(string(sched.State)),
							"createdAt": llx.TimeDataPtr(sched.CreationDate),
							"updatedAt": llx.TimeDataPtr(sched.LastModificationDate),
						})
					if err != nil {
						return nil, err
					}
					mqlSchedRes := mqlSched.(*mqlAwsEventbridgeSchedule)
					mqlSchedRes.cacheName = sched.Name
					mqlSchedRes.cacheGroupName = sched.GroupName
					mqlSchedRes.cacheRegion = region
					res = append(res, mqlSchedRes)
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

type mqlAwsEventbridgeScheduleInternal struct {
	cacheName      *string
	cacheGroupName *string
	cacheRegion    string
	cacheRoleArn   *string
	cacheKmsKeyArn *string
	fetched        bool
	lock           sync.Mutex
}

func (a *mqlAwsEventbridgeSchedule) fetchDetails() error {
	if a.fetched {
		return nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return nil
	}

	if a.cacheName == nil {
		a.fetched = true
		return nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Scheduler(a.cacheRegion)
	ctx := context.Background()

	input := &scheduler.GetScheduleInput{
		Name: a.cacheName,
	}
	if a.cacheGroupName != nil {
		input.GroupName = a.cacheGroupName
	}

	resp, err := svc.GetSchedule(ctx, input)
	if err != nil {
		if Is400AccessDeniedError(err) {
			log.Warn().Str("schedule", *a.cacheName).Msg("access denied getting schedule details")
			a.fetched = true
			return nil
		}
		return err
	}

	if resp.ScheduleExpression != nil {
		a.ScheduleExpression = plugin.TValue[string]{Data: *resp.ScheduleExpression, State: plugin.StateIsSet}
	} else {
		a.ScheduleExpression = plugin.TValue[string]{Data: "", State: plugin.StateIsSet}
	}
	if resp.Description != nil {
		a.Description = plugin.TValue[string]{Data: *resp.Description, State: plugin.StateIsSet}
	} else {
		a.Description = plugin.TValue[string]{Data: "", State: plugin.StateIsSet}
	}
	if resp.Target != nil {
		if resp.Target.Arn != nil {
			a.TargetArn = plugin.TValue[string]{Data: *resp.Target.Arn, State: plugin.StateIsSet}
		} else {
			a.TargetArn = plugin.TValue[string]{Data: "", State: plugin.StateIsSet}
		}
		a.cacheRoleArn = resp.Target.RoleArn
	} else {
		a.TargetArn = plugin.TValue[string]{Data: "", State: plugin.StateIsSet}
	}
	a.cacheKmsKeyArn = resp.KmsKeyArn

	a.fetched = true
	return nil
}

func (a *mqlAwsEventbridgeSchedule) scheduleExpression() (string, error) {
	return "", a.fetchDetails()
}

func (a *mqlAwsEventbridgeSchedule) description() (string, error) {
	return "", a.fetchDetails()
}

func (a *mqlAwsEventbridgeSchedule) targetArn() (string, error) {
	return "", a.fetchDetails()
}

func (a *mqlAwsEventbridgeSchedule) iamRole() (*mqlAwsIamRole, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
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

func (a *mqlAwsEventbridgeSchedule) kmsKey() (*mqlAwsKmsKey, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheKmsKeyArn == nil || *a.cacheKmsKeyArn == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheKmsKeyArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsEventbridge) scheduleGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getScheduleGroups(conn), 5)
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

func (a *mqlAwsEventbridge) getScheduleGroups(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("eventbridge>getScheduleGroups>calling aws with region %s", region)

			svc := conn.Scheduler(region)
			ctx := context.Background()
			res := []any{}

			var nextToken *string
			for {
				resp, err := svc.ListScheduleGroups(ctx, &scheduler.ListScheduleGroupsInput{
					NextToken: nextToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("scheduler not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, sg := range resp.ScheduleGroups {
					mqlSg, err := CreateResource(a.MqlRuntime, "aws.eventbridge.scheduleGroup",
						map[string]*llx.RawData{
							"__id":      llx.StringDataPtr(sg.Arn),
							"arn":       llx.StringDataPtr(sg.Arn),
							"name":      llx.StringDataPtr(sg.Name),
							"region":    llx.StringData(region),
							"state":     llx.StringData(string(sg.State)),
							"createdAt": llx.TimeDataPtr(sg.CreationDate),
							"updatedAt": llx.TimeDataPtr(sg.LastModificationDate),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlSg)
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
