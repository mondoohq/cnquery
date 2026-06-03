// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	"go.mondoo.com/mql/v13/types"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// vertexaiRegions lists the known Vertex AI regions to iterate when listing resources.
// TODO: Replace with a dynamic location listing API call when available to avoid
// missing resources in newly added GCP regions. Last updated: 2026-03.
// See: https://cloud.google.com/vertex-ai/docs/general/locations
var vertexaiRegions = []string{
	"us-central1",
	"us-east1",
	"us-east4",
	"us-south1",
	"us-west1",
	"us-west2",
	"us-west4",
	"northamerica-northeast1",
	"northamerica-northeast2",
	"southamerica-east1",
	"europe-central2",
	"europe-north1",
	"europe-southwest1",
	"europe-west1",
	"europe-west2",
	"europe-west3",
	"europe-west4",
	"europe-west6",
	"europe-west8",
	"europe-west9",
	"asia-east1",
	"asia-east2",
	"asia-northeast1",
	"asia-northeast3",
	"asia-south1",
	"asia-southeast1",
	"asia-southeast2",
	"australia-southeast1",
	"australia-southeast2",
	"me-central1",
	"me-central2",
	"me-west1",
	"africa-south1",
}

// vertexaiMaxConcurrency is the maximum number of regions to query concurrently.
const vertexaiMaxConcurrency = 10

func vertexaiEndpoint(region string) string {
	return fmt.Sprintf("%s-aiplatform.googleapis.com:443", region)
}

// isVertexAIRegionSkippable returns true for errors indicating the Vertex AI API
// is not enabled or the region is not supported for this project.
func isVertexAIRegionSkippable(err error) bool {
	if s, ok := status.FromError(err); ok {
		switch s.Code() {
		case codes.PermissionDenied, codes.Unimplemented:
			return true
		case codes.InvalidArgument, codes.NotFound:
			// "not enabled" and "is not supported" surface as these codes
			return true
		}
	}
	return false
}

type mqlGcpProjectVertexaiServiceInternal struct {
	// skippedRegions tracks regions where the API is not available.
	// We track skipped (not enabled) so that each resource type can
	// independently discover regions — a region enabled for models
	// but not endpoints won't be lost.
	skippedRegions map[string]bool
	lock           sync.Mutex
}

// getRegions returns the list of candidate regions, excluding any that have
// been previously marked as skipped.
func (g *mqlGcpProjectVertexaiService) getRegions() []string {
	g.lock.Lock()
	defer g.lock.Unlock()
	if len(g.skippedRegions) == 0 {
		return vertexaiRegions
	}
	regions := make([]string, 0, len(vertexaiRegions))
	for _, r := range vertexaiRegions {
		if !g.skippedRegions[r] {
			regions = append(regions, r)
		}
	}
	return regions
}

// markRegionSkipped records a region where the API returned a skippable error.
func (g *mqlGcpProjectVertexaiService) markRegionSkipped(region string) {
	g.lock.Lock()
	defer g.lock.Unlock()
	if g.skippedRegions == nil {
		g.skippedRegions = make(map[string]bool)
	}
	g.skippedRegions[region] = true
}

// vertexaiRegionResult holds the result of listing resources in a single region.
type vertexaiRegionResult struct {
	items   []any
	skipped bool
	err     error
	region  string
}

func (g *mqlGcpProject) vertexai() (*mqlGcpProjectVertexaiService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	res, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService", map[string]*llx.RawData{
		"projectId": llx.StringData(g.Id.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectVertexaiService), nil
}

func initGcpProjectVertexaiService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}
	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	args["projectId"] = llx.StringData(conn.ResourceID())
	return args, nil, nil
}

func (g *mqlGcpProjectVertexaiService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("gcp.project/%s/vertexaiService", g.ProjectId.Data), nil
}

// listAcrossRegions runs fn concurrently across all candidate regions with bounded
// concurrency. It collects results, marks skipped regions, and returns the aggregated items.
func (g *mqlGcpProjectVertexaiService) listAcrossRegions(
	fn func(ctx context.Context, region string) ([]any, bool, error),
) ([]any, error) {
	regions := g.getRegions()
	ctx := context.Background()

	results := make([]vertexaiRegionResult, len(regions))
	sem := make(chan struct{}, vertexaiMaxConcurrency)
	var wg sync.WaitGroup

	for i, region := range regions {
		wg.Add(1)
		go func(idx int, r string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			items, skipped, err := fn(ctx, r)
			results[idx] = vertexaiRegionResult{items: items, skipped: skipped, err: err, region: r}
		}(i, region)
	}
	wg.Wait()

	var res []any
	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}
		if r.skipped {
			g.markRegionSkipped(r.region)
		}
		res = append(res, r.items...)
	}
	return res, nil
}

func (g *mqlGcpProjectVertexaiService) models() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewModelClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListModels(ctx, &aiplatformpb.ListModelsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			model, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			mqlModel, err := newMqlVertexaiModel(g.MqlRuntime, model)
			if err != nil {
				return nil, false, err
			}
			items = append(items, mqlModel)
		}
		return items, false, nil
	})
}

// newMqlVertexaiModel maps a Vertex AI Model proto into an MQL resource. It is
// shared by the models() lister and the deployment.model() reference.
func newMqlVertexaiModel(runtime *plugin.Runtime, model *aiplatformpb.Model) (*mqlGcpProjectVertexaiServiceModel, error) {
	modelSourceInfo, err := protoToDict(model.ModelSourceInfo)
	if err != nil {
		return nil, err
	}
	containerSpec, err := protoToDict(model.ContainerSpec)
	if err != nil {
		return nil, err
	}
	encryptionSpec, err := protoToDict(model.EncryptionSpec)
	if err != nil {
		return nil, err
	}

	deploymentTypes := make([]any, 0, len(model.SupportedDeploymentResourcesTypes))
	for _, dt := range model.SupportedDeploymentResourcesTypes {
		deploymentTypes = append(deploymentTypes, dt.String())
	}

	mqlModel, err := CreateResource(runtime, "gcp.project.vertexaiService.model", map[string]*llx.RawData{
		"name":                              llx.StringData(model.Name),
		"displayName":                       llx.StringData(model.DisplayName),
		"description":                       llx.StringData(model.Description),
		"versionId":                         llx.StringData(model.VersionId),
		"versionAliases":                    llx.ArrayData(convert.SliceAnyToInterface(model.VersionAliases), types.String),
		"versionDescription":                llx.StringData(model.VersionDescription),
		"modelSourceInfo":                   llx.DictData(modelSourceInfo),
		"containerSpec":                     llx.DictData(containerSpec),
		"supportedDeploymentResourcesTypes": llx.ArrayData(deploymentTypes, types.String),
		"supportedInputStorageFormats":      llx.ArrayData(convert.SliceAnyToInterface(model.SupportedInputStorageFormats), types.String),
		"supportedOutputStorageFormats":     llx.ArrayData(convert.SliceAnyToInterface(model.SupportedOutputStorageFormats), types.String),
		"trainingPipeline":                  llx.StringData(model.TrainingPipeline),
		"artifactUri":                       llx.StringData(model.ArtifactUri),
		"encryptionSpec":                    llx.DictData(encryptionSpec),
		"labels":                            llx.MapData(convert.MapToInterfaceMap(model.Labels), types.String),
		"etag":                              llx.StringData(model.Etag),
		"createdAt":                         llx.TimeDataPtr(timestampAsTimePtr(model.CreateTime)),
		"updatedAt":                         llx.TimeDataPtr(timestampAsTimePtr(model.UpdateTime)),
	})
	if err != nil {
		return nil, err
	}
	res := mqlModel.(*mqlGcpProjectVertexaiServiceModel)
	res.cacheKmsKeyName = model.GetEncryptionSpec().GetKmsKeyName()
	return res, nil
}

func (g *mqlGcpProjectVertexaiServiceModel) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) endpoints() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewEndpointClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListEndpoints(ctx, &aiplatformpb.ListEndpointsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			ep, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			mqlEndpoint, err := newMqlVertexaiEndpoint(g.MqlRuntime, projectId, ep)
			if err != nil {
				return nil, false, err
			}
			items = append(items, mqlEndpoint)
		}
		return items, false, nil
	})
}

