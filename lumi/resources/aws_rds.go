package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
)

func (d *lumiAwsRds) id() (string, error) {
	return "aws.rds", nil
}

const (
	rdsInstanceArnPattern = "arn:aws:rds:%s:%s:db:%s"
)

func (d *lumiAwsRds) GetDbInstances() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(d.getDbInstances(), 5)
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

func (d *lumiAwsRds) getDbInstances() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(d.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	regions, err := at.GetRegions()
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
				dbInstances, err := svc.DescribeDBInstancesRequest(&rds.DescribeDBInstancesInput{Marker: marker}).Send(ctx)
				if err != nil {
					return nil, err
				}
				for _, dbInstance := range dbInstances.DBInstances {
					stringSliceInterface := []interface{}{}
					for _, logExport := range dbInstance.EnabledCloudwatchLogsExports {
						stringSliceInterface = append(stringSliceInterface, logExport)
					}
					lumiDBInstance, err := d.Runtime.CreateResource("aws.rds.dbinstance",
						"arn", toString(dbInstance.DBInstanceArn),
						"name", toString(dbInstance.DBName),
						"backupRetentionPeriod", toInt64(dbInstance.BackupRetentionPeriod),
						"storageEncrypted", toBool(dbInstance.StorageEncrypted),
						"region", regionVal,
						"publiclyAccessible", toBool(dbInstance.PubliclyAccessible),
						"enabledCloudwatchLogsExports", stringSliceInterface,
						"enhancedMonitoringResourceArn", toString(dbInstance.EnhancedMonitoringResourceArn),
						"multiAZ", toBool(dbInstance.MultiAZ),
						"id", toString(dbInstance.DBInstanceIdentifier),
						"deletionProtection", toBool(dbInstance.DeletionProtection),
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

func (d *lumiAwsRds) GetDbClusters() ([]interface{}, error) {
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(d.getDbClusters(), 5)
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
	obj, err := p.Runtime.CreateResource("aws.rds")
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

func (d *lumiAwsRds) getDbClusters() []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	at, err := awstransport(d.Runtime.Motor.Transport)
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
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
				dbClusters, err := svc.DescribeDBClustersRequest(&rds.DescribeDBClustersInput{Marker: marker}).Send(ctx)
				if err != nil {
					return nil, err
				}

				for _, cluster := range dbClusters.DBClusters {
					lumiRdsDbInstances := []interface{}{}
					for _, instance := range cluster.DBClusterMembers {
						lumiInstance, err := d.Runtime.CreateResource("aws.rds.dbinstance",
							"arn", fmt.Sprintf(rdsInstanceArnPattern, regionVal, account.ID, toString(instance.DBInstanceIdentifier)),
						)
						if err != nil {
							return nil, err
						}
						lumiRdsDbInstances = append(lumiRdsDbInstances, lumiInstance)
					}
					lumiDbCluster, err := d.Runtime.CreateResource("aws.rds.dbcluster",
						"arn", toString(cluster.DBClusterArn),
						"region", regionVal,
						"id", toString(cluster.DBClusterIdentifier),
						"members", lumiRdsDbInstances,
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
	at, err := awstransport(d.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.Rds(region)
	ctx := context.Background()
	res := []interface{}{}

	var marker *string
	for {
		snapshots, err := svc.DescribeDBClusterSnapshotsRequest(&rds.DescribeDBClusterSnapshotsInput{DBClusterIdentifier: &dbClusterId, Marker: marker}).Send(ctx)
		if err != nil {
			return nil, err
		}
		for _, snapshot := range snapshots.DBClusterSnapshots {
			lumiDbSnapshot, err := d.Runtime.CreateResource("aws.rds.snapshot",
				"arn", toString(snapshot.DBClusterSnapshotArn),
				"id", toString(snapshot.DBClusterSnapshotIdentifier),
				"type", toString(snapshot.SnapshotType),
				"region", region,
				"encrypted", toBool(snapshot.StorageEncrypted),
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
	at, err := awstransport(d.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.Rds(region)
	ctx := context.Background()
	res := []interface{}{}

	var marker *string
	for {
		snapshots, err := svc.DescribeDBSnapshotsRequest(&rds.DescribeDBSnapshotsInput{DBInstanceIdentifier: &instanceId, Marker: marker}).Send(ctx)
		if err != nil {
			return nil, err
		}
		for _, snapshot := range snapshots.DBSnapshots {
			lumiDbSnapshot, err := d.Runtime.CreateResource("aws.rds.snapshot",
				"arn", toString(snapshot.DBSnapshotArn),
				"id", toString(snapshot.DBSnapshotIdentifier),
				"type", toString(snapshot.SnapshotType),
				"region", region,
				"encrypted", toBool(snapshot.Encrypted),
				"isClusterSnapshot", false,
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
	at, err := awstransport(d.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}

	svc := at.Rds(region)
	ctx := context.Background()
	if isCluster == true {
		snapshotAttributes, err := svc.DescribeDBClusterSnapshotAttributesRequest(&rds.DescribeDBClusterSnapshotAttributesInput{DBClusterSnapshotIdentifier: &snapshotId}).Send(ctx)
		if err != nil {
			return nil, err
		}
		return jsonToDictSlice(snapshotAttributes.DBClusterSnapshotAttributesResult.DBClusterSnapshotAttributes)
	}
	snapshotAttributes, err := svc.DescribeDBSnapshotAttributesRequest(&rds.DescribeDBSnapshotAttributesInput{DBSnapshotIdentifier: &snapshotId}).Send(ctx)
	if err != nil {
		return nil, err
	}
	return jsonToDictSlice(snapshotAttributes.DBSnapshotAttributesResult.DBSnapshotAttributes)
}
