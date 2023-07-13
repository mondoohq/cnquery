package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/databasemigrationservice"
	"github.com/aws/aws-sdk-go-v2/service/databasemigrationservice/types"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (d *mqlAwsDms) id() (string, error) {
	return "aws.dms", nil
}

func (d *mqlAwsDms) GetReplicationInstances() ([]interface{}, error) {
	provider, err := awsProvider(d.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []types.ReplicationInstance{}
	poolOfJobs := jobpool.CreatePool(d.getReplicationInstances(provider), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]types.ReplicationInstance)...)
	}
	return core.JsonToDictSlice(res)
}

func (d *mqlAwsDms) getReplicationInstances(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.Dms(regionVal)
			ctx := context.Background()
			replicationInstancesAggregated := []types.ReplicationInstance{}

			var marker *string
			for {
				replicationInstances, err := svc.DescribeReplicationInstances(ctx, &databasemigrationservice.DescribeReplicationInstancesInput{Marker: marker})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return tasks, nil
					}
					return nil, err
				}
				replicationInstancesAggregated = append(replicationInstancesAggregated, replicationInstances.ReplicationInstances...)

				if replicationInstances.Marker == nil {
					break
				}
				marker = replicationInstances.Marker
			}
			return jobpool.JobResult(replicationInstancesAggregated), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}
