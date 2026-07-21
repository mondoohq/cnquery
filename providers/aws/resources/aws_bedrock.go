// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagent"
	bedrockagenttypes "github.com/aws/aws-sdk-go-v2/service/bedrockagent/types"
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

// --- Bedrock Agents ---

func (a *mqlAwsBedrock) agents() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getAgents(conn), 5)
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

func (a *mqlAwsBedrock) getAgents(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.BedrockAgent(region)
			ctx := context.Background()
			res := []any{}

			paginator := bedrockagent.NewListAgentsPaginator(svc, &bedrockagent.ListAgentsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("bedrock agents is not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, ag := range page.AgentSummaries {
					compositeID := "aws.bedrock.agent/" + region + "/" + convert.ToValue(ag.AgentId)
					mqlAg, err := CreateResource(a.MqlRuntime, "aws.bedrock.agent",
						map[string]*llx.RawData{
							"__id":   llx.StringData(compositeID),
							"id":     llx.StringDataPtr(ag.AgentId),
							"name":   llx.StringDataPtr(ag.AgentName),
							"region": llx.StringData(region),
							"status": llx.StringData(string(ag.AgentStatus)),
						})
					if err != nil {
						return nil, err
					}
					mqlAgRes := mqlAg.(*mqlAwsBedrockAgent)
					mqlAgRes.cacheRegion = region
					res = append(res, mqlAg)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsBedrockAgentInternal struct {
	cacheRegion string
	detailOnce  sync.Once
	detailErr   error
	detail      *bedrockagent.GetAgentOutput
}

func (a *mqlAwsBedrockAgent) id() (string, error) {
	return "aws.bedrock.agent/" + a.Region.Data + "/" + a.Id.Data, nil
}

func (a *mqlAwsBedrockAgent) fetchDetail() (*bedrockagent.GetAgentOutput, error) {
	a.detailOnce.Do(func() {
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		svc := conn.BedrockAgent(a.cacheRegion)
		ctx := context.Background()
		agentId := a.Id.Data
		a.detail, a.detailErr = svc.GetAgent(ctx, &bedrockagent.GetAgentInput{AgentId: &agentId})
	})
	return a.detail, a.detailErr
}

func (a *mqlAwsBedrockAgent) arn() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil || detail.Agent == nil {
		return "", nil
	}
	return convert.ToValue(detail.Agent.AgentArn), nil
}

func (a *mqlAwsBedrockAgent) description() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil || detail.Agent == nil {
		return "", nil
	}
	return convert.ToValue(detail.Agent.Description), nil
}

func (a *mqlAwsBedrockAgent) foundationModel() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil || detail.Agent == nil {
		return "", nil
	}
	return convert.ToValue(detail.Agent.FoundationModel), nil
}

func (a *mqlAwsBedrockAgent) instruction() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil || detail.Agent == nil {
		return "", nil
	}
	return convert.ToValue(detail.Agent.Instruction), nil
}

func (a *mqlAwsBedrockAgent) idleSessionTtl() (int64, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return 0, err
	}
	if detail == nil || detail.Agent == nil || detail.Agent.IdleSessionTTLInSeconds == nil {
		return 0, nil
	}
	return int64(*detail.Agent.IdleSessionTTLInSeconds), nil
}

func (a *mqlAwsBedrockAgent) agentResourceRoleArn() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil || detail.Agent == nil {
		return "", nil
	}
	return convert.ToValue(detail.Agent.AgentResourceRoleArn), nil
}

