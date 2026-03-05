// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/timestreaminfluxdb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsTimestreamInfluxdb) id() (string, error) {
	return "aws.timestream.influxdb", nil
}

func (a *mqlAwsTimestreamInfluxdb) instances() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getInstances(conn), 5)
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

func (a *mqlAwsTimestreamInfluxdb) getInstances(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("timestream.influxdb>getInstances>calling aws with region %s", region)

			svc := conn.TimestreamInfluxDB(region)
			ctx := context.Background()
			res := []any{}

			paginator := timestreaminfluxdb.NewListDbInstancesPaginator(svc, &timestreaminfluxdb.ListDbInstancesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Debug().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("timestream influxdb service not available in region")
						return res, nil
					}
					return nil, err
				}
				for _, instance := range page.Items {
					mqlInstance, err := CreateResource(a.MqlRuntime, "aws.timestream.influxdb.instance",
						map[string]*llx.RawData{
							"__id":             llx.StringDataPtr(instance.Arn),
							"arn":              llx.StringDataPtr(instance.Arn),
							"id":               llx.StringDataPtr(instance.Id),
							"name":             llx.StringDataPtr(instance.Name),
							"allocatedStorage": llx.IntDataDefault(instance.AllocatedStorage, 0),
							"dbInstanceType":   llx.StringData(string(instance.DbInstanceType)),
							"dbStorageType":    llx.StringData(string(instance.DbStorageType)),
							"deploymentType":   llx.StringData(string(instance.DeploymentType)),
							"endpoint":         llx.StringDataPtr(instance.Endpoint),
							"networkType":      llx.StringData(string(instance.NetworkType)),
							"port":             llx.IntDataDefault(instance.Port, 0),
							"status":           llx.StringData(string(instance.Status)),
							"region":           llx.StringData(region),
						})
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

func (a *mqlAwsTimestreamInfluxdb) clusters() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getClusters(conn), 5)
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

func (a *mqlAwsTimestreamInfluxdb) getClusters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("timestream.influxdb>getClusters>calling aws with region %s", region)

			svc := conn.TimestreamInfluxDB(region)
			ctx := context.Background()
			res := []any{}

			paginator := timestreaminfluxdb.NewListDbClustersPaginator(svc, &timestreaminfluxdb.ListDbClustersInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Debug().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("timestream influxdb service not available in region")
						return res, nil
					}
					return nil, err
				}
				for _, cluster := range page.Items {
					mqlCluster, err := CreateResource(a.MqlRuntime, "aws.timestream.influxdb.cluster",
						map[string]*llx.RawData{
							"__id":             llx.StringDataPtr(cluster.Arn),
							"arn":              llx.StringDataPtr(cluster.Arn),
							"id":               llx.StringDataPtr(cluster.Id),
							"name":             llx.StringDataPtr(cluster.Name),
							"allocatedStorage": llx.IntDataDefault(cluster.AllocatedStorage, 0),
							"dbInstanceType":   llx.StringData(string(cluster.DbInstanceType)),
							"dbStorageType":    llx.StringData(string(cluster.DbStorageType)),
							"deploymentType":   llx.StringData(string(cluster.DeploymentType)),
							"endpoint":         llx.StringDataPtr(cluster.Endpoint),
							"readerEndpoint":   llx.StringDataPtr(cluster.ReaderEndpoint),
							"networkType":      llx.StringData(string(cluster.NetworkType)),
							"port":             llx.IntDataDefault(cluster.Port, 0),
							"status":           llx.StringData(string(cluster.Status)),
							"region":           llx.StringData(region),
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

func (a *mqlAwsTimestreamInfluxdbCluster) tags() (map[string]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.TimestreamInfluxDB(a.Region.Data)
	ctx := context.Background()
	arn := a.Arn.Data

	resp, err := svc.ListTagsForResource(ctx, &timestreaminfluxdb.ListTagsForResourceInput{
		ResourceArn: &arn,
	})
	if err != nil {
		return nil, err
	}
	tags := make(map[string]interface{})
	for k, v := range resp.Tags {
		tags[k] = v
	}
	return tags, nil
}

func (a *mqlAwsTimestreamInfluxdbInstance) tags() (map[string]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.TimestreamInfluxDB(a.Region.Data)
	ctx := context.Background()
	arn := a.Arn.Data

	resp, err := svc.ListTagsForResource(ctx, &timestreaminfluxdb.ListTagsForResourceInput{
		ResourceArn: &arn,
	})
	if err != nil {
		return nil, err
	}
	tags := make(map[string]interface{})
	for k, v := range resp.Tags {
		tags[k] = v
	}
	return tags, nil
}
