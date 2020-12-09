package resources

import (
	"context"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
)

func (t *lumiAwsCloudtrail) id() (string, error) {
	return "aws.cloudtrail", nil
}

func (t *lumiAwsCloudtrail) GetTrails() ([]interface{}, error) {
	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.Cloudtrail("")
	ctx := context.Background()

	trailsResp, err := svc.DescribeTrailsRequest(&cloudtrail.DescribeTrailsInput{}).Send(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not gather aws iam virtual-mfa-devices")
	}

	res := []interface{}{}
	for i := range trailsResp.TrailList {
		trail := trailsResp.TrailList[i]

		// trail.S3BucketName
		var s3Bucket interface{}
		if trail.S3BucketName != nil {
			lumiAwsS3Bucket, err := t.Runtime.CreateResource("aws.s3.bucket",
				"name", toString(trail.S3BucketName),
			)
			if err != nil {
				return nil, err
			}
			s3Bucket = lumiAwsS3Bucket
		}

		lumiAwsCloudtrailTrail, err := t.Runtime.CreateResource("aws.cloudtrail.trail",
			"arn", toString(trail.TrailARN),
			"name", toString(trail.Name),
			"kmsKeyId", toString(trail.KmsKeyId),
			"isMultiRegionTrail", toBool(trail.IsMultiRegionTrail),
			"isOrganizationTrail", toBool(trail.IsOrganizationTrail),
			"logFileValidationEnabled", toBool(trail.LogFileValidationEnabled),
			"includeGlobalServiceEvents", toBool(trail.IncludeGlobalServiceEvents),
			"s3bucket", s3Bucket,
			"snsTopicARN", toString(trail.SnsTopicARN),
			// TODO: link to log group
			"cloudWatchLogsLogGroupArn", toString(trail.CloudWatchLogsLogGroupArn),
			// TODO: link to watch logs grou
			"cloudWatchLogsRoleArn", toString(trail.CloudWatchLogsRoleArn),
		)
		if err != nil {
			return nil, err
		}

		res = append(res, lumiAwsCloudtrailTrail)
	}

	return res, nil
}

func (t *lumiAwsCloudtrailTrail) id() (string, error) {
	return t.Arn()
}

func (t *lumiAwsCloudtrailTrail) GetStatus() (interface{}, error) {
	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.Cloudtrail("")
	ctx := context.Background()

	arnValue, err := t.Arn()
	if err != nil {
		return nil, err
	}

	trailstatus, err := svc.GetTrailStatusRequest(&cloudtrail.GetTrailStatusInput{
		Name: &arnValue,
	}).Send(ctx)
	if err != nil {
		return nil, err
	}

	lumiAwsCloudtrailTrailStatus, err := t.Runtime.CreateResource("aws.cloudtrail.trailstatus",
		"arn", arnValue,
		"isLogging", toBool(trailstatus.IsLogging),
		"latestCloudWatchLogsDeliveryError", toString(trailstatus.LatestCloudWatchLogsDeliveryError),
		"latestCloudWatchLogsDeliveryTime", trailstatus.LatestCloudWatchLogsDeliveryTime,
		"latestDeliveryError", toString(trailstatus.LatestDeliveryError),
		"latestDeliveryTime", trailstatus.LatestDeliveryTime,
		"latestDigestDeliveryError", toString(trailstatus.LatestDigestDeliveryError),
		"latestDigestDeliveryTime", trailstatus.LatestDigestDeliveryTime,
		"latestNotificationError", toString(trailstatus.LatestNotificationError),
		"latestNotificationTime", trailstatus.LatestNotificationTime,
		"startLoggingTime", trailstatus.StartLoggingTime,
		"stopLoggingTime", trailstatus.StopLoggingTime,
	)
	if err != nil {
		return nil, err
	}
	return lumiAwsCloudtrailTrailStatus, nil
}

func (t *lumiAwsCloudtrailTrailstatus) id() (string, error) {
	return t.Arn()
}

func (t *lumiAwsCloudtrailTrail) GetMgmtEvents() (interface{}, error) {
	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.Cloudtrail("")
	ctx := context.Background()

	arnValue, err := t.Arn()
	if err != nil {
		return nil, err
	}

	trailmgmtevents, err := svc.GetEventSelectorsRequest(&cloudtrail.GetEventSelectorsInput{
		TrailName: &arnValue,
	}).Send(ctx)
	if err != nil {
		return nil, err
	}
	eventSelectors := []interface{}{}
	for i, trailmgmtevent := range trailmgmtevents.EventSelectors {
		stringState, err := cloudtrail.ReadWriteType.MarshalValue(trailmgmtevent.ReadWriteType)
		if err != nil {
			return nil, err
		}

		lumiAwsTrailEventSelectors, err := t.Runtime.CreateResource("aws.cloudtrail.trailmgmtevents.eventselectors",
			"id", arnValue+"/"+strconv.Itoa(i),
			"includeManagementEvents", toBool(trailmgmtevent.IncludeManagementEvents),
			"readWriteType", stringState,
		)
		eventSelectors = append(eventSelectors, lumiAwsTrailEventSelectors)
	}

	lumiAwsCloudtrailTrailMgmtEvents, err := t.Runtime.CreateResource("aws.cloudtrail.trailmgmtevents",
		"arn", toString(trailmgmtevents.TrailARN),
		"eventSelectors", eventSelectors,
	)
	if err != nil {
		return nil, err
	}
	return lumiAwsCloudtrailTrailMgmtEvents, nil
}

