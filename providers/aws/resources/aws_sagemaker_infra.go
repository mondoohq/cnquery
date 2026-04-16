// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	sagemakerTypes "github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
	"go.mondoo.com/mql/v13/types"
)

// ---- Notebook Instance Lifecycle Configs ----

func (a *mqlAwsSagemaker) notebookInstanceLifecycleConfigs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getNotebookInstanceLifecycleConfigs(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getNotebookInstanceLifecycleConfigs(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListNotebookInstanceLifecycleConfigsPaginator(svc, &sagemaker.ListNotebookInstanceLifecycleConfigsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, lc := range page.NotebookInstanceLifecycleConfigs {
					mqlLc, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerNotebookInstanceLifecycleConfig,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(lc.NotebookInstanceLifecycleConfigArn),
							"name":           llx.StringDataPtr(lc.NotebookInstanceLifecycleConfigName),
							"region":         llx.StringData(region),
							"createdAt":      llx.TimeDataPtr(lc.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(lc.LastModifiedTime),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlLc)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerNotebookInstanceLifecycleConfigInternal struct {
	region string
}

func (a *mqlAwsSagemakerNotebookInstanceLifecycleConfig) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerNotebookInstanceLifecycleConfig) onCreate() ([]any, error) {
	return a.fetchLifecycleScripts("OnCreate")
}

func (a *mqlAwsSagemakerNotebookInstanceLifecycleConfig) onStart() ([]any, error) {
	return a.fetchLifecycleScripts("OnStart")
}

func (a *mqlAwsSagemakerNotebookInstanceLifecycleConfig) fetchLifecycleScripts(hookType string) ([]any, error) {
	name := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(region)
	ctx := context.Background()

	resp, err := svc.DescribeNotebookInstanceLifecycleConfig(ctx, &sagemaker.DescribeNotebookInstanceLifecycleConfigInput{
		NotebookInstanceLifecycleConfigName: &name,
	})
	if err != nil {
		return nil, err
	}

	var hooks []sagemakerTypes.NotebookInstanceLifecycleHook
	if hookType == "OnCreate" {
		hooks = resp.OnCreate
	} else {
		hooks = resp.OnStart
	}

	parentArn := a.Arn.Data
	res := make([]any, 0, len(hooks))
	for i, hook := range hooks {
		content := ""
		if hook.Content != nil {
			decoded, err := base64.StdEncoding.DecodeString(*hook.Content)
			if err != nil {
				return nil, err
			}
			content = string(decoded)
		}
		mqlScript, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerLifecycleConfigScript,
			map[string]*llx.RawData{
				"content": llx.StringData(content),
			})
		if err != nil {
			return nil, err
		}
		script := mqlScript.(*mqlAwsSagemakerLifecycleConfigScript)
		script.cacheParentArn = parentArn
		script.cacheHookType = hookType
		script.cacheIndex = i
		res = append(res, mqlScript)
	}
	return res, nil
}

type mqlAwsSagemakerLifecycleConfigScriptInternal struct {
	cacheParentArn string
	cacheHookType  string
	cacheIndex     int
}

func (a *mqlAwsSagemakerLifecycleConfigScript) id() (string, error) {
	return fmt.Sprintf("%s/script/%s/%d", a.cacheParentArn, a.cacheHookType, a.cacheIndex), nil
}

// ---- Code Repositories ----

func (a *mqlAwsSagemaker) codeRepositories() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getCodeRepositories(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getCodeRepositories(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListCodeRepositoriesPaginator(svc, &sagemaker.ListCodeRepositoriesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, repo := range page.CodeRepositorySummaryList {
					mqlRepo, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerCodeRepository,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(repo.CodeRepositoryArn),
							"name":           llx.StringDataPtr(repo.CodeRepositoryName),
							"region":         llx.StringData(region),
							"createdAt":      llx.TimeDataPtr(repo.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(repo.LastModifiedTime),
						})
					if err != nil {
						return nil, err
					}
					r := mqlRepo.(*mqlAwsSagemakerCodeRepository)
					r.cacheGitConfig = repo.GitConfig
					res = append(res, mqlRepo)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerCodeRepositoryInternal struct {
	cacheGitConfig *sagemakerTypes.GitConfig
}

func (a *mqlAwsSagemakerCodeRepository) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerCodeRepository) gitConfig() (*mqlAwsSagemakerCodeRepositoryGitConfig, error) {
	if a.cacheGitConfig == nil {
		a.GitConfig.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	gc := a.cacheGitConfig
	mqlGc, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerCodeRepositoryGitConfig,
		map[string]*llx.RawData{
			"repositoryUrl": llx.StringData(convert.ToValue(gc.RepositoryUrl)),
			"branch":        llx.StringData(convert.ToValue(gc.Branch)),
			"secretArn":     llx.StringData(convert.ToValue(gc.SecretArn)),
		})
	if err != nil {
		return nil, err
	}
	gitConfig := mqlGc.(*mqlAwsSagemakerCodeRepositoryGitConfig)
	gitConfig.cacheParentArn = a.Arn.Data
	return gitConfig, nil
}

type mqlAwsSagemakerCodeRepositoryGitConfigInternal struct {
	cacheParentArn string
}

func (a *mqlAwsSagemakerCodeRepositoryGitConfig) id() (string, error) {
	return a.cacheParentArn + "/gitConfig", nil
}

// ---- Images ----

func (a *mqlAwsSagemaker) images() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getImages(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getImages(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListImagesPaginator(svc, &sagemaker.ListImagesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, img := range page.Images {
					mqlImg, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerImage,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(img.ImageArn),
							"name":           llx.StringDataPtr(img.ImageName),
							"displayName":    llx.StringData(convert.ToValue(img.DisplayName)),
							"description":    llx.StringData(convert.ToValue(img.Description)),
							"region":         llx.StringData(region),
							"status":         llx.StringData(string(img.ImageStatus)),
							"createdAt":      llx.TimeDataPtr(img.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(img.LastModifiedTime),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlImg)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerImageInternal struct {
	detailsFetched bool
	detailsLock    sync.Mutex
	cacheRoleArn   *string
}

func (a *mqlAwsSagemakerImage) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerImage) fetchDetails() error {
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

	resp, err := svc.DescribeImage(ctx, &sagemaker.DescribeImageInput{ImageName: &name})
	if err != nil {
		return err
	}

	a.cacheRoleArn = resp.RoleArn
	a.detailsFetched = true
	return nil
}

func (a *mqlAwsSagemakerImage) iamRole() (*mqlAwsIamRole, error) {
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

func (a *mqlAwsSagemakerImage) versions() ([]any, error) {
	region := a.Region.Data
	name := a.Name.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(region)
	ctx := context.Background()

	res := []any{}
	paginator := sagemaker.NewListImageVersionsPaginator(svc, &sagemaker.ListImageVersionsInput{
		ImageName: &name,
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			if Is400AccessDeniedError(err) {
				log.Warn().Str("region", region).Msg("error accessing region for AWS API")
				return res, nil
			}
			return nil, err
		}

		for _, v := range page.ImageVersions {
			var version int64
			if v.Version != nil {
				version = int64(*v.Version)
			}
			mqlVersion, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerImageVersion,
				map[string]*llx.RawData{
					"arn":            llx.StringDataPtr(v.ImageVersionArn),
					"version":        llx.IntData(version),
					"region":         llx.StringData(region),
					"status":         llx.StringData(string(v.ImageVersionStatus)),
					"createdAt":      llx.TimeDataPtr(v.CreationTime),
					"lastModifiedAt": llx.TimeDataPtr(v.LastModifiedTime),
					"imageArn":       llx.StringDataPtr(v.ImageArn),
				})
			if err != nil {
				return nil, err
			}
			iv := mqlVersion.(*mqlAwsSagemakerImageVersion)
			iv.cacheImageName = name
			res = append(res, mqlVersion)
		}
	}
	return res, nil
}

// ---- Image Versions ----

type mqlAwsSagemakerImageVersionInternal struct {
	detailsFetched       bool
	detailsLock          sync.Mutex
	cacheImageName       string
	cacheBaseImage       string
	cacheContainerImage  string
	cacheMlFramework     string
	cacheProgrammingLang string
}

func (a *mqlAwsSagemakerImageVersion) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerImageVersion) fetchDetails() error {
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

	imageName := a.cacheImageName
	version := int32(a.Version.Data)
	resp, err := svc.DescribeImageVersion(ctx, &sagemaker.DescribeImageVersionInput{
		ImageName: &imageName,
		Version:   &version,
	})
	if err != nil {
		return err
	}

	a.cacheBaseImage = convert.ToValue(resp.BaseImage)
	a.cacheContainerImage = convert.ToValue(resp.ContainerImage)
	a.cacheMlFramework = convert.ToValue(resp.MLFramework)
	a.cacheProgrammingLang = convert.ToValue(resp.ProgrammingLang)
	a.detailsFetched = true
	return nil
}

func (a *mqlAwsSagemakerImageVersion) baseImage() (string, error) {
	if err := a.fetchDetails(); err != nil {
		return "", err
	}
	return a.cacheBaseImage, nil
}

func (a *mqlAwsSagemakerImageVersion) containerImage() (string, error) {
	if err := a.fetchDetails(); err != nil {
		return "", err
	}
	return a.cacheContainerImage, nil
}

func (a *mqlAwsSagemakerImageVersion) mlFramework() (string, error) {
	if err := a.fetchDetails(); err != nil {
		return "", err
	}
	return a.cacheMlFramework, nil
}

func (a *mqlAwsSagemakerImageVersion) programmingLang() (string, error) {
	if err := a.fetchDetails(); err != nil {
		return "", err
	}
	return a.cacheProgrammingLang, nil
}

// ---- App Image Configs ----

func (a *mqlAwsSagemaker) appImageConfigs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getAppImageConfigs(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getAppImageConfigs(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListAppImageConfigsPaginator(svc, &sagemaker.ListAppImageConfigsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, config := range page.AppImageConfigs {
					mqlConfig, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerAppImageConfig,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(config.AppImageConfigArn),
							"name":           llx.StringDataPtr(config.AppImageConfigName),
							"region":         llx.StringData(region),
							"createdAt":      llx.TimeDataPtr(config.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(config.LastModifiedTime),
						})
					if err != nil {
						return nil, err
					}
					c := mqlConfig.(*mqlAwsSagemakerAppImageConfig)
					c.cacheKernelGatewayImageConfig = config.KernelGatewayImageConfig
					c.cacheJupyterLabAppImageConfig = config.JupyterLabAppImageConfig
					c.cacheCodeEditorAppImageConfig = config.CodeEditorAppImageConfig
					res = append(res, mqlConfig)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerAppImageConfigInternal struct {
	cacheKernelGatewayImageConfig *sagemakerTypes.KernelGatewayImageConfig
	cacheJupyterLabAppImageConfig *sagemakerTypes.JupyterLabAppImageConfig
	cacheCodeEditorAppImageConfig *sagemakerTypes.CodeEditorAppImageConfig
}

func (a *mqlAwsSagemakerAppImageConfig) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerAppImageConfig) kernelGatewayImageConfig() (*mqlAwsSagemakerAppImageConfigKernelGatewayConfig, error) {
	if a.cacheKernelGatewayImageConfig == nil {
		a.KernelGatewayImageConfig.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	kg := a.cacheKernelGatewayImageConfig
	parentArn := a.Arn.Data

	// Build kernel specs
	kernelSpecs := make([]any, 0, len(kg.KernelSpecs))
	for _, ks := range kg.KernelSpecs {
		mqlKs, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerAppImageConfigKernelSpec,
			map[string]*llx.RawData{
				"name":        llx.StringData(convert.ToValue(ks.Name)),
				"displayName": llx.StringData(convert.ToValue(ks.DisplayName)),
			})
		if err != nil {
			return nil, err
		}
		spec := mqlKs.(*mqlAwsSagemakerAppImageConfigKernelSpec)
		spec.cacheParentArn = parentArn
		kernelSpecs = append(kernelSpecs, mqlKs)
	}

	// Build file system config as dict
	var fsConfig any
	if kg.FileSystemConfig != nil {
		fsDict, err := convert.JsonToDict(kg.FileSystemConfig)
		if err != nil {
			return nil, err
		}
		fsConfig = fsDict
	}

	mqlKgConfig, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerAppImageConfigKernelGatewayConfig,
		map[string]*llx.RawData{
			"kernelSpecs":      llx.ArrayData(kernelSpecs, types.Resource(ResourceAwsSagemakerAppImageConfigKernelSpec)),
			"fileSystemConfig": llx.DictData(fsConfig),
		})
	if err != nil {
		return nil, err
	}
	kgRes := mqlKgConfig.(*mqlAwsSagemakerAppImageConfigKernelGatewayConfig)
	kgRes.cacheParentArn = parentArn
	return kgRes, nil
}

func (a *mqlAwsSagemakerAppImageConfig) jupyterLabAppImageConfig() (any, error) {
	if a.cacheJupyterLabAppImageConfig == nil {
		return nil, nil
	}
	return convert.JsonToDict(a.cacheJupyterLabAppImageConfig)
}

func (a *mqlAwsSagemakerAppImageConfig) codeEditorAppImageConfig() (any, error) {
	if a.cacheCodeEditorAppImageConfig == nil {
		return nil, nil
	}
	return convert.JsonToDict(a.cacheCodeEditorAppImageConfig)
}

type mqlAwsSagemakerAppImageConfigKernelGatewayConfigInternal struct {
	cacheParentArn string
}

func (a *mqlAwsSagemakerAppImageConfigKernelGatewayConfig) id() (string, error) {
	return a.cacheParentArn + "/kernelGateway", nil
}

type mqlAwsSagemakerAppImageConfigKernelSpecInternal struct {
	cacheParentArn string
}

func (a *mqlAwsSagemakerAppImageConfigKernelSpec) id() (string, error) {
	return a.cacheParentArn + "/kernelSpec/" + a.Name.Data, nil
}

// ---- Apps ----

func (a *mqlAwsSagemaker) apps() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getApps(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getApps(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListAppsPaginator(svc, &sagemaker.ListAppsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, app := range page.Apps {
					domainId := convert.ToValue(app.DomainId)
					appType := string(app.AppType)
					appName := convert.ToValue(app.AppName)
					userProfileName := convert.ToValue(app.UserProfileName)
					spaceName := convert.ToValue(app.SpaceName)

					// Construct the ARN: arn:aws:sagemaker:region:account:app/domainId/appType/appName
					// For user-profile apps: arn:aws:sagemaker:region:account:app/domainId/userProfile/appType/appName
					// For space apps: arn:aws:sagemaker:region:account:app/domainId/space/appType/appName
					// Use a composite ID for uniqueness
					appArn := fmt.Sprintf("arn:aws:sagemaker:%s:%s:app/%s", region, conn.AccountId(), domainId)
					if userProfileName != "" {
						appArn += "/" + userProfileName
					} else if spaceName != "" {
						appArn += "/" + spaceName
					}
					appArn += "/" + appType + "/" + appName

					mqlApp, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerApp,
						map[string]*llx.RawData{
							"arn":             llx.StringData(appArn),
							"name":            llx.StringData(appName),
							"appType":         llx.StringData(appType),
							"region":          llx.StringData(region),
							"status":          llx.StringData(string(app.Status)),
							"createdAt":       llx.TimeDataPtr(app.CreationTime),
							"domainId":        llx.StringData(domainId),
							"userProfileName": llx.StringData(userProfileName),
							"spaceName":       llx.StringData(spaceName),
						})
					if err != nil {
						return nil, err
					}
					ap := mqlApp.(*mqlAwsSagemakerApp)
					ap.cacheResourceSpec = app.ResourceSpec
					res = append(res, mqlApp)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerAppInternal struct {
	cacheResourceSpec *sagemakerTypes.ResourceSpec
}

func (a *mqlAwsSagemakerApp) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerApp) domain() (*mqlAwsSagemakerDomain, error) {
	// Domain lookup by ID is not straightforward - mark as null
	a.Domain.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAwsSagemakerApp) userProfile() (*mqlAwsSagemakerUserProfile, error) {
	// User profile cross-resource lookup is complex - mark as null
	a.UserProfile.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAwsSagemakerApp) space() (*mqlAwsSagemakerSpace, error) {
	// Space cross-resource lookup is complex - mark as null
	a.Space.State = plugin.StateIsNull | plugin.StateIsSet
	return nil, nil
}

func (a *mqlAwsSagemakerApp) resourceSpec() (*mqlAwsSagemakerAppResourceSpec, error) {
	if a.cacheResourceSpec == nil {
		a.ResourceSpec.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	rs := a.cacheResourceSpec
	mqlRs, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerAppResourceSpec,
		map[string]*llx.RawData{
			"instanceType":               llx.StringData(string(rs.InstanceType)),
			"sageMakerImageArn":          llx.StringData(convert.ToValue(rs.SageMakerImageArn)),
			"sageMakerImageVersionArn":   llx.StringData(convert.ToValue(rs.SageMakerImageVersionArn)),
			"sageMakerImageVersionAlias": llx.StringData(convert.ToValue(rs.SageMakerImageVersionAlias)),
			"lifecycleConfigArn":         llx.StringData(convert.ToValue(rs.LifecycleConfigArn)),
		})
	if err != nil {
		return nil, err
	}
	rsRes := mqlRs.(*mqlAwsSagemakerAppResourceSpec)
	rsRes.cacheAppArn = a.Arn.Data
	return rsRes, nil
}

type mqlAwsSagemakerAppResourceSpecInternal struct {
	cacheAppArn string
}

func (a *mqlAwsSagemakerAppResourceSpec) id() (string, error) {
	return a.cacheAppArn + "/resourceSpec", nil
}

// ---- Studio Lifecycle Configs ----

func (a *mqlAwsSagemaker) studioLifecycleConfigs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getStudioLifecycleConfigs(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getStudioLifecycleConfigs(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListStudioLifecycleConfigsPaginator(svc, &sagemaker.ListStudioLifecycleConfigsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS API")
						return res, nil
					}
					return nil, err
				}

				for _, lc := range page.StudioLifecycleConfigs {
					mqlLc, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerStudioLifecycleConfig,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(lc.StudioLifecycleConfigArn),
							"name":           llx.StringDataPtr(lc.StudioLifecycleConfigName),
							"region":         llx.StringData(region),
							"appType":        llx.StringData(string(lc.StudioLifecycleConfigAppType)),
							"createdAt":      llx.TimeDataPtr(lc.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(lc.LastModifiedTime),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlLc)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsSagemakerStudioLifecycleConfig) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerStudioLifecycleConfig) content() (string, error) {
	name := a.Name.Data
	region := a.Region.Data
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Sagemaker(region)
	ctx := context.Background()

	resp, err := svc.DescribeStudioLifecycleConfig(ctx, &sagemaker.DescribeStudioLifecycleConfigInput{
		StudioLifecycleConfigName: &name,
	})
	if err != nil {
		return "", err
	}

	if resp.StudioLifecycleConfigContent == nil {
		return "", nil
	}

	decoded, err := base64.StdEncoding.DecodeString(*resp.StudioLifecycleConfigContent)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
