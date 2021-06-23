package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	accessanalyzer "github.com/aws/aws-sdk-go-v2/service/accessanalyzer"
	"github.com/aws/aws-sdk-go-v2/service/accessanalyzer/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

func (a *lumiAwsAccessAnalyzer) id() (string, error) {
	return "aws.accessAnalyzer", nil
}
func (e *lumiAwsAccessanalyzerAnalyzer) id() (string, error) {
	return e.Arn()
}

func (a *lumiAwsAccessAnalyzer) GetAnalyzers() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getAnalyzers(), 5)
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

func (a *lumiAwsAccessAnalyzer) getAnalyzers() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(a.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {

			svc := at.AccessAnalyzer(regionVal)
			ctx := context.Background()
			res := []interface{}{}
			log.Info().Msgf("analyzer service client %s %v", regionVal, svc)
			nextToken := aws.String("no_token_to_start_with")
			params := &accessanalyzer.ListAnalyzersInput{Type: types.TypeAccount}
			for nextToken != nil {
				analyzers, err := svc.ListAnalyzers(ctx, params)
				if err != nil {
					log.Error().Err(err).Msgf("analyzer region call %s", regionVal)
					return nil, err
				}
				for _, analyzer := range analyzers.Analyzers {
					lumiAnalyzer, err := a.Runtime.CreateResource("aws.accessanalyzer",
						"arn", toString(analyzer.Arn),
						"name", toString(analyzer.Name),
						"status", string(analyzer.Status),
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiAnalyzer)
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
