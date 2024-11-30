// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
)

func (a *mqlAwsDynamodb) id() (string, error) {
	return "aws.dynamodb", nil
}

func (a *mqlAwsDynamodb) exports() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getExports(conn), 5)
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

func (a *mqlAwsDynamodbExport) id() (string, error) {
	return a.Arn.Data, nil
}

type mqlAwsDynamodbExportInternal struct {
	exportCache *ddtypes.ExportDescription
	region      string
	arn         string
	lock        sync.Mutex
}

func (a *mqlAwsDynamodbExport) fetchExport() (*ddtypes.ExportDescription, error) {
	if a.exportCache != nil {
		return a.exportCache, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	ctx := context.Background()
	svc := conn.Dynamodb(a.region)
	desc, err := svc.DescribeExport(ctx, &dynamodb.DescribeExportInput{ExportArn: aws.String(a.arn)})
	if err != nil {
		return nil, err
	}
	a.exportCache = desc.ExportDescription
	return desc.ExportDescription, nil
}

func (a *mqlAwsDynamodb) getExports(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("dynamodb>getExports>calling aws with region %s", regionVal)

			svc := conn.Dynamodb(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			// no pagination required
			listExportsResp, err := svc.ListExports(ctx, &dynamodb.ListExportsInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, errors.Wrap(err, "could not gather aws dynamodb exports")
			}
			for i := range listExportsResp.ExportSummaries {
				exp := listExportsResp.ExportSummaries[i]
				mqlExport, err := CreateResource(a.MqlRuntime, "aws.dynamodb.export",
					map[string]*llx.RawData{
						"arn":    llx.StringDataPtr(exp.ExportArn),
						"type":   llx.StringData(string(exp.ExportType)),
						"status": llx.StringData(string(exp.ExportStatus)),
					})
				if err != nil {
					return nil, err
				}
				mqlExport.(*mqlAwsDynamodbExport).arn = *exp.ExportArn
				mqlExport.(*mqlAwsDynamodbExport).region = region
				res = append(res, mqlExport)
			}

			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsDynamodbExport) s3Prefix() (string, error) {
	exp, err := a.fetchExport()
	if err != nil {
		return "", err
	}
	if exp.S3Prefix != nil {
		return *exp.S3Prefix, nil
	}
	return "", nil
}

func (a *mqlAwsDynamodbExport) itemCount() (int64, error) {
	exp, err := a.fetchExport()
	if err != nil {
		return 0, err
	}
	if exp.ItemCount != nil {
		return *exp.ItemCount, nil
	}
	return 0, nil
}

func (a *mqlAwsDynamodbExport) format() (string, error) {
	exp, err := a.fetchExport()
	if err != nil {
		return "", err
	}
	return string(exp.ExportFormat), nil
}

func (a *mqlAwsDynamodbExport) startTime() (*time.Time, error) {
	exp, err := a.fetchExport()
	if err != nil {
		return nil, err
	}
	return exp.StartTime, nil
}

func (a *mqlAwsDynamodbExport) endTime() (*time.Time, error) {
	exp, err := a.fetchExport()
	if err != nil {
		return nil, err
	}
	return exp.EndTime, nil
}

func (a *mqlAwsDynamodbExport) s3SseAlgorithm() (string, error) {
	exp, err := a.fetchExport()
	if err != nil {
		return "", err
	}
	return string(exp.S3SseAlgorithm), nil
}

func (a *mqlAwsDynamodbExport) s3Bucket() (*mqlAwsS3Bucket, error) {
	exp, err := a.fetchExport()
	if err != nil {
		return nil, err
	}
	mqlS3Bucket, err := NewResource(a.MqlRuntime, "aws.s3.bucket",
		map[string]*llx.RawData{
			"name": llx.StringDataPtr(exp.S3Bucket),
		})
	if err != nil {
		return nil, err
	}
	return mqlS3Bucket.(*mqlAwsS3Bucket), nil
}

func (a *mqlAwsDynamodbExport) kmsKey() (*mqlAwsKmsKey, error) {
	exp, err := a.fetchExport()
	if err != nil {
		return nil, err
	}
	if exp.S3SseKmsKeyId == nil {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	mqlKey, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{
			"arn": llx.StringData(fmt.Sprintf(kmsKeyArnPattern, a.region, conn.AccountId(), convert.ToString(exp.S3SseKmsKeyId))),
		})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsDynamodbExport) table() (*mqlAwsDynamodbTable, error) {
	exp, err := a.fetchExport()
	if err != nil {
		return nil, err
	}
	mqltable, err := NewResource(a.MqlRuntime, "aws.dynamodb.table",
		map[string]*llx.RawData{
			"arn": llx.StringDataPtr(exp.TableArn),
		})
	if err != nil {
		return nil, err
	}
	return mqltable.(*mqlAwsDynamodbTable), nil
}

func (a *mqlAwsDynamodb) backups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getBackups(conn), 5)
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

func (a *mqlAwsDynamodb) getBackups(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("dynamodb>getBackups>calling aws with region %s", regionVal)

			svc := conn.Dynamodb(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			// no pagination required
			listBackupsResp, err := svc.ListBackups(ctx, &dynamodb.ListBackupsInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, errors.Wrap(err, "could not gather aws dynamodb backups")
			}
			backupSummary, err := convert.JsonToDictSlice(listBackupsResp.BackupSummaries)
			if err != nil {
				return nil, err
			}
			return jobpool.JobResult(backupSummary), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initAwsDynamodbTable(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
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
		return nil, nil, errors.New("arn required to fetch dynamodb table")
	}

	// load all rds db instances
	obj, err := CreateResource(runtime, "aws.dynamodb", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	dynamodb := obj.(*mqlAwsDynamodb)

	rawResources := dynamodb.GetTables()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	arnVal := args["arn"].Value.(string)
	for i := range rawResources.Data {
		dbInstance := rawResources.Data[i].(*mqlAwsDynamodbTable)
		if dbInstance.Arn.Data == arnVal {
			return args, dbInstance, nil
		}
	}
	return nil, nil, errors.New("dynamo db table does not exist")
}

func (a *mqlAwsDynamodbTable) backups() ([]interface{}, error) {
	tableName := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Dynamodb(region)
	ctx := context.Background()

	listBackupsResp, err := svc.ListBackups(ctx, &dynamodb.ListBackupsInput{TableName: &tableName})
	if err != nil {
		return nil, errors.Wrap(err, "could not gather aws dynamodb backups")
	}
	return convert.JsonToDictSlice(listBackupsResp.BackupSummaries)
}

func (a *mqlAwsDynamodbTable) tags() (map[string]interface{}, error) {
	tableArn := a.Arn.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Dynamodb(region)
	ctx := context.Background()
	tags, err := svc.ListTagsOfResource(ctx, &dynamodb.ListTagsOfResourceInput{ResourceArn: &tableArn})
	if err != nil {
		return nil, err
	}

	return dynamoDBTagsToMap(tags.Tags), nil
}

func (a *mqlAwsDynamodb) limits() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getLimits(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.(interface{}))
	}
	return res, nil
}

func (a *mqlAwsDynamodb) getLimits(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)

	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("dynamodb>getLimits>calling aws with region %s", regionVal)

			svc := conn.Dynamodb(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			// no pagination required
			limitsResp, err := svc.DescribeLimits(ctx, &dynamodb.DescribeLimitsInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, errors.Wrap(err, "could not gather aws dynamodb backups")
			}

			mqlLimits, err := CreateResource(a.MqlRuntime, "aws.dynamodb.limit",
				map[string]*llx.RawData{
					"arn":             llx.StringData(fmt.Sprintf(limitsArn, regionVal, conn.AccountId())),
					"region":          llx.StringData(regionVal),
					"accountMaxRead":  llx.IntData(*limitsResp.AccountMaxReadCapacityUnits),
					"accountMaxWrite": llx.IntData(*limitsResp.AccountMaxWriteCapacityUnits),
					"tableMaxRead":    llx.IntData(*limitsResp.TableMaxReadCapacityUnits),
					"tableMaxWrite":   llx.IntData(*limitsResp.TableMaxWriteCapacityUnits),
				})
			if err != nil {
				return nil, err
			}
			return jobpool.JobResult(mqlLimits), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsDynamodb) globalTables() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Dynamodb("")
	ctx := context.Background()

	// no pagination required
	listGlobalTablesResp, err := svc.ListGlobalTables(ctx, &dynamodb.ListGlobalTablesInput{})
	if err != nil {
		return nil, errors.Wrap(err, "could not gather aws dynamodb global tables")
	}
	res := []interface{}{}
	for _, table := range listGlobalTablesResp.GlobalTables {
		mqlTable, err := CreateResource(a.MqlRuntime, "aws.dynamodb.globaltable",
			map[string]*llx.RawData{
				"arn":  llx.StringData(fmt.Sprintf(dynamoGlobalTableArnPattern, conn.AccountId(), convert.ToString(table.GlobalTableName))),
				"name": llx.StringDataPtr(table.GlobalTableName),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlTable)
	}
	return res, nil
}

func (a *mqlAwsDynamodb) tables() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getTables(conn), 5)
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

