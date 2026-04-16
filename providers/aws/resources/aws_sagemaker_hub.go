// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

// ---- Hubs ----

func (a *mqlAwsSagemaker) hubs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getHubs(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getHubs(conn *connection.AwsConnection) []*jobpool.Job {
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

			var nextToken *string
			for {
				page, err := svc.ListHubs(ctx, &sagemaker.ListHubsInput{NextToken: nextToken})
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS SageMaker hubs")
						return res, nil
					}
					return nil, err
				}

				for _, hub := range page.HubSummaries {
					mqlHub, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerHub,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(hub.HubArn),
							"name":           llx.StringDataPtr(hub.HubName),
							"description":    llx.StringDataPtr(hub.HubDescription),
							"region":         llx.StringData(region),
							"status":         llx.StringData(string(hub.HubStatus)),
							"createdAt":      llx.TimeDataPtr(hub.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(hub.LastModifiedTime),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlHub)
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

type mqlAwsSagemakerHubInternal struct {
	fetched       bool
	fetchLock     sync.Mutex
	cacheDescribe *sagemaker.DescribeHubOutput
}

func (a *mqlAwsSagemakerHub) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerHub) fetchDetails() (*sagemaker.DescribeHubOutput, error) {
	if a.fetched {
		return a.cacheDescribe, nil
	}
	a.fetchLock.Lock()
	defer a.fetchLock.Unlock()
	if a.fetched {
		return a.cacheDescribe, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(a.Region.Data)
	ctx := context.Background()
	name := a.Name.Data

	resp, err := svc.DescribeHub(ctx, &sagemaker.DescribeHubInput{
		HubName: &name,
	})
	if err != nil {
		return nil, err
	}
	a.cacheDescribe = resp
	a.fetched = true
	return resp, nil
}

func (a *mqlAwsSagemakerHub) s3StorageConfig() (*mqlAwsSagemakerHubS3StorageConfig, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if resp.S3StorageConfig == nil {
		a.S3StorageConfig.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlRes, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerHubS3StorageConfig,
		map[string]*llx.RawData{
			"s3OutputPath": llx.StringDataPtr(resp.S3StorageConfig.S3OutputPath),
		})
	if err != nil {
		return nil, err
	}
	sc := mqlRes.(*mqlAwsSagemakerHubS3StorageConfig)
	sc.cacheHubArn = a.Arn.Data
	return sc, nil
}

// ---- Hub S3 Storage Config ----

type mqlAwsSagemakerHubS3StorageConfigInternal struct {
	cacheHubArn   string
	cacheKmsKeyId *string
}

func (a *mqlAwsSagemakerHubS3StorageConfig) id() (string, error) {
	return a.cacheHubArn + "/s3StorageConfig", nil
}

func (a *mqlAwsSagemakerHubS3StorageConfig) kmsKey() (*mqlAwsKmsKey, error) {
	if a.cacheKmsKeyId == nil || *a.cacheKmsKeyId == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlKey, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheKmsKeyId)})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

// ---- Hub Content ----

// hubContent is not listed at the root level (aws.sagemaker). It is a resource type
// created by other code (e.g., hub.contents() if added later). We implement id() and
// basic methods here.

func (a *mqlAwsSagemakerHubContent) id() (string, error) {
	return a.Arn.Data, nil
}

// ---- MLflow Tracking Servers ----

