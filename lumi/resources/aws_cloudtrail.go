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

		lumiAwsCloudtrailTrail, err := t.Runtime.CreateResource("aws.cloudtrail.trail",
			"Arn", toString(trail.TrailARN),
			"Name", toString(trail.Name),
			"IsMultiRegionTrail", toBool(trail.IsMultiRegionTrail),
			"IsOrganizationTrail", toBool(trail.IsOrganizationTrail),
			"LogFileValidationEnabled", toBool(trail.LogFileValidationEnabled),
			"IncludeGlobalServiceEvents", toBool(trail.IncludeGlobalServiceEvents),
			"S3BucketName", toString(trail.S3BucketName),
			"SnsTopicARN", toString(trail.SnsTopicARN),
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
		"Arn", arnValue,
		"IsLogging", toBool(trailstatus.IsLogging),
		"LatestCloudWatchLogsDeliveryError", toString(trailstatus.LatestCloudWatchLogsDeliveryError),
		"LatestCloudWatchLogsDeliveryTime", toTime(trailstatus.LatestCloudWatchLogsDeliveryTime),
		"LatestDeliveryError", toString(trailstatus.LatestDeliveryError),
		"LatestDeliveryTime", toTime(trailstatus.LatestDeliveryTime),
		"LatestDigestDeliveryError", toString(trailstatus.LatestDigestDeliveryError),
		"LatestDigestDeliveryTime", toTime(trailstatus.LatestDigestDeliveryTime),
		"LatestNotificationError", toString(trailstatus.LatestNotificationError),
		"LatestNotificationTime", toTime(trailstatus.LatestNotificationTime),
		"StartLoggingTime", toTime(trailstatus.StartLoggingTime),
		"StopLoggingTime", toTime(trailstatus.StopLoggingTime),
	)
	if err != nil {
		return nil, err
	}
	return lumiAwsCloudtrailTrailStatus, nil
}

func (t *lumiAwsCloudtrailTrailstatus) id() (string, error) {
	return t.Arn()
}
