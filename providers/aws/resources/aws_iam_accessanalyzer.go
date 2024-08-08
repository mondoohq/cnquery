// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer"
	aatypes "github.com/aws/aws-sdk-go-v2/service/accessanalyzer/types"
	"github.com/aws/aws-sdk-go/aws"
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
			ctx := context.Background()
			res := []interface{}{}
			nextToken := aws.String("no_token_to_start_with")
			params := &accessanalyzer.ListAnalyzersInput{Type: aatypes.TypeAccount}
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
							"createdAt":              llx.TimeDataPtr(analyzer.CreatedAt),
							"lastResourceAnalyzed":   llx.StringDataPtr(analyzer.LastResourceAnalyzed),
							"lastResourceAnalyzedAt": llx.TimeDataPtr(analyzer.LastResourceAnalyzedAt),
							"name":                   llx.StringDataPtr(analyzer.Name),
							"status":                 llx.StringData(string(analyzer.Status)),
							"tags":                   llx.MapData(strMapToInterface(analyzer.Tags), types.String),
							"type":                   llx.StringData(string(analyzer.Type)),
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
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}