func (a *mqlAwsDynamodb) getTables(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("dynamodb>getTables>calling aws with region %s", regionVal)

			svc := conn.Dynamodb(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			// no pagination required
			listTablesResp, err := svc.ListTables(ctx, &dynamodb.ListTablesInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, errors.Wrap(err, "could not gather aws dynamodb tables")
			}
			for _, tableName := range listTablesResp.TableNames {
				// call describe table to get real info/details about the table
				table, err := svc.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: &tableName})
				if err != nil {
					return nil, errors.Wrap(err, "could not get aws dynamodb table")
				}
				sseDict, err := convert.JsonToDict(table.Table.SSEDescription)
				if err != nil {
					return nil, err
				}
				throughputDict, err := convert.JsonToDict(table.Table.ProvisionedThroughput)
				if err != nil {
					return nil, err
				}

				mqlTable, err := CreateResource(a.MqlRuntime, "aws.dynamodb.table",
					map[string]*llx.RawData{
						"arn":                       llx.StringData(fmt.Sprintf(dynamoTableArnPattern, regionVal, conn.AccountId(), tableName)),
						"name":                      llx.StringData(tableName),
						"region":                    llx.StringData(regionVal),
						"sseDescription":            llx.DictData(sseDict),
						"provisionedThroughput":     llx.DictData(throughputDict),
						"createdTime":               llx.TimeDataPtr(table.Table.CreationDateTime),
						"createdAt":                 llx.TimeDataPtr(table.Table.CreationDateTime),
						"deletionProtectionEnabled": llx.BoolDataPtr(table.Table.DeletionProtectionEnabled),
						"globalTableVersion":        llx.StringDataPtr(table.Table.GlobalTableVersion),
						"id":                        llx.StringDataPtr(table.Table.TableId),
						"sizeBytes":                 llx.IntDataPtr(table.Table.TableSizeBytes),
						"status":                    llx.StringData(string(table.Table.TableStatus)),
						"items":                     llx.IntDataPtr(table.Table.ItemCount),
						"latestStreamArn":           llx.StringDataPtr(table.Table.LatestStreamArn),
					})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlTable)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func dynamoDBTagsToMap(tags []ddtypes.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[convert.ToString(tag.Key)] = convert.ToString(tag.Value)
		}
	}

	return tagsMap
}

