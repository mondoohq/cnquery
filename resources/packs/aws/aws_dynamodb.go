package aws

import (
	"context"
	"fmt"

	"errors"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rs/zerolog/log"
	aws_provider "go.mondoo.com/cnquery/motor/providers/aws"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/library/jobpool"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (d *mqlAwsDynamodb) id() (string, error) {
	return "aws.dynamodb", nil
}

const (
	dynamoTableArnPattern       = "arn:aws:dynamodb:%s:%s:table/%s"
	limitsArn                   = "arn:aws:dynamodb:%s:%s"
	dynamoGlobalTableArnPattern = "arn:aws:dynamodb:-:%s:globaltable/%s"
)

func (d *mqlAwsDynamodb) GetBackups() ([]interface{}, error) {
	provider, err := awsProvider(d.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(d.getBackups(provider), 5)
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

func (d *mqlAwsDynamodb) getBackups(provider *aws_provider.Provider) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := provider.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := provider.Dynamodb(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			// no pagination required
			listBackupsResp, err := svc.ListBackups(ctx, &dynamodb.ListBackupsInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, errors.Join(err, errors.New("could not gather aws dynamodb backups"))
			}
			backupSummary, err := core.JsonToDictSlice(listBackupsResp.BackupSummaries)
			if err != nil {
				return nil, err
			}
			return jobpool.JobResult(backupSummary), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (d *mqlAwsDynamodbTable) init(args *resources.Args) (*resources.Args, AwsDynamodbTable, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(d.MqlResource().MotorRuntime); ids != nil {
			(*args)["name"] = ids.name
			(*args)["arn"] = ids.arn
		}
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch dynamodb table")
	}

	// load all rds db instances
	obj, err := d.MotorRuntime.CreateResource("aws.dynamodb")
	if err != nil {
		return nil, nil, err
	}
	dynamodb := obj.(AwsDynamodb)

	rawResources, err := dynamodb.Tables()
	if err != nil {
		return nil, nil, err
	}

	arnVal := (*args)["arn"].(string)
	for i := range rawResources {
		dbInstance := rawResources[i].(AwsDynamodbTable)
		mqlDbArn, err := dbInstance.Arn()
		if err != nil {
			return nil, nil, errors.New("rds dynamo db table does not exist")
		}
		if mqlDbArn == arnVal {
			return args, dbInstance, nil
		}
	}
	return nil, nil, errors.New("dynamo db table does not exist")
}

func (d *mqlAwsDynamodbTable) GetBackups() ([]interface{}, error) {
	tableName, err := d.Name()
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
	svc := provider.Dynamodb(region)
	ctx := context.Background()

	listBackupsResp, err := svc.ListBackups(ctx, &dynamodb.ListBackupsInput{TableName: &tableName})
	if err != nil {
		return nil, errors.Join(err, errors.New("could not gather aws dynamodb backups"))
	}
	return core.JsonToDictSlice(listBackupsResp.BackupSummaries)
}

func (d *mqlAwsDynamodbTable) GetTags() (map[string]interface{}, error) {
	tableArn, err := d.Arn()
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
	svc := provider.Dynamodb(region)
	ctx := context.Background()
	tags, err := svc.ListTagsOfResource(ctx, &dynamodb.ListTagsOfResourceInput{ResourceArn: &tableArn})
	if err != nil {
		return nil, err
	}

	return dynamoDBTagsToMap(tags.Tags), nil
}

func (d *mqlAwsDynamodb) GetLimits() ([]interface{}, error) {
	provider, err := awsProvider(d.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(d.getLimits(provider), 5)
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

func (d *mqlAwsDynamodb) getLimits(provider *aws_provider.Provider) []*jobpool.Job {
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

			svc := provider.Dynamodb(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			// no pagination required
			limitsResp, err := svc.DescribeLimits(ctx, &dynamodb.DescribeLimitsInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, errors.Join(err, errors.New("could not gather aws dynamodb backups"))
			}

			mqlLimits, err := d.MotorRuntime.CreateResource("aws.dynamodb.limit",
				"arn", fmt.Sprintf(limitsArn, regionVal, account.ID),
				"region", regionVal,
				"accountMaxRead", *limitsResp.AccountMaxReadCapacityUnits,
				"accountMaxWrite", *limitsResp.AccountMaxWriteCapacityUnits,
				"tableMaxRead", *limitsResp.TableMaxReadCapacityUnits,
				"tableMaxWrite", *limitsResp.TableMaxWriteCapacityUnits,
			)
			if err != nil {
				return nil, err
			}
			return jobpool.JobResult(mqlLimits), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (d *mqlAwsDynamodb) GetGlobalTables() ([]interface{}, error) {
	provider, err := awsProvider(d.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	account, err := provider.Account()
	if err != nil {
		return nil, err
	}
	svc := provider.Dynamodb("")
	ctx := context.Background()

	// no pagination required
	listGlobalTablesResp, err := svc.ListGlobalTables(ctx, &dynamodb.ListGlobalTablesInput{})
	if err != nil {
		return nil, errors.Join(err, errors.New("could not gather aws dynamodb global tables"))
	}
	res := []interface{}{}
	for _, table := range listGlobalTablesResp.GlobalTables {
		mqlTable, err := d.MotorRuntime.CreateResource("aws.dynamodb.globaltable",
			"arn", fmt.Sprintf(dynamoGlobalTableArnPattern, account.ID, core.ToString(table.GlobalTableName)),
			"name", core.ToString(table.GlobalTableName),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlTable)
	}
	return res, nil
}

func (d *mqlAwsDynamodb) GetTables() ([]interface{}, error) {
	provider, err := awsProvider(d.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(d.getTables(provider), 5)
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

func (d *mqlAwsDynamodb) getTables(provider *aws_provider.Provider) []*jobpool.Job {
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

			svc := provider.Dynamodb(regionVal)
			ctx := context.Background()
			res := []interface{}{}

			// no pagination required
			listTablesResp, err := svc.ListTables(ctx, &dynamodb.ListTablesInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", regionVal).Msg("error accessing region for AWS API")
					return res, nil
				}
				return nil, errors.Join(err, errors.New("could not gather aws dynamodb tables"))
			}
			for _, tableName := range listTablesResp.TableNames {
				// call describe table to get real info/details about the table
				table, err := svc.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: &tableName})
				if err != nil {
					return nil, errors.Join(err, errors.New("could not get aws dynamodb table"))
				}
				sseDict, err := core.JsonToDict(table.Table.SSEDescription)
				if err != nil {
					return nil, err
				}
				throughputDict, err := core.JsonToDict(table.Table.ProvisionedThroughput)
				if err != nil {
					return nil, err
				}

				mqlTable, err := d.MotorRuntime.CreateResource("aws.dynamodb.table",
					"arn", fmt.Sprintf(dynamoTableArnPattern, regionVal, account.ID, tableName),
					"name", tableName,
					"region", regionVal,
					"sseDescription", sseDict,
					"provisionedThroughput", throughputDict,
				)
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

func dynamoDBTagsToMap(tags []types.Tag) map[string]interface{} {
	tagsMap := make(map[string]interface{})

	if len(tags) > 0 {
		for i := range tags {
			tag := tags[i]
			tagsMap[core.ToString(tag.Key)] = core.ToString(tag.Value)
		}
	}

	return tagsMap
}

func (d *mqlAwsDynamodbGlobaltable) GetReplicaSettings() ([]interface{}, error) {
	tableName, err := d.Name()
	if err != nil {
		return nil, err
	}
	provider, err := awsProvider(d.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}
	svc := provider.Dynamodb("")
	ctx := context.Background()

	// no pagination required
	tableSettingsResp, err := svc.DescribeGlobalTableSettings(ctx, &dynamodb.DescribeGlobalTableSettingsInput{GlobalTableName: &tableName})
	if err != nil {
		return nil, errors.Join(err, errors.New("could not gather aws dynamodb table settings"))
	}
	return core.JsonToDictSlice(tableSettingsResp.ReplicaSettings)
}

func (d *mqlAwsDynamodbGlobaltable) init(args *resources.Args) (*resources.Args, AwsDynamodbGlobaltable, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if len(*args) == 0 {
		if ids := getAssetIdentifier(d.MqlResource().MotorRuntime); ids != nil {
			(*args)["name"] = ids.name
			(*args)["arn"] = ids.arn
		}
	}

	if (*args)["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch dynamodb table")
	}

	obj, err := d.MotorRuntime.CreateResource("aws.dynamodb")
	if err != nil {
		return nil, nil, err
	}
	dynamodb := obj.(AwsDynamodb)

	rawResources, err := dynamodb.GlobalTables()
	if err != nil {
		return nil, nil, err
	}

	arnVal := (*args)["arn"].(string)
	for i := range rawResources {
		dbInstance := rawResources[i].(AwsDynamodbGlobaltable)
		mqlDbArn, err := dbInstance.Arn()
		if err != nil {
			return nil, nil, errors.New("rds dynamo db table does not exist")
		}
		if mqlDbArn == arnVal {
			return args, dbInstance, nil
		}
	}
	return nil, nil, errors.New("dynamo db table does not exist")
}

func (d *mqlAwsDynamodbTable) GetContinuousBackups() (interface{}, error) {
	tableName, err := d.Name()
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
	svc := provider.Dynamodb(region)
	ctx := context.Background()

	// no pagination required
	continuousBackupsResp, err := svc.DescribeContinuousBackups(ctx, &dynamodb.DescribeContinuousBackupsInput{TableName: &tableName})
	if err != nil {
		return nil, errors.Join(err, errors.New("could not gather aws dynamodb continuous backups"))
	}
	return core.JsonToDict(continuousBackupsResp.ContinuousBackupsDescription)
}

func (d *mqlAwsDynamodbGlobaltable) id() (string, error) {
	return d.Arn()
}

func (d *mqlAwsDynamodbTable) id() (string, error) {
	return d.Arn()
}

func (d *mqlAwsDynamodbLimit) id() (string, error) {
	return d.Arn()
}