// newMqlVertexaiEndpoint maps a Vertex AI Endpoint proto into an MQL resource,
// including its deployed-model sub-resources. It is shared by the endpoints()
// lister and the typed endpoint() references.
func newMqlVertexaiEndpoint(runtime *plugin.Runtime, projectId string, ep *aiplatformpb.Endpoint) (*mqlGcpProjectVertexaiServiceEndpoint, error) {
	deployedModels := make([]any, 0, len(ep.DeployedModels))
	deployments := make([]any, 0, len(ep.DeployedModels))
	for _, dm := range ep.DeployedModels {
		d, err := protoToDict(dm)
		if err != nil {
			return nil, err
		}
		deployedModels = append(deployedModels, d)

		mqlDeployment, err := CreateResource(runtime, "gcp.project.vertexaiService.endpoint.deployment", map[string]*llx.RawData{
			"__id":                    llx.StringData(fmt.Sprintf("%s/deployment/%s", ep.Name, dm.Id)),
			"id":                      llx.StringData(dm.Id),
			"displayName":             llx.StringData(dm.DisplayName),
			"modelVersionId":          llx.StringData(dm.ModelVersionId),
			"serviceAccountEmail":     llx.StringData(dm.ServiceAccount),
			"disableContainerLogging": llx.BoolData(dm.DisableContainerLogging),
			"enableAccessLogging":     llx.BoolData(dm.EnableAccessLogging),
			"createdAt":               llx.TimeDataPtr(timestampAsTimePtr(dm.CreateTime)),
		})
		if err != nil {
			return nil, err
		}
		mqlDeploymentRes := mqlDeployment.(*mqlGcpProjectVertexaiServiceEndpointDeployment)
		mqlDeploymentRes.cacheModelName = dm.Model
		mqlDeploymentRes.cacheProjectId = projectId
		deployments = append(deployments, mqlDeployment)
	}
	encryptionSpec, err := protoToDict(ep.EncryptionSpec)
	if err != nil {
		return nil, err
	}

	trafficSplit := make(map[string]any, len(ep.TrafficSplit))
	for k, v := range ep.TrafficSplit {
		trafficSplit[k] = int64(v)
	}

	mqlEndpoint, err := CreateResource(runtime, "gcp.project.vertexaiService.endpoint", map[string]*llx.RawData{
		"name":                        llx.StringData(ep.Name),
		"displayName":                 llx.StringData(ep.DisplayName),
		"description":                 llx.StringData(ep.Description),
		"deployedModels":              llx.ArrayData(deployedModels, types.Dict),
		"deployments":                 llx.ArrayData(deployments, types.Resource("gcp.project.vertexaiService.endpoint.deployment")),
		"encryptionSpec":              llx.DictData(encryptionSpec),
		"network":                     llx.StringData(ep.Network),
		"enablePrivateServiceConnect": llx.BoolData(ep.EnablePrivateServiceConnect),
		"trafficSplit":                llx.MapData(trafficSplit, types.Int),
		"labels":                      llx.MapData(convert.MapToInterfaceMap(ep.Labels), types.String),
		"etag":                        llx.StringData(ep.Etag),
		"createdAt":                   llx.TimeDataPtr(timestampAsTimePtr(ep.CreateTime)),
		"updatedAt":                   llx.TimeDataPtr(timestampAsTimePtr(ep.UpdateTime)),
	})
	if err != nil {
		return nil, err
	}
	res := mqlEndpoint.(*mqlGcpProjectVertexaiServiceEndpoint)
	res.cacheKmsKeyName = ep.GetEncryptionSpec().GetKmsKeyName()
	return res, nil
}

// resolveVertexaiEndpointRef fetches the Vertex AI endpoint identified by
// endpointName (a full resource name) and maps it into a fully-populated MQL
// resource. Like resolveVertexaiModelRef, it resolves directly via GetEndpoint
// because the gcp.project.vertexaiService.endpoint resource has no init
// function, so a bare NewResource(..., {"name": ...}) would yield a husk.
func resolveVertexaiEndpointRef(runtime *plugin.Runtime, field *plugin.TValue[*mqlGcpProjectVertexaiServiceEndpoint], endpointName string) (*mqlGcpProjectVertexaiServiceEndpoint, error) {
	region := vertexaiRegionFromName(endpointName)
	if endpointName == "" || region == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	conn := runtime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := aiplatform.NewEndpointClient(ctx,
		option.WithCredentials(creds),
		option.WithEndpoint(vertexaiEndpoint(region)),
	)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	ep, err := client.GetEndpoint(ctx, &aiplatformpb.GetEndpointRequest{Name: endpointName})
	if err != nil {
		if isVertexAIRegionSkippable(err) {
			field.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	return newMqlVertexaiEndpoint(runtime, vertexaiProjectFromName(endpointName), ep)
}

func (g *mqlGcpProjectVertexaiServiceEndpoint) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

type mqlGcpProjectVertexaiServiceEndpointDeploymentInternal struct {
	cacheModelName string
	cacheProjectId string
}

// vertexaiRegionFromName extracts the location from a Vertex AI resource name
// of the form projects/{project}/locations/{location}/....
func vertexaiRegionFromName(name string) string {
	parts := strings.Split(name, "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "locations" {
			return parts[i+1]
		}
	}
	return ""
}

// vertexaiProjectFromName extracts the project from a Vertex AI resource name
// of the form projects/{project}/locations/{location}/....
func vertexaiProjectFromName(name string) string {
	parts := strings.Split(name, "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "projects" {
			return parts[i+1]
		}
	}
	return ""
}

