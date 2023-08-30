// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/providers/aws/connection"

	"go.mondoo.com/cnquery/types"
)

func (a *mqlAwsCloudwatch) id() (string, error) {
	return "aws.cloudwatch", nil
}

func (a *mqlAwsCloudwatch) metrics() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getMetrics(conn), 5)
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

func (a *mqlAwsCloudwatch) getMetrics(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Cloudwatch(regionVal)
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
						mqlDimension, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.cloudwatch.metricdimension",
							map[string]*llx.RawData{
								"name":  llx.StringData(toString(d.Name)),
								"value": llx.StringData(toString(d.Value)),
							})
						if err != nil {
							return nil, err
						}
						dimensions = append(dimensions, mqlDimension)
					}

					mqlMetric, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.cloudwatch.metric",
						map[string]*llx.RawData{
							"name":       llx.StringData(toString(metric.MetricName)),
							"namespace":  llx.StringData(toString(metric.Namespace)),
							"region":     llx.StringData(regionVal),
							"dimensions": llx.ArrayData(dimensions, types.Resource("aws.cloudwatch.metricdimension")),
						})
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

func (a *mqlAwsCloudwatchMetricdimension) id() (string, error) {
	name := a.Name.Data
	val := a.Name.Data

	return name + "/" + val, nil
}

func (a *mqlAwsCloudwatchMetricstatistics) id() (string, error) {
	region := a.Region.Data
	namespace := a.Namespace.Data
	name := a.Name.Data
	label := a.Label.Data
	return namespace + "/" + name + "/" + region + "/" + label, nil
}

// allow the user to query for a specific namespace metric in a specific region
func initAwsCloudwatchMetric(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	namespaceRaw := args["namespace"]
	if namespaceRaw == nil {
		return args, nil, nil
	}

	namespace, ok := namespaceRaw.Value.(string)
	if !ok {
		return args, nil, nil
	}

	nameRaw := args["name"]
	if nameRaw == nil {
		return args, nil, nil
	}

	name, ok := nameRaw.Value.(string)
	if !ok {
		return args, nil, nil
	}

	regionRaw := args["region"]
	if namespaceRaw == nil {
		return args, nil, nil
	}

	region, ok := regionRaw.Value.(string)
	if !ok {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.AwsConnection)

	svc := conn.Cloudwatch(region)

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
		mqlDimension, err := runtime.CreateResource(runtime, "aws.cloudwatch.metricdimension",
			map[string]*llx.RawData{
				"name":  llx.StringData(toString(d.Name)),
				"value": llx.StringData(toString(d.Value)),
			})
		if err != nil {
			return args, nil, err
		}
		dimensions = append(dimensions, mqlDimension)
	}

	args["name"] = llx.StringData(name)
	args["namespace"] = llx.StringData(namespace)
	args["region"] = llx.StringData(region)
	args["dimensions"] = llx.ArrayData(dimensions, types.Resource("aws.cloudwatch.metricdimension"))

	return args, nil, nil
}

func (a *mqlAwsCloudwatchMetric) dimensions() ([]interface{}, error) {
	name := a.Name.Data
	namespace := a.Namespace.Data
	regionVal := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Cloudwatch(regionVal)
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
		mqlDimension, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.cloudwatch.metricdimension",
			map[string]*llx.RawData{
				"name":  llx.StringData(toString(d.Name)),
				"value": llx.StringData(toString(d.Value)),
			})
		if err != nil {
			return nil, err
		}
		dimensions = append(dimensions, mqlDimension)
	}
	return dimensions, nil
}

