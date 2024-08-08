// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer"
	aatypes "github.com/aws/aws-sdk-go-v2/service/accessanalyzer/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
	"go.mondoo.com/cnquery/v11/types"
)

func (a *mqlAwsIamAccessanalyzerAnalyzer) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsIamAccessAnalyzer) analyzers() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getAnalyzers(conn), 5)
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

func (a *mqlAwsIamAccessAnalyzer) getAnalyzers(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for i := range regions {
		regionVal := regions[i]
		f := func() (jobpool.JobResult, error) {
			svc := conn.AccessAnalyzer(regionVal)
			res := []interface{}{}

			// we need to iterate over all the analyzers types in the account
			analyzerTypes := []aatypes.Type{aatypes.TypeAccount, aatypes.TypeOrganization, aatypes.TypeAccountUnusedAccess, aatypes.TypeOrganizationUnusedAccess}
			for _, analyzerType := range analyzerTypes {
				ctx := context.Background()

				// query all the analyzers in the account / region
				nextToken := aws.String("no_token_to_start_with")
				params := &accessanalyzer.ListAnalyzersInput{Type: analyzerType}
				for nextToken != nil {
					analyzers, err := svc.ListAnalyzers(ctx, params)
					if err != nil {
						if Is400AccessDeniedError(err) {
							log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
							return res, nil
						}
						log.Error().Err(err).Str("region", regionVal).Msg("error listing analyzers")
						return nil, err
					}
					for _, analyzer := range analyzers.Analyzers {
						mqlAnalyzer, err := CreateResource(a.MqlRuntime, "aws.iam.accessanalyzer.analyzer",
							map[string]*llx.RawData{
								"arn":                    llx.StringDataPtr(analyzer.Arn),
								"name":                   llx.StringDataPtr(analyzer.Name),
								"status":                 llx.StringData(string(analyzer.Status)),
								"type":                   llx.StringData(string(analyzer.Type)),
								"region":                 llx.StringData(regionVal),
								"tags":                   llx.MapData(strMapToInterface(analyzer.Tags), types.String),
								"createdAt":              llx.TimeDataPtr(analyzer.CreatedAt),
								"lastResourceAnalyzed":   llx.StringDataPtr(analyzer.LastResourceAnalyzed),
								"lastResourceAnalyzedAt": llx.TimeDataPtr(analyzer.LastResourceAnalyzedAt),
							})
						if err != nil {
							return nil, err
						}
						res = append(res, mqlAnalyzer)
					}
					nextToken = analyzers.NextToken
					if analyzers.NextToken != nil {
						params.NextToken = nextToken
					}
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsIamAccessAnalyzer) findings() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	analyzerMap := map[string][]string{}

	// we need to retrieve all the analyzers first and we group them by region to request all findings
	analyzerList := a.GetAnalyzers()
	for _, analyzer := range analyzerList.Data {
		analyzerInstance, ok := analyzer.(*mqlAwsIamAccessanalyzerAnalyzer)
		if !ok {
			return nil, errors.New("error casting to analyzer instance")
		}

		region := analyzerInstance.GetRegion().Data
		if analyzerMap[region] == nil {
			analyzerMap[region] = []string{}
		}

		analyzerMap[region] = append(analyzerMap[region], analyzerInstance.GetArn().Data)
	}

	// collect the list of findings
	res := []interface{}{}

	// start ppol and run the jobs
	poolOfJobs := jobpool.CreatePool(a.listFindings(conn, analyzerMap), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}

	for i := range poolOfJobs.Jobs {
		results := poolOfJobs.Jobs[i].Result.([]interface{})
		res = append(res, results...)
	}
	return res, nil
}

func (a *mqlAwsIamAccessAnalyzer) listFindings(conn *connection.AwsConnection, analyzerMap map[string][]string) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for i := range regions {
		regionVal := regions[i]
		f := func() (jobpool.JobResult, error) {
			svc := conn.AccessAnalyzer(regionVal)
			res := []interface{}{}

			analyzerList := analyzerMap[regionVal]
			for _, analyzerArn := range analyzerList {

				ctx := context.Background()

				nextToken := aws.String("no_token_to_start_with")
				params := &accessanalyzer.ListFindingsV2Input{
					AnalyzerArn: aws.String(analyzerArn),
					Filter: map[string]aatypes.Criterion{
						"status": {
							Eq: []string{"ACTIVE"},
						},
					},
				}
				for nextToken != nil {
					findings, err := svc.ListFindingsV2(ctx, params)
					if err != nil {
						if Is400AccessDeniedError(err) {
							log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
							return res, nil
						}
						log.Error().Err(err).Str("region", regionVal).Msg("error listing analyzers")
						return nil, err
					}
					for _, finding := range findings.Findings {
						mqlIamAnalyzserFindings, err := CreateResource(a.MqlRuntime, "aws.iam.accessanalyzer.finding",
							map[string]*llx.RawData{
								"__id":                 llx.StringDataPtr(finding.Id),
								"id":                   llx.StringDataPtr(finding.Id),
								"error":                llx.StringDataPtr(finding.Error),
								"resourceArn":          llx.StringDataPtr(finding.Resource),
								"resourceOwnerAccount": llx.StringDataPtr(finding.ResourceOwnerAccount),
								"resourceType":         llx.StringData(string(finding.ResourceType)),
								"status":               llx.StringData(string(finding.Status)),
								"type":                 llx.StringData(string(finding.FindingType)),
								"createdAt":            llx.TimeDataPtr(finding.CreatedAt),
								"updatedAt":            llx.TimeDataPtr(finding.UpdatedAt),
								"analyzedAt":           llx.TimeDataPtr(finding.AnalyzedAt),
								"region":               llx.StringData(regionVal),
								"analyzerArn":          llx.StringData(analyzerArn),
							})
						if err != nil {
							return nil, err
						}
						res = append(res, mqlIamAnalyzserFindings)
					}
					nextToken = findings.NextToken
					if findings.NextToken != nil {
						params.NextToken = nextToken
					}
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}
