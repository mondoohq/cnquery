// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/kafka"
	kafka_types "github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsMsk) id() (string, error) {
	return "aws.msk", nil
}

func (a *mqlAwsMsk) clusters() ([]any, error) {
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

func (a *mqlAwsMsk) getClusters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("msk>getClusters>calling aws with region %s", region)

			svc := conn.Kafka(region)
			ctx := context.Background()
			res := []any{}

			paginator := kafka.NewListClustersV2Paginator(svc, &kafka.ListClustersV2Input{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("MSK service not available in region")
						return res, nil
					}
					return nil, err
				}
				for _, cluster := range page.ClusterInfoList {
					mqlCluster, err := newMqlAwsMskCluster(a.MqlRuntime, region, conn.AccountId(), cluster)
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

func newMqlAwsMskCluster(runtime *plugin.Runtime, region string, accountID string, cluster kafka_types.Cluster) (*mqlAwsMskCluster, error) {
	tags := make(map[string]any)
	for k, v := range cluster.Tags {
		tags[k] = v
	}

	var createdAt *llx.RawData
	if cluster.CreationTime != nil {
		createdAt = llx.TimeData(*cluster.CreationTime)
	} else {
		createdAt = llx.NilData
	}

	clusterType := ""
	if cluster.ClusterType != "" {
		clusterType = string(cluster.ClusterType)
	}

	resource, err := CreateResource(runtime, "aws.msk.cluster",
		map[string]*llx.RawData{
			"__id":           llx.StringDataPtr(cluster.ClusterArn),
			"arn":            llx.StringDataPtr(cluster.ClusterArn),
			"name":           llx.StringDataPtr(cluster.ClusterName),
			"state":          llx.StringData(string(cluster.State)),
			"clusterType":    llx.StringData(clusterType),
			"region":         llx.StringData(region),
			"currentVersion": llx.StringDataPtr(cluster.CurrentVersion),
			"createdAt":      createdAt,
			"tags":           llx.MapData(tags, types.String),
		})
	if err != nil {
		return nil, err
	}

	mqlCluster := resource.(*mqlAwsMskCluster)
	mqlCluster.region = region
	mqlCluster.accountID = accountID

	// Cache provisioned cluster details for lazy-loaded fields.
	if cluster.Provisioned != nil {
		p := cluster.Provisioned
		mqlCluster.provisioned = p

		if p.BrokerNodeGroupInfo != nil {
			bni := p.BrokerNodeGroupInfo
			// Cache security group ARNs.
			sgs := []string{}
			for _, sg := range bni.SecurityGroups {
				sgs = append(sgs, NewSecurityGroupArn(region, accountID, sg))
			}
			mqlCluster.setSecurityGroupArns(sgs)

			// Cache subnet IDs.
			mqlCluster.cacheSubnetIds = bni.ClientSubnets
		}

		if p.EncryptionInfo != nil {
			if p.EncryptionInfo.EncryptionAtRest != nil {
				mqlCluster.cacheKmsKeyId = p.EncryptionInfo.EncryptionAtRest.DataVolumeKMSKeyId
			}
		}
	}

	return mqlCluster, nil
}

type mqlAwsMskClusterInternal struct {
	securityGroupIdHandler
	cacheKmsKeyId  *string
	cacheSubnetIds []string
	region         string
	accountID      string
	provisioned    *kafka_types.Provisioned
}

func (a *mqlAwsMskCluster) kafkaVersion() (string, error) {
	if a.provisioned == nil {
		a.KafkaVersion.State = plugin.StateIsNull | plugin.StateIsSet
		return "", nil
	}
	if a.provisioned.CurrentBrokerSoftwareInfo != nil && a.provisioned.CurrentBrokerSoftwareInfo.KafkaVersion != nil {
		return *a.provisioned.CurrentBrokerSoftwareInfo.KafkaVersion, nil
	}
	a.KafkaVersion.State = plugin.StateIsNull | plugin.StateIsSet
	return "", nil
}

func (a *mqlAwsMskCluster) numberOfBrokerNodes() (int64, error) {
	if a.provisioned == nil {
		a.NumberOfBrokerNodes.State = plugin.StateIsNull | plugin.StateIsSet
		return 0, nil
	}
	if a.provisioned.NumberOfBrokerNodes != nil {
		return int64(*a.provisioned.NumberOfBrokerNodes), nil
	}
	a.NumberOfBrokerNodes.State = plugin.StateIsNull | plugin.StateIsSet
	return 0, nil
}

func (a *mqlAwsMskCluster) brokerInstanceType() (string, error) {
	if a.provisioned == nil {
		a.BrokerInstanceType.State = plugin.StateIsNull | plugin.StateIsSet
		return "", nil
	}
	if a.provisioned.BrokerNodeGroupInfo != nil && a.provisioned.BrokerNodeGroupInfo.InstanceType != nil {
		return *a.provisioned.BrokerNodeGroupInfo.InstanceType, nil
	}
	a.BrokerInstanceType.State = plugin.StateIsNull | plugin.StateIsSet
	return "", nil
}

func (a *mqlAwsMskCluster) encryptionInTransitClientBroker() (string, error) {
	if a.provisioned == nil {
		a.EncryptionInTransitClientBroker.State = plugin.StateIsNull | plugin.StateIsSet
		return "", nil
	}
	if a.provisioned.EncryptionInfo != nil && a.provisioned.EncryptionInfo.EncryptionInTransit != nil {
		return string(a.provisioned.EncryptionInfo.EncryptionInTransit.ClientBroker), nil
	}
	a.EncryptionInTransitClientBroker.State = plugin.StateIsNull | plugin.StateIsSet
	return "", nil
}

func (a *mqlAwsMskCluster) encryptionInTransitInCluster() (bool, error) {
	if a.provisioned == nil {
		a.EncryptionInTransitInCluster.State = plugin.StateIsNull | plugin.StateIsSet
		return false, nil
	}
	if a.provisioned.EncryptionInfo != nil && a.provisioned.EncryptionInfo.EncryptionInTransit != nil {
		if a.provisioned.EncryptionInfo.EncryptionInTransit.InCluster != nil {
			return *a.provisioned.EncryptionInfo.EncryptionInTransit.InCluster, nil
		}
	}
	// Default to true as AWS enables this by default for provisioned clusters.
	return true, nil
}

func (a *mqlAwsMskCluster) kmsKey() (*mqlAwsKmsKey, error) {
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

func (a *mqlAwsMskCluster) iamAuthEnabled() (bool, error) {
	if a.provisioned == nil {
		a.IamAuthEnabled.State = plugin.StateIsNull | plugin.StateIsSet
		return false, nil
	}
	if a.provisioned.ClientAuthentication != nil {
		if a.provisioned.ClientAuthentication.Sasl != nil && a.provisioned.ClientAuthentication.Sasl.Iam != nil {
			if a.provisioned.ClientAuthentication.Sasl.Iam.Enabled != nil {
				return *a.provisioned.ClientAuthentication.Sasl.Iam.Enabled, nil
			}
		}
	}
	return false, nil
}

func (a *mqlAwsMskCluster) scramAuthEnabled() (bool, error) {
	if a.provisioned == nil {
		a.ScramAuthEnabled.State = plugin.StateIsNull | plugin.StateIsSet
		return false, nil
	}
	if a.provisioned.ClientAuthentication != nil {
		if a.provisioned.ClientAuthentication.Sasl != nil && a.provisioned.ClientAuthentication.Sasl.Scram != nil {
			if a.provisioned.ClientAuthentication.Sasl.Scram.Enabled != nil {
				return *a.provisioned.ClientAuthentication.Sasl.Scram.Enabled, nil
			}
		}
	}
	return false, nil
}

func (a *mqlAwsMskCluster) tlsAuthEnabled() (bool, error) {
	if a.provisioned == nil {
		a.TlsAuthEnabled.State = plugin.StateIsNull | plugin.StateIsSet
		return false, nil
	}
	if a.provisioned.ClientAuthentication != nil {
		if a.provisioned.ClientAuthentication.Tls != nil {
			if a.provisioned.ClientAuthentication.Tls.Enabled != nil {
				return *a.provisioned.ClientAuthentication.Tls.Enabled, nil
			}
		}
	}
	return false, nil
}

func (a *mqlAwsMskCluster) publicAccess() (bool, error) {
	if a.provisioned == nil {
		a.PublicAccess.State = plugin.StateIsNull | plugin.StateIsSet
		return false, nil
	}
	if a.provisioned.BrokerNodeGroupInfo != nil {
		ci := a.provisioned.BrokerNodeGroupInfo.ConnectivityInfo
		if ci != nil && ci.PublicAccess != nil && ci.PublicAccess.Type != nil {
			return *ci.PublicAccess.Type != "DISABLED", nil
		}
	}
	return false, nil
}

func (a *mqlAwsMskCluster) cloudwatchLogsEnabled() (bool, error) {
	if a.provisioned == nil {
		a.CloudwatchLogsEnabled.State = plugin.StateIsNull | plugin.StateIsSet
		return false, nil
	}
	if a.provisioned.LoggingInfo != nil {
		bl := a.provisioned.LoggingInfo.BrokerLogs
		if bl.CloudWatchLogs != nil && bl.CloudWatchLogs.Enabled != nil {
			return *bl.CloudWatchLogs.Enabled, nil
		}
	}
	return false, nil
}

func (a *mqlAwsMskCluster) cloudwatchLogsGroup() (string, error) {
	if a.provisioned == nil {
		a.CloudwatchLogsGroup.State = plugin.StateIsNull | plugin.StateIsSet
		return "", nil
	}
	if a.provisioned.LoggingInfo != nil {
		bl := a.provisioned.LoggingInfo.BrokerLogs
		if bl.CloudWatchLogs != nil && bl.CloudWatchLogs.LogGroup != nil {
			return *bl.CloudWatchLogs.LogGroup, nil
		}
	}
	a.CloudwatchLogsGroup.State = plugin.StateIsNull | plugin.StateIsSet
	return "", nil
}

func (a *mqlAwsMskCluster) s3LogsEnabled() (bool, error) {
	if a.provisioned == nil {
		a.S3LogsEnabled.State = plugin.StateIsNull | plugin.StateIsSet
		return false, nil
	}
	if a.provisioned.LoggingInfo != nil {
		bl := a.provisioned.LoggingInfo.BrokerLogs
		if bl.S3 != nil && bl.S3.Enabled != nil {
			return *bl.S3.Enabled, nil
		}
	}
	return false, nil
}

func (a *mqlAwsMskCluster) s3LogsBucket() (string, error) {
	if a.provisioned == nil {
		a.S3LogsBucket.State = plugin.StateIsNull | plugin.StateIsSet
		return "", nil
	}
	if a.provisioned.LoggingInfo != nil {
		bl := a.provisioned.LoggingInfo.BrokerLogs
		if bl.S3 != nil && bl.S3.Bucket != nil {
			return *bl.S3.Bucket, nil
		}
	}
	a.S3LogsBucket.State = plugin.StateIsNull | plugin.StateIsSet
	return "", nil
}

func (a *mqlAwsMskCluster) firehoseLogsEnabled() (bool, error) {
	if a.provisioned == nil {
		a.FirehoseLogsEnabled.State = plugin.StateIsNull | plugin.StateIsSet
		return false, nil
	}
	if a.provisioned.LoggingInfo != nil {
		bl := a.provisioned.LoggingInfo.BrokerLogs
		if bl.Firehose != nil && bl.Firehose.Enabled != nil {
			return *bl.Firehose.Enabled, nil
		}
	}
	return false, nil
}

func (a *mqlAwsMskCluster) enhancedMonitoring() (string, error) {
	if a.provisioned == nil {
		a.EnhancedMonitoring.State = plugin.StateIsNull | plugin.StateIsSet
		return "", nil
	}
	return string(a.provisioned.EnhancedMonitoring), nil
}

func (a *mqlAwsMskCluster) jmxExporterEnabled() (bool, error) {
	if a.provisioned == nil {
		a.JmxExporterEnabled.State = plugin.StateIsNull | plugin.StateIsSet
		return false, nil
	}
	if a.provisioned.OpenMonitoring != nil && a.provisioned.OpenMonitoring.Prometheus != nil {
		if a.provisioned.OpenMonitoring.Prometheus.JmxExporter != nil && a.provisioned.OpenMonitoring.Prometheus.JmxExporter.EnabledInBroker != nil {
			return *a.provisioned.OpenMonitoring.Prometheus.JmxExporter.EnabledInBroker, nil
		}
	}
	return false, nil
}

func (a *mqlAwsMskCluster) nodeExporterEnabled() (bool, error) {
	if a.provisioned == nil {
		a.NodeExporterEnabled.State = plugin.StateIsNull | plugin.StateIsSet
		return false, nil
	}
	if a.provisioned.OpenMonitoring != nil && a.provisioned.OpenMonitoring.Prometheus != nil {
		if a.provisioned.OpenMonitoring.Prometheus.NodeExporter != nil && a.provisioned.OpenMonitoring.Prometheus.NodeExporter.EnabledInBroker != nil {
			return *a.provisioned.OpenMonitoring.Prometheus.NodeExporter.EnabledInBroker, nil
		}
	}
	return false, nil
}

func (a *mqlAwsMskCluster) securityGroups() ([]any, error) {
	if a.provisioned == nil {
		a.SecurityGroups.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return a.newSecurityGroupResources(a.MqlRuntime)
}

func (a *mqlAwsMskCluster) subnets() ([]any, error) {
	if a.provisioned == nil {
		a.Subnets.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res := []any{}
	for _, subnetId := range a.cacheSubnetIds {
		mqlSubnet, err := NewResource(a.MqlRuntime, "aws.vpc.subnet",
			map[string]*llx.RawData{
				"arn": llx.StringData(fmt.Sprintf(subnetArnPattern, a.region, a.accountID, subnetId)),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSubnet)
	}
	return res, nil
}
