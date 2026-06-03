// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/generativeai"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciAiGenerativeAi) id() (string, error) {
	return "oci.ai.generativeAi", nil
}

func (o *mqlOciAiGenerativeAi) compartmentID() string {
	return o.MqlRuntime.Connection.(*connection.OciConnection).TenantID()
}

// listRegional runs fetch against the Generative AI API in every subscribed
// region concurrently and flattens the results. Regions where Generative AI is
// not available are skipped (see ociRegionServiceUnavailable).
func (o *mqlOciAiGenerativeAi) listRegional(fetch func(svc *generativeai.GenerativeAiClient) ([]any, error)) ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	regions, err := ociAgentRegions(o.MqlRuntime)
	if err != nil {
		return nil, err
	}

	tasks := make([]*jobpool.Job, 0, len(regions))
	for _, region := range regions {
		regionResource, ok := region.(*mqlOciRegion)
		if !ok {
			return nil, errors.New("invalid region type")
		}
		regionID := regionResource.Id.Data
		tasks = append(tasks, jobpool.NewJob(func() (jobpool.JobResult, error) {
			svc, err := conn.GenerativeAiClient(regionID)
			if err != nil {
				return nil, err
			}
			items, err := fetch(svc)
			if err != nil {
				if ociRegionServiceUnavailable(err) {
					return jobpool.JobResult([]any{}), nil
				}
				return nil, err
			}
			return jobpool.JobResult(items), nil
		}))
	}

	poolOfJobs := jobpool.CreatePool(tasks, 5)
	poolOfJobs.Run()
	if poolOfJobs.HasErrors() {
		return nil, poolOfJobs.GetErrors()
	}
	res := []any{}
	for i := range poolOfJobs.Jobs {
		res = append(res, poolOfJobs.Jobs[i].Result.([]any)...)
	}
	return res, nil
}

// ----- dedicated AI clusters -----

func (o *mqlOciAiGenerativeAi) dedicatedAiClusters() ([]any, error) {
	return o.listRegional(o.fetchDedicatedAiClusters)
}

