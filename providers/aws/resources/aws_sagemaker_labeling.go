// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	sagemakerTypes "github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/aws/connection"
)

// ---- Labeling Jobs ----

func (a *mqlAwsSagemaker) labelingJobs() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getLabelingJobs(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getLabelingJobs(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListLabelingJobsPaginator(svc, &sagemaker.ListLabelingJobsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS SageMaker labeling jobs")
						return res, nil
					}
					return nil, err
				}

				for _, job := range page.LabelingJobSummaryList {
					var eagerTags map[string]any
					if conn.Filters.General.HasTags() {
						tags, err := getSagemakerTags(ctx, svc, job.LabelingJobArn)
						if err != nil {
							return nil, err
						}
						if conn.Filters.General.IsFilteredOutByTags(mapStringInterfaceToStringString(tags)) {
							log.Debug().Interface("labelingJob", job.LabelingJobArn).Msg("skipping sagemaker labeling job due to filters")
							continue
						}
						eagerTags = tags
					}

					mqlJob, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerLabelingJob,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(job.LabelingJobArn),
							"name":           llx.StringDataPtr(job.LabelingJobName),
							"region":         llx.StringData(region),
							"status":         llx.StringData(string(job.LabelingJobStatus)),
							"createdAt":      llx.TimeDataPtr(job.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(job.LastModifiedTime),
						})
					if err != nil {
						return nil, err
					}
					lj := mqlJob.(*mqlAwsSagemakerLabelingJob)
					if eagerTags != nil {
						lj.cacheTags = eagerTags
						lj.tagsFetched = true
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

type mqlAwsSagemakerLabelingJobInternal struct {
	sagemakerTagsCache
	fetched       bool
	fetchLock     sync.Mutex
	cacheDescribe *sagemaker.DescribeLabelingJobOutput
}

func (a *mqlAwsSagemakerLabelingJob) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerLabelingJob) tags() (map[string]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	return a.fetchTags(conn, a.Region.Data, a.Arn.Data)
}

func (a *mqlAwsSagemakerLabelingJob) fetchDetails() (*sagemaker.DescribeLabelingJobOutput, error) {
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

	resp, err := svc.DescribeLabelingJob(ctx, &sagemaker.DescribeLabelingJobInput{
		LabelingJobName: &name,
	})
	if err != nil {
		return nil, err
	}
	a.cacheDescribe = resp
	a.fetched = true
	return resp, nil
}

func (a *mqlAwsSagemakerLabelingJob) inputConfig() (*mqlAwsSagemakerLabelingJobInputConfig, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if resp.InputConfig == nil {
		a.InputConfig.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	var s3Uri string
	if resp.InputConfig.DataSource != nil && resp.InputConfig.DataSource.S3DataSource != nil {
		s3Uri = convert.ToValue(resp.InputConfig.DataSource.S3DataSource.ManifestS3Uri)
	}
	dataAttrs, _ := convert.JsonToDict(resp.InputConfig.DataAttributes)

	mqlRes, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerLabelingJobInputConfig,
		map[string]*llx.RawData{
			"s3Uri":          llx.StringData(s3Uri),
			"dataAttributes": llx.DictData(dataAttrs),
		})
	if err != nil {
		return nil, err
	}
	ic := mqlRes.(*mqlAwsSagemakerLabelingJobInputConfig)
	ic.cacheParentArn = a.Arn.Data
	return ic, nil
}

func (a *mqlAwsSagemakerLabelingJob) outputConfig() (*mqlAwsSagemakerLabelingJobOutputConfig, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if resp.OutputConfig == nil {
		a.OutputConfig.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	mqlRes, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerLabelingJobOutputConfig,
		map[string]*llx.RawData{
			"s3Uri": llx.StringDataPtr(resp.OutputConfig.S3OutputPath),
		})
	if err != nil {
		return nil, err
	}
	oc := mqlRes.(*mqlAwsSagemakerLabelingJobOutputConfig)
	oc.cacheParentArn = a.Arn.Data
	oc.cacheKmsKeyId = resp.OutputConfig.KmsKeyId
	return oc, nil
}

func (a *mqlAwsSagemakerLabelingJob) iamRole() (*mqlAwsIamRole, error) {
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

func (a *mqlAwsSagemakerLabelingJob) humanTaskConfig() (*mqlAwsSagemakerLabelingJobHumanTaskConfig, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if resp.HumanTaskConfig == nil {
		a.HumanTaskConfig.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	htc := resp.HumanTaskConfig

	var numWorkers, timeLimit, maxConcurrent int64
	if htc.NumberOfHumanWorkersPerDataObject != nil {
		numWorkers = int64(*htc.NumberOfHumanWorkersPerDataObject)
	}
	if htc.TaskTimeLimitInSeconds != nil {
		timeLimit = int64(*htc.TaskTimeLimitInSeconds)
	}
	if htc.MaxConcurrentTaskCount != nil {
		maxConcurrent = int64(*htc.MaxConcurrentTaskCount)
	}

	mqlRes, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerLabelingJobHumanTaskConfig,
		map[string]*llx.RawData{
			"workteamArn":                       llx.StringDataPtr(htc.WorkteamArn),
			"taskTitle":                         llx.StringDataPtr(htc.TaskTitle),
			"taskDescription":                   llx.StringDataPtr(htc.TaskDescription),
			"numberOfHumanWorkersPerDataObject": llx.IntData(numWorkers),
			"taskTimeLimitInSeconds":            llx.IntData(timeLimit),
			"annotationConsolidationLambdaArn":  llx.StringDataPtr(htc.AnnotationConsolidationConfig.AnnotationConsolidationLambdaArn),
			"preHumanTaskLambdaArn":             llx.StringDataPtr(htc.PreHumanTaskLambdaArn),
			"maxConcurrentTaskCount":            llx.IntData(maxConcurrent),
		})
	if err != nil {
		return nil, err
	}
	htcRes := mqlRes.(*mqlAwsSagemakerLabelingJobHumanTaskConfig)
	htcRes.cacheParentArn = a.Arn.Data
	return htcRes, nil
}

func (a *mqlAwsSagemakerLabelingJob) failureReason() (string, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return "", err
	}
	return convert.ToValue(resp.FailureReason), nil
}

// ---- Labeling Job Sub-resources ----

type mqlAwsSagemakerLabelingJobInputConfigInternal struct {
	cacheParentArn string
}

func (a *mqlAwsSagemakerLabelingJobInputConfig) id() (string, error) {
	return a.cacheParentArn + "/inputConfig", nil
}

type mqlAwsSagemakerLabelingJobOutputConfigInternal struct {
	cacheParentArn string
	cacheKmsKeyId  *string
}

func (a *mqlAwsSagemakerLabelingJobOutputConfig) id() (string, error) {
	return a.cacheParentArn + "/outputConfig", nil
}

func (a *mqlAwsSagemakerLabelingJobOutputConfig) kmsKey() (*mqlAwsKmsKey, error) {
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

type mqlAwsSagemakerLabelingJobHumanTaskConfigInternal struct {
	cacheParentArn string
}

func (a *mqlAwsSagemakerLabelingJobHumanTaskConfig) id() (string, error) {
	return a.cacheParentArn + "/humanTaskConfig", nil
}

func (a *mqlAwsSagemakerLabelingJobHumanTaskConfig) workteam() (*mqlAwsSagemakerWorkteam, error) {
	workteamArn := a.WorkteamArn.Data
	if workteamArn == "" {
		a.Workteam.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, ResourceAwsSagemakerWorkteam,
		map[string]*llx.RawData{"arn": llx.StringData(workteamArn)})
	if err != nil {
		// Cross-lookup may fail if workteam is not in the list; return null
		a.Workteam.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return res.(*mqlAwsSagemakerWorkteam), nil
}

// ---- Workforces ----

func (a *mqlAwsSagemaker) workforces() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getWorkforces(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getWorkforces(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListWorkforcesPaginator(svc, &sagemaker.ListWorkforcesInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS SageMaker workforces")
						return res, nil
					}
					return nil, err
				}

				for _, wf := range page.Workforces {
					mqlWF, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerWorkforce,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(wf.WorkforceArn),
							"name":           llx.StringDataPtr(wf.WorkforceName),
							"region":         llx.StringData(region),
							"status":         llx.StringData(string(wf.Status)),
							"createdAt":      llx.TimeDataPtr(wf.CreateDate),
							"lastModifiedAt": llx.TimeDataPtr(wf.LastUpdatedDate),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlWF)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerWorkforceInternal struct {
	fetched       bool
	fetchLock     sync.Mutex
	cacheDescribe *sagemaker.DescribeWorkforceOutput
}

func (a *mqlAwsSagemakerWorkforce) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerWorkforce) fetchDetails() (*sagemaker.DescribeWorkforceOutput, error) {
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

	resp, err := svc.DescribeWorkforce(ctx, &sagemaker.DescribeWorkforceInput{
		WorkforceName: &name,
	})
	if err != nil {
		return nil, err
	}
	a.cacheDescribe = resp
	a.fetched = true
	return resp, nil
}

func (a *mqlAwsSagemakerWorkforce) cidrs() ([]any, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if resp.Workforce == nil || resp.Workforce.SourceIpConfig == nil {
		return nil, nil
	}
	res := make([]any, 0, len(resp.Workforce.SourceIpConfig.Cidrs))
	for _, cidr := range resp.Workforce.SourceIpConfig.Cidrs {
		res = append(res, cidr)
	}
	return res, nil
}

func (a *mqlAwsSagemakerWorkforce) cognitoConfig() (*mqlAwsSagemakerWorkforceCognitoConfig, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if resp.Workforce == nil || resp.Workforce.CognitoConfig == nil {
		a.CognitoConfig.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	cc := resp.Workforce.CognitoConfig
	mqlRes, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerWorkforceCognitoConfig,
		map[string]*llx.RawData{
			"userPool": llx.StringDataPtr(cc.UserPool),
			"clientId": llx.StringDataPtr(cc.ClientId),
		})
	if err != nil {
		return nil, err
	}
	ccRes := mqlRes.(*mqlAwsSagemakerWorkforceCognitoConfig)
	ccRes.cacheWorkforceArn = a.Arn.Data
	return ccRes, nil
}

func (a *mqlAwsSagemakerWorkforce) oidcConfig() (*mqlAwsSagemakerWorkforceOidcConfig, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if resp.Workforce == nil || resp.Workforce.OidcConfig == nil {
		a.OidcConfig.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	oc := resp.Workforce.OidcConfig
	mqlRes, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerWorkforceOidcConfig,
		map[string]*llx.RawData{
			"issuer":                llx.StringDataPtr(oc.Issuer),
			"clientId":              llx.StringDataPtr(oc.ClientId),
			"authorizationEndpoint": llx.StringDataPtr(oc.AuthorizationEndpoint),
			"tokenEndpoint":         llx.StringDataPtr(oc.TokenEndpoint),
			"userInfoEndpoint":      llx.StringDataPtr(oc.UserInfoEndpoint),
			"logoutEndpoint":        llx.StringDataPtr(oc.LogoutEndpoint),
			"jwksUri":               llx.StringDataPtr(oc.JwksUri),
		})
	if err != nil {
		return nil, err
	}
	ocRes := mqlRes.(*mqlAwsSagemakerWorkforceOidcConfig)
	ocRes.cacheWorkforceArn = a.Arn.Data
	return ocRes, nil
}

// ---- Workforce Sub-resources ----

type mqlAwsSagemakerWorkforceCognitoConfigInternal struct {
	cacheWorkforceArn string
}

func (a *mqlAwsSagemakerWorkforceCognitoConfig) id() (string, error) {
	return a.cacheWorkforceArn + "/cognitoConfig", nil
}

type mqlAwsSagemakerWorkforceOidcConfigInternal struct {
	cacheWorkforceArn string
}

func (a *mqlAwsSagemakerWorkforceOidcConfig) id() (string, error) {
	return a.cacheWorkforceArn + "/oidcConfig", nil
}

// ---- Work Teams ----

func (a *mqlAwsSagemaker) workteams() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getWorkteams(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getWorkteams(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListWorkteamsPaginator(svc, &sagemaker.ListWorkteamsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS SageMaker work teams")
						return res, nil
					}
					return nil, err
				}

				for _, wt := range page.Workteams {
					var notifArn string
					if wt.NotificationConfiguration != nil && wt.NotificationConfiguration.NotificationTopicArn != nil {
						notifArn = *wt.NotificationConfiguration.NotificationTopicArn
					}

					mqlWT, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerWorkteam,
						map[string]*llx.RawData{
							"arn":                  llx.StringDataPtr(wt.WorkteamArn),
							"name":                 llx.StringDataPtr(wt.WorkteamName),
							"description":          llx.StringDataPtr(wt.Description),
							"region":               llx.StringData(region),
							"createdAt":            llx.TimeDataPtr(wt.CreateDate),
							"lastModifiedAt":       llx.TimeDataPtr(wt.LastUpdatedDate),
							"notificationTopicArn": llx.StringData(notifArn),
						})
					if err != nil {
						return nil, err
					}
					wtRes := mqlWT.(*mqlAwsSagemakerWorkteam)
					wtRes.cacheMembers = wt.MemberDefinitions
					wtRes.cacheWorkforceArn = wt.WorkforceArn
					res = append(res, mqlWT)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerWorkteamInternal struct {
	cacheMembers      []sagemakerTypes.MemberDefinition
	cacheWorkforceArn *string
}

func (a *mqlAwsSagemakerWorkteam) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerWorkteam) memberDefinitions() ([]any, error) {
	res := make([]any, 0, len(a.cacheMembers))
	for i, md := range a.cacheMembers {
		var cognitoPool, cognitoGroup, cognitoClient, oidcGroup string
		if md.CognitoMemberDefinition != nil {
			cognitoPool = convert.ToValue(md.CognitoMemberDefinition.UserPool)
			cognitoGroup = convert.ToValue(md.CognitoMemberDefinition.UserGroup)
			cognitoClient = convert.ToValue(md.CognitoMemberDefinition.ClientId)
		}
		if md.OidcMemberDefinition != nil && len(md.OidcMemberDefinition.Groups) > 0 {
			oidcGroup = md.OidcMemberDefinition.Groups[0]
		}
		mqlMD, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerWorkteamMemberDefinition,
			map[string]*llx.RawData{
				"cognitoUserPool":  llx.StringData(cognitoPool),
				"cognitoUserGroup": llx.StringData(cognitoGroup),
				"cognitoClientId":  llx.StringData(cognitoClient),
				"oidcGroupName":    llx.StringData(oidcGroup),
			})
		if err != nil {
			return nil, err
		}
		mdRes := mqlMD.(*mqlAwsSagemakerWorkteamMemberDefinition)
		mdRes.cacheParentArn = a.Arn.Data
		mdRes.cacheIndex = i
		res = append(res, mqlMD)
	}
	return res, nil
}

func (a *mqlAwsSagemakerWorkteam) notificationTopic() (*mqlAwsSnsTopic, error) {
	topicArn := a.NotificationTopicArn.Data
	if topicArn == "" {
		a.NotificationTopic.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, "aws.sns.topic",
		map[string]*llx.RawData{"arn": llx.StringData(topicArn)})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAwsSnsTopic), nil
}

func (a *mqlAwsSagemakerWorkteam) workforce() (*mqlAwsSagemakerWorkforce, error) {
	if a.cacheWorkforceArn == nil || *a.cacheWorkforceArn == "" {
		a.Workforce.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, ResourceAwsSagemakerWorkforce,
		map[string]*llx.RawData{"arn": llx.StringDataPtr(a.cacheWorkforceArn)})
	if err != nil {
		// Workforce may not be in the list; return null
		a.Workforce.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return res.(*mqlAwsSagemakerWorkforce), nil
}

// ---- Work Team Sub-resources ----

type mqlAwsSagemakerWorkteamMemberDefinitionInternal struct {
	cacheParentArn string
	cacheIndex     int
}

func (a *mqlAwsSagemakerWorkteamMemberDefinition) id() (string, error) {
	return a.cacheParentArn + "/memberDefinition/" + a.CognitoUserGroup.Data + "/" + a.OidcGroupName.Data, nil
}

// ---- Flow Definitions ----

func (a *mqlAwsSagemaker) flowDefinitions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getFlowDefinitions(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getFlowDefinitions(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListFlowDefinitionsPaginator(svc, &sagemaker.ListFlowDefinitionsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS SageMaker flow definitions")
						return res, nil
					}
					return nil, err
				}

				for _, fd := range page.FlowDefinitionSummaries {
					mqlFD, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerFlowDefinition,
						map[string]*llx.RawData{
							"arn":       llx.StringDataPtr(fd.FlowDefinitionArn),
							"name":      llx.StringDataPtr(fd.FlowDefinitionName),
							"region":    llx.StringData(region),
							"status":    llx.StringData(string(fd.FlowDefinitionStatus)),
							"createdAt": llx.TimeDataPtr(fd.CreationTime),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlFD)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerFlowDefinitionInternal struct {
	fetched       bool
	fetchLock     sync.Mutex
	cacheDescribe *sagemaker.DescribeFlowDefinitionOutput
}

func (a *mqlAwsSagemakerFlowDefinition) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerFlowDefinition) fetchDetails() (*sagemaker.DescribeFlowDefinitionOutput, error) {
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

	resp, err := svc.DescribeFlowDefinition(ctx, &sagemaker.DescribeFlowDefinitionInput{
		FlowDefinitionName: &name,
	})
	if err != nil {
		return nil, err
	}
	a.cacheDescribe = resp
	a.fetched = true
	return resp, nil
}

func (a *mqlAwsSagemakerFlowDefinition) humanLoopConfig() (*mqlAwsSagemakerFlowDefinitionHumanLoopConfig, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if resp.HumanLoopConfig == nil {
		a.HumanLoopConfig.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	hlc := resp.HumanLoopConfig

	var taskCount, timeLimit int64
	if hlc.TaskCount != nil {
		taskCount = int64(*hlc.TaskCount)
	}
	if hlc.TaskTimeLimitInSeconds != nil {
		timeLimit = int64(*hlc.TaskTimeLimitInSeconds)
	}

	mqlRes, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerFlowDefinitionHumanLoopConfig,
		map[string]*llx.RawData{
			"workteamArn":            llx.StringDataPtr(hlc.WorkteamArn),
			"humanTaskUiArn":         llx.StringDataPtr(hlc.HumanTaskUiArn),
			"taskTitle":              llx.StringDataPtr(hlc.TaskTitle),
			"taskDescription":        llx.StringDataPtr(hlc.TaskDescription),
			"taskCount":              llx.IntData(taskCount),
			"taskTimeLimitInSeconds": llx.IntData(timeLimit),
		})
	if err != nil {
		return nil, err
	}
	hlcRes := mqlRes.(*mqlAwsSagemakerFlowDefinitionHumanLoopConfig)
	hlcRes.cacheParentArn = a.Arn.Data
	return hlcRes, nil
}

func (a *mqlAwsSagemakerFlowDefinition) outputConfig() (*mqlAwsSagemakerFlowDefinitionOutputConfig, error) {
	resp, err := a.fetchDetails()
	if err != nil {
		return nil, err
	}
	if resp.OutputConfig == nil {
		a.OutputConfig.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}

	mqlRes, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerFlowDefinitionOutputConfig,
		map[string]*llx.RawData{
			"s3Uri": llx.StringDataPtr(resp.OutputConfig.S3OutputPath),
		})
	if err != nil {
		return nil, err
	}
	ocRes := mqlRes.(*mqlAwsSagemakerFlowDefinitionOutputConfig)
	ocRes.cacheParentArn = a.Arn.Data
	ocRes.cacheKmsKeyId = resp.OutputConfig.KmsKeyId
	return ocRes, nil
}

func (a *mqlAwsSagemakerFlowDefinition) iamRole() (*mqlAwsIamRole, error) {
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

// ---- Flow Definition Sub-resources ----

type mqlAwsSagemakerFlowDefinitionHumanLoopConfigInternal struct {
	cacheParentArn string
}

func (a *mqlAwsSagemakerFlowDefinitionHumanLoopConfig) id() (string, error) {
	return a.cacheParentArn + "/humanLoopConfig", nil
}

func (a *mqlAwsSagemakerFlowDefinitionHumanLoopConfig) workteam() (*mqlAwsSagemakerWorkteam, error) {
	workteamArn := a.WorkteamArn.Data
	if workteamArn == "" {
		a.Workteam.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(a.MqlRuntime, ResourceAwsSagemakerWorkteam,
		map[string]*llx.RawData{"arn": llx.StringData(workteamArn)})
	if err != nil {
		// Cross-lookup may fail if workteam is not in the list; return null
		a.Workteam.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return res.(*mqlAwsSagemakerWorkteam), nil
}

type mqlAwsSagemakerFlowDefinitionOutputConfigInternal struct {
	cacheParentArn string
	cacheKmsKeyId  *string
}

func (a *mqlAwsSagemakerFlowDefinitionOutputConfig) id() (string, error) {
	return a.cacheParentArn + "/outputConfig", nil
}

func (a *mqlAwsSagemakerFlowDefinitionOutputConfig) kmsKey() (*mqlAwsKmsKey, error) {
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
