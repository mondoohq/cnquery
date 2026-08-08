// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/generativeaiagent"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciAi) id() (string, error) {
	return "oci.ai", nil
}

func (o *mqlOciAiAgents) id() (string, error) {
	return "oci.ai.agents", nil
}

// ----- agents -----

func (o *mqlOciAiAgents) agents() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeTenancyRoot,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			log.Debug().Msgf("calling oci generative ai agents with region %s", region)

			svc, err := conn.GenerativeAiAgentClient(region)
			if err != nil {
				return nil, err
			}

			items, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]generativeaiagent.AgentSummary, *string, error) {
				response, err := svc.ListAgents(ctx, generativeaiagent.ListAgentsRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, nil, err
				}
				return response.Items, response.OpcNextPage, nil
			})
			if err != nil {
				if ociRegionServiceUnavailable(err) {
					log.Debug().Str("region", region).Msg("generative ai agents not available in region, skipping")
					return []any{}, nil
				}
				return nil, err
			}

			res := make([]any, 0, len(items))
			for i := range items {
				a := items[i]

				llmConfig, err := convert.JsonToDict(a.LlmConfig)
				if err != nil {
					return nil, err
				}

				mqlAgent, err := CreateResource(o.MqlRuntime, "oci.ai.agents.agent", map[string]*llx.RawData{
					"id":               llx.StringDataPtr(a.Id),
					"name":             llx.StringDataPtr(a.DisplayName),
					"compartmentID":    llx.StringDataPtr(a.CompartmentId),
					"description":      llx.StringDataPtr(a.Description),
					"welcomeMessage":   llx.StringDataPtr(a.WelcomeMessage),
					"llmConfig":        llx.DictData(llmConfig),
					"state":            llx.StringData(string(a.LifecycleState)),
					"lifecycleDetails": llx.StringDataPtr(a.LifecycleDetails),
					"created":          sdkTimeData(a.TimeCreated),
					"timeUpdated":      sdkTimeData(a.TimeUpdated),
					"freeformTags":     llx.MapData(strMapToAny(a.FreeformTags), types.String),
					"definedTags":      llx.MapData(definedTagsToAny(a.DefinedTags), types.Any),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlAgent)
			}
			return res, nil
		})
}

func initOciAiAgentsAgent(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idVal := ociArgString(args, "id")
	if idVal == "" {
		return args, nil, nil
	}
	obj, err := CreateResource(runtime, "oci.ai.agents", nil)
	if err != nil {
		return nil, nil, err
	}
	list := obj.(*mqlOciAiAgents).GetAgents()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		a := raw.(*mqlOciAiAgentsAgent)
		if a.Id.Data == idVal {
			return args, a, nil
		}
	}
	return nil, nil, errors.New("oci.ai.agents.agent not found: " + idVal)
}

func (o *mqlOciAiAgentsAgent) id() (string, error) {
	return "oci.ai.agents.agent/" + o.Id.Data, nil
}

func (o *mqlOciAiAgentsAgent) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

// ----- agent endpoints -----

