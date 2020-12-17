package resources

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
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
		return []*jobpool.Job{&jobpool.Job{Err: err}}
	}
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{&jobpool.Job{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.Cloudtrail(regionVal)
			ctx := context.Background()

			// no pagination required
			trailsResp, err := svc.DescribeTrailsRequest(&cloudtrail.DescribeTrailsInput{}).Send(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "could not gather aws cloudtrail trails")
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
				// only include trail if this region is the home region for the trail
				// we do this to avoid getting duped results from multiregion trails
				if regionVal != toString(trail.HomeRegion) {
					continue
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
					"region", toString(trail.HomeRegion),
				)
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
	trailstatus, err := svc.GetTrailStatusRequest(&cloudtrail.GetTrailStatusInput{
		Name: &arnValue,
	}).Send(ctx)
	if err != nil {
		return nil, err
	}

	return jsonToDict(trailstatus)
}

func (t *lumiAwsCloudtrailTrail) GetLogGroup() (interface{}, error) {
	arnValue, err := t.CloudWatchLogsLogGroupArn()
	if err != nil {
		return nil, err
	}

	if err != nil || len(arnValue) < 6 {
		return nil, errors.Wrap(err, "unable to parse cloud watch log group arn")
	}
	// arn:aws:logs:<region>:<aws_account_number>:log-group:GROUPVAL:*
	logGroupArn := strings.Split(arnValue, ":")
	groupName := logGroupArn[6]
	region := logGroupArn[3]

	at, err := awstransport(t.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.CloudwatchLogs(region)
	ctx := context.Background()

	nextToken := aws.String("no_token_to_start_with")
	params := &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: &groupName,
	}
	for nextToken != nil {
		logGroups, err := svc.DescribeLogGroupsRequest(params).Send(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "could not gather aws cloudwatch log groups")
		}
		nextToken = logGroups.NextToken
		if logGroups.NextToken != nil {
			params.NextToken = nextToken
		}
		for _, loggroup := range logGroups.LogGroups {
			if toString(loggroup.Arn) == arnValue {
				lumiLogGroup, err := t.Runtime.CreateResource("aws.cloudwatch.loggroup",
					"arn", toString(loggroup.Arn),
					"name", toString(loggroup.LogGroupName),
				)
				if err != nil {
					return nil, err
				}
				return lumiLogGroup, nil
			}
		}
	}
	return nil, errors.New("unable to find matching log group")
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
	trailmgmtevents, err := svc.GetEventSelectorsRequest(&cloudtrail.GetEventSelectorsInput{
		TrailName: &arnValue,
	}).Send(ctx)
	if err != nil {
		return nil, err
	}
	return jsonToDictSlice(trailmgmtevents.EventSelectors)
}
