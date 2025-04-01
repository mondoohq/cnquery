// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
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
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}
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

	if arn == "" {
		return nil, nil, errors.New("arn or name required to fetch aws cloudtrail trail")
	}

	log.Debug().Str("arn", arn).Msg("init cloudtrail trail with arn")

	// load all s3 buckets
	obj, err := CreateResource(runtime, "aws.cloudtrail", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	awsCloudtrail := obj.(*mqlAwsCloudtrail)

	rawResources := awsCloudtrail.GetTrails()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	for i := range rawResources.Data {
		trail := rawResources.Data[i].(*mqlAwsCloudtrailTrail)
		if trail.Arn.Data == arn {
			return args, trail, nil
		}
	}
	return args, nil, errors.New("cloudtrail trail does not exist")
}

type mqlAwsCloudtrailTrailInternal struct {
	trailCache types.Trail
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
			log.Debug().Msgf("cloudtrail>getTrails>calling aws with region %s", regionVal)

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
				if regionVal != convert.ToValue(trail.HomeRegion) {
					continue
				}
				args := map[string]*llx.RawData{
					"arn":                        llx.StringDataPtr(trail.TrailARN),
					"name":                       llx.StringDataPtr(trail.Name),
					"isMultiRegionTrail":         llx.BoolDataPtr(trail.IsMultiRegionTrail),
					"isOrganizationTrail":        llx.BoolDataPtr(trail.IsOrganizationTrail),
					"logFileValidationEnabled":   llx.BoolDataPtr(trail.LogFileValidationEnabled),
					"includeGlobalServiceEvents": llx.BoolDataPtr(trail.IncludeGlobalServiceEvents),
					"snsTopicARN":                llx.StringDataPtr(trail.SnsTopicARN),
					"cloudWatchLogsRoleArn":      llx.StringDataPtr(trail.CloudWatchLogsRoleArn),
					"cloudWatchLogsLogGroupArn":  llx.StringDataPtr(trail.CloudWatchLogsLogGroupArn),
					"region":                     llx.StringDataPtr(trail.HomeRegion),
				}

				mqlTrail, err := CreateResource(a.MqlRuntime, "aws.cloudtrail.trail", args)
				if err != nil {
					return nil, err
				}
				mqlTrail.(*mqlAwsCloudtrailTrail).trailCache = trail

				res = append(res, mqlTrail)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsCloudtrailTrail) s3bucket() (*mqlAwsS3Bucket, error) {
	if a.trailCache.S3BucketName != nil {
		mqlBucket, err := NewResource(a.MqlRuntime, "aws.s3.bucket",
			map[string]*llx.RawData{"name": llx.StringDataPtr(a.trailCache.S3BucketName)},
		)
		if err == nil {
			return mqlBucket.(*mqlAwsS3Bucket), nil
		} else {
			log.Error().Err(err).Msg("cannot get s3 bucket")
		}
	}
	a.S3bucket.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (a *mqlAwsCloudtrailTrail) logGroup() (*mqlAwsCloudwatchLoggroup, error) {
	if a.trailCache.CloudWatchLogsLogGroupArn != nil {
		mqlLoggroup, err := NewResource(a.MqlRuntime, "aws.cloudwatch.loggroup",
			map[string]*llx.RawData{"arn": llx.StringDataPtr(a.trailCache.CloudWatchLogsLogGroupArn)},
		)
		if err == nil {
			return mqlLoggroup.(*mqlAwsCloudwatchLoggroup), nil
		} else {
			log.Error().Err(err).Msg("cannot get log group")
		}
	}
	a.LogGroup.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}

func (a *mqlAwsCloudtrailTrail) kmsKey() (*mqlAwsKmsKey, error) {
	// add kms key if there is one
	if a.trailCache.KmsKeyId != nil {
		mqlKeyResource, err := NewResource(a.MqlRuntime, "aws.kms.key",
			map[string]*llx.RawData{"arn": llx.StringDataPtr(a.trailCache.KmsKeyId)},
		)
		if err == nil {
			return mqlKeyResource.(*mqlAwsKmsKey), nil
		} else {
			log.Error().Err(err).Msg("cannot get key")
		}
	}
	a.KmsKey.State = plugin.StateIsSet | plugin.StateIsNull
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
	eventSelectorsOutput, err := svc.GetEventSelectors(ctx, &cloudtrail.GetEventSelectorsInput{
		TrailName: &arnValue,
	})
	if err != nil {
		return nil, err
	}

	// Basic event selectors
	basicSelectors, err := convert.JsonToDictSlice(eventSelectorsOutput.EventSelectors)
	if err != nil {
		return nil, err
	}

	allSelectors := basicSelectors

	// Advanced event selectors if they exist
	if len(eventSelectorsOutput.AdvancedEventSelectors) > 0 {
		advancedSelectors, err := convert.JsonToDictSlice(eventSelectorsOutput.AdvancedEventSelectors)
		if err != nil {
			return nil, err
		}

		// Basic plus advanced event selectors
		allSelectors = append(basicSelectors, advancedSelectors...)
	}

	return allSelectors, nil
}