func (a *mqlAwsSagemaker) mlflowTrackingServers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getMlflowTrackingServers(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getMlflowTrackingServers(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListMlflowTrackingServersPaginator(svc, &sagemaker.ListMlflowTrackingServersInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS SageMaker MLflow tracking servers")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("SageMaker MLflow tracking servers not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, ts := range page.TrackingServerSummaries {
					mqlTS, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerMlflowTrackingServer,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(ts.TrackingServerArn),
							"name":           llx.StringDataPtr(ts.TrackingServerName),
							"region":         llx.StringData(region),
							"status":         llx.StringData(string(ts.TrackingServerStatus)),
							"createdAt":      llx.TimeDataPtr(ts.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(ts.LastModifiedTime),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlTS)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerMlflowTrackingServerInternal struct {
	fetched       bool
	fetchLock     sync.Mutex
	cacheDescribe *sagemaker.DescribeMlflowTrackingServerOutput
}

func (a *mqlAwsSagemakerMlflowTrackingServer) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerMlflowTrackingServer) fetchDetails() (*sagemaker.DescribeMlflowTrackingServerOutput, error) {
	if a.fetched {
		return a.cacheDescribe, nil
	}
	a.fetchLock.Lock()
	defer a.fetchLock.Unlock()
	if a.fetched {
		return a.cacheDescribe, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(a.Region.Data)
	ctx := context.Background()
	name := a.Name.Data

	resp, err := svc.DescribeMlflowTrackingServer(ctx, &sagemaker.DescribeMlflowTrackingServerInput{
		TrackingServerName: &name,
	})
	if err != nil {
		return nil, err
	}
	a.cacheDescribe = resp
	a.fetched = true
	return resp, nil
}

func (a *mqlAwsSagemakerMlflowTrackingServer) artifactStoreUri() (string, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return "", err
	}
	return convert.ToValue(resp.ArtifactStoreUri), nil
}

func (a *mqlAwsSagemakerMlflowTrackingServer) trackingServerSize() (string, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return "", err
	}
	return string(resp.TrackingServerSize), nil
}

func (a *mqlAwsSagemakerMlflowTrackingServer) trackingServerUrl() (string, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return "", err
	}
	return convert.ToValue(resp.TrackingServerUrl), nil
}

func (a *mqlAwsSagemakerMlflowTrackingServer) iamRole() (*mqlAwsIamRole, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if resp.RoleArn == nil || *resp.RoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(resp.RoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

// ---- Compilation Jobs ----

func (a *mqlAwsSagemaker) compilationJobs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getCompilationJobs(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getCompilationJobs(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListCompilationJobsPaginator(svc, &sagemaker.ListCompilationJobsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS SageMaker compilation jobs")
						return res, nil
					}
					return nil, err
				}

				for _, job := range page.CompilationJobSummaries {
					mqlJob, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerCompilationJob,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(job.CompilationJobArn),
							"name":           llx.StringDataPtr(job.CompilationJobName),
							"region":         llx.StringData(region),
							"status":         llx.StringData(string(job.CompilationJobStatus)),
							"createdAt":      llx.TimeDataPtr(job.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(job.LastModifiedTime),
						})
					if err != nil {
						return nil, err
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

type mqlAwsSagemakerCompilationJobInternal struct {
	fetched       bool
	fetchLock     sync.Mutex
	cacheDescribe *sagemaker.DescribeCompilationJobOutput
}

func (a *mqlAwsSagemakerCompilationJob) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerCompilationJob) fetchDetails() (*sagemaker.DescribeCompilationJobOutput, error) {
	if a.fetched {
		return a.cacheDescribe, nil
	}
	a.fetchLock.Lock()
	defer a.fetchLock.Unlock()
	if a.fetched {
		return a.cacheDescribe, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(a.Region.Data)
	ctx := context.Background()
	name := a.Name.Data

	resp, err := svc.DescribeCompilationJob(ctx, &sagemaker.DescribeCompilationJobInput{
		CompilationJobName: &name,
	})
	if err != nil {
		return nil, err
	}
	a.cacheDescribe = resp
	a.fetched = true
	return resp, nil
}

func (a *mqlAwsSagemakerCompilationJob) inputConfig() (*mqlAwsSagemakerCompilationJobInputConfig, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if resp.InputConfig == nil {
		a.InputConfig.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	ic := resp.InputConfig
	mqlRes, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerCompilationJobInputConfig,
		map[string]*llx.RawData{
			"s3Uri":            llx.StringDataPtr(ic.S3Uri),
			"dataInputConfig":  llx.StringDataPtr(ic.DataInputConfig),
			"framework":        llx.StringData(string(ic.Framework)),
			"frameworkVersion": llx.StringDataPtr(ic.FrameworkVersion),
		})
	if err != nil {
		return nil, err
	}
	icRes := mqlRes.(*mqlAwsSagemakerCompilationJobInputConfig)
	icRes.cacheParentArn = a.Arn.Data
	return icRes, nil
}

func (a *mqlAwsSagemakerCompilationJob) outputConfig() (*mqlAwsSagemakerCompilationJobOutputConfig, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if resp.OutputConfig == nil {
		a.OutputConfig.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	oc := resp.OutputConfig
	// Extract fields via JsonToDict since field names vary by SDK version
	ocDict, _ := convert.JsonToDict(oc)
	var s3Uri, targetDevice string
	var kmsKeyId *string
	var targetPlatform any
	if ocDict != nil {
		if v, ok := ocDict["s3OutputPath"].(string); ok {
			s3Uri = v
		}
		if v, ok := ocDict["targetDevice"].(string); ok {
			targetDevice = v
		}
		if v, ok := ocDict["kmsKeyId"].(string); ok {
			kmsKeyId = &v
		}
		targetPlatform = ocDict["targetPlatform"]
	}

	mqlRes, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerCompilationJobOutputConfig,
		map[string]*llx.RawData{
			"s3Uri":          llx.StringData(s3Uri),
			"targetDevice":   llx.StringData(targetDevice),
			"targetPlatform": llx.DictData(targetPlatform),
		})
	if err != nil {
		return nil, err
	}
	ocRes := mqlRes.(*mqlAwsSagemakerCompilationJobOutputConfig)
	ocRes.cacheParentArn = a.Arn.Data
	ocRes.cacheKmsKeyId = kmsKeyId
	return ocRes, nil
}

func (a *mqlAwsSagemakerCompilationJob) iamRole() (*mqlAwsIamRole, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if resp.RoleArn == nil || *resp.RoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(resp.RoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsSagemakerCompilationJob) vpc() (*mqlAwsVpc, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	var subnetIds []string
	if resp.VpcConfig != nil {
		subnetIds = resp.VpcConfig.Subnets
	}
	return sagemakerResolveVpc(a.MqlRuntime, a.Region.Data, subnetIds, &a.Vpc)
}

func (a *mqlAwsSagemakerCompilationJob) failureReason() (string, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return "", err
	}
	return convert.ToValue(resp.FailureReason), nil
}

// ---- Compilation Job Sub-resources ----

type mqlAwsSagemakerCompilationJobInputConfigInternal struct {
	cacheParentArn string
}

func (a *mqlAwsSagemakerCompilationJobInputConfig) id() (string, error) {
	return a.cacheParentArn + "/inputConfig", nil
}

type mqlAwsSagemakerCompilationJobOutputConfigInternal struct {
	cacheParentArn string
	cacheKmsKeyId  *string
}

func (a *mqlAwsSagemakerCompilationJobOutputConfig) id() (string, error) {
	return a.cacheParentArn + "/outputConfig", nil
}

func (a *mqlAwsSagemakerCompilationJobOutputConfig) kmsKey() (*mqlAwsKmsKey, error) {
	if a.cacheKmsKeyId == nil || *a.cacheKmsKeyId == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlKey, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheKmsKeyId)})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

// ---- Optimization Jobs ----

func (a *mqlAwsSagemaker) optimizationJobs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getOptimizationJobs(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getOptimizationJobs(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListOptimizationJobsPaginator(svc, &sagemaker.ListOptimizationJobsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS SageMaker optimization jobs")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("SageMaker optimization jobs not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, job := range page.OptimizationJobSummaries {
					mqlJob, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerOptimizationJob,
						map[string]*llx.RawData{
							"arn":                    llx.StringDataPtr(job.OptimizationJobArn),
							"name":                   llx.StringDataPtr(job.OptimizationJobName),
							"region":                 llx.StringData(region),
							"status":                 llx.StringData(string(job.OptimizationJobStatus)),
							"createdAt":              llx.TimeDataPtr(job.CreationTime),
							"lastModifiedAt":         llx.TimeDataPtr(job.LastModifiedTime),
							"deploymentInstanceType": llx.StringData(string(job.DeploymentInstanceType)),
						})
					if err != nil {
						return nil, err
					}
					oj := mqlJob.(*mqlAwsSagemakerOptimizationJob)
					oj.cacheOptimizationTypes = job.OptimizationTypes
					res = append(res, mqlJob)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerOptimizationJobInternal struct {
	fetched                bool
	fetchLock              sync.Mutex
	cacheDescribe          *sagemaker.DescribeOptimizationJobOutput
	cacheOptimizationTypes []string
}

func (a *mqlAwsSagemakerOptimizationJob) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerOptimizationJob) fetchDetails() (*sagemaker.DescribeOptimizationJobOutput, error) {
	if a.fetched {
		return a.cacheDescribe, nil
	}
	a.fetchLock.Lock()
	defer a.fetchLock.Unlock()
	if a.fetched {
		return a.cacheDescribe, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(a.Region.Data)
	ctx := context.Background()
	name := a.Name.Data

	resp, err := svc.DescribeOptimizationJob(ctx, &sagemaker.DescribeOptimizationJobInput{
		OptimizationJobName: &name,
	})
	if err != nil {
		return nil, err
	}
	a.cacheDescribe = resp
	a.fetched = true
	return resp, nil
}

func (a *mqlAwsSagemakerOptimizationJob) optimizationConfigs() ([]any, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	res := make([]any, 0, len(resp.OptimizationConfigs))
	for _, cfg := range resp.OptimizationConfigs {
		dict, err := convert.JsonToDict(cfg)
		if err != nil {
			return nil, err
		}
		res = append(res, dict)
	}
	return res, nil
}

func (a *mqlAwsSagemakerOptimizationJob) iamRole() (*mqlAwsIamRole, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if resp.RoleArn == nil || *resp.RoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(resp.RoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsSagemakerOptimizationJob) vpc() (*mqlAwsVpc, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	var subnetIds []string
	if resp.VpcConfig != nil {
		subnetIds = resp.VpcConfig.Subnets
	}
	return sagemakerResolveVpc(a.MqlRuntime, a.Region.Data, subnetIds, &a.Vpc)
}
