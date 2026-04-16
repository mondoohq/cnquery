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
	"go.mondoo.com/mql/v13/types"
)

// ---- Actions ----

func (a *mqlAwsSagemaker) actions() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getActions(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getActions(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListActionsPaginator(svc, &sagemaker.ListActionsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS SageMaker actions")
						return res, nil
					}
					return nil, err
				}

				for _, action := range page.ActionSummaries {
					props := make(map[string]any)
					// Properties are not available in list summary; lazy-load via describe
					mqlAction, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerAction,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(action.ActionArn),
							"name":           llx.StringDataPtr(action.ActionName),
							"region":         llx.StringData(region),
							"status":         llx.StringData(string(action.Status)),
							"createdAt":      llx.TimeDataPtr(action.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(action.LastModifiedTime),
							"actionType":     llx.StringDataPtr(action.ActionType),
							"properties":     llx.MapData(props, types.String),
						})
					if err != nil {
						return nil, err
					}
					act := mqlAction.(*mqlAwsSagemakerAction)
					if action.Source != nil {
						act.cacheSourceUri = action.Source.SourceUri
						act.cacheSourceType = action.Source.SourceType
						act.cacheSourceId = action.Source.SourceId
					}
					res = append(res, mqlAction)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerActionInternal struct {
	detailsFetched  bool
	detailsLock     sync.Mutex
	cacheSourceUri  *string
	cacheSourceType *string
	cacheSourceId   *string
	cacheProperties map[string]string
}

func (a *mqlAwsSagemakerAction) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerAction) fetchDetails() error {
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

	resp, err := svc.DescribeAction(ctx, &sagemaker.DescribeActionInput{
		ActionName: &name,
	})
	if err != nil {
		return err
	}

	if resp.Source != nil {
		a.cacheSourceUri = resp.Source.SourceUri
		a.cacheSourceType = nil
		a.cacheSourceId = nil
	}
	a.cacheProperties = resp.Properties
	a.detailsFetched = true
	return nil
}

