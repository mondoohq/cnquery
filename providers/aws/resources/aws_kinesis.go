// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/firehose"
	firehose_types "github.com/aws/aws-sdk-go-v2/service/firehose/types"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	kinesis_types "github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsKinesis) id() (string, error) {
	return "aws.kinesis", nil
}

// streams lists Kinesis data streams across all regions
func (a *mqlAwsKinesis) streams() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getStreams(conn), 5)
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

func (a *mqlAwsKinesis) getStreams(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("kinesis>getStreams>calling aws with region %s", region)

			svc := conn.Kinesis(region)
			ctx := context.Background()
			res := []any{}

			paginator := kinesis.NewListStreamsPaginator(svc, &kinesis.ListStreamsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, streamSummary := range page.StreamSummaries {
					mqlStream, err := newMqlAwsKinesisStream(a.MqlRuntime, region, svc, ctx, &streamSummary)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlStream)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlAwsKinesisStream(runtime *plugin.Runtime, region string, svc *kinesis.Client, ctx context.Context, summary *kinesis_types.StreamSummary) (*mqlAwsKinesisStream, error) {
	// Use fields available from ListStreams StreamSummary
	streamModeDetails, err := convert.JsonToDict(summary.StreamModeDetails)
	if err != nil {
		return nil, err
	}

	args := map[string]*llx.RawData{
		"__id":              llx.StringDataPtr(summary.StreamARN),
		"arn":               llx.StringDataPtr(summary.StreamARN),
		"name":              llx.StringDataPtr(summary.StreamName),
		"status":            llx.StringData(string(summary.StreamStatus)),
		"streamModeDetails": llx.DictData(streamModeDetails),
		"createdAt":         llx.TimeDataPtr(summary.StreamCreationTimestamp),
		"region":            llx.StringData(region),
	}

	// Fetch additional fields from DescribeStreamSummary (encryption, retention, shard/consumer counts)
	descResp, err := svc.DescribeStreamSummary(ctx, &kinesis.DescribeStreamSummaryInput{
		StreamARN: summary.StreamARN,
	})
	if err != nil {
		log.Warn().Str("stream", convert.ToValue(summary.StreamName)).Err(err).Msg("could not describe stream, using defaults for detail fields")
		args["encryptionType"] = llx.StringData("")
		args["keyId"] = llx.StringData("")
		args["retentionPeriodHours"] = llx.IntData(0)
		args["openShardCount"] = llx.IntData(0)
		args["consumerCount"] = llx.IntData(0)
		args["enhancedMonitoring"] = llx.ArrayData([]any{}, types.Any)
	} else if descResp.StreamDescriptionSummary != nil {
		desc := descResp.StreamDescriptionSummary
		enhancedMonitoring, err := convert.JsonToDictSlice(desc.EnhancedMonitoring)
		if err != nil {
			return nil, err
		}
		args["encryptionType"] = llx.StringData(string(desc.EncryptionType))
		args["keyId"] = llx.StringDataPtr(desc.KeyId)
		args["retentionPeriodHours"] = llx.IntDataDefault(desc.RetentionPeriodHours, 0)
		args["openShardCount"] = llx.IntDataDefault(desc.OpenShardCount, 0)
		args["consumerCount"] = llx.IntDataDefault(desc.ConsumerCount, 0)
		args["enhancedMonitoring"] = llx.ArrayData(enhancedMonitoring, types.Any)
	} else {
		log.Warn().Str("stream", convert.ToValue(summary.StreamName)).Msg("nil stream description summary")
		args["encryptionType"] = llx.StringData("")
		args["keyId"] = llx.StringData("")
		args["retentionPeriodHours"] = llx.IntData(0)
		args["openShardCount"] = llx.IntData(0)
		args["consumerCount"] = llx.IntData(0)
		args["enhancedMonitoring"] = llx.ArrayData([]any{}, types.Any)
	}

	resource, err := CreateResource(runtime, "aws.kinesis.stream", args)
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsKinesisStream), nil
}

func (a *mqlAwsKinesisStream) consumers() ([]any, error) {
	arn := a.Arn.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Kinesis(region)
	ctx := context.Background()
	res := []any{}

	paginator := kinesis.NewListStreamConsumersPaginator(svc, &kinesis.ListStreamConsumersInput{
		StreamARN: &arn,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, consumer := range page.Consumers {
			mqlConsumer, err := newMqlAwsKinesisStreamConsumer(a.MqlRuntime, region, consumer, arn)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlConsumer)
		}
	}
	return res, nil
}

func newMqlAwsKinesisStreamConsumer(runtime *plugin.Runtime, region string, consumer kinesis_types.Consumer, streamArn string) (*mqlAwsKinesisStreamConsumer, error) {
	resource, err := CreateResource(runtime, "aws.kinesis.streamConsumer",
		map[string]*llx.RawData{
			"__id":      llx.StringDataPtr(consumer.ConsumerARN),
			"arn":       llx.StringDataPtr(consumer.ConsumerARN),
			"name":      llx.StringDataPtr(consumer.ConsumerName),
			"status":    llx.StringData(string(consumer.ConsumerStatus)),
			"createdAt": llx.TimeDataPtr(consumer.ConsumerCreationTimestamp),
			"region":    llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	mqlConsumer := resource.(*mqlAwsKinesisStreamConsumer)
	mqlConsumer.cacheStreamArn = streamArn
	return mqlConsumer, nil
}

type mqlAwsKinesisStreamConsumerInternal struct {
	cacheStreamArn string
}

func (a *mqlAwsKinesisStreamConsumer) stream() (*mqlAwsKinesisStream, error) {
	if a.cacheStreamArn == "" {
		a.Stream.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlStream, err := NewResource(a.MqlRuntime, "aws.kinesis.stream",
		map[string]*llx.RawData{
			"arn": llx.StringData(a.cacheStreamArn),
		})
	if err != nil {
		return nil, err
	}
	return mqlStream.(*mqlAwsKinesisStream), nil
}

// streamConsumers lists all enhanced fan-out consumers across all streams
func (a *mqlAwsKinesis) streamConsumers() ([]any, error) {
	streams := a.GetStreams()
	if streams.Error != nil {
		return nil, streams.Error
	}

	res := []any{}
	for _, s := range streams.Data {
		stream := s.(*mqlAwsKinesisStream)
		consumers := stream.GetConsumers()
		if consumers.Error != nil {
			return nil, consumers.Error
		}
		res = append(res, consumers.Data...)
	}
	return res, nil
}

func (a *mqlAwsKinesisStream) tags() (map[string]interface{}, error) {
	arn := a.Arn.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Kinesis(region)
	ctx := context.Background()

	tags := make(map[string]interface{})
	var exclusiveStartTagKey *string
	for {
		input := &kinesis.ListTagsForStreamInput{
			StreamARN:            &arn,
			ExclusiveStartTagKey: exclusiveStartTagKey,
		}
		resp, err := svc.ListTagsForStream(ctx, input)
		if err != nil {
			return nil, err
		}

		for _, tag := range resp.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}

		if resp.HasMoreTags == nil || !*resp.HasMoreTags {
			break
		}
		if len(resp.Tags) > 0 {
			exclusiveStartTagKey = resp.Tags[len(resp.Tags)-1].Key
		}
	}
	return tags, nil
}

// firehoseDeliveryStreams lists Firehose delivery streams across all regions
func (a *mqlAwsKinesis) firehoseDeliveryStreams() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getFirehoseDeliveryStreams(conn), 5)
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

func (a *mqlAwsKinesis) getFirehoseDeliveryStreams(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("kinesis>getFirehoseDeliveryStreams>calling aws with region %s", region)

			svc := conn.Firehose(region)
			ctx := context.Background()
			res := []any{}

			// Firehose doesn't have a paginator — use manual pagination
			var exclusiveStartName *string
			for {
				page, err := svc.ListDeliveryStreams(ctx, &firehose.ListDeliveryStreamsInput{
					ExclusiveStartDeliveryStreamName: exclusiveStartName,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, streamName := range page.DeliveryStreamNames {
					descResp, err := svc.DescribeDeliveryStream(ctx, &firehose.DescribeDeliveryStreamInput{
						DeliveryStreamName: &streamName,
					})
					if err != nil {
						log.Warn().Str("stream", streamName).Err(err).Msg("could not describe firehose delivery stream")
						continue
					}
					if descResp.DeliveryStreamDescription == nil {
						log.Warn().Str("stream", streamName).Msg("nil delivery stream description")
						continue
					}
					mqlStream, err := newMqlAwsKinesisFirehoseDeliveryStream(a.MqlRuntime, region, descResp.DeliveryStreamDescription)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlStream)
				}

				if page.HasMoreDeliveryStreams == nil || !*page.HasMoreDeliveryStreams {
					break
				}
				if len(page.DeliveryStreamNames) > 0 {
					last := page.DeliveryStreamNames[len(page.DeliveryStreamNames)-1]
					exclusiveStartName = &last
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlAwsKinesisFirehoseDeliveryStream(runtime *plugin.Runtime, region string, stream *firehose_types.DeliveryStreamDescription) (*mqlAwsKinesisFirehoseDeliveryStream, error) {
	encryption, err := convert.JsonToDict(stream.DeliveryStreamEncryptionConfiguration)
	if err != nil {
		return nil, err
	}

	source, err := convert.JsonToDict(stream.Source)
	if err != nil {
		return nil, err
	}

	destinations, err := convert.JsonToDictSlice(stream.Destinations)
	if err != nil {
		return nil, err
	}

	resource, err := CreateResource(runtime, "aws.kinesis.firehoseDeliveryStream",
		map[string]*llx.RawData{
			"__id":               llx.StringDataPtr(stream.DeliveryStreamARN),
			"arn":                llx.StringDataPtr(stream.DeliveryStreamARN),
			"name":               llx.StringDataPtr(stream.DeliveryStreamName),
			"status":             llx.StringData(string(stream.DeliveryStreamStatus)),
			"deliveryStreamType": llx.StringData(string(stream.DeliveryStreamType)),
			"encryption":         llx.DictData(encryption),
			"source":             llx.DictData(source),
			"destinations":       llx.ArrayData(destinations, types.Any),
			"createdAt":          llx.TimeDataPtr(stream.CreateTimestamp),
			"region":             llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsKinesisFirehoseDeliveryStream), nil
}

func (a *mqlAwsKinesisFirehoseDeliveryStream) tags() (map[string]interface{}, error) {
	name := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Firehose(region)
	ctx := context.Background()

	tags := make(map[string]interface{})
	var exclusiveStartTagKey *string
	for {
		input := &firehose.ListTagsForDeliveryStreamInput{
			DeliveryStreamName:   &name,
			ExclusiveStartTagKey: exclusiveStartTagKey,
		}
		resp, err := svc.ListTagsForDeliveryStream(ctx, input)
		if err != nil {
			return nil, err
		}

		for _, tag := range resp.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}

		if resp.HasMoreTags == nil || !*resp.HasMoreTags {
			break
		}
		if len(resp.Tags) > 0 {
			exclusiveStartTagKey = resp.Tags[len(resp.Tags)-1].Key
		}
	}
	return tags, nil
}
