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
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsIamAccessanalyzerAnalyzer) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsIamAccessAnalyzer) analyzers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getAnalyzers(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsIamAccessAnalyzer) getAnalyzers(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.AccessAnalyzer(region)
			res := []any{}

			// we need to iterate over all the analyzers types in the account
			analyzerTypes := []aatypes.Type{aatypes.TypeAccount, aatypes.TypeOrganization, aatypes.TypeAccountUnusedAccess, aatypes.TypeOrganizationUnusedAccess, aatypes.TypeAccountInternalAccess, aatypes.TypeOrganizationInternalAccess}
			for _, analyzerType := range analyzerTypes {
				ctx := context.Background()

				// query all the analyzers in the account / region
				params := &accessanalyzer.ListAnalyzersInput{Type: analyzerType}
				paginator := accessanalyzer.NewListAnalyzersPaginator(svc, params)
				for paginator.HasMorePages() {
					analyzers, err := paginator.NextPage(ctx)
					if err != nil {
						if Is400AccessDeniedError(err) {
							log.Warn().Str("region", region).Msg("error accessing region for AWS API")
							return res, nil
						}
						log.Error().Err(err).Str("region", region).Msg("error listing analyzers")
						return nil, err
					}
					for _, analyzer := range analyzers.Analyzers {
						mqlAnalyzer, err := CreateResource(a.MqlRuntime, "aws.iam.accessanalyzer.analyzer",
							map[string]*llx.RawData{
								"arn":                    llx.StringDataPtr(analyzer.Arn),
								"name":                   llx.StringDataPtr(analyzer.Name),
								"status":                 llx.StringData(string(analyzer.Status)),
								"type":                   llx.StringData(string(analyzer.Type)),
								"region":                 llx.StringData(region),
								"tags":                   llx.MapData(toInterfaceMap(analyzer.Tags), types.String),
								"createdAt":              llx.TimeDataPtr(analyzer.CreatedAt),
								"lastResourceAnalyzed":   llx.StringDataPtr(analyzer.LastResourceAnalyzed),
								"lastResourceAnalyzedAt": llx.TimeDataPtr(analyzer.LastResourceAnalyzedAt),
							})
						if err != nil {
							return nil, err
						}
						res = append(res, mqlAnalyzer)
					}
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsIamAccessAnalyzer) getAnalyzerMap() (map[string][]string, error) {
	analyzerMap := map[string][]string{}
	analyzerList := a.GetAnalyzers()
	if analyzerList.Error != nil {
		return nil, analyzerList.Error
	}
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
	return analyzerMap, nil
}

func (a *mqlAwsIamAccessAnalyzer) findings() ([]any, error) {
	return a.listFindingsWithStatus("ACTIVE")
}

func (a *mqlAwsIamAccessAnalyzer) archivedFindings() ([]any, error) {
	return a.listFindingsWithStatus("ARCHIVED")
}

func (a *mqlAwsIamAccessAnalyzer) listFindingsWithStatus(status string) ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	analyzerMap, err := a.getAnalyzerMap()
	if err != nil {
		return nil, err
	}

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.listFindings(conn, analyzerMap, status), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		results := poolOfJobs.Jobs[i].Result.([]any)
		res = append(res, results...)
	}
	return res, nil
}

func (a *mqlAwsIamAccessAnalyzer) listFindings(conn *connection.AwsConnection, analyzerMap map[string][]string, status string) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.AccessAnalyzer(region)
			res := []any{}

			analyzerList := analyzerMap[region]
			for _, analyzerArn := range analyzerList {
				ctx := context.Background()

				params := &accessanalyzer.ListFindingsV2Input{
					AnalyzerArn: aws.String(analyzerArn),
					Filter: map[string]aatypes.Criterion{
						"status": {
							Eq: []string{status},
						},
					},
				}
				paginator := accessanalyzer.NewListFindingsV2Paginator(svc, params)
				for paginator.HasMorePages() {
					findings, err := paginator.NextPage(ctx)
					if err != nil {
						if Is400AccessDeniedError(err) {
							log.Warn().Str("region", region).Msg("error accessing region for AWS API")
							return res, nil
						}
						log.Error().Err(err).Str("region", region).Msg("error listing findings")
						return nil, err
					}
					for _, finding := range findings.Findings {
						mqlFinding, err := CreateResource(a.MqlRuntime, "aws.iam.accessanalyzer.finding",
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
								"region":               llx.StringData(region),
								"analyzerArn":          llx.StringData(analyzerArn),
							})
						if err != nil {
							return nil, err
						}
						res = append(res, mqlFinding)
					}
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}
