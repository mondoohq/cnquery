package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/cockroachdb/errors"
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