// resolveVertexaiModelRef fetches the Vertex AI model identified by modelName
// (a full resource name) and maps it into a fully-populated MQL resource. It is
// shared by every accessor that exposes a typed model reference. We resolve the
// model directly via GetModel rather than NewResource because the
// gcp.project.vertexaiService.model resource has no init function — a bare
// NewResource(..., {"name": ...}) would yield a husk with only the name set, so
// any field traversal beyond name would fail.
func resolveVertexaiModelRef(runtime *plugin.Runtime, field *plugin.TValue[*mqlGcpProjectVertexaiServiceModel], modelName string) (*mqlGcpProjectVertexaiServiceModel, error) {
	region := vertexaiRegionFromName(modelName)
	if modelName == "" || region == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	conn := runtime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := aiplatform.NewModelClient(ctx,
		option.WithCredentials(creds),
		option.WithEndpoint(vertexaiEndpoint(region)),
	)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	m, err := client.GetModel(ctx, &aiplatformpb.GetModelRequest{Name: modelName})
	if err != nil {
		if isVertexAIRegionSkippable(err) {
			field.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}
	return newMqlVertexaiModel(runtime, m)
}

func (d *mqlGcpProjectVertexaiServiceEndpointDeployment) model() (*mqlGcpProjectVertexaiServiceModel, error) {
	return resolveVertexaiModelRef(d.MqlRuntime, &d.Model, d.cacheModelName)
}

func (d *mqlGcpProjectVertexaiServiceEndpointDeployment) serviceAccount() (*mqlGcpProjectIamServiceServiceAccount, error) {
	if d.ServiceAccountEmail.Error != nil {
		return nil, d.ServiceAccountEmail.Error
	}
	email := d.ServiceAccountEmail.Data
	if email == "" || d.cacheProjectId == "" {
		d.ServiceAccount.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	res, err := NewResource(d.MqlRuntime, "gcp.project.iamService.serviceAccount", map[string]*llx.RawData{
		"projectId": llx.StringData(d.cacheProjectId),
		"email":     llx.StringData(email),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectIamServiceServiceAccount), nil
}

func (g *mqlGcpProjectVertexaiService) pipelineJobs() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewPipelineClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListPipelineJobs(ctx, &aiplatformpb.ListPipelineJobsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			job, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			pipelineSpec, err := protoToDict(job.PipelineSpec)
			if err != nil {
				return nil, false, err
			}
			runtimeConfig, err := protoToDict(job.RuntimeConfig)
			if err != nil {
				return nil, false, err
			}
			encryptionSpec, err := protoToDict(job.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}
			templateMetadata, err := protoToDict(job.TemplateMetadata)
			if err != nil {
				return nil, false, err
			}

			mqlJob, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.pipelineJob", map[string]*llx.RawData{
				"name":             llx.StringData(job.Name),
				"displayName":      llx.StringData(job.DisplayName),
				"state":            llx.StringData(job.State.String()),
				"pipelineSpec":     llx.DictData(pipelineSpec),
				"runtimeConfig":    llx.DictData(runtimeConfig),
				"serviceAccount":   llx.StringData(job.ServiceAccount),
				"network":          llx.StringData(job.Network),
				"encryptionSpec":   llx.DictData(encryptionSpec),
				"templateUri":      llx.StringData(job.TemplateUri),
				"templateMetadata": llx.DictData(templateMetadata),
				"labels":           llx.MapData(convert.MapToInterfaceMap(job.Labels), types.String),
				"createdAt":        llx.TimeDataPtr(timestampAsTimePtr(job.CreateTime)),
				"updatedAt":        llx.TimeDataPtr(timestampAsTimePtr(job.UpdateTime)),
				"startTime":        llx.TimeDataPtr(timestampAsTimePtr(job.StartTime)),
				"endTime":          llx.TimeDataPtr(timestampAsTimePtr(job.EndTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlJob.(*mqlGcpProjectVertexaiServicePipelineJob).cacheKmsKeyName = job.GetEncryptionSpec().GetKmsKeyName()
			items = append(items, mqlJob)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServicePipelineJob) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) datasets() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewDatasetClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListDatasets(ctx, &aiplatformpb.ListDatasetsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			ds, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			// Dataset.Metadata is a *structpb.Value which can be any JSON type
			// (struct, list, string, etc.), so use AsInterface() instead of protoToDict.
			var metadata any
			if ds.Metadata != nil {
				metadata = ds.Metadata.AsInterface()
			}
			encryptionSpec, err := protoToDict(ds.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			mqlDs, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.dataset", map[string]*llx.RawData{
				"name":              llx.StringData(ds.Name),
				"displayName":       llx.StringData(ds.DisplayName),
				"description":       llx.StringData(ds.Description),
				"metadataSchemaUri": llx.StringData(ds.MetadataSchemaUri),
				"metadata":          llx.DictData(metadata),
				"encryptionSpec":    llx.DictData(encryptionSpec),
				"labels":            llx.MapData(convert.MapToInterfaceMap(ds.Labels), types.String),
				"etag":              llx.StringData(ds.Etag),
				"createdAt":         llx.TimeDataPtr(timestampAsTimePtr(ds.CreateTime)),
				"updatedAt":         llx.TimeDataPtr(timestampAsTimePtr(ds.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlDs.(*mqlGcpProjectVertexaiServiceDataset).cacheKmsKeyName = ds.GetEncryptionSpec().GetKmsKeyName()
			items = append(items, mqlDs)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServiceDataset) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) featureOnlineStores() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewFeatureOnlineStoreAdminClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListFeatureOnlineStores(ctx, &aiplatformpb.ListFeatureOnlineStoresRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			store, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			bigtable, err := protoToDict(store.GetBigtable())
			if err != nil {
				return nil, false, err
			}
			optimized, err := protoToDict(store.GetOptimized())
			if err != nil {
				return nil, false, err
			}
			dedicatedServingEndpoint, err := protoToDict(store.DedicatedServingEndpoint)
			if err != nil {
				return nil, false, err
			}
			encryptionSpec, err := protoToDict(store.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			mqlStore, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.featureOnlineStore", map[string]*llx.RawData{
				"name":                     llx.StringData(store.Name),
				"state":                    llx.StringData(store.State.String()),
				"bigtable":                 llx.DictData(bigtable),
				"optimized":                llx.DictData(optimized),
				"dedicatedServingEndpoint": llx.DictData(dedicatedServingEndpoint),
				"encryptionSpec":           llx.DictData(encryptionSpec),
				"labels":                   llx.MapData(convert.MapToInterfaceMap(store.Labels), types.String),
				"etag":                     llx.StringData(store.Etag),
				"satisfiesPzs":             llx.BoolData(store.SatisfiesPzs),
				"satisfiesPzi":             llx.BoolData(store.SatisfiesPzi),
				"createdAt":                llx.TimeDataPtr(timestampAsTimePtr(store.CreateTime)),
				"updatedAt":                llx.TimeDataPtr(timestampAsTimePtr(store.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlStore.(*mqlGcpProjectVertexaiServiceFeatureOnlineStore).cacheKmsKeyName = store.GetEncryptionSpec().GetKmsKeyName()
			items = append(items, mqlStore)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServiceFeatureOnlineStore) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

// ---------------------------------------------------------------
// Tensorboards
// ---------------------------------------------------------------

func (g *mqlGcpProjectVertexaiServiceTensorboard) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) tensorboards() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewTensorboardClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListTensorboards(ctx, &aiplatformpb.ListTensorboardsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			tb, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			encryptionSpec, err := protoToDict(tb.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			mqlTB, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.tensorboard", map[string]*llx.RawData{
				"name":           llx.StringData(tb.Name),
				"displayName":    llx.StringData(tb.DisplayName),
				"description":    llx.StringData(tb.Description),
				"isDefault":      llx.BoolData(tb.IsDefault),
				"labels":         llx.MapData(convert.MapToInterfaceMap(tb.Labels), types.String),
				"encryptionSpec": llx.DictData(encryptionSpec),
				"created":        llx.TimeDataPtr(timestampAsTimePtr(tb.CreateTime)),
				"updated":        llx.TimeDataPtr(timestampAsTimePtr(tb.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlTB.(*mqlGcpProjectVertexaiServiceTensorboard).cacheKmsKeyName = tb.GetEncryptionSpec().GetKmsKeyName()
			items = append(items, mqlTB)
		}
		return items, false, nil
	})
}

// ---------------------------------------------------------------
// Custom Jobs
// ---------------------------------------------------------------

func (g *mqlGcpProjectVertexaiServiceCustomJob) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func initGcpProjectVertexaiServiceCustomJob(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if len(args) == 0 {
		args = make(map[string]*llx.RawData)
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["projectId"] = llx.StringData(ids.project)
			args["location"] = llx.StringData(ids.region)
		} else {
			return nil, nil, errors.New("no asset identifier found")
		}
	}

	nameRaw := args["name"]
	if nameRaw == nil {
		return args, nil, nil
	}
	name := nameRaw.Value.(string)

	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}

	// Accept either the full resource path or a short name + location from
	// the asset-identifier-driven discovery path.
	var fullName, region string
	if strings.HasPrefix(name, "projects/") {
		fullName = name
		region = parseLocationFromPath(name)
	} else {
		locRaw := args["location"]
		projRaw := args["projectId"]
		if locRaw == nil || projRaw == nil {
			return nil, nil, errors.New("vertexai custom job init: projectId and location required when name is not a full resource path")
		}
		region = locRaw.Value.(string)
		fullName = fmt.Sprintf("projects/%s/locations/%s/customJobs/%s", projRaw.Value.(string), region, name)
	}

	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, nil, err
	}
	ctx := context.Background()
	client, err := aiplatform.NewJobClient(ctx,
		option.WithCredentials(creds),
		option.WithEndpoint(vertexaiEndpoint(region)),
	)
	if err != nil {
		return nil, nil, err
	}
	defer client.Close()

	job, err := client.GetCustomJob(ctx, &aiplatformpb.GetCustomJobRequest{Name: fullName})
	if err != nil {
		return nil, nil, err
	}

	res, err := mqlVertexAICustomJobFromProto(runtime, job)
	if err != nil {
		return nil, nil, err
	}
	delete(args, "location")
	return args, res, nil
}

func mqlVertexAICustomJobFromProto(runtime *plugin.Runtime, job *aiplatformpb.CustomJob) (*mqlGcpProjectVertexaiServiceCustomJob, error) {
	jobSpec, err := protoToDict(job.JobSpec)
	if err != nil {
		return nil, err
	}
	encryptionSpec, err := protoToDict(job.EncryptionSpec)
	if err != nil {
		return nil, err
	}
	errorDict, err := protoToDict(job.Error)
	if err != nil {
		return nil, err
	}

	res, err := CreateResource(runtime, "gcp.project.vertexaiService.customJob", map[string]*llx.RawData{
		"name":           llx.StringData(job.Name),
		"displayName":    llx.StringData(job.DisplayName),
		"state":          llx.StringData(job.State.String()),
		"jobSpec":        llx.DictData(jobSpec),
		"labels":         llx.MapData(convert.MapToInterfaceMap(job.Labels), types.String),
		"encryptionSpec": llx.DictData(encryptionSpec),
		"error":          llx.DictData(errorDict),
		"created":        llx.TimeDataPtr(timestampAsTimePtr(job.CreateTime)),
		"updated":        llx.TimeDataPtr(timestampAsTimePtr(job.UpdateTime)),
		"startTime":      llx.TimeDataPtr(timestampAsTimePtr(job.StartTime)),
		"endTime":        llx.TimeDataPtr(timestampAsTimePtr(job.EndTime)),
	})
	if err != nil {
		return nil, err
	}
	res.(*mqlGcpProjectVertexaiServiceCustomJob).cacheKmsKeyName = job.GetEncryptionSpec().GetKmsKeyName()
	return res.(*mqlGcpProjectVertexaiServiceCustomJob), nil
}

func (g *mqlGcpProjectVertexaiService) customJobs() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewJobClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListCustomJobs(ctx, &aiplatformpb.ListCustomJobsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			job, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			mqlJob, err := mqlVertexAICustomJobFromProto(g.MqlRuntime, job)
			if err != nil {
				return nil, false, err
			}
			items = append(items, mqlJob)
		}
		return items, false, nil
	})
}

// ---------------------------------------------------------------
// Indexes
// ---------------------------------------------------------------

func (g *mqlGcpProjectVertexaiServiceIndex) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) indexes() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewIndexClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListIndexes(ctx, &aiplatformpb.ListIndexesRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			idx, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			var metadata any
			if idx.Metadata != nil {
				metadata = idx.Metadata.AsInterface()
			}
			encryptionSpec, err := protoToDict(idx.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}
			indexStats, err := protoToDict(idx.IndexStats)
			if err != nil {
				return nil, false, err
			}

			deployedIndexes := make([]any, 0, len(idx.DeployedIndexes))
			for _, di := range idx.DeployedIndexes {
				d, err := protoToDict(di)
				if err != nil {
					return nil, false, err
				}
				deployedIndexes = append(deployedIndexes, d)
			}

			mqlIdx, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.index", map[string]*llx.RawData{
				"name":              llx.StringData(idx.Name),
				"displayName":       llx.StringData(idx.DisplayName),
				"description":       llx.StringData(idx.Description),
				"metadataSchemaUri": llx.StringData(idx.MetadataSchemaUri),
				"metadata":          llx.DictData(metadata),
				"deployedIndexes":   llx.ArrayData(deployedIndexes, types.Dict),
				"labels":            llx.MapData(convert.MapToInterfaceMap(idx.Labels), types.String),
				"encryptionSpec":    llx.DictData(encryptionSpec),
				"indexUpdateMethod": llx.StringData(idx.IndexUpdateMethod.String()),
				"indexStats":        llx.DictData(indexStats),
				"created":           llx.TimeDataPtr(timestampAsTimePtr(idx.CreateTime)),
				"updated":           llx.TimeDataPtr(timestampAsTimePtr(idx.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlIdx.(*mqlGcpProjectVertexaiServiceIndex).cacheKmsKeyName = idx.GetEncryptionSpec().GetKmsKeyName()
			items = append(items, mqlIdx)
		}
		return items, false, nil
	})
}