func (a *mqlAwsBedrockAgent) iamRole() (*mqlAwsIamRole, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.Agent == nil || detail.Agent.AgentResourceRoleArn == nil || *detail.Agent.AgentResourceRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(detail.Agent.AgentResourceRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsBedrockAgent) customerEncryptionKey() (*mqlAwsKmsKey, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.Agent == nil || detail.Agent.CustomerEncryptionKeyArn == nil || *detail.Agent.CustomerEncryptionKeyArn == "" {
		a.CustomerEncryptionKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(detail.Agent.CustomerEncryptionKeyArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsBedrockAgent) createdAt() (*time.Time, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.Agent == nil {
		return nil, nil
	}
	return detail.Agent.CreatedAt, nil
}

func (a *mqlAwsBedrockAgent) updatedAt() (*time.Time, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.Agent == nil {
		return nil, nil
	}
	return detail.Agent.UpdatedAt, nil
}

func (a *mqlAwsBedrockAgent) actionGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.BedrockAgent(a.cacheRegion)
	ctx := context.Background()
	agentId := a.Id.Data
	draft := "DRAFT"
	res := []any{}
	paginator := bedrockagent.NewListAgentActionGroupsPaginator(svc, &bedrockagent.ListAgentActionGroupsInput{
		AgentId:      &agentId,
		AgentVersion: &draft,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				return res, nil
			}
			return nil, err
		}
		summaries, err := convert.JsonToDictSlice(page.ActionGroupSummaries)
		if err != nil {
			return nil, err
		}
		res = append(res, summaries...)
	}
	return res, nil
}

func (a *mqlAwsBedrockAgent) knowledgeBases() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.BedrockAgent(a.cacheRegion)
	ctx := context.Background()
	agentId := a.Id.Data
	draft := "DRAFT"
	res := []any{}
	paginator := bedrockagent.NewListAgentKnowledgeBasesPaginator(svc, &bedrockagent.ListAgentKnowledgeBasesInput{
		AgentId:      &agentId,
		AgentVersion: &draft,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				return res, nil
			}
			return nil, err
		}
		summaries, err := convert.JsonToDictSlice(page.AgentKnowledgeBaseSummaries)
		if err != nil {
			return nil, err
		}
		res = append(res, summaries...)
	}
	return res, nil
}

// --- Bedrock Knowledge Bases ---

func (a *mqlAwsBedrock) knowledgeBases() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getKnowledgeBases(conn), 5)
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

func (a *mqlAwsBedrock) getKnowledgeBases(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.BedrockAgent(region)
			ctx := context.Background()
			res := []any{}

			paginator := bedrockagent.NewListKnowledgeBasesPaginator(svc, &bedrockagent.ListKnowledgeBasesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("bedrock agents is not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, kb := range page.KnowledgeBaseSummaries {
					compositeID := "aws.bedrock.knowledgeBase/" + region + "/" + convert.ToValue(kb.KnowledgeBaseId)
					mqlKB, err := CreateResource(a.MqlRuntime, "aws.bedrock.knowledgeBase",
						map[string]*llx.RawData{
							"__id":   llx.StringData(compositeID),
							"id":     llx.StringDataPtr(kb.KnowledgeBaseId),
							"name":   llx.StringDataPtr(kb.Name),
							"region": llx.StringData(region),
							"status": llx.StringData(string(kb.Status)),
						})
					if err != nil {
						return nil, err
					}
					mqlKBRes := mqlKB.(*mqlAwsBedrockKnowledgeBase)
					mqlKBRes.cacheRegion = region
					res = append(res, mqlKB)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsBedrockKnowledgeBaseInternal struct {
	cacheRegion string
	detailOnce  sync.Once
	detailErr   error
	detail      *bedrockagent.GetKnowledgeBaseOutput
}

func (a *mqlAwsBedrockKnowledgeBase) id() (string, error) {
	return "aws.bedrock.knowledgeBase/" + a.Region.Data + "/" + a.Id.Data, nil
}

func (a *mqlAwsBedrockKnowledgeBase) fetchDetail() (*bedrockagent.GetKnowledgeBaseOutput, error) {
	a.detailOnce.Do(func() {
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		svc := conn.BedrockAgent(a.cacheRegion)
		ctx := context.Background()
		kbId := a.Id.Data
		a.detail, a.detailErr = svc.GetKnowledgeBase(ctx, &bedrockagent.GetKnowledgeBaseInput{KnowledgeBaseId: &kbId})
	})
	return a.detail, a.detailErr
}

func (a *mqlAwsBedrockKnowledgeBase) arn() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil || detail.KnowledgeBase == nil {
		return "", nil
	}
	return convert.ToValue(detail.KnowledgeBase.KnowledgeBaseArn), nil
}

func (a *mqlAwsBedrockKnowledgeBase) description() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil || detail.KnowledgeBase == nil {
		return "", nil
	}
	return convert.ToValue(detail.KnowledgeBase.Description), nil
}

func (a *mqlAwsBedrockKnowledgeBase) roleArn() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil || detail.KnowledgeBase == nil {
		return "", nil
	}
	return convert.ToValue(detail.KnowledgeBase.RoleArn), nil
}

