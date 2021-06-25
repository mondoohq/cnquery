package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

func (t *lumiAwsCloudtrail) id() (string, error) {
	return "aws.cloudtrail", nil
}

func (t *lumiAwsCloudtrail) GetTrails() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(t.getTrails(), 5)
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

func (t *lumiAwsCloudtrail) getTrails() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.Cloudtrail(regionVal)
			ctx := context.Background()

			// no pagination required
			trailsResp, err := svc.DescribeTrails(ctx, &cloudtrail.DescribeTrailsInput{})
			if err != nil {
				return nil, errors.Wrap(err, "could not gather aws cloudtrail trails")
			}

			res := []interface{}{}
			for i := range trailsResp.TrailList {
				trail := trailsResp.TrailList[i]

				// only include trail if this region is the home region for the trail
				// we do this to avoid getting duped results from multiregion trails
				if regionVal != toString(trail.HomeRegion) {
					continue
				}

				args := []interface{}{
					"arn", toString(trail.TrailARN),
					"name", toString(trail.Name),
					"isMultiRegionTrail", toBool(trail.IsMultiRegionTrail),
					"isOrganizationTrail", toBool(trail.IsOrganizationTrail),
					"logFileValidationEnabled", toBool(trail.LogFileValidationEnabled),
					"includeGlobalServiceEvents", toBool(trail.IncludeGlobalServiceEvents),
					"snsTopicARN", toString(trail.SnsTopicARN),
					"cloudWatchLogsRoleArn", toString(trail.CloudWatchLogsRoleArn),
					"region", toString(trail.HomeRegion),
				}

				// trail.S3BucketName
				if trail.S3BucketName != nil {
					lumiAwsS3Bucket, err := t.Runtime.CreateResource("aws.s3.bucket",
						"name", toString(trail.S3BucketName),
					)
					if err != nil {
						return nil, err
					}
					s3Bucket := lumiAwsS3Bucket.(AwsS3Bucket)
					args = append(args, "s3bucket", s3Bucket)

				}

				// add kms key if there is one
				if trail.KmsKeyId != nil {
					lumiKeyResource, err := t.Runtime.CreateResource("aws.kms.key",
						"arn", toString(trail.KmsKeyId),
					)
					if err != nil {
						return nil, err
					}
					lumiKey := lumiKeyResource.(AwsKmsKey)
					args = append(args, "kmsKey", lumiKey)
				}
				if trail.CloudWatchLogsLogGroupArn != nil {
					lumiLoggroup, err := t.Runtime.CreateResource("aws.cloudwatch.loggroup",
						"arn", toString(trail.CloudWatchLogsLogGroupArn),
					)
					if err != nil {
						return nil, err
					}
					lumiLog := lumiLoggroup.(AwsCloudwatchLoggroup)
					args = append(args, "logGroup", lumiLog)
				}

				lumiAwsCloudtrailTrail, err := t.Runtime.CreateResource("aws.cloudtrail.trail", args...)
				if err != nil {
					return nil, err
				}

				res = append(res, lumiAwsCloudtrailTrail)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}
func (s *lumiAwsCloudtrailTrail) GetS3bucket() (interface{}, error) {
	// no s3 bucket on the trail object
	return nil, nil
}

func (s *lumiAwsCloudtrailTrail) GetLogGroup() (interface{}, error) {
	// no log group on the trail object
	return nil, nil
}

func (s *lumiAwsCloudtrailTrail) GetKmsKey() (interface{}, error) {
	// no key id on the trail object
	return nil, nil
}

func (t *lumiAwsCloudtrailTrail) id() (string, error) {
	return t.Arn()
}

func (t *lumiAwsCloudtrailTrail) GetStatus() (interface{}, error) {
	regionValue, err := t.Region()
	if err != nil {
		return nil, err
	}
	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.Cloudtrail(regionValue)
	ctx := context.Background()

	arnValue, err := t.Arn()
	if err != nil {
		return nil, err
	}

	// no pagination required
	trailstatus, err := svc.GetTrailStatus(ctx, &cloudtrail.GetTrailStatusInput{
		Name: &arnValue,
	})
	if err != nil {
		return nil, err
	}

	return jsonToDict(trailstatus)
}

func (t *lumiAwsCloudtrailTrail) GetEventSelectors() (interface{}, error) {
	regionValue, err := t.Region()
	if err != nil {
		return nil, err
	}
	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.Cloudtrail(regionValue)
	ctx := context.Background()

	arnValue, err := t.Arn()
	if err != nil {
		return nil, err
	}

	// no pagination required
	trailmgmtevents, err := svc.GetEventSelectors(ctx, &cloudtrail.GetEventSelectorsInput{
		TrailName: &arnValue,
	})
	if err != nil {
		return nil, err
	}
	return jsonToDictSlice(trailmgmtevents.EventSelectors)
}
