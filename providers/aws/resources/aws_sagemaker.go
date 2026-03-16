// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/aws/smithy-go/transport/http"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

func (a *mqlAwsSagemaker) id() (string, error) {
	return ResourceAwsSagemaker, nil
}

func (a *mqlAwsSagemaker) endpoints() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getEndpoints(conn), 5)
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

func (a *mqlAwsSagemaker) getEndpoints(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sagemaker(region)
			ctx := context.Background()
			res := []any{}

			params := &sagemaker.ListEndpointsInput{}
			paginator := sagemaker.NewListEndpointsPaginator(svc, params)
			for paginator.HasMorePages() {
				endpoints, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, endpoint := range endpoints.Endpoints {
					// Only fetch tags eagerly when tag-based filters are configured
					var eagerTags map[string]any
					if conn.Filters.General.HasTags() {
						tags, err := getSagemakerTags(ctx, svc, endpoint.EndpointArn)
						if err != nil {
							return nil, err
						}
						if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
							log.Debug().Interface("endpoint", endpoint.EndpointArn).Msg("skipping sagemaker endpoint due to filters")
							continue
						}
						eagerTags = tags
					}

					mqlEndpoint, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerEndpoint,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(endpoint.EndpointArn),
							"name":           llx.StringDataPtr(endpoint.EndpointName),
							"region":         llx.StringData(region),
							"createdAt":      llx.TimeDataPtr(endpoint.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(endpoint.LastModifiedTime),
							"status":         llx.StringData(string(endpoint.EndpointStatus)),
						})
					if err != nil {
						return nil, err
					}
					ep := mqlEndpoint.(*mqlAwsSagemakerEndpoint)
					if eagerTags != nil {
						ep.cacheTags = eagerTags
						ep.tagsFetched = true
					}
					res = append(res, mqlEndpoint)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsSagemakerEndpoint) config() (map[string]any, error) {
	name := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Sagemaker(region)
	ctx := context.Background()
	config, err := svc.DescribeEndpointConfig(ctx, &sagemaker.DescribeEndpointConfigInput{EndpointConfigName: &name})
	if err != nil {
		return nil, err
	}
	return convert.JsonToDict(config)
}

func (a *mqlAwsSagemaker) notebookInstances() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getNotebookInstances(conn), 5)
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

