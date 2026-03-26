// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/firehose"
	firehose_types "github.com/aws/aws-sdk-go-v2/service/firehose/types"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	kinesis_types "github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	kmsSDK "github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
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
					mqlStream, err := newMqlAwsKinesisStream(a.MqlRuntime, region, &streamSummary)
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

func newMqlAwsKinesisStream(runtime *plugin.Runtime, region string, summary *kinesis_types.StreamSummary) (*mqlAwsKinesisStream, error) {
	// Use fields available from ListStreams StreamSummary
	streamModeDetails, err := convert.JsonToDict(summary.StreamModeDetails)
	if err != nil {
		return nil, err
	}

	resource, err := CreateResource(runtime, "aws.kinesis.stream",
		map[string]*llx.RawData{
			"__id":              llx.StringDataPtr(summary.StreamARN),
			"arn":               llx.StringDataPtr(summary.StreamARN),
			"name":              llx.StringDataPtr(summary.StreamName),
			"status":            llx.StringData(string(summary.StreamStatus)),
			"streamModeDetails": llx.DictData(streamModeDetails),
			"createdAt":         llx.TimeDataPtr(summary.StreamCreationTimestamp),
			"region":            llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsKinesisStream), nil
}

type mqlAwsKinesisStreamInternal struct {
	fetched          bool
	cachedEncType    string
	cachedKeyId      string
	cachedRetention  int64
	cachedOpenShards int64
	cachedConsumers  int64
	cachedEnhMonitor []any
	lock             sync.Mutex
}

func (a *mqlAwsKinesisStream) fetchStreamDetails() error {
	if a.fetched {
		return nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Kinesis(a.Region.Data)
	ctx := context.Background()

	arnVal := a.Arn.Data
	descResp, err := svc.DescribeStreamSummary(ctx, &kinesis.DescribeStreamSummaryInput{
		StreamARN: &arnVal,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			log.Warn().Str("stream", arnVal).Msg("access denied describing kinesis stream, using defaults")
			a.fetched = true
			return nil
		}
		return err
	}
	if descResp.StreamDescriptionSummary != nil {
		desc := descResp.StreamDescriptionSummary
		a.cachedEncType = string(desc.EncryptionType)
		if desc.KeyId != nil {
			a.cachedKeyId = *desc.KeyId
		}
		if desc.RetentionPeriodHours != nil {
			a.cachedRetention = int64(*desc.RetentionPeriodHours)
		}
		if desc.OpenShardCount != nil {
			a.cachedOpenShards = int64(*desc.OpenShardCount)
		}
		if desc.ConsumerCount != nil {
			a.cachedConsumers = int64(*desc.ConsumerCount)
		}
		var err2 error
		a.cachedEnhMonitor, err2 = convert.JsonToDictSlice(desc.EnhancedMonitoring)
		if err2 != nil {
			return err2
		}
	}
	a.fetched = true
	return nil
}

func (a *mqlAwsKinesisStream) encryptionType() (string, error) {
	if err := a.fetchStreamDetails(); err != nil {
		return "", err
	}
	return a.cachedEncType, nil
}

func (a *mqlAwsKinesisStream) keyId() (string, error) {
	if err := a.fetchStreamDetails(); err != nil {
		return "", err
	}
	return a.cachedKeyId, nil
}

func (a *mqlAwsKinesisStream) kmsKey() (*mqlAwsKmsKey, error) {
	if err := a.fetchStreamDetails(); err != nil {
		return nil, err
	}
	if a.cachedKeyId == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	keyArn := a.cachedKeyId
	if strings.HasPrefix(keyArn, "arn:") {
		// Already an ARN (handles all partitions: aws, aws-cn, aws-us-gov)
	} else if strings.HasPrefix(keyArn, "alias/") {
		// Resolve alias to key ARN via DescribeKey
		svc := conn.Kms(a.Region.Data)
		resp, err := svc.DescribeKey(context.Background(), &kmsSDK.DescribeKeyInput{KeyId: &keyArn})
		if err != nil {
			return nil, err
		}
		keyArn = *resp.KeyMetadata.Arn
	} else {
		// Assume raw key ID, construct ARN
		keyArn = fmt.Sprintf(kmsKeyArnPattern, a.Region.Data, conn.AccountId(), keyArn)
	}

	mqlKey, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{
			"arn": llx.StringData(keyArn),
		})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsKinesisStream) retentionPeriodHours() (int64, error) {
	if err := a.fetchStreamDetails(); err != nil {
		return 0, err
	}
	return a.cachedRetention, nil
}

func (a *mqlAwsKinesisStream) openShardCount() (int64, error) {
	if err := a.fetchStreamDetails(); err != nil {
		return 0, err
	}
	return a.cachedOpenShards, nil
}

func (a *mqlAwsKinesisStream) consumerCount() (int64, error) {
	if err := a.fetchStreamDetails(); err != nil {
		return 0, err
	}
	return a.cachedConsumers, nil
}

func (a *mqlAwsKinesisStream) enhancedMonitoring() ([]any, error) {
	if err := a.fetchStreamDetails(); err != nil {
		return nil, err
	}
	return a.cachedEnhMonitor, nil
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

func (a *mqlAwsKinesisStream) tags() (map[string]any, error) {
	arn := a.Arn.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Kinesis(region)
	ctx := context.Background()

	tags := make(map[string]any)
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

type mqlAwsKinesisFirehoseDeliveryStreamInternal struct {
	cacheDestinations []firehose_types.DestinationDescription
	cacheRegion       string
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

	resource, err := CreateResource(runtime, "aws.kinesis.firehoseDeliveryStream",
		map[string]*llx.RawData{
			"__id":               llx.StringDataPtr(stream.DeliveryStreamARN),
			"arn":                llx.StringDataPtr(stream.DeliveryStreamARN),
			"name":               llx.StringDataPtr(stream.DeliveryStreamName),
			"status":             llx.StringData(string(stream.DeliveryStreamStatus)),
			"deliveryStreamType": llx.StringData(string(stream.DeliveryStreamType)),
			"encryption":         llx.DictData(encryption),
			"source":             llx.DictData(source),
			"createdAt":          llx.TimeDataPtr(stream.CreateTimestamp),
			"region":             llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	mqlStream := resource.(*mqlAwsKinesisFirehoseDeliveryStream)
	mqlStream.cacheDestinations = stream.Destinations
	mqlStream.cacheRegion = region
	return mqlStream, nil
}

func (a *mqlAwsKinesisFirehoseDeliveryStream) destinations() ([]any, error) {
	res := []any{}
	streamArn := a.Arn.Data
	region := a.cacheRegion

	for i := range a.cacheDestinations {
		dest := a.cacheDestinations[i]
		destId := convert.ToValue(dest.DestinationId)

		// Determine destination type
		destType := "unknown"
		if dest.ExtendedS3DestinationDescription != nil {
			destType = "s3"
		} else if dest.RedshiftDestinationDescription != nil {
			destType = "redshift"
		} else if dest.AmazonopensearchserviceDestinationDescription != nil {
			destType = "opensearch"
		} else if dest.ElasticsearchDestinationDescription != nil {
			destType = "elasticsearch"
		} else if dest.SplunkDestinationDescription != nil {
			destType = "splunk"
		} else if dest.HttpEndpointDestinationDescription != nil {
			destType = "httpEndpoint"
		}

		mqlDest, err := CreateResource(a.MqlRuntime, "aws.kinesis.firehoseDeliveryStream.destination",
			map[string]*llx.RawData{
				"__id":          llx.StringData(streamArn + "/destination/" + destId),
				"destinationId": llx.StringData(destId),
				"type":          llx.StringData(destType),
				"region":        llx.StringData(region),
			})
		if err != nil {
			return nil, err
		}
		mqlDest.(*mqlAwsKinesisFirehoseDeliveryStreamDestination).cacheDest = &dest
		mqlDest.(*mqlAwsKinesisFirehoseDeliveryStreamDestination).cacheStreamArn = streamArn
		res = append(res, mqlDest)
	}
	return res, nil
}

type mqlAwsKinesisFirehoseDeliveryStreamDestinationInternal struct {
	cacheDest      *firehose_types.DestinationDescription
	cacheStreamArn string
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestination) id() (string, error) {
	return a.cacheStreamArn + "/destination/" + a.DestinationId.Data, nil
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestination) s3() (*mqlAwsKinesisFirehoseDeliveryStreamDestinationS3, error) {
	if a.cacheDest == nil || a.cacheDest.ExtendedS3DestinationDescription == nil {
		a.S3.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	s := a.cacheDest.ExtendedS3DestinationDescription
	streamArn := a.cacheStreamArn
	destId := a.DestinationId.Data
	region := a.Region.Data

	var bufSize, bufInterval int64
	if s.BufferingHints != nil {
		if s.BufferingHints.SizeInMBs != nil {
			bufSize = int64(*s.BufferingHints.SizeInMBs)
		}
		if s.BufferingHints.IntervalInSeconds != nil {
			bufInterval = int64(*s.BufferingHints.IntervalInSeconds)
		}
	}

	s3Id := streamArn + "/destination/" + destId + "/s3"
	cwLogging, err := newMqlCloudWatchLogging(a.MqlRuntime, s3Id, s.CloudWatchLoggingOptions)
	if err != nil {
		return nil, err
	}
	dfConversion, _ := convert.JsonToDict(s.DataFormatConversionConfiguration)
	dynPartitioning, _ := convert.JsonToDict(s.DynamicPartitioningConfiguration)
	processingConfig, _ := convert.JsonToDict(s.ProcessingConfiguration)

	mqlS3, err := CreateResource(a.MqlRuntime, "aws.kinesis.firehoseDeliveryStream.destination.s3",
		map[string]*llx.RawData{
			"__id":                       llx.StringData(s3Id),
			"bucketArn":                  llx.StringDataPtr(s.BucketARN),
			"compressionFormat":          llx.StringData(string(s.CompressionFormat)),
			"prefix":                     llx.StringDataPtr(s.Prefix),
			"errorOutputPrefix":          llx.StringDataPtr(s.ErrorOutputPrefix),
			"fileExtension":              llx.StringDataPtr(s.FileExtension),
			"bufferingSizeInMBs":         llx.IntData(bufSize),
			"bufferingIntervalInSeconds": llx.IntData(bufInterval),
			"s3BackupMode":               llx.StringData(string(s.S3BackupMode)),
			"cloudWatchLogging":          llx.ResourceData(cwLogging, "aws.kinesis.firehoseDeliveryStream.cloudWatchLogging"),
			"dataFormatConversion":       llx.DictData(dfConversion),
			"dynamicPartitioning":        llx.DictData(dynPartitioning),
			"processingConfiguration":    llx.DictData(processingConfig),
			"region":                     llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	mqlS3Res := mqlS3.(*mqlAwsKinesisFirehoseDeliveryStreamDestinationS3)
	mqlS3Res.cacheRoleArn = s.RoleARN
	return mqlS3Res, nil
}

type mqlAwsKinesisFirehoseDeliveryStreamDestinationS3Internal struct {
	cacheRoleArn *string
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestinationS3) iamRole() (*mqlAwsIamRole, error) {
	if a.cacheRoleArn == nil || *a.cacheRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestinationS3) id() (string, error) {
	return a.BucketArn.Data + "/" + a.Region.Data, nil
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestination) redshift() (*mqlAwsKinesisFirehoseDeliveryStreamDestinationRedshift, error) {
	if a.cacheDest == nil || a.cacheDest.RedshiftDestinationDescription == nil {
		a.Redshift.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	r := a.cacheDest.RedshiftDestinationDescription
	streamArn := a.cacheStreamArn
	destId := a.DestinationId.Data
	region := a.Region.Data

	var tableName, copyOptions string
	if r.CopyCommand != nil {
		tableName = convert.ToValue(r.CopyCommand.DataTableName)
		copyOptions = convert.ToValue(r.CopyCommand.CopyOptions)
	}

	var retryDuration int64
	if r.RetryOptions != nil && r.RetryOptions.DurationInSeconds != nil {
		retryDuration = int64(*r.RetryOptions.DurationInSeconds)
	}

	redshiftId := streamArn + "/destination/" + destId + "/redshift"
	cwLogging, err := newMqlCloudWatchLogging(a.MqlRuntime, redshiftId, r.CloudWatchLoggingOptions)
	if err != nil {
		return nil, err
	}
	processingConfig, _ := convert.JsonToDict(r.ProcessingConfiguration)

	mqlR, err := CreateResource(a.MqlRuntime, "aws.kinesis.firehoseDeliveryStream.destination.redshift",
		map[string]*llx.RawData{
			"__id":                    llx.StringData(redshiftId),
			"clusterJdbcUrl":          llx.StringDataPtr(r.ClusterJDBCURL),
			"username":                llx.StringDataPtr(r.Username),
			"copyCommandTableName":    llx.StringData(tableName),
			"copyCommandOptions":      llx.StringData(copyOptions),
			"s3BackupMode":            llx.StringData(string(r.S3BackupMode)),
			"cloudWatchLogging":       llx.ResourceData(cwLogging, "aws.kinesis.firehoseDeliveryStream.cloudWatchLogging"),
			"processingConfiguration": llx.DictData(processingConfig),
			"retryDurationInSeconds":  llx.IntData(retryDuration),
			"region":                  llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	mqlRRes := mqlR.(*mqlAwsKinesisFirehoseDeliveryStreamDestinationRedshift)
	mqlRRes.cacheRoleArn = r.RoleARN
	return mqlRRes, nil
}

type mqlAwsKinesisFirehoseDeliveryStreamDestinationRedshiftInternal struct {
	cacheRoleArn *string
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestinationRedshift) iamRole() (*mqlAwsIamRole, error) {
	if a.cacheRoleArn == nil || *a.cacheRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestinationRedshift) id() (string, error) {
	return a.ClusterJdbcUrl.Data + "/" + a.Region.Data, nil
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestination) elasticsearch() (*mqlAwsKinesisFirehoseDeliveryStreamDestinationElasticsearch, error) {
	if a.cacheDest == nil || a.cacheDest.ElasticsearchDestinationDescription == nil {
		a.Elasticsearch.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	e := a.cacheDest.ElasticsearchDestinationDescription
	streamArn := a.cacheStreamArn
	destId := a.DestinationId.Data
	region := a.Region.Data

	var bufSize, bufInterval int64
	if e.BufferingHints != nil {
		if e.BufferingHints.SizeInMBs != nil {
			bufSize = int64(*e.BufferingHints.SizeInMBs)
		}
		if e.BufferingHints.IntervalInSeconds != nil {
			bufInterval = int64(*e.BufferingHints.IntervalInSeconds)
		}
	}

	var retryDuration int64
	if e.RetryOptions != nil && e.RetryOptions.DurationInSeconds != nil {
		retryDuration = int64(*e.RetryOptions.DurationInSeconds)
	}

	esId := streamArn + "/destination/" + destId + "/elasticsearch"
	cwLogging, err := newMqlCloudWatchLogging(a.MqlRuntime, esId, e.CloudWatchLoggingOptions)
	if err != nil {
		return nil, err
	}
	processingConfig, _ := convert.JsonToDict(e.ProcessingConfiguration)

	mqlE, err := CreateResource(a.MqlRuntime, "aws.kinesis.firehoseDeliveryStream.destination.elasticsearch",
		map[string]*llx.RawData{
			"__id":                       llx.StringData(esId),
			"domainArn":                  llx.StringDataPtr(e.DomainARN),
			"clusterEndpoint":            llx.StringDataPtr(e.ClusterEndpoint),
			"indexName":                  llx.StringDataPtr(e.IndexName),
			"indexRotationPeriod":        llx.StringData(string(e.IndexRotationPeriod)),
			"s3BackupMode":               llx.StringData(string(e.S3BackupMode)),
			"bufferingSizeInMBs":         llx.IntData(bufSize),
			"bufferingIntervalInSeconds": llx.IntData(bufInterval),
			"cloudWatchLogging":          llx.ResourceData(cwLogging, "aws.kinesis.firehoseDeliveryStream.cloudWatchLogging"),
			"processingConfiguration":    llx.DictData(processingConfig),
			"retryDurationInSeconds":     llx.IntData(retryDuration),
			"region":                     llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	mqlERes := mqlE.(*mqlAwsKinesisFirehoseDeliveryStreamDestinationElasticsearch)
	mqlERes.cacheRoleArn = e.RoleARN
	return mqlERes, nil
}

type mqlAwsKinesisFirehoseDeliveryStreamDestinationElasticsearchInternal struct {
	cacheRoleArn *string
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestinationElasticsearch) iamRole() (*mqlAwsIamRole, error) {
	if a.cacheRoleArn == nil || *a.cacheRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestinationElasticsearch) id() (string, error) {
	return a.DomainArn.Data + "/" + a.IndexName.Data, nil
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestination) opensearch() (*mqlAwsKinesisFirehoseDeliveryStreamDestinationOpensearch, error) {
	if a.cacheDest == nil || a.cacheDest.AmazonopensearchserviceDestinationDescription == nil {
		a.Opensearch.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	o := a.cacheDest.AmazonopensearchserviceDestinationDescription
	streamArn := a.cacheStreamArn
	destId := a.DestinationId.Data
	region := a.Region.Data

	var bufSize, bufInterval int64
	if o.BufferingHints != nil {
		if o.BufferingHints.SizeInMBs != nil {
			bufSize = int64(*o.BufferingHints.SizeInMBs)
		}
		if o.BufferingHints.IntervalInSeconds != nil {
			bufInterval = int64(*o.BufferingHints.IntervalInSeconds)
		}
	}

	var retryDuration int64
	if o.RetryOptions != nil && o.RetryOptions.DurationInSeconds != nil {
		retryDuration = int64(*o.RetryOptions.DurationInSeconds)
	}

	osId := streamArn + "/destination/" + destId + "/opensearch"
	cwLogging, err := newMqlCloudWatchLogging(a.MqlRuntime, osId, o.CloudWatchLoggingOptions)
	if err != nil {
		return nil, err
	}
	processingConfig, _ := convert.JsonToDict(o.ProcessingConfiguration)

	mqlO, err := CreateResource(a.MqlRuntime, "aws.kinesis.firehoseDeliveryStream.destination.opensearch",
		map[string]*llx.RawData{
			"__id":                       llx.StringData(osId),
			"domainArn":                  llx.StringDataPtr(o.DomainARN),
			"clusterEndpoint":            llx.StringDataPtr(o.ClusterEndpoint),
			"indexName":                  llx.StringDataPtr(o.IndexName),
			"indexRotationPeriod":        llx.StringData(string(o.IndexRotationPeriod)),
			"s3BackupMode":               llx.StringData(string(o.S3BackupMode)),
			"bufferingSizeInMBs":         llx.IntData(bufSize),
			"bufferingIntervalInSeconds": llx.IntData(bufInterval),
			"cloudWatchLogging":          llx.ResourceData(cwLogging, "aws.kinesis.firehoseDeliveryStream.cloudWatchLogging"),
			"processingConfiguration":    llx.DictData(processingConfig),
			"retryDurationInSeconds":     llx.IntData(retryDuration),
			"region":                     llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	mqlORes := mqlO.(*mqlAwsKinesisFirehoseDeliveryStreamDestinationOpensearch)
	mqlORes.cacheRoleArn = o.RoleARN
	return mqlORes, nil
}

type mqlAwsKinesisFirehoseDeliveryStreamDestinationOpensearchInternal struct {
	cacheRoleArn *string
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestinationOpensearch) iamRole() (*mqlAwsIamRole, error) {
	if a.cacheRoleArn == nil || *a.cacheRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestinationOpensearch) id() (string, error) {
	return a.DomainArn.Data + "/" + a.IndexName.Data, nil
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestination) splunk() (*mqlAwsKinesisFirehoseDeliveryStreamDestinationSplunk, error) {
	if a.cacheDest == nil || a.cacheDest.SplunkDestinationDescription == nil {
		a.Splunk.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	sp := a.cacheDest.SplunkDestinationDescription
	streamArn := a.cacheStreamArn
	destId := a.DestinationId.Data
	region := a.Region.Data

	var bufSize, bufInterval int64
	if sp.BufferingHints != nil {
		if sp.BufferingHints.SizeInMBs != nil {
			bufSize = int64(*sp.BufferingHints.SizeInMBs)
		}
		if sp.BufferingHints.IntervalInSeconds != nil {
			bufInterval = int64(*sp.BufferingHints.IntervalInSeconds)
		}
	}

	var hecTimeout int64
	if sp.HECAcknowledgmentTimeoutInSeconds != nil {
		hecTimeout = int64(*sp.HECAcknowledgmentTimeoutInSeconds)
	}

	var retryDuration int64
	if sp.RetryOptions != nil && sp.RetryOptions.DurationInSeconds != nil {
		retryDuration = int64(*sp.RetryOptions.DurationInSeconds)
	}

	splunkId := streamArn + "/destination/" + destId + "/splunk"
	cwLogging, err := newMqlCloudWatchLogging(a.MqlRuntime, splunkId, sp.CloudWatchLoggingOptions)
	if err != nil {
		return nil, err
	}
	processingConfig, _ := convert.JsonToDict(sp.ProcessingConfiguration)

	mqlSp, err := CreateResource(a.MqlRuntime, "aws.kinesis.firehoseDeliveryStream.destination.splunk",
		map[string]*llx.RawData{
			"__id":                              llx.StringData(splunkId),
			"hecEndpoint":                       llx.StringDataPtr(sp.HECEndpoint),
			"hecEndpointType":                   llx.StringData(string(sp.HECEndpointType)),
			"hecAcknowledgmentTimeoutInSeconds": llx.IntData(hecTimeout),
			"s3BackupMode":                      llx.StringData(string(sp.S3BackupMode)),
			"bufferingSizeInMBs":                llx.IntData(bufSize),
			"bufferingIntervalInSeconds":        llx.IntData(bufInterval),
			"cloudWatchLogging":                 llx.ResourceData(cwLogging, "aws.kinesis.firehoseDeliveryStream.cloudWatchLogging"),
			"processingConfiguration":           llx.DictData(processingConfig),
			"retryDurationInSeconds":            llx.IntData(retryDuration),
			"region":                            llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	return mqlSp.(*mqlAwsKinesisFirehoseDeliveryStreamDestinationSplunk), nil
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestinationSplunk) id() (string, error) {
	return a.HecEndpoint.Data + "/" + a.Region.Data, nil
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestination) httpEndpoint() (*mqlAwsKinesisFirehoseDeliveryStreamDestinationHttpEndpoint, error) {
	if a.cacheDest == nil || a.cacheDest.HttpEndpointDestinationDescription == nil {
		a.HttpEndpoint.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	h := a.cacheDest.HttpEndpointDestinationDescription
	streamArn := a.cacheStreamArn
	destId := a.DestinationId.Data
	region := a.Region.Data

	var endpointUrl, endpointName string
	if h.EndpointConfiguration != nil {
		endpointUrl = convert.ToValue(h.EndpointConfiguration.Url)
		endpointName = convert.ToValue(h.EndpointConfiguration.Name)
	}

	var bufSize, bufInterval int64
	if h.BufferingHints != nil {
		if h.BufferingHints.SizeInMBs != nil {
			bufSize = int64(*h.BufferingHints.SizeInMBs)
		}
		if h.BufferingHints.IntervalInSeconds != nil {
			bufInterval = int64(*h.BufferingHints.IntervalInSeconds)
		}
	}

	var retryDuration int64
	if h.RetryOptions != nil && h.RetryOptions.DurationInSeconds != nil {
		retryDuration = int64(*h.RetryOptions.DurationInSeconds)
	}

	httpId := streamArn + "/destination/" + destId + "/httpEndpoint"
	cwLogging, err := newMqlCloudWatchLogging(a.MqlRuntime, httpId, h.CloudWatchLoggingOptions)
	if err != nil {
		return nil, err
	}
	processingConfig, _ := convert.JsonToDict(h.ProcessingConfiguration)
	requestConfig, _ := convert.JsonToDict(h.RequestConfiguration)

	mqlH, err := CreateResource(a.MqlRuntime, "aws.kinesis.firehoseDeliveryStream.destination.httpEndpoint",
		map[string]*llx.RawData{
			"__id":                       llx.StringData(httpId),
			"endpointUrl":                llx.StringData(endpointUrl),
			"endpointName":               llx.StringData(endpointName),
			"s3BackupMode":               llx.StringData(string(h.S3BackupMode)),
			"bufferingSizeInMBs":         llx.IntData(bufSize),
			"bufferingIntervalInSeconds": llx.IntData(bufInterval),
			"cloudWatchLogging":          llx.ResourceData(cwLogging, "aws.kinesis.firehoseDeliveryStream.cloudWatchLogging"),
			"processingConfiguration":    llx.DictData(processingConfig),
			"requestConfiguration":       llx.DictData(requestConfig),
			"retryDurationInSeconds":     llx.IntData(retryDuration),
			"region":                     llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	mqlHRes := mqlH.(*mqlAwsKinesisFirehoseDeliveryStreamDestinationHttpEndpoint)
	mqlHRes.cacheRoleArn = h.RoleARN
	return mqlHRes, nil
}

type mqlAwsKinesisFirehoseDeliveryStreamDestinationHttpEndpointInternal struct {
	cacheRoleArn *string
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestinationHttpEndpoint) iamRole() (*mqlAwsIamRole, error) {
	if a.cacheRoleArn == nil || *a.cacheRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamDestinationHttpEndpoint) id() (string, error) {
	return a.EndpointUrl.Data + "/" + a.Region.Data, nil
}

func newMqlCloudWatchLogging(runtime *plugin.Runtime, parentId string, opts *firehose_types.CloudWatchLoggingOptions) (plugin.Resource, error) {
	var enabled bool
	var logGroupName, logStreamName string
	if opts != nil {
		if opts.Enabled != nil {
			enabled = *opts.Enabled
		}
		if opts.LogGroupName != nil {
			logGroupName = *opts.LogGroupName
		}
		if opts.LogStreamName != nil {
			logStreamName = *opts.LogStreamName
		}
	}
	return CreateResource(runtime, "aws.kinesis.firehoseDeliveryStream.cloudWatchLogging",
		map[string]*llx.RawData{
			"__id":          llx.StringData(parentId + "/cloudWatchLogging"),
			"enabled":       llx.BoolData(enabled),
			"logGroupName":  llx.StringData(logGroupName),
			"logStreamName": llx.StringData(logStreamName),
		})
}

func (a *mqlAwsKinesisFirehoseDeliveryStreamCloudWatchLogging) id() (string, error) {
	return a.LogGroupName.Data + "/" + a.LogStreamName.Data, nil
}

func (a *mqlAwsKinesisFirehoseDeliveryStream) tags() (map[string]any, error) {
	name := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Firehose(region)
	ctx := context.Background()

	tags := make(map[string]any)
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
