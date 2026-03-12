// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/timestreaminfluxdb"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
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
							"engineType":       llx.StringData(string(cluster.EngineType)),
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

// Instance detail caching

type mqlAwsTimestreamInfluxdbInstanceInternal struct {
	securityGroupIdHandler
	detailOnce sync.Once
	detailErr  error
	detail     *timestreaminfluxdb.GetDbInstanceOutput
}

func (a *mqlAwsTimestreamInfluxdbInstance) fetchDetail() (*timestreaminfluxdb.GetDbInstanceOutput, error) {
	a.detailOnce.Do(func() {
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		svc := conn.TimestreamInfluxDB(a.Region.Data)
		ctx := context.Background()
		id := a.Id.Data
		a.detail, a.detailErr = svc.GetDbInstance(ctx, &timestreaminfluxdb.GetDbInstanceInput{
			Identifier: &id,
		})
		if a.detailErr == nil && a.detail != nil {
			accountID := conn.AccountId()
			region := a.Region.Data
			sgs := make([]string, 0, len(a.detail.VpcSecurityGroupIds))
			for _, sgID := range a.detail.VpcSecurityGroupIds {
				sgs = append(sgs, NewSecurityGroupArn(region, accountID, sgID))
			}
			a.setSecurityGroupArns(sgs)
		}
	})
	return a.detail, a.detailErr
}

func (a *mqlAwsTimestreamInfluxdbInstance) publiclyAccessible() (bool, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return false, err
	}
	return convert.ToValue(detail.PubliclyAccessible), nil
}

func (a *mqlAwsTimestreamInfluxdbInstance) vpcSubnets() ([]any, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	accountID := conn.AccountId()
	res := make([]any, 0, len(detail.VpcSubnetIds))
	for _, subnetID := range detail.VpcSubnetIds {
		sub, err := NewResource(a.MqlRuntime, ResourceAwsVpcSubnet, map[string]*llx.RawData{
			"arn": llx.StringData(fmt.Sprintf(subnetArnPattern, region, accountID, subnetID)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, sub)
	}
	return res, nil
}

func (a *mqlAwsTimestreamInfluxdbInstance) securityGroups() ([]any, error) {
	if _, err := a.fetchDetail(); err != nil {
		return nil, err
	}
	return a.newSecurityGroupResources(a.MqlRuntime)
}

func (a *mqlAwsTimestreamInfluxdbInstance) influxAuthParametersSecretArn() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	return convert.ToValue(detail.InfluxAuthParametersSecretArn), nil
}

func (a *mqlAwsTimestreamInfluxdbInstance) logDeliveryEnabled() (bool, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return false, err
	}
	if detail.LogDeliveryConfiguration != nil && detail.LogDeliveryConfiguration.S3Configuration != nil {
		return convert.ToValue(detail.LogDeliveryConfiguration.S3Configuration.Enabled), nil
	}
	return false, nil
}

func (a *mqlAwsTimestreamInfluxdbInstance) logDeliveryS3Bucket() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail.LogDeliveryConfiguration != nil && detail.LogDeliveryConfiguration.S3Configuration != nil {
		return convert.ToValue(detail.LogDeliveryConfiguration.S3Configuration.BucketName), nil
	}
	return "", nil
}

func (a *mqlAwsTimestreamInfluxdbInstance) dbParameterGroupIdentifier() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	return convert.ToValue(detail.DbParameterGroupIdentifier), nil
}

func (a *mqlAwsTimestreamInfluxdbInstance) availabilityZone() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	return convert.ToValue(detail.AvailabilityZone), nil
}

func (a *mqlAwsTimestreamInfluxdbInstance) tags() (map[string]any, error) {
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
	tags := make(map[string]any)
	for k, v := range resp.Tags {
		tags[k] = v
	}
	return tags, nil
}

// Cluster detail caching

type mqlAwsTimestreamInfluxdbClusterInternal struct {
	securityGroupIdHandler
	detailOnce sync.Once
	detailErr  error
	detail     *timestreaminfluxdb.GetDbClusterOutput
}

func (a *mqlAwsTimestreamInfluxdbCluster) fetchDetail() (*timestreaminfluxdb.GetDbClusterOutput, error) {
	a.detailOnce.Do(func() {
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		svc := conn.TimestreamInfluxDB(a.Region.Data)
		ctx := context.Background()
		id := a.Id.Data
		a.detail, a.detailErr = svc.GetDbCluster(ctx, &timestreaminfluxdb.GetDbClusterInput{
			DbClusterId: &id,
		})
		if a.detailErr == nil && a.detail != nil {
			accountID := conn.AccountId()
			region := a.Region.Data
			sgs := make([]string, 0, len(a.detail.VpcSecurityGroupIds))
			for _, sgID := range a.detail.VpcSecurityGroupIds {
				sgs = append(sgs, NewSecurityGroupArn(region, accountID, sgID))
			}
			a.setSecurityGroupArns(sgs)
		}
	})
	return a.detail, a.detailErr
}

func (a *mqlAwsTimestreamInfluxdbCluster) publiclyAccessible() (bool, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return false, err
	}
	return convert.ToValue(detail.PubliclyAccessible), nil
}

func (a *mqlAwsTimestreamInfluxdbCluster) vpcSubnets() ([]any, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	region := a.Region.Data
	accountID := conn.AccountId()
	res := make([]any, 0, len(detail.VpcSubnetIds))
	for _, subnetID := range detail.VpcSubnetIds {
		sub, err := NewResource(a.MqlRuntime, ResourceAwsVpcSubnet, map[string]*llx.RawData{
			"arn": llx.StringData(fmt.Sprintf(subnetArnPattern, region, accountID, subnetID)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, sub)
	}
	return res, nil
}

func (a *mqlAwsTimestreamInfluxdbCluster) securityGroups() ([]any, error) {
	if _, err := a.fetchDetail(); err != nil {
		return nil, err
	}
	return a.newSecurityGroupResources(a.MqlRuntime)
}

func (a *mqlAwsTimestreamInfluxdbCluster) influxAuthParametersSecretArn() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	return convert.ToValue(detail.InfluxAuthParametersSecretArn), nil
}

func (a *mqlAwsTimestreamInfluxdbCluster) logDeliveryEnabled() (bool, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return false, err
	}
	if detail.LogDeliveryConfiguration != nil && detail.LogDeliveryConfiguration.S3Configuration != nil {
		return convert.ToValue(detail.LogDeliveryConfiguration.S3Configuration.Enabled), nil
	}
	return false, nil
}

func (a *mqlAwsTimestreamInfluxdbCluster) logDeliveryS3Bucket() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail.LogDeliveryConfiguration != nil && detail.LogDeliveryConfiguration.S3Configuration != nil {
		return convert.ToValue(detail.LogDeliveryConfiguration.S3Configuration.BucketName), nil
	}
	return "", nil
}

func (a *mqlAwsTimestreamInfluxdbCluster) dbParameterGroupIdentifier() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	return convert.ToValue(detail.DbParameterGroupIdentifier), nil
}

func (a *mqlAwsTimestreamInfluxdbCluster) failoverMode() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	return string(detail.FailoverMode), nil
}

func (a *mqlAwsTimestreamInfluxdbCluster) tags() (map[string]any, error) {
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
	tags := make(map[string]any)
	for k, v := range resp.Tags {
		tags[k] = v
	}
	return tags, nil
}
