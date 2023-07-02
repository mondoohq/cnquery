package aws

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

const (
	cloudwatchAlarmArnPattern = "arn:aws:cloudwatch:%s:%s:metricalarm/%s/%s"
)

func (t *mqlAwsCloudwatch) id() (string, error) {
	return "aws.cloudwatch", nil
}

func (t *mqlAwsCloudwatch) GetMetrics() ([]interface{}, error) {
	provider, err := awsProvider(t.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(t.getMetrics(provider), 5)
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

func (t *mqlAwsCloudwatch) getMetrics(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Cloudwatch(regionVal)
			ctx := context.Background()

			res := []interface{}{}
			nextToken := aws.String("no_token_to_start_with")
			params := &cloudwatch.ListMetricsInput{}
			for nextToken != nil {
				metrics, err := svc.ListMetrics(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, metric := range metrics.Metrics {
					dimensions := []interface{}{}
					for _, d := range metric.Dimensions {
						mqlDimension, err := t.MotorRuntime.CreateResource("aws.cloudwatch.metricdimension",
							"name", core.ToString(d.Name),
							"value", core.ToString(d.Value),
						)
						if err != nil {
							return nil, err
						}
						dimensions = append(dimensions, mqlDimension)
					}

					mqlMetric, err := t.MotorRuntime.CreateResource("aws.cloudwatch.metric",
						"name", core.ToString(metric.MetricName),
						"namespace", core.ToString(metric.Namespace),
						"region", regionVal,
						"dimensions", dimensions,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlMetric)
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

func (t *mqlAwsCloudwatchMetricdimension) id() (string, error) {
	name, err := t.Name()
	if err != nil {
		return "", err
	}
	val, err := t.Value()
	if err != nil {
		return "", err
	}
	return name + "/" + val, nil
}

func (t *mqlAwsCloudwatchMetricstatistics) id() (string, error) {
	region, err := t.Region()
	if err != nil {
		return "", err
	}
	namespace, err := t.Namespace()
	if err != nil {
		return "", err
	}
	name, err := t.Name()
	if err != nil {
		return "", err
	}
	label, err := t.Label()
	if err != nil {
		return "", err
	}
	return namespace + "/" + name + "/" + region + "/" + label, nil
}

// allow the user to query for a specific namespace metric in a specific region
func (p *mqlAwsCloudwatchMetric) init(args *resources.Args) (*resources.Args, AwsCloudwatchMetric, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	namespaceRaw := (*args)["namespace"]
	if namespaceRaw == nil {
		return args, nil, nil
	}

	namespace, ok := namespaceRaw.(string)
	if !ok {
		return args, nil, nil
	}

	nameRaw := (*args)["name"]
	if nameRaw == nil {
		return args, nil, nil
	}

	name, ok := nameRaw.(string)
	if !ok {
		return args, nil, nil
	}

	regionRaw := (*args)["region"]
	if namespaceRaw == nil {
		return args, nil, nil
	}

	region, ok := regionRaw.(string)
	if !ok {
		return args, nil, nil
	}
	at, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return args, nil, err
	}
	svc := at.Cloudwatch(region)

	ctx := context.Background()

	params := &cloudwatch.ListMetricsInput{
		Namespace:  &namespace,
		MetricName: &name,
	}
	metrics, err := svc.ListMetrics(ctx, params)
	if err != nil {
		return args, nil, err
	}
	if len(metrics.Metrics) == 0 {
		return nil, nil, nil
	}
	if len(metrics.Metrics) > 1 {
		return nil, nil, errors.New("more than one metric found for " + namespace + " " + name + " in region " + region)
	}
	dimensions := []interface{}{}

	metric := metrics.Metrics[0]
	for _, d := range metric.Dimensions {
		mqlDimension, err := p.MotorRuntime.CreateResource("aws.cloudwatch.metricdimension",
			"name", core.ToString(d.Name),
			"value", core.ToString(d.Value),
		)
		if err != nil {
			return args, nil, err
		}
		dimensions = append(dimensions, mqlDimension)
	}

	(*args)["name"] = name
	(*args)["namespace"] = namespace
	(*args)["region"] = region
	(*args)["dimensions"] = dimensions

	return args, nil, nil
}

func (p *mqlAwsCloudwatchMetric) GetDimensions() ([]interface{}, error) {
	name, err := p.Name()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse metric name"))
	}
	namespace, err := p.Namespace()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse metric namespace"))
	}
	regionVal, err := p.Region()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse metric region"))
	}

	at, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := at.Cloudwatch(regionVal)
	ctx := context.Background()

	params := &cloudwatch.ListMetricsInput{
		Namespace:  &namespace,
		MetricName: &name,
	}
	metrics, err := svc.ListMetrics(ctx, params)
	if err != nil {
		return nil, err
	}
	if len(metrics.Metrics) == 0 {
		return nil, nil
	}
	if len(metrics.Metrics) > 1 {
		return nil, errors.New("more than one metric found for " + namespace + " " + name + " in region " + regionVal)
	}
	dimensions := []interface{}{}

	metric := metrics.Metrics[0]
	for _, d := range metric.Dimensions {
		mqlDimension, err := p.MotorRuntime.CreateResource("aws.cloudwatch.metricdimension",
			"name", core.ToString(d.Name),
			"value", core.ToString(d.Value),
		)
		if err != nil {
			return nil, err
		}
		dimensions = append(dimensions, mqlDimension)
	}
	return dimensions, nil
}

// allow the user to query for a specific namespace metric in a specific region
func (p *mqlAwsCloudwatchMetricstatistics) init(args *resources.Args) (*resources.Args, AwsCloudwatchMetricstatistics, error) {
	if len(*args) > 3 {
		return args, nil, nil
	}

	namespaceRaw := (*args)["namespace"]
	if namespaceRaw == nil {
		return args, nil, nil
	}

	namespace, ok := namespaceRaw.(string)
	if !ok {
		return args, nil, nil
	}

	nameRaw := (*args)["name"]
	if nameRaw == nil {
		return args, nil, nil
	}

	name, ok := nameRaw.(string)
	if !ok {
		return args, nil, nil
	}

	regionRaw := (*args)["region"]
	if namespaceRaw == nil {
		return args, nil, nil
	}

	region, ok := regionRaw.(string)
	if !ok {
		return args, nil, nil
	}
	at, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return args, nil, err
	}
	svc := at.Cloudwatch(region)
	ctx := context.Background()

	now := time.Now()
	dayAgo := time.Now().Add(-24 * time.Hour)
	params := &cloudwatch.GetMetricStatisticsInput{
		MetricName: &name,
		Namespace:  &namespace,
		StartTime:  &dayAgo,
		EndTime:    &now,
		Period:     aws.Int32(3600),
		Statistics: []types.Statistic{types.StatisticSum, types.StatisticAverage, types.StatisticMaximum, types.StatisticMinimum},
	}
	// no pagination required
	statsResp, err := svc.GetMetricStatistics(ctx, params)
	if err != nil {
		return args, nil, err
	}
	datapoints := []interface{}{}
	for _, datapoint := range statsResp.Datapoints {
		mqlDatapoint, err := p.MotorRuntime.CreateResource("aws.cloudwatch.metric.datapoint",
			"timestamp", datapoint.Timestamp,
			"maximum", core.ToFloat64(datapoint.Maximum),
			"minimum", core.ToFloat64(datapoint.Minimum),
			"average", core.ToFloat64(datapoint.Average),
			"sum", core.ToFloat64(datapoint.Sum),
			"unit", string(datapoint.Unit),
		)
		if err != nil {
			return args, nil, err
		}
		datapoints = append(datapoints, mqlDatapoint)
	}

	if err != nil {
		return args, nil, err
	}

	(*args)["label"] = core.ToString(statsResp.Label)
	(*args)["datapoints"] = datapoints
	(*args)["name"] = name
	(*args)["namespace"] = namespace
	(*args)["region"] = region
	return args, nil, nil
}

func (t *mqlAwsCloudwatchMetric) GetStatistics() (interface{}, error) {
	metricName, err := t.Name()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse metric name"))
	}
	namespace, err := t.Namespace()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse metric namespace"))
	}
	dimensions, err := t.Dimensions()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse metric dimensions"))
	}
	regionVal, err := t.Region()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse metric region"))
	}

	at, err := awsProvider(t.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := at.Cloudwatch(regionVal)
	ctx := context.Background()

	now := time.Now()
	dayAgo := time.Now().Add(-24 * time.Hour)
	typedDimensions := make([]types.Dimension, len(dimensions))
	for i, d := range dimensions {
		dimension := d.(*mqlAwsCloudwatchMetricdimension)
		name, err := dimension.Name()
		if err != nil {
			return nil, errors.Join(err, errors.New("unable to parse metric dimension name"))
		}
		val, err := dimension.Value()
		if err != nil {
			return nil, errors.Join(err, errors.New("unable to parse metric dimension value"))
		}
		typedDimensions[i].Name = &name
		typedDimensions[i].Value = &val
	}
	params := &cloudwatch.GetMetricStatisticsInput{
		MetricName: &metricName,
		Namespace:  &namespace,
		Dimensions: typedDimensions,
		StartTime:  &dayAgo,
		EndTime:    &now,
		Period:     aws.Int32(3600),
		Statistics: []types.Statistic{types.StatisticSum, types.StatisticAverage, types.StatisticMaximum, types.StatisticMinimum},
	}
	// no pagination required
	statsResp, err := svc.GetMetricStatistics(ctx, params)
	if err != nil {
		return nil, errors.Join(err, errors.New("could not gather aws cloudwatch stats"))
	}
	datapoints := []interface{}{}
	for _, datapoint := range statsResp.Datapoints {
		mqlDatapoint, err := t.MotorRuntime.CreateResource("aws.cloudwatch.metric.datapoint",
			"id", formatDatapointId(datapoint),
			"timestamp", datapoint.Timestamp,
			"maximum", core.ToFloat64(datapoint.Maximum),
			"minimum", core.ToFloat64(datapoint.Minimum),
			"average", core.ToFloat64(datapoint.Average),
			"sum", core.ToFloat64(datapoint.Sum),
			"unit", string(datapoint.Unit),
		)
		if err != nil {
			return nil, err
		}
		datapoints = append(datapoints, mqlDatapoint)
	}
	mqlStat, err := t.MotorRuntime.CreateResource("aws.cloudwatch.metricstatistics",
		"label", core.ToString(statsResp.Label),
		"datapoints", datapoints,
		"name", metricName,
		"namespace", namespace,
		"region", regionVal,
	)
	if err != nil {
		return nil, err
	}

	return mqlStat, nil
}

