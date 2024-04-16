// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/smithy-go/transport/http"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
)

func (a *mqlAwsSns) id() (string, error) {
	return "aws.sns", nil
}

func (a *mqlAwsSnsTopic) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSnsSubscription) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSns) topics() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getTopics(conn), 5)
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

func (a *mqlAwsSnsTopic) init(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch sns topic")
	}
	arnVal := args["arn"].Value.(string)
	arn, err := arn.Parse(arnVal)
	if err != nil {
		return nil, nil, err
	}

	args["arn"] = llx.StringData(arnVal)
	args["region"] = llx.StringData(arn.Region)
	return args, nil, nil
}

func (a *mqlAwsSns) getTopics(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sns(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &sns.ListTopicsInput{}
			for nextToken != nil {
				topics, err := svc.ListTopics(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, topic := range topics.Topics {
					mqlTopic, err := CreateResource(a.MqlRuntime, "aws.sns.topic",
						map[string]*llx.RawData{
							"arn":    llx.StringDataPtr(topic.TopicArn),
							"region": llx.StringData(regionVal),
						},
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlTopic)
				}
				nextToken = topics.NextToken
				if topics.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsSnsTopic) attributes() (interface{}, error) {
	arn := a.Arn.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Sns(region)
	ctx := context.Background()

	topicAttributes, err := svc.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{TopicArn: &arn})
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(topicAttributes.Attributes)
}

func (a *mqlAwsSnsTopic) tags() (map[string]interface{}, error) {
	arn := a.Arn.Data
	region := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Sns(region)
	ctx := context.Background()

	return getSNSTags(ctx, svc, &arn)
}

func getSNSTags(ctx context.Context, svc *sns.Client, arn *string) (map[string]interface{}, error) {
	resp, err := svc.ListTagsForResource(ctx, &sns.ListTagsForResourceInput{ResourceArn: arn})
	var respErr *http.ResponseError
	if err != nil {
		if errors.As(err, &respErr) {
			if respErr.HTTPStatusCode() == 404 || respErr.HTTPStatusCode() == 400 { // some sns topics do not support tags..
				return nil, nil
			}
		}
		return nil, err
	}
	tags := make(map[string]interface{})
	for _, t := range resp.Tags {
		tags[*t.Key] = *t.Value
	}
	return tags, nil
}

func (a *mqlAwsSnsTopic) subscriptions() ([]interface{}, error) {
	arnValue := a.Arn.Data
	regionVal := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Sns(regionVal)
	ctx := context.Background()

	mqlSubs := []interface{}{}
	params := &sns.ListSubscriptionsByTopicInput{TopicArn: &arnValue}
	nextToken := aws.String("no_token_to_start_with")
	for nextToken != nil {
		subsByTopic, err := svc.ListSubscriptionsByTopic(ctx, params)
		if err != nil {
			return nil, err
		}
		nextToken = subsByTopic.NextToken
		if subsByTopic.NextToken != nil {
			params.NextToken = nextToken
		}
		for _, sub := range subsByTopic.Subscriptions {
			mqlSub, err := CreateResource(a.MqlRuntime, "aws.sns.subscription",
				map[string]*llx.RawData{
					"arn":      llx.StringDataPtr(sub.SubscriptionArn),
					"protocol": llx.StringDataPtr(sub.Protocol),
				})
			if err != nil {
				return nil, err
			}
			mqlSubs = append(mqlSubs, mqlSub)
		}
	}
	return mqlSubs, nil
}
