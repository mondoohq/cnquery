package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/emr"
	"github.com/aws/aws-sdk-go-v2/service/emr/types"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
	aws_transport "go.mondoo.io/mondoo/motor/providers/aws"
)

func (e *lumiAwsEmr) id() (string, error) {
	return "aws.emr", nil
}

func (e *lumiAwsEmr) GetClusters() ([]interface{}, error) {
	at, err := awstransport(e.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getClusters(at), 5)
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

func (e *lumiAwsEmrCluster) id() (string, error) {
	return e.Arn()
}

func (e *lumiAwsEmr) getClusters(at *aws_transport.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			svc := at.Emr(regionVal)
			ctx := context.Background()

			res := []interface{}{}

			var marker *string
			for {
				clusters, err := svc.ListClusters(ctx, &emr.ListClustersInput{Marker: marker})
				if err != nil {
					return nil, err
				}
				for _, cluster := range clusters.Clusters {
					jsonStatus, err := jsonToDict(cluster.Status)
					if err != nil {
						return nil, err
					}
					lumiCluster, err := e.MotorRuntime.CreateResource("aws.emr.cluster",
						"arn", toString(cluster.ClusterArn),
						"name", toString(cluster.Name),
						"normalizedInstanceHours", toInt64From32(cluster.NormalizedInstanceHours),
						"outpostArn", toString(cluster.OutpostArn),
						"status", jsonStatus,
						"id", toString(cluster.Id),
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiCluster)
				}
				if clusters.Marker == nil {
					break
				}
				marker = clusters.Marker
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (e *lumiAwsEmrCluster) GetMasterInstances() ([]interface{}, error) {
	arn, err := e.Arn()
	if err != nil {
		return nil, err
	}
	id, err := e.Id()
	if err != nil {
		return nil, err
	}
	region, err := getRegionFromArn(arn)
	if err != nil {
		return nil, err
	}
	res := []types.Instance{}
	at, err := awstransport(e.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.Emr(region)
	ctx := context.Background()
	var marker *string
	for {
		instances, err := svc.ListInstances(ctx, &emr.ListInstancesInput{
			Marker:             marker,
			ClusterId:          &id,
			InstanceGroupTypes: []types.InstanceGroupType{"MASTER"},
		})
		if err != nil {
			return nil, err
		}
		res = append(res, instances.Instances...)
		if instances.Marker == nil {
			break
		}
		marker = instances.Marker
	}
	return jsonToDictSlice(res)
}
