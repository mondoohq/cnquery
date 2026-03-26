// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/docdb"
	docdb_types "github.com/aws/aws-sdk-go-v2/service/docdb/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsDocumentdb) id() (string, error) {
	return "aws.documentdb", nil
}

func (a *mqlAwsDocumentdb) clusters() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getDbClusters(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}

	return res, nil
}

func (a *mqlAwsDocumentdb) getDbClusters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("documentdb>getDbClusters>calling aws with region %s", region)

			svc := conn.DocumentDB(region)
			ctx := context.Background()
			res := []any{}

			paginator := docdb.NewDescribeDBClustersPaginator(svc, &docdb.DescribeDBClustersInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, cluster := range page.DBClusters {
					mqlCluster, err := newMqlAwsDocumentdbCluster(a.MqlRuntime, region, conn.AccountId(), cluster)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlCluster)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlAwsDocumentdbCluster(runtime *plugin.Runtime, region string, accountID string, cluster docdb_types.DBCluster) (*mqlAwsDocumentdbCluster, error) {
	resource, err := CreateResource(runtime, "aws.documentdb.cluster",
		map[string]*llx.RawData{
			"__id":                         llx.StringDataPtr(cluster.DBClusterArn),
			"arn":                          llx.StringDataPtr(cluster.DBClusterArn),
			"name":                         llx.StringDataPtr(cluster.DBClusterIdentifier),
			"clusterIdentifier":            llx.StringDataPtr(cluster.DBClusterIdentifier),
			"engine":                       llx.StringDataPtr(cluster.Engine),
			"engineVersion":                llx.StringDataPtr(cluster.EngineVersion),
			"region":                       llx.StringData(region),
			"availabilityZones":            llx.ArrayData(convert.SliceAnyToInterface(cluster.AvailabilityZones), types.String),
			"backupRetentionPeriod":        llx.IntDataPtr(cluster.BackupRetentionPeriod),
			"createdAt":                    llx.TimeDataPtr(cluster.ClusterCreateTime),
			"clusterParameterGroup":        llx.StringDataPtr(cluster.DBClusterParameterGroup),
			"subnetGroup":                  llx.StringDataPtr(cluster.DBSubnetGroup),
			"clusterResourceId":            llx.StringDataPtr(cluster.DbClusterResourceId),
			"deletionProtection":           llx.BoolDataPtr(cluster.DeletionProtection),
			"earliestRestorableTime":       llx.TimeDataPtr(cluster.EarliestRestorableTime),
			"enabledCloudwatchLogsExports": llx.ArrayData(convert.SliceAnyToInterface(cluster.EnabledCloudwatchLogsExports), types.String),
			"endpoint":                     llx.StringDataPtr(cluster.Endpoint),
			"masterUsername":               llx.StringDataPtr(cluster.MasterUsername),
			"multiAZ":                      llx.BoolDataPtr(cluster.MultiAZ),
			"port":                         llx.IntDataPtr(cluster.Port),
			"preferredBackupWindow":        llx.StringDataPtr(cluster.PreferredBackupWindow),
			"preferredMaintenanceWindow":   llx.StringDataPtr(cluster.PreferredMaintenanceWindow),
			"status":                       llx.StringDataPtr(cluster.Status),
			"storageEncrypted":             llx.BoolDataPtr(cluster.StorageEncrypted),
			"storageType":                  llx.StringDataPtr(cluster.StorageType),
			"networkType":                  llx.StringDataPtr(cluster.NetworkType),
		})
	if err != nil {
		return nil, err
	}
	sgsArn := []string{}
	for _, sg := range cluster.VpcSecurityGroups {
		if sg.VpcSecurityGroupId != nil {
			sgsArn = append(sgsArn, NewSecurityGroupArn(region, accountID, *sg.VpcSecurityGroupId))
		}
	}
	mqlCluster := resource.(*mqlAwsDocumentdbCluster)
	mqlCluster.cacheKmsKeyId = cluster.KmsKeyId
	mqlCluster.setSecurityGroupArns(sgsArn)
	return mqlCluster, nil
}

func (a *mqlAwsDocumentdbCluster) securityGroups() ([]any, error) {
	return a.newSecurityGroupResources(a.MqlRuntime)
}

type mqlAwsDocumentdbClusterInternal struct {
	securityGroupIdHandler
	cacheKmsKeyId *string
}

func (a *mqlAwsDocumentdbCluster) kmsKey() (*mqlAwsKmsKey, error) {
	if a.cacheKmsKeyId == nil || *a.cacheKmsKeyId == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlKey, err := NewResource(a.MqlRuntime, ResourceAwsKmsKey,
		map[string]*llx.RawData{
			"arn": llx.StringDataPtr(a.cacheKmsKeyId),
		})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsDocumentdbCluster) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.DocumentDB(a.Region.Data)
	ctx := context.Background()
	arn := a.Arn.Data

	resp, err := svc.ListTagsForResource(ctx, &docdb.ListTagsForResourceInput{
		ResourceName: &arn,
	})
	if err != nil {
		return nil, err
	}
	tags := make(map[string]any)
	for _, t := range resp.TagList {
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
		}
	}
	return tags, nil
}

