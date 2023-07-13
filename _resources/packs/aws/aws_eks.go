package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (e *mqlAwsEks) id() (string, error) {
	return "aws.eks", nil
}

func (e *mqlAwsEks) GetClusters() ([]interface{}, error) {
	provider, err := awsProvider(e.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(e.getClusters(provider), 5)
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

func (e *mqlAwsEks) getClusters(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.Eks(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			describeClusterRes, err := svc.ListClusters(ctx, &eks.ListClustersInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
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
				encryptionConfig, _ := core.JsonToDictSlice(cluster.EncryptionConfig)
				logging, _ := core.JsonToDict(cluster.Logging)
				kubernetesNetworkConfig, _ := core.JsonToDict(cluster.KubernetesNetworkConfig)
				vpcConfig, _ := core.JsonToDict(cluster.ResourcesVpcConfig)

				args := []interface{}{
					"arn", core.ToString(cluster.Arn),
					"name", core.ToString(cluster.Name),
					"region", regionVal,
					"version", core.ToString(cluster.Version),
					"platformVersion", core.ToString(cluster.PlatformVersion),
					"tags", core.StrMapToInterface(cluster.Tags),
					"status", string(cluster.Status),
					"encryptionConfig", encryptionConfig,
					"createdAt", cluster.CreatedAt,
					"endpoint", core.ToString(cluster.Endpoint),
					"logging", logging,
					"networkConfig", kubernetesNetworkConfig,
					"resourcesVpcConfig", vpcConfig,
				}

				mqlFilesystem, err := e.MotorRuntime.CreateResource("aws.eks.cluster", args...)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlFilesystem)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (e *mqlAwsEksCluster) id() (string, error) {
	return e.Arn()
}
