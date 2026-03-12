// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/glue"
	glue_types "github.com/aws/aws-sdk-go-v2/service/glue/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsGlue) id() (string, error) {
	return "aws.glue", nil
}

func (a *mqlAwsGlue) crawlers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getCrawlers(conn), 5)
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

func (a *mqlAwsGlue) getCrawlers(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("glue>getCrawlers>calling aws with region %s", region)

			svc := conn.Glue(region)
			ctx := context.Background()
			res := []any{}

			paginator := glue.NewGetCrawlersPaginator(svc, &glue.GetCrawlersInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, crawler := range page.Crawlers {
					mqlCrawler, err := newMqlAwsGlueCrawler(a.MqlRuntime, region, conn.AccountId(), crawler)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlCrawler)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlAwsGlueCrawler(runtime *plugin.Runtime, region string, accountID string, crawler glue_types.Crawler) (*mqlAwsGlueCrawler, error) {
	arn := fmt.Sprintf("arn:aws:glue:%s:%s:crawler/%s", region, accountID, convert.ToValue(crawler.Name))

	targets, err := convert.JsonToDict(crawler.Targets)
	if err != nil {
		return nil, err
	}

	schemaChangePolicy, err := convert.JsonToDict(crawler.SchemaChangePolicy)
	if err != nil {
		return nil, err
	}

	var schedule string
	if crawler.Schedule != nil {
		schedule = convert.ToValue(crawler.Schedule.ScheduleExpression)
	}

	resource, err := CreateResource(runtime, "aws.glue.crawler",
		map[string]*llx.RawData{
			"__id":                  llx.StringData(arn),
			"arn":                   llx.StringData(arn),
			"name":                  llx.StringDataPtr(crawler.Name),
			"role":                  llx.StringDataPtr(crawler.Role),
			"databaseName":          llx.StringDataPtr(crawler.DatabaseName),
			"description":           llx.StringDataPtr(crawler.Description),
			"targets":               llx.DictData(targets),
			"schedule":              llx.StringData(schedule),
			"classifiers":           llx.ArrayData(convert.SliceAnyToInterface(crawler.Classifiers), types.String),
			"schemaChangePolicy":    llx.DictData(schemaChangePolicy),
			"state":                 llx.StringData(string(crawler.State)),
			"configuration":         llx.StringDataPtr(crawler.Configuration),
			"securityConfiguration": llx.StringDataPtr(crawler.CrawlerSecurityConfiguration),
			"createdAt":             llx.TimeDataPtr(crawler.CreationTime),
			"updatedAt":             llx.TimeDataPtr(crawler.LastUpdated),
			"region":                llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsGlueCrawler), nil
}

func (a *mqlAwsGlueCrawler) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Glue(a.Region.Data)
	ctx := context.Background()
	arn := a.Arn.Data

	resp, err := svc.GetTags(ctx, &glue.GetTagsInput{
		ResourceArn: &arn,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			return nil, nil
		}
		return nil, err
	}
	return toInterfaceMap(resp.Tags), nil
}

func (a *mqlAwsGlue) jobs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getJobs(conn), 5)
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

func (a *mqlAwsGlue) getJobs(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("glue>getJobs>calling aws with region %s", region)

			svc := conn.Glue(region)
			ctx := context.Background()
			res := []any{}

			paginator := glue.NewGetJobsPaginator(svc, &glue.GetJobsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, job := range page.Jobs {
					mqlJob, err := newMqlAwsGlueJob(a.MqlRuntime, region, conn.AccountId(), job)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlJob)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlAwsGlueJob(runtime *plugin.Runtime, region string, accountID string, job glue_types.Job) (*mqlAwsGlueJob, error) {
	arn := fmt.Sprintf("arn:aws:glue:%s:%s:job/%s", region, accountID, convert.ToValue(job.Name))

	command, err := convert.JsonToDict(job.Command)
	if err != nil {
		return nil, err
	}

	var connections []any
	if job.Connections != nil {
		connections = convert.SliceAnyToInterface(job.Connections.Connections)
	}

	var maxCapacity float64
	if job.MaxCapacity != nil {
		maxCapacity = *job.MaxCapacity
	}

	resource, err := CreateResource(runtime, "aws.glue.job",
		map[string]*llx.RawData{
			"__id":                  llx.StringData(arn),
			"arn":                   llx.StringData(arn),
			"name":                  llx.StringDataPtr(job.Name),
			"description":           llx.StringDataPtr(job.Description),
			"role":                  llx.StringDataPtr(job.Role),
			"command":               llx.DictData(command),
			"maxRetries":            llx.IntData(int64(job.MaxRetries)),
			"timeout":               llx.IntDataDefault(job.Timeout, 0),
			"glueVersion":           llx.StringDataPtr(job.GlueVersion),
			"numberOfWorkers":       llx.IntDataDefault(job.NumberOfWorkers, 0),
			"workerType":            llx.StringData(string(job.WorkerType)),
			"maxCapacity":           llx.FloatData(maxCapacity),
			"connections":           llx.ArrayData(connections, types.String),
			"defaultArguments":      llx.MapData(toInterfaceMap(job.DefaultArguments), types.String),
			"securityConfiguration": llx.StringDataPtr(job.SecurityConfiguration),
			"executionClass":        llx.StringData(string(job.ExecutionClass)),
			"createdAt":             llx.TimeDataPtr(job.CreatedOn),
			"updatedAt":             llx.TimeDataPtr(job.LastModifiedOn),
			"region":                llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsGlueJob), nil
}

func (a *mqlAwsGlueJob) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Glue(a.Region.Data)
	ctx := context.Background()
	arn := a.Arn.Data

	resp, err := svc.GetTags(ctx, &glue.GetTagsInput{
		ResourceArn: &arn,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			return nil, nil
		}
		return nil, err
	}
	return toInterfaceMap(resp.Tags), nil
}

func (a *mqlAwsGlue) securityConfigurations() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getSecurityConfigurations(conn), 5)
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

func (a *mqlAwsGlue) getSecurityConfigurations(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("glue>getSecurityConfigurations>calling aws with region %s", region)

			svc := conn.Glue(region)
			ctx := context.Background()
			res := []any{}

			paginator := glue.NewGetSecurityConfigurationsPaginator(svc, &glue.GetSecurityConfigurationsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, secConf := range page.SecurityConfigurations {
					mqlSecConf, err := newMqlAwsGlueSecurityConfiguration(a.MqlRuntime, region, conn.AccountId(), secConf)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlSecConf)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlAwsGlueSecurityConfiguration(runtime *plugin.Runtime, region string, accountID string, secConf glue_types.SecurityConfiguration) (*mqlAwsGlueSecurityConfiguration, error) {
	id := fmt.Sprintf("arn:aws:glue:%s:%s:security-configuration/%s", region, accountID, convert.ToValue(secConf.Name))

	var s3Enc, cwEnc, jbEnc any
	if secConf.EncryptionConfiguration != nil {
		var err error
		if len(secConf.EncryptionConfiguration.S3Encryption) > 0 {
			s3Enc, err = convert.JsonToDict(secConf.EncryptionConfiguration.S3Encryption[0])
			if err != nil {
				return nil, err
			}
		}
		cwEnc, err = convert.JsonToDict(secConf.EncryptionConfiguration.CloudWatchEncryption)
		if err != nil {
			return nil, err
		}
		jbEnc, err = convert.JsonToDict(secConf.EncryptionConfiguration.JobBookmarksEncryption)
		if err != nil {
			return nil, err
		}
	}

	resource, err := CreateResource(runtime, "aws.glue.securityConfiguration",
		map[string]*llx.RawData{
			"__id":                   llx.StringData(id),
			"name":                   llx.StringDataPtr(secConf.Name),
			"createdAt":              llx.TimeDataPtr(secConf.CreatedTimeStamp),
			"s3Encryption":           llx.DictData(s3Enc),
			"cloudWatchEncryption":   llx.DictData(cwEnc),
			"jobBookmarksEncryption": llx.DictData(jbEnc),
			"region":                 llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsGlueSecurityConfiguration), nil
}

func (a *mqlAwsGlue) databases() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getDatabases(conn), 5)
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

func (a *mqlAwsGlue) getDatabases(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("glue>getDatabases>calling aws with region %s", region)

			svc := conn.Glue(region)
			ctx := context.Background()
			res := []any{}

			paginator := glue.NewGetDatabasesPaginator(svc, &glue.GetDatabasesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, db := range page.DatabaseList {
					mqlDb, err := newMqlAwsGlueDatabase(a.MqlRuntime, region, db)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlDb)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlAwsGlueDatabase(runtime *plugin.Runtime, region string, db glue_types.Database) (*mqlAwsGlueDatabase, error) {
	id := fmt.Sprintf("glue/database/%s/%s/%s", region, convert.ToValue(db.CatalogId), convert.ToValue(db.Name))

	var params map[string]any
	if db.Parameters != nil {
		params = toInterfaceMap(db.Parameters)
	}

	resource, err := CreateResource(runtime, "aws.glue.database",
		map[string]*llx.RawData{
			"__id":        llx.StringData(id),
			"name":        llx.StringDataPtr(db.Name),
			"catalogId":   llx.StringDataPtr(db.CatalogId),
			"description": llx.StringDataPtr(db.Description),
			"locationUri": llx.StringDataPtr(db.LocationUri),
			"parameters":  llx.MapData(params, types.String),
			"createdAt":   llx.TimeDataPtr(db.CreateTime),
			"region":      llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsGlueDatabase), nil
}

func (a *mqlAwsGlueDatabase) tables() ([]any, error) {
	dbName := a.Name.Data
	region := a.Region.Data
	catalogId := a.CatalogId.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Glue(region)
	ctx := context.Background()
	res := []any{}

	paginator := glue.NewGetTablesPaginator(svc, &glue.GetTablesInput{
		DatabaseName: &dbName,
		CatalogId:    &catalogId,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, table := range page.TableList {
			mqlTable, err := newMqlAwsGlueDatabaseTable(a.MqlRuntime, region, table)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlTable)
		}
	}
	return res, nil
}

func newMqlAwsGlueDatabaseTable(runtime *plugin.Runtime, region string, table glue_types.Table) (*mqlAwsGlueDatabaseTable, error) {
	id := fmt.Sprintf("glue/table/%s/%s/%s/%s", region, convert.ToValue(table.CatalogId), convert.ToValue(table.DatabaseName), convert.ToValue(table.Name))

	storageDescriptor, err := convert.JsonToDict(table.StorageDescriptor)
	if err != nil {
		return nil, err
	}

	var params map[string]any
	if table.Parameters != nil {
		params = toInterfaceMap(table.Parameters)
	}

	resource, err := CreateResource(runtime, "aws.glue.database.table",
		map[string]*llx.RawData{
			"__id":              llx.StringData(id),
			"name":              llx.StringDataPtr(table.Name),
			"databaseName":      llx.StringDataPtr(table.DatabaseName),
			"catalogId":         llx.StringDataPtr(table.CatalogId),
			"description":       llx.StringDataPtr(table.Description),
			"owner":             llx.StringDataPtr(table.Owner),
			"createdAt":         llx.TimeDataPtr(table.CreateTime),
			"updatedAt":         llx.TimeDataPtr(table.UpdateTime),
			"lastAccessedAt":    llx.TimeDataPtr(table.LastAccessTime),
			"retention":         llx.IntData(int64(table.Retention)),
			"storageDescriptor": llx.DictData(storageDescriptor),
			"tableType":         llx.StringDataPtr(table.TableType),
			"parameters":        llx.MapData(params, types.String),
			"createdBy":         llx.StringDataPtr(table.CreatedBy),
			"region":            llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsGlueDatabaseTable), nil
}

func (a *mqlAwsGlue) catalogEncryptionSettings() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getCatalogEncryptionSettings(conn), 5)
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

func (a *mqlAwsGlue) getCatalogEncryptionSettings(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("glue>getCatalogEncryptionSettings>calling aws with region %s", region)

			svc := conn.Glue(region)
			ctx := context.Background()

			resp, err := svc.GetDataCatalogEncryptionSettings(ctx, &glue.GetDataCatalogEncryptionSettingsInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS API")
					return []any{}, nil
				}
				return nil, err
			}

			if resp.DataCatalogEncryptionSettings == nil {
				return jobpool.JobResult([]any{}), nil
			}
			settingsDict, err := convert.JsonToDict(resp.DataCatalogEncryptionSettings)
			if err != nil {
				return nil, err
			}
			if settingsDict == nil {
				settingsDict = map[string]any{}
			}

			// Include region in the settings dict for identification
			settingsDict["region"] = region

			return jobpool.JobResult([]any{settingsDict}), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsGlue) workflows() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getWorkflows(conn), 5)
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

func (a *mqlAwsGlue) getWorkflows(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("glue>getWorkflows>calling aws with region %s", region)

			svc := conn.Glue(region)
			ctx := context.Background()
			res := []any{}

			// ListWorkflows returns only names, so we need to batch-get the details
			paginator := glue.NewListWorkflowsPaginator(svc, &glue.ListWorkflowsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				if len(page.Workflows) == 0 {
					continue
				}

				// BatchGetWorkflows to get full details
				batchResp, err := svc.BatchGetWorkflows(ctx, &glue.BatchGetWorkflowsInput{
					Names: page.Workflows,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						return res, nil
					}
					return nil, err
				}

				for _, wf := range batchResp.Workflows {
					mqlWf, err := newMqlAwsGlueWorkflow(a.MqlRuntime, region, conn.AccountId(), wf)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlWf)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlAwsGlueWorkflow(runtime *plugin.Runtime, region string, accountID string, wf glue_types.Workflow) (*mqlAwsGlueWorkflow, error) {
	id := fmt.Sprintf("arn:aws:glue:%s:%s:workflow/%s", region, accountID, convert.ToValue(wf.Name))

	var maxConcurrentRuns int64
	if wf.MaxConcurrentRuns != nil {
		maxConcurrentRuns = int64(*wf.MaxConcurrentRuns)
	}

	resource, err := CreateResource(runtime, "aws.glue.workflow",
		map[string]*llx.RawData{
			"__id":                 llx.StringData(id),
			"name":                 llx.StringDataPtr(wf.Name),
			"region":               llx.StringData(region),
			"description":          llx.StringDataPtr(wf.Description),
			"defaultRunProperties": llx.MapData(toInterfaceMap(wf.DefaultRunProperties), types.String),
			"maxConcurrentRuns":    llx.IntData(maxConcurrentRuns),
			"createdAt":            llx.TimeDataPtr(wf.CreatedOn),
			"updatedAt":            llx.TimeDataPtr(wf.LastModifiedOn),
		})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAwsGlueWorkflow), nil
}

func (a *mqlAwsGlueWorkflow) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Glue(a.Region.Data)
	ctx := context.Background()

	arn := fmt.Sprintf("arn:aws:glue:%s:%s:workflow/%s", a.Region.Data, conn.AccountId(), a.Name.Data)
	resp, err := svc.GetTags(ctx, &glue.GetTagsInput{
		ResourceArn: &arn,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			return nil, nil
		}
		return nil, err
	}
	return toInterfaceMap(resp.Tags), nil
}
