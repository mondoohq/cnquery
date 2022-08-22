package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/resources/library/jobpool"
	aws_transport "go.mondoo.io/mondoo/motor/providers/aws"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (t *mqlAwsCloudtrail) id() (string, error) {
	return "aws.cloudtrail", nil
}

func (t *mqlAwsCloudtrail) GetTrails() ([]interface{}, error) {
	at, err := awstransport(t.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(t.getTrails(at), 5)
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

func (t *mqlAwsCloudtrail) getTrails(at *aws_transport.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
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
				if regionVal != core.ToString(trail.HomeRegion) {
					continue
				}

				args := []interface{}{
					"arn", core.ToString(trail.TrailARN),
					"name", core.ToString(trail.Name),
					"isMultiRegionTrail", core.ToBool(trail.IsMultiRegionTrail),
					"isOrganizationTrail", core.ToBool(trail.IsOrganizationTrail),
					"logFileValidationEnabled", core.ToBool(trail.LogFileValidationEnabled),
					"includeGlobalServiceEvents", core.ToBool(trail.IncludeGlobalServiceEvents),
					"snsTopicARN", core.ToString(trail.SnsTopicARN),
					"cloudWatchLogsRoleArn", core.ToString(trail.CloudWatchLogsRoleArn),
					"region", core.ToString(trail.HomeRegion),
				}

				// trail.S3BucketName
				if trail.S3BucketName != nil {
					mqlAwsS3Bucket, err := t.MotorRuntime.CreateResource("aws.s3.bucket",
						"name", core.ToString(trail.S3BucketName),
					)
					if err != nil {
						return nil, err
					}
					s3Bucket := mqlAwsS3Bucket.(AwsS3Bucket)
					args = append(args, "s3bucket", s3Bucket)

				}

				// add kms key if there is one
				if trail.KmsKeyId != nil {
					mqlKeyResource, err := t.MotorRuntime.CreateResource("aws.kms.key",
						"arn", core.ToString(trail.KmsKeyId),
					)
					if err != nil {
						return nil, err
					}
					mqlKey := mqlKeyResource.(AwsKmsKey)
					args = append(args, "kmsKey", mqlKey)
				}
				if trail.CloudWatchLogsLogGroupArn != nil {
					mqlLoggroup, err := t.MotorRuntime.CreateResource("aws.cloudwatch.loggroup",
						"arn", core.ToString(trail.CloudWatchLogsLogGroupArn),
					)
					if err != nil {
						return nil, err
					}
					mqlLog := mqlLoggroup.(AwsCloudwatchLoggroup)
					args = append(args, "logGroup", mqlLog)
				}

				mqlAwsCloudtrailTrail, err := t.MotorRuntime.CreateResource("aws.cloudtrail.trail", args...)
				if err != nil {
					return nil, err
				}

				res = append(res, mqlAwsCloudtrailTrail)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (s *mqlAwsCloudtrailTrail) GetS3bucket() (interface{}, error) {
	// no s3 bucket on the trail object
	return nil, nil
}

func (s *mqlAwsCloudtrailTrail) GetLogGroup() (interface{}, error) {
	// no log group on the trail object
	return nil, nil
}

func (s *mqlAwsCloudtrailTrail) GetKmsKey() (interface{}, error) {
	// no key id on the trail object
	return nil, nil
}

func (t *mqlAwsCloudtrailTrail) id() (string, error) {
	return t.Arn()
}

func (t *mqlAwsCloudtrailTrail) GetStatus() (interface{}, error) {
	regionValue, err := t.Region()
	if err != nil {
		return nil, err
	}
	at, err := awstransport(t.MotorRuntime.Motor.Provider)
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

	return core.JsonToDict(trailstatus)
}

func (t *mqlAwsCloudtrailTrail) GetEventSelectors() (interface{}, error) {
	regionValue, err := t.Region()
	if err != nil {
		return nil, err
	}
	at, err := awstransport(t.MotorRuntime.Motor.Provider)
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
	return core.JsonToDictSlice(trailmgmtevents.EventSelectors)
}
