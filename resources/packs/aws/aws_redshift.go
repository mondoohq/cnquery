package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/redshift"
	"github.com/aws/aws-sdk-go-v2/service/redshift/types"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.io/mondoo/motor/providers/aws"
	"go.mondoo.io/mondoo/resources/library/jobpool"
	"go.mondoo.io/mondoo/resources/packs/core"
)

func (r *mqlAwsRedshift) id() (string, error) {
	return "aws.redshift", nil
}

const (
	redshiftClusterArnPattern = "arn:aws:redshift:%s:%s:cluster/%s"
)

func (r *mqlAwsRedshift) GetClusters() ([]interface{}, error) {
	provider, err := awsProvider(r.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(r.getClusters(provider), 5)
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

func (r *mqlAwsRedshift) getClusters(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)

	account, err := provider.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.Redshift(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			var marker *string
			for {
				clusters, err := svc.DescribeClusters(ctx, &redshift.DescribeClustersInput{Marker: marker})
				if err != nil {
					return nil, err
				}
				for _, cluster := range clusters.Clusters {
					var names []interface{}
					for _, group := range cluster.ClusterParameterGroups {
						names = append(names, core.ToString(group.ParameterGroupName))
					}
					mqlDBInstance, err := r.MotorRuntime.CreateResource("aws.redshift.cluster",
						"arn", fmt.Sprintf(redshiftClusterArnPattern, regionVal, account.ID, core.ToString(cluster.ClusterIdentifier)),
						"name", core.ToString(cluster.ClusterIdentifier),
						"region", regionVal,
						"encrypted", cluster.Encrypted,
						"nodeType", core.ToString(cluster.NodeType),
						"allowVersionUpgrade", cluster.AllowVersionUpgrade,
						"preferredMaintenanceWindow", core.ToString(cluster.PreferredMaintenanceWindow),
						"automatedSnapshotRetentionPeriod", int64(cluster.AutomatedSnapshotRetentionPeriod),
						"publiclyAccessible", cluster.PubliclyAccessible,
						"clusterParameterGroupNames", names,
						"tags", redshiftTagsToMap(cluster.Tags),
					)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlDBInstance)
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

func redshiftTagsToMap(tags []types.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[core.ToString(tag.Key)] = core.ToString(tag.Value)
		}
	}

	return tagsMap
}

func (r *mqlAwsRedshiftCluster) id() (string, error) {
	return r.Arn()
}

func (r *mqlAwsRedshiftCluster) GetParameters() ([]interface{}, error) {
	clusterGroupNames, err := r.ClusterParameterGroupNames()
	if err != nil {
		return nil, err
	}
	region, err := r.Region()
	if err != nil {
		return nil, err
	}
	provider, err := awsProvider(r.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Redshift(region)
	ctx := context.Background()
	res := []types.Parameter{}
	for _, name := range clusterGroupNames {
		stringName := name.(string)
		params, err := svc.DescribeClusterParameters(ctx, &redshift.DescribeClusterParametersInput{ParameterGroupName: &stringName})
		if err != nil {
			return nil, err
		}
		res = append(res, params.Parameters...)
	}
	return core.JsonToDictSlice(res)
}

func (r *mqlAwsRedshiftCluster) GetLogging() (interface{}, error) {
	name, err := r.Name()
	if err != nil {
		return nil, err
	}
	region, err := r.Region()
	if err != nil {
		return nil, err
	}
	provider, err := awsProvider(r.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Redshift(region)
	ctx := context.Background()

	params, err := svc.DescribeLoggingStatus(ctx, &redshift.DescribeLoggingStatusInput{ClusterIdentifier: &name})
	if err != nil {
		return nil, err
	}
	return core.JsonToDict(params)
}