func (o *mqlOciAiGenerativeAi) fetchDedicatedAiClusters(svc *generativeai.GenerativeAiClient) ([]any, error) {
	ctx := context.Background()
	var items []generativeai.DedicatedAiClusterSummary
	var page *string
	for {
		resp, err := svc.ListDedicatedAiClusters(ctx, generativeai.ListDedicatedAiClustersRequest{
			CompartmentId: common.String(o.compartmentID()),
			Page:          page,
		})
		if err != nil {
			return nil, err
		}
		items = append(items, resp.Items...)
		if resp.OpcNextPage == nil {
			break
		}
		page = resp.OpcNextPage
	}

	res := make([]any, 0, len(items))
	for i := range items {
		c := items[i]
		capacity, err := convert.JsonToDict(c.Capacity)
		if err != nil {
			return nil, err
		}
		mqlCluster, err := CreateResource(o.MqlRuntime, "oci.ai.generativeAi.dedicatedAiCluster", map[string]*llx.RawData{
			"id":               llx.StringDataPtr(c.Id),
			"name":             llx.StringDataPtr(c.DisplayName),
			"compartmentID":    llx.StringDataPtr(c.CompartmentId),
			"type":             llx.StringData(string(c.Type)),
			"unitCount":        llx.IntDataPtr(c.UnitCount),
			"unitShape":        llx.StringData(string(c.UnitShape)),
			"capacity":         llx.DictData(capacity),
			"description":      llx.StringDataPtr(c.Description),
			"state":            llx.StringData(string(c.LifecycleState)),
			"lifecycleDetails": llx.StringDataPtr(c.LifecycleDetails),
			"created":          sdkTimeData(c.TimeCreated),
			"timeUpdated":      sdkTimeData(c.TimeUpdated),
			"freeformTags":     llx.MapData(strMapToAny(c.FreeformTags), types.String),
			"definedTags":      llx.MapData(definedTagsToAny(c.DefinedTags), types.Any),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlCluster)
	}
	return res, nil
}

func initOciAiGenerativeAiDedicatedAiCluster(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return findOciGenerativeAiByID(runtime, args, (*mqlOciAiGenerativeAi).GetDedicatedAiClusters)
}

func (o *mqlOciAiGenerativeAiDedicatedAiCluster) id() (string, error) {
	return "oci.ai.generativeAi.dedicatedAiCluster/" + o.Id.Data, nil
}

func (o *mqlOciAiGenerativeAiDedicatedAiCluster) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

// ----- models -----

func (o *mqlOciAiGenerativeAi) models() ([]any, error) {
	return o.listRegional(o.fetchModels)
}

func (o *mqlOciAiGenerativeAi) fetchModels(svc *generativeai.GenerativeAiClient) ([]any, error) {
	ctx := context.Background()
	var items []generativeai.ModelSummary
	var page *string
	for {
		resp, err := svc.ListModels(ctx, generativeai.ListModelsRequest{
			CompartmentId: common.String(o.compartmentID()),
			Page:          page,
		})
		if err != nil {
			return nil, err
		}
		items = append(items, resp.Items...)
		if resp.OpcNextPage == nil {
			break
		}
		page = resp.OpcNextPage
	}

	res := make([]any, 0, len(items))
	for i := range items {
		m := items[i]

		capabilities := make([]any, 0, len(m.Capabilities))
		for _, capability := range m.Capabilities {
			capabilities = append(capabilities, string(capability))
		}
		fineTune, err := convert.JsonToDict(m.FineTuneDetails)
		if err != nil {
			return nil, err
		}

		mqlModel, err := CreateResource(o.MqlRuntime, "oci.ai.generativeAi.model", map[string]*llx.RawData{
			"id":                  llx.StringDataPtr(m.Id),
			"name":                llx.StringDataPtr(m.DisplayName),
			"compartmentID":       llx.StringDataPtr(m.CompartmentId),
			"type":                llx.StringData(string(m.Type)),
			"capabilities":        llx.ArrayData(capabilities, types.String),
			"vendor":              llx.StringDataPtr(m.Vendor),
			"version":             llx.StringDataPtr(m.Version),
			"baseModelID":         llx.StringDataPtr(m.BaseModelId),
			"fineTuneDetails":     llx.DictData(fineTune),
			"isLongTermSupported": llx.BoolDataPtr(m.IsLongTermSupported),
			"state":               llx.StringData(string(m.LifecycleState)),
			"lifecycleDetails":    llx.StringDataPtr(m.LifecycleDetails),
			"timeDeprecated":      sdkTimeData(m.TimeDeprecated),
			"created":             sdkTimeData(m.TimeCreated),
			"freeformTags":        llx.MapData(strMapToAny(m.FreeformTags), types.String),
			"definedTags":         llx.MapData(definedTagsToAny(m.DefinedTags), types.Any),
		})
		if err != nil {
			return nil, err
		}
		mqlModelTyped := mqlModel.(*mqlOciAiGenerativeAiModel)
		mqlModelTyped.cacheBaseModelID = stringValue(m.BaseModelId)
		res = append(res, mqlModelTyped)
	}
	return res, nil
}

type mqlOciAiGenerativeAiModelInternal struct {
	cacheBaseModelID string
}

func initOciAiGenerativeAiModel(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return findOciGenerativeAiByID(runtime, args, (*mqlOciAiGenerativeAi).GetModels)
}

func (o *mqlOciAiGenerativeAiModel) id() (string, error) {
	return "oci.ai.generativeAi.model/" + o.Id.Data, nil
}

func (o *mqlOciAiGenerativeAiModel) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciAiGenerativeAiModel) baseModel() (*mqlOciAiGenerativeAiModel, error) {
	return resolveOciGenerativeAiModel(o.MqlRuntime, o.cacheBaseModelID, &o.BaseModel)
}

// ----- endpoints -----

func (o *mqlOciAiGenerativeAi) endpoints() ([]any, error) {
	return o.listRegional(o.fetchEndpoints)
}

