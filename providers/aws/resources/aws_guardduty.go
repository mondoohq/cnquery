// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/guardduty"
	"github.com/aws/aws-sdk-go-v2/service/guardduty/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
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

func (a *mqlAwsGuardduty) findings() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	// we need to retrieve all the detectors first and we group them by region to request all findings
	detectorMap := map[string][]string{}
	detectorList := a.GetDetectors()
	for _, detector := range detectorList.Data {
		detectorInstance, ok := detector.(*mqlAwsGuarddutyDetector)
		if !ok {
			return nil, errors.New("error casting to detector instance")
		}

		region := detectorInstance.GetRegion().Data
		if detectorMap[region] == nil {
			detectorMap[region] = []string{}
		}

		detectorMap[region] = append(detectorMap[region], detectorInstance.GetId().Data)
	}

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.listFindings(conn, detectorMap), 5)
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

func (a *mqlAwsGuardduty) listFindings(conn *connection.AwsConnection, detectorMap map[string][]string) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Guardduty(regionVal)

			res := []interface{}{}
			detectorList := detectorMap[regionVal]
			for _, detectorId := range detectorList {
				ctx := context.Background()

				findingIds := []string{}
				params := &guardduty.ListFindingsInput{
					DetectorId: &detectorId,
					FindingCriteria: &types.FindingCriteria{
						Criterion: map[string]types.Condition{
							"region": {
								Equals: []string{regionVal},
							},
							"service.archived": {
								Equals: []string{"false"},
							},
						},
					},
				}

				nextToken := aws.String("no_token_to_start_with")
				for nextToken != nil {
					detectors, err := svc.ListFindings(ctx, params)
					if err != nil {
						if Is400AccessDeniedError(err) {
							log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
							return nil, nil
						}
						return nil, err
					}

					findingIds = append(findingIds, detectors.FindingIds...)
					nextToken = detectors.NextToken
					// AWS returns empty string as pointer :-)
					if nextToken != nil && *nextToken != "" {
						params.NextToken = nextToken
					} else {
						nextToken = nil
					}
				}

				// fetch all findings
				findingDetails, err := svc.GetFindings(ctx, &guardduty.GetFindingsInput{
					FindingIds: findingIds,
					DetectorId: &detectorId,
				})
				if err != nil {
					return nil, err
				}

				for _, finding := range findingDetails.Findings {
					mqlFinding, err := newMqlAwsGuardDutyFinding(a.MqlRuntime, finding)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlFinding)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlAwsGuardDutyFinding(runtime *plugin.Runtime, finding types.Finding) (*mqlAwsGuarddutyFinding, error) {
	var severity float64
	if finding.Severity != nil {
		severity = *finding.Severity
	}

	var confidence float64
	if finding.Confidence != nil {
		confidence = *finding.Confidence
	}

	res, err := CreateResource(runtime, "aws.guardduty.finding", map[string]*llx.RawData{
		"__id":        llx.StringDataPtr(finding.Arn),
		"arn":         llx.StringDataPtr(finding.Arn),
		"id":          llx.StringDataPtr(finding.Id),
		"region":      llx.StringDataPtr(finding.Region),
		"title":       llx.StringDataPtr(finding.Title),
		"description": llx.StringDataPtr(finding.Description),
		"severity":    llx.FloatData(severity),
		"confidence":  llx.FloatData(confidence),
		"type":        llx.StringDataPtr(finding.Type),
		"createdAt":   llx.TimeDataPtr(parseAwsTimestampPtr(finding.CreatedAt)),
		"updatedAt":   llx.TimeDataPtr(parseAwsTimestampPtr(finding.UpdatedAt)),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsGuarddutyFinding), nil
}

func parseAwsTimestampPtr(value *string) *time.Time {
	if value == nil {
		return nil
	}
	return parseAwsTimestamp(*value)
}

func parseAwsTimestamp(value string) *time.Time {
	timestamp, err := time.Parse(time.RFC3339, value)
	if err != nil {
		log.Warn().Err(err).Str("timestamp", value).Msg("failed to parse timestamp")
		return nil
	}
	return &timestamp
}
