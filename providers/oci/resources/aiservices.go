// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/oracle/oci-go-sdk/v65/aidocument"
	"github.com/oracle/oci-go-sdk/v65/ailanguage"
	"github.com/oracle/oci-go-sdk/v65/aispeech"
	"github.com/oracle/oci-go-sdk/v65/aivision"
	"github.com/oracle/oci-go-sdk/v65/common"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/jobpool"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

// ociListRegionalAI runs fetch against a single-purpose AI service in every
// subscribed region concurrently and flattens the results. Regions where the
// service is not available are skipped (see ociRegionServiceUnavailable).
func ociListRegionalAI(runtime *plugin.Runtime, fetch func(region string) ([]any, error)) ([]any, error) {
	regions, err := ociAgentRegions(runtime)
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
			items, err := fetch(regionID)
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

// ociListRegionalAIClient adds typed per-region client creation to
// ociListRegionalAI: it builds the client with factory and hands it to fetch,
// so each call site only supplies the list-and-map logic.
func ociListRegionalAIClient[C any](runtime *plugin.Runtime, factory func(region string) (C, error), fetch func(svc C) ([]any, error)) ([]any, error) {
	return ociListRegionalAI(runtime, func(region string) ([]any, error) {
		svc, err := factory(region)
		if err != nil {
			return nil, err
		}
		return fetch(svc)
	})
}

// findOciAIResourceByID powers the init functions for the single-purpose AI
// resources: it lists the requested collection and returns the entry whose id
// matches the "id" argument, so a resource can be selected directly by OCID.
func findOciAIResourceByID(runtime *plugin.Runtime, args map[string]*llx.RawData, serviceName string, list func(plugin.Resource) *plugin.TValue[[]any]) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idVal := ociArgString(args, "id")
	if idVal == "" {
		return args, nil, nil
	}
	obj, err := CreateResource(runtime, serviceName, nil)
	if err != nil {
		return nil, nil, err
	}
	items := list(obj)
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

// =====================================================================
// Language
// =====================================================================

func (o *mqlOciAiLanguage) id() (string, error) { return "oci.ai.language", nil }

func (o *mqlOciAiLanguage) projects() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	return ociListRegionalAIClient(o.MqlRuntime, conn.AILanguageClient, func(svc *ailanguage.AIServiceLanguageClient) ([]any, error) {
		var items []ailanguage.ProjectSummary
		var page *string
		for {
			resp, err := svc.ListProjects(context.Background(), ailanguage.ListProjectsRequest{
				CompartmentId: common.String(conn.TenantID()), Page: page,
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
			p := items[i]
			r, err := CreateResource(o.MqlRuntime, "oci.ai.language.project", map[string]*llx.RawData{
				"id":               llx.StringDataPtr(p.Id),
				"name":             llx.StringDataPtr(p.DisplayName),
				"compartmentID":    llx.StringDataPtr(p.CompartmentId),
				"description":      llx.StringDataPtr(p.Description),
				"state":            llx.StringData(string(p.LifecycleState)),
				"lifecycleDetails": llx.StringDataPtr(p.LifecycleDetails),
				"created":          sdkTimeData(p.TimeCreated),
				"freeformTags":     llx.MapData(strMapToAny(p.FreeformTags), types.String),
				"definedTags":      llx.MapData(definedTagsToAny(p.DefinedTags), types.Any),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}
		return res, nil
	})
}

func (o *mqlOciAiLanguage) models() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	return ociListRegionalAIClient(o.MqlRuntime, conn.AILanguageClient, func(svc *ailanguage.AIServiceLanguageClient) ([]any, error) {
		var items []ailanguage.ModelSummary
		var page *string
		for {
			resp, err := svc.ListModels(context.Background(), ailanguage.ListModelsRequest{
				CompartmentId: common.String(conn.TenantID()), Page: page,
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
			details, err := convert.JsonToDict(m.ModelDetails)
			if err != nil {
				return nil, err
			}
			r, err := CreateResource(o.MqlRuntime, "oci.ai.language.model", map[string]*llx.RawData{
				"id":               llx.StringDataPtr(m.Id),
				"name":             llx.StringDataPtr(m.DisplayName),
				"compartmentID":    llx.StringDataPtr(m.CompartmentId),
				"projectID":        llx.StringDataPtr(m.ProjectId),
				"description":      llx.StringDataPtr(m.Description),
				"version":          llx.StringDataPtr(m.Version),
				"modelDetails":     llx.DictData(details),
				"state":            llx.StringData(string(m.LifecycleState)),
				"lifecycleDetails": llx.StringDataPtr(m.LifecycleDetails),
				"created":          sdkTimeData(m.TimeCreated),
				"freeformTags":     llx.MapData(strMapToAny(m.FreeformTags), types.String),
				"definedTags":      llx.MapData(definedTagsToAny(m.DefinedTags), types.Any),
			})
			if err != nil {
				return nil, err
			}
			r.(*mqlOciAiLanguageModel).cacheProjectID = stringValue(m.ProjectId)
			res = append(res, r)
		}
		return res, nil
	})
}

func (o *mqlOciAiLanguage) endpoints() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	return ociListRegionalAIClient(o.MqlRuntime, conn.AILanguageClient, func(svc *ailanguage.AIServiceLanguageClient) ([]any, error) {
		var items []ailanguage.EndpointSummary
		var page *string
		for {
			resp, err := svc.ListEndpoints(context.Background(), ailanguage.ListEndpointsRequest{
				CompartmentId: common.String(conn.TenantID()), Page: page,
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
			r, err := CreateResource(o.MqlRuntime, "oci.ai.language.endpoint", map[string]*llx.RawData{
				"id":               llx.StringDataPtr(e.Id),
				"name":             llx.StringDataPtr(e.DisplayName),
				"compartmentID":    llx.StringDataPtr(e.CompartmentId),
				"projectID":        llx.StringDataPtr(e.ProjectId),
				"modelID":          llx.StringDataPtr(e.ModelId),
				"alias":            llx.StringDataPtr(e.Alias),
				"description":      llx.StringDataPtr(e.Description),
				"inferenceUnits":   llx.IntData(intValue(e.InferenceUnits)),
				"state":            llx.StringData(string(e.LifecycleState)),
				"lifecycleDetails": llx.StringDataPtr(e.LifecycleDetails),
				"created":          sdkTimeData(e.TimeCreated),
				"freeformTags":     llx.MapData(strMapToAny(e.FreeformTags), types.String),
				"definedTags":      llx.MapData(definedTagsToAny(e.DefinedTags), types.Any),
			})
			if err != nil {
				return nil, err
			}
			ep := r.(*mqlOciAiLanguageEndpoint)
			ep.cacheProjectID = stringValue(e.ProjectId)
			ep.cacheModelID = stringValue(e.ModelId)
			res = append(res, ep)
		}
		return res, nil
	})
}

type mqlOciAiLanguageModelInternal struct {
	cacheProjectID string
}
type mqlOciAiLanguageEndpointInternal struct {
	cacheProjectID string
	cacheModelID   string
}

func initOciAiLanguageProject(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return findOciAIResourceByID(runtime, args, "oci.ai.language", func(o plugin.Resource) *plugin.TValue[[]any] { return o.(*mqlOciAiLanguage).GetProjects() })
}

func initOciAiLanguageModel(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return findOciAIResourceByID(runtime, args, "oci.ai.language", func(o plugin.Resource) *plugin.TValue[[]any] { return o.(*mqlOciAiLanguage).GetModels() })
}

func initOciAiLanguageEndpoint(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return findOciAIResourceByID(runtime, args, "oci.ai.language", func(o plugin.Resource) *plugin.TValue[[]any] { return o.(*mqlOciAiLanguage).GetEndpoints() })
}

func (o *mqlOciAiLanguageProject) id() (string, error) {
	return "oci.ai.language.project/" + o.Id.Data, nil
}
func (o *mqlOciAiLanguageProject) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciAiLanguageModel) id() (string, error) {
	return "oci.ai.language.model/" + o.Id.Data, nil
}
func (o *mqlOciAiLanguageModel) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}
func (o *mqlOciAiLanguageModel) project() (*mqlOciAiLanguageProject, error) {
	if o.cacheProjectID == "" {
		o.Project.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.ai.language.project", map[string]*llx.RawData{"id": llx.StringData(o.cacheProjectID)})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciAiLanguageProject), nil
}

func (o *mqlOciAiLanguageEndpoint) id() (string, error) {
	return "oci.ai.language.endpoint/" + o.Id.Data, nil
}
func (o *mqlOciAiLanguageEndpoint) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}
func (o *mqlOciAiLanguageEndpoint) project() (*mqlOciAiLanguageProject, error) {
	if o.cacheProjectID == "" {
		o.Project.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.ai.language.project", map[string]*llx.RawData{"id": llx.StringData(o.cacheProjectID)})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciAiLanguageProject), nil
}
func (o *mqlOciAiLanguageEndpoint) model() (*mqlOciAiLanguageModel, error) {
	if o.cacheModelID == "" {
		o.Model.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.ai.language.model", map[string]*llx.RawData{"id": llx.StringData(o.cacheModelID)})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciAiLanguageModel), nil
}

// =====================================================================
// Vision
// =====================================================================

func (o *mqlOciAiVision) id() (string, error) { return "oci.ai.vision", nil }

func (o *mqlOciAiVision) projects() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	return ociListRegionalAIClient(o.MqlRuntime, conn.AIVisionClient, func(svc *aivision.AIServiceVisionClient) ([]any, error) {
		var items []aivision.ProjectSummary
		var page *string
		for {
			resp, err := svc.ListProjects(context.Background(), aivision.ListProjectsRequest{
				CompartmentId: common.String(conn.TenantID()), Page: page,
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
			p := items[i]
			r, err := CreateResource(o.MqlRuntime, "oci.ai.vision.project", map[string]*llx.RawData{
				"id":               llx.StringDataPtr(p.Id),
				"name":             llx.StringDataPtr(p.DisplayName),
				"compartmentID":    llx.StringDataPtr(p.CompartmentId),
				"state":            llx.StringData(string(p.LifecycleState)),
				"lifecycleDetails": llx.StringDataPtr(p.LifecycleDetails),
				"created":          sdkTimeData(p.TimeCreated),
				"freeformTags":     llx.MapData(strMapToAny(p.FreeformTags), types.String),
				"definedTags":      llx.MapData(definedTagsToAny(p.DefinedTags), types.Any),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}
		return res, nil
	})
}

func (o *mqlOciAiVision) models() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	return ociListRegionalAIClient(o.MqlRuntime, conn.AIVisionClient, func(svc *aivision.AIServiceVisionClient) ([]any, error) {
		var items []aivision.ModelSummary
		var page *string
		for {
			resp, err := svc.ListModels(context.Background(), aivision.ListModelsRequest{
				CompartmentId: common.String(conn.TenantID()), Page: page,
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
			var prec float64
			if m.Precision != nil {
				prec = float64(*m.Precision)
			}
			r, err := CreateResource(o.MqlRuntime, "oci.ai.vision.model", map[string]*llx.RawData{
				"id":               llx.StringDataPtr(m.Id),
				"name":             llx.StringDataPtr(m.DisplayName),
				"compartmentID":    llx.StringDataPtr(m.CompartmentId),
				"projectID":        llx.StringDataPtr(m.ProjectId),
				"modelType":        llx.StringData(string(m.ModelType)),
				"modelVersion":     llx.StringDataPtr(m.ModelVersion),
				"description":      llx.StringDataPtr(m.Description),
				"precision":        llx.FloatData(prec),
				"state":            llx.StringData(string(m.LifecycleState)),
				"lifecycleDetails": llx.StringDataPtr(m.LifecycleDetails),
				"created":          sdkTimeData(m.TimeCreated),
				"freeformTags":     llx.MapData(strMapToAny(m.FreeformTags), types.String),
				"definedTags":      llx.MapData(definedTagsToAny(m.DefinedTags), types.Any),
			})
			if err != nil {
				return nil, err
			}
			r.(*mqlOciAiVisionModel).cacheProjectID = stringValue(m.ProjectId)
			res = append(res, r)
		}
		return res, nil
	})
}

type mqlOciAiVisionModelInternal struct {
	cacheProjectID string
}

func initOciAiVisionProject(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return findOciAIResourceByID(runtime, args, "oci.ai.vision", func(o plugin.Resource) *plugin.TValue[[]any] { return o.(*mqlOciAiVision).GetProjects() })
}

func initOciAiVisionModel(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return findOciAIResourceByID(runtime, args, "oci.ai.vision", func(o plugin.Resource) *plugin.TValue[[]any] { return o.(*mqlOciAiVision).GetModels() })
}

func (o *mqlOciAiVisionProject) id() (string, error) {
	return "oci.ai.vision.project/" + o.Id.Data, nil
}
func (o *mqlOciAiVisionProject) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciAiVisionModel) id() (string, error) { return "oci.ai.vision.model/" + o.Id.Data, nil }
func (o *mqlOciAiVisionModel) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}
func (o *mqlOciAiVisionModel) project() (*mqlOciAiVisionProject, error) {
	if o.cacheProjectID == "" {
		o.Project.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.ai.vision.project", map[string]*llx.RawData{"id": llx.StringData(o.cacheProjectID)})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciAiVisionProject), nil
}

// =====================================================================
// Speech
// =====================================================================

func (o *mqlOciAiSpeech) id() (string, error) { return "oci.ai.speech", nil }

func (o *mqlOciAiSpeech) customizations() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	return ociListRegionalAIClient(o.MqlRuntime, conn.AISpeechClient, func(svc *aispeech.AIServiceSpeechClient) ([]any, error) {
		var items []aispeech.CustomizationSummary
		var page *string
		for {
			resp, err := svc.ListCustomizations(context.Background(), aispeech.ListCustomizationsRequest{
				CompartmentId: common.String(conn.TenantID()), Page: page,
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
			r, err := CreateResource(o.MqlRuntime, "oci.ai.speech.customization", map[string]*llx.RawData{
				"id":               llx.StringDataPtr(c.Id),
				"name":             llx.StringDataPtr(c.DisplayName),
				"compartmentID":    llx.StringDataPtr(c.CompartmentId),
				"alias":            llx.StringDataPtr(c.Alias),
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
			res = append(res, r)
		}
		return res, nil
	})
}

func initOciAiSpeechCustomization(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return findOciAIResourceByID(runtime, args, "oci.ai.speech", func(o plugin.Resource) *plugin.TValue[[]any] { return o.(*mqlOciAiSpeech).GetCustomizations() })
}

func (o *mqlOciAiSpeechCustomization) id() (string, error) {
	return "oci.ai.speech.customization/" + o.Id.Data, nil
}
func (o *mqlOciAiSpeechCustomization) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

// =====================================================================
// Document Understanding
// =====================================================================

func (o *mqlOciAiDocument) id() (string, error) { return "oci.ai.document", nil }

func (o *mqlOciAiDocument) projects() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	return ociListRegionalAIClient(o.MqlRuntime, conn.AIDocumentClient, func(svc *aidocument.AIServiceDocumentClient) ([]any, error) {
		var items []aidocument.ProjectSummary
		var page *string
		for {
			resp, err := svc.ListProjects(context.Background(), aidocument.ListProjectsRequest{
				CompartmentId: common.String(conn.TenantID()), Page: page,
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
			p := items[i]
			r, err := CreateResource(o.MqlRuntime, "oci.ai.document.project", map[string]*llx.RawData{
				"id":               llx.StringDataPtr(p.Id),
				"name":             llx.StringDataPtr(p.DisplayName),
				"compartmentID":    llx.StringDataPtr(p.CompartmentId),
				"description":      llx.StringDataPtr(p.Description),
				"state":            llx.StringData(string(p.LifecycleState)),
				"lifecycleDetails": llx.StringDataPtr(p.LifecycleDetails),
				"created":          sdkTimeData(p.TimeCreated),
				"freeformTags":     llx.MapData(strMapToAny(p.FreeformTags), types.String),
				"definedTags":      llx.MapData(definedTagsToAny(p.DefinedTags), types.Any),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, r)
		}
		return res, nil
	})
}

func (o *mqlOciAiDocument) models() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	return ociListRegionalAIClient(o.MqlRuntime, conn.AIDocumentClient, func(svc *aidocument.AIServiceDocumentClient) ([]any, error) {
		var items []aidocument.ModelSummary
		var page *string
		for {
			resp, err := svc.ListModels(context.Background(), aidocument.ListModelsRequest{
				CompartmentId: common.String(conn.TenantID()), Page: page,
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
			var prec float64
			if m.Precision != nil {
				prec = float64(*m.Precision)
			}
			r, err := CreateResource(o.MqlRuntime, "oci.ai.document.model", map[string]*llx.RawData{
				"id":               llx.StringDataPtr(m.Id),
				"name":             llx.StringDataPtr(m.DisplayName),
				"compartmentID":    llx.StringDataPtr(m.CompartmentId),
				"projectID":        llx.StringDataPtr(m.ProjectId),
				"modelType":        llx.StringData(string(m.ModelType)),
				"modelVersion":     llx.StringDataPtr(m.ModelVersion),
				"description":      llx.StringDataPtr(m.Description),
				"inferenceUnits":   llx.IntData(intValue(m.InferenceUnits)),
				"precision":        llx.FloatData(prec),
				"isComposedModel":  llx.BoolData(boolValue(m.IsComposedModel)),
				"state":            llx.StringData(string(m.LifecycleState)),
				"lifecycleDetails": llx.StringDataPtr(m.LifecycleDetails),
				"created":          sdkTimeData(m.TimeCreated),
				"freeformTags":     llx.MapData(strMapToAny(m.FreeformTags), types.String),
				"definedTags":      llx.MapData(definedTagsToAny(m.DefinedTags), types.Any),
			})
			if err != nil {
				return nil, err
			}
			r.(*mqlOciAiDocumentModel).cacheProjectID = stringValue(m.ProjectId)
			res = append(res, r)
		}
		return res, nil
	})
}

type mqlOciAiDocumentModelInternal struct {
	cacheProjectID string
}

func initOciAiDocumentProject(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return findOciAIResourceByID(runtime, args, "oci.ai.document", func(o plugin.Resource) *plugin.TValue[[]any] { return o.(*mqlOciAiDocument).GetProjects() })
}

func initOciAiDocumentModel(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	return findOciAIResourceByID(runtime, args, "oci.ai.document", func(o plugin.Resource) *plugin.TValue[[]any] { return o.(*mqlOciAiDocument).GetModels() })
}

func (o *mqlOciAiDocumentProject) id() (string, error) {
	return "oci.ai.document.project/" + o.Id.Data, nil
}
func (o *mqlOciAiDocumentProject) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciAiDocumentModel) id() (string, error) {
	return "oci.ai.document.model/" + o.Id.Data, nil
}
func (o *mqlOciAiDocumentModel) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}
func (o *mqlOciAiDocumentModel) project() (*mqlOciAiDocumentProject, error) {
	if o.cacheProjectID == "" {
		o.Project.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.ai.document.project", map[string]*llx.RawData{"id": llx.StringData(o.cacheProjectID)})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciAiDocumentProject), nil
}