func (a *mqlAwsBedrockKnowledgeBase) iamRole() (*mqlAwsIamRole, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.KnowledgeBase == nil || detail.KnowledgeBase.RoleArn == nil || *detail.KnowledgeBase.RoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(detail.KnowledgeBase.RoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsBedrockKnowledgeBase) storageConfiguration() (any, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.KnowledgeBase == nil {
		return nil, nil
	}
	result, _ := convert.JsonToDict(detail.KnowledgeBase.StorageConfiguration)
	return result, nil
}

func (a *mqlAwsBedrockKnowledgeBase) knowledgeBaseConfiguration() (any, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.KnowledgeBase == nil {
		return nil, nil
	}
	result, _ := convert.JsonToDict(detail.KnowledgeBase.KnowledgeBaseConfiguration)
	return result, nil
}

func (a *mqlAwsBedrockKnowledgeBase) createdAt() (*time.Time, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.KnowledgeBase == nil {
		return nil, nil
	}
	return detail.KnowledgeBase.CreatedAt, nil
}

func (a *mqlAwsBedrockKnowledgeBase) updatedAt() (*time.Time, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.KnowledgeBase == nil {
		return nil, nil
	}
	return detail.KnowledgeBase.UpdatedAt, nil
}

func (a *mqlAwsBedrockKnowledgeBase) dataSources() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.BedrockAgent(a.cacheRegion)
	ctx := context.Background()
	kbId := a.Id.Data
	res := []any{}
	paginator := bedrockagent.NewListDataSourcesPaginator(svc, &bedrockagent.ListDataSourcesInput{
		KnowledgeBaseId: &kbId,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				return res, nil
			}
			return nil, err
		}
		summaries, err := convert.JsonToDictSlice(page.DataSourceSummaries)
		if err != nil {
			return nil, err
		}
		res = append(res, summaries...)
	}
	return res, nil
}

// --- Bedrock Flows ---

func (a *mqlAwsBedrock) flows() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getFlows(conn), 5)
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

