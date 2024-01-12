// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v10/providers/aws/connection"

	"go.mondoo.com/cnquery/v10/types"
)

func (a *mqlAwsEks) id() (string, error) {
	return "aws.eks", nil
}

func (a *mqlAwsEks) clusters() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getClusters(conn), 5)
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

func (a *mqlAwsEks) getClusters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}} // return the error
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("eks>getClusters>calling aws with region %s", regionVal)

			svc := conn.Eks(regionVal)
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
				encryptionConfig, _ := convert.JsonToDictSlice(cluster.EncryptionConfig)
				logging, _ := convert.JsonToDict(cluster.Logging)
				kubernetesNetworkConfig, _ := convert.JsonToDict(cluster.KubernetesNetworkConfig)
				vpcConfig, _ := convert.JsonToDict(cluster.ResourcesVpcConfig)

				args := map[string]*llx.RawData{
					"arn":                llx.StringDataPtr(cluster.Arn),
					"name":               llx.StringDataPtr(cluster.Name),
					"region":             llx.StringData(regionVal),
					"version":            llx.StringDataPtr(cluster.Version),
					"platformVersion":    llx.StringDataPtr(cluster.PlatformVersion),
					"tags":               llx.MapData(strMapToInterface(cluster.Tags), types.String),
					"status":             llx.StringData(string(cluster.Status)),
					"encryptionConfig":   llx.ArrayData(encryptionConfig, types.Any),
					"createdAt":          llx.TimeDataPtr(cluster.CreatedAt),
					"endpoint":           llx.StringDataPtr(cluster.Endpoint),
					"logging":            llx.MapData(logging, types.Any),
					"networkConfig":      llx.MapData(kubernetesNetworkConfig, types.Any),
					"resourcesVpcConfig": llx.MapData(vpcConfig, types.Any),
				}

				mqlFilesystem, err := CreateResource(a.MqlRuntime, "aws.eks.cluster", args)
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

func (a *mqlAwsEksCluster) id() (string, error) {
	return a.Arn.Data, nil
}
