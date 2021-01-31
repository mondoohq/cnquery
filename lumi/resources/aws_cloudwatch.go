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
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

const (
	cloudwatchAlarmArnPattern = "arn:aws:cloudwatch:%s:%s:metricalarm/%s/%s"
)

func (t *lumiAwsCloudwatch) id() (string, error) {
	return "aws.cloudwatch", nil
}
func (t *lumiAwsCloudwatch) GetMetrics() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(t.getMetrics(), 5)
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
func (t *lumiAwsCloudwatch) getMetrics() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {

			svc := at.Cloudwatch(regionVal)
			ctx := context.Background()

			res := []interface{}{}
			nextToken := aws.String("no_token_to_start_with")
			params := &cloudwatch.ListMetricsInput{}
			for nextToken != nil {
				metrics, err := svc.ListMetrics(ctx, params)
				if err != nil {
					return nil, err
				}
				for _, metric := range metrics.Metrics {
					lumiMetric, err := t.Runtime.CreateResource("aws.cloudwatch.metric",
						"name", toString(metric.MetricName),
						"namespace", toString(metric.Namespace),
						"region", regionVal,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiMetric)
				}
				nextToken = metrics.NextToken
				if metrics.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}
func (t *lumiAwsCloudwatchMetric) GetAlarms() ([]interface{}, error) {
	metricName, err := t.Name()
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse metric name")
	}
	namespace, err := t.Namespace()
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
	alarmsResp, err := svc.DescribeAlarmsForMetric(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "could not gather aws cloudwatch alarms")
	}
	res := []interface{}{}
	for _, alarm := range alarmsResp.MetricAlarms {
		lumiAlarm, err := t.Runtime.CreateResource("aws.cloudwatch.metricsalarm",
			"arn", toString(alarm.AlarmArn),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAlarm)
	}
	return res, nil
}

