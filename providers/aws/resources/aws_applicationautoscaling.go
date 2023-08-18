package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	aatypes "github.com/aws/aws-sdk-go-v2/service/applicationautoscaling/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/types"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/aws/connection"
	"go.mondoo.com/cnquery/providers/aws/resources/jobpool"
)

func (a *mqlAwsApplicationAutoscaling) id() (string, error) {
	return "aws.applicationAutoscaling." + a.Namespace.Data, nil
}

func (a *mqlAwsApplicationautoscalingTarget) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsApplicationAutoscaling) scalableTargets() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	namespace := a.Namespace.Data

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getTargets(conn, aatypes.ServiceNamespace(namespace)), 5)
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

func (a *mqlAwsApplicationAutoscaling) getTargets(conn *connection.AwsConnection, namespace aatypes.ServiceNamespace) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := conn.ApplicationAutoscaling(regionVal)
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
					return nil, errors.Wrap(err, "could not gather application autoscaling scalable targets")
				}

				for _, target := range resp.ScalableTargets {
					targetState, err := convert.JsonToDict(target.SuspendedState)
					if err != nil {
						return nil, err
					}
					mqlSTarget, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.applicationautoscaling.target",
						map[string]*llx.RawData{
							"arn":               llx.StringData(fmt.Sprintf("arn:aws:application-autoscaling:%s:%s:%s/%s", regionVal, conn.AccountId(), namespace, toString(target.ResourceId))),
							"namespace":         llx.StringData(string(target.ServiceNamespace)),
							"scalableDimension": llx.StringData(string(target.ScalableDimension)),
							"minCapacity":       llx.IntData(toInt64From32(target.MinCapacity)),
							"maxCapacity":       llx.IntData(toInt64From32(target.MaxCapacity)),
							"suspendedState":    llx.MapData(targetState, types.Any),
						})
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