// ---------------------------------------------------------------
// Index Endpoints
// ---------------------------------------------------------------

func (g *mqlGcpProjectVertexaiServiceIndexEndpoint) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) indexEndpoints() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewIndexEndpointClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListIndexEndpoints(ctx, &aiplatformpb.ListIndexEndpointsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			ep, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			encryptionSpec, err := protoToDict(ep.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			deployedIndexes := make([]any, 0, len(ep.DeployedIndexes))
			for _, di := range ep.DeployedIndexes {
				d, err := protoToDict(di)
				if err != nil {
					return nil, false, err
				}
				deployedIndexes = append(deployedIndexes, d)
			}

			mqlEP, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.indexEndpoint", map[string]*llx.RawData{
				"name":                     llx.StringData(ep.Name),
				"displayName":              llx.StringData(ep.DisplayName),
				"description":              llx.StringData(ep.Description),
				"deployedIndexes":          llx.ArrayData(deployedIndexes, types.Dict),
				"network":                  llx.StringData(ep.Network),
				"publicEndpointEnabled":    llx.BoolData(ep.PublicEndpointEnabled),
				"publicEndpointDomainName": llx.StringData(ep.PublicEndpointDomainName),
				"labels":                   llx.MapData(convert.MapToInterfaceMap(ep.Labels), types.String),
				"encryptionSpec":           llx.DictData(encryptionSpec),
				"created":                  llx.TimeDataPtr(timestampAsTimePtr(ep.CreateTime)),
				"updated":                  llx.TimeDataPtr(timestampAsTimePtr(ep.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlEP.(*mqlGcpProjectVertexaiServiceIndexEndpoint).cacheKmsKeyName = ep.GetEncryptionSpec().GetKmsKeyName()
			items = append(items, mqlEP)
		}
		return items, false, nil
	})
}

// ---------------------------------------------------------------
// Metadata Stores
// ---------------------------------------------------------------

