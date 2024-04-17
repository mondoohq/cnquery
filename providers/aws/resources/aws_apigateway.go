// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"

	"go.mondoo.com/cnquery/v11/types"
)

func (a *mqlAwsApigateway) id() (string, error) {
	return "aws.apigateway", nil
}

func (a *mqlAwsApigateway) restApis() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getRestApis(conn), 5)
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

func (a *mqlAwsApigateway) getRestApis(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("gateway>getRestApis>calling AWS with region %s", regionVal)

			svc := conn.Apigateway(regionVal)
			ctx := context.Background()

			res := []interface{}{}
			var position *string
			for {
				restApisResp, err := svc.GetRestApis(ctx, &apigateway.GetRestApisInput{Position: position})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, errors.Wrap(err, "could not gather AWS API Gateway REST APIs")
				}

				for _, restApi := range restApisResp.Items {
					mqlRestApi, err := CreateResource(a.MqlRuntime, "aws.apigateway.restapi",
						map[string]*llx.RawData{
							"arn":         llx.StringData(fmt.Sprintf(apiArnPattern, regionVal, conn.AccountId(), convert.ToString(restApi.Id))),
							"id":          llx.StringData(convert.ToString(restApi.Id)),
							"name":        llx.StringData(convert.ToString(restApi.Name)),
							"description": llx.StringData(convert.ToString(restApi.Description)),
							"createdDate": llx.TimeDataPtr(restApi.CreatedDate),
							"region":      llx.StringData(regionVal),
							"tags":        llx.MapData(strMapToInterface(restApi.Tags), types.String),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlRestApi)
				}
				if restApisResp.Position == nil {
					break
				}
				position = restApisResp.Position
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initAwsApigatewayRestapi(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch gateway restapi")
	}

	obj, err := CreateResource(runtime, "aws.apigateway", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	gw := obj.(*mqlAwsApigateway)

	rawResources := gw.GetRestApis()
	if rawResources.Error != nil {
		return nil, nil, err
	}

	arnVal := args["arn"].Value.(string)
	for i := range rawResources.Data {
		restApi := rawResources.Data[i].(*mqlAwsApigatewayRestapi)
		if restApi.Arn.Data == arnVal {
			return args, restApi, nil
		}
	}
	return nil, nil, errors.New("gateway restapi does not exist")
}

func (a *mqlAwsApigatewayRestapi) stages() ([]interface{}, error) {
	restApiId := a.Id.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Apigateway(region)
	ctx := context.Background()

	// no pagination required
	stagesResp, err := svc.GetStages(ctx, &apigateway.GetStagesInput{RestApiId: &restApiId})
	if err != nil {
		return nil, errors.Wrap(err, "could not gather AWS API Gateway stages")
	}
	res := []interface{}{}
	for _, stage := range stagesResp.Item {
		dictMethodSettings, err := convert.JsonToDict(stage.MethodSettings)
		if err != nil {
			return nil, err
		}
		mqlStage, err := CreateResource(a.MqlRuntime, "aws.apigateway.stage",
			map[string]*llx.RawData{
				"arn":            llx.StringData(fmt.Sprintf(apiStageArnPattern, region, conn.AccountId(), restApiId, convert.ToString(stage.StageName))),
				"name":           llx.StringData(convert.ToString(stage.StageName)),
				"description":    llx.StringData(convert.ToString(stage.Description)),
				"tracingEnabled": llx.BoolData(stage.TracingEnabled),
				"deploymentId":   llx.StringData(convert.ToString(stage.DeploymentId)),
				"methodSettings": llx.MapData(dictMethodSettings, types.Any),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlStage)
	}
	return res, nil
}

func (a *mqlAwsApigatewayRestapi) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsApigatewayStage) id() (string, error) {
	return a.Arn.Data, nil
}
