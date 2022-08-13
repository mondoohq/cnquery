package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
	aws_transport "go.mondoo.io/mondoo/motor/providers/aws"
)

func (d *lumiAwsRds) id() (string, error) {
	return "aws.rds", nil
}

const (
	rdsInstanceArnPattern = "arn:aws:rds:%s:%s:db:%s"
)

func (d *lumiAwsRds) GetDbInstances() ([]interface{}, error) {
	at, err := awstransport(d.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(d.getDbInstances(at), 5)
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

func (d *lumiAwsRds) getDbInstances(at *aws_transport.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	account, err := at.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			res := []interface{}{}
			svc := at.Rds(regionVal)
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
						lumiSg, err := d.MotorRuntime.CreateResource("aws.ec2.securitygroup",
							"arn", fmt.Sprintf(securityGroupArnPattern, regionVal, account.ID, toString(dbInstance.VpcSecurityGroups[i].VpcSecurityGroupId)),
						)
						if err != nil {
							return nil, err
						}
						sgs = append(sgs, lumiSg)
					}

					lumiDBInstance, err := d.MotorRuntime.CreateResource("aws.rds.dbinstance",
						"arn", toString(dbInstance.DBInstanceArn),
						"name", toString(dbInstance.DBName),
						"backupRetentionPeriod", int64(dbInstance.BackupRetentionPeriod),
						"storageEncrypted", dbInstance.StorageEncrypted,
						"region", regionVal,
						"publiclyAccessible", dbInstance.PubliclyAccessible,
						"enabledCloudwatchLogsExports", stringSliceInterface,
						"enhancedMonitoringResourceArn", toString(dbInstance.EnhancedMonitoringResourceArn),
						"multiAZ", dbInstance.MultiAZ,
						"id", toString(dbInstance.DBInstanceIdentifier),
						"deletionProtection", dbInstance.DeletionProtection,
						"tags", rdsTagsToMap(dbInstance.TagList),
						"dbInstanceClass", toString(dbInstance.DBInstanceClass),
						"dbInstanceIdentifier", toString(dbInstance.DBInstanceIdentifier),
						"engine", toString(dbInstance.Engine),
						"securityGroups", sgs,
						"status", toString(dbInstance.DBInstanceStatus),
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiDBInstance)
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
			tagsMap[toString(tag.Key)] = toString(tag.Value)
		}
	}

	return tagsMap
}

