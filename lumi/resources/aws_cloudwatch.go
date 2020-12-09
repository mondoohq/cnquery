package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/cockroachdb/errors"
)

func (t *lumiAwsCloudwatch) id() (string, error) {
	return "aws.cloudwatch", nil
}

func (t *lumiAwsCloudwatch) GetAlarms() ([]interface{}, error) {
	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.Cloudwatch("")
	ctx := context.Background()

	nextToken := aws.String("no_token_to_start_with")
	params := &cloudwatch.DescribeAlarmsInput{}
	res := []interface{}{}
	for nextToken != nil {
		alarmsResp, err := svc.DescribeAlarmsRequest(params).Send(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not gather aws cloudwatch alarms")
		}
		nextToken = alarmsResp.NextToken
		if alarmsResp.NextToken != nil {
			params.NextToken = nextToken
		}
		for _, alarm := range alarmsResp.MetricAlarms {
			actions := []interface{}{}
			for _, action := range alarm.AlarmActions {
				actions = append(actions, action)
			}
			lumiAwsCloudwatchAlarm, err := t.Runtime.CreateResource("aws.cloudwatch.alarm",
				"arn", toString(alarm.AlarmArn),
				"name", toString(alarm.AlarmName),
				"alarmActions", actions,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, lumiAwsCloudwatchAlarm)
		}
	}
	return res, nil
}

func (t *lumiAwsCloudwatchAlarm) id() (string, error) {
	return t.Arn()
}