func (a *mqlAwsBedrock) getFlows(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			svc := conn.BedrockAgent(region)
			ctx := context.Background()
			res := []any{}

			paginator := bedrockagent.NewListFlowsPaginator(svc, &bedrockagent.ListFlowsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					if IsServiceNotAvailableInRegionError(err) {
						log.Debug().Str("region", region).Msg("bedrock agents is not available in region")
						return res, nil
					}
					return nil, err
				}

				for _, fl := range page.FlowSummaries {
					mqlFL, err := newMqlBedrockFlow(a.MqlRuntime, fl, region)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlFL)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func newMqlBedrockFlow(runtime *plugin.Runtime, fl bedrockagenttypes.FlowSummary, region string) (*mqlAwsBedrockFlow, error) {
	res, err := CreateResource(runtime, "aws.bedrock.flow",
		map[string]*llx.RawData{
			"__id":      llx.StringDataPtr(fl.Arn),
			"id":        llx.StringDataPtr(fl.Id),
			"arn":       llx.StringDataPtr(fl.Arn),
			"name":      llx.StringDataPtr(fl.Name),
			"region":    llx.StringData(region),
			"status":    llx.StringData(string(fl.Status)),
			"createdAt": llx.TimeDataPtr(fl.CreatedAt),
			"updatedAt": llx.TimeDataPtr(fl.UpdatedAt),
		})
	if err != nil {
		return nil, err
	}
	mqlFL := res.(*mqlAwsBedrockFlow)
	mqlFL.cacheRegion = region
	return mqlFL, nil
}

type mqlAwsBedrockFlowInternal struct {
	cacheRegion string
	detailOnce  sync.Once
	detailErr   error
	detail      *bedrockagent.GetFlowOutput
}

func (a *mqlAwsBedrockFlow) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsBedrockFlow) fetchDetail() (*bedrockagent.GetFlowOutput, error) {
	a.detailOnce.Do(func() {
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		svc := conn.BedrockAgent(a.cacheRegion)
		ctx := context.Background()
		flowId := a.Id.Data
		a.detail, a.detailErr = svc.GetFlow(ctx, &bedrockagent.GetFlowInput{FlowIdentifier: &flowId})
	})
	return a.detail, a.detailErr
}

func (a *mqlAwsBedrockFlow) version() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil {
		return "", nil
	}
	return convert.ToValue(detail.Version), nil
}

func (a *mqlAwsBedrockFlow) description() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil {
		return "", nil
	}
	return convert.ToValue(detail.Description), nil
}

func (a *mqlAwsBedrockFlow) executionRoleArn() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil {
		return "", nil
	}
	return convert.ToValue(detail.ExecutionRoleArn), nil
}