func (g *mqlGcpProjectVertexaiServiceMetadataStore) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) metadataStores() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewMetadataClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListMetadataStores(ctx, &aiplatformpb.ListMetadataStoresRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			store, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			state, err := protoToDict(store.State)
			if err != nil {
				return nil, false, err
			}
			encryptionSpec, err := protoToDict(store.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}
			dataplexConfig, err := protoToDict(store.DataplexConfig)
			if err != nil {
				return nil, false, err
			}

			mqlStore, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.metadataStore", map[string]*llx.RawData{
				"name":           llx.StringData(store.Name),
				"description":    llx.StringData(store.Description),
				"state":          llx.DictData(state),
				"encryptionSpec": llx.DictData(encryptionSpec),
				"dataplexConfig": llx.DictData(dataplexConfig),
				"created":        llx.TimeDataPtr(timestampAsTimePtr(store.CreateTime)),
				"updated":        llx.TimeDataPtr(timestampAsTimePtr(store.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlStore.(*mqlGcpProjectVertexaiServiceMetadataStore).cacheKmsKeyName = store.GetEncryptionSpec().GetKmsKeyName()
			items = append(items, mqlStore)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiService) notebookRuntimes() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewNotebookClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListNotebookRuntimes(ctx, &aiplatformpb.ListNotebookRuntimesRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			rt, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			machineSpec, err := protoToDict(rt.MachineSpec)
			if err != nil {
				return nil, false, err
			}
			networkSpec, err := protoToDict(rt.NetworkSpec)
			if err != nil {
				return nil, false, err
			}
			idleShutdownConfig, err := protoToDict(rt.IdleShutdownConfig)
			if err != nil {
				return nil, false, err
			}
			eucConfig, err := protoToDict(rt.EucConfig)
			if err != nil {
				return nil, false, err
			}
			shieldedVmConfig, err := protoToDict(rt.ShieldedVmConfig)
			if err != nil {
				return nil, false, err
			}
			softwareConfig, err := protoToDict(rt.SoftwareConfig)
			if err != nil {
				return nil, false, err
			}
			encryptionSpec, err := protoToDict(rt.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			mqlRt, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.notebookRuntime", map[string]*llx.RawData{
				"name":                llx.StringData(rt.Name),
				"displayName":         llx.StringData(rt.DisplayName),
				"description":         llx.StringData(rt.Description),
				"runtimeUser":         llx.StringData(rt.RuntimeUser),
				"proxyUri":            llx.StringData(rt.ProxyUri),
				"healthState":         llx.StringData(rt.HealthState.String()),
				"runtimeState":        llx.StringData(rt.RuntimeState.String()),
				"notebookRuntimeType": llx.StringData(rt.NotebookRuntimeType.String()),
				"isUpgradable":        llx.BoolData(rt.IsUpgradable),
				"version":             llx.StringData(rt.Version),
				"networkTags":         llx.ArrayData(convert.SliceAnyToInterface(rt.NetworkTags), types.String),
				"machineSpec":         llx.DictData(machineSpec),
				"networkSpec":         llx.DictData(networkSpec),
				"idleShutdownConfig":  llx.DictData(idleShutdownConfig),
				"eucConfig":           llx.DictData(eucConfig),
				"shieldedVmConfig":    llx.DictData(shieldedVmConfig),
				"softwareConfig":      llx.DictData(softwareConfig),
				"labels":              llx.MapData(convert.MapToInterfaceMap(rt.Labels), types.String),
				"encryptionSpec":      llx.DictData(encryptionSpec),
				"createdAt":           llx.TimeDataPtr(timestampAsTimePtr(rt.CreateTime)),
				"updatedAt":           llx.TimeDataPtr(timestampAsTimePtr(rt.UpdateTime)),
				"expirationTime":      llx.TimeDataPtr(timestampAsTimePtr(rt.ExpirationTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlRtTyped := mqlRt.(*mqlGcpProjectVertexaiServiceNotebookRuntime)
			mqlRtTyped.cacheKmsKeyName = rt.GetEncryptionSpec().GetKmsKeyName()
			mqlRtTyped.cacheServiceAccountEmail = rt.ServiceAccount
			mqlRtTyped.cacheProjectId = projectId
			items = append(items, mqlRt)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServiceNotebookRuntime) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) notebookRuntimeTemplates() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewNotebookClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListNotebookRuntimeTemplates(ctx, &aiplatformpb.ListNotebookRuntimeTemplatesRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			tmpl, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			machineSpec, err := protoToDict(tmpl.MachineSpec)
			if err != nil {
				return nil, false, err
			}
			networkSpec, err := protoToDict(tmpl.NetworkSpec)
			if err != nil {
				return nil, false, err
			}
			idleShutdownConfig, err := protoToDict(tmpl.IdleShutdownConfig)
			if err != nil {
				return nil, false, err
			}
			eucConfig, err := protoToDict(tmpl.EucConfig)
			if err != nil {
				return nil, false, err
			}
			shieldedVmConfig, err := protoToDict(tmpl.ShieldedVmConfig)
			if err != nil {
				return nil, false, err
			}
			softwareConfig, err := protoToDict(tmpl.SoftwareConfig)
			if err != nil {
				return nil, false, err
			}
			encryptionSpec, err := protoToDict(tmpl.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			mqlTmpl, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.notebookRuntimeTemplate", map[string]*llx.RawData{
				"name":                llx.StringData(tmpl.Name),
				"displayName":         llx.StringData(tmpl.DisplayName),
				"description":         llx.StringData(tmpl.Description),
				"isDefault":           llx.BoolData(tmpl.IsDefault),
				"notebookRuntimeType": llx.StringData(tmpl.NotebookRuntimeType.String()),
				"networkTags":         llx.ArrayData(convert.SliceAnyToInterface(tmpl.NetworkTags), types.String),
				"machineSpec":         llx.DictData(machineSpec),
				"networkSpec":         llx.DictData(networkSpec),
				"idleShutdownConfig":  llx.DictData(idleShutdownConfig),
				"eucConfig":           llx.DictData(eucConfig),
				"shieldedVmConfig":    llx.DictData(shieldedVmConfig),
				"softwareConfig":      llx.DictData(softwareConfig),
				"etag":                llx.StringData(tmpl.Etag),
				"labels":              llx.MapData(convert.MapToInterfaceMap(tmpl.Labels), types.String),
				"encryptionSpec":      llx.DictData(encryptionSpec),
				"createdAt":           llx.TimeDataPtr(timestampAsTimePtr(tmpl.CreateTime)),
				"updatedAt":           llx.TimeDataPtr(timestampAsTimePtr(tmpl.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlTmpl.(*mqlGcpProjectVertexaiServiceNotebookRuntimeTemplate).cacheKmsKeyName = tmpl.GetEncryptionSpec().GetKmsKeyName()
			items = append(items, mqlTmpl)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServiceNotebookRuntimeTemplate) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) notebookExecutionJobs() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewNotebookClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListNotebookExecutionJobs(ctx, &aiplatformpb.ListNotebookExecutionJobsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			job, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			encryptionSpec, err := protoToDict(job.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			var timeoutSeconds int64
			if job.ExecutionTimeout != nil {
				timeoutSeconds = int64(job.ExecutionTimeout.AsDuration().Seconds())
			}

			mqlJob, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.notebookExecutionJob", map[string]*llx.RawData{
				"name":                    llx.StringData(job.Name),
				"displayName":             llx.StringData(job.DisplayName),
				"jobState":                llx.StringData(job.JobState.String()),
				"kernelName":              llx.StringData(job.KernelName),
				"scheduleResourceName":    llx.StringData(job.ScheduleResourceName),
				"executionTimeoutSeconds": llx.IntData(timeoutSeconds),
				"labels":                  llx.MapData(convert.MapToInterfaceMap(job.Labels), types.String),
				"encryptionSpec":          llx.DictData(encryptionSpec),
				"createdAt":               llx.TimeDataPtr(timestampAsTimePtr(job.CreateTime)),
				"updatedAt":               llx.TimeDataPtr(timestampAsTimePtr(job.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlJob.(*mqlGcpProjectVertexaiServiceNotebookExecutionJob).cacheKmsKeyName = job.GetEncryptionSpec().GetKmsKeyName()
			items = append(items, mqlJob)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServiceNotebookExecutionJob) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) reasoningEngines() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewReasoningEngineClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListReasoningEngines(ctx, &aiplatformpb.ListReasoningEnginesRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			engine, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			spec, err := protoToDict(engine.Spec)
			if err != nil {
				return nil, false, err
			}
			encryptionSpec, err := protoToDict(engine.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			mqlEngine, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.reasoningEngine", map[string]*llx.RawData{
				"name":                llx.StringData(engine.Name),
				"displayName":         llx.StringData(engine.DisplayName),
				"description":         llx.StringData(engine.Description),
				"serviceAccountEmail": llx.StringData(engine.GetSpec().GetServiceAccount()),
				"agentFramework":      llx.StringData(engine.GetSpec().GetAgentFramework()),
				"spec":                llx.DictData(spec),
				"encryptionSpec":      llx.DictData(encryptionSpec),
				"labels":              llx.MapData(convert.MapToInterfaceMap(engine.Labels), types.String),
				"etag":                llx.StringData(engine.Etag),
				"createdAt":           llx.TimeDataPtr(timestampAsTimePtr(engine.CreateTime)),
				"updatedAt":           llx.TimeDataPtr(timestampAsTimePtr(engine.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlEngineRes := mqlEngine.(*mqlGcpProjectVertexaiServiceReasoningEngine)
			mqlEngineRes.cacheKmsKeyName = engine.GetEncryptionSpec().GetKmsKeyName()
			mqlEngineRes.cacheProjectId = projectId
			items = append(items, mqlEngine)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServiceReasoningEngine) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (a *mqlGcpProjectVertexaiServiceReasoningEngine) serviceAccount() (*mqlGcpProjectIamServiceServiceAccount, error) {
	if a.ServiceAccountEmail.Error != nil {
		return nil, a.ServiceAccountEmail.Error
	}
	email := a.ServiceAccountEmail.Data
	if email == "" || a.cacheProjectId == "" {
		a.ServiceAccount.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	res, err := NewResource(a.MqlRuntime, "gcp.project.iamService.serviceAccount", map[string]*llx.RawData{
		"projectId": llx.StringData(a.cacheProjectId),
		"email":     llx.StringData(email),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectIamServiceServiceAccount), nil
}

func (g *mqlGcpProjectVertexaiService) ragCorpora() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewVertexRagDataClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListRagCorpora(ctx, &aiplatformpb.ListRagCorporaRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			corpus, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			corpusStatus, err := protoToDict(corpus.CorpusStatus)
			if err != nil {
				return nil, false, err
			}
			encryptionSpec, err := protoToDict(corpus.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			mqlCorpus, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.ragCorpus", map[string]*llx.RawData{
				"name":           llx.StringData(corpus.Name),
				"displayName":    llx.StringData(corpus.DisplayName),
				"description":    llx.StringData(corpus.Description),
				"corpusStatus":   llx.DictData(corpusStatus),
				"encryptionSpec": llx.DictData(encryptionSpec),
				"createdAt":      llx.TimeDataPtr(timestampAsTimePtr(corpus.CreateTime)),
				"updatedAt":      llx.TimeDataPtr(timestampAsTimePtr(corpus.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlCorpus.(*mqlGcpProjectVertexaiServiceRagCorpus).cacheKmsKeyName = corpus.GetEncryptionSpec().GetKmsKeyName()
			items = append(items, mqlCorpus)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServiceRagCorpus) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) featureGroups() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewFeatureRegistryClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListFeatureGroups(ctx, &aiplatformpb.ListFeatureGroupsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			fg, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			bigQuery, err := protoToDict(fg.GetBigQuery())
			if err != nil {
				return nil, false, err
			}

			mqlFg, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.featureGroup", map[string]*llx.RawData{
				"name":        llx.StringData(fg.Name),
				"description": llx.StringData(fg.Description),
				"bigQuery":    llx.DictData(bigQuery),
				"etag":        llx.StringData(fg.Etag),
				"labels":      llx.MapData(convert.MapToInterfaceMap(fg.Labels), types.String),
				"createdAt":   llx.TimeDataPtr(timestampAsTimePtr(fg.CreateTime)),
				"updatedAt":   llx.TimeDataPtr(timestampAsTimePtr(fg.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			items = append(items, mqlFg)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServiceFeatureGroup) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) persistentResources() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewPersistentResourceClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListPersistentResources(ctx, &aiplatformpb.ListPersistentResourcesRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			pr, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			resourcePools := make([]any, 0, len(pr.ResourcePools))
			for _, rp := range pr.ResourcePools {
				d, err := protoToDict(rp)
				if err != nil {
					return nil, false, err
				}
				resourcePools = append(resourcePools, d)
			}
			resourceRuntimeSpec, err := protoToDict(pr.ResourceRuntimeSpec)
			if err != nil {
				return nil, false, err
			}
			encryptionSpec, err := protoToDict(pr.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			mqlPr, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.persistentResource", map[string]*llx.RawData{
				"name":                llx.StringData(pr.Name),
				"displayName":         llx.StringData(pr.DisplayName),
				"state":               llx.StringData(pr.State.String()),
				"reservedIpRanges":    llx.ArrayData(convert.SliceAnyToInterface(pr.ReservedIpRanges), types.String),
				"resourcePools":       llx.ArrayData(resourcePools, types.Dict),
				"resourceRuntimeSpec": llx.DictData(resourceRuntimeSpec),
				"labels":              llx.MapData(convert.MapToInterfaceMap(pr.Labels), types.String),
				"encryptionSpec":      llx.DictData(encryptionSpec),
				"createdAt":           llx.TimeDataPtr(timestampAsTimePtr(pr.CreateTime)),
				"startedAt":           llx.TimeDataPtr(timestampAsTimePtr(pr.StartTime)),
				"updatedAt":           llx.TimeDataPtr(timestampAsTimePtr(pr.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			prMql := mqlPr.(*mqlGcpProjectVertexaiServicePersistentResource)
			prMql.cacheKmsKeyName = pr.GetEncryptionSpec().GetKmsKeyName()
			prMql.cacheNetworkUrl = pr.Network
			items = append(items, mqlPr)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServicePersistentResource) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) schedules() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewScheduleClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListSchedules(ctx, &aiplatformpb.ListSchedulesRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			sched, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			mqlSched, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.schedule", map[string]*llx.RawData{
				"name":                  llx.StringData(sched.Name),
				"displayName":           llx.StringData(sched.DisplayName),
				"state":                 llx.StringData(sched.State.String()),
				"cron":                  llx.StringData(sched.GetCron()),
				"maxRunCount":           llx.IntData(sched.MaxRunCount),
				"startedRunCount":       llx.IntData(sched.StartedRunCount),
				"maxConcurrentRunCount": llx.IntData(sched.MaxConcurrentRunCount),
				"allowQueueing":         llx.BoolData(sched.AllowQueueing),
				"catchUp":               llx.BoolData(sched.CatchUp),
				"startedAt":             llx.TimeDataPtr(timestampAsTimePtr(sched.StartTime)),
				"endedAt":               llx.TimeDataPtr(timestampAsTimePtr(sched.EndTime)),
				"createdAt":             llx.TimeDataPtr(timestampAsTimePtr(sched.CreateTime)),
				"updatedAt":             llx.TimeDataPtr(timestampAsTimePtr(sched.UpdateTime)),
				"nextRunTime":           llx.TimeDataPtr(timestampAsTimePtr(sched.NextRunTime)),
			})
			if err != nil {
				return nil, false, err
			}
			items = append(items, mqlSched)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServiceSchedule) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) deploymentResourcePools() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewDeploymentResourcePoolClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListDeploymentResourcePools(ctx, &aiplatformpb.ListDeploymentResourcePoolsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			pool, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			dedicatedResources, err := protoToDict(pool.DedicatedResources)
			if err != nil {
				return nil, false, err
			}
			encryptionSpec, err := protoToDict(pool.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			mqlPool, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.deploymentResourcePool", map[string]*llx.RawData{
				"name":                    llx.StringData(pool.Name),
				"disableContainerLogging": llx.BoolData(pool.DisableContainerLogging),
				"dedicatedResources":      llx.DictData(dedicatedResources),
				"encryptionSpec":          llx.DictData(encryptionSpec),
				"createdAt":               llx.TimeDataPtr(timestampAsTimePtr(pool.CreateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlPoolRes := mqlPool.(*mqlGcpProjectVertexaiServiceDeploymentResourcePool)
			mqlPoolRes.cacheKmsKeyName = pool.GetEncryptionSpec().GetKmsKeyName()
			mqlPoolRes.cacheProjectId = projectId
			mqlPoolRes.cacheServiceAccountEmail = pool.ServiceAccount
			items = append(items, mqlPool)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServiceDeploymentResourcePool) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) cachedContents() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewGenAiCacheClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListCachedContents(ctx, &aiplatformpb.ListCachedContentsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			cc, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			usageMetadata, err := protoToDict(cc.UsageMetadata)
			if err != nil {
				return nil, false, err
			}
			encryptionSpec, err := protoToDict(cc.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			mqlCc, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.cachedContent", map[string]*llx.RawData{
				"name":           llx.StringData(cc.Name),
				"displayName":    llx.StringData(cc.DisplayName),
				"model":          llx.StringData(cc.Model),
				"usageMetadata":  llx.DictData(usageMetadata),
				"expireTime":     llx.TimeDataPtr(timestampAsTimePtr(cc.GetExpireTime())),
				"encryptionSpec": llx.DictData(encryptionSpec),
				"createdAt":      llx.TimeDataPtr(timestampAsTimePtr(cc.CreateTime)),
				"updatedAt":      llx.TimeDataPtr(timestampAsTimePtr(cc.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlCc.(*mqlGcpProjectVertexaiServiceCachedContent).cacheKmsKeyName = cc.GetEncryptionSpec().GetKmsKeyName()
			items = append(items, mqlCc)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServiceCachedContent) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) batchPredictionJobs() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewJobClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListBatchPredictionJobs(ctx, &aiplatformpb.ListBatchPredictionJobsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			job, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			inputConfig, err := protoToDict(job.InputConfig)
			if err != nil {
				return nil, false, err
			}
			outputConfig, err := protoToDict(job.OutputConfig)
			if err != nil {
				return nil, false, err
			}
			encryptionSpec, err := protoToDict(job.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			mqlJob, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.batchPredictionJob", map[string]*llx.RawData{
				"name":                    llx.StringData(job.Name),
				"displayName":             llx.StringData(job.DisplayName),
				"modelVersionId":          llx.StringData(job.ModelVersionId),
				"inputConfig":             llx.DictData(inputConfig),
				"outputConfig":            llx.DictData(outputConfig),
				"state":                   llx.StringData(job.State.String()),
				"generateExplanation":     llx.BoolData(job.GenerateExplanation),
				"disableContainerLogging": llx.BoolData(job.DisableContainerLogging),
				"labels":                  llx.MapData(convert.MapToInterfaceMap(job.Labels), types.String),
				"encryptionSpec":          llx.DictData(encryptionSpec),
				"createdAt":               llx.TimeDataPtr(timestampAsTimePtr(job.CreateTime)),
				"startedAt":               llx.TimeDataPtr(timestampAsTimePtr(job.StartTime)),
				"endedAt":                 llx.TimeDataPtr(timestampAsTimePtr(job.EndTime)),
				"updatedAt":               llx.TimeDataPtr(timestampAsTimePtr(job.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlBatchJob := mqlJob.(*mqlGcpProjectVertexaiServiceBatchPredictionJob)
			mqlBatchJob.cacheKmsKeyName = job.GetEncryptionSpec().GetKmsKeyName()
			mqlBatchJob.cacheModelName = job.Model
			mqlBatchJob.cacheServiceAccountEmail = job.ServiceAccount
			mqlBatchJob.cacheProjectId = projectId
			items = append(items, mqlJob)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServiceBatchPredictionJob) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) tuningJobs() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewGenAiTuningClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListTuningJobs(ctx, &aiplatformpb.ListTuningJobsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			job, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			tunedModel, err := protoToDict(job.TunedModel)
			if err != nil {
				return nil, false, err
			}
			tuningDataStats, err := protoToDict(job.TuningDataStats)
			if err != nil {
				return nil, false, err
			}
			encryptionSpec, err := protoToDict(job.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			mqlJob, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.tuningJob", map[string]*llx.RawData{
				"name":                  llx.StringData(job.Name),
				"tunedModelDisplayName": llx.StringData(job.TunedModelDisplayName),
				"description":           llx.StringData(job.Description),
				"baseModel":             llx.StringData(job.GetBaseModel()),
				"state":                 llx.StringData(job.State.String()),
				"experiment":            llx.StringData(job.Experiment),
				"tunedModel":            llx.DictData(tunedModel),
				"tuningDataStats":       llx.DictData(tuningDataStats),
				"labels":                llx.MapData(convert.MapToInterfaceMap(job.Labels), types.String),
				"encryptionSpec":        llx.DictData(encryptionSpec),
				"createdAt":             llx.TimeDataPtr(timestampAsTimePtr(job.CreateTime)),
				"startedAt":             llx.TimeDataPtr(timestampAsTimePtr(job.StartTime)),
				"endedAt":               llx.TimeDataPtr(timestampAsTimePtr(job.EndTime)),
				"updatedAt":             llx.TimeDataPtr(timestampAsTimePtr(job.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlTuningJob := mqlJob.(*mqlGcpProjectVertexaiServiceTuningJob)
			mqlTuningJob.cacheKmsKeyName = job.GetEncryptionSpec().GetKmsKeyName()
			mqlTuningJob.cacheServiceAccountEmail = job.ServiceAccount
			mqlTuningJob.cacheProjectId = projectId
			items = append(items, mqlJob)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServiceTuningJob) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) trainingPipelines() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewPipelineClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListTrainingPipelines(ctx, &aiplatformpb.ListTrainingPipelinesRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			pipeline, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			inputDataConfig, err := protoToDict(pipeline.InputDataConfig)
			if err != nil {
				return nil, false, err
			}
			encryptionSpec, err := protoToDict(pipeline.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			mqlPipeline, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.trainingPipeline", map[string]*llx.RawData{
				"name":                   llx.StringData(pipeline.Name),
				"displayName":            llx.StringData(pipeline.DisplayName),
				"trainingTaskDefinition": llx.StringData(pipeline.TrainingTaskDefinition),
				"inputDataConfig":        llx.DictData(inputDataConfig),
				"state":                  llx.StringData(pipeline.State.String()),
				"labels":                 llx.MapData(convert.MapToInterfaceMap(pipeline.Labels), types.String),
				"encryptionSpec":         llx.DictData(encryptionSpec),
				"createdAt":              llx.TimeDataPtr(timestampAsTimePtr(pipeline.CreateTime)),
				"startedAt":              llx.TimeDataPtr(timestampAsTimePtr(pipeline.StartTime)),
				"endedAt":                llx.TimeDataPtr(timestampAsTimePtr(pipeline.EndTime)),
				"updatedAt":              llx.TimeDataPtr(timestampAsTimePtr(pipeline.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			pipelineRes := mqlPipeline.(*mqlGcpProjectVertexaiServiceTrainingPipeline)
			pipelineRes.cacheKmsKeyName = pipeline.GetEncryptionSpec().GetKmsKeyName()
			pipelineRes.cacheParentModel = pipeline.ParentModel
			if pipeline.ModelId != "" {
				pipelineRes.cacheModelName = fmt.Sprintf("projects/%s/locations/%s/models/%s", projectId, region, pipeline.ModelId)
			}
			items = append(items, mqlPipeline)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServiceTrainingPipeline) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) hyperparameterTuningJobs() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewJobClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListHyperparameterTuningJobs(ctx, &aiplatformpb.ListHyperparameterTuningJobsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			job, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			studySpec, err := protoToDict(job.StudySpec)
			if err != nil {
				return nil, false, err
			}
			trialJobSpec, err := protoToDict(job.TrialJobSpec)
			if err != nil {
				return nil, false, err
			}
			encryptionSpec, err := protoToDict(job.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			mqlJob, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.hyperparameterTuningJob", map[string]*llx.RawData{
				"name":                llx.StringData(job.Name),
				"displayName":         llx.StringData(job.DisplayName),
				"state":               llx.StringData(job.State.String()),
				"maxTrialCount":       llx.IntData(int64(job.MaxTrialCount)),
				"parallelTrialCount":  llx.IntData(int64(job.ParallelTrialCount)),
				"maxFailedTrialCount": llx.IntData(int64(job.MaxFailedTrialCount)),
				"studySpec":           llx.DictData(studySpec),
				"trialJobSpec":        llx.DictData(trialJobSpec),
				"labels":              llx.MapData(convert.MapToInterfaceMap(job.Labels), types.String),
				"encryptionSpec":      llx.DictData(encryptionSpec),
				"createdAt":           llx.TimeDataPtr(timestampAsTimePtr(job.CreateTime)),
				"startedAt":           llx.TimeDataPtr(timestampAsTimePtr(job.StartTime)),
				"endedAt":             llx.TimeDataPtr(timestampAsTimePtr(job.EndTime)),
				"updatedAt":           llx.TimeDataPtr(timestampAsTimePtr(job.UpdateTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlJob.(*mqlGcpProjectVertexaiServiceHyperparameterTuningJob).cacheKmsKeyName = job.GetEncryptionSpec().GetKmsKeyName()
			items = append(items, mqlJob)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServiceHyperparameterTuningJob) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

func (g *mqlGcpProjectVertexaiService) modelDeploymentMonitoringJobs() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(aiplatform.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	return g.listAcrossRegions(func(ctx context.Context, region string) ([]any, bool, error) {
		client, err := aiplatform.NewJobClient(ctx,
			option.WithCredentials(creds),
			option.WithEndpoint(vertexaiEndpoint(region)),
		)
		if err != nil {
			return nil, false, err
		}
		defer client.Close()

		it := client.ListModelDeploymentMonitoringJobs(ctx, &aiplatformpb.ListModelDeploymentMonitoringJobsRequest{
			Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, region),
		})

		var items []any
		for {
			job, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				if isVertexAIRegionSkippable(err) {
					return nil, true, nil
				}
				return nil, false, err
			}

			encryptionSpec, err := protoToDict(job.EncryptionSpec)
			if err != nil {
				return nil, false, err
			}

			mqlJob, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.modelDeploymentMonitoringJob", map[string]*llx.RawData{
				"name":                         llx.StringData(job.Name),
				"displayName":                  llx.StringData(job.DisplayName),
				"state":                        llx.StringData(job.State.String()),
				"scheduleState":                llx.StringData(job.ScheduleState.String()),
				"enableMonitoringPipelineLogs": llx.BoolData(job.EnableMonitoringPipelineLogs),
				"labels":                       llx.MapData(convert.MapToInterfaceMap(job.Labels), types.String),
				"encryptionSpec":               llx.DictData(encryptionSpec),
				"createdAt":                    llx.TimeDataPtr(timestampAsTimePtr(job.CreateTime)),
				"updatedAt":                    llx.TimeDataPtr(timestampAsTimePtr(job.UpdateTime)),
				"nextScheduleTime":             llx.TimeDataPtr(timestampAsTimePtr(job.NextScheduleTime)),
			})
			if err != nil {
				return nil, false, err
			}
			mqlMonJob := mqlJob.(*mqlGcpProjectVertexaiServiceModelDeploymentMonitoringJob)
			mqlMonJob.cacheKmsKeyName = job.GetEncryptionSpec().GetKmsKeyName()
			mqlMonJob.cacheEndpointName = job.Endpoint
			items = append(items, mqlJob)
		}
		return items, false, nil
	})
}

func (g *mqlGcpProjectVertexaiServiceModelDeploymentMonitoringJob) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

// ---------------------------------------------------------------
// KMS key references
// ---------------------------------------------------------------

type mqlGcpProjectVertexaiServiceModelInternal struct {
	cacheKmsKeyName string
}

func (a *mqlGcpProjectVertexaiServiceModel) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

type mqlGcpProjectVertexaiServiceEndpointInternal struct {
	cacheKmsKeyName string
}

func (a *mqlGcpProjectVertexaiServiceEndpoint) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

type mqlGcpProjectVertexaiServicePipelineJobInternal struct {
	cacheKmsKeyName string
}

func (a *mqlGcpProjectVertexaiServicePipelineJob) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

type mqlGcpProjectVertexaiServiceDatasetInternal struct {
	cacheKmsKeyName string
}

func (a *mqlGcpProjectVertexaiServiceDataset) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

type mqlGcpProjectVertexaiServiceFeatureOnlineStoreInternal struct {
	cacheKmsKeyName string
}

func (a *mqlGcpProjectVertexaiServiceFeatureOnlineStore) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

type mqlGcpProjectVertexaiServiceTensorboardInternal struct {
	cacheKmsKeyName string
}

func (a *mqlGcpProjectVertexaiServiceTensorboard) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

type mqlGcpProjectVertexaiServiceCustomJobInternal struct {
	cacheKmsKeyName string
}

func (a *mqlGcpProjectVertexaiServiceCustomJob) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

type mqlGcpProjectVertexaiServiceIndexInternal struct {
	cacheKmsKeyName string
}

func (a *mqlGcpProjectVertexaiServiceIndex) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

type mqlGcpProjectVertexaiServiceIndexEndpointInternal struct {
	cacheKmsKeyName string
}

func (a *mqlGcpProjectVertexaiServiceIndexEndpoint) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

type mqlGcpProjectVertexaiServiceMetadataStoreInternal struct {
	cacheKmsKeyName string
}

func (a *mqlGcpProjectVertexaiServiceMetadataStore) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

// vertexaiServiceAccountRef resolves a cached service-account email (scoped to
// the given project) to a typed gcp.project.iamService.serviceAccount, marking
// the field null when either value is missing.
func vertexaiServiceAccountRef(runtime *plugin.Runtime, field *plugin.TValue[*mqlGcpProjectIamServiceServiceAccount], projectId, email string) (*mqlGcpProjectIamServiceServiceAccount, error) {
	if email == "" || projectId == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(runtime, "gcp.project.iamService.serviceAccount", map[string]*llx.RawData{
		"projectId": llx.StringData(projectId),
		"email":     llx.StringData(email),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectIamServiceServiceAccount), nil
}

type mqlGcpProjectVertexaiServiceNotebookRuntimeInternal struct {
	cacheKmsKeyName          string
	cacheServiceAccountEmail string
	cacheProjectId           string
}

func (a *mqlGcpProjectVertexaiServiceNotebookRuntime) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

func (a *mqlGcpProjectVertexaiServiceNotebookRuntime) serviceAccount() (*mqlGcpProjectIamServiceServiceAccount, error) {
	return vertexaiServiceAccountRef(a.MqlRuntime, &a.ServiceAccount, a.cacheProjectId, a.cacheServiceAccountEmail)
}

type mqlGcpProjectVertexaiServiceNotebookRuntimeTemplateInternal struct {
	cacheKmsKeyName string
}

func (a *mqlGcpProjectVertexaiServiceNotebookRuntimeTemplate) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

type mqlGcpProjectVertexaiServiceNotebookExecutionJobInternal struct {
	cacheKmsKeyName string
}

func (a *mqlGcpProjectVertexaiServiceNotebookExecutionJob) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

type mqlGcpProjectVertexaiServiceReasoningEngineInternal struct {
	cacheKmsKeyName string
	cacheProjectId  string
}

func (a *mqlGcpProjectVertexaiServiceReasoningEngine) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

type mqlGcpProjectVertexaiServiceRagCorpusInternal struct {
	cacheKmsKeyName string
}

func (a *mqlGcpProjectVertexaiServiceRagCorpus) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

type mqlGcpProjectVertexaiServicePersistentResourceInternal struct {
	cacheKmsKeyName string
	cacheNetworkUrl string
}

func (a *mqlGcpProjectVertexaiServicePersistentResource) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

func (a *mqlGcpProjectVertexaiServicePersistentResource) network() (*mqlGcpProjectComputeServiceNetwork, error) {
	if a.cacheNetworkUrl == "" {
		a.Network.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return getNetworkByUrl(a.cacheNetworkUrl, a.MqlRuntime)
}

type mqlGcpProjectVertexaiServiceDeploymentResourcePoolInternal struct {
	cacheKmsKeyName          string
	cacheProjectId           string
	cacheServiceAccountEmail string
}

func (a *mqlGcpProjectVertexaiServiceDeploymentResourcePool) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

func (a *mqlGcpProjectVertexaiServiceDeploymentResourcePool) serviceAccount() (*mqlGcpProjectIamServiceServiceAccount, error) {
	if a.cacheServiceAccountEmail == "" || a.cacheProjectId == "" {
		a.ServiceAccount.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	res, err := NewResource(a.MqlRuntime, "gcp.project.iamService.serviceAccount", map[string]*llx.RawData{
		"projectId": llx.StringData(a.cacheProjectId),
		"email":     llx.StringData(a.cacheServiceAccountEmail),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectIamServiceServiceAccount), nil
}

type mqlGcpProjectVertexaiServiceCachedContentInternal struct {
	cacheKmsKeyName string
}

func (a *mqlGcpProjectVertexaiServiceCachedContent) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

type mqlGcpProjectVertexaiServiceBatchPredictionJobInternal struct {
	cacheKmsKeyName          string
	cacheModelName           string
	cacheServiceAccountEmail string
	cacheProjectId           string
}

func (a *mqlGcpProjectVertexaiServiceBatchPredictionJob) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

func (a *mqlGcpProjectVertexaiServiceBatchPredictionJob) serviceAccount() (*mqlGcpProjectIamServiceServiceAccount, error) {
	return vertexaiServiceAccountRef(a.MqlRuntime, &a.ServiceAccount, a.cacheProjectId, a.cacheServiceAccountEmail)
}

func (a *mqlGcpProjectVertexaiServiceBatchPredictionJob) model() (*mqlGcpProjectVertexaiServiceModel, error) {
	return resolveVertexaiModelRef(a.MqlRuntime, &a.Model, a.cacheModelName)
}

type mqlGcpProjectVertexaiServiceTuningJobInternal struct {
	cacheKmsKeyName          string
	cacheServiceAccountEmail string
	cacheProjectId           string
}

func (a *mqlGcpProjectVertexaiServiceTuningJob) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

func (a *mqlGcpProjectVertexaiServiceTuningJob) serviceAccount() (*mqlGcpProjectIamServiceServiceAccount, error) {
	return vertexaiServiceAccountRef(a.MqlRuntime, &a.ServiceAccount, a.cacheProjectId, a.cacheServiceAccountEmail)
}

type mqlGcpProjectVertexaiServiceTrainingPipelineInternal struct {
	cacheKmsKeyName  string
	cacheModelName   string
	cacheParentModel string
}

func (a *mqlGcpProjectVertexaiServiceTrainingPipeline) model() (*mqlGcpProjectVertexaiServiceModel, error) {
	return resolveVertexaiModelRef(a.MqlRuntime, &a.Model, a.cacheModelName)
}

func (a *mqlGcpProjectVertexaiServiceTrainingPipeline) parentModel() (*mqlGcpProjectVertexaiServiceModel, error) {
	return resolveVertexaiModelRef(a.MqlRuntime, &a.ParentModel, a.cacheParentModel)
}

func (a *mqlGcpProjectVertexaiServiceTrainingPipeline) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

type mqlGcpProjectVertexaiServiceHyperparameterTuningJobInternal struct {
	cacheKmsKeyName string
}

func (a *mqlGcpProjectVertexaiServiceHyperparameterTuningJob) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

type mqlGcpProjectVertexaiServiceModelDeploymentMonitoringJobInternal struct {
	cacheKmsKeyName   string
	cacheEndpointName string
}

func (a *mqlGcpProjectVertexaiServiceModelDeploymentMonitoringJob) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(a.MqlRuntime, &a.KmsKey, a.cacheKmsKeyName)
}

func (a *mqlGcpProjectVertexaiServiceModelDeploymentMonitoringJob) endpoint() (*mqlGcpProjectVertexaiServiceEndpoint, error) {
	return resolveVertexaiEndpointRef(a.MqlRuntime, &a.Endpoint, a.cacheEndpointName)
}