func (o *mqlOciAiAgents) agentEndpoints() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeTenancyRoot,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			svc, err := conn.GenerativeAiAgentClient(region)
			if err != nil {
				return nil, err
			}

			items, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]generativeaiagent.AgentEndpointSummary, *string, error) {
				response, err := svc.ListAgentEndpoints(ctx, generativeaiagent.ListAgentEndpointsRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, nil, err
				}
				return response.Items, response.OpcNextPage, nil
			})
			if err != nil {
				if ociRegionServiceUnavailable(err) {
					return []any{}, nil
				}
				return nil, err
			}

			res := make([]any, 0, len(items))
			for i := range items {
				e := items[i]

				var modInput, modOutput bool
				if e.ContentModerationConfig != nil {
					modInput = boolValue(e.ContentModerationConfig.ShouldEnableOnInput)
					modOutput = boolValue(e.ContentModerationConfig.ShouldEnableOnOutput)
				}
				guardrail, err := convert.JsonToDict(e.GuardrailConfig)
				if err != nil {
					return nil, err
				}
				var idleTimeout int64
				if e.SessionConfig != nil {
					idleTimeout = intValue(e.SessionConfig.IdleTimeoutInSeconds)
				}

				mqlEndpoint, err := CreateResource(o.MqlRuntime, "oci.ai.agents.endpoint", map[string]*llx.RawData{
					"id":                          llx.StringDataPtr(e.Id),
					"name":                        llx.StringDataPtr(e.DisplayName),
					"compartmentID":               llx.StringDataPtr(e.CompartmentId),
					"agentID":                     llx.StringDataPtr(e.AgentId),
					"description":                 llx.StringDataPtr(e.Description),
					"contentModerationOnInput":    llx.BoolData(modInput),
					"contentModerationOnOutput":   llx.BoolData(modOutput),
					"guardrailConfig":             llx.DictData(guardrail),
					"sessionIdleTimeoutInSeconds": llx.IntData(idleTimeout),
					"sessionEnabled":              llx.BoolDataPtr(e.ShouldEnableSession),
					"traceEnabled":                llx.BoolDataPtr(e.ShouldEnableTrace),
					"citationEnabled":             llx.BoolDataPtr(e.ShouldEnableCitation),
					"multiLanguageEnabled":        llx.BoolDataPtr(e.ShouldEnableMultiLanguage),
					"metadata":                    llx.MapData(strMapToAny(e.Metadata), types.String),
					"state":                       llx.StringData(string(e.LifecycleState)),
					"lifecycleDetails":            llx.StringDataPtr(e.LifecycleDetails),
					"created":                     sdkTimeData(e.TimeCreated),
					"timeUpdated":                 sdkTimeData(e.TimeUpdated),
					"freeformTags":                llx.MapData(strMapToAny(e.FreeformTags), types.String),
					"definedTags":                 llx.MapData(definedTagsToAny(e.DefinedTags), types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlEndpointTyped := mqlEndpoint.(*mqlOciAiAgentsEndpoint)
				mqlEndpointTyped.cacheAgentID = stringValue(e.AgentId)
				res = append(res, mqlEndpointTyped)
			}
			return res, nil
		})
}

type mqlOciAiAgentsEndpointInternal struct {
	cacheAgentID string
}

func initOciAiAgentsEndpoint(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idVal := ociArgString(args, "id")
	if idVal == "" {
		return args, nil, nil
	}
	obj, err := CreateResource(runtime, "oci.ai.agents", nil)
	if err != nil {
		return nil, nil, err
	}
	list := obj.(*mqlOciAiAgents).GetAgentEndpoints()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		e := raw.(*mqlOciAiAgentsEndpoint)
		if e.Id.Data == idVal {
			return args, e, nil
		}
	}
	return nil, nil, errors.New("oci.ai.agents.endpoint not found: " + idVal)
}

func (o *mqlOciAiAgentsEndpoint) id() (string, error) {
	return "oci.ai.agents.endpoint/" + o.Id.Data, nil
}

func (o *mqlOciAiAgentsEndpoint) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciAiAgentsEndpoint) agent() (*mqlOciAiAgentsAgent, error) {
	return resolveOciAiAgent(o.MqlRuntime, o.cacheAgentID, &o.Agent)
}

// ----- knowledge bases -----

func (o *mqlOciAiAgents) knowledgeBases() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeTenancyRoot,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			svc, err := conn.GenerativeAiAgentClient(region)
			if err != nil {
				return nil, err
			}

			items, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]generativeaiagent.KnowledgeBaseSummary, *string, error) {
				response, err := svc.ListKnowledgeBases(ctx, generativeaiagent.ListKnowledgeBasesRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, nil, err
				}
				return response.Items, response.OpcNextPage, nil
			})
			if err != nil {
				if ociRegionServiceUnavailable(err) {
					return []any{}, nil
				}
				return nil, err
			}

			res := make([]any, 0, len(items))
			for i := range items {
				kb := items[i]

				mqlKb, err := CreateResource(o.MqlRuntime, "oci.ai.agents.knowledgeBase", map[string]*llx.RawData{
					"id":               llx.StringDataPtr(kb.Id),
					"name":             llx.StringDataPtr(kb.DisplayName),
					"compartmentID":    llx.StringDataPtr(kb.CompartmentId),
					"description":      llx.StringDataPtr(kb.Description),
					"state":            llx.StringData(string(kb.LifecycleState)),
					"lifecycleDetails": llx.StringDataPtr(kb.LifecycleDetails),
					"created":          sdkTimeData(kb.TimeCreated),
					"timeUpdated":      sdkTimeData(kb.TimeUpdated),
					"freeformTags":     llx.MapData(strMapToAny(kb.FreeformTags), types.String),
					"definedTags":      llx.MapData(definedTagsToAny(kb.DefinedTags), types.Any),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, mqlKb)
			}
			return res, nil
		})
}

