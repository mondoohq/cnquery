// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsSfn) id() (string, error) {
	return "aws.sfn", nil
}

func (a *mqlAwsSfn) stateMachines() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getStateMachines(conn), 5)
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

func (a *mqlAwsSfn) getStateMachines(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("sfn>getStateMachines>calling aws with region %s", region)

			svc := conn.Sfn(region)
			ctx := context.Background()
			res := []any{}

			paginator := sfn.NewListStateMachinesPaginator(svc, &sfn.ListStateMachinesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Warn().Str("region", region).Msg("Step Functions is not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, sm := range page.StateMachines {
					mqlSm, err := CreateResource(a.MqlRuntime, "aws.sfn.stateMachine",
						map[string]*llx.RawData{
							"__id":      llx.StringDataPtr(sm.StateMachineArn),
							"arn":       llx.StringDataPtr(sm.StateMachineArn),
							"name":      llx.StringDataPtr(sm.Name),
							"region":    llx.StringData(region),
							"type":      llx.StringData(string(sm.Type)),
							"createdAt": llx.TimeDataPtr(sm.CreationDate),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlSm)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsSfnStateMachine) id() (string, error) {
	return a.Arn.Data, nil
}

type mqlAwsSfnStateMachineInternal struct {
	fetched  bool
	lock     sync.Mutex
	descResp *sfn.DescribeStateMachineOutput
}

func (a *mqlAwsSfnStateMachine) fetchDetail() (*sfn.DescribeStateMachineOutput, error) {
	if a.fetched {
		return a.descResp, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.descResp, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	svc := conn.Sfn(region)
	ctx := context.Background()

	arn := a.Arn.Data
	resp, err := svc.DescribeStateMachine(ctx, &sfn.DescribeStateMachineInput{
		StateMachineArn: &arn,
	})
	if err != nil {
		return nil, err
	}
	a.fetched = true
	a.descResp = resp
	return resp, nil
}

func (a *mqlAwsSfnStateMachine) status() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	return string(resp.Status), nil
}

func (a *mqlAwsSfnStateMachine) definition() (string, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	return convert.ToValue(resp.Definition), nil
}

func (a *mqlAwsSfnStateMachine) iamRole() (*mqlAwsIamRole, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if resp.RoleArn == nil || *resp.RoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlRole, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{
			"arn": llx.StringDataPtr(resp.RoleArn),
		})
	if err != nil {
		return nil, err
	}
	return mqlRole.(*mqlAwsIamRole), nil
}

func (a *mqlAwsSfnStateMachine) loggingConfiguration() (map[string]any, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if resp.LoggingConfiguration == nil {
		return map[string]any{}, nil
	}
	cfg := resp.LoggingConfiguration
	destinations := make([]any, 0, len(cfg.Destinations))
	for _, d := range cfg.Destinations {
		if d.CloudWatchLogsLogGroup != nil {
			destinations = append(destinations, convert.ToValue(d.CloudWatchLogsLogGroup.LogGroupArn))
		}
	}
	return map[string]any{
		"level":                string(cfg.Level),
		"includeExecutionData": cfg.IncludeExecutionData,
		"destinations":         destinations,
	}, nil
}

func (a *mqlAwsSfnStateMachine) tracingEnabled() (bool, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return false, err
	}
	if resp.TracingConfiguration == nil {
		return false, nil
	}
	return resp.TracingConfiguration.Enabled, nil
}

func (a *mqlAwsSfnStateMachine) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	svc := conn.Sfn(region)
	ctx := context.Background()

	arn := a.Arn.Data
	resp, err := svc.ListTagsForResource(ctx, &sfn.ListTagsForResourceInput{
		ResourceArn: &arn,
	})
	if err != nil {
		return nil, err
	}
	return sfnTagsToMap(resp.Tags), nil
}

func (a *mqlAwsSfnStateMachine) encryptionConfiguration() (map[string]any, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if resp.EncryptionConfiguration == nil {
		return map[string]any{}, nil
	}
	cfg := resp.EncryptionConfiguration
	result := map[string]any{
		"type": string(cfg.Type),
	}
	if cfg.KmsKeyId != nil {
		result["kmsKeyId"] = *cfg.KmsKeyId
	}
	if cfg.KmsDataKeyReusePeriodSeconds != nil && *cfg.KmsDataKeyReusePeriodSeconds > 0 {
		result["kmsDataKeyReusePeriodSeconds"] = *cfg.KmsDataKeyReusePeriodSeconds
	}
	return result, nil
}

// Activities

func (a *mqlAwsSfn) activities() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getActivities(conn), 5)
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

func (a *mqlAwsSfn) getActivities(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sfn(region)
			ctx := context.Background()
			res := []any{}

			paginator := sfn.NewListActivitiesPaginator(svc, &sfn.ListActivitiesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						return res, nil
					}
					return nil, err
				}

				for _, activity := range page.Activities {
					mqlActivity, err := CreateResource(a.MqlRuntime, "aws.sfn.activity",
						map[string]*llx.RawData{
							"__id":      llx.StringDataPtr(activity.ActivityArn),
							"arn":       llx.StringDataPtr(activity.ActivityArn),
							"name":      llx.StringDataPtr(activity.Name),
							"region":    llx.StringData(region),
							"createdAt": llx.TimeDataPtr(activity.CreationDate),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlActivity)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsSfnActivity) id() (string, error) {
	return a.Arn.Data, nil
}

func sfnTagsToMap(tags []sfntypes.Tag) map[string]any {
	tagsMap := make(map[string]any)
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			tagsMap[*tag.Key] = *tag.Value
		}
	}
	return tagsMap
}
