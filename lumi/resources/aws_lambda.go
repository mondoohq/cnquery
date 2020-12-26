package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

func (l *lumiAwsLambda) id() (string, error) {
	return "aws.lambda", nil
}

func (l *lumiAwsLambda) GetFunctions() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(l.getFunctions(), 5)
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

func (l *lumiAwsLambda) getFunctions() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(l.Runtime.Motor.Transport)
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
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.Lambda(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			var marker *string
			for {
				functionsResp, err := svc.ListFunctionsRequest(&lambda.ListFunctionsInput{Marker: marker}).Send(ctx)
				if err != nil {
					return nil, errors.Wrap(err, "could not gather aws lambda functions")
				}
				for _, function := range functionsResp.Functions {
					vpcConfigJson, err := jsonToDict(function.VpcConfig)
					if err != nil {
						return nil, err
					}
					var dlqTarget string
					if function.DeadLetterConfig != nil {
						dlqTarget = toString(function.DeadLetterConfig.TargetArn)
					}
					lumiFunc, err := l.Runtime.CreateResource("aws.lambda.function",
						"arn", toString(function.FunctionArn),
						"name", toString(function.FunctionName),
						"dlqTargetArn", dlqTarget,
						"vpcConfig", vpcConfigJson,
						"region", regionVal,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiFunc)
				}
				if functionsResp.NextMarker == nil {
					break
				}
				marker = functionsResp.NextMarker
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (l *lumiAwsLambdaFunction) GetConcurrency() (int64, error) {
	funcName, err := l.Name()
	if err != nil {
		return 0, err
	}
	region, err := l.Region()
	if err != nil {
		return 0, err
	}
	at, err := awstransport(l.Runtime.Motor.Transport)
	if err != nil {
		return 0, err
	}
	svc := at.Lambda(region)
	ctx := context.Background()

	// no pagination required
	functionConcurrency, err := svc.GetFunctionConcurrencyRequest(&lambda.GetFunctionConcurrencyInput{FunctionName: &funcName}).Send(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "could not gather aws lambda function concurrency")
	}
	if functionConcurrency.ReservedConcurrentExecutions != nil {
		return *functionConcurrency.ReservedConcurrentExecutions, nil
	}

	return 0, nil
}

func (l *lumiAwsLambdaFunction) GetPolicy() (interface{}, error) {
	funcArn, err := l.Arn()
	if err != nil {
		return nil, err
	}
	region, err := l.Region()
	if err != nil {
		return 0, err
	}
	at, err := awstransport(l.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.Lambda(region)
	ctx := context.Background()

	// no pagination required
	functionPolicy, err := svc.GetPolicyRequest(&lambda.GetPolicyInput{FunctionName: &funcArn}).Send(ctx)
	isAwsErr, code := IsAwsCode(err)
	if err != nil && isAwsErr && code == "ResourceNotFoundException" {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return jsonToDict(functionPolicy)
}

func (l *lumiAwsLambdaFunction) id() (string, error) {
	return l.Arn()
}
