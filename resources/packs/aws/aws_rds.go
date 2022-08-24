package aws

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (d *mqlAwsRds) id() (string, error) {
	return "aws.rds", nil
}

const (
	rdsInstanceArnPattern = "arn:aws:rds:%s:%s:db:%s"
)

func (d *mqlAwsRds) GetDbInstances() ([]interface{}, error) {
	provider, err := awsProvider(d.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(d.getDbInstances(provider), 5)
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

func (d *mqlAwsRds) getDbInstances(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	account, err := provider.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			res := []interface{}{}
			svc := provider.Rds(regionVal)
			ctx := context.Background()

			var marker *string
			for {
				dbInstances, err := svc.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{Marker: marker})
				if err != nil {
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
						mqlSg, err := d.MotorRuntime.CreateResource("aws.ec2.securitygroup",
							"arn", fmt.Sprintf(securityGroupArnPattern, regionVal, account.ID, core.ToString(dbInstance.VpcSecurityGroups[i].VpcSecurityGroupId)),
						)
						if err != nil {
							return nil, err
						}
						sgs = append(sgs, mqlSg)
					}

					mqlDBInstance, err := d.MotorRuntime.CreateResource("aws.rds.dbinstance",
						"arn", core.ToString(dbInstance.DBInstanceArn),
						"name", core.ToString(dbInstance.DBName),
						"backupRetentionPeriod", int64(dbInstance.BackupRetentionPeriod),
						"storageEncrypted", dbInstance.StorageEncrypted,
						"region", regionVal,
						"publiclyAccessible", dbInstance.PubliclyAccessible,
						"enabledCloudwatchLogsExports", stringSliceInterface,
						"enhancedMonitoringResourceArn", core.ToString(dbInstance.EnhancedMonitoringResourceArn),
						"multiAZ", dbInstance.MultiAZ,
						"id", core.ToString(dbInstance.DBInstanceIdentifier),
						"deletionProtection", dbInstance.DeletionProtection,
						"tags", rdsTagsToMap(dbInstance.TagList),
						"dbInstanceClass", core.ToString(dbInstance.DBInstanceClass),
						"dbInstanceIdentifier", core.ToString(dbInstance.DBInstanceIdentifier),
						"engine", core.ToString(dbInstance.Engine),
						"securityGroups", sgs,
						"status", core.ToString(dbInstance.DBInstanceStatus),
					)
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

func rdsTagsToMap(tags []types.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[core.ToString(tag.Key)] = core.ToString(tag.Value)
		}
	}

	return tagsMap
}

func (d *mqlAwsRds) GetDbClusters() ([]interface{}, error) {
	provider, err := awsProvider(d.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(d.getDbClusters(provider), 5)
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

func (p *mqlAwsRdsDbinstance) init(args *resources.Args) (*resources.Args, AwsRdsDbinstance, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch rds db instance")
	}

	// load all rds db instances
	obj, err := p.MotorRuntime.CreateResource("aws.rds")
	if err != nil {
		return nil, nil, err
	}
	rds := obj.(AwsRds)

	rawResources, err := rds.DbInstances()
	if err != nil {
		return nil, nil, err
	}

	arnVal := (*args)["arn"].(string)
	for i := range rawResources {
		dbInstance := rawResources[i].(AwsRdsDbinstance)
		mqlDbArn, err := dbInstance.Arn()
		if err != nil {
			return nil, nil, errors.New("rds db instance does not exist")
		}
		if mqlDbArn == arnVal {
			return args, dbInstance, nil
		}
	}
	return nil, nil, errors.New("rds db instance does not exist")
}

func (d *mqlAwsRds) getDbClusters(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	account, err := provider.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			res := []interface{}{}
			svc := provider.Rds(regionVal)
			ctx := context.Background()

			var marker *string
			for {
				dbClusters, err := svc.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{Marker: marker})
				if err != nil {
					return nil, err
				}

				for _, cluster := range dbClusters.DBClusters {
					mqlRdsDbInstances := []interface{}{}
					for _, instance := range cluster.DBClusterMembers {
						mqlInstance, err := d.MotorRuntime.CreateResource("aws.rds.dbinstance",
							"arn", fmt.Sprintf(rdsInstanceArnPattern, regionVal, account.ID, core.ToString(instance.DBInstanceIdentifier)),
						)
						if err != nil {
							return nil, err
						}
						mqlRdsDbInstances = append(mqlRdsDbInstances, mqlInstance)
					}
					mqlDbCluster, err := d.MotorRuntime.CreateResource("aws.rds.dbcluster",
						"arn", core.ToString(cluster.DBClusterArn),
						"region", regionVal,
						"id", core.ToString(cluster.DBClusterIdentifier),
						"members", mqlRdsDbInstances,
						"tags", rdsTagsToMap(cluster.TagList),
					)
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

func (d *mqlAwsRdsDbcluster) GetSnapshots() ([]interface{}, error) {
	dbClusterId, err := d.Id()
	if err != nil {
		return nil, err
	}
	region, err := d.Region()
	if err != nil {
		return nil, err
	}
	provider, err := awsProvider(d.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Rds(region)
	ctx := context.Background()
	res := []interface{}{}

	var marker *string
	for {
		snapshots, err := svc.DescribeDBClusterSnapshots(ctx, &rds.DescribeDBClusterSnapshotsInput{DBClusterIdentifier: &dbClusterId, Marker: marker})
		if err != nil {
			return nil, err
		}
		for _, snapshot := range snapshots.DBClusterSnapshots {
			mqlDbSnapshot, err := d.MotorRuntime.CreateResource("aws.rds.snapshot",
				"arn", core.ToString(snapshot.DBClusterSnapshotArn),
				"id", core.ToString(snapshot.DBClusterSnapshotIdentifier),
				"type", core.ToString(snapshot.SnapshotType),
				"region", region,
				"encrypted", snapshot.StorageEncrypted,
				"isClusterSnapshot", true,
			)
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

func (d *mqlAwsRdsDbinstance) GetSnapshots() ([]interface{}, error) {
	instanceId, err := d.Id()
	if err != nil {
		return nil, err
	}
	region, err := d.Region()
	if err != nil {
		return nil, err
	}
	provider, err := awsProvider(d.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Rds(region)
	ctx := context.Background()
	res := []interface{}{}

	var marker *string
	for {
		snapshots, err := svc.DescribeDBSnapshots(ctx, &rds.DescribeDBSnapshotsInput{DBInstanceIdentifier: &instanceId, Marker: marker})
		if err != nil {
			return nil, err
		}
		for _, snapshot := range snapshots.DBSnapshots {
			mqlDbSnapshot, err := d.MotorRuntime.CreateResource("aws.rds.snapshot",
				"arn", core.ToString(snapshot.DBSnapshotArn),
				"id", core.ToString(snapshot.DBSnapshotIdentifier),
				"type", core.ToString(snapshot.SnapshotType),
				"region", region,
				"encrypted", snapshot.Encrypted,
				"isClusterSnapshot", false,
				"tags", rdsTagsToMap(snapshot.TagList),
			)
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

func (d *mqlAwsRdsDbinstance) id() (string, error) {
	return d.Arn()
}

func (d *mqlAwsRdsDbcluster) id() (string, error) {
	return d.Arn()
}

func (d *mqlAwsRdsSnapshot) id() (string, error) {
	return d.Arn()
}

func (d *mqlAwsRdsSnapshot) GetAttributes() ([]interface{}, error) {
	snapshotId, err := d.Id()
	if err != nil {
		return nil, err
	}
	region, err := d.Region()
	if err != nil {
		return nil, err
	}
	isCluster, err := d.IsClusterSnapshot()
	if err != nil {
		return nil, err
	}
	provider, err := awsProvider(d.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Rds(region)
	ctx := context.Background()
	if isCluster == true {
		snapshotAttributes, err := svc.DescribeDBClusterSnapshotAttributes(ctx, &rds.DescribeDBClusterSnapshotAttributesInput{DBClusterSnapshotIdentifier: &snapshotId})
		if err != nil {
			return nil, err
		}
		return core.JsonToDictSlice(snapshotAttributes.DBClusterSnapshotAttributesResult.DBClusterSnapshotAttributes)
	}
	snapshotAttributes, err := svc.DescribeDBSnapshotAttributes(ctx, &rds.DescribeDBSnapshotAttributesInput{DBSnapshotIdentifier: &snapshotId})
	if err != nil {
		return nil, err
	}
	return core.JsonToDictSlice(snapshotAttributes.DBSnapshotAttributesResult.DBSnapshotAttributes)
}
