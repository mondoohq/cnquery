// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
)

func (a *mqlAwsSqs) id() (string, error) {
	return "aws.sqs", nil
}

func (a *mqlAwsSqsQueue) id() (string, error) {
	return a.Url.Data, nil
}

func (a *mqlAwsSqs) queues() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getQueues(conn), 5)
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

func (a *mqlAwsSqs) getQueues(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sqs(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			nextToken := aws.String("no_token_to_start_with")
			params := &sqs.ListQueuesInput{}
			for nextToken != nil {
				qs, err := svc.ListQueues(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, q := range qs.QueueUrls {
					mqlTopic, err := CreateResource(a.MqlRuntime, "aws.sqs.queue",
						map[string]*llx.RawData{
							"url":    llx.StringData(q),
							"region": llx.StringData(regionVal),
						},
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlTopic)
				}
				nextToken = qs.NextToken
				if qs.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSqsQueueInternal struct {
	fetched   bool
	queueAtts map[string]string
	lock      sync.Mutex
}

func (a *mqlAwsSqsQueue) fetchAttributes() (map[string]string, error) {
	if a.fetched {
		return a.queueAtts, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()
	svc := conn.Sqs(a.Region.Data)
	desc, err := svc.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{QueueUrl: aws.String(a.Url.Data), AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameAll}})
	if err != nil {
		return nil, err
	}
	a.fetched = true
	a.queueAtts = desc.Attributes
	return desc.Attributes, nil
}

func (a *mqlAwsSqsQueue) kmsKey() (*mqlAwsKmsKey, error) {
	atts, err := a.fetchAttributes()
	if err != nil {
		return nil, err
	}
	if atts["KmsMasterKeyId"] == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	id := atts["KmsMasterKeyId"]
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	mqlKey, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{
			"arn": llx.StringData(fmt.Sprintf(kmsKeyArnPattern, a.Region.Data, conn.AccountId(), id)),
		})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsSqsQueue) deadLetterQueue() (*mqlAwsSqsQueue, error) {
	atts, err := a.fetchAttributes()
	if err != nil {
		return nil, err
	}
	c := atts["RedrivePolicy"]
	if c == "" {
		a.DeadLetterQueue.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	var r redrivePolicy
	err = json.Unmarshal([]byte(c), &r)
	if err != nil {
		return nil, err
	}
	parsedArn, err := arn.Parse(r.DeadLetterTargetArn)
	if err != nil {
		return nil, err
	}
	// "https://sqs.us-east-1.amazonaws.com/921877552404/Test-Preslav-Queue"
	url := fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/%s", a.Region.Data, parsedArn.AccountID, parsedArn.Resource)
	q, err := NewResource(a.MqlRuntime, "aws.sqs.queue",
		map[string]*llx.RawData{
			"arn":    llx.StringData(r.DeadLetterTargetArn),
			"url":    llx.StringData(url),
			"region": llx.StringData(a.Region.Data),
		})
	if err != nil {
		return nil, err
	}
	return q.(*mqlAwsSqsQueue), nil
}

func (a *mqlAwsSqsQueue) arn() (string, error) {
	atts, err := a.fetchAttributes()
	if err != nil {
		return "", err
	}
	return atts["QueueArn"], nil
}

func (a *mqlAwsSqsQueue) createdAt() (*time.Time, error) {
	atts, err := a.fetchAttributes()
	if err != nil {
		return nil, err
	}
	i, err := strconv.ParseInt(atts["CreatedTimestamp"], 10, 64)
	t := time.Unix(i, 0)
	return &t, nil
}

func (a *mqlAwsSqsQueue) deliveryDelaySeconds() (int64, error) {
	atts, err := a.fetchAttributes()
	if err != nil {
		return 0, err
	}
	c, err := strconv.Atoi(atts["DelaySeconds"])
	if err != nil {
		return 0, err
	}
	return int64(c), nil
}

func (a *mqlAwsSqsQueue) lastModified() (*time.Time, error) {
	atts, err := a.fetchAttributes()
	if err != nil {
		return nil, err
	}
	i, err := strconv.ParseInt(atts["LastModifiedTimestamp"], 10, 64)
	t := time.Unix(i, 0)
	return &t, nil
}

type redrivePolicy struct {
	DeadLetterTargetArn string `json:"deadLetterTargetArn,omitempty"`
	MaxReceiveCount     int    `json:"maxReceiveCount,omitempty"`
}

func (a *mqlAwsSqsQueue) maxReceiveCount() (int64, error) {
	atts, err := a.fetchAttributes()
	if err != nil {
		return 0, err
	}
	c := atts["RedrivePolicy"]
	if c == "" {
		return 0, nil
	}
	log.Info().Msgf("redrive %v", c)
	var r redrivePolicy
	err = json.Unmarshal([]byte(c), &r)
	if err != nil {
		return 0, err
	}
	return int64(r.MaxReceiveCount), nil
}

func (a *mqlAwsSqsQueue) maximumMessageSize() (int64, error) {
	atts, err := a.fetchAttributes()
	if err != nil {
		return 0, err
	}
	c, err := strconv.Atoi(atts["MaximumMessageSize"])
	if err != nil {
		return 0, err
	}
	return int64(c), nil
}

func (a *mqlAwsSqsQueue) messageRetentionPeriodSeconds() (int64, error) {
	atts, err := a.fetchAttributes()
	if err != nil {
		return 0, err
	}
	c, err := strconv.Atoi(atts["MessageRetentionPeriod"])
	if err != nil {
		return 0, err
	}
	return int64(c), nil
}

func (a *mqlAwsSqsQueue) receiveMessageWaitTimeSeconds() (int64, error) {
	atts, err := a.fetchAttributes()
	if err != nil {
		return 0, err
	}
	c, err := strconv.Atoi(atts["ReceiveMessageWaitTimeSeconds"])
	if err != nil {
		return 0, err
	}
	return int64(c), nil
}

func (a *mqlAwsSqsQueue) sqsManagedSseEnabled() (bool, error) {
	atts, err := a.fetchAttributes()
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(atts["SqsManagedSseEnabled"])
}

func (a *mqlAwsSqsQueue) queueType() (string, error) {
	atts, err := a.fetchAttributes()
	if err != nil {
		return "", err
	}
	if atts["FifoQueue"] == "true" {
		return "fifo", nil
	}
	return "standard", nil
}

func (a *mqlAwsSqsQueue) visibilityTimeoutSeconds() (int64, error) {
	atts, err := a.fetchAttributes()
	if err != nil {
		return 0, err
	}
	c, err := strconv.Atoi(atts["VisibilityTimeout"])
	if err != nil {
		return 0, err
	}
	return int64(c), nil
}
