// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/keyspaces"
	kstypes "github.com/aws/aws-sdk-go-v2/service/keyspaces/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"

	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsKeyspaces) id() (string, error) {
	return "aws.keyspaces", nil
}

func (a *mqlAwsKeyspaces) keyspaces() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getKeyspaces(conn), 5)
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

func (a *mqlAwsKeyspaces) getKeyspaces(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}
	for _, region := range regions {
		region := region
		f := func() (jobpool.JobResult, error) {
			svc := conn.Keyspaces(region)
			ctx := context.Background()
			res := []any{}
			paginator := keyspaces.NewListKeyspacesPaginator(svc, &keyspaces.ListKeyspacesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS Keyspaces API")
						return res, nil
					}
					return nil, err
				}
				for _, ks := range page.Keyspaces {
					replicationStrategy := ""
					if ks.ReplicationStrategy != "" {
						replicationStrategy = string(ks.ReplicationStrategy)
					}
					replicationRegions := make([]any, 0, len(ks.ReplicationRegions))
					for _, r := range ks.ReplicationRegions {
						replicationRegions = append(replicationRegions, r)
					}

					mqlKS, err := CreateResource(a.MqlRuntime, ResourceAwsKeyspacesKeyspace,
						map[string]*llx.RawData{
							"arn":                 llx.StringDataPtr(ks.ResourceArn),
							"name":                llx.StringDataPtr(ks.KeyspaceName),
							"region":              llx.StringData(region),
							"replicationStrategy": llx.StringData(replicationStrategy),
							"replicationRegions":  llx.ArrayData(replicationRegions, types.String),
						})
					if err != nil {
						return nil, err
					}
					k := mqlKS.(*mqlAwsKeyspacesKeyspace)
					k.region = region
					res = append(res, mqlKS)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsKeyspacesKeyspaceInternal struct {
	region string
}

func (a *mqlAwsKeyspacesKeyspace) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsKeyspacesKeyspace) status() (string, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Keyspaces(a.region)
	name := a.Name.Data
	resp, err := svc.GetKeyspace(context.Background(), &keyspaces.GetKeyspaceInput{KeyspaceName: &name})
	if err != nil {
		return "", err
	}
	if len(resp.ReplicationGroupStatuses) > 0 {
		return string(resp.ReplicationGroupStatuses[0].KeyspaceStatus), nil
	}
	return "ACTIVE", nil
}

func (a *mqlAwsKeyspacesKeyspace) tables() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Keyspaces(a.region)
	name := a.Name.Data
	ctx := context.Background()

	res := []any{}
	paginator := keyspaces.NewListTablesPaginator(svc, &keyspaces.ListTablesInput{KeyspaceName: &name})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, table := range page.Tables {
			mqlTable, err := CreateResource(a.MqlRuntime, ResourceAwsKeyspacesTable,
				map[string]*llx.RawData{
					"arn":          llx.StringDataPtr(table.ResourceArn),
					"name":         llx.StringDataPtr(table.TableName),
					"keyspaceName": llx.StringDataPtr(table.KeyspaceName),
					"region":       llx.StringData(a.region),
				})
			if err != nil {
				return nil, err
			}
			t := mqlTable.(*mqlAwsKeyspacesTable)
			t.region = a.region
			res = append(res, mqlTable)
		}
	}
	return res, nil
}

func (a *mqlAwsKeyspacesKeyspace) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Keyspaces(a.region)
	arn := a.Arn.Data
	return getKeyspacesTags(context.Background(), svc, &arn)
}

// ---- Tables ----

type mqlAwsKeyspacesTableInternal struct {
	cacheKmsKeyId  *string
	region         string
	fetched        bool
	fetchedDetails *keyspaces.GetTableOutput
	lock           sync.Mutex
}

