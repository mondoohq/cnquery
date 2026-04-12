// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/batch"
	batch_types "github.com/aws/aws-sdk-go-v2/service/batch/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsBatch) id() (string, error) {
	return "aws.batch", nil
}

func (a *mqlAwsBatch) computeEnvironments() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getComputeEnvironments(conn), 5)
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

func (a *mqlAwsBatch) getComputeEnvironments(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("batch>getComputeEnvironments>calling aws with region %s", region)

			svc := conn.Batch(region)
			ctx := context.Background()
			res := []any{}

			var nextToken *string
			for {
				resp, err := svc.DescribeComputeEnvironments(ctx, &batch.DescribeComputeEnvironmentsInput{
					NextToken: nextToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Warn().Str("region", region).Msg("Batch is not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, ce := range resp.ComputeEnvironments {
					tags := make(map[string]any)
					for k, v := range ce.Tags {
						tags[k] = v
					}

					mqlCe, err := CreateResource(a.MqlRuntime, "aws.batch.computeEnvironment",
						map[string]*llx.RawData{
							"__id":                       llx.StringDataPtr(ce.ComputeEnvironmentArn),
							"arn":                        llx.StringDataPtr(ce.ComputeEnvironmentArn),
							"name":                       llx.StringDataPtr(ce.ComputeEnvironmentName),
							"region":                     llx.StringData(region),
							"state":                      llx.StringData(string(ce.State)),
							"status":                     llx.StringData(string(ce.Status)),
							"statusReason":               llx.StringDataPtr(ce.StatusReason),
							"type":                       llx.StringData(string(ce.Type)),
							"containerOrchestrationType": llx.StringData(string(ce.ContainerOrchestrationType)),
							"tags":                       llx.MapData(tags, types.String),
						})
					if err != nil {
						return nil, err
					}
					mqlCeRes := mqlCe.(*mqlAwsBatchComputeEnvironment)
					mqlCeRes.cacheComputeResources = ce.ComputeResources
					mqlCeRes.cacheServiceRoleArn = ce.ServiceRole
					mqlCeRes.cacheEcsClusterArn = ce.EcsClusterArn
					res = append(res, mqlCeRes)
				}

				if resp.NextToken == nil {
					break
				}
				nextToken = resp.NextToken
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsBatchComputeEnvironmentInternal struct {
	cacheComputeResources *batch_types.ComputeResource
	cacheServiceRoleArn   *string
	cacheEcsClusterArn    *string
}

func (a *mqlAwsBatchComputeEnvironment) maxVcpus() (int64, error) {
	if a.cacheComputeResources == nil || a.cacheComputeResources.MaxvCpus == nil {
		return 0, nil
	}
	return int64(*a.cacheComputeResources.MaxvCpus), nil
}

func (a *mqlAwsBatchComputeEnvironment) minVcpus() (int64, error) {
	if a.cacheComputeResources == nil || a.cacheComputeResources.MinvCpus == nil {
		return 0, nil
	}
	return int64(*a.cacheComputeResources.MinvCpus), nil
}

func (a *mqlAwsBatchComputeEnvironment) desiredVcpus() (int64, error) {
	if a.cacheComputeResources == nil || a.cacheComputeResources.DesiredvCpus == nil {
		return 0, nil
	}
	return int64(*a.cacheComputeResources.DesiredvCpus), nil
}

func (a *mqlAwsBatchComputeEnvironment) computeResourceType() (string, error) {
	if a.cacheComputeResources == nil {
		return "", nil
	}
	return string(a.cacheComputeResources.Type), nil
}

func (a *mqlAwsBatchComputeEnvironment) instanceTypes() ([]any, error) {
	if a.cacheComputeResources == nil {
		return []any{}, nil
	}
	res := make([]any, len(a.cacheComputeResources.InstanceTypes))
	for i, t := range a.cacheComputeResources.InstanceTypes {
		res[i] = t
	}
	return res, nil
}

func (a *mqlAwsBatchComputeEnvironment) allocationStrategy() (string, error) {
	if a.cacheComputeResources == nil {
		return "", nil
	}
	return string(a.cacheComputeResources.AllocationStrategy), nil
}

func (a *mqlAwsBatchComputeEnvironment) subnets() ([]any, error) {
	if a.cacheComputeResources == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, subnetId := range a.cacheComputeResources.Subnets {
		mqlSubnet, err := NewResource(a.MqlRuntime, "aws.vpc.subnet",
			map[string]*llx.RawData{
				"id": llx.StringData(subnetId),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSubnet)
	}
	return res, nil
}

func (a *mqlAwsBatchComputeEnvironment) securityGroups() ([]any, error) {
	if a.cacheComputeResources == nil {
		return []any{}, nil
	}
	res := []any{}
	for _, sgId := range a.cacheComputeResources.SecurityGroupIds {
		mqlSg, err := NewResource(a.MqlRuntime, "aws.ec2.securitygroup",
			map[string]*llx.RawData{
				"id": llx.StringData(sgId),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSg)
	}
	return res, nil
}

func (a *mqlAwsBatchComputeEnvironment) iamRole() (*mqlAwsIamRole, error) {
	if a.cacheServiceRoleArn == nil || *a.cacheServiceRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheServiceRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsBatchComputeEnvironment) ecsCluster() (*mqlAwsEcsCluster, error) {
	if a.cacheEcsClusterArn == nil || *a.cacheEcsClusterArn == "" {
		a.EcsCluster.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.ecs.cluster",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheEcsClusterArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsEcsCluster), nil
}

func (a *mqlAwsBatch) jobQueues() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getJobQueues(conn), 5)
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

func (a *mqlAwsBatch) getJobQueues(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("batch>getJobQueues>calling aws with region %s", region)

			svc := conn.Batch(region)
			ctx := context.Background()
			res := []any{}

			var nextToken *string
			for {
				resp, err := svc.DescribeJobQueues(ctx, &batch.DescribeJobQueuesInput{
					NextToken: nextToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Warn().Str("region", region).Msg("Batch is not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, jq := range resp.JobQueues {
					tags := make(map[string]any)
					for k, v := range jq.Tags {
						tags[k] = v
					}

					ceOrder, err := convert.JsonToDictSlice(jq.ComputeEnvironmentOrder)
					if err != nil {
						return nil, err
					}

					mqlJq, err := CreateResource(a.MqlRuntime, "aws.batch.jobQueue",
						map[string]*llx.RawData{
							"__id":         llx.StringDataPtr(jq.JobQueueArn),
							"arn":          llx.StringDataPtr(jq.JobQueueArn),
							"name":         llx.StringDataPtr(jq.JobQueueName),
							"region":       llx.StringData(region),
							"state":        llx.StringData(string(jq.State)),
							"status":       llx.StringData(string(jq.Status)),
							"statusReason": llx.StringDataPtr(jq.StatusReason),
							"priority":     llx.IntDataDefault(jq.Priority, 0),
							"tags":         llx.MapData(tags, types.String),
						})
					if err != nil {
						return nil, err
					}
					mqlJqRes := mqlJq.(*mqlAwsBatchJobQueue)
					mqlJqRes.cacheComputeEnvironmentOrder = ceOrder
					res = append(res, mqlJqRes)
				}

				if resp.NextToken == nil {
					break
				}
				nextToken = resp.NextToken
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsBatchJobQueueInternal struct {
	cacheComputeEnvironmentOrder []any
}

func (a *mqlAwsBatchJobQueue) computeEnvironmentOrder() ([]any, error) {
	return a.cacheComputeEnvironmentOrder, nil
}

func (a *mqlAwsBatch) jobDefinitions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getJobDefinitions(conn), 5)
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

func (a *mqlAwsBatch) getJobDefinitions(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("batch>getJobDefinitions>calling aws with region %s", region)

			svc := conn.Batch(region)
			ctx := context.Background()
			res := []any{}

			var nextToken *string
			for {
				resp, err := svc.DescribeJobDefinitions(ctx, &batch.DescribeJobDefinitionsInput{
					Status:    aws.String("ACTIVE"),
					NextToken: nextToken,
				})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Warn().Str("region", region).Msg("Batch is not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, jd := range resp.JobDefinitions {
					tags := make(map[string]any)
					for k, v := range jd.Tags {
						tags[k] = v
					}

					mqlJd, err := CreateResource(a.MqlRuntime, "aws.batch.jobDefinition",
						map[string]*llx.RawData{
							"__id":     llx.StringDataPtr(jd.JobDefinitionArn),
							"arn":      llx.StringDataPtr(jd.JobDefinitionArn),
							"name":     llx.StringDataPtr(jd.JobDefinitionName),
							"region":   llx.StringData(region),
							"revision": llx.IntDataPtr(jd.Revision),
							"type":     llx.StringDataPtr(jd.Type),
							"status":   llx.StringDataPtr(jd.Status),
							"tags":     llx.MapData(tags, types.String),
						})
					if err != nil {
						return nil, err
					}
					mqlJdRes := mqlJd.(*mqlAwsBatchJobDefinition)
					mqlJdRes.cacheContainerProperties = jd.ContainerProperties
					mqlJdRes.cacheNodeProperties = jd.NodeProperties
					mqlJdRes.cacheRetryStrategy = jd.RetryStrategy
					mqlJdRes.cacheTimeout = jd.Timeout
					res = append(res, mqlJdRes)
				}

				if resp.NextToken == nil {
					break
				}
				nextToken = resp.NextToken
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsBatchJobDefinitionInternal struct {
	cacheContainerProperties *batch_types.ContainerProperties
	cacheNodeProperties      *batch_types.NodeProperties
	cacheRetryStrategy       *batch_types.RetryStrategy
	cacheTimeout             *batch_types.JobTimeout
}

func (a *mqlAwsBatchJobDefinition) containerProperties() (any, error) {
	if a.cacheContainerProperties == nil {
		return nil, nil
	}
	return convert.JsonToDict(a.cacheContainerProperties)
}

func (a *mqlAwsBatchJobDefinition) container() (*mqlAwsBatchJobDefinitionContainerProperties, error) {
	cp := a.cacheContainerProperties
	if cp == nil {
		a.Container.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	image := convert.ToValue(cp.Image)
	vcpus := int64(0)
	if cp.Vcpus != nil {
		vcpus = int64(*cp.Vcpus)
	}
	memory := int64(0)
	if cp.Memory != nil {
		memory = int64(*cp.Memory)
	}
	command := make([]any, len(cp.Command))
	for i, c := range cp.Command {
		command[i] = c
	}

	env, err := convert.JsonToDictSlice(cp.Environment)
	if err != nil {
		return nil, err
	}
	resReqs, err := convert.JsonToDictSlice(cp.ResourceRequirements)
	if err != nil {
		return nil, err
	}
	logConfig, err := convert.JsonToDict(cp.LogConfiguration)
	if err != nil {
		return nil, err
	}
	linuxParams, err := convert.JsonToDict(cp.LinuxParameters)
	if err != nil {
		return nil, err
	}
	fargateConfig, err := convert.JsonToDict(cp.FargatePlatformConfiguration)
	if err != nil {
		return nil, err
	}

	privileged := false
	if cp.Privileged != nil {
		privileged = *cp.Privileged
	}
	readonlyRoot := false
	if cp.ReadonlyRootFilesystem != nil {
		readonlyRoot = *cp.ReadonlyRootFilesystem
	}

	res, err := CreateResource(a.MqlRuntime, "aws.batch.jobDefinition.containerProperties",
		map[string]*llx.RawData{
			"__id":                   llx.StringData(a.Arn.Data + "/containerProperties"),
			"image":                  llx.StringData(image),
			"vcpus":                  llx.IntData(vcpus),
			"memory":                 llx.IntData(memory),
			"command":                llx.ArrayData(command, types.String),
			"environment":            llx.ArrayData(env, types.Dict),
			"privileged":             llx.BoolData(privileged),
			"readonlyRootFilesystem": llx.BoolData(readonlyRoot),
			"resourceRequirements":   llx.ArrayData(resReqs, types.Dict),
			"logConfiguration":       llx.DictData(logConfig),
			"linuxParameters":        llx.DictData(linuxParams),
			"fargateConfig":          llx.DictData(fargateConfig),
		})
	if err != nil {
		return nil, err
	}
	mqlCP := res.(*mqlAwsBatchJobDefinitionContainerProperties)
	mqlCP.cacheJobRoleArn = cp.JobRoleArn
	mqlCP.cacheExecutionRoleArn = cp.ExecutionRoleArn
	return mqlCP, nil
}

type mqlAwsBatchJobDefinitionContainerPropertiesInternal struct {
	cacheJobRoleArn       *string
	cacheExecutionRoleArn *string
}

func (a *mqlAwsBatchJobDefinitionContainerProperties) id() (string, error) {
	// __id is set via CreateResource args
	return a.__id, nil
}

func (a *mqlAwsBatchJobDefinitionContainerProperties) jobRole() (*mqlAwsIamRole, error) {
	if a.cacheJobRoleArn == nil || *a.cacheJobRoleArn == "" {
		a.JobRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheJobRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsBatchJobDefinitionContainerProperties) executionRole() (*mqlAwsIamRole, error) {
	if a.cacheExecutionRoleArn == nil || *a.cacheExecutionRoleArn == "" {
		a.ExecutionRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheExecutionRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsBatchJobDefinition) nodeProperties() (any, error) {
	if a.cacheNodeProperties == nil {
		return nil, nil
	}
	dict, err := convert.JsonToDict(a.cacheNodeProperties)
	if err != nil {
		return nil, err
	}
	return dict, nil
}

func (a *mqlAwsBatchJobDefinition) retryStrategy() (any, error) {
	if a.cacheRetryStrategy == nil {
		return nil, nil
	}
	return convert.JsonToDict(a.cacheRetryStrategy)
}

func (a *mqlAwsBatchJobDefinition) retry() (*mqlAwsBatchJobDefinitionRetryStrategy, error) {
	rs := a.cacheRetryStrategy
	if rs == nil {
		a.Retry.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	attempts := int64(0)
	if rs.Attempts != nil {
		attempts = int64(*rs.Attempts)
	}
	evalOnExit, err := convert.JsonToDictSlice(rs.EvaluateOnExit)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(a.MqlRuntime, "aws.batch.jobDefinition.retryStrategy",
		map[string]*llx.RawData{
			"__id":           llx.StringData(a.Arn.Data + "/retryStrategy"),
			"attempts":       llx.IntData(attempts),
			"evaluateOnExit": llx.ArrayData(evalOnExit, types.Dict),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsBatchJobDefinitionRetryStrategy), nil
}

func (a *mqlAwsBatchJobDefinitionRetryStrategy) id() (string, error) {
	// __id is set via CreateResource args
	return a.__id, nil
}

func (a *mqlAwsBatchJobDefinition) timeout() (any, error) {
	if a.cacheTimeout == nil {
		return nil, nil
	}
	return convert.JsonToDict(a.cacheTimeout)
}

func (a *mqlAwsBatchJobDefinition) jobTimeout() (*mqlAwsBatchJobDefinitionTimeout, error) {
	t := a.cacheTimeout
	if t == nil {
		a.JobTimeout.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	attemptDuration := int64(0)
	if t.AttemptDurationSeconds != nil {
		attemptDuration = int64(*t.AttemptDurationSeconds)
	}

	res, err := CreateResource(a.MqlRuntime, "aws.batch.jobDefinition.timeout",
		map[string]*llx.RawData{
			"__id":                   llx.StringData(a.Arn.Data + "/timeout"),
			"attemptDurationSeconds": llx.IntData(attemptDuration),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsBatchJobDefinitionTimeout), nil
}

func (a *mqlAwsBatchJobDefinitionTimeout) id() (string, error) {
	// __id is set via CreateResource args
	return a.__id, nil
}