func (t *lumiAwsCloudwatch) GetAlarms() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(t.getAlarms(), 5)
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
func (t *lumiAwsCloudwatch) getAlarms() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {

			svc := at.Cloudwatch(regionVal)
			ctx := context.Background()

			res := []interface{}{}
			nextToken := aws.String("no_token_to_start_with")
			params := &cloudwatch.DescribeAlarmsInput{}
			for nextToken != nil {

				alarms, err := svc.DescribeAlarms(ctx, params)
				if err != nil {
					return nil, err
				}

				for _, alarm := range alarms.MetricAlarms {
					actions := []interface{}{}
					for _, action := range alarm.AlarmActions {
						lumiAlarmAction, err := t.Runtime.CreateResource("aws.sns.topic",
							"arn", action,
							"region", regionVal,
						)
						if err != nil {
							return nil, err
						}
						actions = append(actions, lumiAlarmAction)
					}
					insuffActions := []interface{}{}
					for _, action := range alarm.InsufficientDataActions {
						lumiInsuffAction, err := t.Runtime.CreateResource("aws.sns.topic",
							"arn", action,
							"region", regionVal,
						)
						if err != nil {
							return nil, err
						}
						insuffActions = append(insuffActions, lumiInsuffAction)
					}

					okActions := []interface{}{}
					for _, action := range alarm.OKActions {
						lumiokAction, err := t.Runtime.CreateResource("aws.sns.topic",
							"arn", action,
							"region", regionVal,
						)
						if err != nil {
							return nil, err
						}
						okActions = append(okActions, lumiokAction)
					}

					lumiAlarm, err := t.Runtime.CreateResource("aws.cloudwatch.metricsalarm",
						"arn", toString(alarm.AlarmArn),
						"metricName", toString(alarm.MetricName),
						"metricNamespace", toString(alarm.Namespace),
						"region", regionVal,
						"state", string(alarm.StateValue),
						"stateReason", toString(alarm.StateReason),
						"insufficientDataActions", insuffActions,
						"okActions", okActions,
						"name", toString(alarm.AlarmName),
						"actions", actions,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiAlarm)
				}
				nextToken = alarms.NextToken
				if alarms.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
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
		subsByTopic, err := svc.ListSubscriptionsByTopic(ctx, params)
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
		return []*jobpool.Job{{Err: err}}
	}
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
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
				logGroups, err := svc.DescribeLogGroups(ctx, params)
				if err != nil {
					return nil, errors.Wrap(err, "could not gather aws cloudwatch log groups")
				}
				nextToken = logGroups.NextToken
				if logGroups.NextToken != nil {
					params.NextToken = nextToken
				}
				for _, loggroup := range logGroups.LogGroups {
					args := []interface{}{
						"arn", toString(loggroup.Arn),
						"name", toString(loggroup.LogGroupName),
					}
					// add kms key if there is one
					if loggroup.KmsKeyId != nil {
						lumiKeyResource, err := t.Runtime.CreateResource("aws.kms.key",
							"arn", toString(loggroup.KmsKeyId),
						)
						if err != nil {
							return nil, err
						}
						lumiKey := lumiKeyResource.(AwsKmsKey)
						args = append(args, "kmsKey", lumiKey)
					}

					lumiLogGroup, err := t.Runtime.CreateResource("aws.cloudwatch.loggroup", args...)
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

func (c *lumiAwsCloudwatchLoggroup) init(args *lumi.Args) (*lumi.Args, AwsCloudwatchLoggroup, error) {
	if len(*args) > 1 {
		return args, nil, nil
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch cloudwatch log group")
	}

	obj, err := c.Runtime.CreateResource("aws.cloudwatch")
	if err != nil {
		return nil, nil, err
	}
	cloudwatch := obj.(AwsCloudwatch)

	rawResources, err := cloudwatch.LogGroups()
	if err != nil {
		return nil, nil, err
	}

	arnVal := (*args)["arn"].(string)
	for i := range rawResources {
		loggroup := rawResources[i].(AwsCloudwatchLoggroup)
		lumiLgArn, err := loggroup.Arn()
		if err != nil {
			return nil, nil, errors.New("cloudwatch log group does not exist")
		}
		if lumiLgArn == arnVal {
			return args, loggroup, nil
		}
	}
	return nil, nil, errors.New("cloudwatch log group does not exist")
}

func (s *lumiAwsCloudwatchLoggroup) GetKmsKey() (interface{}, error) {
	// no key id on the log group object
	return nil, nil
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
		metricsResp, err := svc.DescribeMetricFilters(ctx, params)
		if err != nil {
			return nil, errors.Wrap(err, "could not gather log metric filters")
		}
		nextToken = metricsResp.NextToken
		if metricsResp.NextToken != nil {
			params.NextToken = nextToken
		}
		for _, m := range metricsResp.MetricFilters {
			lumiCloudwatchMetrics := []interface{}{}
			for _, mt := range m.MetricTransformations {
				lumiAwsMetric, err := t.Runtime.CreateResource("aws.cloudwatch.metric",
					"name", toString(mt.MetricName),
					"namespace", toString(mt.MetricNamespace),
					"region", region,
				)
				if err != nil {
					return nil, err
				}
				lumiCloudwatchMetrics = append(lumiCloudwatchMetrics, lumiAwsMetric)
			}
			lumiAwsLogGroupMetricFilters, err := t.Runtime.CreateResource("aws.cloudwatch.loggroup.metricsfilter",
				"id", groupName+"/"+region+"/"+toString(m.FilterName),
				"filterName", toString(m.FilterName),
				"filterPattern", toString(m.FilterPattern),
				"metrics", lumiCloudwatchMetrics,
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
	return t.Arn()
}

func (t *lumiAwsCloudwatchMetric) id() (string, error) {
	region, err := t.Region()
	if err != nil {
		return "", err
	}
	name, err := t.Name()
	if err != nil {
		return "", err
	}
	namespace, err := t.Namespace()
	if err != nil {
		return "", err
	}

	return region + "/" + namespace + "/" + name, nil
}

func (t *lumiAwsCloudwatchMetricsalarm) init(args *lumi.Args) (*lumi.Args, AwsCloudwatchMetricsalarm, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch aws cloudwatch metrics alarm")
	}

	// load all cloudwatch metrics alarm
	obj, err := t.Runtime.CreateResource("aws.cloudwatch")
	if err != nil {
		return nil, nil, err
	}
	aws := obj.(AwsCloudwatch)

	rawResources, err := aws.Alarms()
	if err != nil {
		return nil, nil, err
	}

	arnVal := (*args)["arn"].(string)
	for i := range rawResources {
		alarm := rawResources[i].(AwsCloudwatchMetricsalarm)
		lumiAlarmArn, err := alarm.Arn()
		if err != nil {
			return nil, nil, errors.New("cloudwatch alarm does not exist")
		}
		if lumiAlarmArn == arnVal {
			return args, alarm, nil
		}
	}
	return nil, nil, errors.New("cloudwatch alarm does not exist")
}
