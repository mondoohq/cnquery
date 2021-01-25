package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/emr"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

func (e *lumiAwsEmr) id() (string, error) {
	return "aws.emr", nil
}

func (e *lumiAwsEmr) GetClusters() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getClusters(), 5)
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

func (e *lumiAwsEmr) getClusters() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(e.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
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
				clusters, err := svc.ListClustersRequest(&emr.ListClustersInput{Marker: marker}).Send(ctx)
				if err != nil {
					return nil, err
				}
				for _, cluster := range clusters.Clusters {
					jsonStatus, err := jsonToDict(cluster.Status)
					if err != nil {
						return nil, err
					}
					lumiCluster, err := e.Runtime.CreateResource("aws.emr.cluster",
						"arn", toString(cluster.ClusterArn),
						"name", toString(cluster.Name),
						"normalizedInstanceHours", toInt64(cluster.NormalizedInstanceHours),
						"outpostArn", toString(cluster.OutpostArn),
						"status", jsonStatus,
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