func (a *mqlAwsSagemaker) getNotebookInstances(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sagemaker(region)
			ctx := context.Background()
			res := []any{}

			params := &sagemaker.ListNotebookInstancesInput{}
			paginator := sagemaker.NewListNotebookInstancesPaginator(svc, params)
			for paginator.HasMorePages() {
				notebookInstances, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}
				for _, instance := range notebookInstances.NotebookInstances {
					// Only fetch tags eagerly when tag-based filters are configured
					var eagerTags map[string]any
					if conn.Filters.General.HasTags() {
						tags, err := getSagemakerTags(ctx, svc, instance.NotebookInstanceArn)
						if err != nil {
							return nil, err
						}
						if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
							log.Debug().Interface("notebook", instance.NotebookInstanceArn).Msg("skipping sagemaker notebook instance due to filters")
							continue
						}
						eagerTags = tags
					}

					mqlNb, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerNotebookinstance,
						map[string]*llx.RawData{
							"arn":            llx.StringData(convert.ToValue(instance.NotebookInstanceArn)),
							"name":           llx.StringData(convert.ToValue(instance.NotebookInstanceName)),
							"region":         llx.StringData(region),
							"createdAt":      llx.TimeDataPtr(instance.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(instance.LastModifiedTime),
							"status":         llx.StringData(string(instance.NotebookInstanceStatus)),
							"url":            llx.StringDataPtr(instance.Url),
						})
					if err != nil {
						return nil, err
					}
					nb := mqlNb.(*mqlAwsSagemakerNotebookinstance)
					if eagerTags != nil {
						nb.cacheTags = eagerTags
						nb.tagsFetched = true
					}
					res = append(res, mqlNb)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func initAwsSagemakerNotebookinstance(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["arn"] == nil {
		return nil, nil, errors.New("arn required to fetch sagemaker notebookinstance")
	}

	obj, err := CreateResource(runtime, "aws.sagemaker", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	sm := obj.(*mqlAwsSagemaker)

	rawResources := sm.GetNotebookInstances()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	arnVal := args["arn"].Value.(string)
	for _, rawResource := range rawResources.Data {
		ni := rawResource.(*mqlAwsSagemakerNotebookinstance)
		if ni.Arn.Data == arnVal {
			return args, ni, nil
		}
	}
	return nil, nil, errors.New("sagemaker notebookinstance does not exist")
}

func (a *mqlAwsSagemakerNotebookinstance) details() (*mqlAwsSagemakerNotebookinstancedetails, error) {
	name := a.Name.Data
	region := a.Region.Data

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(region)
	ctx := context.Background()
	instanceDetails, err := svc.DescribeNotebookInstance(ctx, &sagemaker.DescribeNotebookInstanceInput{NotebookInstanceName: &name})
	if err != nil {
		return nil, err
	}
	args := map[string]*llx.RawData{
		"arn":                  llx.StringDataPtr(instanceDetails.NotebookInstanceArn),
		"directInternetAccess": llx.BoolData(string(instanceDetails.DirectInternetAccess) == "Enabled"),
		"rootAccess":           llx.BoolData(string(instanceDetails.RootAccess) == "Enabled"),
	}
	if instanceDetails.InstanceMetadataServiceConfiguration != nil {
		args["minimumInstanceMetadataServiceVersion"] = llx.StringDataPtr(instanceDetails.InstanceMetadataServiceConfiguration.MinimumInstanceMetadataServiceVersion)
	} else {
		args["minimumInstanceMetadataServiceVersion"] = llx.StringData("1")
	}

	mqlInstanceDetails, err := CreateResource(a.MqlRuntime, "aws.sagemaker.notebookinstancedetails", args)
	if err != nil {
		return nil, err
	}
	details := mqlInstanceDetails.(*mqlAwsSagemakerNotebookinstancedetails)
	details.cacheKmsKey = instanceDetails.KmsKeyId
	details.cacheSubnetId = instanceDetails.SubnetId
	details.region = region
	return details, nil
}

type mqlAwsSagemakerNotebookinstancedetailsInternal struct {
	cacheKmsKey   *string
	cacheSubnetId *string
	region        string
}

func (a *mqlAwsSagemakerNotebookinstancedetails) kmsKey() (*mqlAwsKmsKey, error) {
	if a.cacheKmsKey != nil && *a.cacheKmsKey != "" {
		mqlKeyResource, err := NewResource(a.MqlRuntime, "aws.kms.key",
			map[string]*llx.RawData{"arn": llx.StringData(convert.ToValue(a.cacheKmsKey))},
		)
		if err != nil {
			return nil, err
		}
		return mqlKeyResource.(*mqlAwsKmsKey), nil
	}
	a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAwsSagemakerNotebookinstancedetails) subnet() (*mqlAwsVpcSubnet, error) {
	if a.cacheSubnetId != nil && *a.cacheSubnetId != "" {
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		arn := fmt.Sprintf(subnetArnPattern, a.region, conn.AccountId(), *a.cacheSubnetId)
		res, err := NewResource(a.MqlRuntime, ResourceAwsVpcSubnet, map[string]*llx.RawData{"arn": llx.StringData(arn)})
		if err != nil {
			return nil, err
		}
		return res.(*mqlAwsVpcSubnet), nil
	}
	a.Subnet.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

// sagemakerTagsCache provides lazy-loaded tag caching with double-check locking.
// Embed in Internal structs for SageMaker resources that need lazy tags.
type sagemakerTagsCache struct {
	cacheTags   map[string]any
	tagsFetched bool
	tagsLock    sync.Mutex
}

func (c *sagemakerTagsCache) fetchTags(conn *connection.AwsConnection, region, arn string) (map[string]any, error) {
	if c.tagsFetched {
		return c.cacheTags, nil
	}
	c.tagsLock.Lock()
	defer c.tagsLock.Unlock()
	if c.tagsFetched {
		return c.cacheTags, nil
	}

	svc := conn.Sagemaker(region)
	ctx := context.Background()
	tags, err := getSagemakerTags(ctx, svc, &arn)
	if err != nil {
		return nil, err
	}
	c.cacheTags = tags
	c.tagsFetched = true
	return tags, nil
}

type mqlAwsSagemakerEndpointInternal struct {
	sagemakerTagsCache
}

func (a *mqlAwsSagemakerEndpoint) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return a.fetchTags(conn, a.Region.Data, a.Arn.Data)
}

func (a *mqlAwsSagemakerEndpoint) id() (string, error) {
	return a.Arn.Data, nil
}

type mqlAwsSagemakerNotebookinstanceInternal struct {
	sagemakerTagsCache
}

func (a *mqlAwsSagemakerNotebookinstance) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return a.fetchTags(conn, a.Region.Data, a.Arn.Data)
}

func (a *mqlAwsSagemakerNotebookinstance) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerNotebookinstancedetails) id() (string, error) {
	return a.Arn.Data, nil
}

