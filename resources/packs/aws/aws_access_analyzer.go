package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	accessanalyzer "github.com/aws/aws-sdk-go-v2/service/accessanalyzer"
	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer/types"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.io/mondoo/motor/providers/aws"
	"go.mondoo.io/mondoo/resources/library/jobpool"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (a *mqlAwsAccessAnalyzer) id() (string, error) {
	return "aws.accessAnalyzer", nil
}

func (e *mqlAwsAccessanalyzerAnalyzer) id() (string, error) {
	return e.Arn()
}

func (a *mqlAwsAccessAnalyzer) GetAnalyzers() ([]interface{}, error) {
	provider, err := awsProvider(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getAnalyzers(provider), 5)
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

func (a *mqlAwsAccessAnalyzer) getAnalyzers(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := provider.AccessAnalyzer(regionVal)
			ctx := context.Background()
			res := []interface{}{}
			nextToken := aws.String("no_token_to_start_with")
			params := &accessanalyzer.ListAnalyzersInput{Type: types.TypeAccount}
			for nextToken != nil {
				analyzers, err := svc.ListAnalyzers(ctx, params)
				if err != nil {
					log.Error().Err(err).Str("region", regionVal).Msg("error listing analyzers")
					return nil, err
				}
				for _, analyzer := range analyzers.Analyzers {
					mqlAnalyzer, err := a.MotorRuntime.CreateResource("aws.accessanalyzer.analyzer",
						"arn", core.ToString(analyzer.Arn),
						"name", core.ToString(analyzer.Name),
						"status", string(analyzer.Status),
						"type", string(analyzer.Type),
						"tags", core.StrMapToInterface(analyzer.Tags),
					)
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