func (a *mqlAwsBedrockFlow) iamRole() (*mqlAwsIamRole, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.ExecutionRoleArn == nil || *detail.ExecutionRoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(detail.ExecutionRoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsBedrockFlow) customerEncryptionKey() (*mqlAwsKmsKey, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.CustomerEncryptionKeyArn == nil || *detail.CustomerEncryptionKeyArn == "" {
		a.CustomerEncryptionKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(detail.CustomerEncryptionKeyArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsKmsKey), nil
}

// --- Bedrock Evaluation Jobs ---

func (a *mqlAwsBedrock) evaluationJobs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getEvaluationJobs(conn), 5)
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

func (a *mqlAwsBedrock) getEvaluationJobs(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := bedrock.NewListEvaluationJobsPaginator(svc, &bedrock.ListEvaluationJobsInput{})
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
					mqlJob, err := newMqlBedrockEvaluationJob(a.MqlRuntime, j, region)
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

func newMqlBedrockEvaluationJob(runtime *plugin.Runtime, j bedrocktypes.EvaluationSummary, region string) (*mqlAwsBedrockEvaluationJob, error) {
	modelIds := []any{}
	if j.InferenceConfigSummary != nil {
		if j.InferenceConfigSummary.ModelConfigSummary != nil {
			for _, id := range j.InferenceConfigSummary.ModelConfigSummary.BedrockModelIdentifiers {
				modelIds = append(modelIds, id)
			}
			for _, id := range j.InferenceConfigSummary.ModelConfigSummary.PrecomputedInferenceSourceIdentifiers {
				modelIds = append(modelIds, id)
			}
		}
		if j.InferenceConfigSummary.RagConfigSummary != nil {
			for _, id := range j.InferenceConfigSummary.RagConfigSummary.BedrockKnowledgeBaseIdentifiers {
				modelIds = append(modelIds, id)
			}
			for _, id := range j.InferenceConfigSummary.RagConfigSummary.PrecomputedRagSourceIdentifiers {
				modelIds = append(modelIds, id)
			}
		}
	}
	res, err := CreateResource(runtime, "aws.bedrock.evaluationJob",
		map[string]*llx.RawData{
			"__id":             llx.StringDataPtr(j.JobArn),
			"jobArn":           llx.StringDataPtr(j.JobArn),
			"jobName":          llx.StringDataPtr(j.JobName),
			"region":           llx.StringData(region),
			"status":           llx.StringData(string(j.Status)),
			"jobType":          llx.StringData(string(j.JobType)),
			"applicationType":  llx.StringData(string(j.ApplicationType)),
			"modelIdentifiers": llx.ArrayData(modelIds, types.String),
			"createdAt":        llx.TimeDataPtr(j.CreationTime),
		})
	if err != nil {
		return nil, err
	}
	mqlJob := res.(*mqlAwsBedrockEvaluationJob)
	mqlJob.cacheRegion = region
	return mqlJob, nil
}

type mqlAwsBedrockEvaluationJobInternal struct {
	cacheRegion string
	detailOnce  sync.Once
	detailErr   error
	detail      *bedrock.GetEvaluationJobOutput
}

func (a *mqlAwsBedrockEvaluationJob) id() (string, error) {
	return a.JobArn.Data, nil
}

func (a *mqlAwsBedrockEvaluationJob) fetchDetail() (*bedrock.GetEvaluationJobOutput, error) {
	a.detailOnce.Do(func() {
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		svc := conn.Bedrock(a.cacheRegion)
		ctx := context.Background()
		jobArn := a.JobArn.Data
		a.detail, a.detailErr = svc.GetEvaluationJob(ctx, &bedrock.GetEvaluationJobInput{JobIdentifier: &jobArn})
	})
	return a.detail, a.detailErr
}

func (a *mqlAwsBedrockEvaluationJob) roleArn() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil {
		return "", nil
	}
	return convert.ToValue(detail.RoleArn), nil
}

func (a *mqlAwsBedrockEvaluationJob) iamRole() (*mqlAwsIamRole, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.RoleArn == nil || *detail.RoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(detail.RoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsBedrockEvaluationJob) customerEncryptionKey() (*mqlAwsKmsKey, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.CustomerEncryptionKeyId == nil || *detail.CustomerEncryptionKeyId == "" {
		a.CustomerEncryptionKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(detail.CustomerEncryptionKeyId)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsKmsKey), nil
}

func (a *mqlAwsBedrockEvaluationJob) jobDescription() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil {
		return "", nil
	}
	return convert.ToValue(detail.JobDescription), nil
}

// --- Bedrock Model Import Jobs ---

func (a *mqlAwsBedrock) modelImportJobs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getModelImportJobs(conn), 5)
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

func (a *mqlAwsBedrock) getModelImportJobs(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := bedrock.NewListModelImportJobsPaginator(svc, &bedrock.ListModelImportJobsInput{})
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

				for _, j := range page.ModelImportJobSummaries {
					mqlJob, err := CreateResource(a.MqlRuntime, "aws.bedrock.modelImportJob",
						map[string]*llx.RawData{
							"__id":             llx.StringDataPtr(j.JobArn),
							"jobArn":           llx.StringDataPtr(j.JobArn),
							"jobName":          llx.StringDataPtr(j.JobName),
							"region":           llx.StringData(region),
							"status":           llx.StringData(string(j.Status)),
							"importedModelArn": llx.StringDataPtr(j.ImportedModelArn),
							"createdAt":        llx.TimeDataPtr(j.CreationTime),
							"lastModifiedTime": llx.TimeDataPtr(j.LastModifiedTime),
							"endTime":          llx.TimeDataPtr(j.EndTime),
						})
					if err != nil {
						return nil, err
					}
					mqlJobRes := mqlJob.(*mqlAwsBedrockModelImportJob)
					mqlJobRes.cacheRegion = region
					res = append(res, mqlJob)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsBedrockModelImportJobInternal struct {
	cacheRegion string
	detailOnce  sync.Once
	detailErr   error
	detail      *bedrock.GetModelImportJobOutput
}

func (a *mqlAwsBedrockModelImportJob) id() (string, error) {
	return a.JobArn.Data, nil
}

func (a *mqlAwsBedrockModelImportJob) fetchDetail() (*bedrock.GetModelImportJobOutput, error) {
	a.detailOnce.Do(func() {
		conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
		svc := conn.Bedrock(a.cacheRegion)
		ctx := context.Background()
		jobArn := a.JobArn.Data
		a.detail, a.detailErr = svc.GetModelImportJob(ctx, &bedrock.GetModelImportJobInput{JobIdentifier: &jobArn})
	})
	return a.detail, a.detailErr
}

func (a *mqlAwsBedrockModelImportJob) modelDataSource() (any, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return nil, nil
	}
	result, _ := convert.JsonToDict(detail.ModelDataSource)
	return result, nil
}

func (a *mqlAwsBedrockModelImportJob) roleArn() (string, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return "", err
	}
	if detail == nil {
		return "", nil
	}
	return convert.ToValue(detail.RoleArn), nil
}

