// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/emr"
	emrtypes "github.com/aws/aws-sdk-go-v2/service/emr/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"

	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsEmr) id() (string, error) {
	return "aws.emr", nil
}

func (a *mqlAwsEmr) clusters() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getClusters(conn), 5)
	poolOfJobs.Run()

	// check for errors
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	// get all the results
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}
	return res, nil
}

func (a *mqlAwsEmrCluster) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsEmr) getClusters(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Emr(region)
			ctx := context.Background()

			res := []any{}

			params := &emr.ListClustersInput{}
			paginator := emr.NewListClustersPaginator(svc, params)
			for paginator.HasMorePages() {
				clusters, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, cluster := range clusters.Clusters {
					jsonStatus, err := convert.JsonToDict(cluster.Status)
					if err != nil {
						return nil, err
					}
					mqlCluster, err := CreateResource(a.MqlRuntime, "aws.emr.cluster",
						map[string]*llx.RawData{
							"arn":                     llx.StringDataPtr(cluster.ClusterArn),
							"name":                    llx.StringDataPtr(cluster.Name),
							"normalizedInstanceHours": llx.IntDataDefault(cluster.NormalizedInstanceHours, 0),
							"outpostArn":              llx.StringDataPtr(cluster.OutpostArn),
							"status":                  llx.MapData(jsonStatus, types.String),
							"id":                      llx.StringDataPtr(cluster.Id),
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

type mqlAwsEmrClusterInternal struct {
	clusterDetailsFetched      bool
	clusterDetailsLock         sync.Mutex
	cacheSecurityConfig        string
	cacheLogUri                string
	cacheTags                  map[string]any
	cacheTerminationProtected  bool
	cacheMasterPublicDnsName   string
	cacheLogEncryptionKmsKeyId *string
}

func (a *mqlAwsEmrCluster) fetchClusterDetails() error {
	if a.clusterDetailsFetched {
		return nil
	}
	a.clusterDetailsLock.Lock()
	defer a.clusterDetailsLock.Unlock()
	if a.clusterDetailsFetched {
		return nil
	}

	region, err := GetRegionFromArn(a.Arn.Data)
	if err != nil {
		return err
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Emr(region)
	ctx := context.Background()

	id := a.Id.Data
	resp, err := svc.DescribeCluster(ctx, &emr.DescribeClusterInput{ClusterId: &id})
	if err != nil {
		return err
	}

	if resp.Cluster.SecurityConfiguration != nil {
		a.cacheSecurityConfig = *resp.Cluster.SecurityConfiguration
	}
	if resp.Cluster.LogUri != nil {
		a.cacheLogUri = *resp.Cluster.LogUri
	}
	tags := make(map[string]any)
	for _, t := range resp.Cluster.Tags {
		if t.Key != nil && t.Value != nil {
			tags[*t.Key] = *t.Value
		}
	}
	a.cacheTags = tags
	if resp.Cluster.TerminationProtected != nil {
		a.cacheTerminationProtected = *resp.Cluster.TerminationProtected
	}
	if resp.Cluster.MasterPublicDnsName != nil {
		a.cacheMasterPublicDnsName = *resp.Cluster.MasterPublicDnsName
	}
	a.cacheLogEncryptionKmsKeyId = resp.Cluster.LogEncryptionKmsKeyId
	a.clusterDetailsFetched = true
	return nil
}

func (a *mqlAwsEmrCluster) tags() (map[string]any, error) {
	if err := a.fetchClusterDetails(); err != nil {
		return nil, err
	}
	return a.cacheTags, nil
}

func (a *mqlAwsEmrCluster) securityConfiguration() (string, error) {
	if err := a.fetchClusterDetails(); err != nil {
		return "", err
	}
	return a.cacheSecurityConfig, nil
}

func (a *mqlAwsEmrCluster) logUri() (string, error) {
	if err := a.fetchClusterDetails(); err != nil {
		return "", err
	}
	return a.cacheLogUri, nil
}

func (a *mqlAwsEmrCluster) terminationProtected() (bool, error) {
	if err := a.fetchClusterDetails(); err != nil {
		return false, err
	}
	return a.cacheTerminationProtected, nil
}

func (a *mqlAwsEmrCluster) masterPublicDnsName() (string, error) {
	if err := a.fetchClusterDetails(); err != nil {
		return "", err
	}
	return a.cacheMasterPublicDnsName, nil
}

func (a *mqlAwsEmrCluster) logEncryptionKmsKey() (*mqlAwsKmsKey, error) {
	if err := a.fetchClusterDetails(); err != nil {
		return nil, err
	}
	if a.cacheLogEncryptionKmsKeyId == nil || *a.cacheLogEncryptionKmsKeyId == "" {
		a.LogEncryptionKmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlKey, err := NewResource(a.MqlRuntime, ResourceAwsKmsKey,
		map[string]*llx.RawData{
			"arn": llx.StringDataPtr(a.cacheLogEncryptionKmsKeyId),
		})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsEmrCluster) encryptionConfiguration() (*mqlAwsEmrClusterEncryptionConfiguration, error) {
	if err := a.fetchClusterDetails(); err != nil {
		return nil, err
	}
	if a.cacheSecurityConfig == "" {
		a.EncryptionConfiguration.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	region, err := GetRegionFromArn(a.Arn.Data)
	if err != nil {
		return nil, err
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Emr(region)
	ctx := context.Background()

	resp, err := svc.DescribeSecurityConfiguration(ctx, &emr.DescribeSecurityConfigurationInput{
		Name: &a.cacheSecurityConfig,
	})
	if err != nil {
		return nil, err
	}
	if resp.SecurityConfiguration == nil {
		a.EncryptionConfiguration.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	var configJSON map[string]any
	if err := json.Unmarshal([]byte(*resp.SecurityConfiguration), &configJSON); err != nil {
		return nil, err
	}

	encConfigRaw, ok := configJSON["EncryptionConfiguration"]
	if !ok {
		a.EncryptionConfiguration.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	encConfig, ok := encConfigRaw.(map[string]any)
	if !ok {
		a.EncryptionConfiguration.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	atRestEnabled, ok := encConfig["EnableAtRestEncryption"].(bool)
	if !ok {
		if _, exists := encConfig["EnableAtRestEncryption"]; exists {
			log.Warn().Str("cluster", a.Arn.Data).Msg("unexpected type for EnableAtRestEncryption in security configuration")
		}
	}
	inTransitEnabled, ok := encConfig["EnableInTransitEncryption"].(bool)
	if !ok {
		if _, exists := encConfig["EnableInTransitEncryption"]; exists {
			log.Warn().Str("cluster", a.Arn.Data).Msg("unexpected type for EnableInTransitEncryption in security configuration")
		}
	}

	var atRestConfig any
	if v, exists := encConfig["AtRestEncryptionConfiguration"]; exists {
		atRestConfig, err = convert.JsonToDict(v)
		if err != nil {
			return nil, err
		}
	}
	var inTransitConfig any
	if v, exists := encConfig["InTransitEncryptionConfiguration"]; exists {
		inTransitConfig, err = convert.JsonToDict(v)
		if err != nil {
			return nil, err
		}
	}

	res, err := CreateResource(a.MqlRuntime, "aws.emr.cluster.encryptionConfiguration",
		map[string]*llx.RawData{
			"__id":                   llx.StringData(a.Arn.Data + "/encryptionConfiguration"),
			"atRestEnabled":          llx.BoolData(atRestEnabled),
			"inTransitEnabled":       llx.BoolData(inTransitEnabled),
			"atRestConfiguration":    llx.DictData(atRestConfig),
			"inTransitConfiguration": llx.DictData(inTransitConfig),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsEmrClusterEncryptionConfiguration), nil
}

func (a *mqlAwsEmrCluster) masterInstances() ([]any, error) {
	arn := a.Arn.Data
	id := a.Id.Data
	region, err := GetRegionFromArn(arn)
	if err != nil {
		return nil, err
	}
	res := []emrtypes.Instance{}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Emr(region)
	ctx := context.Background()

	params := &emr.ListInstancesInput{
		ClusterId:          &id,
		InstanceGroupTypes: []emrtypes.InstanceGroupType{"MASTER"},
	}
	paginator := emr.NewListInstancesPaginator(svc, params)
	for paginator.HasMorePages() {
		instances, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		res = append(res, instances.Instances...)
	}
	return convert.JsonToDictSlice(res)
}

// ── Block Public Access Configuration ───────────────────────────────────────

func (a *mqlAwsEmr) blockPublicAccessConfiguration() (any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	regions, err := conn.Regions()
	if err != nil {
		return nil, err
	}
	if len(regions) == 0 {
		return nil, nil
	}
	// Account-level config: fetch from the first available region
	svc := conn.Emr(regions[0])
	ctx := context.Background()
	resp, err := svc.GetBlockPublicAccessConfiguration(ctx, &emr.GetBlockPublicAccessConfigurationInput{})
	if err != nil {
		if Is400AccessDeniedError(err) {
			return nil, nil
		}
		return nil, err
	}
	return convert.JsonToDict(resp.BlockPublicAccessConfiguration)
}

// ── Steps ───────────────────────────────────────────────────────────────────

func (a *mqlAwsEmrCluster) steps() ([]any, error) {
	arn := a.Arn.Data
	clusterId := a.Id.Data
	region, err := GetRegionFromArn(arn)
	if err != nil {
		return nil, err
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Emr(region)
	ctx := context.Background()

	res := []any{}
	paginator := emr.NewListStepsPaginator(svc, &emr.ListStepsInput{ClusterId: &clusterId})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				return res, nil
			}
			return nil, err
		}
		for _, step := range page.Steps {
			var status string
			var createdAt, startedAt, endedAt *llx.RawData
			if step.Status != nil {
				status = string(step.Status.State)
				if step.Status.Timeline != nil {
					createdAt = llx.TimeDataPtr(step.Status.Timeline.CreationDateTime)
					startedAt = llx.TimeDataPtr(step.Status.Timeline.StartDateTime)
					endedAt = llx.TimeDataPtr(step.Status.Timeline.EndDateTime)
				}
			}
			if createdAt == nil {
				createdAt = llx.TimeDataPtr(nil)
			}
			if startedAt == nil {
				startedAt = llx.TimeDataPtr(nil)
			}
			if endedAt == nil {
				endedAt = llx.TimeDataPtr(nil)
			}

			var jar string
			var args []any
			if step.Config != nil {
				jar = convert.ToValue(step.Config.Jar)
				args = make([]any, len(step.Config.Args))
				for i, a := range step.Config.Args {
					args[i] = a
				}
			}

			mqlStep, err := CreateResource(a.MqlRuntime, "aws.emr.cluster.step",
				map[string]*llx.RawData{
					"__id":            llx.StringData(fmt.Sprintf("%s/step/%s", arn, convert.ToValue(step.Id))),
					"id":              llx.StringDataPtr(step.Id),
					"name":            llx.StringDataPtr(step.Name),
					"actionOnFailure": llx.StringData(string(step.ActionOnFailure)),
					"status":          llx.StringData(status),
					"jar":             llx.StringData(jar),
					"args":            llx.ArrayData(args, types.String),
					"createdAt":       createdAt,
					"startedAt":       startedAt,
					"endedAt":         endedAt,
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlStep)
		}
	}
	return res, nil
}

func (a *mqlAwsEmrClusterStep) id() (string, error) {
	return a.__id, nil
}

// ── Instance Groups ─────────────────────────────────────────────────────────

func (a *mqlAwsEmrCluster) instanceGroups() ([]any, error) {
	arn := a.Arn.Data
	clusterId := a.Id.Data
	region, err := GetRegionFromArn(arn)
	if err != nil {
		return nil, err
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Emr(region)
	ctx := context.Background()

	res := []any{}
	paginator := emr.NewListInstanceGroupsPaginator(svc, &emr.ListInstanceGroupsInput{ClusterId: &clusterId})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				return res, nil
			}
			return nil, err
		}
		for _, ig := range page.InstanceGroups {
			var status string
			if ig.Status != nil {
				status = string(ig.Status.State)
			}
			mqlIg, err := CreateResource(a.MqlRuntime, "aws.emr.cluster.instanceGroup",
				map[string]*llx.RawData{
					"__id":                   llx.StringData(fmt.Sprintf("%s/instanceGroup/%s", arn, convert.ToValue(ig.Id))),
					"id":                     llx.StringDataPtr(ig.Id),
					"name":                   llx.StringDataPtr(ig.Name),
					"instanceGroupType":      llx.StringData(string(ig.InstanceGroupType)),
					"instanceType":           llx.StringDataPtr(ig.InstanceType),
					"market":                 llx.StringData(string(ig.Market)),
					"requestedInstanceCount": llx.IntDataDefault(ig.RequestedInstanceCount, 0),
					"runningInstanceCount":   llx.IntDataDefault(ig.RunningInstanceCount, 0),
					"status":                 llx.StringData(status),
					"bidPrice":               llx.StringDataPtr(ig.BidPrice),
					"ebsOptimized":           llx.BoolDataPtr(ig.EbsOptimized),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlIg)
		}
	}
	return res, nil
}

func (a *mqlAwsEmrClusterInstanceGroup) id() (string, error) {
	return a.__id, nil
}

// ── Bootstrap Actions ───────────────────────────────────────────────────────

func (a *mqlAwsEmrCluster) bootstrapActions() ([]any, error) {
	arn := a.Arn.Data
	clusterId := a.Id.Data
	region, err := GetRegionFromArn(arn)
	if err != nil {
		return nil, err
	}
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Emr(region)
	ctx := context.Background()

	res := []any{}
	idx := 0
	paginator := emr.NewListBootstrapActionsPaginator(svc, &emr.ListBootstrapActionsInput{ClusterId: &clusterId})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				return res, nil
			}
			return nil, err
		}
		for _, cmd := range page.BootstrapActions {
			args := make([]any, len(cmd.Args))
			for i, a := range cmd.Args {
				args[i] = a
			}
			mqlBa, err := CreateResource(a.MqlRuntime, "aws.emr.cluster.bootstrapAction",
				map[string]*llx.RawData{
					"__id":       llx.StringData(fmt.Sprintf("%s/bootstrapAction/%d", arn, idx)),
					"name":       llx.StringDataPtr(cmd.Name),
					"scriptPath": llx.StringDataPtr(cmd.ScriptPath),
					"args":       llx.ArrayData(args, types.String),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlBa)
			idx++
		}
	}
	return res, nil
}

func (a *mqlAwsEmrClusterBootstrapAction) id() (string, error) {
	return a.__id, nil
}
