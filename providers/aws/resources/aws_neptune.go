// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/service/neptune"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/types"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
)

func (a *mqlAwsNeptune) id() (string, error) {
	return "aws.neptune", nil
}

func (a *mqlAwsNeptune) clusters() ([]interface{}, error) {
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
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
		}
	}

	return res, nil
}

func (a *mqlAwsNeptune) getDbClusters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("neptune>getDbClusters>calling aws with region %s", regionVal)

			svc := conn.Neptune(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			var marker *string
			for {
				cluster, err := svc.DescribeDBClusters(ctx, &neptune.DescribeDBClustersInput{
					Marker: marker,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				if len(cluster.DBClusters) == 0 {
					return nil, nil
				}
				for i := range cluster.DBClusters {
					cluster := cluster.DBClusters[i]

					mqlCluster, err := CreateResource(a.MqlRuntime, "aws.neptune.cluster",
						map[string]*llx.RawData{
							"__id":                             llx.StringDataPtr(cluster.DBClusterArn),
							"arn":                              llx.StringDataPtr(cluster.DBClusterArn),
							"name":                             llx.StringDataPtr(cluster.DatabaseName),
							"clusterIdentifier":                llx.StringDataPtr(cluster.DBClusterIdentifier),
							"globalClusterIdentifier":          llx.StringDataPtr(cluster.GlobalClusterIdentifier),
							"engine":                           llx.StringDataPtr(cluster.Engine),
							"engineVersion":                    llx.StringDataPtr(cluster.EngineVersion),
							"kmsKeyId":                         llx.StringDataPtr(cluster.KmsKeyId),
							"region":                           llx.StringData(regionVal),
							"automaticRestartTime":             llx.TimeDataPtr(cluster.AutomaticRestartTime),
							"availabilityZones":                llx.ArrayData(convert.SliceAnyToInterface(cluster.AvailabilityZones), types.String),
							"backupRetentionPeriod":            llx.IntDataPtr(cluster.BackupRetentionPeriod),
							"createdAt":                        llx.TimeDataPtr(cluster.ClusterCreateTime),
							"crossAccountClone":                llx.BoolDataPtr(cluster.CrossAccountClone),
							"clusterParameterGroup":            llx.StringDataPtr(cluster.DBClusterParameterGroup),
							"subnetGroup":                      llx.StringDataPtr(cluster.DBSubnetGroup),
							"clusterResourceId":                llx.StringDataPtr(cluster.DbClusterResourceId),
							"deletionProtection":               llx.BoolDataPtr(cluster.DeletionProtection),
							"earliestRestorableTime":           llx.TimeDataPtr(cluster.EarliestRestorableTime),
							"endpoint":                         llx.StringDataPtr(cluster.Endpoint),
							"iamDatabaseAuthenticationEnabled": llx.BoolDataPtr(cluster.IAMDatabaseAuthenticationEnabled),
							"latestRestorableTime":             llx.TimeDataPtr(cluster.LatestRestorableTime),
							"masterUsername":                   llx.StringDataPtr(cluster.MasterUsername),
							"multiAZ":                          llx.BoolDataPtr(cluster.MultiAZ),
							"port":                             llx.IntDataPtr(cluster.Port),
							"preferredBackupWindow":            llx.StringDataPtr(cluster.PreferredBackupWindow),
							"preferredMaintenanceWindow":       llx.StringDataPtr(cluster.PreferredMaintenanceWindow),
							"status":                           llx.StringDataPtr(cluster.Status),
							"storageEncrypted":                 llx.BoolDataPtr(cluster.StorageEncrypted),
							"storageType":                      llx.StringDataPtr(cluster.StorageType),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlCluster)
				}
				if cluster.Marker == nil {
					break
				}
				marker = cluster.Marker
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsNeptune) instances() ([]interface{}, error) {
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
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
		}
	}

	return res, nil
}

func (a *mqlAwsNeptune) getDbInstances(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("neptune>getDbInstances>calling aws with region %s", regionVal)

			svc := conn.Neptune(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			var marker *string
			for {
				cluster, err := svc.DescribeDBInstances(ctx, &neptune.DescribeDBInstancesInput{
					Marker: marker,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				if len(cluster.DBInstances) == 0 {
					return nil, nil
				}
				for i := range cluster.DBInstances {
					instance := cluster.DBInstances[i]

					endpoint, _ := convert.JsonToDictSlice(instance.Endpoint)

					mqlCluster, err := CreateResource(a.MqlRuntime, "aws.neptune.instance",
						map[string]*llx.RawData{
							"__id":                             llx.StringDataPtr(instance.DBInstanceArn),
							"arn":                              llx.StringDataPtr(instance.DBInstanceArn),
							"name":                             llx.StringDataPtr(instance.DBName),
							"clusterIdentifier":                llx.StringDataPtr(instance.DBClusterIdentifier),
							"engine":                           llx.StringDataPtr(instance.Engine),
							"engineVersion":                    llx.StringDataPtr(instance.EngineVersion),
							"kmsKeyId":                         llx.StringDataPtr(instance.KmsKeyId),
							"region":                           llx.StringData(regionVal),
							"autoMinorVersionUpgrade":          llx.BoolDataPtr(instance.AutoMinorVersionUpgrade),
							"availabilityZone":                 llx.StringDataPtr(instance.AvailabilityZone),
							"backupRetentionPeriod":            llx.IntDataPtr(instance.BackupRetentionPeriod),
							"createdAt":                        llx.TimeDataPtr(instance.InstanceCreateTime),
							"instanceClass":                    llx.StringDataPtr(instance.DBInstanceClass),
							"deletionProtection":               llx.BoolDataPtr(instance.DeletionProtection),
							"monitoringInterval":               llx.IntDataPtr(instance.MonitoringInterval),
							"monitoringRoleArn":                llx.StringDataPtr(instance.MonitoringRoleArn),
							"latestRestorableTime":             llx.TimeDataPtr(instance.LatestRestorableTime),
							"enabledCloudwatchLogsExports":     llx.ArrayData(convert.SliceAnyToInterface(instance.EnabledCloudwatchLogsExports), types.String),
							"enhancedMonitoringResourceArn":    llx.StringDataPtr(instance.EnhancedMonitoringResourceArn),
							"endpoint":                         llx.DictData(endpoint),
							"iamDatabaseAuthenticationEnabled": llx.BoolDataPtr(instance.IAMDatabaseAuthenticationEnabled),
							"masterUsername":                   llx.StringDataPtr(instance.MasterUsername),
							"multiAZ":                          llx.BoolDataPtr(instance.MultiAZ),
							"port":                             llx.IntDataPtr(instance.DbInstancePort),
							"preferredBackupWindow":            llx.StringDataPtr(instance.PreferredBackupWindow),
							"preferredMaintenanceWindow":       llx.StringDataPtr(instance.PreferredMaintenanceWindow),
							"status":                           llx.StringDataPtr(instance.DBInstanceStatus),
							"storageType":                      llx.StringDataPtr(instance.StorageType),
							"storageEncrypted":                 llx.BoolDataPtr(instance.StorageEncrypted),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlCluster)
				}
				if cluster.Marker == nil {
					break
				}
				marker = cluster.Marker
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}
