package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi/library/jobpool"
	aws_transport "go.mondoo.io/mondoo/motor/transports/aws"
)

func (d *lumiAwsDynamodb) id() (string, error) {
	return "aws.dynamodb", nil
}

const (
	dynamoTableArnPattern       = "arn:aws:dynamodb:%s:%s:table/%s"
	limitsArn                   = "arn:aws:dynamodb:%s:%s"
	dynamoGlobalTableArnPattern = "arn:aws:dynamodb:-:%s:globaltable/%s"
)

func (d *lumiAwsDynamodb) GetBackups() ([]interface{}, error) {
	at, err := awstransport(d.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(d.getBackups(at), 5)
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

func (d *lumiAwsDynamodb) getBackups(at *aws_transport.Transport) []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
	regions, err := at.GetRegions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		regionVal := region
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("calling aws with region %s", regionVal)

			svc := at.Dynamodb(regionVal)
			ctx := context.Background()

			// no pagination required
			listBackupsResp, err := svc.ListBackups(ctx, &dynamodb.ListBackupsInput{})
			if err != nil {
				return nil, errors.Wrap(err, "could not gather aws dynamodb backups")
			}
			backupSummary, err := jsonToDictSlice(listBackupsResp.BackupSummaries)
			if err != nil {
				return nil, err
			}
			return jobpool.JobResult(backupSummary), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (d *lumiAwsDynamodbTable) GetBackups() ([]interface{}, error) {
	tableName, err := d.Name()
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
	svc := at.Dynamodb(region)
	ctx := context.Background()

	// no pagination required
	listBackupsResp, err := svc.ListBackups(ctx, &dynamodb.ListBackupsInput{TableName: &tableName})
	if err != nil {
		return nil, errors.Wrap(err, "could not gather aws dynamodb backups")
	}
	return jsonToDictSlice(listBackupsResp.BackupSummaries)
}

func (d *lumiAwsDynamodb) GetLimits() ([]interface{}, error) {
	at, err := awstransport(d.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(d.getLimits(at), 5)
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

func (d *lumiAwsDynamodb) getLimits(at *aws_transport.Transport) []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)

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

			svc := at.Dynamodb(regionVal)
			ctx := context.Background()

			// no pagination required
			limitsResp, err := svc.DescribeLimits(ctx, &dynamodb.DescribeLimitsInput{})
			if err != nil {
				return nil, errors.Wrap(err, "could not gather aws dynamodb backups")
			}

			lumiLimits, err := d.Runtime.CreateResource("aws.dynamodb.limit",
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
			return jobpool.JobResult(lumiLimits), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (d *lumiAwsDynamodb) GetGlobalTables() ([]interface{}, error) {
	at, err := awstransport(d.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	account, err := at.Account()
	if err != nil {
		return nil, err
	}
	svc := at.Dynamodb("")
	ctx := context.Background()

	// no pagination required
	listGlobalTablesResp, err := svc.ListGlobalTables(ctx, &dynamodb.ListGlobalTablesInput{})
	if err != nil {
		return nil, errors.Wrap(err, "could not gather aws dynamodb global tables")
	}
	res := []interface{}{}
	for _, table := range listGlobalTablesResp.GlobalTables {
		lumiTable, err := d.Runtime.CreateResource("aws.dynamodb.globaltable",
			"arn", fmt.Sprintf(dynamoGlobalTableArnPattern, account.ID, toString(table.GlobalTableName)),
			"name", toString(table.GlobalTableName),
		)
		if err != nil {
			return nil, err
		}
		res = append(res, lumiTable)
	}
	return res, nil
}

func (d *lumiAwsDynamodb) GetTables() ([]interface{}, error) {
	at, err := awstransport(d.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(d.getTables(at), 5)
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

func (d *lumiAwsDynamodb) getTables(at *aws_transport.Transport) []*jobpool.Job {
	var tasks = make([]*jobpool.Job, 0)
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

			svc := at.Dynamodb(regionVal)
			ctx := context.Background()

			// no pagination required
			listTablesResp, err := svc.ListTables(ctx, &dynamodb.ListTablesInput{})
			if err != nil {
				return nil, errors.Wrap(err, "could not gather aws dynamodb tables")
			}
			res := []interface{}{}
			for _, tableName := range listTablesResp.TableNames {
				// call describe table to get real info/details about the table
				table, err := svc.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: &tableName})
				if err != nil {
					return nil, errors.Wrap(err, "could not get aws dynamodb table")
				}
				sseDict, err := jsonToDict(table.Table.SSEDescription)
				if err != nil {
					return nil, err
				}
				throughputDict, err := jsonToDict(table.Table.ProvisionedThroughput)
				if err != nil {
					return nil, err
				}
				lumiTable, err := d.Runtime.CreateResource("aws.dynamodb.table",
					"arn", fmt.Sprintf(dynamoTableArnPattern, regionVal, account.ID, tableName),
					"name", tableName,
					"region", regionVal,
					"sseDescription", sseDict,
					"provisionedThroughput", throughputDict,
				)
				if err != nil {
					return nil, err
				}
				res = append(res, lumiTable)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (d *lumiAwsDynamodbGlobaltable) GetReplicaSettings() ([]interface{}, error) {
	tableName, err := d.Name()
	if err != nil {
		return nil, err
	}
	at, err := awstransport(d.Runtime.Motor.Transport)
	if err != nil {
		return nil, err
	}
	svc := at.Dynamodb("")
	ctx := context.Background()

	// no pagination required
	tableSettingsResp, err := svc.DescribeGlobalTableSettings(ctx, &dynamodb.DescribeGlobalTableSettingsInput{GlobalTableName: &tableName})
	if err != nil {
		return nil, errors.Wrap(err, "could not gather aws dynamodb table settings")
	}
	return jsonToDictSlice(tableSettingsResp.ReplicaSettings)
}

func (d *lumiAwsDynamodbTable) GetContinuousBackups() (interface{}, error) {
	tableName, err := d.Name()
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
	svc := at.Dynamodb(region)
	ctx := context.Background()

	// no pagination required
	continuousBackupsResp, err := svc.DescribeContinuousBackups(ctx, &dynamodb.DescribeContinuousBackupsInput{TableName: &tableName})
	if err != nil {
		return nil, errors.Wrap(err, "could not gather aws dynamodb continuous backups")
	}
	return jsonToDict(continuousBackupsResp.ContinuousBackupsDescription)
}

func (d *lumiAwsDynamodbGlobaltable) id() (string, error) {
	return d.Arn()
}

func (d *lumiAwsDynamodbTable) id() (string, error) {
	return d.Arn()
}

func (d *lumiAwsDynamodbLimit) id() (string, error) {
	return d.Arn()
}