// ---- Models ----

func (a *mqlAwsSagemaker) models() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getModels(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getModels(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sagemaker(region)
			ctx := context.Background()
			res := []any{}

			paginator := sagemaker.NewListModelsPaginator(svc, &sagemaker.ListModelsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, model := range page.Models {
					var eagerTags map[string]any
					if conn.Filters.General.HasTags() {
						tags, err := getSagemakerTags(ctx, svc, model.ModelArn)
						if err != nil {
							return nil, err
						}
						if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
							log.Debug().Interface("model", model.ModelArn).Msg("skipping sagemaker model due to filters")
							continue
						}
						eagerTags = tags
					}

					mqlModel, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerModel,
						map[string]*llx.RawData{
							"arn":       llx.StringDataPtr(model.ModelArn),
							"name":      llx.StringDataPtr(model.ModelName),
							"region":    llx.StringData(region),
							"createdAt": llx.TimeDataPtr(model.CreationTime),
						})
					if err != nil {
						return nil, err
					}
					m := mqlModel.(*mqlAwsSagemakerModel)
					if eagerTags != nil {
						m.cacheTags = eagerTags
						m.tagsFetched = true
					}
					res = append(res, mqlModel)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerModelInternal struct {
	sagemakerTagsCache
	detailsFetched              bool
	detailsLock                 sync.Mutex
	cacheRoleArn                *string
	cacheEnableNetworkIsolation bool
	cachePrimaryContainer       any
	cacheVpcConfig              any
}

func (a *mqlAwsSagemakerModel) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerModel) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return a.fetchTags(conn, a.Region.Data, a.Arn.Data)
}

