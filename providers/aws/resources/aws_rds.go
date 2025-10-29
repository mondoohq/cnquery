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
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v12/providers/aws/connection"
	"go.mondoo.com/cnquery/v12/types"
)

// The cluster and instance API also return data for non-RDS engines like Neptune and DocumentDB. We have to filter
// these out since we have specific resources for them.
var nonRdsEngines = []string{"neptune", "docdb"}

func (a *mqlAwsRds) id() (string, error) {
	return "aws.rds", nil
}

// instances returns all RDS instances
func (a *mqlAwsRds) instances() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getDbInstances(conn), 5)
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

func (a *mqlAwsRds) clusterParameterGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getClusterParameterGroups(conn), 5)
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

func (a *mqlAwsRds) parameterGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getParameterGroups(conn), 5)
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

func (a *mqlAwsRds) getClusterParameterGroups(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("rds>getClusterParameterGroup>calling aws with region %s", region)
			res := []any{}
			svc := conn.Rds(region)
			ctx := context.Background()

			params := &rds.DescribeDBClusterParameterGroupsInput{}
			paginator := rds.NewDescribeDBClusterParameterGroupsPaginator(svc, params)
			for paginator.HasMorePages() {
				DBClusterParameterGroups, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, dbClusterParameterGroup := range DBClusterParameterGroups.DBClusterParameterGroups {
					mqlParameterGroup, err := newMqlAwsRdsClusterParameterGroup(a.MqlRuntime, region, dbClusterParameterGroup)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlParameterGroup)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlAwsRdsClusterParameterGroup(runtime *plugin.Runtime, region string, parameterGroup rds_types.DBClusterParameterGroup) (*mqlAwsRdsClusterParameterGroup, error) {
	resource, err := CreateResource(runtime, "aws.rds.clusterParameterGroup",
		map[string]*llx.RawData{
			"__id":        llx.StringData(fmt.Sprintf("%s/%s", *parameterGroup.DBClusterParameterGroupArn, *parameterGroup.DBClusterParameterGroupName)),
			"arn":         llx.StringDataPtr(parameterGroup.DBClusterParameterGroupArn),
			"family":      llx.StringDataPtr(parameterGroup.DBParameterGroupFamily),
			"name":        llx.StringDataPtr(parameterGroup.DBClusterParameterGroupName),
			"description": llx.StringDataPtr(parameterGroup.Description),
			"region":      llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	mqlParameterGroup := resource.(*mqlAwsRdsClusterParameterGroup)
	return mqlParameterGroup, nil
}

func (a *mqlAwsRds) getParameterGroups(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("rds>getParameterGroup>calling aws with region %s", region)
			res := []any{}
			svc := conn.Rds(region)
			ctx := context.Background()

			params := &rds.DescribeDBParameterGroupsInput{}
			paginator := rds.NewDescribeDBParameterGroupsPaginator(svc, params)
			for paginator.HasMorePages() {
				dbParameterGroups, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, dbParameterGroup := range dbParameterGroups.DBParameterGroups {
					mqlParameterGroup, err := newMqlAwsParameterGroup(a.MqlRuntime, region, dbParameterGroup)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlParameterGroup)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsRds) getDbInstances(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("rds>getDbInstances>calling aws with region %s", region)

			res := []any{}
			svc := conn.Rds(region)
			ctx := context.Background()

			params := &rds.DescribeDBInstancesInput{}
			paginator := rds.NewDescribeDBInstancesPaginator(svc, params)
			for paginator.HasMorePages() {
				dbInstances, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
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

					mqlDBInstance, err := newMqlAwsRdsInstance(a.MqlRuntime, region, conn.AccountId(), dbInstance)
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

// pendingMaintenanceActions returns all pending maintenance actions for all RDS instances
func (a *mqlAwsRds) allPendingMaintenanceActions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getPendingMaintenanceActions(conn), 5)
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

func (a *mqlAwsRds) getPendingMaintenanceActions(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("rds>getDbInstances>calling aws with region %s", region)

			res := []any{}
			svc := conn.Rds(region)
			ctx := context.Background()

			params := &rds.DescribePendingMaintenanceActionsInput{}
			paginator := rds.NewDescribePendingMaintenanceActionsPaginator(svc, params)
			for paginator.HasMorePages() {
				pendingMaintainanceList, err := paginator.NextPage(ctx)
				if err != nil {
					return nil, err
				}
				for _, resp := range pendingMaintainanceList.PendingMaintenanceActions {
					if resp.ResourceIdentifier == nil {
						continue
					}
					for _, action := range resp.PendingMaintenanceActionDetails {
						resourceArn := *resp.ResourceIdentifier
						mqlPendingAction, err := newMqlAwsPendingMaintenanceAction(a.MqlRuntime, resourceArn, action)
						if err != nil {
							return nil, err
						}
						res = append(res, mqlPendingAction)
					}
				}
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

func newMqlAwsParameterGroup(runtime *plugin.Runtime, region string, parameterGroup rds_types.DBParameterGroup) (*mqlAwsRdsParameterGroup, error) {
	resource, err := CreateResource(runtime, "aws.rds.parameterGroup",
		map[string]*llx.RawData{
			"__id":        llx.StringData(fmt.Sprintf("%s/%s", *parameterGroup.DBParameterGroupArn, *parameterGroup.DBParameterGroupName)),
			"arn":         llx.StringDataPtr(parameterGroup.DBParameterGroupArn),
			"family":      llx.StringDataPtr(parameterGroup.DBParameterGroupFamily),
			"name":        llx.StringDataPtr(parameterGroup.DBParameterGroupName),
			"description": llx.StringDataPtr(parameterGroup.Description),
			"region":      llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	mqlParameterGroup := resource.(*mqlAwsRdsParameterGroup)
	return mqlParameterGroup, nil
}

func (a mqlAwsRdsClusterParameterGroup) parameters() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	svc := conn.Rds(a.Region.Data)
	ctx := context.Background()

	params := &rds.DescribeDBClusterParametersInput{
		DBClusterParameterGroupName: &a.Name.Data,
	}
	paginator := rds.NewDescribeDBClusterParametersPaginator(svc, params)
	for paginator.HasMorePages() {
		parameters, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, parameter := range parameters.Parameters {
			mqlParameter, err := newMqlAwsRdsParameterGroupParameter(a.MqlRuntime, parameter)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlParameter)
		}
	}
	return res, nil
}

func (a *mqlAwsRdsParameterGroup) parameters() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	svc := conn.Rds(a.Region.Data)
	ctx := context.Background()

	params := &rds.DescribeDBParametersInput{
		DBParameterGroupName: &a.Name.Data,
	}
	paginator := rds.NewDescribeDBParametersPaginator(svc, params)
	for paginator.HasMorePages() {
		parameters, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, parameter := range parameters.Parameters {
			mqlParameter, err := newMqlAwsRdsParameterGroupParameter(a.MqlRuntime, parameter)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlParameter)
		}
	}
	return res, nil
}

func newMqlAwsRdsParameterGroupParameter(runtime *plugin.Runtime, parameter rds_types.Parameter) (*mqlAwsRdsParameterGroupParameter, error) {
	engineModes := []any{}
	for _, engineMode := range parameter.SupportedEngineModes {
		engineModes = append(engineModes, engineMode)
	}

	resource, err := CreateResource(runtime, "aws.rds.parameterGroup.parameter",
		map[string]*llx.RawData{
			"__id":                 llx.StringDataPtr(parameter.ParameterName),
			"name":                 llx.StringDataPtr(parameter.ParameterName),
			"value":                llx.StringDataPtr(parameter.ParameterValue),
			"allowedValues":        llx.StringDataPtr(parameter.AllowedValues),
			"applyType":            llx.StringDataPtr(parameter.ApplyType),
			"applyMethod":          llx.StringData(string(parameter.ApplyMethod)),
			"dataType":             llx.StringDataPtr(parameter.DataType),
			"description":          llx.StringDataPtr(parameter.Description),
			"isModifiable":         llx.BoolDataPtr(parameter.IsModifiable),
			"source":               llx.StringDataPtr(parameter.Source),
			"minimumEngineVersion": llx.StringDataPtr(parameter.MinimumEngineVersion),
			"supportedEngineModes": llx.ArrayData(engineModes, types.String),
		})
	if err != nil {
		return nil, err
	}
	mqlParameter := resource.(*mqlAwsRdsParameterGroupParameter)
	return mqlParameter, nil
}

func newMqlAwsRdsInstance(runtime *plugin.Runtime, region string, accountID string, dbInstance rds_types.DBInstance) (*mqlAwsRdsDbinstance, error) {
	stringSliceInterface := []any{}
	for _, logExport := range dbInstance.EnabledCloudwatchLogsExports {
		stringSliceInterface = append(stringSliceInterface, logExport)
	}
	sgsArn := []string{}
	for i := range dbInstance.VpcSecurityGroups {
		sgsArn = append(sgsArn, NewSecurityGroupArn(region, accountID, convert.ToValue(dbInstance.VpcSecurityGroups[i].VpcSecurityGroupId)))
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
	rawResources := rds.GetClusters()

	arnVal := args["arn"].Value.(string)
	for _, rawResource := range rawResources.Data {
		dbInstance := rawResource.(*mqlAwsRdsDbcluster)
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
	rawResources := rds.GetInstances()

	arnVal := args["arn"].Value.(string)
	for _, rawResource := range rawResources.Data {
		dbInstance := rawResource.(*mqlAwsRdsDbinstance)
		if dbInstance.Arn.Data == arnVal {
			return args, dbInstance, nil
		}
	}
	return nil, nil, errors.New("rds db instance does not exist")
}

func (a *mqlAwsRdsDbinstance) subnets() ([]any, error) {
	if a.cacheSubnets != nil {
		res := []any{}
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		for i := range a.cacheSubnets.Subnets {
			subnet := a.cacheSubnets.Subnets[i]
			sub, err := NewResource(a.MqlRuntime, "aws.vpc.subnet", map[string]*llx.RawData{"arn": llx.StringData(fmt.Sprintf(subnetArnPattern, a.region, conn.AccountId(), convert.ToValue(subnet.SubnetIdentifier)))})
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

func (a *mqlAwsRdsDbinstance) securityGroups() ([]any, error) {
	return a.newSecurityGroupResources(a.MqlRuntime)
}

func (a *mqlAwsRdsDbinstance) snapshots() ([]any, error) {
	instanceId := a.Id.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Rds(region)
	ctx := context.Background()
	res := []any{}

	params := &rds.DescribeDBSnapshotsInput{DBInstanceIdentifier: &instanceId}
	paginator := rds.NewDescribeDBSnapshotsPaginator(svc, params)
	for paginator.HasMorePages() {
		snapshots, err := paginator.NextPage(ctx)
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
	}
	return res, nil
}

// pendingMaintenanceActions returns all pending maintenance actions for the RDS instance
func (a *mqlAwsRdsDbinstance) pendingMaintenanceActions() ([]any, error) {
	instanceArn := a.Arn.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Rds(region)
	ctx := context.Background()
	res := []any{}

	params := &rds.DescribePendingMaintenanceActionsInput{
		ResourceIdentifier: &instanceArn,
	}
	paginator := rds.NewDescribePendingMaintenanceActionsPaginator(svc, params)
	for paginator.HasMorePages() {
		pendingMaintainanceList, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, resp := range pendingMaintainanceList.PendingMaintenanceActions {
			if resp.ResourceIdentifier == nil {
				continue
			}
			for _, action := range resp.PendingMaintenanceActionDetails {
				resourceArn := *resp.ResourceIdentifier
				mqlDbSnapshot, err := newMqlAwsPendingMaintenanceAction(a.MqlRuntime, resourceArn, action)
				if err != nil {
					return nil, err
				}
				res = append(res, mqlDbSnapshot)
			}
		}
	}
	return res, nil
}

// newMqlAwsPendingMaintenanceAction creates a new mqlAwsRdsPendingMaintenanceActions from a rds_types.PendingMaintenanceAction
func newMqlAwsPendingMaintenanceAction(runtime *plugin.Runtime, resourceArn string, maintenanceAction rds_types.PendingMaintenanceAction) (*mqlAwsRdsPendingMaintenanceAction, error) {
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

func rdsTagsToMap(tags []rds_types.Tag) map[string]any {
	tagsMap := make(map[string]any)

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[convert.ToValue(tag.Key)] = convert.ToValue(tag.Value)
		}
	}

	return tagsMap
}

// clusters returns all RDS clusters
func (a *mqlAwsRds) clusters() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getDbClusters(conn), 5)
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

func (a *mqlAwsRds) getDbClusters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("rds>getDbClusters>calling aws with region %s", region)

			res := []any{}
			svc := conn.Rds(region)
			ctx := context.Background()

			params := &rds.DescribeDBClustersInput{}
			paginator := rds.NewDescribeDBClustersPaginator(svc, params)
			for paginator.HasMorePages() {
				dbClusters, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
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

					mqlDbCluster, err := newMqlAwsRdsCluster(a.MqlRuntime, region, conn.AccountId(), cluster)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlDbCluster)
				}
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
	mqlRdsDbInstances := []any{}
	for _, instance := range cluster.DBClusterMembers {
		mqlInstance, err := NewResource(runtime, "aws.rds.dbinstance",
			map[string]*llx.RawData{
				"arn": llx.StringData(fmt.Sprintf(rdsInstanceArnPattern, region, accountID, convert.ToValue(instance.DBInstanceIdentifier))),
			})
		if err != nil {
			return nil, err
		}
		mqlRdsDbInstances = append(mqlRdsDbInstances, mqlInstance)
	}
	sgsArns := []string{}
	for i := range cluster.VpcSecurityGroups {
		sgsArns = append(sgsArns, NewSecurityGroupArn(region, accountID, convert.ToValue(cluster.VpcSecurityGroups[i].VpcSecurityGroupId)))
	}
	stringSliceAZs := []any{}
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
			"databaseInsightsMode":       llx.StringData(string(cluster.DatabaseInsightsMode)),
			"deletionProtection":         llx.BoolDataPtr(cluster.DeletionProtection),
			"endpoint":                   llx.StringDataPtr(cluster.Endpoint),
			"engine":                     llx.StringDataPtr(cluster.Engine),
			"engineLifecycleSupport":     llx.StringDataPtr(cluster.EngineLifecycleSupport),
			"engineVersion":              llx.StringDataPtr(cluster.EngineVersion),
			"globalClusterIdentifier":    llx.StringDataPtr(cluster.GlobalClusterIdentifier),
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
			"parameterGroupName":         llx.StringDataPtr(cluster.DBClusterParameterGroup),
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

func (a *mqlAwsRdsDbcluster) securityGroups() ([]any, error) {
	return a.newSecurityGroupResources(a.MqlRuntime)
}

func (a *mqlAwsRdsDbcluster) snapshots() ([]any, error) {
	dbClusterId := a.Id.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Rds(region)
	ctx := context.Background()
	res := []any{}

	params := &rds.DescribeDBClusterSnapshotsInput{DBClusterIdentifier: &dbClusterId}
	paginator := rds.NewDescribeDBClusterSnapshotsPaginator(svc, params)
	for paginator.HasMorePages() {
		snapshots, err := paginator.NextPage(ctx)
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

func (a *mqlAwsRdsDbinstance) backupSettings() ([]any, error) {
	instanceId := a.Id.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Rds(region)
	ctx := context.Background()
	res := []any{}
	params := &rds.DescribeDBInstanceAutomatedBackupsInput{DBInstanceIdentifier: &instanceId}
	paginator := rds.NewDescribeDBInstanceAutomatedBackupsPaginator(svc, params)
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
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
	}
	return res, nil
}

func (a *mqlAwsRdsDbcluster) backupSettings() ([]any, error) {
	clusterId := a.Id.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Rds(region)
	ctx := context.Background()
	res := []any{}

	params := &rds.DescribeDBClusterAutomatedBackupsInput{DBClusterIdentifier: &clusterId}
	paginator := rds.NewDescribeDBClusterAutomatedBackupsPaginator(svc, params)
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
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
	}
	return res, nil
}

func (a *mqlAwsRdsSnapshot) attributes() ([]any, error) {
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
