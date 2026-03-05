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
					// Get full stream details via DescribeStreamSummary
					descResp, err := svc.DescribeStreamSummary(ctx, &kinesis.DescribeStreamSummaryInput{
						StreamARN: streamSummary.StreamARN,
					})
					if err != nil {
						log.Warn().Str("stream", convert.ToValue(streamSummary.StreamName)).Err(err).Msg("could not describe stream")
						continue
					}
					if descResp.StreamDescriptionSummary == nil {
						log.Warn().Str("stream", convert.ToValue(streamSummary.StreamName)).Msg("nil stream description summary")
						continue
					}
					mqlStream, err := newMqlAwsKinesisStream(a.MqlRuntime, region, descResp.StreamDescriptionSummary)
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

func newMqlAwsKinesisStream(runtime *plugin.Runtime, region string, stream *kinesis_types.StreamDescriptionSummary) (*mqlAwsKinesisStream, error) {
	enhancedMonitoring, err := convert.JsonToDictSlice(stream.EnhancedMonitoring)
	if err != nil {
		return nil, err
	}

	streamModeDetails, err := convert.JsonToDict(stream.StreamModeDetails)
	if err != nil {
		return nil, err
	}

	resource, err := CreateResource(runtime, "aws.kinesis.stream",
		map[string]*llx.RawData{
			"__id":                 llx.StringDataPtr(stream.StreamARN),
			"arn":                  llx.StringDataPtr(stream.StreamARN),
			"name":                 llx.StringDataPtr(stream.StreamName),
			"status":               llx.StringData(string(stream.StreamStatus)),
			"encryptionType":       llx.StringData(string(stream.EncryptionType)),
			"keyId":                llx.StringDataPtr(stream.KeyId),
			"retentionPeriodHours": llx.IntDataDefault(stream.RetentionPeriodHours, 0),
			"openShardCount":       llx.IntDataDefault(stream.OpenShardCount, 0),
			"consumerCount":        llx.IntDataDefault(stream.ConsumerCount, 0),
			"streamModeDetails":    llx.DictData(streamModeDetails),
			"enhancedMonitoring":   llx.ArrayData(enhancedMonitoring, types.Any),
			"createdAt":            llx.TimeDataPtr(stream.StreamCreationTimestamp),
			"region":               llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsKinesisStream), nil
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