func (a *mqlAwsDocumentdb) instances() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getDbInstances(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}

	return res, nil
}

func (a *mqlAwsDocumentdb) getDbInstances(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("documentdb>getDbInstances>calling aws with region %s", region)

			svc := conn.DocumentDB(region)
			ctx := context.Background()
			res := []any{}

			paginator := docdb.NewDescribeDBInstancesPaginator(svc, &docdb.DescribeDBInstancesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, instance := range page.DBInstances {
					mqlInstance, err := newMqlAwsDocumentdbInstance(a.MqlRuntime, region, instance)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlInstance)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlAwsDocumentdbInstance(runtime *plugin.Runtime, region string, instance docdb_types.DBInstance) (*mqlAwsDocumentdbInstance, error) {
	endpoint, _ := convert.JsonToDict(instance.Endpoint)

	resource, err := CreateResource(runtime, "aws.documentdb.instance",
		map[string]*llx.RawData{
			"__id":                         llx.StringDataPtr(instance.DBInstanceArn),
			"arn":                          llx.StringDataPtr(instance.DBInstanceArn),
			"name":                         llx.StringDataPtr(instance.DBInstanceIdentifier),
			"clusterIdentifier":            llx.StringDataPtr(instance.DBClusterIdentifier),
			"engine":                       llx.StringDataPtr(instance.Engine),
			"engineVersion":                llx.StringDataPtr(instance.EngineVersion),
			"createdAt":                    llx.TimeDataPtr(instance.InstanceCreateTime),
			"region":                       llx.StringData(region),
			"autoMinorVersionUpgrade":      llx.BoolDataPtr(instance.AutoMinorVersionUpgrade),
			"availabilityZone":             llx.StringDataPtr(instance.AvailabilityZone),
			"backupRetentionPeriod":        llx.IntDataPtr(instance.BackupRetentionPeriod),
			"instanceClass":                llx.StringDataPtr(instance.DBInstanceClass),
			"enabledCloudwatchLogsExports": llx.ArrayData(convert.SliceAnyToInterface(instance.EnabledCloudwatchLogsExports), types.String),
			"endpoint":                     llx.MapData(endpoint, types.Any),
			"preferredBackupWindow":        llx.StringDataPtr(instance.PreferredBackupWindow),
			"preferredMaintenanceWindow":   llx.StringDataPtr(instance.PreferredMaintenanceWindow),
			"promotionTier":                llx.IntDataPtr(instance.PromotionTier),
			"status":                       llx.StringDataPtr(instance.DBInstanceStatus),
			"storageEncrypted":             llx.BoolDataPtr(instance.StorageEncrypted),
			"certificateAuthority":         llx.StringDataPtr(instance.CACertificateIdentifier),
		})
	if err != nil {
		return nil, err
	}
	mqlInstance := resource.(*mqlAwsDocumentdbInstance)
	mqlInstance.cacheKmsKeyId = instance.KmsKeyId
	return mqlInstance, nil
}

type mqlAwsDocumentdbInstanceInternal struct {
	cacheKmsKeyId *string
}

func (a *mqlAwsDocumentdbInstance) kmsKey() (*mqlAwsKmsKey, error) {
	if a.cacheKmsKeyId == nil || *a.cacheKmsKeyId == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlKey, err := NewResource(a.MqlRuntime, ResourceAwsKmsKey,
		map[string]*llx.RawData{
			"arn": llx.StringDataPtr(a.cacheKmsKeyId),
		})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsDocumentdbSnapshot) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsDocumentdb) snapshots() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getSnapshots(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
		}
	}
	return res, nil
}

