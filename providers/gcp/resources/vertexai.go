// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
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
	enabledRegions []string
	regionsFetched bool
	lock           sync.Mutex
}

// getRegions returns the cached list of enabled regions, or all known regions
// if no discovery has been done yet.
func (g *mqlGcpProjectVertexaiService) getRegions() []string {
	g.lock.Lock()
	defer g.lock.Unlock()
	if g.regionsFetched {
		return g.enabledRegions
	}
	return vertexaiRegions
}

// cacheEnabledRegions stores the discovered enabled regions. Only the first
// caller's results are cached; subsequent calls are no-ops.
func (g *mqlGcpProjectVertexaiService) cacheEnabledRegions(regions []string) {
	g.lock.Lock()
	defer g.lock.Unlock()
	if !g.regionsFetched {
		g.enabledRegions = regions
		g.regionsFetched = true
	}
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

	ctx := context.Background()
	var res []any
	var enabledRegions []string
	for _, region := range g.getRegions() {
		items, skipped, err := func() ([]any, bool, error) {
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

				modelSourceInfo, err := protoToDict(model.ModelSourceInfo)
				if err != nil {
					return nil, false, err
				}
				containerSpec, err := protoToDict(model.ContainerSpec)
				if err != nil {
					return nil, false, err
				}
				encryptionSpec, err := protoToDict(model.EncryptionSpec)
				if err != nil {
					return nil, false, err
				}

				deploymentTypes := make([]any, 0, len(model.SupportedDeploymentResourcesTypes))
				for _, dt := range model.SupportedDeploymentResourcesTypes {
					deploymentTypes = append(deploymentTypes, dt.String())
				}

				mqlModel, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.model", map[string]*llx.RawData{
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
					return nil, false, err
				}
				items = append(items, mqlModel)
			}
			return items, false, nil
		}()
		if err != nil {
			return nil, err
		}
		if !skipped {
			enabledRegions = append(enabledRegions, region)
		}
		res = append(res, items...)
	}
	g.cacheEnabledRegions(enabledRegions)
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

	ctx := context.Background()
	var res []any
	var enabledRegions []string
	for _, region := range g.getRegions() {
		items, skipped, err := func() ([]any, bool, error) {
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

				deployedModels := make([]any, 0, len(ep.DeployedModels))
				for _, dm := range ep.DeployedModels {
					d, err := protoToDict(dm)
					if err != nil {
						return nil, false, err
					}
					deployedModels = append(deployedModels, d)
				}
				encryptionSpec, err := protoToDict(ep.EncryptionSpec)
				if err != nil {
					return nil, false, err
				}

				trafficSplit := make(map[string]any, len(ep.TrafficSplit))
				for k, v := range ep.TrafficSplit {
					trafficSplit[k] = int64(v)
				}

				mqlEndpoint, err := CreateResource(g.MqlRuntime, "gcp.project.vertexaiService.endpoint", map[string]*llx.RawData{
					"name":                        llx.StringData(ep.Name),
					"displayName":                 llx.StringData(ep.DisplayName),
					"description":                 llx.StringData(ep.Description),
					"deployedModels":              llx.ArrayData(deployedModels, types.Dict),
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
					return nil, false, err
				}
				items = append(items, mqlEndpoint)
			}
			return items, false, nil
		}()
		if err != nil {
			return nil, err
		}
		if !skipped {
			enabledRegions = append(enabledRegions, region)
		}
		res = append(res, items...)
	}
	g.cacheEnabledRegions(enabledRegions)
	return res, nil
}

func (g *mqlGcpProjectVertexaiServiceEndpoint) id() (string, error) {
	return g.Name.Data, g.Name.Error
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

	ctx := context.Background()
	var res []any
	var enabledRegions []string
	for _, region := range g.getRegions() {
		items, skipped, err := func() ([]any, bool, error) {
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
				items = append(items, mqlJob)
			}
			return items, false, nil
		}()
		if err != nil {
			return nil, err
		}
		if !skipped {
			enabledRegions = append(enabledRegions, region)
		}
		res = append(res, items...)
	}
	g.cacheEnabledRegions(enabledRegions)
	return res, nil
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

	ctx := context.Background()
	var res []any
	var enabledRegions []string
	for _, region := range g.getRegions() {
		items, skipped, err := func() ([]any, bool, error) {
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
				items = append(items, mqlDs)
			}
			return items, false, nil
		}()
		if err != nil {
			return nil, err
		}
		if !skipped {
			enabledRegions = append(enabledRegions, region)
		}
		res = append(res, items...)
	}
	g.cacheEnabledRegions(enabledRegions)
	return res, nil
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

	ctx := context.Background()
	var res []any
	var enabledRegions []string
	for _, region := range g.getRegions() {
		items, skipped, err := func() ([]any, bool, error) {
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
				items = append(items, mqlStore)
			}
			return items, false, nil
		}()
		if err != nil {
			return nil, err
		}
		if !skipped {
			enabledRegions = append(enabledRegions, region)
		}
		res = append(res, items...)
	}
	g.cacheEnabledRegions(enabledRegions)
	return res, nil
}

func (g *mqlGcpProjectVertexaiServiceFeatureOnlineStore) id() (string, error) {
	return g.Name.Data, g.Name.Error
}
