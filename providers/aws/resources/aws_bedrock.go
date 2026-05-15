// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsBedrock) id() (string, error) {
	return "aws.bedrock", nil
}

// --- Foundation Models ---

func (a *mqlAwsBedrock) foundationModels() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	// ListFoundationModels returns a global catalog, identical across all regions.
	// Query once from the default region to avoid duplicates.
	svc := conn.Bedrock("")
	ctx := context.Background()

	resp, err := svc.ListFoundationModels(ctx, &bedrock.ListFoundationModelsInput{})
	if err != nil {
		if Is400AccessDeniedError(err) {
			log.Warn().Msg("error accessing bedrock API")
			return nil, nil
		}
		if IsServiceNotAvailableInRegionError(err) {
			log.Debug().Msg("bedrock is not available in the default region")
			return nil, nil
		}
		return nil, err
	}

	res := []any{}
	for _, fm := range resp.ModelSummaries {
		var lifecycleStatus string
		if fm.ModelLifecycle != nil {
			lifecycleStatus = string(fm.ModelLifecycle.Status)
		}

		mqlFM, err := CreateResource(a.MqlRuntime, "aws.bedrock.foundationModel",
			map[string]*llx.RawData{
				"__id":                       llx.StringDataPtr(fm.ModelArn),
				"modelArn":                   llx.StringDataPtr(fm.ModelArn),
				"modelId":                    llx.StringDataPtr(fm.ModelId),
				"modelName":                  llx.StringDataPtr(fm.ModelName),
				"providerName":               llx.StringDataPtr(fm.ProviderName),
				"inputModalities":            llx.ArrayData(enumSliceToAny(fm.InputModalities), types.String),
				"outputModalities":           llx.ArrayData(enumSliceToAny(fm.OutputModalities), types.String),
				"customizationsSupported":    llx.ArrayData(enumSliceToAny(fm.CustomizationsSupported), types.String),
				"inferenceTypesSupported":    llx.ArrayData(enumSliceToAny(fm.InferenceTypesSupported), types.String),
				"responseStreamingSupported": llx.BoolDataPtr(fm.ResponseStreamingSupported),
				"modelLifecycleStatus":       llx.StringData(lifecycleStatus),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlFM)
	}
	return res, nil
}

func (a *mqlAwsBedrockFoundationModel) id() (string, error) {
	return a.ModelArn.Data, nil
}

// enumSliceToAny converts a slice of any enum type (that implements ~string) to []any
func enumSliceToAny[T ~string](s []T) []any {
	res := make([]any, len(s))
	for i, v := range s {
		res[i] = string(v)
	}
	return res
}

// --- Custom Models ---

func (a *mqlAwsBedrock) customModels() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getCustomModels(conn), 5)
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

