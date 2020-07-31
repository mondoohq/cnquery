package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/pkg/errors"
)

func cloudtrailClient() *cloudtrail.Client {
	// TODO: cfg needs to come from the transport
	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile("mondoo-inc"))
	// cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic(err)
	}
	cfg.Region = endpoints.UsEast1RegionID

	// iterate over each region?
	svc := cloudtrail.New(cfg)
	return svc
}

func (t *lumiAwsCloudtrail) id() (string, error) {
	return "aws.cloudtrail", nil
}

func (t *lumiAwsCloudtrail) GetTrails() ([]interface{}, error) {
	svc := cloudtrailClient()
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
			"isMultiRegionTrail", toBool(trail.IsMultiRegionTrail),
			"isOrganizationTrail", toBool(trail.IsOrganizationTrail),
			"logFileValidationEnabled", toBool(trail.LogFileValidationEnabled),
			"includeGlobalServiceEvents", toBool(trail.IncludeGlobalServiceEvents),
			"s3Bucket", s3Bucket,
			"snsTopicARN", toString(trail.SnsTopicARN),
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
	arnValue, err := t.Arn()
	if err != nil {
		return nil, err
	}

	svc := cloudtrailClient()
	ctx := context.Background()
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
		"latestCloudWatchLogsDeliveryTime", toTime(trailstatus.LatestCloudWatchLogsDeliveryTime),
		"latestDeliveryError", toString(trailstatus.LatestDeliveryError),
		"latestDeliveryTime", toTime(trailstatus.LatestDeliveryTime),
		"latestDigestDeliveryError", toString(trailstatus.LatestDigestDeliveryError),
		"latestDigestDeliveryTime", toTime(trailstatus.LatestDigestDeliveryTime),
		"latestNotificationError", toString(trailstatus.LatestNotificationError),
		"latestNotificationTime", toTime(trailstatus.LatestNotificationTime),
		"startLoggingTime", toTime(trailstatus.StartLoggingTime),
		"stopLoggingTime", toTime(trailstatus.StopLoggingTime),
	)
	if err != nil {
		return nil, err
	}
	return lumiAwsCloudtrailTrailStatus, nil
}

func (t *lumiAwsCloudtrailTrailstatus) id() (string, error) {
	return t.Arn()
}