func (o *mqlOciAiGenerativeAi) fetchEndpoints(svc *generativeai.GenerativeAiClient) ([]any, error) {
	ctx := context.Background()
	var items []generativeai.EndpointSummary
	var page *string
	for {
		resp, err := svc.ListEndpoints(ctx, generativeai.ListEndpointsRequest{
			CompartmentId: common.String(o.compartmentID()),
			Page:          page,
		})
		if err != nil {
			return nil, err
		}
		items = append(items, resp.Items...)
		if resp.OpcNextPage == nil {
			break
		}
		page = resp.OpcNextPage
	}

	res := make([]any, 0, len(items))
	for i := range items {
		e := items[i]

		var cmEnabled, piEnabled, piiEnabled bool
		var cmMode, piMode, piiMode string
		if cfg := e.ContentModerationConfig; cfg != nil {
			cmEnabled = boolValue(cfg.IsEnabled)
			cmMode = string(cfg.Mode)
		}
		if cfg := e.PromptInjectionConfig; cfg != nil {
			piEnabled = boolValue(cfg.IsEnabled)
			piMode = string(cfg.Mode)
		}
		if cfg := e.PiiDetectionConfig; cfg != nil {
			piiEnabled = boolValue(cfg.IsEnabled)
			piiMode = string(cfg.Mode)
		}

		mqlEndpoint, err := CreateResource(o.MqlRuntime, "oci.ai.generativeAi.endpoint", map[string]*llx.RawData{
			"id":                            llx.StringDataPtr(e.Id),
			"name":                          llx.StringDataPtr(e.DisplayName),
			"compartmentID":                 llx.StringDataPtr(e.CompartmentId),
			"modelID":                       llx.StringDataPtr(e.ModelId),
			"dedicatedAiClusterID":          llx.StringDataPtr(e.DedicatedAiClusterId),
			"generativeAiPrivateEndpointID": llx.StringDataPtr(e.GenerativeAiPrivateEndpointId),
			"description":                   llx.StringDataPtr(e.Description),
			"contentModerationEnabled":      llx.BoolData(cmEnabled),
			"contentModerationMode":         llx.StringData(cmMode),
			"promptInjectionEnabled":        llx.BoolData(piEnabled),
			"promptInjectionMode":           llx.StringData(piMode),
			"piiDetectionEnabled":           llx.BoolData(piiEnabled),
			"piiDetectionMode":              llx.StringData(piiMode),
			"state":                         llx.StringData(string(e.LifecycleState)),
			"lifecycleDetails":              llx.StringDataPtr(e.LifecycleDetails),
			"created":                       sdkTimeData(e.TimeCreated),
			"timeUpdated":                   sdkTimeData(e.TimeUpdated),
			"freeformTags":                  llx.MapData(strMapToAny(e.FreeformTags), types.String),
			"definedTags":                   llx.MapData(definedTagsToAny(e.DefinedTags), types.Any),
		})
		if err != nil {
			return nil, err
		}
		mqlEndpointTyped := mqlEndpoint.(*mqlOciAiGenerativeAiEndpoint)
		mqlEndpointTyped.cacheModelID = stringValue(e.ModelId)
		mqlEndpointTyped.cacheDedicatedAiClusterID = stringValue(e.DedicatedAiClusterId)
		res = append(res, mqlEndpointTyped)
	}
	return res, nil
}

type mqlOciAiGenerativeAiEndpointInternal struct {
	cacheModelID              string
	cacheDedicatedAiClusterID string
}

func initOciAiGenerativeAiEndpoint(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return findOciGenerativeAiByID(runtime, args, (*mqlOciAiGenerativeAi).GetEndpoints)
}

func (o *mqlOciAiGenerativeAiEndpoint) id() (string, error) {
	return "oci.ai.generativeAi.endpoint/" + o.Id.Data, nil
}

func (o *mqlOciAiGenerativeAiEndpoint) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciAiGenerativeAiEndpoint) model() (*mqlOciAiGenerativeAiModel, error) {
	return resolveOciGenerativeAiModel(o.MqlRuntime, o.cacheModelID, &o.Model)
}

func (o *mqlOciAiGenerativeAiEndpoint) dedicatedAiCluster() (*mqlOciAiGenerativeAiDedicatedAiCluster, error) {
	if o.cacheDedicatedAiClusterID == "" {
		o.DedicatedAiCluster.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.ai.generativeAi.dedicatedAiCluster", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheDedicatedAiClusterID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciAiGenerativeAiDedicatedAiCluster), nil
}

// ----- helpers -----

// findOciGenerativeAiByID powers the init functions for Generative AI
// resources: it lists the requested collection and returns the entry whose id
// matches the "id" argument, so a resource can be selected directly by OCID.
func findOciGenerativeAiByID(runtime *plugin.Runtime, args map[string]*llx.RawData, list func(svc *mqlOciAiGenerativeAi) *plugin.TValue[[]any]) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idVal := ociArgString(args, "id")
	if idVal == "" {
		return args, nil, nil
	}
	obj, err := CreateResource(runtime, "oci.ai.generativeAi", nil)
	if err != nil {
		return nil, nil, err
	}
	svc := obj.(*mqlOciAiGenerativeAi)
	items := list(svc)
	if items.Error != nil {
		return nil, nil, items.Error
	}
	for _, raw := range items.Data {
		res := raw.(plugin.Resource)
		if idField, ok := raw.(interface{ GetId() *plugin.TValue[string] }); ok && idField.GetId().Data == idVal {
			return args, res, nil
		}
	}
	return args, nil, nil
}

// resolveOciGenerativeAiModel resolves a typed model resource from a model
// OCID. Returns (nil, nil) and marks the field null when the OCID is empty.
func resolveOciGenerativeAiModel(runtime *plugin.Runtime, id string, field *plugin.TValue[*mqlOciAiGenerativeAiModel]) (*mqlOciAiGenerativeAiModel, error) {
	if id == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(runtime, "oci.ai.generativeAi.model", map[string]*llx.RawData{
		"id": llx.StringData(id),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciAiGenerativeAiModel), nil
}