func initOciAiAgentsKnowledgeBase(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idVal := ociArgString(args, "id")
	if idVal == "" {
		return args, nil, nil
	}
	obj, err := CreateResource(runtime, "oci.ai.agents", nil)
	if err != nil {
		return nil, nil, err
	}
	list := obj.(*mqlOciAiAgents).GetKnowledgeBases()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		kb := raw.(*mqlOciAiAgentsKnowledgeBase)
		if kb.Id.Data == idVal {
			return args, kb, nil
		}
	}
	return nil, nil, errors.New("oci.ai.agents.knowledgeBase not found: " + idVal)
}

func (o *mqlOciAiAgentsKnowledgeBase) id() (string, error) {
	return "oci.ai.agents.knowledgeBase/" + o.Id.Data, nil
}

func (o *mqlOciAiAgentsKnowledgeBase) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

// ----- data sources -----

func (o *mqlOciAiAgents) dataSources() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeTenancyRoot,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			svc, err := conn.GenerativeAiAgentClient(region)
			if err != nil {
				return nil, err
			}

			items, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]generativeaiagent.DataSourceSummary, *string, error) {
				response, err := svc.ListDataSources(ctx, generativeaiagent.ListDataSourcesRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, nil, err
				}
				return response.Items, response.OpcNextPage, nil
			})
			if err != nil {
				if ociRegionServiceUnavailable(err) {
					return []any{}, nil
				}
				return nil, err
			}

			res := make([]any, 0, len(items))
			for i := range items {
				ds := items[i]

				mqlDs, err := CreateResource(o.MqlRuntime, "oci.ai.agents.dataSource", map[string]*llx.RawData{
					"id":               llx.StringDataPtr(ds.Id),
					"name":             llx.StringDataPtr(ds.DisplayName),
					"compartmentID":    llx.StringDataPtr(ds.CompartmentId),
					"knowledgeBaseID":  llx.StringDataPtr(ds.KnowledgeBaseId),
					"description":      llx.StringDataPtr(ds.Description),
					"state":            llx.StringData(string(ds.LifecycleState)),
					"lifecycleDetails": llx.StringDataPtr(ds.LifecycleDetails),
					"created":          sdkTimeData(ds.TimeCreated),
					"timeUpdated":      sdkTimeData(ds.TimeUpdated),
					"freeformTags":     llx.MapData(strMapToAny(ds.FreeformTags), types.String),
					"definedTags":      llx.MapData(definedTagsToAny(ds.DefinedTags), types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlDsTyped := mqlDs.(*mqlOciAiAgentsDataSource)
				mqlDsTyped.cacheKnowledgeBaseID = stringValue(ds.KnowledgeBaseId)
				res = append(res, mqlDsTyped)
			}
			return res, nil
		})
}

type mqlOciAiAgentsDataSourceInternal struct {
	cacheKnowledgeBaseID string
}

func initOciAiAgentsDataSource(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idVal := ociArgString(args, "id")
	if idVal == "" {
		return args, nil, nil
	}
	obj, err := CreateResource(runtime, "oci.ai.agents", nil)
	if err != nil {
		return nil, nil, err
	}
	list := obj.(*mqlOciAiAgents).GetDataSources()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		ds := raw.(*mqlOciAiAgentsDataSource)
		if ds.Id.Data == idVal {
			return args, ds, nil
		}
	}
	return nil, nil, errors.New("oci.ai.agents.dataSource not found: " + idVal)
}

func (o *mqlOciAiAgentsDataSource) id() (string, error) {
	return "oci.ai.agents.dataSource/" + o.Id.Data, nil
}

