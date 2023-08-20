package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/redshift"
	redshifttypes "github.com/aws/aws-sdk-go-v2/service/redshift/types"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers/aws/connection"
	"go.mondoo.com/cnquery/providers/aws/resources/jobpool"
	"go.mondoo.com/cnquery/types"
)

func (a *mqlAwsRedshift) id() (string, error) {
	return "aws.redshift", nil
}

const (
	redshiftClusterArnPattern = "arn:aws:redshift:%s:%s:cluster/%s"
)

func (a *mqlAwsRedshift) clusters() ([]interface{}, error) {
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

func (a *mqlAwsRedshift) getClusters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)

	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := conn.Redshift(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			var marker *string
			for {
				clusters, err := svc.DescribeClusters(ctx, &redshift.DescribeClustersInput{Marker: marker})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, cluster := range clusters.Clusters {
					var names []interface{}
					for _, group := range cluster.ClusterParameterGroups {
						names = append(names, toString(group.ParameterGroupName))
					}
					mqlDBInstance, err := a.MqlRuntime.CreateResource(a.MqlRuntime, "aws.redshift.cluster",
						map[string]*llx.RawData{
							"arn":                              llx.StringData(fmt.Sprintf(redshiftClusterArnPattern, regionVal, conn.AccountId(), toString(cluster.ClusterIdentifier))),
							"name":                             llx.StringData(toString(cluster.ClusterIdentifier)),
							"region":                           llx.StringData(regionVal),
							"encrypted":                        llx.BoolData(cluster.Encrypted),
							"nodeType":                         llx.StringData(toString(cluster.NodeType)),
							"allowVersionUpgrade":              llx.BoolData(cluster.AllowVersionUpgrade),
							"preferredMaintenanceWindow":       llx.StringData(toString(cluster.PreferredMaintenanceWindow)),
							"automatedSnapshotRetentionPeriod": llx.IntData(int64(cluster.AutomatedSnapshotRetentionPeriod)),
							"publiclyAccessible":               llx.BoolData(cluster.PubliclyAccessible),
							"clusterParameterGroupNames":       llx.ArrayData(names, types.String),
							"tags":                             llx.MapData(redshiftTagsToMap(cluster.Tags), types.String),
						})
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

func redshiftTagsToMap(tags []redshifttypes.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[toString(tag.Key)] = toString(tag.Value)
		}
	}

	return tagsMap
}

func (a *mqlAwsRedshiftCluster) id() (string, error) {
	return a.Arn.Data, nil
}

func initAwsRedshiftCluster(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	// if lenargs == 0 {
	// 	if ids := getAssetIdentifier(d.MqlResource().MotorRuntime); ids != nil {
	// 		(*args)["name"] = ids.name
	// 		(*args)["arn"] = ids.arn
	// 	}
	// }

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch redshift cluster")
	}

	// load all rds db instances
	obj, err := runtime.CreateResource(runtime, "aws.redshift", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	redshift := obj.(*mqlAwsRedshift)

	rawResources, err := redshift.clusters()
	if err != nil {
		return nil, nil, err
	}

	arnVal := args["arn"].Value.(string)
	for i := range rawResources {
		cluster := rawResources[i].(*mqlAwsRedshiftCluster)

		if cluster.Arn.Data == arnVal {
			return args, cluster, nil
		}
	}
	return nil, nil, errors.New("redshift cluster does not exist")
}

func (a *mqlAwsRedshiftCluster) parameters() ([]interface{}, error) {
	clusterGroupNames := a.ClusterParameterGroupNames.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Redshift(region)
	ctx := context.Background()
	res := []redshifttypes.Parameter{}
	for _, name := range clusterGroupNames {
		stringName := name.(string)
		params, err := svc.DescribeClusterParameters(ctx, &redshift.DescribeClusterParametersInput{ParameterGroupName: &stringName})
		if err != nil {
			return nil, err
		}
		res = append(res, params.Parameters...)
	}
	return convert.JsonToDictSlice(res)
}

func (a *mqlAwsRedshiftCluster) logging() (interface{}, error) {
	name := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Redshift(region)
	ctx := context.Background()

	params, err := svc.DescribeLoggingStatus(ctx, &redshift.DescribeLoggingStatusInput{ClusterIdentifier: &name})
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(params)
}
