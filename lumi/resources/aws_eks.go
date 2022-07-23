package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"
)

func (e *lumiAwsEks) id() (string, error) {
	return "aws.eks", nil
}

func (e *lumiAwsEks) GetClusters() ([]interface{}, error) {
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

func (e *lumiAwsEks) getClusters(at *aws_transport.Transport) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.Eks(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			describeClusterRes, err := svc.ListClusters(ctx, &eks.ListClustersInput{})
			if err != nil {
				return nil, err
			}

			if describeClusterRes == nil {
				return jobpool.JobResult(res), nil
			}

			for i := range describeClusterRes.Clusters {
				clusterName := describeClusterRes.Clusters[i]

				// get cluster details
				log.Debug().Str("cluster", clusterName).Str("region", region).Msg("get info for cluster")
				describeClusterOutput, err := svc.DescribeCluster(ctx, &eks.DescribeClusterInput{
					Name: aws.String(clusterName),
				})
				if err != nil {
					return nil, err
				}

				if describeClusterOutput == nil {
					continue
				}

				cluster := describeClusterOutput.Cluster
				encryptionConfig, _ := jsonToDictSlice(cluster.EncryptionConfig)
				logging, _ := jsonToDict(cluster.Logging)
				kubernetesNetworkConfig, _ := jsonToDict(cluster.KubernetesNetworkConfig)
				vpcConfig, _ := jsonToDict(cluster.ResourcesVpcConfig)

				args := []interface{}{
					"arn", toString(cluster.Arn),
					"name", toString(cluster.Name),
					"region", regionVal,
					"version", toString(cluster.Version),
					"platformVersion", toString(cluster.PlatformVersion),
					"tags", mapTagsToLumiMapTags(cluster.Tags),
					"status", string(cluster.Status),
					"encryptionConfig", encryptionConfig,
					"createdAt", cluster.CreatedAt,
					"endpoint", toString(cluster.Endpoint),
					"logging", logging,
					"networkConfig", kubernetesNetworkConfig,
					"resourcesVpcConfig", vpcConfig,
				}

				lumiFilesystem, err := e.MotorRuntime.CreateResource("aws.eks.cluster", args...)
				if err != nil {
					return nil, err
				}
				res = append(res, lumiFilesystem)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (e *lumiAwsEksCluster) id() (string, error) {
	return e.Arn()
}
