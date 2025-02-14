// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/smithy-go/transport/http"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
	"go.mondoo.com/cnquery/v11/types"
)

// The cluster and instance API also return data for non-RDS engines like Neptune and DocumentDB. We have to filter
// these out since we have specific resources for them.
var nonRdsEngines = []string{"neptune", "docdb"}

func (a *mqlAwsRds) id() (string, error) {
	return "aws.rds", nil
}

// Deprecated: use instances() instead
func (a *mqlAwsRds) dbInstances() ([]interface{}, error) {
	return a.instances()
}

// instances returns all RDS instances
func (a *mqlAwsRds) instances() ([]interface{}, error) {
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
					// we cannot filter it in the api call since the api does not support it negative filters
					if slices.Contains(nonRdsEngines, *dbInstance.Engine) {
						log.Debug().Str("engine", *dbInstance.Engine).Msg("skipping non-RDS engine")
						continue
					}

					mqlDBInstance, err := newMqlAwsRdsInstance(a.MqlRuntime, regionVal, conn.AccountId(), dbInstance)
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

// pendingMaintenanceActions returns all pending maintaince actions for all RDS instances
func (a *mqlAwsRds) allPendingMaintenanceActions() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getPendingMaintenanceActions(conn), 5)
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

func (a *mqlAwsRds) getPendingMaintenanceActions(conn *connection.AwsConnection) []*jobpool.Job {
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
				pendingMaintainanceList, err := svc.DescribePendingMaintenanceActions(ctx, &rds.DescribePendingMaintenanceActionsInput{
					Marker: marker,
				})
				if err != nil {
					return nil, err
				}
				for _, resp := range pendingMaintainanceList.PendingMaintenanceActions {
					if resp.ResourceIdentifier == nil {
						continue
					}
					for _, action := range resp.PendingMaintenanceActionDetails {
						resourceArn := *resp.ResourceIdentifier
						mqlPendingAction, err := newMqlAwsPendingMaintenanceAction(a.MqlRuntime, region, resourceArn, action)
						if err != nil {
							return nil, err
						}
						res = append(res, mqlPendingAction)
					}
				}
				if pendingMaintainanceList.Marker == nil {
					break
				}
				marker = pendingMaintainanceList.Marker
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsRdsDbinstance) id() (string, error) {
	return a.Arn.Data, nil
}

type mqlAwsRdsDbinstanceInternal struct {
	securityGroupIdHandler
	cacheSubnets *rds_types.DBSubnetGroup
	region       string
}

