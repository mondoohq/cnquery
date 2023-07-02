package aws

import (
	"context"
	"fmt"

	"errors"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (t *mqlAwsCloudtrail) id() (string, error) {
	return "aws.cloudtrail", nil
}

func (t *mqlAwsCloudtrail) GetTrails() ([]interface{}, error) {
	provider, err := awsProvider(t.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(t.getTrails(provider), 5)
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

func (p *mqlAwsCloudtrailTrail) init(args *resources.Args) (*resources.Args, AwsCloudtrailTrail, error) {
	if len(*args) >= 2 {
		return args, nil, nil
	}
	if len(*args) == 0 {
		if ids := getAssetIdentifier(p.MqlResource().MotorRuntime); ids != nil {
			(*args)["name"] = ids.name
			(*args)["arn"] = ids.arn
		}
	}
	if (*args)["arn"] == nil && (*args)["name"] == nil {
		return nil, nil, errors.New("arn or name required to fetch aws cloudtrail trail")
	}

	// construct arn of cloudtrail if missing
	var arn string
	if (*args)["arn"] != nil {
		arn = (*args)["arn"].(string)
	} else {
		nameVal := (*args)["name"].(string)
		arn = fmt.Sprintf(s3ArnPattern, nameVal)
	}
	log.Debug().Str("arn", arn).Msg("init cloudtrail trail with arn")

	// load all s3 buckets
	obj, err := p.MotorRuntime.CreateResource("aws.cloudtrail")
	if err != nil {
		return nil, nil, err
	}
	awsCloudtrail := obj.(AwsCloudtrail)

	rawResources, err := awsCloudtrail.Trails()
	if err != nil {
		return nil, nil, err
	}

	for i := range rawResources {
		trail := rawResources[i].(AwsCloudtrailTrail)
		mqlTrailArn, err := trail.Arn()
		if err != nil {
			return nil, nil, err
		}
		if mqlTrailArn == arn {
			return args, trail, nil
		}
	}
	return args, nil, err
}

func (t *mqlAwsCloudtrail) getTrails(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.Cloudtrail(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			// no pagination required
			trailsResp, err := svc.DescribeTrails(ctx, &cloudtrail.DescribeTrailsInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, errors.Join(err, errors.New("could not gather aws cloudtrail trails"))
			}

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
					// means the key does not exist or we have no access to it
					// dont err out, just assign nil
					if err != nil {
						args = append(args, "kmsKey", nil)
					} else {
						mqlKey := mqlKeyResource.(AwsKmsKey)
						args = append(args, "kmsKey", mqlKey)
					}
				}
				if trail.CloudWatchLogsLogGroupArn != nil {
					mqlLoggroup, err := t.MotorRuntime.CreateResource("aws.cloudwatch.loggroup",
						"arn", core.ToString(trail.CloudWatchLogsLogGroupArn),
					)
					// means the log group does not exist or we have no access to it
					// dont err out, just assign nil
					if err != nil {
						args = append(args, "logGroup", nil)
					} else {
						mqlLog := mqlLoggroup.(AwsCloudwatchLoggroup)
						args = append(args, "logGroup", mqlLog)
					}
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
	provider, err := awsProvider(t.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Cloudtrail(regionValue)
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
	provider, err := awsProvider(t.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Cloudtrail(regionValue)
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
