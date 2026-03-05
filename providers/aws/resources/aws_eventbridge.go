// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridge_types "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsEventbridge) id() (string, error) {
	return "aws.eventbridge", nil
}

func (a *mqlAwsEventbridge) eventBuses() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getEventBuses(conn), 5)
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

func (a *mqlAwsEventbridge) getEventBuses(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("eventbridge>getEventBuses>calling aws with region %s", region)

			svc := conn.EventBridge(region)
			ctx := context.Background()
			res := []any{}

			var nextToken *string
			for {
				resp, err := svc.ListEventBuses(ctx, &eventbridge.ListEventBusesInput{
					NextToken: nextToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, bus := range resp.EventBuses {
					mqlBus, err := CreateResource(a.MqlRuntime, "aws.eventbridge.eventBus",
						map[string]*llx.RawData{
							"__id":   llx.StringDataPtr(bus.Arn),
							"arn":    llx.StringDataPtr(bus.Arn),
							"name":   llx.StringDataPtr(bus.Name),
							"region": llx.StringData(region),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlBus)
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

func (a *mqlAwsEventbridge) rules() ([]any, error) {
	buses, err := a.eventBuses()
	if err != nil {
		return nil, err
	}
	res := []any{}
	for _, b := range buses {
		bus := b.(*mqlAwsEventbridgeEventBus)
		rules, err := bus.rules()
		if err != nil {
			return nil, err
		}
		res = append(res, rules...)
	}
	return res, nil
}

func (a *mqlAwsEventbridgeEventBus) tags() (map[string]any, error) {
	arn := a.Arn.Data
	region := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.EventBridge(region)
	ctx := context.Background()

	resp, err := svc.ListTagsForResource(ctx, &eventbridge.ListTagsForResourceInput{
		ResourceARN: &arn,
	})
	if err != nil {
		return nil, err
	}
	tags := make(map[string]any)
	for _, t := range resp.Tags {
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
		}
	}
	return tags, nil
}

func (a *mqlAwsEventbridgeEventBus) rules() ([]any, error) {
	busName := a.Name.Data
	region := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.EventBridge(region)
	ctx := context.Background()

	res := []any{}
	var nextToken *string
	for {
		resp, err := svc.ListRules(ctx, &eventbridge.ListRulesInput{
			EventBusName: &busName,
			NextToken:    nextToken,
		})
		if err != nil {
			if Is400AccessDeniedError(err) {
				return res, nil
			}
			return nil, err
		}

		for _, rule := range resp.Rules {
			mqlRule, err := CreateResource(a.MqlRuntime, "aws.eventbridge.rule",
				map[string]*llx.RawData{
					"__id":               llx.StringDataPtr(rule.Arn),
					"arn":                llx.StringDataPtr(rule.Arn),
					"name":               llx.StringDataPtr(rule.Name),
					"region":             llx.StringData(region),
					"eventBusName":       llx.StringDataPtr(rule.EventBusName),
					"state":              llx.StringData(string(rule.State)),
					"description":        llx.StringDataPtr(rule.Description),
					"eventPattern":       llx.StringDataPtr(rule.EventPattern),
					"scheduleExpression": llx.StringDataPtr(rule.ScheduleExpression),
					"roleArn":            llx.StringDataPtr(rule.RoleArn),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRule)
		}

		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}
	return res, nil
}

func (a *mqlAwsEventbridgeRule) tags() (map[string]any, error) {
	arn := a.Arn.Data
	region := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.EventBridge(region)
	ctx := context.Background()

	resp, err := svc.ListTagsForResource(ctx, &eventbridge.ListTagsForResourceInput{
		ResourceARN: &arn,
	})
	if err != nil {
		return nil, err
	}
	tags := make(map[string]any)
	for _, t := range resp.Tags {
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
		}
	}
	return tags, nil
}

func (a *mqlAwsEventbridgeRule) targets() ([]any, error) {
	ruleName := a.Name.Data
	busName := a.EventBusName.Data
	region := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.EventBridge(region)
	ctx := context.Background()

	var allTargets []eventbridge_types.Target
	var nextToken *string
	for {
		resp, err := svc.ListTargetsByRule(ctx, &eventbridge.ListTargetsByRuleInput{
			Rule:         &ruleName,
			EventBusName: &busName,
			NextToken:    nextToken,
		})
		if err != nil {
			return nil, err
		}
		allTargets = append(allTargets, resp.Targets...)
		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}

	targets, err := convert.JsonToDictSlice(allTargets)
	if err != nil {
		return nil, err
	}
	return targets, nil
}
