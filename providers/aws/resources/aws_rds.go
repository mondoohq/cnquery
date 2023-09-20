// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/providers/aws/connection"

	"go.mondoo.com/cnquery/types"
)

func (a *mqlAwsRds) id() (string, error) {
	return "aws.rds", nil
}

func (a *mqlAwsRds) dbInstances() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getDbInstances(conn), 5)
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

func (a *mqlAwsRds) getDbInstances(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			res := []interface{}{}
			svc := conn.Rds(regionVal)
			ctx := context.Background()

			var marker *string
			for {
				dbInstances, err := svc.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{Marker: marker})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, dbInstance := range dbInstances.DBInstances {
					stringSliceInterface := []interface{}{}
					for _, logExport := range dbInstance.EnabledCloudwatchLogsExports {
						stringSliceInterface = append(stringSliceInterface, logExport)
					}
					sgs := []interface{}{}
					for i := range dbInstance.VpcSecurityGroups {
						// NOTE: this will create the resource and determine the data in its init method
						mqlSg, err := NewResource(a.MqlRuntime, "aws.ec2.securitygroup",
							map[string]*llx.RawData{
								"arn": llx.StringData(fmt.Sprintf(securityGroupArnPattern, regionVal, conn.AccountId(), convert.ToString(dbInstance.VpcSecurityGroups[i].VpcSecurityGroupId))),
							})
						if err != nil {
							return nil, err
						}
						sgs = append(sgs, mqlSg.(*mqlAwsEc2Securitygroup))
					}

					mqlDBInstance, err := CreateResource(a.MqlRuntime, "aws.rds.dbinstance",
						map[string]*llx.RawData{
							"arn":                           llx.StringData(convert.ToString(dbInstance.DBInstanceArn)),
							"name":                          llx.StringData(convert.ToString(dbInstance.DBName)),
							"backupRetentionPeriod":         llx.IntData(int64(dbInstance.BackupRetentionPeriod)),
							"storageEncrypted":              llx.BoolData(dbInstance.StorageEncrypted),
							"region":                        llx.StringData(regionVal),
							"publiclyAccessible":            llx.BoolData(dbInstance.PubliclyAccessible),
							"enabledCloudwatchLogsExports":  llx.ArrayData(stringSliceInterface, types.String),
							"enhancedMonitoringResourceArn": llx.StringData(convert.ToString(dbInstance.EnhancedMonitoringResourceArn)),
							"multiAZ":                       llx.BoolData(dbInstance.MultiAZ),
							"id":                            llx.StringData(convert.ToString(dbInstance.DBInstanceIdentifier)),
							"deletionProtection":            llx.BoolData(dbInstance.DeletionProtection),
							"tags":                          llx.MapData(rdsTagsToMap(dbInstance.TagList), types.String),
							"dbInstanceClass":               llx.StringData(convert.ToString(dbInstance.DBInstanceClass)),
							"dbInstanceIdentifier":          llx.StringData(convert.ToString(dbInstance.DBInstanceIdentifier)),
							"engine":                        llx.StringData(convert.ToString(dbInstance.Engine)),
							"securityGroups":                llx.ArrayData(sgs, types.Resource("aws.ec2.securitygroup")),
							"status":                        llx.StringData(convert.ToString(dbInstance.DBInstanceStatus)),
							"autoMinorVersionUpgrade":       llx.BoolData(dbInstance.AutoMinorVersionUpgrade),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlDBInstance)
				}
				if dbInstances.Marker == nil {
					break
				}
				marker = dbInstances.Marker
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func rdsTagsToMap(tags []rdstypes.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[convert.ToString(tag.Key)] = convert.ToString(tag.Value)
		}
	}

	return tagsMap
}

func (a *mqlAwsRds) dbClusters() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getDbClusters(conn), 5)
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

func initAwsRdsDbinstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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
		return nil, nil, errors.New("arn required to fetch rds db instance")
	}

	// load all rds db instances
	obj, err := CreateResource(runtime, "aws.rds", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	rds := obj.(*mqlAwsRds)

	rawResources, err := rds.dbInstances()
	if err != nil {
		return nil, nil, err
	}

	arnVal := args["arn"].Value.(string)
	for i := range rawResources {
		dbInstance := rawResources[i].(*mqlAwsRdsDbinstance)
		if dbInstance.Arn.Data == arnVal {
			return args, dbInstance, nil
		}
	}
	return nil, nil, errors.New("rds db instance does not exist")
}

