// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package aws

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/aws/aws-sdk-go-v2/service/guardduty/types"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (g *mqlAwsGuardduty) id() (string, error) {
	return "aws.guardduty", nil
}

func (g *mqlAwsGuardduty) GetDetectors() ([]interface{}, error) {
	provider, err := awsProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(g.getDetectors(provider), 5)
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

func (g *mqlAwsGuarddutyDetector) id() (string, error) {
	return g.Id()
}

func (g *mqlAwsGuardduty) getDetectors(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.Guardduty(regionVal)
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
					mqlCluster, err := g.MotorRuntime.CreateResource("aws.guardduty.detector",
						"id", id,
						"region", regionVal,
					)
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

func (g *mqlAwsGuarddutyDetector) GetUnarchivedFindings() ([]interface{}, error) {
	id, err := g.Id()
	if err != nil {
		return nil, err
	}
	region, err := g.Region()
	if err != nil {
		return nil, err
	}
	provider, err := awsProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := provider.Guardduty(region)
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
	return core.JsonToDictSlice(findingDetails.Findings)
}

func (g *mqlAwsGuarddutyDetector) init(args *resources.Args) (*resources.Args, AwsGuarddutyDetector, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if (*args)["id"] == nil && (*args)["region"] == nil {
		return nil, nil, errors.New("name and region required to fetch codebuild project")
	}

	id := (*args)["id"].(string)
	region := (*args)["region"].(string)
	provider, err := awsProvider(g.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}
	svc := provider.Guardduty(region)
	ctx := context.Background()
	detector, err := svc.GetDetector(ctx, &guardduty.GetDetectorInput{DetectorId: &id})
	if err != nil {
		return nil, nil, err
	}

	(*args)["status"] = string(detector.Status)
	(*args)["findingPublishingFrequency"] = string(detector.FindingPublishingFrequency)
	return args, nil, nil
}