// allow the user to query for a specific namespace metric in a specific region
func initAwsCloudwatchMetricstatistics(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 3 {
		return args, nil, nil
	}

	namespaceRaw := args["namespace"]
	if namespaceRaw == nil {
		return args, nil, nil
	}

	namespace, ok := namespaceRaw.Value.(string)
	if !ok {
		return args, nil, nil
	}

	nameRaw := args["name"]
	if nameRaw == nil {
		return args, nil, nil
	}

	name, ok := nameRaw.Value.(string)
	if !ok {
		return args, nil, nil
	}

	regionRaw := args["region"]
	if namespaceRaw == nil {
		return args, nil, nil
	}

	region, ok := regionRaw.Value.(string)
	if !ok {
		return args, nil, nil
	}
	conn := runtime.Connection.(*connection.AwsConnection)

	svc := conn.Cloudwatch(region)
	ctx := context.Background()

	now := time.Now()
	dayAgo := time.Now().Add(-24 * time.Hour)
	params := &cloudwatch.GetMetricStatisticsInput{
		MetricName: &name,
		Namespace:  &namespace,
		StartTime:  &dayAgo,
		EndTime:    &now,
		Period:     aws.Int32(3600),
		Statistics: []cloudwatchtypes.Statistic{cloudwatchtypes.StatisticSum, cloudwatchtypes.StatisticAverage, cloudwatchtypes.StatisticMaximum, cloudwatchtypes.StatisticMinimum},
	}
	// no pagination required
	statsResp, err := svc.GetMetricStatistics(ctx, params)
	if err != nil {
		return args, nil, err
	}
	datapoints := []interface{}{}
	for _, datapoint := range statsResp.Datapoints {
		mqlDatapoint, err := runtime.CreateResource(runtime, "aws.cloudwatch.metric.datapoint",
			map[string]*llx.RawData{
				"timestamp": llx.TimeData(toTime(datapoint.Timestamp)),
				"maximum":   llx.FloatData(toFloat64(datapoint.Maximum)),
				"minimum":   llx.FloatData(toFloat64(datapoint.Minimum)),
				"average":   llx.FloatData(toFloat64(datapoint.Average)),
				"sum":       llx.FloatData(toFloat64(datapoint.Sum)),
				"unit":      llx.StringData(string(datapoint.Unit)),
			})
		if err != nil {
			return args, nil, err
		}
		datapoints = append(datapoints, mqlDatapoint)
	}

	if err != nil {
		return args, nil, err
	}

	args["label"] = llx.StringData(toString(statsResp.Label))
	args["datapoints"] = llx.ArrayData(datapoints, types.Resource("aws.cloudwatch.metric.datapoint"))
	args["name"] = llx.StringData(name)
	args["namespace"] = llx.StringData(namespace)
	args["region"] = llx.StringData(region)
	return args, nil, nil
}

func (a *mqlAwsCloudwatchMetric) statistics() (*mqlAwsCloudwatchMetricstatistics, error) {
	metricName := a.Name.Data
	namespace := a.Namespace.Data
	dimensions := a.Dimensions.Data
	regionVal := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Cloudwatch(regionVal)
	ctx := context.Background()

	now := time.Now()
	dayAgo := time.Now().Add(-24 * time.Hour)
	typedDimensions := make([]cloudwatchtypes.Dimension, len(dimensions))
	for i, d := range dimensions {
		dimension := d.(*mqlAwsCloudwatchMetricdimension)
		name := dimension.Name.Data
		val := dimension.Value.Data

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
		Statistics: []cloudwatchtypes.Statistic{cloudwatchtypes.StatisticSum, cloudwatchtypes.StatisticAverage, cloudwatchtypes.StatisticMaximum, cloudwatchtypes.StatisticMinimum},
	}
	// no pagination required
	statsResp, err := svc.GetMetricStatistics(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "could not gather AWS CloudWatch stats")
	}
	datapoints := []interface{}{}
	for _, datapoint := range statsResp.Datapoints {
		mqlDatapoint, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.cloudwatch.metric.datapoint",
			map[string]*llx.RawData{
				"id":        llx.StringData(formatDatapointId(datapoint)),
				"timestamp": llx.TimeData(toTime(datapoint.Timestamp)),
				"maximum":   llx.FloatData(toFloat64(datapoint.Maximum)),
				"minimum":   llx.FloatData(toFloat64(datapoint.Minimum)),
				"average":   llx.FloatData(toFloat64(datapoint.Average)),
				"sum":       llx.FloatData(toFloat64(datapoint.Sum)),
				"unit":      llx.StringData(string(datapoint.Unit)),
			})
		if err != nil {
			return nil, err
		}
		datapoints = append(datapoints, mqlDatapoint)
	}
	mqlStat, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.cloudwatch.metricstatistics",
		map[string]*llx.RawData{
			"label":      llx.StringData(toString(statsResp.Label)),
			"datapoints": llx.ArrayData(datapoints, types.Resource("aws.cloudwatch.metric.datapoint")),
			"name":       llx.StringData(metricName),
			"namespace":  llx.StringData(namespace),
			"region":     llx.StringData(regionVal),
		})
	if err != nil {
		return nil, err
	}

	return mqlStat.(*mqlAwsCloudwatchMetricstatistics), nil
}