func (a *mqlAwsSagemakerModel) fetchDetails() error {
	if a.detailsFetched {
		return nil
	}
	a.detailsLock.Lock()
	defer a.detailsLock.Unlock()
	if a.detailsFetched {
		return nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(a.Region.Data)
	ctx := context.Background()
	name := a.Name.Data
	resp, err := svc.DescribeModel(ctx, &sagemaker.DescribeModelInput{ModelName: &name})
	if err != nil {
		return err
	}

	a.cacheRoleArn = resp.ExecutionRoleArn
	if resp.EnableNetworkIsolation != nil {
		a.cacheEnableNetworkIsolation = *resp.EnableNetworkIsolation
	}
	a.cachePrimaryContainer, _ = convert.JsonToDict(resp.PrimaryContainer)
	a.cacheVpcConfig, _ = convert.JsonToDict(resp.VpcConfig)
	a.detailsFetched = true
	return nil
}

func (a *mqlAwsSagemakerModel) enableNetworkIsolation() (bool, error) {
	if err := a.fetchDetails(); err != nil {
		return false, err
	}
	return a.cacheEnableNetworkIsolation, nil
}

func (a *mqlAwsSagemakerModel) iamRole() (*mqlAwsIamRole, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheRoleArn == nil || *a.cacheRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsSagemakerModel) primaryContainer() (map[string]any, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cachePrimaryContainer == nil {
		return nil, nil
	}
	return a.cachePrimaryContainer.(map[string]any), nil
}

func (a *mqlAwsSagemakerModel) vpcConfig() (map[string]any, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheVpcConfig == nil {
		return nil, nil
	}
	return a.cacheVpcConfig.(map[string]any), nil
}

// ---- Training Jobs ----

func (a *mqlAwsSagemaker) trainingJobs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getTrainingJobs(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getTrainingJobs(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sagemaker(region)
			ctx := context.Background()
			res := []any{}

			paginator := sagemaker.NewListTrainingJobsPaginator(svc, &sagemaker.ListTrainingJobsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, job := range page.TrainingJobSummaries {
					var eagerTags map[string]any
					if conn.Filters.General.HasTags() {
						tags, err := getSagemakerTags(ctx, svc, job.TrainingJobArn)
						if err != nil {
							return nil, err
						}
						if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
							log.Debug().Interface("trainingjob", job.TrainingJobArn).Msg("skipping sagemaker training job due to filters")
							continue
						}
						eagerTags = tags
					}

					mqlJob, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerTrainingjob,
						map[string]*llx.RawData{
							"arn":             llx.StringDataPtr(job.TrainingJobArn),
							"name":            llx.StringDataPtr(job.TrainingJobName),
							"region":          llx.StringData(region),
							"status":          llx.StringData(string(job.TrainingJobStatus)),
							"createdAt":       llx.TimeDataPtr(job.CreationTime),
							"lastModifiedAt":  llx.TimeDataPtr(job.LastModifiedTime),
							"trainingEndTime": llx.TimeDataPtr(job.TrainingEndTime),
						})
					if err != nil {
						return nil, err
					}
					tj := mqlJob.(*mqlAwsSagemakerTrainingjob)
					if eagerTags != nil {
						tj.cacheTags = eagerTags
						tj.tagsFetched = true
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

type mqlAwsSagemakerTrainingjobInternal struct {
	sagemakerTagsCache
	detailsFetched                   bool
	detailsLock                      sync.Mutex
	cacheRoleArn                     *string
	cacheAlgorithmSpec               any
	cacheHyperParams                 map[string]string
	cacheEnableNetworkIsolation      bool
	cacheEnableInterContainerEncrypt bool
	cacheFailureReason               *string
	cacheBillableTime                int64
	cacheVpcConfig                   any
	cacheOutputDataConfig            any
	cacheResourceConfig              any
	cacheStoppingCondition           any
}

func (a *mqlAwsSagemakerTrainingjob) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerTrainingjob) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return a.fetchTags(conn, a.Region.Data, a.Arn.Data)
}

func (a *mqlAwsSagemakerTrainingjob) fetchDetails() error {
	if a.detailsFetched {
		return nil
	}
	a.detailsLock.Lock()
	defer a.detailsLock.Unlock()
	if a.detailsFetched {
		return nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(a.Region.Data)
	ctx := context.Background()
	name := a.Name.Data
	resp, err := svc.DescribeTrainingJob(ctx, &sagemaker.DescribeTrainingJobInput{TrainingJobName: &name})
	if err != nil {
		return err
	}

	a.cacheRoleArn = resp.RoleArn
	a.cacheAlgorithmSpec, _ = convert.JsonToDict(resp.AlgorithmSpecification)
	a.cacheHyperParams = resp.HyperParameters
	if resp.EnableNetworkIsolation != nil {
		a.cacheEnableNetworkIsolation = *resp.EnableNetworkIsolation
	}
	if resp.EnableInterContainerTrafficEncryption != nil {
		a.cacheEnableInterContainerEncrypt = *resp.EnableInterContainerTrafficEncryption
	}
	a.cacheFailureReason = resp.FailureReason
	if resp.BillableTimeInSeconds != nil {
		a.cacheBillableTime = int64(*resp.BillableTimeInSeconds)
	}
	a.cacheVpcConfig, _ = convert.JsonToDict(resp.VpcConfig)
	a.cacheOutputDataConfig, _ = convert.JsonToDict(resp.OutputDataConfig)
	a.cacheResourceConfig, _ = convert.JsonToDict(resp.ResourceConfig)
	a.cacheStoppingCondition, _ = convert.JsonToDict(resp.StoppingCondition)
	a.detailsFetched = true
	return nil
}

func (a *mqlAwsSagemakerTrainingjob) iamRole() (*mqlAwsIamRole, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheRoleArn == nil || *a.cacheRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsSagemakerTrainingjob) algorithmSpecification() (map[string]any, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheAlgorithmSpec == nil {
		return nil, nil
	}
	return a.cacheAlgorithmSpec.(map[string]any), nil
}

func (a *mqlAwsSagemakerTrainingjob) hyperParameters() (map[string]any, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheHyperParams == nil {
		return nil, nil
	}
	result := make(map[string]any, len(a.cacheHyperParams))
	for k, v := range a.cacheHyperParams {
		result[k] = v
	}
	return result, nil
}

func (a *mqlAwsSagemakerTrainingjob) enableNetworkIsolation() (bool, error) {
	if err := a.fetchDetails(); err != nil {
		return false, err
	}
	return a.cacheEnableNetworkIsolation, nil
}

func (a *mqlAwsSagemakerTrainingjob) enableInterContainerTrafficEncryption() (bool, error) {
	if err := a.fetchDetails(); err != nil {
		return false, err
	}
	return a.cacheEnableInterContainerEncrypt, nil
}

func (a *mqlAwsSagemakerTrainingjob) failureReason() (string, error) {
	if err := a.fetchDetails(); err != nil {
		return "", err
	}
	return convert.ToValue(a.cacheFailureReason), nil
}

func (a *mqlAwsSagemakerTrainingjob) billableTimeInSeconds() (int64, error) {
	if err := a.fetchDetails(); err != nil {
		return 0, err
	}
	return a.cacheBillableTime, nil
}

func (a *mqlAwsSagemakerTrainingjob) vpcConfig() (map[string]any, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheVpcConfig == nil {
		return nil, nil
	}
	return a.cacheVpcConfig.(map[string]any), nil
}

func (a *mqlAwsSagemakerTrainingjob) outputDataConfig() (map[string]any, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheOutputDataConfig == nil {
		return nil, nil
	}
	return a.cacheOutputDataConfig.(map[string]any), nil
}

func (a *mqlAwsSagemakerTrainingjob) resourceConfig() (map[string]any, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheResourceConfig == nil {
		return nil, nil
	}
	return a.cacheResourceConfig.(map[string]any), nil
}

func (a *mqlAwsSagemakerTrainingjob) stoppingCondition() (map[string]any, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheStoppingCondition == nil {
		return nil, nil
	}
	return a.cacheStoppingCondition.(map[string]any), nil
}

// ---- Processing Jobs ----

func (a *mqlAwsSagemaker) processingJobs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getProcessingJobs(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getProcessingJobs(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sagemaker(region)
			ctx := context.Background()
			res := []any{}

			paginator := sagemaker.NewListProcessingJobsPaginator(svc, &sagemaker.ListProcessingJobsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, job := range page.ProcessingJobSummaries {
					var eagerTags map[string]any
					if conn.Filters.General.HasTags() {
						tags, err := getSagemakerTags(ctx, svc, job.ProcessingJobArn)
						if err != nil {
							return nil, err
						}
						if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
							log.Debug().Interface("processingjob", job.ProcessingJobArn).Msg("skipping sagemaker processing job due to filters")
							continue
						}
						eagerTags = tags
					}

					mqlJob, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerProcessingjob,
						map[string]*llx.RawData{
							"arn":               llx.StringDataPtr(job.ProcessingJobArn),
							"name":              llx.StringDataPtr(job.ProcessingJobName),
							"region":            llx.StringData(region),
							"status":            llx.StringData(string(job.ProcessingJobStatus)),
							"createdAt":         llx.TimeDataPtr(job.CreationTime),
							"lastModifiedAt":    llx.TimeDataPtr(job.LastModifiedTime),
							"processingEndTime": llx.TimeDataPtr(job.ProcessingEndTime),
							"failureReason":     llx.StringDataPtr(job.FailureReason),
							"exitMessage":       llx.StringDataPtr(job.ExitMessage),
						})
					if err != nil {
						return nil, err
					}
					pj := mqlJob.(*mqlAwsSagemakerProcessingjob)
					if eagerTags != nil {
						pj.cacheTags = eagerTags
						pj.tagsFetched = true
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

type mqlAwsSagemakerProcessingjobInternal struct {
	sagemakerTagsCache
	detailsFetched                   bool
	detailsLock                      sync.Mutex
	cacheRoleArn                     *string
	cacheEnableNetworkIsolation      bool
	cacheEnableInterContainerEncrypt bool
	cacheVpcConfig                   any
	cacheProcessingResources         any
	cacheEnvironment                 map[string]string
}

func (a *mqlAwsSagemakerProcessingjob) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerProcessingjob) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return a.fetchTags(conn, a.Region.Data, a.Arn.Data)
}

func (a *mqlAwsSagemakerProcessingjob) fetchDetails() error {
	if a.detailsFetched {
		return nil
	}
	a.detailsLock.Lock()
	defer a.detailsLock.Unlock()
	if a.detailsFetched {
		return nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(a.Region.Data)
	ctx := context.Background()
	name := a.Name.Data
	resp, err := svc.DescribeProcessingJob(ctx, &sagemaker.DescribeProcessingJobInput{ProcessingJobName: &name})
	if err != nil {
		return err
	}

	a.cacheRoleArn = resp.RoleArn
	if resp.NetworkConfig != nil {
		if resp.NetworkConfig.EnableNetworkIsolation != nil {
			a.cacheEnableNetworkIsolation = *resp.NetworkConfig.EnableNetworkIsolation
		}
		if resp.NetworkConfig.EnableInterContainerTrafficEncryption != nil {
			a.cacheEnableInterContainerEncrypt = *resp.NetworkConfig.EnableInterContainerTrafficEncryption
		}
		a.cacheVpcConfig, _ = convert.JsonToDict(resp.NetworkConfig.VpcConfig)
	}
	a.cacheProcessingResources, _ = convert.JsonToDict(resp.ProcessingResources)
	a.cacheEnvironment = resp.Environment
	a.detailsFetched = true
	return nil
}

func (a *mqlAwsSagemakerProcessingjob) iamRole() (*mqlAwsIamRole, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheRoleArn == nil || *a.cacheRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsSagemakerProcessingjob) enableNetworkIsolation() (bool, error) {
	if err := a.fetchDetails(); err != nil {
		return false, err
	}
	return a.cacheEnableNetworkIsolation, nil
}

func (a *mqlAwsSagemakerProcessingjob) enableInterContainerTrafficEncryption() (bool, error) {
	if err := a.fetchDetails(); err != nil {
		return false, err
	}
	return a.cacheEnableInterContainerEncrypt, nil
}

func (a *mqlAwsSagemakerProcessingjob) vpcConfig() (map[string]any, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheVpcConfig == nil {
		return nil, nil
	}
	return a.cacheVpcConfig.(map[string]any), nil
}

func (a *mqlAwsSagemakerProcessingjob) processingResources() (map[string]any, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheProcessingResources == nil {
		return nil, nil
	}
	return a.cacheProcessingResources.(map[string]any), nil
}

func (a *mqlAwsSagemakerProcessingjob) environment() (map[string]any, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheEnvironment == nil {
		return nil, nil
	}
	result := make(map[string]any, len(a.cacheEnvironment))
	for k, v := range a.cacheEnvironment {
		result[k] = v
	}
	return result, nil
}

// ---- Pipelines ----

func (a *mqlAwsSagemaker) pipelines() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getPipelines(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getPipelines(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sagemaker(region)
			ctx := context.Background()
			res := []any{}

			paginator := sagemaker.NewListPipelinesPaginator(svc, &sagemaker.ListPipelinesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, pipeline := range page.PipelineSummaries {
					var eagerTags map[string]any
					if conn.Filters.General.HasTags() {
						tags, err := getSagemakerTags(ctx, svc, pipeline.PipelineArn)
						if err != nil {
							return nil, err
						}
						if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
							log.Debug().Interface("pipeline", pipeline.PipelineArn).Msg("skipping sagemaker pipeline due to filters")
							continue
						}
						eagerTags = tags
					}

					mqlPipeline, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerPipeline,
						map[string]*llx.RawData{
							"arn":               llx.StringDataPtr(pipeline.PipelineArn),
							"name":              llx.StringDataPtr(pipeline.PipelineName),
							"displayName":       llx.StringDataPtr(pipeline.PipelineDisplayName),
							"description":       llx.StringDataPtr(pipeline.PipelineDescription),
							"region":            llx.StringData(region),
							"createdAt":         llx.TimeDataPtr(pipeline.CreationTime),
							"lastModifiedAt":    llx.TimeDataPtr(pipeline.LastModifiedTime),
							"lastExecutionTime": llx.TimeDataPtr(pipeline.LastExecutionTime),
						})
					if err != nil {
						return nil, err
					}
					p := mqlPipeline.(*mqlAwsSagemakerPipeline)
					p.cacheRoleArn = pipeline.RoleArn
					if eagerTags != nil {
						p.cacheTags = eagerTags
						p.tagsFetched = true
					}
					res = append(res, mqlPipeline)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerPipelineInternal struct {
	sagemakerTagsCache
	detailsFetched         bool
	detailsLock            sync.Mutex
	cacheRoleArn           *string
	cachePipelineStatus    *string
	cacheDefinition        *string
	cacheParallelismConfig any
}

func (a *mqlAwsSagemakerPipeline) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerPipeline) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return a.fetchTags(conn, a.Region.Data, a.Arn.Data)
}

func (a *mqlAwsSagemakerPipeline) fetchDetails() error {
	if a.detailsFetched {
		return nil
	}
	a.detailsLock.Lock()
	defer a.detailsLock.Unlock()
	if a.detailsFetched {
		return nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(a.Region.Data)
	ctx := context.Background()
	name := a.Name.Data
	resp, err := svc.DescribePipeline(ctx, &sagemaker.DescribePipelineInput{PipelineName: &name})
	if err != nil {
		return err
	}

	a.cacheRoleArn = resp.RoleArn
	status := string(resp.PipelineStatus)
	a.cachePipelineStatus = &status
	a.cacheDefinition = resp.PipelineDefinition
	a.cacheParallelismConfig, _ = convert.JsonToDict(resp.ParallelismConfiguration)
	a.detailsFetched = true
	return nil
}

func (a *mqlAwsSagemakerPipeline) iamRole() (*mqlAwsIamRole, error) {
	// RoleArn is eagerly cached from the list summary; fall back to fetchDetails
	if a.cacheRoleArn == nil {
		if err := a.fetchDetails(); err != nil {
			return nil, err
		}
	}
	if a.cacheRoleArn == nil || *a.cacheRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsSagemakerPipeline) pipelineStatus() (string, error) {
	if err := a.fetchDetails(); err != nil {
		return "", err
	}
	return convert.ToValue(a.cachePipelineStatus), nil
}

func (a *mqlAwsSagemakerPipeline) definition() (string, error) {
	if err := a.fetchDetails(); err != nil {
		return "", err
	}
	return convert.ToValue(a.cacheDefinition), nil
}

func (a *mqlAwsSagemakerPipeline) parallelismConfiguration() (map[string]any, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheParallelismConfig == nil {
		return nil, nil
	}
	return a.cacheParallelismConfig.(map[string]any), nil
}

// ---- Domains ----

func (a *mqlAwsSagemaker) domains() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getDomains(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getDomains(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Sagemaker(region)
			ctx := context.Background()
			res := []any{}

			paginator := sagemaker.NewListDomainsPaginator(svc, &sagemaker.ListDomainsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, domain := range page.Domains {
					var eagerTags map[string]any
					if conn.Filters.General.HasTags() {
						tags, err := getSagemakerTags(ctx, svc, domain.DomainArn)
						if err != nil {
							return nil, err
						}
						if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
							log.Debug().Interface("domain", domain.DomainArn).Msg("skipping sagemaker domain due to filters")
							continue
						}
						eagerTags = tags
					}

					mqlDomain, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerDomain,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(domain.DomainArn),
							"domainId":       llx.StringDataPtr(domain.DomainId),
							"name":           llx.StringDataPtr(domain.DomainName),
							"region":         llx.StringData(region),
							"status":         llx.StringData(string(domain.Status)),
							"url":            llx.StringDataPtr(domain.Url),
							"createdAt":      llx.TimeDataPtr(domain.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(domain.LastModifiedTime),
						})
					if err != nil {
						return nil, err
					}
					d := mqlDomain.(*mqlAwsSagemakerDomain)
					if eagerTags != nil {
						d.cacheTags = eagerTags
						d.tagsFetched = true
					}
					res = append(res, mqlDomain)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerDomainInternal struct {
	sagemakerTagsCache
	detailsFetched           bool
	detailsLock              sync.Mutex
	cacheAuthMode            *string
	cacheAppNetworkAccess    *string
	cacheVpcId               *string
	cacheKmsKeyId            *string
	cacheHomeEfsId           *string
	cacheDefaultUserSettings any
}

func (a *mqlAwsSagemakerDomain) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerDomain) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return a.fetchTags(conn, a.Region.Data, a.Arn.Data)
}

func (a *mqlAwsSagemakerDomain) fetchDetails() error {
	if a.detailsFetched {
		return nil
	}
	a.detailsLock.Lock()
	defer a.detailsLock.Unlock()
	if a.detailsFetched {
		return nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(a.Region.Data)
	ctx := context.Background()
	domainId := a.DomainId.Data
	resp, err := svc.DescribeDomain(ctx, &sagemaker.DescribeDomainInput{DomainId: &domainId})
	if err != nil {
		return err
	}

	authMode := string(resp.AuthMode)
	a.cacheAuthMode = &authMode
	appNetworkAccess := string(resp.AppNetworkAccessType)
	a.cacheAppNetworkAccess = &appNetworkAccess
	a.cacheVpcId = resp.VpcId
	a.cacheKmsKeyId = resp.KmsKeyId
	a.cacheHomeEfsId = resp.HomeEfsFileSystemId
	a.cacheDefaultUserSettings, _ = convert.JsonToDict(resp.DefaultUserSettings)
	a.detailsFetched = true
	return nil
}

func (a *mqlAwsSagemakerDomain) authMode() (string, error) {
	if err := a.fetchDetails(); err != nil {
		return "", err
	}
	return convert.ToValue(a.cacheAuthMode), nil
}

func (a *mqlAwsSagemakerDomain) appNetworkAccessType() (string, error) {
	if err := a.fetchDetails(); err != nil {
		return "", err
	}
	return convert.ToValue(a.cacheAppNetworkAccess), nil
}

func (a *mqlAwsSagemakerDomain) vpc() (*mqlAwsVpc, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheVpcId == nil || *a.cacheVpcId == "" {
		a.Vpc.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.vpc",
		map[string]*llx.RawData{"id": llx.StringDataPtr(a.cacheVpcId)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsVpc), nil
}

func (a *mqlAwsSagemakerDomain) kmsKey() (*mqlAwsKmsKey, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheKmsKeyId == nil || *a.cacheKmsKeyId == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheKmsKeyId)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsSagemakerDomain) homeEfsFileSystemId() (string, error) {
	if err := a.fetchDetails(); err != nil {
		return "", err
	}
	return convert.ToValue(a.cacheHomeEfsId), nil
}

func (a *mqlAwsSagemakerDomain) defaultUserSettings() (map[string]any, error) {
	if err := a.fetchDetails(); err != nil {
		return nil, err
	}
	if a.cacheDefaultUserSettings == nil {
		return nil, nil
	}
	return a.cacheDefaultUserSettings.(map[string]any), nil
}

func getSagemakerTags(ctx context.Context, svc *sagemaker.Client, arn *string) (map[string]any, error) {
	tags := make(map[string]any)
	var nextToken *string
	for {
		resp, err := svc.ListTags(ctx, &sagemaker.ListTagsInput{
			ResourceArn: arn,
			NextToken:   nextToken,
		})
		var respErr *http.ResponseError
		if err != nil {
			if errors.As(err, &respErr) {
				if respErr.HTTPStatusCode() == 404 {
					return nil, nil
				}
			}
			return nil, err
		}
		for _, t := range resp.Tags {
			if t.Key != nil && t.Value != nil {
				tags[*t.Key] = *t.Value
			}
		}
		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}
	return tags, nil
}