func (a *mqlAwsSagemakerAction) source() (*mqlAwsSagemakerLineageSource, error) {
	// Source info is eagerly cached from list, but fetch details for completeness
	if a.cacheSourceUri == nil {
		if err := a.fetchDetails(); err != nil {
			return nil, err
		}
	}
	if a.cacheSourceUri == nil || *a.cacheSourceUri == "" {
		a.Source.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return sagemakerCreateLineageSource(a.MqlRuntime, a.Arn.Data, a.cacheSourceUri, a.cacheSourceType, a.cacheSourceId)
}

// ---- Artifacts ----

func (a *mqlAwsSagemaker) artifacts() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getArtifacts(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getArtifacts(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListArtifactsPaginator(svc, &sagemaker.ListArtifactsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS SageMaker artifacts")
						return res, nil
					}
					return nil, err
				}

				for _, artifact := range page.ArtifactSummaries {
					props := make(map[string]any)
					mqlArtifact, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerArtifact,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(artifact.ArtifactArn),
							"region":         llx.StringData(region),
							"createdAt":      llx.TimeDataPtr(artifact.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(artifact.LastModifiedTime),
							"artifactType":   llx.StringDataPtr(artifact.ArtifactType),
							"properties":     llx.MapData(props, types.String),
						})
					if err != nil {
						return nil, err
					}
					art := mqlArtifact.(*mqlAwsSagemakerArtifact)
					if artifact.Source != nil {
						art.cacheSourceUri = artifact.Source.SourceUri
						// ArtifactSource only has SourceUri and SourceTypes (plural)
						art.cacheSourceType = nil
						art.cacheSourceId = nil
					}
					res = append(res, mqlArtifact)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerArtifactInternal struct {
	detailsFetched  bool
	detailsLock     sync.Mutex
	cacheSourceUri  *string
	cacheSourceType *string
	cacheSourceId   *string
	cacheProperties map[string]string
}

func (a *mqlAwsSagemakerArtifact) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerArtifact) fetchDetails() error {
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
	arn := a.Arn.Data

	resp, err := svc.DescribeArtifact(ctx, &sagemaker.DescribeArtifactInput{
		ArtifactArn: &arn,
	})
	if err != nil {
		return err
	}

	if resp.Source != nil {
		a.cacheSourceUri = resp.Source.SourceUri
		a.cacheSourceType = nil
		a.cacheSourceId = nil
	}
	a.cacheProperties = resp.Properties
	a.detailsFetched = true
	return nil
}

func (a *mqlAwsSagemakerArtifact) source() (*mqlAwsSagemakerLineageSource, error) {
	if a.cacheSourceUri == nil {
		if err := a.fetchDetails(); err != nil {
			return nil, err
		}
	}
	if a.cacheSourceUri == nil || *a.cacheSourceUri == "" {
		a.Source.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return sagemakerCreateLineageSource(a.MqlRuntime, a.Arn.Data, a.cacheSourceUri, a.cacheSourceType, a.cacheSourceId)
}

// ---- Contexts ----

func (a *mqlAwsSagemaker) contexts() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getContexts(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getContexts(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListContextsPaginator(svc, &sagemaker.ListContextsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS SageMaker contexts")
						return res, nil
					}
					return nil, err
				}

				for _, smCtx := range page.ContextSummaries {
					props := make(map[string]any)
					mqlCtx, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerContext,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(smCtx.ContextArn),
							"name":           llx.StringDataPtr(smCtx.ContextName),
							"region":         llx.StringData(region),
							"createdAt":      llx.TimeDataPtr(smCtx.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(smCtx.LastModifiedTime),
							"contextType":    llx.StringDataPtr(smCtx.ContextType),
							"properties":     llx.MapData(props, types.String),
						})
					if err != nil {
						return nil, err
					}
					ctxRes := mqlCtx.(*mqlAwsSagemakerContext)
					if smCtx.Source != nil {
						ctxRes.cacheSourceUri = smCtx.Source.SourceUri
						ctxRes.cacheSourceType = smCtx.Source.SourceType
						ctxRes.cacheSourceId = smCtx.Source.SourceId
					}
					res = append(res, mqlCtx)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

type mqlAwsSagemakerContextInternal struct {
	detailsFetched  bool
	detailsLock     sync.Mutex
	cacheSourceUri  *string
	cacheSourceType *string
	cacheSourceId   *string
	cacheProperties map[string]string
}

func (a *mqlAwsSagemakerContext) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsSagemakerContext) fetchDetails() error {
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

	resp, err := svc.DescribeContext(ctx, &sagemaker.DescribeContextInput{
		ContextName: &name,
	})
	if err != nil {
		return err
	}

	if resp.Source != nil {
		a.cacheSourceUri = resp.Source.SourceUri
		a.cacheSourceType = nil
		a.cacheSourceId = nil
	}
	a.cacheProperties = resp.Properties
	a.detailsFetched = true
	return nil
}

func (a *mqlAwsSagemakerContext) source() (*mqlAwsSagemakerLineageSource, error) {
	if a.cacheSourceUri == nil {
		if err := a.fetchDetails(); err != nil {
			return nil, err
		}
	}
	if a.cacheSourceUri == nil || *a.cacheSourceUri == "" {
		a.Source.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	return sagemakerCreateLineageSource(a.MqlRuntime, a.Arn.Data, a.cacheSourceUri, a.cacheSourceType, a.cacheSourceId)
}

// ---- Lineage Source (shared sub-resource) ----

type mqlAwsSagemakerLineageSourceInternal struct {
	cacheParentArn string
}

func (a *mqlAwsSagemakerLineageSource) id() (string, error) {
	return a.cacheParentArn + "/source/" + a.SourceUri.Data, nil
}

// sagemakerCreateLineageSource creates the shared lineage.source sub-resource.
func sagemakerCreateLineageSource(runtime *plugin.Runtime, parentArn string, sourceUri, sourceType, sourceId *string) (*mqlAwsSagemakerLineageSource, error) {
	mqlRes, err := CreateResource(runtime, ResourceAwsSagemakerLineageSource,
		map[string]*llx.RawData{
			"sourceUri":  llx.StringData(convert.ToValue(sourceUri)),
			"sourceType": llx.StringData(convert.ToValue(sourceType)),
			"sourceId":   llx.StringData(convert.ToValue(sourceId)),
		})
	if err != nil {
		return nil, err
	}
	src := mqlRes.(*mqlAwsSagemakerLineageSource)
	src.cacheParentArn = parentArn
	return src, nil
}

// ---- Lineage Groups ----

func (a *mqlAwsSagemaker) lineageGroups() ([]any, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	res := []any{}
	poolOfJobs := jobpool.CreatePool(a.getLineageGroups(conn), 5)
	poolOfJobs.Run()

	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}

	return res, nil
}

func (a *mqlAwsSagemaker) getLineageGroups(conn *connection.AwsConnection) []*jobpool.Job {
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

			paginator := sagemaker.NewListLineageGroupsPaginator(svc, &sagemaker.ListLineageGroupsInput{})
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					if Is400AccessDeniedError(err) {
						log.Warn().Str("region", region).Msg("error accessing region for AWS SageMaker lineage groups")
						return res, nil
					}
					return nil, err
				}

				for _, lg := range page.LineageGroupSummaries {
					mqlLG, err := CreateResource(a.MqlRuntime, ResourceAwsSagemakerLineageGroup,
						map[string]*llx.RawData{
							"arn":            llx.StringDataPtr(lg.LineageGroupArn),
							"name":           llx.StringDataPtr(lg.LineageGroupName),
							"displayName":    llx.StringDataPtr(lg.DisplayName),
							"region":         llx.StringData(region),
							"createdAt":      llx.TimeDataPtr(lg.CreationTime),
							"lastModifiedAt": llx.TimeDataPtr(lg.LastModifiedTime),
						})
					if err != nil {
						return nil, err
					}
					res = append(res, mqlLG)
				}
			}
			return jobpool.JobResult(res), nil
		}
		tasks = append(tasks, jobpool.NewJob(f))
	}
	return tasks
}

func (a *mqlAwsSagemakerLineageGroup) id() (string, error) {
	return a.Arn.Data, nil
}
