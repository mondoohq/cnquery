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
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v10/providers/aws/connection"

	"go.mondoo.com/cnquery/v10/types"
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
			log.Debug().Msgf("redshift>getClusters>calling aws with region %s", regionVal)

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
						names = append(names, convert.ToString(group.ParameterGroupName))
					}
					mqlDBInstance, err := CreateResource(a.MqlRuntime, "aws.redshift.cluster",
						map[string]*llx.RawData{
							"allowVersionUpgrade":              llx.BoolDataPtr(cluster.AllowVersionUpgrade),
							"arn":                              llx.StringData(fmt.Sprintf(redshiftClusterArnPattern, regionVal, conn.AccountId(), convert.ToString(cluster.ClusterIdentifier))),
							"automatedSnapshotRetentionPeriod": llx.IntData(convert.ToInt64From32(cluster.AutomatedSnapshotRetentionPeriod)),
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
							"numberOfNodes":                    llx.IntData(convert.ToInt64From32(cluster.NumberOfNodes)),
							"preferredMaintenanceWindow":       llx.StringDataPtr(cluster.PreferredMaintenanceWindow),
							"publiclyAccessible":               llx.BoolDataPtr(cluster.PubliclyAccessible),
							"region":                           llx.StringData(regionVal),
							"tags":                             llx.MapData(redshiftTagsToMap(cluster.Tags), types.String),
							"vpcId":                            llx.StringDataPtr(cluster.VpcId),
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
			tagsMap[convert.ToString(tag.Key)] = convert.ToString(tag.Value)
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
	for i := range rawResources.Data {
		cluster := rawResources.Data[i].(*mqlAwsRedshiftCluster)

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
