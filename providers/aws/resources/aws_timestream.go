// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/timestreamwrite"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/cnquery/v11/providers/aws/connection"
	"golang.org/x/exp/slices"
)

// AWS TimeStream LiveAnalytics is not available in all regions
var timeStreamLiveRegions = []string{
	"us-gov-west-1",
	// "ap-south-1", // only InfluxDB is available
	"ap-northeast-1",
	// "ap-southeast-1", // only InfluxDB is available
	"ap-southeast-2",
	"eu-central-1",
	"eu-west-1",
	// "eu-north-1", // only InfluxDB is available
	"us-east-1",
	"us-east-2",
	"us-west-2",
}

func (a *mqlAwsTimestreamLiveanalytics) id() (string, error) {
	return "aws.timestream.liveanalytics", nil
}

func (a *mqlAwsTimestreamLiveanalytics) databases() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getDatabases(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		if poolOfJobs.Jobs[i].Result != nil {
			res = append(res, poolOfJobs.Jobs[i].Result.([]interface{})...)
		}
	}

	return res, nil
}

func (a *mqlAwsTimestreamLiveanalytics) getDatabases(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		if !slices.Contains(timeStreamLiveRegions, region) {
			log.Debug().Str("region", region).Msg("skipping region since timestream is not available in this region")
			continue
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("timestream>getDatabases>calling aws with region %s", region)

			svc := conn.TimestreamLiveAnalytics(region)
			ctx := context.Background()
			res := []interface{}{}

			params := &timestreamwrite.ListDatabasesInput{}
			paginator := timestreamwrite.NewListDatabasesPaginator(svc, params)
			for paginator.HasMorePages() {
				resp, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				if len(resp.Databases) == 0 {
					return nil, nil
				}
				for _, database := range resp.Databases {
					mqlCluster, err := CreateResource(a.MqlRuntime, "aws.timestream.liveanalytics.database",
						map[string]*llx.RawData{
							"__id":       llx.StringDataPtr(database.Arn),
							"arn":        llx.StringDataPtr(database.Arn),
							"name":       llx.StringDataPtr(database.DatabaseName),
							"kmsKeyId":   llx.StringDataPtr(database.KmsKeyId),
							"region":     llx.StringData(region),
							"createdAt":  llx.TimeDataPtr(database.CreationTime),
							"updatedAt":  llx.TimeDataPtr(database.LastUpdatedTime),
							"tableCount": llx.IntData(database.TableCount),
						})
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

func (a *mqlAwsTimestreamLiveanalytics) tables() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []interface{}{}
	poolOfJobs := jobpool.CreatePool(a.getTables(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for _, job := range poolOfJobs.Jobs {
		if job.Result != nil {
			res = append(res, job.Result.([]interface{})...)
		}
	}

	return res, nil
}

func (a *mqlAwsTimestreamLiveanalytics) getTables(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		if !slices.Contains(timeStreamLiveRegions, region) {
			log.Debug().Str("region", region).Msg("skipping region since timestream is not available in this region")
			continue
		}
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("timestream>getTables>calling aws with region %s", region)

			svc := conn.TimestreamLiveAnalytics(region)
			ctx := context.Background()
			res := []interface{}{}

			params := &timestreamwrite.ListTablesInput{}
			paginator := timestreamwrite.NewListTablesPaginator(svc, params)
			for paginator.HasMorePages() {
				resp, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				if len(resp.Tables) == 0 {
					return nil, nil
				}
				for _, table := range resp.Tables {
					magneticStoreProperties, _ := convert.JsonToDictSlice(table.MagneticStoreWriteProperties)
					retentionProperties, _ := convert.JsonToDictSlice(table.RetentionProperties)

					mqlCluster, err := CreateResource(a.MqlRuntime, "aws.timestream.liveanalytics.table",
						map[string]*llx.RawData{
							"__id":                         llx.StringDataPtr(table.Arn),
							"arn":                          llx.StringDataPtr(table.Arn),
							"databaseName":                 llx.StringDataPtr(table.DatabaseName),
							"name":                         llx.StringDataPtr(table.TableName),
							"createdAt":                    llx.TimeDataPtr(table.CreationTime),
							"updatedAt":                    llx.TimeDataPtr(table.LastUpdatedTime),
							"magneticStoreWriteProperties": llx.DictData(magneticStoreProperties),
							"retentionProperties":          llx.DictData(retentionProperties),
							"region":                       llx.StringData(region),
						})
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
