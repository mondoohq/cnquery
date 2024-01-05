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
	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v9/providers/aws/connection"

	"go.mondoo.com/cnquery/v9/types"
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
			log.Debug().Msgf("rds>getDbInstances>calling aws with region %s", regionVal)

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
							"arn":                           llx.StringDataPtr(dbInstance.DBInstanceArn),
							"autoMinorVersionUpgrade":       llx.BoolDataPtr(dbInstance.AutoMinorVersionUpgrade),
							"availabilityZone":              llx.StringDataPtr(dbInstance.AvailabilityZone),
							"backupRetentionPeriod":         llx.IntData(convert.ToInt64From32(dbInstance.BackupRetentionPeriod)),
							"createdTime":                   llx.TimeDataPtr(dbInstance.InstanceCreateTime),
							"dbInstanceClass":               llx.StringDataPtr(dbInstance.DBInstanceClass),
							"dbInstanceIdentifier":          llx.StringDataPtr(dbInstance.DBInstanceIdentifier),
							"deletionProtection":            llx.BoolDataPtr(dbInstance.DeletionProtection),
							"enabledCloudwatchLogsExports":  llx.ArrayData(stringSliceInterface, types.String),
							"engine":                        llx.StringDataPtr(dbInstance.Engine),
							"engineVersion":                 llx.StringDataPtr(dbInstance.EngineVersion),
							"enhancedMonitoringResourceArn": llx.StringDataPtr(dbInstance.EnhancedMonitoringResourceArn),
							"id":                            llx.StringDataPtr(dbInstance.DBInstanceIdentifier),
							"multiAZ":                       llx.BoolDataPtr(dbInstance.MultiAZ),
							"name":                          llx.StringDataPtr(dbInstance.DBName),
							"publiclyAccessible":            llx.BoolDataPtr(dbInstance.PubliclyAccessible),
							"region":                        llx.StringData(regionVal),
							"securityGroups":                llx.ArrayData(sgs, types.Resource("aws.ec2.securitygroup")),
							"status":                        llx.StringDataPtr(dbInstance.DBInstanceStatus),
							"storageAllocated":              llx.IntData(convert.ToInt64From32(dbInstance.AllocatedStorage)),
							"storageEncrypted":              llx.BoolDataPtr(dbInstance.StorageEncrypted),
							"storageIops":                   llx.IntData(convert.ToInt64From32(dbInstance.Iops)),
							"storageType":                   llx.StringDataPtr(dbInstance.StorageType),
							"tags":                          llx.MapData(rdsTagsToMap(dbInstance.TagList), types.String),
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

	rawResources := rds.GetDbInstances()
	if err != nil {
		return nil, nil, err
	}

	arnVal := args["arn"].Value.(string)
	for i := range rawResources.Data {
		dbInstance := rawResources.Data[i].(*mqlAwsRdsDbinstance)
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
			log.Debug().Msgf("rds>getDbClusters>calling aws with region %s", regionVal)

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
					mqlRdsDbInstances := []interface{}{}
					for _, instance := range cluster.DBClusterMembers {
						mqlInstance, err := NewResource(a.MqlRuntime, "aws.rds.dbinstance",
							map[string]*llx.RawData{
								"arn": llx.StringData(fmt.Sprintf(rdsInstanceArnPattern, regionVal, conn.AccountId(), convert.ToString(instance.DBInstanceIdentifier))),
							})
						if err != nil {
							return nil, err
						}
						mqlRdsDbInstances = append(mqlRdsDbInstances, mqlInstance)
					}
					sgs := []interface{}{}
					for i := range cluster.VpcSecurityGroups {
						// NOTE: this will create the resource and determine the data in its init method
						mqlSg, err := NewResource(a.MqlRuntime, "aws.ec2.securitygroup",
							map[string]*llx.RawData{
								"arn": llx.StringData(fmt.Sprintf(securityGroupArnPattern, regionVal, conn.AccountId(), convert.ToString(cluster.VpcSecurityGroups[i].VpcSecurityGroupId))),
							})
						if err != nil {
							return nil, err
						}
						sgs = append(sgs, mqlSg.(*mqlAwsEc2Securitygroup))
					}
					stringSliceAZs := []interface{}{}
					for _, zone := range cluster.AvailabilityZones {
						stringSliceAZs = append(stringSliceAZs, zone)
					}
					mqlDbCluster, err := CreateResource(a.MqlRuntime, "aws.rds.dbcluster",
						map[string]*llx.RawData{
							"arn":                     llx.StringDataPtr(cluster.DBClusterArn),
							"autoMinorVersionUpgrade": llx.BoolDataPtr(cluster.AutoMinorVersionUpgrade),
							"availabilityZones":       llx.ArrayData(stringSliceAZs, types.String),
							"backupRetentionPeriod":   llx.IntData(convert.ToInt64From32(cluster.BackupRetentionPeriod)),
							"clusterDbInstanceClass":  llx.StringDataPtr(cluster.DBClusterInstanceClass),
							"createdTime":             llx.TimeDataPtr(cluster.ClusterCreateTime),
							"deletionProtection":      llx.BoolDataPtr(cluster.DeletionProtection),
							"endpoint":                llx.StringDataPtr(cluster.Endpoint),
							"engine":                  llx.StringDataPtr(cluster.Engine),
							"engineVersion":           llx.StringDataPtr(cluster.EngineVersion),
							"id":                      llx.StringDataPtr(cluster.DBClusterIdentifier),
							"members":                 llx.ArrayData(mqlRdsDbInstances, types.Resource("aws.rds.dbinstance")),
							"multiAZ":                 llx.BoolDataPtr(cluster.MultiAZ),
							"port":                    llx.IntData(convert.ToInt64From32(cluster.Port)),
							"publiclyAccessible":      llx.BoolDataPtr(cluster.PubliclyAccessible),
							"region":                  llx.StringData(regionVal),
							"securityGroups":          llx.ArrayData(sgs, types.Resource("aws.ec2.securitygroup")),
							"status":                  llx.StringDataPtr(cluster.Status),
							"storageAllocated":        llx.IntData(convert.ToInt64From32(cluster.AllocatedStorage)),
							"storageEncrypted":        llx.BoolDataPtr(cluster.StorageEncrypted),
							"storageIops":             llx.IntData(convert.ToInt64From32(cluster.Iops)),
							"storageType":             llx.StringDataPtr(cluster.StorageType),
							"tags":                    llx.MapData(rdsTagsToMap(cluster.TagList), types.String),
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
					"arn":               llx.StringDataPtr(snapshot.DBClusterSnapshotArn),
					"id":                llx.StringDataPtr(snapshot.DBClusterSnapshotIdentifier),
					"type":              llx.StringDataPtr(snapshot.SnapshotType),
					"region":            llx.StringData(region),
					"encrypted":         llx.BoolDataPtr(snapshot.StorageEncrypted),
					"isClusterSnapshot": llx.BoolData(true),
					"tags":              llx.MapData(rdsTagsToMap(snapshot.TagList), types.String),
					"engine":            llx.StringDataPtr(snapshot.Engine),
					"status":            llx.StringDataPtr(snapshot.Status),
					"allocatedStorage":  llx.IntData(convert.ToInt64From32(snapshot.AllocatedStorage)),
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
					"arn":               llx.StringDataPtr(snapshot.DBSnapshotArn),
					"id":                llx.StringDataPtr(snapshot.DBSnapshotIdentifier),
					"type":              llx.StringDataPtr(snapshot.SnapshotType),
					"region":            llx.StringData(region),
					"encrypted":         llx.BoolDataPtr(snapshot.Encrypted),
					"isClusterSnapshot": llx.BoolData(false),
					"tags":              llx.MapData(rdsTagsToMap(snapshot.TagList), types.String),
					"engine":            llx.StringDataPtr(snapshot.Engine),
					"status":            llx.StringDataPtr(snapshot.Status),
					"allocatedStorage":  llx.IntData(convert.ToInt64From32(snapshot.AllocatedStorage)),
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
