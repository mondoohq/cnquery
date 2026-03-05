// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/memorydb"
	memorydb_types "github.com/aws/aws-sdk-go-v2/service/memorydb/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsMemorydb) id() (string, error) {
	return "aws.memorydb", nil
}

func (a *mqlAwsMemorydb) clusters() ([]any, error) {
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

func (a *mqlAwsMemorydb) getClusters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("memorydb>getClusters>calling aws with region %s", region)

			svc := conn.Memorydb(region)
			ctx := context.Background()
			res := []any{}

			paginator := memorydb.NewDescribeClustersPaginator(svc, &memorydb.DescribeClustersInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("memorydb service not available in region")
						return res, nil
					}
					return nil, err
				}
				for _, cluster := range page.Clusters {
					mqlCluster, err := newMqlAwsMemorydbCluster(a.MqlRuntime, region, conn.AccountId(), cluster)
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

func newMqlAwsMemorydbCluster(runtime *plugin.Runtime, region string, accountID string, cluster memorydb_types.Cluster) (*mqlAwsMemorydbCluster, error) {
	sgs := []string{}
	for _, sg := range cluster.SecurityGroups {
		if sg.SecurityGroupId != nil {
			sgs = append(sgs, NewSecurityGroupArn(region, accountID, convert.ToValue(sg.SecurityGroupId)))
		}
	}

	resource, err := CreateResource(runtime, "aws.memorydb.cluster",
		map[string]*llx.RawData{
			"__id":                    llx.StringDataPtr(cluster.ARN),
			"arn":                     llx.StringDataPtr(cluster.ARN),
			"name":                    llx.StringDataPtr(cluster.Name),
			"description":             llx.StringDataPtr(cluster.Description),
			"status":                  llx.StringDataPtr(cluster.Status),
			"nodeType":                llx.StringDataPtr(cluster.NodeType),
			"engine":                  llx.StringDataPtr(cluster.Engine),
			"engineVersion":           llx.StringDataPtr(cluster.EngineVersion),
			"enginePatchVersion":      llx.StringDataPtr(cluster.EnginePatchVersion),
			"numberOfShards":          llx.IntDataDefault(cluster.NumberOfShards, 0),
			"tlsEnabled":              llx.BoolDataPtr(cluster.TLSEnabled),
			"aclName":                 llx.StringDataPtr(cluster.ACLName),
			"parameterGroupName":      llx.StringDataPtr(cluster.ParameterGroupName),
			"subnetGroupName":         llx.StringDataPtr(cluster.SubnetGroupName),
			"snapshotRetentionLimit":  llx.IntDataDefault(cluster.SnapshotRetentionLimit, 0),
			"snapshotWindow":          llx.StringDataPtr(cluster.SnapshotWindow),
			"maintenanceWindow":       llx.StringDataPtr(cluster.MaintenanceWindow),
			"autoMinorVersionUpgrade": llx.BoolDataPtr(cluster.AutoMinorVersionUpgrade),
			"region":                  llx.StringData(region),
		})
	if err != nil {
		return nil, err
	}

	mqlCluster := resource.(*mqlAwsMemorydbCluster)
	mqlCluster.cacheKmsKeyId = cluster.KmsKeyId
	mqlCluster.setSecurityGroupArns(sgs)
	return mqlCluster, nil
}

type mqlAwsMemorydbClusterInternal struct {
	securityGroupIdHandler
	cacheKmsKeyId *string
}

func (a *mqlAwsMemorydbCluster) kmsKey() (*mqlAwsKmsKey, error) {
	if a.cacheKmsKeyId == nil || *a.cacheKmsKeyId == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlKey, err := NewResource(a.MqlRuntime, ResourceAwsKmsKey,
		map[string]*llx.RawData{
			"arn": llx.StringDataPtr(a.cacheKmsKeyId),
		})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsMemorydbCluster) securityGroups() ([]any, error) {
	return a.newSecurityGroupResources(a.MqlRuntime)
}

func (a *mqlAwsMemorydbCluster) tags() (map[string]interface{}, error) {
	arn := a.Arn.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Memorydb(region)
	ctx := context.Background()

	resp, err := svc.ListTags(ctx, &memorydb.ListTagsInput{
		ResourceArn: &arn,
	})
	if err != nil {
		return nil, err
	}

	tags := make(map[string]interface{})
	for _, tag := range resp.TagList {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}
	return tags, nil
}