func (t *mqlAwsCloudwatchMetricDatapoint) id() (string, error) {
	return t.Id()
}

func formatDatapointId(d types.Datapoint) string {
	byteConfig, err := json.Marshal(d)
	if err != nil {
		return ""
	}
	h := sha256.New()
	h.Write(byteConfig)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (t *mqlAwsCloudwatchMetric) GetAlarms() ([]interface{}, error) {
	metricName, err := t.Name()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse metric name"))
	}
	namespace, err := t.Namespace()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse metric namespace"))
	}
	regionVal, err := t.Region()
	if err != nil {
		return nil, errors.Join(err, errors.New("unable to parse metric region"))
	}

	at, err := awsProvider(t.MotorRuntime.Motor.Provider)
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
		return nil, errors.Join(err, errors.New("could not gather aws cloudwatch alarms"))
	}
	res := []interface{}{}
	for _, alarm := range alarmsResp.MetricAlarms {
		mqlAlarm, err := t.MotorRuntime.CreateResource("aws.cloudwatch.metricsalarm",
			"arn", core.ToString(alarm.AlarmArn),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAlarm)
	}
	return res, nil
}

func (t *mqlAwsCloudwatch) GetAlarms() ([]interface{}, error) {
	at, err := awsProvider(t.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(t.getAlarms(at), 5)
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

func (t *mqlAwsCloudwatch) getAlarms(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Cloudwatch(regionVal)
			ctx := context.Background()

			res := []interface{}{}
			nextToken := aws.String("no_token_to_start_with")
			params := &cloudwatch.DescribeAlarmsInput{}
			for nextToken != nil {

				alarms, err := svc.DescribeAlarms(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, alarm := range alarms.MetricAlarms {
					actions := []interface{}{}
					for _, action := range alarm.AlarmActions {
						mqlAlarmAction, err := t.MotorRuntime.CreateResource("aws.sns.topic",
							"arn", action,
							"region", regionVal,
						)
						if err != nil {
							return nil, err
						}
						actions = append(actions, mqlAlarmAction)
					}
					insuffActions := []interface{}{}
					for _, action := range alarm.InsufficientDataActions {
						mqlInsuffAction, err := t.MotorRuntime.CreateResource("aws.sns.topic",
							"arn", action,
							"region", regionVal,
						)
						if err != nil {
							return nil, err
						}
						insuffActions = append(insuffActions, mqlInsuffAction)
					}

					okActions := []interface{}{}
					for _, action := range alarm.OKActions {
						mqlokAction, err := t.MotorRuntime.CreateResource("aws.sns.topic",
							"arn", action,
							"region", regionVal,
						)
						if err != nil {
							return nil, err
						}
						okActions = append(okActions, mqlokAction)
					}

					mqlAlarm, err := t.MotorRuntime.CreateResource("aws.cloudwatch.metricsalarm",
						"arn", core.ToString(alarm.AlarmArn),
						"metricName", core.ToString(alarm.MetricName),
						"metricNamespace", core.ToString(alarm.Namespace),
						"region", regionVal,
						"state", string(alarm.StateValue),
						"stateReason", core.ToString(alarm.StateReason),
						"insufficientDataActions", insuffActions,
						"okActions", okActions,
						"name", core.ToString(alarm.AlarmName),
						"actions", actions,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlAlarm)
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

func (t *mqlAwsSnsTopic) GetSubscriptions() ([]interface{}, error) {
	arnValue, err := t.Arn()
	if err != nil {
		return nil, err
	}
	regionVal, err := t.Region()
	if err != nil {
		return nil, err
	}
	at, err := awsProvider(t.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := at.Sns(regionVal)
	ctx := context.Background()

	mqlSubs := []interface{}{}
	params := &sns.ListSubscriptionsByTopicInput{TopicArn: &arnValue}
	nextToken := aws.String("no_token_to_start_with")
	for nextToken != nil {
		subsByTopic, err := svc.ListSubscriptionsByTopic(ctx, params)
		if err != nil {
			return nil, errors.Join(err, errors.New("could not gather sns subscriptions info"))
		}
		nextToken = subsByTopic.NextToken
		if subsByTopic.NextToken != nil {
			params.NextToken = nextToken
		}
		for _, sub := range subsByTopic.Subscriptions {
			mqlSub, err := t.MotorRuntime.CreateResource("aws.sns.subscription",
				"arn", core.ToString(sub.SubscriptionArn),
				"protocol", core.ToString(sub.Protocol),
			)
			if err != nil {
				return nil, err
			}
			mqlSubs = append(mqlSubs, mqlSub)
		}
	}
	return mqlSubs, nil
}

func (t *mqlAwsCloudwatch) GetLogGroups() ([]interface{}, error) {
	at, err := awsProvider(t.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(t.getLogGroups(at), 5)
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

func (t *mqlAwsCloudwatch) getLogGroups(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.CloudwatchLogs(regionVal)
			ctx := context.Background()

			nextToken := aws.String("no_token_to_start_with")
			params := &cloudwatchlogs.DescribeLogGroupsInput{}
			res := []interface{}{}
			for nextToken != nil {
				logGroups, err := svc.DescribeLogGroups(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, errors.Join(err, errors.New("could not gather aws cloudwatch log groups"))
				}
				nextToken = logGroups.NextToken
				if logGroups.NextToken != nil {
					params.NextToken = nextToken
				}
				for _, loggroup := range logGroups.LogGroups {
					args := []interface{}{
						"arn", core.ToString(loggroup.Arn),
						"name", core.ToString(loggroup.LogGroupName),
						"region", regionVal,
					}
					// add kms key if there is one
					if loggroup.KmsKeyId != nil {
						mqlKeyResource, err := t.MotorRuntime.CreateResource("aws.kms.key",
							"arn", core.ToString(loggroup.KmsKeyId),
						)
						if err != nil {
							return nil, err
						}
						mqlKey := mqlKeyResource.(AwsKmsKey)
						args = append(args, "kmsKey", mqlKey)
					}

					mqlLogGroup, err := t.MotorRuntime.CreateResource("aws.cloudwatch.loggroup", args...)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlLogGroup)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (c *mqlAwsCloudwatchLoggroup) init(args *resources.Args) (*resources.Args, AwsCloudwatchLoggroup, error) {
	if len(*args) > 1 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(c.MqlResource().MotorRuntime); ids != nil {
			(*args)["name"] = ids.name
			(*args)["arn"] = ids.arn
		}
	}
	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch cloudwatch log group")
	}

	obj, err := c.MotorRuntime.CreateResource("aws.cloudwatch")
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
		mqlLgArn, err := loggroup.Arn()
		if err != nil {
			return nil, nil, errors.New("cloudwatch log group does not exist")
		}
		if mqlLgArn == arnVal {
			return args, loggroup, nil
		}
	}
	return nil, nil, errors.New("cloudwatch log group does not exist")
}

func (s *mqlAwsCloudwatchLoggroup) GetKmsKey() (interface{}, error) {
	// no key id on the log group object
	return nil, nil
}

func (t *mqlAwsCloudwatchLoggroup) id() (string, error) {
	return t.Arn()
}

func (t *mqlAwsCloudwatchLoggroup) GetMetricsFilters() ([]interface{}, error) {
	arnValue, err := t.Arn()
	if err != nil || len(arnValue) < 6 {
		return nil, errors.Join(err, errors.New("unable to parse cloud watch log group arn"))
	}
	// arn:aws:logs:<region>:<aws_account_number>:log-group:GROUPVAL:*
	logGroupArn := strings.Split(arnValue, ":")
	groupName := logGroupArn[6]
	region := logGroupArn[3]

	at, err := awsProvider(t.MotorRuntime.Motor.Provider)
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
			return nil, errors.Join(err, errors.New("could not gather log metric filters"))
		}
		nextToken = metricsResp.NextToken
		if metricsResp.NextToken != nil {
			params.NextToken = nextToken
		}
		for _, m := range metricsResp.MetricFilters {
			mqlCloudwatchMetrics := []interface{}{}
			for _, mt := range m.MetricTransformations {
				mqlAwsMetric, err := t.MotorRuntime.CreateResource("aws.cloudwatch.metric",
					"name", core.ToString(mt.MetricName),
					"namespace", core.ToString(mt.MetricNamespace),
					"region", region,
				)
				if err != nil {
					return nil, err
				}
				mqlCloudwatchMetrics = append(mqlCloudwatchMetrics, mqlAwsMetric)
			}
			mqlAwsLogGroupMetricFilters, err := t.MotorRuntime.CreateResource("aws.cloudwatch.loggroup.metricsfilter",
				"id", groupName+"/"+region+"/"+core.ToString(m.FilterName),
				"filterName", core.ToString(m.FilterName),
				"filterPattern", core.ToString(m.FilterPattern),
				"metrics", mqlCloudwatchMetrics,
			)
			if err != nil {
				return nil, err
			}
			metricFilters = append(metricFilters, mqlAwsLogGroupMetricFilters)
		}
	}

	if err != nil {
		return nil, err
	}
	return metricFilters, nil
}

func (t *mqlAwsCloudwatchLoggroupMetricsfilter) id() (string, error) {
	return t.Id()
}

func (t *mqlAwsCloudwatchMetricsalarm) id() (string, error) {
	return t.Arn()
}

func (t *mqlAwsCloudwatchMetric) id() (string, error) {
	region, err := t.Region()
	if err != nil {
		return "", err
	}
	namespace, err := t.Namespace()
	if err != nil {
		return "", err
	}
	name, err := t.Name()
	if err != nil {
		return "", err
	}
	return region + "/" + namespace + "/" + name, nil
}

func (t *mqlAwsCloudwatchMetricsalarm) init(args *resources.Args) (*resources.Args, AwsCloudwatchMetricsalarm, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch aws cloudwatch metrics alarm")
	}

	// load all cloudwatch metrics alarm
	obj, err := t.MotorRuntime.CreateResource("aws.cloudwatch")
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
		mqlAlarmArn, err := alarm.Arn()
		if err != nil {
			return nil, nil, errors.New("cloudwatch alarm does not exist")
		}
		if mqlAlarmArn == arnVal {
			return args, alarm, nil
		}
	}
	return nil, nil, errors.New("cloudwatch alarm does not exist")
}
