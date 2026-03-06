// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/smithy-go/transport/http"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
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

func (a *mqlAwsSns) topics() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getTopics(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
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
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sns(region)
			ctx := context.Background()
			res := []any{}

			params := &sns.ListTopicsInput{}
			paginator := sns.NewListTopicsPaginator(svc, params)
			for paginator.HasMorePages() {
				topics, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, topic := range topics.Topics {
					mqlTopic, err := CreateResource(a.MqlRuntime, "aws.sns.topic",
						map[string]*llx.RawData{
							"__id":   llx.StringDataPtr(topic.TopicArn),
							"arn":    llx.StringDataPtr(topic.TopicArn),
							"region": llx.StringData(region),
						},
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlTopic)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSnsTopicInternal struct {
	fetched   bool
	topicAtts map[string]string
	lock      sync.Mutex
}

func (a *mqlAwsSnsTopic) fetchTopicAttributes() (map[string]string, error) {
	if a.fetched {
		return a.topicAtts, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.topicAtts, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sns(a.Region.Data)
	ctx := context.Background()
	arn := a.Arn.Data
	resp, err := svc.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{TopicArn: &arn})
	if err != nil {
		return nil, err
	}
	a.fetched = true
	a.topicAtts = resp.Attributes
	return a.topicAtts, nil
}

func (a *mqlAwsSnsTopic) attributes() (any, error) {
	atts, err := a.fetchTopicAttributes()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(atts)
}

func (a *mqlAwsSnsTopic) signatureVersion() (string, error) {
	atts, err := a.fetchTopicAttributes()
	if err != nil {
		return "", err
	}
	return atts["SignatureVersion"], nil
}

func (a *mqlAwsSnsTopic) kmsMasterKey() (*mqlAwsKmsKey, error) {
	atts, err := a.fetchTopicAttributes()
	if err != nil {
		return nil, err
	}
	keyId := atts["KmsMasterKeyId"]
	if keyId != "" {
		mqlKeyResource, err := NewResource(a.MqlRuntime, "aws.kms.key",
			map[string]*llx.RawData{"arn": llx.StringData(keyId)},
		)
		if err != nil {
			return nil, err
		}
		return mqlKeyResource.(*mqlAwsKmsKey), nil
	}
	a.KmsMasterKey.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAwsSnsTopic) tags() (map[string]any, error) {
	arn := a.Arn.Data
	region := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Sns(region)
	ctx := context.Background()

	return getSNSTags(ctx, svc, &arn)
}

func getSNSTags(ctx context.Context, svc *sns.Client, arn *string) (map[string]any, error) {
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
	tags := make(map[string]any)
	for _, t := range resp.Tags {
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
		}
	}
	return tags, nil
}

func (a *mqlAwsSnsTopic) subscriptions() ([]any, error) {
	arnValue := a.Arn.Data
	regionVal := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Sns(regionVal)
	ctx := context.Background()

	mqlSubs := []any{}
	params := &sns.ListSubscriptionsByTopicInput{TopicArn: &arnValue}
	paginator := sns.NewListSubscriptionsByTopicPaginator(svc, params)
	for paginator.HasMorePages() {
		subsByTopic, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, sub := range subsByTopic.Subscriptions {
			// Pending subscriptions have ARN "PendingConfirmation" which is not unique.
			// Synthesize a stable __id from topic ARN + protocol + endpoint.
			subId := convert.ToValue(sub.SubscriptionArn)
			if !arn.IsARN(subId) {
				subId = arnValue + "/" + convert.ToValue(sub.Protocol) + "/" + convert.ToValue(sub.Endpoint)
			}
			mqlSub, err := CreateResource(a.MqlRuntime, "aws.sns.subscription",
				map[string]*llx.RawData{
					"__id":     llx.StringData(subId),
					"arn":      llx.StringDataPtr(sub.SubscriptionArn),
					"protocol": llx.StringDataPtr(sub.Protocol),
					"endpoint": llx.StringDataPtr(sub.Endpoint),
					"owner":    llx.StringDataPtr(sub.Owner),
					"region":   llx.StringData(regionVal),
				})
			if err != nil {
				return nil, err
			}
			mqlSub.(*mqlAwsSnsSubscription).cacheTopicArn = sub.TopicArn
			mqlSubs = append(mqlSubs, mqlSub)
		}
	}
	return mqlSubs, nil
}

// Internal caching for subscription attributes
type mqlAwsSnsSubscriptionInternal struct {
	cacheTopicArn *string
	fetched       bool
	attrs         map[string]string
	lock          sync.Mutex
}

func (a *mqlAwsSnsSubscription) topic() (*mqlAwsSnsTopic, error) {
	if a.cacheTopicArn == nil || *a.cacheTopicArn == "" {
		a.Topic.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlTopic, err := NewResource(a.MqlRuntime, "aws.sns.topic",
		map[string]*llx.RawData{
			"arn": llx.StringDataPtr(a.cacheTopicArn),
		})
	if err != nil {
		return nil, err
	}
	return mqlTopic.(*mqlAwsSnsTopic), nil
}

func (a *mqlAwsSnsSubscription) fetchAttributes() (map[string]string, error) {
	if a.fetched {
		return a.attrs, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.attrs, nil
	}

	arnVal := a.Arn.Data

	// Unconfirmed subscriptions have ARN set to "PendingConfirmation" which is
	// not a valid ARN. GetSubscriptionAttributes will reject it, so return
	// a minimal attribute map with the pending status instead.
	if !arn.IsARN(arnVal) {
		a.fetched = true
		a.attrs = map[string]string{"PendingConfirmation": "true"}
		return a.attrs, nil
	}

	regionVal := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sns(regionVal)
	ctx := context.Background()

	resp, err := svc.GetSubscriptionAttributes(ctx, &sns.GetSubscriptionAttributesInput{
		SubscriptionArn: &arnVal,
	})
	if err != nil {
		return nil, err
	}

	a.fetched = true
	a.attrs = resp.Attributes
	return a.attrs, nil
}

func (a *mqlAwsSnsSubscription) attributes() (any, error) {
	attrs, err := a.fetchAttributes()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(attrs)
}

func (a *mqlAwsSnsSubscription) rawMessageDelivery() (bool, error) {
	attrs, err := a.fetchAttributes()
	if err != nil {
		return false, err
	}
	return attrs["RawMessageDelivery"] == "true", nil
}

func (a *mqlAwsSnsSubscription) filterPolicy() (any, error) {
	attrs, err := a.fetchAttributes()
	if err != nil {
		return nil, err
	}
	val, ok := attrs["FilterPolicy"]
	if !ok || val == "" {
		return nil, nil
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}
	return convert.JsonToDict(result)
}

func (a *mqlAwsSnsSubscription) filterPolicyScope() (string, error) {
	attrs, err := a.fetchAttributes()
	if err != nil {
		return "", err
	}
	return attrs["FilterPolicyScope"], nil
}

func (a *mqlAwsSnsSubscription) redrivePolicy() (any, error) {
	attrs, err := a.fetchAttributes()
	if err != nil {
		return nil, err
	}
	val, ok := attrs["RedrivePolicy"]
	if !ok || val == "" {
		return nil, nil
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}
	return convert.JsonToDict(result)
}

func (a *mqlAwsSnsSubscription) confirmationWasAuthenticated() (bool, error) {
	attrs, err := a.fetchAttributes()
	if err != nil {
		return false, err
	}
	return attrs["ConfirmationWasAuthenticated"] == "true", nil
}

func (a *mqlAwsSnsSubscription) deliveryPolicy() (any, error) {
	attrs, err := a.fetchAttributes()
	if err != nil {
		return nil, err
	}
	val, ok := attrs["DeliveryPolicy"]
	if !ok || val == "" {
		return nil, nil
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}
	return convert.JsonToDict(result)
}

func (a *mqlAwsSnsSubscription) pendingConfirmation() (bool, error) {
	attrs, err := a.fetchAttributes()
	if err != nil {
		return false, err
	}
	return attrs["PendingConfirmation"] == "true", nil
}