func newMqlAwsRdsInstance(runtime *plugin.Runtime, region string, accountID string, dbInstance rds_types.DBInstance) (*mqlAwsRdsDbinstance, error) {
	stringSliceInterface := []interface{}{}
	for _, logExport := range dbInstance.EnabledCloudwatchLogsExports {
		stringSliceInterface = append(stringSliceInterface, logExport)
	}
	sgsArn := []string{}
	for i := range dbInstance.VpcSecurityGroups {
		sgsArn = append(sgsArn, NewSecurityGroupArn(region, accountID, convert.ToString(dbInstance.VpcSecurityGroups[i].VpcSecurityGroupId)))
	}
	var endpointAddress *string
	if dbInstance.Endpoint != nil {
		endpointAddress = dbInstance.Endpoint.Address
	}

	var certificateExpiration *time.Time
	if dbInstance.CertificateDetails != nil {
		certificateExpiration = dbInstance.CertificateDetails.ValidTill
	}

	resource, err := CreateResource(runtime, "aws.rds.dbinstance",
		map[string]*llx.RawData{
			"arn":                           llx.StringDataPtr(dbInstance.DBInstanceArn),
			"autoMinorVersionUpgrade":       llx.BoolDataPtr(dbInstance.AutoMinorVersionUpgrade),
			"availabilityZone":              llx.StringDataPtr(dbInstance.AvailabilityZone),
			"backupRetentionPeriod":         llx.IntDataDefault(dbInstance.BackupRetentionPeriod, 0),
			"createdTime":                   llx.TimeDataPtr(dbInstance.InstanceCreateTime),
			"createdAt":                     llx.TimeDataPtr(dbInstance.InstanceCreateTime),
			"dbInstanceClass":               llx.StringDataPtr(dbInstance.DBInstanceClass),
			"dbInstanceIdentifier":          llx.StringDataPtr(dbInstance.DBInstanceIdentifier),
			"deletionProtection":            llx.BoolDataPtr(dbInstance.DeletionProtection),
			"enabledCloudwatchLogsExports":  llx.ArrayData(stringSliceInterface, types.String),
			"endpoint":                      llx.StringDataPtr(endpointAddress),
			"engine":                        llx.StringDataPtr(dbInstance.Engine),
			"engineLifecycleSupport":        llx.StringDataPtr(dbInstance.EngineLifecycleSupport),
			"engineVersion":                 llx.StringDataPtr(dbInstance.EngineVersion),
			"monitoringInterval":            llx.IntDataPtr(dbInstance.MonitoringInterval),
			"enhancedMonitoringResourceArn": llx.StringDataPtr(dbInstance.EnhancedMonitoringResourceArn),
			"id":                            llx.StringDataPtr(dbInstance.DBInstanceIdentifier),
			"latestRestorableTime":          llx.TimeDataPtr(dbInstance.LatestRestorableTime),
			"masterUsername":                llx.StringDataPtr(dbInstance.MasterUsername),
			"multiAZ":                       llx.BoolDataPtr(dbInstance.MultiAZ),
			"name":                          llx.StringDataPtr(dbInstance.DBName),
			"port":                          llx.IntDataDefault(dbInstance.DbInstancePort, 0),
			"publiclyAccessible":            llx.BoolDataPtr(dbInstance.PubliclyAccessible),
			"region":                        llx.StringData(region),
			"status":                        llx.StringDataPtr(dbInstance.DBInstanceStatus),
			"storageAllocated":              llx.IntDataDefault(dbInstance.AllocatedStorage, 0),
			"storageEncrypted":              llx.BoolDataPtr(dbInstance.StorageEncrypted),
			"storageIops":                   llx.IntDataDefault(dbInstance.Iops, 0),
			"storageType":                   llx.StringDataPtr(dbInstance.StorageType),
			"tags":                          llx.MapData(rdsTagsToMap(dbInstance.TagList), types.String),
			"certificateExpiresAt":          llx.TimeDataPtr(certificateExpiration),
			"certificateAuthority":          llx.StringDataPtr(dbInstance.CACertificateIdentifier),
			"iamDatabaseAuthentication":     llx.BoolDataPtr(dbInstance.IAMDatabaseAuthenticationEnabled),
			"customIamInstanceProfile":      llx.StringDataPtr(dbInstance.CustomIamInstanceProfile),
			"activityStreamMode":            llx.StringData(string(dbInstance.ActivityStreamMode)),
			"activityStreamStatus":          llx.StringData(string(dbInstance.ActivityStreamStatus)),
			"networkType":                   llx.StringDataPtr(dbInstance.NetworkType),
			"preferredMaintenanceWindow":    llx.StringDataPtr(dbInstance.PreferredMaintenanceWindow),
			"preferredBackupWindow":         llx.StringDataPtr(dbInstance.PreferredBackupWindow),
		})
	if err != nil {
		return nil, err
	}
	mqlDBInstance := resource.(*mqlAwsRdsDbinstance)
	mqlDBInstance.region = region
	mqlDBInstance.cacheSubnets = dbInstance.DBSubnetGroup
	mqlDBInstance.setSecurityGroupArns(sgsArn)
	return mqlDBInstance, nil
}