func (a *mqlAwsDocumentdb) getSnapshots(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("documentdb>getSnapshots>calling aws with region %s", region)

			svc := conn.DocumentDB(region)
			ctx := context.Background()
			res := []any{}

			paginator := docdb.NewDescribeDBClusterSnapshotsPaginator(svc, &docdb.DescribeDBClusterSnapshotsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, snapshot := range page.DBClusterSnapshots {
					mqlSnapshot, err := newMqlAwsDocumentdbSnapshot(a.MqlRuntime, region, snapshot)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlSnapshot)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsDocumentdbCluster) snapshots() ([]any, error) {
	clusterIdentifier := a.ClusterIdentifier.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.DocumentDB(region)
	ctx := context.Background()
	res := []any{}

	paginator := docdb.NewDescribeDBClusterSnapshotsPaginator(svc, &docdb.DescribeDBClusterSnapshotsInput{
		DBClusterIdentifier: &clusterIdentifier,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, snapshot := range page.DBClusterSnapshots {
			mqlSnapshot, err := newMqlAwsDocumentdbSnapshot(a.MqlRuntime, region, snapshot)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlSnapshot)
		}
	}
	return res, nil
}

func newMqlAwsDocumentdbSnapshot(runtime *plugin.Runtime, region string, snapshot docdb_types.DBClusterSnapshot) (*mqlAwsDocumentdbSnapshot, error) {
	resource, err := CreateResource(runtime, "aws.documentdb.snapshot",
		map[string]*llx.RawData{
			"__id":              llx.StringDataPtr(snapshot.DBClusterSnapshotArn),
			"arn":               llx.StringDataPtr(snapshot.DBClusterSnapshotArn),
			"id":                llx.StringDataPtr(snapshot.DBClusterSnapshotIdentifier),
			"clusterIdentifier": llx.StringDataPtr(snapshot.DBClusterIdentifier),
			"engine":            llx.StringDataPtr(snapshot.Engine),
			"engineVersion":     llx.StringDataPtr(snapshot.EngineVersion),
			"status":            llx.StringDataPtr(snapshot.Status),
			"snapshotType":      llx.StringDataPtr(snapshot.SnapshotType),
			"port":              llx.IntDataDefault(snapshot.Port, 0),
			"storageEncrypted":  llx.BoolDataPtr(snapshot.StorageEncrypted),
			"storageType":       llx.StringDataPtr(snapshot.StorageType),
			"availabilityZones": llx.ArrayData(convert.SliceAnyToInterface(snapshot.AvailabilityZones), types.String),
			"percentProgress":   llx.IntDataDefault(snapshot.PercentProgress, 0),
			"createdAt":         llx.TimeDataPtr(snapshot.SnapshotCreateTime),
			"clusterCreatedAt":  llx.TimeDataPtr(snapshot.ClusterCreateTime),
			"region":            llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	mqlSnapshot := resource.(*mqlAwsDocumentdbSnapshot)
	mqlSnapshot.cacheKmsKeyId = snapshot.KmsKeyId
	mqlSnapshot.cacheVpcId = snapshot.VpcId
	return mqlSnapshot, nil
}

type mqlAwsDocumentdbSnapshotInternal struct {
	cacheKmsKeyId *string
	cacheVpcId    *string
}

func (a *mqlAwsDocumentdbSnapshot) kmsKey() (*mqlAwsKmsKey, error) {
	if a.cacheKmsKeyId == nil || *a.cacheKmsKeyId == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlKey, err := NewResource(a.MqlRuntime, ResourceAwsKmsKey,
		map[string]*llx.RawData{
			"arn": llx.StringDataPtr(a.cacheKmsKeyId),
		})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsDocumentdbSnapshot) vpc() (*mqlAwsVpc, error) {
	if a.cacheVpcId == nil || *a.cacheVpcId == "" {
		a.Vpc.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	mqlVpc, err := NewResource(a.MqlRuntime, "aws.vpc",
		map[string]*llx.RawData{
			"arn": llx.StringData(fmt.Sprintf(vpcArnPattern, a.Region.Data, conn.AccountId(), *a.cacheVpcId)),
		})
	if err != nil {
		return nil, err
	}
	return mqlVpc.(*mqlAwsVpc), nil
}

func (a *mqlAwsDocumentdbInstance) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.DocumentDB(a.Region.Data)
	ctx := context.Background()
	arn := a.Arn.Data

	resp, err := svc.ListTagsForResource(ctx, &docdb.ListTagsForResourceInput{
		ResourceName: &arn,
	})
	if err != nil {
		return nil, err
	}
	tags := make(map[string]any)
	for _, t := range resp.TagList {
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
		}
	}
	return tags, nil
}