func (a *mqlAwsBedrockModelImportJob) iamRole() (*mqlAwsIamRole, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.RoleArn == nil || *detail.RoleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(detail.RoleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

func (a *mqlAwsBedrockModelImportJob) customerEncryptionKey() (*mqlAwsKmsKey, error) {
	detail, err := a.fetchDetail()
	if err != nil {
		return nil, err
	}
	if detail == nil || detail.ImportedModelKmsKeyArn == nil || *detail.ImportedModelKmsKeyArn == "" {
		a.CustomerEncryptionKey.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.kms.key",
		map[string]*llx.RawData{"arn": llx.StringDataPtr(detail.ImportedModelKmsKeyArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsKmsKey), nil
}

// --- Bedrock Batch Inference Jobs ---

func (a *mqlAwsBedrock) batchInferenceJobs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getBatchInferenceJobs(conn), 5)
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

func (a *mqlAwsBedrock) getBatchInferenceJobs(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := bedrock.NewListModelInvocationJobsPaginator(svc, &bedrock.ListModelInvocationJobsInput{})
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

				for _, j := range page.InvocationJobSummaries {
					inputCfg, _ := convert.JsonToDict(j.InputDataConfig)
					outputCfg, _ := convert.JsonToDict(j.OutputDataConfig)

					mqlJob, err := CreateResource(a.MqlRuntime, "aws.bedrock.batchInferenceJob",
						map[string]*llx.RawData{
							"__id":                   llx.StringDataPtr(j.JobArn),
							"jobArn":                 llx.StringDataPtr(j.JobArn),
							"jobName":                llx.StringDataPtr(j.JobName),
							"region":                 llx.StringData(region),
							"modelId":                llx.StringDataPtr(j.ModelId),
							"status":                 llx.StringData(string(j.Status)),
							"inputDataConfig":        llx.DictData(inputCfg),
							"outputDataConfig":       llx.DictData(outputCfg),
							"roleArn":                llx.StringDataPtr(j.RoleArn),
							"submitTime":             llx.TimeDataPtr(j.SubmitTime),
							"lastModifiedTime":       llx.TimeDataPtr(j.LastModifiedTime),
							"endTime":                llx.TimeDataPtr(j.EndTime),
							"message":                llx.StringDataPtr(j.Message),
							"timeoutDurationInHours": llx.IntData(int64(convert.ToValue(j.TimeoutDurationInHours))),
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

func (a *mqlAwsBedrockBatchInferenceJob) id() (string, error) {
	return a.JobArn.Data, nil
}

func (a *mqlAwsBedrockBatchInferenceJob) iamRole() (*mqlAwsIamRole, error) {
	roleArn := a.RoleArn.Data
	if roleArn == "" {
		a.IamRole.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.iam.role",
		map[string]*llx.RawData{"arn": llx.StringData(roleArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsIamRole), nil
}

// bedrockResourcePolicyJSON fetches the resource-based policy document for a
// Bedrock resource identified by its ARN. Bedrock returns an empty policy for
// resources with no policy attached; access-denied is treated as "no policy"
// so a scan without bedrock:GetResourcePolicy degrades gracefully. Shared by
// the resourcePolicy() accessors on the cross-account-shareable resources.
func bedrockResourcePolicyJSON(runtime *plugin.Runtime, region, arn string) (string, error) {
	conn := runtime.Connection.(*connection.AwsConnection)
	svc := conn.Bedrock(region)
	resp, err := svc.GetResourcePolicy(context.Background(), &bedrock.GetResourcePolicyInput{ResourceArn: &arn})
	if err != nil {
		// A resource with no attached policy, or a resource type that does not
		// support resource-based policies at all (system-defined inference
		// profiles return ValidationException "operation is not recognized"),
		// simply has no policy to report - degrade to empty rather than failing
		// the whole collection.
		if Is400AccessDeniedError(err) || isResourceNotFoundError(err) || isOperationNotSupportedError(err) {
			return "", nil
		}
		return "", err
	}
	return convert.ToValue(resp.ResourcePolicy), nil
}

func (a *mqlAwsBedrockCustomModel) resourcePolicy() (string, error) {
	return bedrockResourcePolicyJSON(a.MqlRuntime, a.Region.Data, a.ModelArn.Data)
}

func (a *mqlAwsBedrockCustomModel) policyStatements() ([]any, error) {
	policy := a.GetResourcePolicy()
	if policy.Error != nil {
		return nil, policy.Error
	}
	return newPolicyStatementResources(a.MqlRuntime, a.ModelArn.Data, policy.Data)
}

func (a *mqlAwsBedrockCustomModel) isPublic() (bool, error) {
	return resourceIsPublic(a.GetPolicyStatements())
}

func (a *mqlAwsBedrockImportedModel) resourcePolicy() (string, error) {
	return bedrockResourcePolicyJSON(a.MqlRuntime, a.Region.Data, a.Arn.Data)
}

func (a *mqlAwsBedrockImportedModel) policyStatements() ([]any, error) {
	policy := a.GetResourcePolicy()
	if policy.Error != nil {
		return nil, policy.Error
	}
	return newPolicyStatementResources(a.MqlRuntime, a.Arn.Data, policy.Data)
}

func (a *mqlAwsBedrockImportedModel) isPublic() (bool, error) {
	return resourceIsPublic(a.GetPolicyStatements())
}

func (a *mqlAwsBedrockEvaluationJob) resourcePolicy() (string, error) {
	return bedrockResourcePolicyJSON(a.MqlRuntime, a.Region.Data, a.JobArn.Data)
}

func (a *mqlAwsBedrockEvaluationJob) policyStatements() ([]any, error) {
	policy := a.GetResourcePolicy()
	if policy.Error != nil {
		return nil, policy.Error
	}
	return newPolicyStatementResources(a.MqlRuntime, a.JobArn.Data, policy.Data)
}

func (a *mqlAwsBedrockEvaluationJob) isPublic() (bool, error) {
	return resourceIsPublic(a.GetPolicyStatements())
}

func (a *mqlAwsBedrockInferenceProfile) resourcePolicy() (string, error) {
	return bedrockResourcePolicyJSON(a.MqlRuntime, a.Region.Data, a.Arn.Data)
}

func (a *mqlAwsBedrockInferenceProfile) policyStatements() ([]any, error) {
	policy := a.GetResourcePolicy()
	if policy.Error != nil {
		return nil, policy.Error
	}
	return newPolicyStatementResources(a.MqlRuntime, a.Arn.Data, policy.Data)
}

func (a *mqlAwsBedrockInferenceProfile) isPublic() (bool, error) {
	return resourceIsPublic(a.GetPolicyStatements())
}