func (o *mqlOciAiAgentsDataSource) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciAiAgentsDataSource) knowledgeBase() (*mqlOciAiAgentsKnowledgeBase, error) {
	if o.cacheKnowledgeBaseID == "" {
		o.KnowledgeBase.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(o.MqlRuntime, "oci.ai.agents.knowledgeBase", map[string]*llx.RawData{
		"id": llx.StringData(o.cacheKnowledgeBaseID),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciAiAgentsKnowledgeBase), nil
}

// ----- tools -----

func (o *mqlOciAiAgents) tools() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	return ociCollect(o.MqlRuntime, ociScopeTenancyRoot,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			svc, err := conn.GenerativeAiAgentClient(region)
			if err != nil {
				return nil, err
			}

			items, err := ociPaginate(ctx, func(ctx context.Context, page *string) ([]generativeaiagent.ToolSummary, *string, error) {
				response, err := svc.ListTools(ctx, generativeaiagent.ListToolsRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, nil, err
				}
				return response.Items, response.OpcNextPage, nil
			})
			if err != nil {
				if ociRegionServiceUnavailable(err) {
					return []any{}, nil
				}
				return nil, err
			}

			res := make([]any, 0, len(items))
			for i := range items {
				t := items[i]

				toolConfig, err := convert.JsonToDict(t.ToolConfig)
				if err != nil {
					return nil, err
				}

				mqlTool, err := CreateResource(o.MqlRuntime, "oci.ai.agents.tool", map[string]*llx.RawData{
					"id":            llx.StringDataPtr(t.Id),
					"name":          llx.StringDataPtr(t.DisplayName),
					"compartmentID": llx.StringDataPtr(t.CompartmentId),
					"agentID":       llx.StringDataPtr(t.AgentId),
					"description":   llx.StringDataPtr(t.Description),
					"toolType":      llx.StringData(ociToolConfigType(toolConfig)),
					"toolConfig":    llx.DictData(toolConfig),
					"metadata":      llx.MapData(strMapToAny(t.Metadata), types.String),
					"state":         llx.StringData(string(t.LifecycleState)),
					"created":       sdkTimeData(t.TimeCreated),
					"timeUpdated":   sdkTimeData(t.TimeUpdated),
					"freeformTags":  llx.MapData(strMapToAny(t.FreeformTags), types.String),
					"definedTags":   llx.MapData(definedTagsToAny(t.DefinedTags), types.Any),
				})
				if err != nil {
					return nil, err
				}
				mqlToolTyped := mqlTool.(*mqlOciAiAgentsTool)
				mqlToolTyped.cacheAgentID = stringValue(t.AgentId)
				res = append(res, mqlToolTyped)
			}
			return res, nil
		})
}

type mqlOciAiAgentsToolInternal struct {
	cacheAgentID string
}

func initOciAiAgentsTool(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	idVal := ociArgString(args, "id")
	if idVal == "" {
		return args, nil, nil
	}
	obj, err := CreateResource(runtime, "oci.ai.agents", nil)
	if err != nil {
		return nil, nil, err
	}
	list := obj.(*mqlOciAiAgents).GetTools()
	if list.Error != nil {
		return nil, nil, list.Error
	}
	for _, raw := range list.Data {
		t := raw.(*mqlOciAiAgentsTool)
		if t.Id.Data == idVal {
			return args, t, nil
		}
	}
	return nil, nil, errors.New("oci.ai.agents.tool not found: " + idVal)
}

func (o *mqlOciAiAgentsTool) id() (string, error) {
	return "oci.ai.agents.tool/" + o.Id.Data, nil
}

func (o *mqlOciAiAgentsTool) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciAiAgentsTool) agent() (*mqlOciAiAgentsAgent, error) {
	return resolveOciAiAgent(o.MqlRuntime, o.cacheAgentID, &o.Agent)
}

// ----- helpers -----

// resolveOciAiAgent resolves a typed agent resource from an agent OCID. Returns
// (nil, nil) and marks the field as null when the OCID is empty.
func resolveOciAiAgent(runtime *plugin.Runtime, id string, field *plugin.TValue[*mqlOciAiAgentsAgent]) (*mqlOciAiAgentsAgent, error) {
	if id == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	r, err := NewResource(runtime, "oci.ai.agents.agent", map[string]*llx.RawData{
		"id": llx.StringData(id),
	})
	if err != nil {
		return nil, err
	}
	return r.(*mqlOciAiAgentsAgent), nil
}

// ociToolConfigType extracts the polymorphic tool-config discriminator
// (toolConfigType) from a tool configuration dict.
func ociToolConfigType(toolConfig any) string {
	m, ok := toolConfig.(map[string]any)
	if !ok {
		return ""
	}
	t, _ := m["toolConfigType"].(string)
	return t
}
