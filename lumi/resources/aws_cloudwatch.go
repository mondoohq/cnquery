package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

func (t *lumiAwsCloudwatch) id() (string, error) {
	return "aws.cloudwatch", nil
}

func (t *lumiAwsCloudwatchMetricsalarm) GetSnsTopics() ([]interface{}, error) {
	metricName, err := t.MetricName()
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse metric name")
	}
	namespace, err := t.MetricNamespace()
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse metric namespace")
	}
	regionVal, err := t.Region()
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse metric region")
	}

	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.Cloudwatch(regionVal)
	ctx := context.Background()

	params := &cloudwatch.DescribeAlarmsForMetricInput{
		MetricName: &metricName,
		Namespace:  &namespace,
	}
	// no pagination required
	alarmsResp, err := svc.DescribeAlarmsForMetricRequest(params).Send(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not gather aws cloudwatch alarms")
	}
	lumiActions := []interface{}{}
	for _, alarm := range alarmsResp.MetricAlarms {
		for _, action := range alarm.AlarmActions {
			lumiAlarmAction, err := t.Runtime.CreateResource("aws.sns.topic",
				"arn", action,
				"region", regionVal,
			)
			if err != nil {
				return nil, err
			}
			lumiActions = append(lumiActions, lumiAlarmAction)
		}
	}
	return lumiActions, nil
}

func (t *lumiAwsSnsTopic) GetSubscriptions() ([]interface{}, error) {
	arnValue, err := t.Arn()
	if err != nil {
		return nil, err
	}
	regionVal, err := t.Region()
	if err != nil {
		return nil, err
	}
	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.Sns(regionVal)
	ctx := context.Background()

	lumiSubs := []interface{}{}
	params := &sns.ListSubscriptionsByTopicInput{TopicArn: &arnValue}
	nextToken := aws.String("no_token_to_start_with")
	for nextToken != nil {
		subsByTopic, err := svc.ListSubscriptionsByTopicRequest(params).Send(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not gather sns subscriptions info")
		}
		nextToken = subsByTopic.NextToken
		if subsByTopic.NextToken != nil {
			params.NextToken = nextToken
		}
		for _, sub := range subsByTopic.Subscriptions {
			lumiSub, err := t.Runtime.CreateResource("aws.sns.subscription",
				"arn", toString(sub.SubscriptionArn),
				"protocol", toString(sub.Protocol),
			)
			if err != nil {
				return nil, err
			}
			lumiSubs = append(lumiSubs, lumiSub)
		}
	}
	return lumiSubs, nil
}

func (t *lumiAwsCloudwatch) GetLogGroups() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(t.getLogGroups(), 5)
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

func (t *lumiAwsCloudwatch) getLogGroups() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}}
	}
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.CloudwatchLogs(regionVal)
			ctx := context.Background()

			nextToken := aws.String("no_token_to_start_with")
			params := &cloudwatchlogs.DescribeLogGroupsInput{}
			res := []interface{}{}
			for nextToken != nil {
				logGroups, err := svc.DescribeLogGroupsRequest(params).Send(ctx)
				if err != nil {
					return nil, errors.Wrap(err, "could not gather aws cloudwatch log groups")
				}
				nextToken = logGroups.NextToken
				if logGroups.NextToken != nil {
					params.NextToken = nextToken
				}
				for _, loggroup := range logGroups.LogGroups {
					lumiLogGroup, err := t.Runtime.CreateResource("aws.cloudwatch.loggroup",
						"arn", loggroup.Arn,
						"name", loggroup.LogGroupName,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiLogGroup)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (t *lumiAwsCloudwatchLoggroup) id() (string, error) {
	return t.Arn()
}

func (t *lumiAwsCloudwatchLoggroup) GetMetricsFilters() ([]interface{}, error) {
	arnValue, err := t.Arn()
	if err != nil || len(arnValue) < 6 {
		return nil, errors.Wrap(err, "unable to parse cloud watch log group arn")
	}
	// arn:aws:logs:<region>:<aws_account_number>:log-group:GROUPVAL:*
	logGroupArn := strings.Split(arnValue, ":")
	groupName := logGroupArn[6]
	region := logGroupArn[3]

	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.CloudwatchLogs(region)
	ctx := context.Background()

	nextToken := aws.String("no_token_to_start_with")
	params := &cloudwatchlogs.DescribeMetricFiltersInput{LogGroupName: &groupName}
	metricFilters := []interface{}{}
	for nextToken != nil {
		metricsResp, err := svc.DescribeMetricFiltersRequest(params).Send(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not gather log metric filters")
		}
		nextToken = metricsResp.NextToken
		if metricsResp.NextToken != nil {
			params.NextToken = nextToken
		}
		for _, m := range metricsResp.MetricFilters {
			lumiCloudwatchAlarms := []interface{}{}
			for _, mt := range m.MetricTransformations {
				lumiAwsAlarm, err := t.Runtime.CreateResource("aws.cloudwatch.metricsalarm",
					"id", groupName+"/"+region+"/"+toString(mt.MetricNamespace)+"/"+toString(mt.MetricName),
					"metricName", toString(mt.MetricName),
					"metricNamespace", toString(mt.MetricNamespace),
					"region", region,
				)
				if err != nil {
					return nil, err
				}
				lumiCloudwatchAlarms = append(lumiCloudwatchAlarms, lumiAwsAlarm)
			}
			lumiAwsLogGroupMetricFilters, err := t.Runtime.CreateResource("aws.cloudwatch.loggroup.metricsfilter",
				"id", groupName+"/"+region+"/"+toString(m.FilterName),
				"filterName", toString(m.FilterName),
				"filterPattern", toString(m.FilterPattern),
				"metricAlarms", lumiCloudwatchAlarms,
			)
			if err != nil {
				return nil, err
			}
			metricFilters = append(metricFilters, lumiAwsLogGroupMetricFilters)
		}
	}

	if err != nil {
		return nil, err
	}
	return metricFilters, nil
}

func (t *lumiAwsCloudwatchLoggroupMetricsfilter) id() (string, error) {
	return t.Id()
}

func (t *lumiAwsCloudwatchMetricsalarm) id() (string, error) {
	return t.Id()
}