func (d *lumiAwsRds) GetDbClusters() ([]interface{}, error) {
	at, err := awstransport(d.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(d.getDbClusters(at), 5)
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

func (p *lumiAwsRdsDbinstance) init(args *lumi.Args) (*lumi.Args, AwsRdsDbinstance, error) {
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
		lumiDbArn, err := dbInstance.Arn()
		if err != nil {
			return nil, nil, errors.New("rds db instance does not exist")
		}
		if lumiDbArn == arnVal {
			return args, dbInstance, nil
		}
	}
	return nil, nil, errors.New("rds db instance does not exist")
}

func (d *lumiAwsRds) getDbClusters(at *aws_transport.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	account, err := at.Account()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			res := []interface{}{}
			svc := at.Rds(regionVal)
			ctx := context.Background()

			var marker *string
			for {
				dbClusters, err := svc.DescribeDBClusters(ctx, &rds.DescribeDBClustersInput{Marker: marker})
				if err != nil {
					return nil, err
				}

				for _, cluster := range dbClusters.DBClusters {
					lumiRdsDbInstances := []interface{}{}
					for _, instance := range cluster.DBClusterMembers {
						lumiInstance, err := d.MotorRuntime.CreateResource("aws.rds.dbinstance",
							"arn", fmt.Sprintf(rdsInstanceArnPattern, regionVal, account.ID, toString(instance.DBInstanceIdentifier)),
						)
						if err != nil {
							return nil, err
						}
						lumiRdsDbInstances = append(lumiRdsDbInstances, lumiInstance)
					}
					lumiDbCluster, err := d.MotorRuntime.CreateResource("aws.rds.dbcluster",
						"arn", toString(cluster.DBClusterArn),
						"region", regionVal,
						"id", toString(cluster.DBClusterIdentifier),
						"members", lumiRdsDbInstances,
						"tags", rdsTagsToMap(cluster.TagList),
					)
					if err != nil {
						return nil, err
					}
					res = append(res, lumiDbCluster)
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

func (d *lumiAwsRdsDbcluster) GetSnapshots() ([]interface{}, error) {
	dbClusterId, err := d.Id()
	if err != nil {
		return nil, err
	}
	region, err := d.Region()
	if err != nil {
		return nil, err
	}
	at, err := awstransport(d.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.Rds(region)
	ctx := context.Background()
	res := []interface{}{}

	var marker *string
	for {
		snapshots, err := svc.DescribeDBClusterSnapshots(ctx, &rds.DescribeDBClusterSnapshotsInput{DBClusterIdentifier: &dbClusterId, Marker: marker})
		if err != nil {
			return nil, err
		}
		for _, snapshot := range snapshots.DBClusterSnapshots {
			lumiDbSnapshot, err := d.MotorRuntime.CreateResource("aws.rds.snapshot",
				"arn", toString(snapshot.DBClusterSnapshotArn),
				"id", toString(snapshot.DBClusterSnapshotIdentifier),
				"type", toString(snapshot.SnapshotType),
				"region", region,
				"encrypted", snapshot.StorageEncrypted,
				"isClusterSnapshot", true,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, lumiDbSnapshot)
		}
		if snapshots.Marker == nil {
			break
		}
		marker = snapshots.Marker
	}
	return res, nil
}

func (d *lumiAwsRdsDbinstance) GetSnapshots() ([]interface{}, error) {
	instanceId, err := d.Id()
	if err != nil {
		return nil, err
	}
	region, err := d.Region()
	if err != nil {
		return nil, err
	}
	at, err := awstransport(d.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.Rds(region)
	ctx := context.Background()
	res := []interface{}{}

	var marker *string
	for {
		snapshots, err := svc.DescribeDBSnapshots(ctx, &rds.DescribeDBSnapshotsInput{DBInstanceIdentifier: &instanceId, Marker: marker})
		if err != nil {
			return nil, err
		}
		for _, snapshot := range snapshots.DBSnapshots {
			lumiDbSnapshot, err := d.MotorRuntime.CreateResource("aws.rds.snapshot",
				"arn", toString(snapshot.DBSnapshotArn),
				"id", toString(snapshot.DBSnapshotIdentifier),
				"type", toString(snapshot.SnapshotType),
				"region", region,
				"encrypted", snapshot.Encrypted,
				"isClusterSnapshot", false,
				"tags", rdsTagsToMap(snapshot.TagList),
			)
			if err != nil {
				return nil, err
			}
			res = append(res, lumiDbSnapshot)
		}
		if snapshots.Marker == nil {
			break
		}
		marker = snapshots.Marker
	}
	return res, nil
}

func (d *lumiAwsRdsDbinstance) id() (string, error) {
	return d.Arn()
}

func (d *lumiAwsRdsDbcluster) id() (string, error) {
	return d.Arn()
}

func (d *lumiAwsRdsSnapshot) id() (string, error) {
	return d.Arn()
}

func (d *lumiAwsRdsSnapshot) GetAttributes() ([]interface{}, error) {
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
	at, err := awstransport(d.MotorRuntime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.Rds(region)
	ctx := context.Background()
	if isCluster == true {
		snapshotAttributes, err := svc.DescribeDBClusterSnapshotAttributes(ctx, &rds.DescribeDBClusterSnapshotAttributesInput{DBClusterSnapshotIdentifier: &snapshotId})
		if err != nil {
			return nil, err
		}
		return jsonToDictSlice(snapshotAttributes.DBClusterSnapshotAttributesResult.DBClusterSnapshotAttributes)
	}
	snapshotAttributes, err := svc.DescribeDBSnapshotAttributes(ctx, &rds.DescribeDBSnapshotAttributesInput{DBSnapshotIdentifier: &snapshotId})
	if err != nil {
		return nil, err
	}
	return jsonToDictSlice(snapshotAttributes.DBSnapshotAttributesResult.DBSnapshotAttributes)
}
