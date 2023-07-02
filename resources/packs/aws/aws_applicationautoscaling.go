package aws

import (
	"context"
	"fmt"

	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling/types"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (a *mqlAwsApplicationAutoscaling) id() (string, error) {
	n, err := a.Namespace()
	if err != nil {
		return "", errors.Join(err, errors.New("namespace required. please provide an aws service as argument. valid values: [comprehend, rds, sagemaker, appstream, elasticmapreduce, dynamodb, lambda, ecs, cassandra, ec2, neptune, kafka, custom-resource, elasticache]"))
	}
	return "aws.applicationAutoscaling." + n, nil
}

func (l *mqlAwsApplicationautoscalingTarget) id() (string, error) {
	return l.Arn()
}

func (a *mqlAwsApplicationAutoscaling) GetScalableTargets() ([]interface{}, error) {
	provider, err := awsProvider(a.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	namespace, err := a.Namespace()
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getTargets(provider, types.ServiceNamespace(namespace)), 5)
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

func (a *mqlAwsApplicationAutoscaling) getTargets(provider *aws_provider.Provider, namespace types.ServiceNamespace) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	account, err := provider.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.ApplicationAutoscaling(regionVal)
			ctx := context.Background()

			res := []interface{}{}
			nextToken := aws.String("no_token_to_start_with")
			params := &applicationautoscaling.DescribeScalableTargetsInput{ServiceNamespace: namespace}
			for nextToken != nil {
				resp, err := svc.DescribeScalableTargets(ctx, params)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, errors.Join(err, errors.New("could not gather application autoscaling scalable targets"))
				}

				for _, target := range resp.ScalableTargets {
					targetState, err := core.JsonToDict(target.SuspendedState)
					if err != nil {
						return nil, err
					}
					mqlSTarget, err := a.MotorRuntime.CreateResource("aws.applicationautoscaling.target",
						"arn", fmt.Sprintf("arn:aws:application-autoscaling:%s:%s:%s/%s", regionVal, account.ID, namespace, core.ToString(target.ResourceId)),
						"namespace", string(target.ServiceNamespace),
						"scalableDimension", string(target.ScalableDimension),
						"minCapacity", core.ToInt64From32(target.MinCapacity),
						"maxCapacity", core.ToInt64From32(target.MaxCapacity),
						"suspendedState", targetState,
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlSTarget)
				}
				nextToken = resp.NextToken
				if resp.NextToken != nil {
					params.NextToken = nextToken
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}