func initAwsRdsDbcluster(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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
		return nil, nil, errors.New("arn required to fetch rds db cluster")
	}

	// load all rds db clusters
	obj, err := CreateResource(runtime, "aws.rds", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	rds := obj.(*mqlAwsRds)

	rawResources := rds.GetDbClusters()
	if err != nil {
		return nil, nil, err
	}

	arnVal := args["arn"].Value.(string)
	for i := range rawResources.Data {
		dbInstance := rawResources.Data[i].(*mqlAwsRdsDbcluster)
		if dbInstance.Arn.Data == arnVal {
			return args, dbInstance, nil
		}
	}
	return nil, nil, errors.New("rds db cluster does not exist")
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

func (a *mqlAwsRdsDbinstance) subnets() ([]interface{}, error) {
	if a.cacheSubnets != nil {
		res := []interface{}{}
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		for i := range a.cacheSubnets.Subnets {
			subnet := a.cacheSubnets.Subnets[i]
			sub, err := NewResource(a.MqlRuntime, "aws.vpc.subnet", map[string]*llx.RawData{"arn": llx.StringData(fmt.Sprintf(subnetArnPattern, a.region, conn.AccountId(), convert.ToString(subnet.SubnetIdentifier)))})
			if err != nil {
				a.Subnets.State = plugin.StateIsNull | plugin.StateIsSet
				return nil, err
			}
			res = append(res, sub)
		}
		return res, nil
	}
	return nil, errors.New("no subnets found for RDS DB instance")
}

func (a *mqlAwsRdsDbinstance) securityGroups() ([]interface{}, error) {
	return a.newSecurityGroupResources(a.MqlRuntime)
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
			mqlDbSnapshot, err := newMqlAwsRdsDbSnapshot(a.MqlRuntime, region, snapshot)
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

// pendingMaintenanceActions returns all pending maintenance actions for the RDS instance
func (a *mqlAwsRdsDbinstance) pendingMaintenanceActions() ([]interface{}, error) {
	instanceArn := a.Arn.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Rds(region)
	ctx := context.Background()
	res := []interface{}{}

	var marker *string
	for {
		pendingMaintainanceList, err := svc.DescribePendingMaintenanceActions(ctx, &rds.DescribePendingMaintenanceActionsInput{
			ResourceIdentifier: &instanceArn,
			Marker:             marker,
		})
		if err != nil {
			return nil, err
		}
		for _, resp := range pendingMaintainanceList.PendingMaintenanceActions {
			if resp.ResourceIdentifier == nil {
				continue
			}
			for _, action := range resp.PendingMaintenanceActionDetails {
				resourceArn := *resp.ResourceIdentifier
				mqlDbSnapshot, err := newMqlAwsPendingMaintenanceAction(a.MqlRuntime, region, resourceArn, action)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlDbSnapshot)
			}
		}
		if pendingMaintainanceList.Marker == nil {
			break
		}
		marker = pendingMaintainanceList.Marker
	}
	return res, nil
}

// newMqlAwsPendingMaintenanceAction creates a new mqlAwsRdsPendingMaintenanceActions from a rds_types.PendingMaintenanceAction
func newMqlAwsPendingMaintenanceAction(runtime *plugin.Runtime, region string, resourceArn string, maintenanceAction rds_types.PendingMaintenanceAction) (*mqlAwsRdsPendingMaintenanceAction, error) {
	action := ""
	if maintenanceAction.Action != nil {
		action = *maintenanceAction.Action
	}

	res, err := CreateResource(runtime, "aws.rds.pendingMaintenanceAction",
		map[string]*llx.RawData{
			"__id":                 llx.StringData(fmt.Sprintf("%s/pendingMaintainance/%s", resourceArn, action)),
			"resourceArn":          llx.StringData(resourceArn),
			"action":               llx.StringDataPtr(maintenanceAction.Action),
			"description":          llx.StringDataPtr(maintenanceAction.Description),
			"autoAppliedAfterDate": llx.TimeDataPtr(maintenanceAction.AutoAppliedAfterDate),
			"currentApplyDate":     llx.TimeDataPtr(maintenanceAction.CurrentApplyDate),
			"forcedApplyDate":      llx.TimeDataPtr(maintenanceAction.ForcedApplyDate),
			"optInStatus":          llx.StringDataPtr(maintenanceAction.OptInStatus),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsRdsPendingMaintenanceAction), nil
}

func rdsTagsToMap(tags []rds_types.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[convert.ToString(tag.Key)] = convert.ToString(tag.Value)
		}
	}

	return tagsMap
}

// Deprecated: use clusters() instead
func (a *mqlAwsRds) dbClusters() ([]interface{}, error) {
	return a.clusters()
}

