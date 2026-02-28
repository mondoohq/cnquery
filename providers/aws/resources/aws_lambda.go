// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/json"
	"maps"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/smithy-go/transport/http"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/providers/aws/resources/awspolicy"
	"go.mondoo.com/mql/v13/types"
)

func (a *mqlAwsLambda) id() (string, error) {
	return "aws.lambda", nil
}

func (a *mqlAwsLambda) functions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getFunctions(conn), 5)
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

func (a *mqlAwsLambda) getFunctions(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("lambda>getFunctions>calling aws with region %s", region)

			svc := conn.Lambda(region)
			ctx := context.Background()
			res := []any{}
			params := &lambda.ListFunctionsInput{}
			paginator := lambda.NewListFunctionsPaginator(svc, params)
			for paginator.HasMorePages() {
				functionsResp, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, errors.Wrap(err, "could not gather aws lambda functions")
				}
				for _, function := range functionsResp.Functions {
					vpcConfigJson, err := convert.JsonToDict(function.VpcConfig)
					if err != nil {
						return nil, err
					}
					var dlqTarget string
					if function.DeadLetterConfig != nil {
						dlqTarget = convert.ToValue(function.DeadLetterConfig.TargetArn)
					}
					tags := make(map[string]string)
					tagsResp, err := svc.ListTags(ctx, &lambda.ListTagsInput{Resource: function.FunctionArn})
					if err == nil {
						maps.Copy(tags, tagsResp.Tags)
					}

					if conn.Filters.General.IsFilteredOutByTags(tags) {
						log.Debug().Interface("function", function.FunctionArn).Msg("excluding function due to filters")
						continue
					}

					// Convert architectures to []any
					architectures := make([]any, len(function.Architectures))
					for i, arch := range function.Architectures {
						architectures[i] = string(arch)
					}

					// Get ephemeral storage size (defaults to 512 MB if not set)
					var ephemeralStorageSize int64 = 512
					if function.EphemeralStorage != nil && function.EphemeralStorage.Size != nil {
						ephemeralStorageSize = int64(*function.EphemeralStorage.Size)
					}

					var tracingMode string
					if function.TracingConfig != nil {
						tracingMode = string(function.TracingConfig.Mode)
					}

					var lastModifiedAt *time.Time
					if function.LastModified != nil {
						if t, err := time.Parse("2006-01-02T15:04:05.000-0700", *function.LastModified); err == nil {
							lastModifiedAt = &t
						}
					}

					// Extract SnapStart fields
					var snapStartApplyOn, snapStartOptimizationStatus string
					if function.SnapStart != nil {
						snapStartApplyOn = string(function.SnapStart.ApplyOn)
						snapStartOptimizationStatus = string(function.SnapStart.OptimizationStatus)
					}

					// Extract environment variables
					envVars := map[string]any{}
					if function.Environment != nil && function.Environment.Variables != nil {
						for k, v := range function.Environment.Variables {
							envVars[k] = v
						}
					}

					// Convert file system configs to dict slice
					fileSystemConfigs, err := convert.JsonToDictSlice(function.FileSystemConfigs)
					if err != nil {
						return nil, err
					}

					funcArn := convert.ToValue(function.FunctionArn)

					// Create logging config sub-resource
					var loggingConfigResource plugin.Resource
					if function.LoggingConfig != nil {
						lc, err := CreateResource(a.MqlRuntime, "aws.lambda.function.loggingConfig",
							map[string]*llx.RawData{
								"__id":                llx.StringData(funcArn + "/loggingConfig"),
								"logFormat":           llx.StringData(string(function.LoggingConfig.LogFormat)),
								"applicationLogLevel": llx.StringData(string(function.LoggingConfig.ApplicationLogLevel)),
								"systemLogLevel":      llx.StringData(string(function.LoggingConfig.SystemLogLevel)),
								"logGroup":            llx.StringDataPtr(function.LoggingConfig.LogGroup),
							})
						if err != nil {
							return nil, err
						}
						loggingConfigResource = lc.(plugin.Resource)
					}

					// Create layer sub-resources
					layers := make([]any, 0, len(function.Layers))
					for _, layer := range function.Layers {
						mqlLayer, err := CreateResource(a.MqlRuntime, "aws.lambda.function.layer",
							map[string]*llx.RawData{
								"__id":                     llx.StringData(funcArn + "/layer/" + convert.ToValue(layer.Arn)),
								"arn":                      llx.StringDataPtr(layer.Arn),
								"codeSize":                 llx.IntData(layer.CodeSize),
								"signingJobArn":            llx.StringDataPtr(layer.SigningJobArn),
								"signingProfileVersionArn": llx.StringDataPtr(layer.SigningProfileVersionArn),
							})
						if err != nil {
							return nil, err
						}
						layers = append(layers, mqlLayer)
					}

					args := map[string]*llx.RawData{
						"arn":                         llx.StringDataPtr(function.FunctionArn),
						"name":                        llx.StringDataPtr(function.FunctionName),
						"runtime":                     llx.StringData(string(function.Runtime)),
						"dlqTargetArn":                llx.StringData(dlqTarget),
						"vpcConfig":                   llx.MapData(vpcConfigJson, types.Any),
						"region":                      llx.StringData(region),
						"tags":                        llx.MapData(toInterfaceMap(tags), types.String),
						"architectures":               llx.ArrayData(architectures, types.String),
						"ephemeralStorageSize":        llx.IntData(ephemeralStorageSize),
						"memorySize":                  llx.IntDataDefault(function.MemorySize, 0),
						"timeout":                     llx.IntDataDefault(function.Timeout, 3),
						"handler":                     llx.StringDataPtr(function.Handler),
						"tracingMode":                 llx.StringData(tracingMode),
						"packageType":                 llx.StringData(string(function.PackageType)),
						"codeSha256":                  llx.StringDataPtr(function.CodeSha256),
						"description":                 llx.StringDataPtr(function.Description),
						"lastModifiedAt":              llx.TimeDataPtr(lastModifiedAt),
						"state":                       llx.StringData(string(function.State)),
						"codeSize":                    llx.IntData(function.CodeSize),
						"stateReason":                 llx.StringDataPtr(function.StateReason),
						"lastUpdateStatus":            llx.StringData(string(function.LastUpdateStatus)),
						"kmsKeyArn":                   llx.StringDataPtr(function.KMSKeyArn),
						"environment":                 llx.MapData(envVars, types.String),
						"snapStartApplyOn":            llx.StringData(snapStartApplyOn),
						"snapStartOptimizationStatus": llx.StringData(snapStartOptimizationStatus),
						"fileSystemConfigs":           llx.ArrayData(fileSystemConfigs, types.Dict),
						"signingProfileVersionArn":    llx.StringDataPtr(function.SigningProfileVersionArn),
						"signingJobArn":               llx.StringDataPtr(function.SigningJobArn),
						"layers":                      llx.ArrayData(layers, types.Resource("aws.lambda.function.layer")),
					}

					if loggingConfigResource != nil {
						args["loggingConfig"] = llx.ResourceData(loggingConfigResource, "aws.lambda.function.loggingConfig")
					} else {
						args["loggingConfig"] = llx.NilData
					}

					mqlFunc, err := CreateResource(a.MqlRuntime, "aws.lambda.function", args)
					if err != nil {
						return nil, err
					}
					mqlFunc.(*mqlAwsLambdaFunction).cacheRoleArn = function.Role
					res = append(res, mqlFunc)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func getLambdaArn(name string, region string, accountId string) string {
	return arn.ARN{
		Region:    region,
		Partition: "aws",
		Service:   "lambda",
		AccountID: accountId,
		Resource:  "function:" + name,
	}.String()
}

func initAwsLambdaFunction(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	name := args["name"]
	region := args["region"]

	var arnVal string
	if args["arn"] == nil {
		if name == nil {
			return nil, nil, errors.New("name required to fetch lambda function")
		}
		if region == nil {
			return nil, nil, errors.New("region required to fetch lambda function")
		}
		arnVal = getLambdaArn(name.String(), region.String(), "")
		if arnVal == "" {
			return nil, nil, errors.New("arn required to fetch lambda function")
		}
	} else {
		arnVal = args["arn"].Value.(string)
	}

	// load all lambda functions
	obj, err := CreateResource(runtime, "aws.lambda", map[string]*llx.RawData{})
	if err != nil {
		return nil, nil, err
	}
	l := obj.(*mqlAwsLambda)

	rawResources := l.GetFunctions()
	if rawResources.Error != nil {
		return nil, nil, rawResources.Error
	}

	for _, rawResource := range rawResources.Data {
		fn := rawResource.(*mqlAwsLambdaFunction)
		if fn.Arn.Data == arnVal {
			return args, fn, nil
		}
	}
	return nil, nil, errors.New("lambda function does not exist")
}

func (a *mqlAwsLambdaFunction) id() (string, error) {
	return a.Arn.Data, nil
}

type mqlAwsLambdaFunctionInternal struct {
	cacheRoleArn *string
}

func (a *mqlAwsLambdaFunction) concurrency() (int64, error) {
	funcName := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Lambda(region)
	ctx := context.Background()

	// no pagination required
	functionConcurrency, err := svc.GetFunctionConcurrency(ctx, &lambda.GetFunctionConcurrencyInput{FunctionName: &funcName})
	if err != nil {
		return 0, errors.Wrap(err, "could not gather aws lambda function concurrency")
	}
	if functionConcurrency.ReservedConcurrentExecutions == nil {
		return 0, nil
	}
	return int64(*functionConcurrency.ReservedConcurrentExecutions), nil
}

func (a *mqlAwsLambdaFunction) policy() (any, error) {
	funcArn := a.Arn.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Lambda(region)
	ctx := context.Background()

	// no pagination required
	functionPolicy, err := svc.GetPolicy(ctx, &lambda.GetPolicyInput{FunctionName: &funcArn})
	var respErr *http.ResponseError
	if err != nil && errors.As(err, &respErr) {
		if respErr.HTTPStatusCode() == 404 {
			return nil, nil
		}
	} else if err != nil {
		return nil, err
	}
	if functionPolicy != nil {
		var policy lambdaPolicyDocument
		err = json.Unmarshal([]byte(*functionPolicy.Policy), &policy)
		if err != nil {
			return nil, err
		}
		return convert.JsonToDict(policy)
	}

	return nil, nil
}

func (a *mqlAwsLambdaFunction) urlConfig() (*mqlAwsLambdaFunctionUrlConfig, error) {
	funcName := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Lambda(region)
	ctx := context.Background()

	resp, err := svc.GetFunctionUrlConfig(ctx, &lambda.GetFunctionUrlConfigInput{FunctionName: &funcName})
	var respErr *http.ResponseError
	if err != nil && errors.As(err, &respErr) {
		if respErr.HTTPStatusCode() == 404 {
			a.UrlConfig.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
	} else if err != nil {
		return nil, err
	}

	var corsAllowOrigins, corsAllowMethods, corsAllowHeaders, corsExposeHeaders []any
	var corsAllowCredentials bool
	var corsMaxAge int64
	if resp.Cors != nil {
		corsAllowOrigins = toInterfaceArr(resp.Cors.AllowOrigins)
		corsAllowMethods = toInterfaceArr(resp.Cors.AllowMethods)
		corsAllowHeaders = toInterfaceArr(resp.Cors.AllowHeaders)
		corsExposeHeaders = toInterfaceArr(resp.Cors.ExposeHeaders)
		if resp.Cors.AllowCredentials != nil {
			corsAllowCredentials = *resp.Cors.AllowCredentials
		}
		if resp.Cors.MaxAge != nil {
			corsMaxAge = int64(*resp.Cors.MaxAge)
		}
	}

	res, err := CreateResource(a.MqlRuntime, "aws.lambda.function.urlConfig",
		map[string]*llx.RawData{
			"__id":                 llx.StringData(a.Arn.Data + "/urlConfig"),
			"functionUrl":          llx.StringDataPtr(resp.FunctionUrl),
			"authType":             llx.StringData(string(resp.AuthType)),
			"corsAllowOrigins":     llx.ArrayData(corsAllowOrigins, types.String),
			"corsAllowMethods":     llx.ArrayData(corsAllowMethods, types.String),
			"corsAllowHeaders":     llx.ArrayData(corsAllowHeaders, types.String),
			"corsAllowCredentials": llx.BoolData(corsAllowCredentials),
			"corsExposeHeaders":    llx.ArrayData(corsExposeHeaders, types.String),
			"corsMaxAge":           llx.IntData(corsMaxAge),
			"createdAt":            llx.TimeDataPtr(parseAwsTimestampPtr(resp.CreationTime)),
			"lastModifiedAt":       llx.TimeDataPtr(parseAwsTimestampPtr(resp.LastModifiedTime)),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsLambdaFunctionUrlConfig), nil
}

func (a *mqlAwsLambdaFunctionUrlConfig) id() (string, error) {
	return a.FunctionUrl.Data, nil
}

func (a *mqlAwsLambdaFunction) role() (*mqlAwsIamRole, error) {
	if a.cacheRoleArn == nil || *a.cacheRoleArn == "" {
		a.Role.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	mqlRole, err := NewResource(a.MqlRuntime, ResourceAwsIamRole,
		map[string]*llx.RawData{
			"arn": llx.StringDataPtr(a.cacheRoleArn),
		})
	if err != nil {
		return nil, err
	}
	return mqlRole.(*mqlAwsIamRole), nil
}

func (a *mqlAwsLambdaFunctionLoggingConfig) id() (string, error) {
	return a.LogGroup.Data, nil
}

func (a *mqlAwsLambdaFunctionLayer) id() (string, error) {
	return a.Arn.Data, nil
}

// ==================== Top-Level Layers ====================

func (a *mqlAwsLambda) layers() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getLayers(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsLambda) getLayers(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("lambda>getLayers>calling aws with region %s", region)

			svc := conn.Lambda(region)
			ctx := context.Background()
			res := []any{}

			paginator := lambda.NewListLayersPaginator(svc, &lambda.ListLayersInput{})
			for paginator.HasMorePages() {
				resp, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, errors.Wrap(err, "could not gather aws lambda layers")
				}
				for _, layer := range resp.Layers {
					var latestVersionArn, description, licenseInfo string
					var latestVersion int64
					var createdDate *time.Time
					compatibleRuntimes := []any{}
					compatibleArchitectures := []any{}

					if layer.LatestMatchingVersion != nil {
						v := layer.LatestMatchingVersion
						latestVersionArn = convert.ToValue(v.LayerVersionArn)
						description = convert.ToValue(v.Description)
						licenseInfo = convert.ToValue(v.LicenseInfo)
						latestVersion = v.Version
						createdDate = parseAwsTimestampPtr(v.CreatedDate)
						for _, rt := range v.CompatibleRuntimes {
							compatibleRuntimes = append(compatibleRuntimes, string(rt))
						}
						for _, arch := range v.CompatibleArchitectures {
							compatibleArchitectures = append(compatibleArchitectures, string(arch))
						}
					}

					mqlLayer, err := CreateResource(a.MqlRuntime, "aws.lambda.layer",
						map[string]*llx.RawData{
							"arn":                     llx.StringDataPtr(layer.LayerArn),
							"name":                    llx.StringDataPtr(layer.LayerName),
							"latestVersionArn":        llx.StringData(latestVersionArn),
							"latestVersion":           llx.IntData(latestVersion),
							"description":             llx.StringData(description),
							"compatibleRuntimes":      llx.ArrayData(compatibleRuntimes, types.String),
							"compatibleArchitectures": llx.ArrayData(compatibleArchitectures, types.String),
							"createdDate":             llx.TimeDataPtr(createdDate),
							"licenseInfo":             llx.StringData(licenseInfo),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlLayer)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsLambdaLayer) id() (string, error) {
	return a.Arn.Data, nil
}

// ==================== Top-Level Event Source Mappings ====================

func (a *mqlAwsLambda) eventSourceMappings() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getEventSourceMappings(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsLambda) getEventSourceMappings(conn *connection.AwsConnection) []*jobpool.Job {
	tasks := make([]*jobpool.Job, 0)
	regions, err := conn.Regions()
	if err != nil {
		return []*jobpool.Job{{Err: err}}
	}

	for _, region := range regions {
		f := func() (jobpool.JobResult, error) {
			log.Debug().Msgf("lambda>getEventSourceMappings>calling aws with region %s", region)

			svc := conn.Lambda(region)
			ctx := context.Background()
			res := []any{}

			paginator := lambda.NewListEventSourceMappingsPaginator(svc, &lambda.ListEventSourceMappingsInput{})
			for paginator.HasMorePages() {
				resp, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, errors.Wrap(err, "could not gather aws lambda event source mappings")
				}
				for _, esm := range resp.EventSourceMappings {
					mqlEsm, err := createEventSourceMappingResource(a.MqlRuntime, esm, region)
					if err != nil {
						return nil, err
					}
					res = append(res, mqlEsm)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

// createEventSourceMappingResource creates an aws.lambda.eventSourceMapping resource from SDK data.
// Shared between top-level listing and per-function listing to ensure cache reuse via UUID-based __id.
func createEventSourceMappingResource(runtime *plugin.Runtime, esm lambdatypes.EventSourceMappingConfiguration, region string) (*mqlAwsLambdaEventSourceMapping, error) {
	var onFailureDestinationArn string
	if esm.DestinationConfig != nil && esm.DestinationConfig.OnFailure != nil {
		onFailureDestinationArn = convert.ToValue(esm.DestinationConfig.OnFailure.Destination)
	}

	filterCriteria, err := convert.JsonToDict(esm.FilterCriteria)
	if err != nil {
		return nil, err
	}

	var maximumConcurrency int64
	if esm.ScalingConfig != nil && esm.ScalingConfig.MaximumConcurrency != nil {
		maximumConcurrency = int64(*esm.ScalingConfig.MaximumConcurrency)
	}

	res, err := CreateResource(runtime, "aws.lambda.eventSourceMapping",
		map[string]*llx.RawData{
			"__id":                           llx.StringDataPtr(esm.UUID),
			"uuid":                           llx.StringDataPtr(esm.UUID),
			"eventSourceArn":                 llx.StringDataPtr(esm.EventSourceArn),
			"functionArn":                    llx.StringDataPtr(esm.FunctionArn),
			"region":                         llx.StringData(region),
			"state":                          llx.StringDataPtr(esm.State),
			"stateTransitionReason":          llx.StringDataPtr(esm.StateTransitionReason),
			"batchSize":                      llx.IntDataDefault(esm.BatchSize, 0),
			"maximumBatchingWindowInSeconds": llx.IntDataDefault(esm.MaximumBatchingWindowInSeconds, 0),
			"parallelizationFactor":          llx.IntDataDefault(esm.ParallelizationFactor, 0),
			"maximumRetryAttempts":           llx.IntDataDefault(esm.MaximumRetryAttempts, -1),
			"maximumRecordAgeInSeconds":      llx.IntDataDefault(esm.MaximumRecordAgeInSeconds, -1),
			"bisectBatchOnFunctionError":     llx.BoolDataPtr(esm.BisectBatchOnFunctionError),
			"lastModified":                   llx.TimeDataPtr(esm.LastModified),
			"lastProcessingResult":           llx.StringDataPtr(esm.LastProcessingResult),
			"topics":                         llx.ArrayData(toInterfaceArr(esm.Topics), types.String),
			"queues":                         llx.ArrayData(toInterfaceArr(esm.Queues), types.String),
			"tumblingWindowInSeconds":        llx.IntDataDefault(esm.TumblingWindowInSeconds, 0),
			"startingPosition":               llx.StringData(string(esm.StartingPosition)),
			"onFailureDestinationArn":        llx.StringData(onFailureDestinationArn),
			"filterCriteria":                 llx.DictData(filterCriteria),
			"maximumConcurrency":             llx.IntData(maximumConcurrency),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsLambdaEventSourceMapping), nil
}

func (a *mqlAwsLambdaEventSourceMapping) id() (string, error) {
	return a.Uuid.Data, nil
}

// ==================== Per-Function Event Source Mappings ====================

func (a *mqlAwsLambdaFunction) eventSourceMappings() ([]any, error) {
	funcName := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Lambda(region)
	ctx := context.Background()
	res := []any{}

	paginator := lambda.NewListEventSourceMappingsPaginator(svc,
		&lambda.ListEventSourceMappingsInput{FunctionName: &funcName})
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				return res, nil
			}
			return nil, errors.Wrap(err, "could not gather lambda event source mappings")
		}
		for _, esm := range resp.EventSourceMappings {
			mqlEsm, err := createEventSourceMappingResource(a.MqlRuntime, esm, region)
			if err != nil {
				return nil, err
			}
			res = append(res, mqlEsm)
		}
	}
	return res, nil
}

// ==================== Per-Function Aliases ====================

func (a *mqlAwsLambdaFunction) aliases() ([]any, error) {
	funcName := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Lambda(region)
	ctx := context.Background()
	res := []any{}

	paginator := lambda.NewListAliasesPaginator(svc,
		&lambda.ListAliasesInput{FunctionName: &funcName})
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				return res, nil
			}
			return nil, errors.Wrap(err, "could not gather lambda aliases")
		}
		for _, alias := range resp.Aliases {
			var routingWeights map[string]any
			if alias.RoutingConfig != nil && alias.RoutingConfig.AdditionalVersionWeights != nil {
				routingWeights = make(map[string]any)
				for k, v := range alias.RoutingConfig.AdditionalVersionWeights {
					routingWeights[k] = v
				}
			}

			mqlAlias, err := CreateResource(a.MqlRuntime, "aws.lambda.function.alias",
				map[string]*llx.RawData{
					"__id":                 llx.StringDataPtr(alias.AliasArn),
					"arn":                  llx.StringDataPtr(alias.AliasArn),
					"name":                 llx.StringDataPtr(alias.Name),
					"functionVersion":      llx.StringDataPtr(alias.FunctionVersion),
					"description":          llx.StringDataPtr(alias.Description),
					"revisionId":           llx.StringDataPtr(alias.RevisionId),
					"routingConfigWeights": llx.MapData(routingWeights, types.Float),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlAlias)
		}
	}
	return res, nil
}

func (a *mqlAwsLambdaFunctionAlias) id() (string, error) {
	return a.Arn.Data, nil
}

// ==================== Per-Function Provisioned Concurrency ====================

func (a *mqlAwsLambdaFunction) provisionedConcurrencyConfigs() ([]any, error) {
	funcName := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Lambda(region)
	ctx := context.Background()
	res := []any{}

	paginator := lambda.NewListProvisionedConcurrencyConfigsPaginator(svc,
		&lambda.ListProvisionedConcurrencyConfigsInput{FunctionName: &funcName})
	for paginator.HasMorePages() {
		resp, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				return res, nil
			}
			return nil, errors.Wrap(err, "could not gather lambda provisioned concurrency configs")
		}
		for _, pcc := range resp.ProvisionedConcurrencyConfigs {
			mqlPcc, err := CreateResource(a.MqlRuntime, "aws.lambda.function.provisionedConcurrencyConfig",
				map[string]*llx.RawData{
					"__id":                          llx.StringData(a.Arn.Data + "/provisionedConcurrency/" + convert.ToValue(pcc.FunctionArn)),
					"functionArn":                   llx.StringDataPtr(pcc.FunctionArn),
					"requestedConcurrentExecutions": llx.IntDataDefault(pcc.RequestedProvisionedConcurrentExecutions, 0),
					"allocatedConcurrentExecutions": llx.IntDataDefault(pcc.AllocatedProvisionedConcurrentExecutions, 0),
					"availableConcurrentExecutions": llx.IntDataDefault(pcc.AvailableProvisionedConcurrentExecutions, 0),
					"status":                        llx.StringData(string(pcc.Status)),
					"statusReason":                  llx.StringDataPtr(pcc.StatusReason),
					"lastModified":                  llx.TimeDataPtr(parseAwsTimestampPtr(pcc.LastModified)),
				})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlPcc)
		}
	}
	return res, nil
}

func (a *mqlAwsLambdaFunctionProvisionedConcurrencyConfig) id() (string, error) {
	return a.FunctionArn.Data, nil
}

// ==================== Per-Function Code Signing Config ====================

func (a *mqlAwsLambdaFunction) codeSigningConfig() (*mqlAwsLambdaCodeSigningConfig, error) {
	funcName := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Lambda(region)
	ctx := context.Background()

	// Step 1: Get the code signing config ARN for this function
	cscResp, err := svc.GetFunctionCodeSigningConfig(ctx,
		&lambda.GetFunctionCodeSigningConfigInput{FunctionName: &funcName})
	var respErr *http.ResponseError
	if err != nil && errors.As(err, &respErr) {
		if respErr.HTTPStatusCode() == 404 {
			a.CodeSigningConfig.State = plugin.StateIsNull | plugin.StateIsSet
			return nil, nil
		}
	} else if err != nil {
		return nil, err
	}

	if cscResp == nil || cscResp.CodeSigningConfigArn == nil || *cscResp.CodeSigningConfigArn == "" {
		a.CodeSigningConfig.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	// Step 2: Get the full code signing config details
	configResp, err := svc.GetCodeSigningConfig(ctx,
		&lambda.GetCodeSigningConfigInput{CodeSigningConfigArn: cscResp.CodeSigningConfigArn})
	if err != nil {
		return nil, errors.Wrap(err, "could not get code signing config")
	}

	csc := configResp.CodeSigningConfig

	allowedArns := []any{}
	if csc.AllowedPublishers != nil {
		for _, publisherArn := range csc.AllowedPublishers.SigningProfileVersionArns {
			allowedArns = append(allowedArns, publisherArn)
		}
	}

	var untrustedAction string
	if csc.CodeSigningPolicies != nil {
		untrustedAction = string(csc.CodeSigningPolicies.UntrustedArtifactOnDeployment)
	}

	res, err := CreateResource(a.MqlRuntime, "aws.lambda.codeSigningConfig",
		map[string]*llx.RawData{
			"arn":                           llx.StringDataPtr(csc.CodeSigningConfigArn),
			"id":                            llx.StringDataPtr(csc.CodeSigningConfigId),
			"description":                   llx.StringDataPtr(csc.Description),
			"allowedPublisherProfileArns":   llx.ArrayData(allowedArns, types.String),
			"untrustedArtifactOnDeployment": llx.StringData(untrustedAction),
			"lastModified":                  llx.TimeDataPtr(parseAwsTimestampPtr(csc.LastModified)),
		})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsLambdaCodeSigningConfig), nil
}

func (a *mqlAwsLambdaCodeSigningConfig) id() (string, error) {
	return a.Arn.Data, nil
}

// ==================== Types ====================

type lambdaPolicyDocument struct {
	Version   string                  `json:"Version,omitempty"`
	Statement []lambdaPolicyStatement `json:"Statement,omitempty"`
}

type lambdaPolicyStatement struct {
	Sid       string              `json:"Sid,omitempty"`
	Effect    string              `json:"Effect,omitempty"`
	Action    string              `json:"Action,omitempty"`
	Resource  string              `json:"Resource,omitempty"`
	Principal awspolicy.Principal `json:"Principal,omitempty"`
}