func (a *mqlAwsRds) getDbClusters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			res := []interface{}{}
			svc := conn.Rds(regionVal)
			ctx := context.Background()

			var marker *string
			for {
				dbClusters, err := svc.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{Marker: marker})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, cluster := range dbClusters.DBClusters {
					// mqlRdsDbInstances := []*mqlRdsDbInstance{}
					// for _, instance := range cluster.DBClusterMembers {
					// 	mqlInstance, err := NewResource(a.MqlRuntime, "aws.rds.dbinstance",
					// 		map[string]*llx.RawData{
					// 			"arn": llx.StringData(fmt.Sprintf(rdsInstanceArnPattern, regionVal, conn, conn.AccountId(), convert.ToString(instance.DBInstanceIdentifier))),
					// 		})
					// 	if err != nil {
					// 		return nil, err
					// 	}
					// 	mqlRdsDbInstances = append(mqlRdsDbInstances, mqlInstance)
					// }
					mqlDbCluster, err := CreateResource(a.MqlRuntime, "aws.rds.dbcluster",
						map[string]*llx.RawData{
							"arn":    llx.StringData(convert.ToString(cluster.DBClusterArn)),
							"region": llx.StringData(regionVal),
							"id":     llx.StringData(convert.ToString(cluster.DBClusterIdentifier)),
							// "members": mqlRdsDbInstances,
							"tags": llx.MapData(rdsTagsToMap(cluster.TagList), types.String),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlDbCluster)
				}

				if dbClusters.Marker == nil {
					break
				}
				marker = dbClusters.Marker
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsRdsDbcluster) snapshots() ([]interface{}, error) {
	dbClusterId := a.Id.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Rds(region)
	ctx := context.Background()
	res := []interface{}{}

	var marker *string
	for {
		snapshots, err := svc.DescribeDBClusterSnapshots(ctx, &rds.DescribeDBClusterSnapshotsInput{DBClusterIdentifier: &dbClusterId, Marker: marker})
		if err != nil {
			return nil, err
		}
		for _, snapshot := range snapshots.DBClusterSnapshots {
			mqlDbSnapshot, err := CreateResource(a.MqlRuntime, "aws.rds.snapshot",
				map[string]*llx.RawData{
					"arn":               llx.StringData(convert.ToString(snapshot.DBClusterSnapshotArn)),
					"id":                llx.StringData(convert.ToString(snapshot.DBClusterSnapshotIdentifier)),
					"type":              llx.StringData(convert.ToString(snapshot.SnapshotType)),
					"region":            llx.StringData(region),
					"encrypted":         llx.BoolData(snapshot.StorageEncrypted),
					"isClusterSnapshot": llx.BoolData(true),
					"tags":              llx.MapData(rdsTagsToMap(snapshot.TagList), types.String),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlDbSnapshot)
		}
		if snapshots.Marker == nil {
			break
		}
		marker = snapshots.Marker
	}
	return res, nil
}

func (a *mqlAwsRdsDbinstance) snapshots() ([]interface{}, error) {
	instanceId := a.Id.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Rds(region)
	ctx := context.Background()
	res := []interface{}{}

	var marker *string
	for {
		snapshots, err := svc.DescribeDBSnapshots(ctx, &rds.DescribeDBSnapshotsInput{DBInstanceIdentifier: &instanceId, Marker: marker})
		if err != nil {
			return nil, err
		}
		for _, snapshot := range snapshots.DBSnapshots {
			mqlDbSnapshot, err := CreateResource(a.MqlRuntime, "aws.rds.snapshot",
				map[string]*llx.RawData{
					"arn":               llx.StringData(convert.ToString(snapshot.DBSnapshotArn)),
					"id":                llx.StringData(convert.ToString(snapshot.DBSnapshotIdentifier)),
					"type":              llx.StringData(convert.ToString(snapshot.SnapshotType)),
					"region":            llx.StringData(region),
					"encrypted":         llx.BoolData(snapshot.Encrypted),
					"isClusterSnapshot": llx.BoolData(false),
					"tags":              llx.MapData(rdsTagsToMap(snapshot.TagList), types.String),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlDbSnapshot)
		}
		if snapshots.Marker == nil {
			break
		}
		marker = snapshots.Marker
	}
	return res, nil
}

func (a *mqlAwsRdsDbinstance) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsRdsDbcluster) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsRdsSnapshot) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsRdsSnapshot) attributes() ([]interface{}, error) {
	snapshotId := a.Id.Data
	region := a.Region.Data
	isCluster := a.IsClusterSnapshot.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Rds(region)
	ctx := context.Background()
	if isCluster == true {
		snapshotAttributes, err := svc.DescribeDBClusterSnapshotAttributes(ctx, &rds.DescribeDBClusterSnapshotAttributesInput{DBClusterSnapshotIdentifier: &snapshotId})
		if err != nil {
			return nil, err
		}
		return convert.JsonToDictSlice(snapshotAttributes.DBClusterSnapshotAttributesResult.DBClusterSnapshotAttributes)
	}
	snapshotAttributes, err := svc.DescribeDBSnapshotAttributes(ctx, &rds.DescribeDBSnapshotAttributesInput{DBSnapshotIdentifier: &snapshotId})
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(snapshotAttributes.DBSnapshotAttributesResult.DBSnapshotAttributes)
}