func (a *mqlAwsKeyspacesTable) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsKeyspacesTable) fetchTableDetails() (*keyspaces.GetTableOutput, error) {
	if a.fetched {
		return a.fetchedDetails, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.fetchedDetails, nil
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Keyspaces(a.region)
	ksName := a.KeyspaceName.Data
	tableName := a.Name.Data
	resp, err := svc.GetTable(context.Background(), &keyspaces.GetTableInput{
		KeyspaceName: &ksName,
		TableName:    &tableName,
	})
	if err != nil {
		return nil, err
	}
	if resp.EncryptionSpecification != nil {
		a.cacheKmsKeyId = resp.EncryptionSpecification.KmsKeyIdentifier
	}
	a.fetchedDetails = resp
	a.fetched = true
	return resp, nil
}

func (a *mqlAwsKeyspacesTable) status() (string, error) {
	resp, err := a.fetchTableDetails()
	if err != nil {
		return "", err
	}
	return string(resp.Status), nil
}

func (a *mqlAwsKeyspacesTable) schemaDefinition() (map[string]any, error) {
	resp, err := a.fetchTableDetails()
	if err != nil {
		return nil, err
	}
	if resp.SchemaDefinition == nil {
		return nil, nil
	}
	return convert.JsonToDict(resp.SchemaDefinition)
}

func (a *mqlAwsKeyspacesTable) capacityMode() (string, error) {
	resp, err := a.fetchTableDetails()
	if err != nil {
		return "", err
	}
	if resp.CapacitySpecification == nil {
		return "", nil
	}
	return string(resp.CapacitySpecification.ThroughputMode), nil
}

func (a *mqlAwsKeyspacesTable) readCapacityUnits() (int64, error) {
	resp, err := a.fetchTableDetails()
	if err != nil {
		return 0, err
	}
	if resp.CapacitySpecification == nil || resp.CapacitySpecification.ReadCapacityUnits == nil {
		return 0, nil
	}
	return *resp.CapacitySpecification.ReadCapacityUnits, nil
}

func (a *mqlAwsKeyspacesTable) writeCapacityUnits() (int64, error) {
	resp, err := a.fetchTableDetails()
	if err != nil {
		return 0, err
	}
	if resp.CapacitySpecification == nil || resp.CapacitySpecification.WriteCapacityUnits == nil {
		return 0, nil
	}
	return *resp.CapacitySpecification.WriteCapacityUnits, nil
}

func (a *mqlAwsKeyspacesTable) encryptionType() (string, error) {
	resp, err := a.fetchTableDetails()
	if err != nil {
		return "", err
	}
	if resp.EncryptionSpecification == nil {
		return "", nil
	}
	return string(resp.EncryptionSpecification.Type), nil
}

func (a *mqlAwsKeyspacesTable) kmsKey() (*mqlAwsKmsKey, error) {
	_, err := a.fetchTableDetails()
	if err != nil {
		return nil, err
	}
	if a.cacheKmsKeyId == nil || *a.cacheKmsKeyId == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.kms.key", map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheKmsKeyId)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsKeyspacesTable) pointInTimeRecoveryEnabled() (bool, error) {
	resp, err := a.fetchTableDetails()
	if err != nil {
		return false, err
	}
	if resp.PointInTimeRecovery == nil {
		return false, nil
	}
	return resp.PointInTimeRecovery.Status == kstypes.PointInTimeRecoveryStatusEnabled, nil
}

func (a *mqlAwsKeyspacesTable) ttlEnabled() (bool, error) {
	resp, err := a.fetchTableDetails()
	if err != nil {
		return false, err
	}
	if resp.Ttl == nil {
		return false, nil
	}
	return resp.Ttl.Status == kstypes.TimeToLiveStatusEnabled, nil
}

func (a *mqlAwsKeyspacesTable) defaultTimeToLive() (int64, error) {
	resp, err := a.fetchTableDetails()
	if err != nil {
		return 0, err
	}
	if resp.DefaultTimeToLive == nil {
		return 0, nil
	}
	return int64(*resp.DefaultTimeToLive), nil
}

func (a *mqlAwsKeyspacesTable) autoScalingSettings() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Keyspaces(a.region)
	ksName := a.KeyspaceName.Data
	tableName := a.Name.Data
	resp, err := svc.GetTableAutoScalingSettings(context.Background(), &keyspaces.GetTableAutoScalingSettingsInput{
		KeyspaceName: &ksName,
		TableName:    &tableName,
	})
	if err != nil {
		// Auto scaling may not be configured
		if Is400AccessDeniedError(err) {
			return nil, nil
		}
		return nil, err
	}
	return convert.JsonToDict(resp)
}

func (a *mqlAwsKeyspacesTable) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Keyspaces(a.region)
	arn := a.Arn.Data
	return getKeyspacesTags(context.Background(), svc, &arn)
}

func getKeyspacesTags(ctx context.Context, svc *keyspaces.Client, arn *string) (map[string]any, error) {
	paginator := keyspaces.NewListTagsForResourcePaginator(svc, &keyspaces.ListTagsForResourceInput{ResourceArn: arn})
	tags := make(map[string]any)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				return tags, nil
			}
			return nil, err
		}
		for _, t := range page.Tags {
			if t.Key != nil && t.Value != nil {
				tags[*t.Key] = *t.Value
			}
		}
	}
	return tags, nil
}
