package resources

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/smithy-go/transport/http"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
	"go.mondoo.io/mondoo/lumi/resources/awspolicy"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"
)

func (l *lumiAwsLambda) id() (string, error) {
	return "aws.lambda", nil
}

func (l *lumiAwsLambda) GetFunctions() ([]interface{}, error) {
	at, err := awstransport(l.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(l.getFunctions(at), 5)
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

func (l *lumiAwsLambda) getFunctions(at *aws_transport.Transport) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
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
				functionsResp, err := svc.ListFunctions(ctx, &lambda.ListFunctionsInput{Marker: marker})
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
					tags := make(map[string]interface{})
					tagsResp, err := svc.ListTags(ctx, &lambda.ListTagsInput{Resource: function.FunctionArn})
					if err == nil {
						for k, v := range tagsResp.Tags {
							tags[k] = v
						}
					}
					lumiFunc, err := l.MotorRuntime.CreateResource("aws.lambda.function",
						"arn", toString(function.FunctionArn),
						"name", toString(function.FunctionName),
						"dlqTargetArn", dlqTarget,
						"vpcConfig", vpcConfigJson,
						"region", regionVal,
						"tags", tags,
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
	at, err := awstransport(l.MotorRuntime.Motor.Transport)
	if err != nil {
		return 0, err
	}
	svc := at.Lambda(region)
	ctx := context.Background()

	// no pagination required
	functionConcurrency, err := svc.GetFunctionConcurrency(ctx, &lambda.GetFunctionConcurrencyInput{FunctionName: &funcName})
	if err != nil {
		return 0, errors.Wrap(err, "could not gather aws lambda function concurrency")
	}
	if functionConcurrency.ReservedConcurrentExecutions != nil {
		return toInt64From32(functionConcurrency.ReservedConcurrentExecutions), nil
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
	at, err := awstransport(l.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.Lambda(region)
	ctx := context.Background()

	// no pagination required
	functionPolicy, err := svc.GetPolicy(ctx, &lambda.GetPolicyInput{FunctionName: &funcArn})
	var respErr *http.ResponseError
	if err != nil && errors.As(err, &respErr) {
		if respErr.HTTPStatusCode() == 404 {
			return nil, nil
		}
	} else if err != nil {
		return nil, err
	}
	if functionPolicy != nil {
		var policy lambdaPolicyDocument
		err = json.Unmarshal([]byte(*functionPolicy.Policy), &policy)
		if err != nil {
			return nil, err
		}
		return jsonToDict(policy)
	}

	return nil, nil
}

func (l *lumiAwsLambdaFunction) id() (string, error) {
	return l.Arn()
}

type lambdaPolicyDocument struct {
	Version   string                  `json:"Version,omitempty"`
	Statement []lambdaPolicyStatement `json:"Statement,omitempty"`
}

type lambdaPolicyStatement struct {
	Sid       string              `json:"Sid,omitempty"`
	Effect    string              `json:"Effect,omitempty"`
	Action    string              `json:"Action,omitempty"`
	Resource  string              `json:"Resource,omitempty"`
	Principal awspolicy.Principal `json:"Principal,omitempty"`
}
