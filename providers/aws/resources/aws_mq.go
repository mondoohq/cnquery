// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/mq"
	mq_types "github.com/aws/aws-sdk-go-v2/service/mq/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsMq) id() (string, error) {
	return "aws.mq", nil
}

func (a *mqlAwsMq) brokers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getBrokers(conn), 5)
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

func (a *mqlAwsMq) getBrokers(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("mq>getBrokers>calling aws with region %s", region)

			svc := conn.Mq(region)
			ctx := context.Background()
			res := []any{}

			var nextToken *string
			for {
				page, err := svc.ListBrokers(ctx, &mq.ListBrokersInput{
					NextToken: nextToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("MQ service not available in region")
						return res, nil
					}
					return nil, err
				}
				for _, broker := range page.BrokerSummaries {
					mqlBroker, err := newMqlAwsMqBroker(a.MqlRuntime, region, conn.AccountId(), broker)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlBroker)
				}
				if page.NextToken == nil {
					break
				}
				nextToken = page.NextToken
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlAwsMqBroker(runtime *plugin.Runtime, region string, accountID string, broker mq_types.BrokerSummary) (*mqlAwsMqBroker, error) {
	var createdAt *llx.RawData
	if broker.Created != nil {
		createdAt = llx.TimeData(*broker.Created)
	} else {
		createdAt = llx.NilData
	}

	resource, err := CreateResource(runtime, "aws.mq.broker",
		map[string]*llx.RawData{
			"__id":             llx.StringDataPtr(broker.BrokerArn),
			"arn":              llx.StringDataPtr(broker.BrokerArn),
			"brokerId":         llx.StringDataPtr(broker.BrokerId),
			"name":             llx.StringDataPtr(broker.BrokerName),
			"state":            llx.StringData(string(broker.BrokerState)),
			"engineType":       llx.StringData(string(broker.EngineType)),
			"deploymentMode":   llx.StringData(string(broker.DeploymentMode)),
			"hostInstanceType": llx.StringDataPtr(broker.HostInstanceType),
			"region":           llx.StringData(region),
			"createdAt":        createdAt,
		})
	if err != nil {
		return nil, err
	}

	mqlBroker := resource.(*mqlAwsMqBroker)
	mqlBroker.region = region
	mqlBroker.accountID = accountID
	if broker.BrokerId != nil {
		mqlBroker.cacheBrokerId = *broker.BrokerId
	}
	return mqlBroker, nil
}

type mqlAwsMqBrokerInternal struct {
	securityGroupIdHandler
	cacheKmsKeyId  *string
	cacheSubnetIds []string
	cacheTags      map[string]any
	region         string
	accountID      string
	cacheBrokerId  string
	fetched        bool
	lock           sync.Mutex
}

// fetchDetails calls DescribeBroker to populate all lazy-loaded fields.
// Most security-relevant fields are only available from the describe call.
func (a *mqlAwsMqBroker) fetchDetails() error {
	if a.fetched {
		return nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Mq(a.region)
	ctx := context.Background()

	resp, err := svc.DescribeBroker(ctx, &mq.DescribeBrokerInput{
		BrokerId: &a.cacheBrokerId,
	})
	if err != nil {
		if Is400AccessDeniedError(err) {
			log.Warn().Str("brokerId", a.cacheBrokerId).Msg("access denied describing MQ broker")
			a.fetched = true
			return nil
		}
		return err
	}

	// Cache security groups.
	sgs := []string{}
	for _, sg := range resp.SecurityGroups {
		sgs = append(sgs, NewSecurityGroupArn(a.region, a.accountID, sg))
	}
	a.setSecurityGroupArns(sgs)

	// Cache subnets.
	a.cacheSubnetIds = resp.SubnetIds

	// Cache KMS key.
	if resp.EncryptionOptions != nil {
		a.cacheKmsKeyId = resp.EncryptionOptions.KmsKeyId
	}

	// Populate all describe-only fields.
	if resp.EngineVersion != nil {
		a.EngineVersion = plugin.TValue[string]{Data: *resp.EngineVersion, State: plugin.StateIsSet}
	}

	if resp.PubliclyAccessible != nil {
		a.PubliclyAccessible = plugin.TValue[bool]{Data: *resp.PubliclyAccessible, State: plugin.StateIsSet}
	}

	a.AuthenticationStrategy = plugin.TValue[string]{Data: string(resp.AuthenticationStrategy), State: plugin.StateIsSet}

	useAwsOwnedKey := true
	if resp.EncryptionOptions != nil && resp.EncryptionOptions.UseAwsOwnedKey != nil {
		useAwsOwnedKey = *resp.EncryptionOptions.UseAwsOwnedKey
	}
	a.UseAwsOwnedKey = plugin.TValue[bool]{Data: useAwsOwnedKey, State: plugin.StateIsSet}

	generalLogs := false
	auditLogs := false
	if resp.Logs != nil {
		if resp.Logs.General != nil {
			generalLogs = *resp.Logs.General
		}
		if resp.Logs.Audit != nil {
			auditLogs = *resp.Logs.Audit
		}
	}
	a.GeneralLogsEnabled = plugin.TValue[bool]{Data: generalLogs, State: plugin.StateIsSet}
	a.AuditLogsEnabled = plugin.TValue[bool]{Data: auditLogs, State: plugin.StateIsSet}

	autoUpgrade := false
	if resp.AutoMinorVersionUpgrade != nil {
		autoUpgrade = *resp.AutoMinorVersionUpgrade
	}
	a.AutoMinorVersionUpgrade = plugin.TValue[bool]{Data: autoUpgrade, State: plugin.StateIsSet}

	a.StorageType = plugin.TValue[string]{Data: string(resp.StorageType), State: plugin.StateIsSet}

	// Cache tags from the describe response.
	cacheTags := make(map[string]any)
	for k, v := range resp.Tags {
		cacheTags[k] = v
	}
	a.cacheTags = cacheTags

	a.fetched = true
	return nil
}

func (a *mqlAwsMqBroker) engineVersion() (string, error) {
	return "", a.fetchDetails()
}

func (a *mqlAwsMqBroker) publiclyAccessible() (bool, error) {
	return false, a.fetchDetails()
}

func (a *mqlAwsMqBroker) authenticationStrategy() (string, error) {
	return "", a.fetchDetails()
}

func (a *mqlAwsMqBroker) useAwsOwnedKey() (bool, error) {
	return false, a.fetchDetails()
}

func (a *mqlAwsMqBroker) kmsKey() (*mqlAwsKmsKey, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
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

func (a *mqlAwsMqBroker) generalLogsEnabled() (bool, error) {
	return false, a.fetchDetails()
}

func (a *mqlAwsMqBroker) auditLogsEnabled() (bool, error) {
	return false, a.fetchDetails()
}

func (a *mqlAwsMqBroker) autoMinorVersionUpgrade() (bool, error) {
	return false, a.fetchDetails()
}

func (a *mqlAwsMqBroker) storageType() (string, error) {
	return "", a.fetchDetails()
}

func (a *mqlAwsMqBroker) tags() (map[string]any, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	return a.cacheTags, nil
}

func (a *mqlAwsMqBroker) securityGroups() ([]any, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	return a.newSecurityGroupResources(a.MqlRuntime)
}

func (a *mqlAwsMqBroker) subnets() ([]any, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
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