// clusters returns all RDS clusters
func (a *mqlAwsRds) clusters() ([]interface{}, error) {
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
					// we cannot filter it in the api call since the api does not support it negative filters
					if slices.Contains(nonRdsEngines, *cluster.Engine) {
						log.Debug().Str("engine", *cluster.Engine).Msg("skipping non-RDS engine")
						continue
					}

					mqlDbCluster, err := newMqlAwsRdsCluster(a.MqlRuntime, regionVal, conn.AccountId(), cluster)
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

type mqlAwsRdsDbclusterInternal struct {
	securityGroupIdHandler
}

func (a *mqlAwsRdsDbcluster) id() (string, error) {
	return a.Arn.Data, nil
}

func newMqlAwsRdsCluster(runtime *plugin.Runtime, region string, accountID string, cluster rds_types.DBCluster) (*mqlAwsRdsDbcluster, error) {
	mqlRdsDbInstances := []interface{}{}
	for _, instance := range cluster.DBClusterMembers {
		mqlInstance, err := NewResource(runtime, "aws.rds.dbinstance",
			map[string]*llx.RawData{
				"arn": llx.StringData(fmt.Sprintf(rdsInstanceArnPattern, region, accountID, convert.ToString(instance.DBInstanceIdentifier))),
			})
		if err != nil {
			return nil, err
		}
		mqlRdsDbInstances = append(mqlRdsDbInstances, mqlInstance)
	}
	sgsArns := []string{}
	for i := range cluster.VpcSecurityGroups {
		sgsArns = append(sgsArns, NewSecurityGroupArn(region, accountID, convert.ToString(cluster.VpcSecurityGroups[i].VpcSecurityGroupId)))
	}
	stringSliceAZs := []interface{}{}
	for _, zone := range cluster.AvailabilityZones {
		stringSliceAZs = append(stringSliceAZs, zone)
	}

	var certificateExpiration *time.Time
	var caIdentifier *string
	if cluster.CertificateDetails != nil {
		certificateExpiration = cluster.CertificateDetails.ValidTill
		caIdentifier = cluster.CertificateDetails.CAIdentifier
	}

	resource, err := CreateResource(runtime, "aws.rds.dbcluster",
		map[string]*llx.RawData{
			"activityStreamMode":         llx.StringData(string(cluster.ActivityStreamMode)),
			"activityStreamStatus":       llx.StringData(string(cluster.ActivityStreamStatus)),
			"arn":                        llx.StringDataPtr(cluster.DBClusterArn),
			"autoMinorVersionUpgrade":    llx.BoolDataPtr(cluster.AutoMinorVersionUpgrade),
			"availabilityZones":          llx.ArrayData(stringSliceAZs, types.String),
			"backupRetentionPeriod":      llx.IntDataDefault(cluster.BackupRetentionPeriod, 0),
			"certificateAuthority":       llx.StringDataPtr(caIdentifier),
			"certificateExpiresAt":       llx.TimeDataPtr(certificateExpiration),
			"clusterDbInstanceClass":     llx.StringDataPtr(cluster.DBClusterInstanceClass),
			"createdAt":                  llx.TimeDataPtr(cluster.ClusterCreateTime),
			"createdTime":                llx.TimeDataPtr(cluster.ClusterCreateTime),
			"deletionProtection":         llx.BoolDataPtr(cluster.DeletionProtection),
			"endpoint":                   llx.StringDataPtr(cluster.Endpoint),
			"engine":                     llx.StringDataPtr(cluster.Engine),
			"engineLifecycleSupport":     llx.StringDataPtr(cluster.EngineLifecycleSupport),
			"engineVersion":              llx.StringDataPtr(cluster.EngineVersion),
			"hostedZoneId":               llx.StringDataPtr(cluster.HostedZoneId),
			"httpEndpointEnabled":        llx.BoolDataPtr(cluster.HttpEndpointEnabled),
			"iamDatabaseAuthentication":  llx.BoolDataPtr(cluster.IAMDatabaseAuthenticationEnabled),
			"id":                         llx.StringDataPtr(cluster.DBClusterIdentifier),
			"latestRestorableTime":       llx.TimeDataPtr(cluster.LatestRestorableTime),
			"masterUsername":             llx.StringDataPtr(cluster.MasterUsername),
			"members":                    llx.ArrayData(mqlRdsDbInstances, types.Resource("aws.rds.dbinstance")),
			"monitoringInterval":         llx.IntDataPtr(cluster.MonitoringInterval),
			"multiAZ":                    llx.BoolDataPtr(cluster.MultiAZ),
			"networkType":                llx.StringDataPtr(cluster.NetworkType),
			"port":                       llx.IntDataDefault(cluster.Port, -1),
			"preferredBackupWindow":      llx.StringDataPtr(cluster.PreferredBackupWindow),
			"preferredMaintenanceWindow": llx.StringDataPtr(cluster.PreferredMaintenanceWindow),
			"publiclyAccessible":         llx.BoolDataPtr(cluster.PubliclyAccessible),
			"region":                     llx.StringData(region),
			"status":                     llx.StringDataPtr(cluster.Status),
			"storageAllocated":           llx.IntDataDefault(cluster.AllocatedStorage, 0),
			"storageEncrypted":           llx.BoolDataPtr(cluster.StorageEncrypted),
			"storageIops":                llx.IntDataDefault(cluster.Iops, 0),
			"storageType":                llx.StringDataPtr(cluster.StorageType),
			"tags":                       llx.MapData(rdsTagsToMap(cluster.TagList), types.String),
		})
	if err != nil {
		return nil, err
	}
	mqlDbCluster := resource.(*mqlAwsRdsDbcluster)
	mqlDbCluster.setSecurityGroupArns(sgsArns)
	return mqlDbCluster, nil
}

func (a *mqlAwsRdsDbcluster) securityGroups() ([]interface{}, error) {
	return a.newSecurityGroupResources(a.MqlRuntime)
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
			mqlDbSnapshot, err := newMqlAwsRdsClusterSnapshot(a.MqlRuntime, region, snapshot)
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

// newMqlAwsRdsClusterSnapshot creates a new mqlAwsRdsSnapshot from a rds_types.DBClusterSnapshot which is only
// used for Aurora clusters not for RDS instances
func newMqlAwsRdsClusterSnapshot(runtime *plugin.Runtime, region string, snapshot rds_types.DBClusterSnapshot) (*mqlAwsRdsSnapshot, error) {
	res, err := CreateResource(runtime, "aws.rds.snapshot",
		map[string]*llx.RawData{
			"allocatedStorage":  llx.IntDataDefault(snapshot.AllocatedStorage, 0),
			"arn":               llx.StringDataPtr(snapshot.DBClusterSnapshotArn),
			"createdAt":         llx.TimeDataPtr(snapshot.SnapshotCreateTime),
			"encrypted":         llx.BoolDataPtr(snapshot.StorageEncrypted),
			"engine":            llx.StringDataPtr(snapshot.Engine),
			"engineVersion":     llx.StringDataPtr(snapshot.EngineVersion),
			"id":                llx.StringDataPtr(snapshot.DBClusterSnapshotIdentifier),
			"port":              llx.IntDataDefault(snapshot.Port, -1),
			"isClusterSnapshot": llx.BoolData(true),
			"region":            llx.StringData(region),
			"status":            llx.StringDataPtr(snapshot.Status),
			"tags":              llx.MapData(rdsTagsToMap(snapshot.TagList), types.String),
			"type":              llx.StringDataPtr(snapshot.SnapshotType),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsRdsSnapshot), nil
}

// newMqlAwsRdsDbSnapshot creates a new mqlAwsRdsSnapshot from a rds_types.DBSnapshot which is only
// used for Aurora clusters not for RDS instances
func newMqlAwsRdsDbSnapshot(runtime *plugin.Runtime, region string, snapshot rds_types.DBSnapshot) (*mqlAwsRdsSnapshot, error) {
	res, err := CreateResource(runtime, "aws.rds.snapshot",
		map[string]*llx.RawData{
			"allocatedStorage":  llx.IntDataDefault(snapshot.AllocatedStorage, 0),
			"arn":               llx.StringDataPtr(snapshot.DBSnapshotArn),
			"createdAt":         llx.TimeDataPtr(snapshot.SnapshotCreateTime),
			"encrypted":         llx.BoolDataPtr(snapshot.Encrypted),
			"engine":            llx.StringDataPtr(snapshot.Engine),
			"engineVersion":     llx.StringDataPtr(snapshot.EngineVersion),
			"id":                llx.StringDataPtr(snapshot.DBSnapshotIdentifier),
			"port":              llx.IntDataDefault(snapshot.Port, -1),
			"isClusterSnapshot": llx.BoolData(false),
			"region":            llx.StringData(region),
			"status":            llx.StringDataPtr(snapshot.Status),
			"tags":              llx.MapData(rdsTagsToMap(snapshot.TagList), types.String),
			"type":              llx.StringDataPtr(snapshot.SnapshotType),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsRdsSnapshot), nil
}

func (a *mqlAwsRdsSnapshot) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsRdsBackupsetting) id() (string, error) {
	return a.Target.Data, nil
}

type mqlAwsRdsBackupsettingInternal struct {
	kmsKeyId *string
}

func (a *mqlAwsRdsBackupsetting) kmsKey() (*mqlAwsKmsKey, error) {
	if a.kmsKeyId == nil {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlKey, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{
			"arn": llx.StringDataPtr(a.kmsKeyId),
		})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsRdsDbinstance) backupSettings() ([]interface{}, error) {
	instanceId := a.Id.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Rds(region)
	ctx := context.Background()
	res := []interface{}{}

	var marker *string
	for {
		resp, err := svc.DescribeDBInstanceAutomatedBackups(ctx, &rds.DescribeDBInstanceAutomatedBackupsInput{DBInstanceIdentifier: &instanceId, Marker: marker})
		var respErr *http.ResponseError
		if err != nil {
			if errors.As(err, &respErr) {
				if respErr.HTTPStatusCode() == 404 {
					return nil, nil
				}
			}
			return nil, err
		}
		for _, backup := range resp.DBInstanceAutomatedBackups {
			var earliest *time.Time
			var latest *time.Time
			if backup.RestoreWindow != nil {
				earliest = backup.RestoreWindow.EarliestTime
				latest = backup.RestoreWindow.LatestTime
			}
			mqlRdsBackup, err := CreateResource(a.MqlRuntime, "aws.rds.backupsetting",
				map[string]*llx.RawData{
					"target":                   llx.StringDataPtr(backup.BackupTarget),
					"retentionPeriod":          llx.IntDataPtr(backup.BackupRetentionPeriod),
					"dedicatedLogVolume":       llx.BoolDataPtr(backup.DedicatedLogVolume),
					"encrypted":                llx.BoolDataPtr(backup.Encrypted),
					"region":                   llx.StringData(region),
					"status":                   llx.StringDataPtr(backup.Status),
					"timezone":                 llx.StringDataPtr(backup.Timezone),
					"earliestRestoreAvailable": llx.TimeDataPtr(earliest),
					"latestRestoreAvailable":   llx.TimeDataPtr(latest),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRdsBackup)
			mqlRdsBackup.(*mqlAwsRdsBackupsetting).kmsKeyId = backup.KmsKeyId
		}
		if resp.Marker == nil {
			break
		}
		marker = resp.Marker
	}
	return res, nil
}

func (a *mqlAwsRdsDbcluster) backupSettings() ([]interface{}, error) {
	clusterId := a.Id.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Rds(region)
	ctx := context.Background()
	res := []interface{}{}

	var marker *string
	for {
		resp, err := svc.DescribeDBClusterAutomatedBackups(ctx, &rds.DescribeDBClusterAutomatedBackupsInput{DBClusterIdentifier: &clusterId, Marker: marker})
		var respErr *http.ResponseError
		if err != nil {
			if errors.As(err, &respErr) {
				if respErr.HTTPStatusCode() == 404 {
					return nil, nil
				}
			}
			return nil, err
		}
		for _, backup := range resp.DBClusterAutomatedBackups {
			var earliest *time.Time
			var latest *time.Time
			if backup.RestoreWindow != nil {
				earliest = backup.RestoreWindow.EarliestTime
				latest = backup.RestoreWindow.LatestTime
			}
			mqlRdsBackup, err := CreateResource(a.MqlRuntime, "aws.rds.backupsetting",
				map[string]*llx.RawData{
					"target":                   llx.StringDataPtr(backup.DBClusterIdentifier),
					"retentionPeriod":          llx.IntDataPtr(backup.BackupRetentionPeriod),
					"dedicatedLogVolume":       llx.NilData,
					"encrypted":                llx.BoolDataPtr(backup.StorageEncrypted),
					"region":                   llx.StringData(region),
					"status":                   llx.StringDataPtr(backup.Status),
					"timezone":                 llx.NilData,
					"earliestRestoreAvailable": llx.TimeDataPtr(earliest),
					"latestRestoreAvailable":   llx.TimeDataPtr(latest),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRdsBackup)
			mqlRdsBackup.(*mqlAwsRdsBackupsetting).kmsKeyId = backup.KmsKeyId
		}
		if resp.Marker == nil {
			break
		}
		marker = resp.Marker
	}
	return res, nil
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
