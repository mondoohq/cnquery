package aws

import (
	"context"
	"fmt"

	"errors"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAwsApigateway) id() (string, error) {
	return "aws.apigateway", nil
}

const (
	apiArnPattern      = "arn:aws:apigateway:%s:%s::/apis/%s"
	apiStageArnPattern = "arn:aws:apigateway:%s:%s::/apis/%s/stages/%s"
)

func (a *mqlAwsApigateway) GetRestApis() ([]interface{}, error) {
	provider, err := awsProvider(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getRestApis(provider), 5)
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

func (a *mqlAwsApigateway) getRestApis(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	account, err := provider.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling AWS with region %s", regionVal)

			svc := provider.Apigateway(regionVal)
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
					return nil, errors.Join(err, errors.New("could not gather AWS API Gateway REST APIs"))
				}

				for _, restApi := range restApisResp.Items {
					mqlRestApi, err := a.MotorRuntime.CreateResource("aws.apigateway.restapi",
						"arn", fmt.Sprintf(apiArnPattern, regionVal, account.ID, core.ToString(restApi.Id)),
						"id", core.ToString(restApi.Id),
						"name", core.ToString(restApi.Name),
						"description", core.ToString(restApi.Description),
						"createdDate", restApi.CreatedDate,
						"region", regionVal,
						"tags", core.StrMapToInterface(restApi.Tags),
					)
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

func (d *mqlAwsApigatewayRestapi) init(args *resources.Args) (*resources.Args, AwsApigatewayRestapi, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(d.MqlResource().MotorRuntime); ids != nil {
			(*args)["name"] = ids.name
			(*args)["arn"] = ids.arn
		}
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch gateway restapi")
	}

	obj, err := d.MotorRuntime.CreateResource("aws.apigateway")
	if err != nil {
		return nil, nil, err
	}
	gw := obj.(AwsApigateway)

	rawResources, err := gw.RestApis()
	if err != nil {
		return nil, nil, err
	}

	arnVal := (*args)["arn"].(string)
	for i := range rawResources {
		restApi := rawResources[i].(AwsApigatewayRestapi)
		mqlRestAPIArn, err := restApi.Arn()
		if err != nil {
			return nil, nil, errors.New("gateway restapi does not exist")
		}
		if mqlRestAPIArn == arnVal {
			return args, restApi, nil
		}
	}
	return nil, nil, errors.New("gateway restapi does not exist")
}

func (a *mqlAwsApigatewayRestapi) GetStages() ([]interface{}, error) {
	restApiId, err := a.Id()
	if err != nil {
		return nil, err
	}
	region, err := a.Region()
	if err != nil {
		return nil, err
	}
	provider, err := awsProvider(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	account, err := provider.Account()
	if err != nil {
		return nil, err
	}
	svc := provider.Apigateway(region)
	ctx := context.Background()

	// no pagination required
	stagesResp, err := svc.GetStages(ctx, &apigateway.GetStagesInput{RestApiId: &restApiId})
	if err != nil {
		return nil, errors.Join(err, errors.New("could not gather AWS API Gateway stages"))
	}
	res := []interface{}{}
	for _, stage := range stagesResp.Item {
		dictMethodSettings, err := core.JsonToDict(stage.MethodSettings)
		if err != nil {
			return nil, err
		}
		mqlStage, err := a.MotorRuntime.CreateResource("aws.apigateway.stage",
			"arn", fmt.Sprintf(apiStageArnPattern, region, account.ID, restApiId, core.ToString(stage.StageName)),
			"name", core.ToString(stage.StageName),
			"description", core.ToString(stage.Description),
			"tracingEnabled", stage.TracingEnabled,
			"deploymentId", core.ToString(stage.DeploymentId),
			"methodSettings", dictMethodSettings,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlStage)
	}
	return res, nil
}

func (l *mqlAwsApigatewayRestapi) id() (string, error) {
	return l.Arn()
}

func (l *mqlAwsApigatewayStage) id() (string, error) {
	return l.Arn()
}