func (a *mqlAwsDynamodbGlobaltable) replicaSettings() ([]interface{}, error) {
	tableName := a.Name.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Dynamodb("")
	ctx := context.Background()

	// no pagination required
	tableSettingsResp, err := svc.DescribeGlobalTableSettings(ctx, &dynamodb.DescribeGlobalTableSettingsInput{GlobalTableName: &tableName})
	if err != nil {
		return nil, errors.Wrap(err, "could not gather aws dynamodb table settings")
	}
	return convert.JsonToDictSlice(tableSettingsResp.ReplicaSettings)
}

func initAwsDynamodbGlobaltable(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			(args)["name"] = llx.StringData(ids.name)
			(args)["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch dynamodb table")
	}

	obj, err := CreateResource(runtime, "aws.dynamodb", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	dynamodb := obj.(*mqlAwsDynamodb)

	rawResources := dynamodb.GetGlobalTables()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	arnVal := args["arn"].Value.(string)
	for i := range rawResources.Data {
		dbInstance := rawResources.Data[i].(*mqlAwsDynamodbGlobaltable)
		if dbInstance.Arn.Data == arnVal {
			return args, dbInstance, nil
		}
	}
	return nil, nil, errors.New("dynamo db table does not exist")
}

func (a *mqlAwsDynamodbTable) continuousBackups() (interface{}, error) {
	tableName := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Dynamodb(region)
	ctx := context.Background()

	// no pagination required
	continuousBackupsResp, err := svc.DescribeContinuousBackups(ctx, &dynamodb.DescribeContinuousBackupsInput{TableName: &tableName})
	if err != nil {
		return nil, errors.Wrap(err, "could not gather aws dynamodb continuous backups")
	}
	return convert.JsonToDict(continuousBackupsResp.ContinuousBackupsDescription)
}

func (a *mqlAwsDynamodbGlobaltable) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsDynamodbTable) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsDynamodbLimit) id() (string, error) {
	return a.Arn.Data, nil
}