func (a *mqlAwsCloudwatchMetricDatapoint) id() (string, error) {
	return a.Id.Data, nil
}

func formatDatapointId(d cloudwatchtypes.Datapoint) string {
	byteConfig, err := json.Marshal(d)
	if err != nil {
		return ""
	}
	h := sha256.New()
	h.Write(byteConfig)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (a *mqlAwsCloudwatchMetric) alarms() ([]interface{}, error) {
	metricName := a.Name.Data
	namespace := a.Namespace.Data
	regionVal := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Cloudwatch(regionVal)
	ctx := context.Background()

	params := &cloudwatch.DescribeAlarmsForMetricInput{
		MetricName: &metricName,
		Namespace:  &namespace,
	}
	// no pagination required
	alarmsResp, err := svc.DescribeAlarmsForMetric(ctx, params)
	if err != nil {
		return nil, errors.Wrap(err, "could not gather AWS CloudWatch alarms")
	}
	res := []interface{}{}
	for _, alarm := range alarmsResp.MetricAlarms {
		mqlAlarm, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.cloudwatch.metricsalarm",
			map[string]*llx.RawData{"arn": llx.StringData(toString(alarm.AlarmArn))})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAlarm)
	}
	return res, nil
}

func (a *mqlAwsCloudwatch) alarms() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getAlarms(conn), 5)
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

func (a *mqlAwsCloudwatch) getAlarms(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Cloudwatch(regionVal)
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
						mqlAlarmAction, err := NewResource(a.MqlRuntime, "aws.sns.topic",
							map[string]*llx.RawData{
								"arn":    llx.StringData(action),
								"region": llx.StringData(regionVal),
							})
						if err != nil {
							return nil, err
						}
						actions = append(actions, mqlAlarmAction)
					}
					insuffActions := []interface{}{}
					for _, action := range alarm.InsufficientDataActions {
						mqlInsuffAction, err := NewResource(a.MqlRuntime, "aws.sns.topic",
							map[string]*llx.RawData{
								"arn":    llx.StringData(action),
								"region": llx.StringData(regionVal),
							})
						if err != nil {
							return nil, err
						}
						insuffActions = append(insuffActions, mqlInsuffAction)
					}

					okActions := []interface{}{}
					for _, action := range alarm.OKActions {
						mqlokAction, err := NewResource(a.MqlRuntime, "aws.sns.topic",
							map[string]*llx.RawData{
								"arn":    llx.StringData(action),
								"region": llx.StringData(regionVal),
							})
						if err != nil {
							return nil, err
						}
						okActions = append(okActions, mqlokAction)
					}

					mqlAlarm, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.cloudwatch.metricsalarm",
						map[string]*llx.RawData{
							"arn":                     llx.StringData(toString(alarm.AlarmArn)),
							"metricName":              llx.StringData(toString(alarm.MetricName)),
							"metricNamespace":         llx.StringData(toString(alarm.Namespace)),
							"region":                  llx.StringData(regionVal),
							"state":                   llx.StringData(string(alarm.StateValue)),
							"stateReason":             llx.StringData(toString(alarm.StateReason)),
							"insufficientDataActions": llx.ArrayData(insuffActions, types.Resource("aws.sns.topic")),
							"okActions":               llx.ArrayData(okActions, types.Resource("aws.sns.topic")),
							"name":                    llx.StringData(toString(alarm.AlarmName)),
							"actions":                 llx.ArrayData(actions, types.Resource("aws.sns.topic")),
						})
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

func (a *mqlAwsCloudwatch) logGroups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getLogGroups(conn), 5)
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

