package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/resources/library/jobpool"
	aws_transport "go.mondoo.io/mondoo/motor/providers/aws"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (a *mqlAwsApigateway) id() (string, error) {
	return "aws.apigateway", nil
}

const (
	apiArnPattern      = "arn:aws:apigateway:%s:%s::/apis/%s"
	apiStageArnPattern = "arn:aws:apigateway:%s:%s::/apis/%s/stages/%s"
)

func (a *mqlAwsApigateway) GetRestApis() ([]interface{}, error) {
	at, err := awstransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getRestApis(at), 5)
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

func (a *mqlAwsApigateway) getRestApis(at *aws_transport.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	account, err := at.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.Apigateway(regionVal)
			ctx := context.Background()

			res := []interface{}{}
			var position *string
			for {
				restApisResp, err := svc.GetRestApis(ctx, &apigateway.GetRestApisInput{Position: position})
				if err != nil {
					return nil, errors.Wrap(err, "could not gather aws apigateway rest apis")
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

func (a *mqlAwsApigatewayRestapi) GetStages() ([]interface{}, error) {
	restApiId, err := a.Id()
	if err != nil {
		return nil, err
	}
	region, err := a.Region()
	if err != nil {
		return nil, err
	}
	at, err := awstransport(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	account, err := at.Account()
	if err != nil {
		return nil, err
	}
	svc := at.Apigateway(region)
	ctx := context.Background()

	// no pagination required
	stagesResp, err := svc.GetStages(ctx, &apigateway.GetStagesInput{RestApiId: &restApiId})
	if err != nil {
		return nil, errors.Wrap(err, "could not gather aws api gateway stages")
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
