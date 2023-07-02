package aws

import (
	"context"
	"encoding/json"

	"errors"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/smithy-go/transport/http"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/aws/awspolicy"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (l *mqlAwsLambda) id() (string, error) {
	return "aws.lambda", nil
}

func (l *mqlAwsLambda) GetFunctions() ([]interface{}, error) {
	provider, err := awsProvider(l.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(l.getFunctions(provider), 5)
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

func (l *mqlAwsLambda) getFunctions(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.Lambda(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			var marker *string
			for {
				functionsResp, err := svc.ListFunctions(ctx, &lambda.ListFunctionsInput{Marker: marker})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, errors.Join(err, errors.New("could not gather aws lambda functions"))
				}
				for _, function := range functionsResp.Functions {
					vpcConfigJson, err := core.JsonToDict(function.VpcConfig)
					if err != nil {
						return nil, err
					}
					var dlqTarget string
					if function.DeadLetterConfig != nil {
						dlqTarget = core.ToString(function.DeadLetterConfig.TargetArn)
					}
					tags := make(map[string]interface{})
					tagsResp, err := svc.ListTags(ctx, &lambda.ListTagsInput{Resource: function.FunctionArn})
					if err == nil {
						for k, v := range tagsResp.Tags {
							tags[k] = v
						}
					}
					mqlFunc, err := l.MotorRuntime.CreateResource("aws.lambda.function",
						"arn", core.ToString(function.FunctionArn),
						"name", core.ToString(function.FunctionName),
						"dlqTargetArn", dlqTarget,
						"vpcConfig", vpcConfigJson,
						"region", regionVal,
						"tags", tags,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlFunc)
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

func (p *mqlAwsLambdaFunction) init(args *resources.Args) (*resources.Args, AwsLambdaFunction, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(p.MqlResource().MotorRuntime); ids != nil {
			(*args)["name"] = ids.name
			(*args)["arn"] = ids.arn
		}
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch lambda function")
	}

	// load all rds db instances
	obj, err := p.MotorRuntime.CreateResource("aws.lambda")
	if err != nil {
		return nil, nil, err
	}
	l := obj.(AwsLambda)

	rawResources, err := l.Functions()
	if err != nil {
		return nil, nil, err
	}

	arnVal := (*args)["arn"].(string)
	for i := range rawResources {
		dbInstance := rawResources[i].(AwsLambdaFunction)
		mqlDbArn, err := dbInstance.Arn()
		if err != nil {
			return nil, nil, errors.New("lambda function does not exist")
		}
		if mqlDbArn == arnVal {
			return args, dbInstance, nil
		}
	}
	return nil, nil, errors.New("lambda function does not exist")
}

func (l *mqlAwsLambdaFunction) GetConcurrency() (int64, error) {
	funcName, err := l.Name()
	if err != nil {
		return 0, err
	}
	region, err := l.Region()
	if err != nil {
		return 0, err
	}
	at, err := awsProvider(l.MotorRuntime.Motor.Provider)
	if err != nil {
		return 0, err
	}
	svc := at.Lambda(region)
	ctx := context.Background()

	// no pagination required
	functionConcurrency, err := svc.GetFunctionConcurrency(ctx, &lambda.GetFunctionConcurrencyInput{FunctionName: &funcName})
	if err != nil {
		return 0, errors.Join(err, errors.New("could not gather aws lambda function concurrency"))
	}
	if functionConcurrency.ReservedConcurrentExecutions != nil {
		return core.ToInt64From32(functionConcurrency.ReservedConcurrentExecutions), nil
	}

	return 0, nil
}

func (l *mqlAwsLambdaFunction) GetPolicy() (interface{}, error) {
	funcArn, err := l.Arn()
	if err != nil {
		return nil, err
	}
	region, err := l.Region()
	if err != nil {
		return 0, err
	}
	provider, err := awsProvider(l.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := provider.Lambda(region)
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
		return core.JsonToDict(policy)
	}

	return nil, nil
}

func (l *mqlAwsLambdaFunction) id() (string, error) {
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
