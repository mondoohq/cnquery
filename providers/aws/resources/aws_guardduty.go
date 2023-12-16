// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/aws/aws-sdk-go-v2/service/guardduty/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v9/providers/aws/connection"
)

func (a *mqlAwsGuardduty) id() (string, error) {
	return "aws.guardduty", nil
}

func (a *mqlAwsGuardduty) detectors() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getDetectors(conn), 5)
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

func (a *mqlAwsGuarddutyDetector) id() (string, error) {
	return a.Id.Data, nil
}

func (a *mqlAwsGuardduty) getDetectors(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Guardduty(regionVal)
			ctx := context.Background()

			res := []interface{}{}
			params := &guardduty.ListDetectorsInput{}

			nextToken := aws.String("no_token_to_start_with")
			for nextToken != nil {
				detectors, err := svc.ListDetectors(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, id := range detectors.DetectorIds {
					mqlCluster, err := CreateResource(a.MqlRuntime, "aws.guardduty.detector",
						map[string]*llx.RawData{
							"id":     llx.StringData(id),
							"region": llx.StringData(regionVal),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlCluster)
				}
				nextToken = detectors.NextToken
				if detectors.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsGuarddutyDetector) unarchivedFindings() ([]interface{}, error) {
	id := a.Id.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Guardduty(region)
	ctx := context.Background()

	findings, err := svc.ListFindings(ctx, &guardduty.ListFindingsInput{
		DetectorId: &id,
		FindingCriteria: &types.FindingCriteria{
			Criterion: map[string]types.Condition{
				"service.archived": {
					Equals: []string{"false"},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	findingDetails, err := svc.GetFindings(ctx, &guardduty.GetFindingsInput{FindingIds: findings.FindingIds, DetectorId: &id})
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(findingDetails.Findings)
}

func initAwsGuarddutyDetector(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["id"] == nil && args["region"] == nil {
		return nil, nil, errors.New("name and region required to fetch guardduty detector")
	}

	id := args["id"].Value.(string)
	region := args["region"].Value.(string)
	conn := runtime.Connection.(*connection.AwsConnection)

	svc := conn.Guardduty(region)
	ctx := context.Background()
	detector, err := svc.GetDetector(ctx, &guardduty.GetDetectorInput{DetectorId: &id})
	if err != nil {
		return nil, nil, err
	}

	args["status"] = llx.StringData(string(detector.Status))
	args["findingPublishingFrequency"] = llx.StringData(string(detector.FindingPublishingFrequency))
	return args, nil, nil
}
