package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/aws/connection"
	"go.mondoo.com/cnquery/providers/aws/resources/jobpool"
)

func (a *mqlAwsCloudtrail) id() (string, error) {
	return "aws.cloudtrail", nil
}

func (a *mqlAwsCloudtrail) trails() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getTrails(conn), 5)
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

func initAwsCloudtrailTrail(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) >= 2 {
		return args, nil, nil
	}
	// if len(*args) == 0 {
	// 	if ids := getAssetIdentifier(p.MqlResource().MotorRuntime); ids != nil {
	// 		args["name"] = ids.name
	// 		args["arn"] = ids.arn
	// 	}
	// }
	if args["arn"] == nil && args["name"] == nil {
		return nil, nil, errors.New("arn or name required to fetch aws cloudtrail trail")
	}

	// construct arn of cloudtrail if missing
	var arn string
	if args["arn"] != nil {
		arn = args["arn"].Value.(string)
	} else {
		nameVal := args["name"].Value.(string)
		arn = fmt.Sprintf(s3ArnPattern, nameVal)
	}
	log.Debug().Str("arn", arn).Msg("init cloudtrail trail with arn")

	// load all s3 buckets
	obj, err := runtime.CreateResource(runtime, "aws.cloudtrail", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	awsCloudtrail := obj.(*mqlAwsCloudtrail)

	rawResources := awsCloudtrail.Trails.Data

	for i := range rawResources {
		trail := rawResources[i].(*mqlAwsCloudtrailTrail)
		if trail.Arn.Data == arn {
			return args, trail, nil
		}
	}
	return args, nil, err
}

func (a *mqlAwsCloudtrail) getTrails(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := conn.Cloudtrail(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			// no pagination required
			trailsResp, err := svc.DescribeTrails(ctx, &cloudtrail.DescribeTrailsInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, errors.Wrap(err, "could not gather aws cloudtrail trails")
			}

			for i := range trailsResp.TrailList {
				trail := trailsResp.TrailList[i]

				// only include trail if this region is the home region for the trail
				// we do this to avoid getting duped results from multiregion trails
				if regionVal != toString(trail.HomeRegion) {
					continue
				}
				args := map[string]*llx.RawData{
					"arn":                        llx.StringData(toString(trail.TrailARN)),
					"name":                       llx.StringData(toString(trail.Name)),
					"isMultiRegionTrail":         llx.BoolData(toBool(trail.IsMultiRegionTrail)),
					"isOrganizationTrail":        llx.BoolData(toBool(trail.IsOrganizationTrail)),
					"logFileValidationEnabled":   llx.BoolData(toBool(trail.LogFileValidationEnabled)),
					"includeGlobalServiceEvents": llx.BoolData(toBool(trail.IncludeGlobalServiceEvents)),
					"snsTopicARN":                llx.StringData(toString(trail.SnsTopicARN)),
					"cloudWatchLogsRoleArn":      llx.StringData(toString(trail.CloudWatchLogsRoleArn)),
					"region":                     llx.StringData(toString(trail.HomeRegion)),
				}

				// trail.S3BucketName
				if trail.S3BucketName != nil {
					mqlAwsS3Bucket, err := NewResource(a.MqlRuntime, "aws.s3.bucket",
						map[string]*llx.RawData{"name": llx.StringData(toString(trail.S3BucketName))},
					)
					if err != nil {
						return nil, err
					}
					args["s3bucket"] = llx.ResourceData(mqlAwsS3Bucket, mqlAwsS3Bucket.MqlName())
				}

				// add kms key if there is one
				if trail.KmsKeyId != nil {
					mqlKeyResource, err := NewResource(a.MqlRuntime, "aws.kms.key",
						map[string]*llx.RawData{"arn": llx.StringData(toString(trail.KmsKeyId))},
					)
					// means the key does not exist or we have no access to it
					// dont err out, just assign nil
					if err != nil {
						args["kmsKey"] = llx.NilData
					} else {
						mqlKey := mqlKeyResource.(*mqlAwsKmsKey)
						args["kmsKey"] = llx.ResourceData(mqlKey, mqlKey.MqlName())
					}
				}
				if trail.CloudWatchLogsLogGroupArn != nil {
					mqlLoggroup, err := NewResource(a.MqlRuntime, "aws.cloudwatch.loggroup",
						map[string]*llx.RawData{"arn": llx.StringData(toString(trail.CloudWatchLogsLogGroupArn))},
					)
					// means the log group does not exist or we have no access to it
					// dont err out, just assign nil
					if err != nil {
						args["logGroup"] = llx.NilData
					} else {
						mqlLog := mqlLoggroup.(*mqlAwsCloudwatchLoggroup)
						args["logGroup"] = llx.ResourceData(mqlLog, mqlLog.MqlName())
					}
				}

				mqlAwsCloudtrailTrail, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.cloudtrail.trail", args)
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

func (a *mqlAwsCloudtrailTrail) s3bucket() (*mqlAwsS3Bucket, error) {
	// no s3 bucket on the trail object
	return nil, nil
}

func (a *mqlAwsCloudtrailTrail) logGroup() (*mqlAwsCloudwatchLoggroup, error) {
	// no log group on the trail object
	return nil, nil
}

func (a *mqlAwsCloudtrailTrail) kmsKey() (*mqlAwsKmsKey, error) {
	// no key id on the trail object
	return nil, nil
}

func (a *mqlAwsCloudtrailTrail) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsCloudtrailTrail) status() (interface{}, error) {
	regionValue := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Cloudtrail(regionValue)
	ctx := context.Background()

	arnValue := a.Arn.Data

	// no pagination required
	trailstatus, err := svc.GetTrailStatus(ctx, &cloudtrail.GetTrailStatusInput{
		Name: &arnValue,
	})
	if err != nil {
		return nil, err
	}

	return convert.JsonToDict(trailstatus)
}

func (a *mqlAwsCloudtrailTrail) eventSelectors() ([]interface{}, error) {
	regionValue := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Cloudtrail(regionValue)
	ctx := context.Background()

	arnValue := a.Arn.Data
	// no pagination required
	trailmgmtevents, err := svc.GetEventSelectors(ctx, &cloudtrail.GetEventSelectorsInput{
		TrailName: &arnValue,
	})
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(trailmgmtevents.EventSelectors)
}