func (t *lumiAwsCloudtrailTrailmgmtevents) id() (string, error) {
	return t.Arn()
}
func (t *lumiAwsCloudtrailTrailmgmteventsEventselectors) id() (string, error) {
	return t.Id()
}

func (t *lumiAwsCloudtrailTrail) GetMetricFilters() (interface{}, error) {
	arnValue, err := t.CloudWatchLogsLogGroupArn()
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
			// metric transformations is an array, but everything seems to imply it will only
			// ever have 0 or 1 items in the array (e.g. CIS says "grab the metric name associated with filter")
			// i tried to add more metric transformations to it but could not :shrug:
			// so here we get the data if there is the expected 1 item..
			metricName, metricValue := "", ""
			if len(m.MetricTransformations) == 1 {
				metricName = toString(m.MetricTransformations[0].MetricName)
				metricValue = toString(m.MetricTransformations[0].MetricValue)
			} else {
				log.Warn().Msg("unexpected length found for metric values array. not including metric name in result")
			}

			lumiAwsTrailMetricFilters, err := t.Runtime.CreateResource("aws.cloudtrail.trailmetricfilters.metricfilters",
				"id", groupName+"/"+region+"/"+toString(m.FilterName),
				"filterName", toString(m.FilterName),
				"filterPattern", toString(m.FilterPattern),
				"metricName", metricName,
				"metricValue", metricValue,
			)
			if err != nil {
				return nil, err
			}
			metricFilters = append(metricFilters, lumiAwsTrailMetricFilters)
		}
	}

	lumiAwsCloudtrailTrailMetricFilters, err := t.Runtime.CreateResource("aws.cloudtrail.trailmetricfilters",
		"id", groupName+"/"+region,
		"metrics", metricFilters,
	)
	if err != nil {
		return nil, err
	}
	return lumiAwsCloudtrailTrailMetricFilters, nil
}

func (t *lumiAwsCloudtrailTrailmetricfilters) id() (string, error) {
	return t.Id()
}

func (t *lumiAwsCloudtrailTrailmetricfiltersMetricfilters) id() (string, error) {
	return t.Id()
}

func (t *lumiAwsCloudtrailTrailmetricfiltersMetricfilters) GetAlarmsForMetric() ([]interface{}, error) {
	metricName, err := t.MetricName()
	if err != nil {
		return nil, err
	}
	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.Cloudwatch("")
	ctx := context.Background()

	namespace := "CloudTrailMetrics"
	params := &cloudwatch.DescribeAlarmsForMetricInput{
		MetricName: &metricName,
		Namespace:  &namespace,
	}
	res := []interface{}{}
	alarmsForMetric, err := svc.DescribeAlarmsForMetricRequest(params).Send(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not gather aws cloudwatch alarms")
	}

	for _, alarm := range alarmsForMetric.MetricAlarms {
		actions := []interface{}{}
		// alarm actions is a list of sns topic arns
		for _, topicArn := range alarm.AlarmActions {
			// to make this information valuable to the user, we call the sns list subscriptions by topic api
			// with each sns topic arn and create an object that includes the topic arn,
			//sub arn, and whetheror not that sub arn is valid
			subscriptionArns, err := t.listSnsSubscriptionsByTopic(topicArn)
			if err != nil {
				return nil, err
			}
			for _, subscriptionArn := range subscriptionArns {
				// the existence of a valid subscription arn("arn:aws:sns:<region>:<aws_account_number>:<SnsTopicName>:<SubscriptionID>"
				// suggests the subscription is active
				isValid := isValidSubscription(subscriptionArn)
				lumiAwsCloudwatchAlarmAction, err := t.Runtime.CreateResource("aws.cloudtrail.alarmactions",
					"snsTopicArn", topicArn,
					"subscriptionArn", subscriptionArn,
					"validSubscription", isValid,
				)
				if err != nil {
					return nil, err
				}
				actions = append(actions, lumiAwsCloudwatchAlarmAction)
			}
		}
		lumiAwsCloudwatchAlarm, err := t.Runtime.CreateResource("aws.cloudtrail.trailmetricfilters.metricfilters.alarms",
			"arn", toString(alarm.AlarmArn),
			"name", toString(alarm.AlarmName),
			"alarmActions", actions,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiAwsCloudwatchAlarm)

	}
	return res, nil
}

func (t *lumiAwsCloudtrailTrailmetricfiltersMetricfiltersAlarms) id() (string, error) {
	return t.Arn()
}

func (t *lumiAwsCloudtrailAlarmactions) id() (string, error) {
	return t.SubscriptionArn()
}

func (t *lumiAwsCloudtrailTrailmetricfiltersMetricfilters) listSnsSubscriptionsByTopic(topicArn string) ([]string, error) {
	subs := []string{}

	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.Sns("")
	ctx := context.Background()

	subsByTopic, err := svc.ListSubscriptionsByTopicRequest(&sns.ListSubscriptionsByTopicInput{TopicArn: &topicArn}).Send(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not gather sns subscriptions info")
	}
	for _, sub := range subsByTopic.Subscriptions {
		subs = append(subs, toString(sub.SubscriptionArn))
	}
	return subs, nil
}

func isValidSubscription(subscriptionArn string) bool {
	if len(subscriptionArn) == 0 {
		return false
	}
	if strings.HasPrefix(subscriptionArn, "arn:aws:sns:") {
		return true
	}
	return false
}