func (a *mqlAwsCloudwatch) getLogGroups(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := conn.CloudwatchLogs(regionVal)
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
					return nil, errors.Wrap(err, "could not gather AWS CloudWatch log groups")
				}
				nextToken = logGroups.NextToken
				if logGroups.NextToken != nil {
					params.NextToken = nextToken
				}
				args := make(map[string]*llx.RawData)
				for _, loggroup := range logGroups.LogGroups {
					args["arn"] = llx.StringData(toString(loggroup.Arn))
					args["name"] = llx.StringData(toString(loggroup.LogGroupName))
					args["region"] = llx.StringData(regionVal)

					// add kms key if there is one
					if loggroup.KmsKeyId != nil {
						mqlKeyResource, err := NewResource(a.MqlRuntime, "aws.kms.key",
							map[string]*llx.RawData{
								"arn": llx.StringData(toString(loggroup.KmsKeyId)),
							})
						if err != nil {
							return nil, err
						}

						mqlKey := mqlKeyResource.(*mqlAwsKmsKey)
						args["kmsKey"] = llx.ResourceData(mqlKey, mqlKey.MqlName())
					}

					mqlLogGroup, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.cloudwatch.loggroup", args)
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

func initAwsCloudwatchLoggroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	// if len(*args) == 0 {
	// 	if ids := getAssetIdentifier(c.MqlResource().MotorRuntime); ids != nil {
	// 		args["name"] = ids.name
	// 		args["arn"] = ids.arn
	// 	}
	// }
	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch cloudwatch log group")
	}

	obj, err := runtime.CreateResource(runtime, "aws.cloudwatch", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	cloudwatch := obj.(*mqlAwsCloudwatch)
	rawResources := cloudwatch.LogGroups.Data

	arnVal := args["arn"].Value.(string)
	for i := range rawResources {
		loggroup := rawResources[i].(*mqlAwsCloudwatchLoggroup)
		mqlLgArn := loggroup.Arn.Data

		if mqlLgArn == arnVal {
			return args, loggroup, nil
		}
	}
	return nil, nil, errors.New("cloudwatch log group does not exist")
}

func (s *mqlAwsCloudwatchLoggroup) kmsKey() (*mqlAwsKmsKey, error) {
	// no key id on the log group object
	return &mqlAwsKmsKey{}, nil
}

func (a *mqlAwsCloudwatchLoggroup) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsCloudwatchLoggroup) metricsFilters() ([]interface{}, error) {
	arnValue := a.Arn.Data

	// arn:aws:logs:<region>:<aws_account_number>:log-group:GROUPVAL:*
	logGroupArn := strings.Split(arnValue, ":")
	groupName := logGroupArn[6]
	region := logGroupArn[3]

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.CloudwatchLogs(region)
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
			mqlCloudwatchMetrics := []interface{}{}
			for _, mt := range m.MetricTransformations {
				mqlAwsMetric, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.cloudwatch.metric",
					map[string]*llx.RawData{
						"name":      llx.StringData(toString(mt.MetricName)),
						"namespace": llx.StringData(toString(mt.MetricNamespace)),
						"region":    llx.StringData(region),
					})
				if err != nil {
					return nil, err
				}
				mqlCloudwatchMetrics = append(mqlCloudwatchMetrics, mqlAwsMetric)
			}
			mqlAwsLogGroupMetricFilters, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.cloudwatch.loggroup.metricsfilter",
				map[string]*llx.RawData{
					"id":            llx.StringData(groupName + "/" + region + "/" + toString(m.FilterName)),
					"filterName":    llx.StringData(toString(m.FilterName)),
					"filterPattern": llx.StringData(toString(m.FilterPattern)),
					"metrics":       llx.ArrayData(mqlCloudwatchMetrics, types.Resource("aws.cloudwatch.metric")),
				})
			if err != nil {
				return nil, err
			}
			metricFilters = append(metricFilters, mqlAwsLogGroupMetricFilters)
		}
	}
	return metricFilters, nil
}

func (a *mqlAwsCloudwatchLoggroupMetricsfilter) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsCloudwatchMetricsalarm) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsCloudwatchMetric) id() (string, error) {
	region := a.Region.Data
	namespace := a.Namespace.Data
	name := a.Name.Data
	return region + "/" + namespace + "/" + name, nil
}

func initAwsCloudwatchMetricsalarm(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch AWS CloudWatch metrics alarm")
	}

	// load all cloudwatch metrics alarm
	obj, err := runtime.CreateResource(runtime, "aws.cloudwatch", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	aws := obj.(*mqlAwsCloudwatch)

	rawResources := aws.Alarms.Data

	arnVal := args["arn"].Value.(string)
	for i := range rawResources {
		alarm := rawResources[i].(*mqlAwsCloudwatchMetricsalarm)
		if alarm.Arn.Data == arnVal {
			return args, alarm, nil
		}
	}
	return nil, nil, errors.New("cloudwatch alarm does not exist")
}