func (a *mqlAwsBedrock) getCustomModels(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Bedrock(region)
			ctx := context.Background()
			res := []any{}

			paginator := bedrock.NewListCustomModelsPaginator(svc, &bedrock.ListCustomModelsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("bedrock is not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, cm := range page.ModelSummaries {
					mqlCM, err := CreateResource(a.MqlRuntime, "aws.bedrock.customModel",
						map[string]*llx.RawData{
							"__id":              llx.StringDataPtr(cm.ModelArn),
							"modelArn":          llx.StringDataPtr(cm.ModelArn),
							"modelName":         llx.StringDataPtr(cm.ModelName),
							"region":            llx.StringData(region),
							"baseModelArn":      llx.StringDataPtr(cm.BaseModelArn),
							"customizationType": llx.StringData(string(cm.CustomizationType)),
						})
					if err != nil {
						return nil, err
					}
					mqlCMRes := mqlCM.(*mqlAwsBedrockCustomModel)
					mqlCMRes.cacheRegion = region
					res = append(res, mqlCM)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsBedrockCustomModelInternal struct {
	cacheRegion   string
	cacheKmsKeyId *string
	fetched       bool
	lock          sync.Mutex
	detail        *bedrock.GetCustomModelOutput
}

func (a *mqlAwsBedrockCustomModel) id() (string, error) {
	return a.ModelArn.Data, nil
}

func (a *mqlAwsBedrockCustomModel) fetchDetail() (*bedrock.GetCustomModelOutput, error) {
	if a.fetched {
		return a.detail, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.detail, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Bedrock(a.cacheRegion)
	ctx := context.Background()

	modelId := a.ModelArn.Data
	resp, err := svc.GetCustomModel(ctx, &bedrock.GetCustomModelInput{
		ModelIdentifier: &modelId,
	})
	if err != nil {
		return nil, err
	}
	a.cacheKmsKeyId = resp.ModelKmsKeyArn
	a.fetched = true
	a.detail = resp
	return resp, nil
}

func (a *mqlAwsBedrockCustomModel) baseModel() (*mqlAwsBedrockFoundationModel, error) {
	arnVal := a.BaseModelArn.Data
	if arnVal == "" {
		a.BaseModel.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.bedrock.foundationModel",
		map[string]*llx.RawData{"modelArn": llx.StringData(arnVal)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsBedrockFoundationModel), nil
}

func (a *mqlAwsBedrockCustomModel) kmsKey() (*mqlAwsKmsKey, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if resp.ModelKmsKeyArn == nil || *resp.ModelKmsKeyArn == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlKey, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(resp.ModelKmsKeyArn)})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsBedrockCustomModel) trainingDataConfig() (any, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	result, _ := convert.JsonToDict(resp.TrainingDataConfig)
	return result, nil
}

func (a *mqlAwsBedrockCustomModel) outputDataConfig() (any, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	result, _ := convert.JsonToDict(resp.OutputDataConfig)
	return result, nil
}

// --- Guardrails ---

func (a *mqlAwsBedrock) guardrails() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getGuardrails(conn), 5)
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

func (a *mqlAwsBedrock) getGuardrails(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Bedrock(region)
			ctx := context.Background()
			res := []any{}

			paginator := bedrock.NewListGuardrailsPaginator(svc, &bedrock.ListGuardrailsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("bedrock is not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, g := range page.Guardrails {
					mqlG, err := newMqlBedrockGuardrail(a.MqlRuntime, g, region)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlG)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlBedrockGuardrail(runtime *plugin.Runtime, g bedrocktypes.GuardrailSummary, region string) (*mqlAwsBedrockGuardrail, error) {
	res, err := CreateResource(runtime, "aws.bedrock.guardrail",
		map[string]*llx.RawData{
			"__id":    llx.StringDataPtr(g.Arn),
			"arn":     llx.StringDataPtr(g.Arn),
			"id":      llx.StringDataPtr(g.Id),
			"name":    llx.StringDataPtr(g.Name),
			"region":  llx.StringData(region),
			"status":  llx.StringData(string(g.Status)),
			"version": llx.StringDataPtr(g.Version),
		})
	if err != nil {
		return nil, err
	}
	mqlG := res.(*mqlAwsBedrockGuardrail)
	mqlG.cacheRegion = region
	return mqlG, nil
}

type mqlAwsBedrockGuardrailInternal struct {
	cacheRegion string
	fetched     bool
	lock        sync.Mutex
	detail      *bedrock.GetGuardrailOutput
}

func (a *mqlAwsBedrockGuardrail) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsBedrockGuardrail) fetchDetail() (*bedrock.GetGuardrailOutput, error) {
	if a.fetched {
		return a.detail, nil
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.fetched {
		return a.detail, nil
	}

	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Bedrock(a.cacheRegion)
	ctx := context.Background()

	guardrailId := a.Id.Data
	resp, err := svc.GetGuardrail(ctx, &bedrock.GetGuardrailInput{
		GuardrailIdentifier: &guardrailId,
	})
	if err != nil {
		return nil, err
	}
	a.fetched = true
	a.detail = resp
	return resp, nil
}

func (a *mqlAwsBedrockGuardrail) kmsKey() (*mqlAwsKmsKey, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if resp.KmsKeyArn == nil || *resp.KmsKeyArn == "" {
		a.KmsKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlKey, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(resp.KmsKeyArn)})
	if err != nil {
		return nil, err
	}
	return mqlKey.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsBedrockGuardrail) contentPolicy() (any, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	result, _ := convert.JsonToDict(resp.ContentPolicy)
	return result, nil
}

func (a *mqlAwsBedrockGuardrail) sensitiveInformationPolicy() (any, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	result, _ := convert.JsonToDict(resp.SensitiveInformationPolicy)
	return result, nil
}

func (a *mqlAwsBedrockGuardrail) topicPolicy() (any, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	result, _ := convert.JsonToDict(resp.TopicPolicy)
	return result, nil
}

func (a *mqlAwsBedrockGuardrail) wordPolicy() (any, error) {
	resp, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	result, _ := convert.JsonToDict(resp.WordPolicy)
	return result, nil
}

// --- Model Invocation Logging Configurations ---

func (a *mqlAwsBedrock) modelInvocationLoggingConfigurations() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getModelInvocationLoggingConfigurations(conn), 5)
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

func (a *mqlAwsBedrock) getModelInvocationLoggingConfigurations(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Bedrock(region)
			ctx := context.Background()
			res := []any{}

			resp, err := svc.GetModelInvocationLoggingConfiguration(ctx, &bedrock.GetModelInvocationLoggingConfigurationInput{})
			if err != nil {
				if Is400AccessDeniedError(err) {
					log.Warn().Str("region", region).Msg("error accessing region for AWS API")
					return res, nil
				}
				if IsServiceNotAvailableInRegionError(err) {
					log.Debug().Str("region", region).Msg("bedrock is not available in region")
					return res, nil
				}
				return nil, err
			}

			if resp.LoggingConfig != nil {
				lc := resp.LoggingConfig
				cwConfig, _ := convert.JsonToDict(lc.CloudWatchConfig)
				s3Config, _ := convert.JsonToDict(lc.S3Config)

				mqlLC, err := CreateResource(a.MqlRuntime, "aws.bedrock.modelInvocationLoggingConfiguration",
					map[string]*llx.RawData{
						"__id":                         llx.StringData("aws.bedrock.modelInvocationLoggingConfiguration/" + region),
						"region":                       llx.StringData(region),
						"cloudWatchConfig":             llx.DictData(cwConfig),
						"s3Config":                     llx.DictData(s3Config),
						"textDataDeliveryEnabled":      llx.BoolDataPtr(lc.TextDataDeliveryEnabled),
						"imageDataDeliveryEnabled":     llx.BoolDataPtr(lc.ImageDataDeliveryEnabled),
						"embeddingDataDeliveryEnabled": llx.BoolDataPtr(lc.EmbeddingDataDeliveryEnabled),
						"videoDataDeliveryEnabled":     llx.BoolDataPtr(lc.VideoDataDeliveryEnabled),
					})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlLC)
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsBedrockModelInvocationLoggingConfiguration) id() (string, error) {
	return "aws.bedrock.modelInvocationLoggingConfiguration/" + a.Region.Data, nil
}

// --- Provisioned Model Throughputs ---

func (a *mqlAwsBedrock) provisionedModelThroughputs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getProvisionedModelThroughputs(conn), 5)
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

func (a *mqlAwsBedrock) getProvisionedModelThroughputs(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Bedrock(region)
			ctx := context.Background()
			res := []any{}

			paginator := bedrock.NewListProvisionedModelThroughputsPaginator(svc, &bedrock.ListProvisionedModelThroughputsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("bedrock is not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, pt := range page.ProvisionedModelSummaries {
					mqlPT, err := CreateResource(a.MqlRuntime, "aws.bedrock.provisionedModelThroughput",
						map[string]*llx.RawData{
							"__id":                 llx.StringDataPtr(pt.ProvisionedModelArn),
							"provisionedModelArn":  llx.StringDataPtr(pt.ProvisionedModelArn),
							"provisionedModelName": llx.StringDataPtr(pt.ProvisionedModelName),
							"region":               llx.StringData(region),
							"modelArn":             llx.StringDataPtr(pt.ModelArn),
							"foundationModelArn":   llx.StringDataPtr(pt.FoundationModelArn),
							"modelUnits":           llx.IntData(int64(convert.ToValue(pt.DesiredModelUnits))),
							"status":               llx.StringData(string(pt.Status)),
							"commitmentDuration":   llx.StringData(string(pt.CommitmentDuration)),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlPT)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsBedrockProvisionedModelThroughput) id() (string, error) {
	return a.ProvisionedModelArn.Data, nil
}

func (a *mqlAwsBedrockProvisionedModelThroughput) foundationModel() (*mqlAwsBedrockFoundationModel, error) {
	arnVal := a.FoundationModelArn.Data
	if arnVal == "" {
		a.FoundationModel.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.bedrock.foundationModel",
		map[string]*llx.RawData{"modelArn": llx.StringData(arnVal)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsBedrockFoundationModel), nil
}

func (a *mqlAwsBedrock) advancedPromptOptimizationJobs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getAdvancedPromptOptimizationJobs(conn), 5)
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

func (a *mqlAwsBedrock) getAdvancedPromptOptimizationJobs(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.Bedrock(region)
			ctx := context.Background()
			res := []any{}

			paginator := bedrock.NewListAdvancedPromptOptimizationJobsPaginator(svc, &bedrock.ListAdvancedPromptOptimizationJobsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("bedrock is not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, j := range page.JobSummaries {
					mqlJob, err := CreateResource(a.MqlRuntime, "aws.bedrock.advancedPromptOptimizationJob",
						map[string]*llx.RawData{
							"__id":           llx.StringDataPtr(j.JobArn),
							"arn":            llx.StringDataPtr(j.JobArn),
							"name":           llx.StringDataPtr(j.JobName),
							"region":         llx.StringData(region),
							"status":         llx.StringData(string(j.JobStatus)),
							"createdAt":      llx.TimeDataPtr(j.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(j.LastModifiedTime),
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

func (a *mqlAwsBedrockAdvancedPromptOptimizationJob) id() (string, error) {
	return a.Arn.Data, nil
}

type mqlAwsBedrockAdvancedPromptOptimizationJobInternal struct {
	detailOnce sync.Once
	detailErr  error
	detail     *bedrock.GetAdvancedPromptOptimizationJobOutput
}

func (a *mqlAwsBedrockAdvancedPromptOptimizationJob) fetchDetail() (*bedrock.GetAdvancedPromptOptimizationJobOutput, error) {
	a.detailOnce.Do(func() {
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		svc := conn.Bedrock(a.Region.Data)
		ctx := context.Background()
		jobIdentifier := a.Arn.Data
		a.detail, a.detailErr = svc.GetAdvancedPromptOptimizationJob(ctx, &bedrock.GetAdvancedPromptOptimizationJobInput{
			JobIdentifier: &jobIdentifier,
		})
	})
	return a.detail, a.detailErr
}

func (a *mqlAwsBedrockAdvancedPromptOptimizationJob) description() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	return convert.ToValue(detail.JobDescription), nil
}

func (a *mqlAwsBedrockAdvancedPromptOptimizationJob) inputS3Uri() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail.InputConfig == nil {
		return "", nil
	}
	return convert.ToValue(detail.InputConfig.S3Uri), nil
}

func (a *mqlAwsBedrockAdvancedPromptOptimizationJob) outputS3Uri() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail.OutputConfig == nil {
		return "", nil
	}
	return convert.ToValue(detail.OutputConfig.S3Uri), nil
}

func (a *mqlAwsBedrockAdvancedPromptOptimizationJob) encryptionKey() (*mqlAwsKmsKey, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail.EncryptionKeyArn == nil || *detail.EncryptionKeyArn == "" {
		a.EncryptionKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(detail.EncryptionKeyArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsBedrockAdvancedPromptOptimizationJob) failureMessage() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	return convert.ToValue(detail.FailureMessage), nil
}

func (a *mqlAwsBedrockAdvancedPromptOptimizationJob) modelConfigurations() ([]any, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	return convert.JsonToDictSlice(detail.ModelConfigurations)
}
