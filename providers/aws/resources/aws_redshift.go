// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/redshift"
	redshifttypes "github.com/aws/aws-sdk-go-v2/service/redshift/types"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"

	"go.mondoo.com/cnquery/v12/types"
)

func (a *mqlAwsRedshift) id() (string, error) {
	return ResourceAwsRedshift, nil
}

const (
	redshiftClusterArnPattern = "arn:aws:redshift:%s:%s:cluster/%s"
)

func (a *mqlAwsRedshift) clusters() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getClusters(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
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
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("redshift>getClusters>calling aws with region %s", region)

			svc := conn.Redshift(region)
			ctx := context.Background()
			res := []any{}

			params := &redshift.DescribeClustersInput{}
			paginator := redshift.NewDescribeClustersPaginator(svc, params)
			for paginator.HasMorePages() {
				clusters, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, cluster := range clusters.Clusters {
					var names []any
					for _, group := range cluster.ClusterParameterGroups {
						names = append(names, convert.ToValue(group.ParameterGroupName))
					}

					if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(redshiftTagsToMap(cluster.Tags))) {
						continue
					}

					mqlDBInstance, err := CreateResource(a.MqlRuntime, ResourceAwsRedshiftCluster,
						map[string]*llx.RawData{
							"allowVersionUpgrade":              llx.BoolDataPtr(cluster.AllowVersionUpgrade),
							"arn":                              llx.StringData(fmt.Sprintf(redshiftClusterArnPattern, region, conn.AccountId(), convert.ToValue(cluster.ClusterIdentifier))),
							"automatedSnapshotRetentionPeriod": llx.IntDataDefault(cluster.AutomatedSnapshotRetentionPeriod, 0),
							"availabilityZone":                 llx.StringDataPtr(cluster.AvailabilityZone),
							"clusterParameterGroupNames":       llx.ArrayData(names, types.String),
							"clusterRevisionNumber":            llx.StringDataPtr(cluster.ClusterRevisionNumber),
							"clusterStatus":                    llx.StringDataPtr(cluster.ClusterStatus),
							"clusterSubnetGroupName":           llx.StringDataPtr(cluster.ClusterSubnetGroupName),
							"clusterVersion":                   llx.StringDataPtr(cluster.ClusterVersion),
							"createdAt":                        llx.TimeDataPtr(cluster.ClusterCreateTime),
							"dbName":                           llx.StringDataPtr(cluster.DBName),
							"encrypted":                        llx.BoolDataPtr(cluster.Encrypted),
							"enhancedVpcRouting":               llx.BoolDataPtr(cluster.EnhancedVpcRouting),
							"masterUsername":                   llx.StringDataPtr(cluster.MasterUsername),
							"name":                             llx.StringDataPtr(cluster.ClusterIdentifier),
							"nextMaintenanceWindowStartTime":   llx.TimeDataPtr(cluster.NextMaintenanceWindowStartTime),
							"nodeType":                         llx.StringDataPtr(cluster.NodeType),
							"numberOfNodes":                    llx.IntDataDefault(cluster.NumberOfNodes, 0),
							"preferredMaintenanceWindow":       llx.StringDataPtr(cluster.PreferredMaintenanceWindow),
							"publiclyAccessible":               llx.BoolDataPtr(cluster.PubliclyAccessible),
							"region":                           llx.StringData(region),
							"tags":                             llx.MapData(redshiftTagsToMap(cluster.Tags), types.String),
							"vpcId":                            llx.StringDataPtr(cluster.VpcId),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlDBInstance)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func redshiftTagsToMap(tags []redshifttypes.Tag) map[string]any {
	tagsMap := make(map[string]any)
	for _, tag := range tags {
		tagsMap[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
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

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch redshift cluster")
	}

	// load all rds db instances
	obj, err := CreateResource(runtime, "aws.redshift", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	redshift := obj.(*mqlAwsRedshift)

	rawResources := redshift.GetClusters()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	arnVal := args["arn"].Value.(string)
	for _, rawResource := range rawResources.Data {
		cluster := rawResource.(*mqlAwsRedshiftCluster)
		if cluster.Arn.Data == arnVal {
			return args, cluster, nil
		}
	}
	return nil, nil, errors.New("redshift cluster does not exist")
}

func (a *mqlAwsRedshiftCluster) parameters() ([]any, error) {
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

func (a *mqlAwsRedshiftCluster) logging() (any, error) {
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
