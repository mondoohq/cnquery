// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"maps"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/smithy-go/transport/http"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"
	"go.mondoo.com/cnquery/v12/providers/aws/resources/awspolicy"
	"go.mondoo.com/cnquery/v12/types"
)

func (a *mqlAwsLambda) id() (string, error) {
	return "aws.lambda", nil
}

func (a *mqlAwsLambda) functions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getFunctions(conn), 5)
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

func (a *mqlAwsLambda) getFunctions(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("lambda>getFunctions>calling aws with region %s", region)

			svc := conn.Lambda(region)
			ctx := context.Background()
			res := []any{}
			params := &lambda.ListFunctionsInput{}
			paginator := lambda.NewListFunctionsPaginator(svc, params)
			for paginator.HasMorePages() {
				functionsResp, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, errors.Wrap(err, "could not gather aws lambda functions")
				}
				for _, function := range functionsResp.Functions {
					vpcConfigJson, err := convert.JsonToDict(function.VpcConfig)
					if err != nil {
						return nil, err
					}
					var dlqTarget string
					if function.DeadLetterConfig != nil {
						dlqTarget = convert.ToValue(function.DeadLetterConfig.TargetArn)
					}
					tags := make(map[string]string)
					tagsResp, err := svc.ListTags(ctx, &lambda.ListTagsInput{Resource: function.FunctionArn})
					if err == nil {
						maps.Copy(tags, tagsResp.Tags)
					}

					if conn.Filters.General.IsFilteredOutByTags(tags) {
						log.Debug().Interface("function", function.FunctionArn).Msg("excluding function due to filters")
						continue
					}

					mqlFunc, err := CreateResource(a.MqlRuntime, "aws.lambda.function",
						map[string]*llx.RawData{
							"arn":          llx.StringDataPtr(function.FunctionArn),
							"name":         llx.StringDataPtr(function.FunctionName),
							"runtime":      llx.StringData(string(function.Runtime)),
							"dlqTargetArn": llx.StringData(dlqTarget),
							"vpcConfig":    llx.MapData(vpcConfigJson, types.Any),
							"region":       llx.StringData(region),
							"tags":         llx.MapData(toInterfaceMap(tags), types.String),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlFunc)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func getLambdaArn(name string, region string, accountId string) string {
	return arn.ARN{
		Region:    region,
		Partition: "aws",
		Service:   "lambda",
		AccountID: accountId,
		Resource:  "function:" + name,
	}.String()
}

func initAwsLambdaFunction(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	name := args["name"]
	region := args["region"]

	var arnVal string
	if args["arn"] == nil {
		if name == nil {
			return nil, nil, errors.New("name required to fetch lambda function")
		}
		if region == nil {
			return nil, nil, errors.New("region required to fetch lambda function")
		}
		arnVal = getLambdaArn(name.String(), region.String(), "")
		if arnVal == "" {
			return nil, nil, errors.New("arn required to fetch lambda function")
		}
	} else {
		arnVal = args["arn"].Value.(string)
	}

	// load all rds db instances
	obj, err := CreateResource(runtime, "aws.lambda", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	l := obj.(*mqlAwsLambda)

	rawResources := l.GetFunctions()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	for _, rawResource := range rawResources.Data {
		dbInstance := rawResource.(*mqlAwsLambdaFunction)
		if dbInstance.Arn.Data == arnVal {
			return args, dbInstance, nil
		}
	}
	return nil, nil, errors.New("lambda function does not exist")
}

func (a *mqlAwsLambdaFunction) concurrency() (int64, error) {
	funcName := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Lambda(region)
	ctx := context.Background()

	// no pagination required
	functionConcurrency, err := svc.GetFunctionConcurrency(ctx, &lambda.GetFunctionConcurrencyInput{FunctionName: &funcName})
	if err != nil {
		return 0, errors.Wrap(err, "could not gather aws lambda function concurrency")
	}
	if functionConcurrency.ReservedConcurrentExecutions == nil {
		return 0, nil
	}
	return int64(*functionConcurrency.ReservedConcurrentExecutions), nil
}

func (a *mqlAwsLambdaFunction) policy() (any, error) {
	funcArn := a.Arn.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Lambda(region)
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
		return convert.JsonToDict(policy)
	}

	return nil, nil
}

func (a *mqlAwsLambdaFunction) id() (string, error) {
	return a.Arn.Data, nil
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
