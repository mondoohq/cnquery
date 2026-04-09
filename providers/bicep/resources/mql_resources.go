// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/types"
)

func createMqlParameters(runtime *plugin.Runtime, filePath string, params []parsedParameter) ([]any, error) {
	var mqlParams []any
	for _, p := range params {
		allowed := sliceToAny(p.allowed)
		decorators := sliceToAny(p.decorators)

		res, err := CreateResource(runtime, "bicep.parameter", map[string]*llx.RawData{
			"__id":         llx.StringData("bicep.parameter:" + filePath + ":" + p.name),
			"name":         llx.StringData(p.name),
			"type":         llx.StringData(p.typ),
			"defaultValue": llx.StringData(p.defaultValue),
			"description":  llx.StringData(p.description),
			"secure":       llx.BoolData(p.secure),
			"allowed":      llx.ArrayData(allowed, types.String),
			"decorators":   llx.ArrayData(decorators, types.String),
		})
		if err != nil {
			return nil, err
		}
		mqlParams = append(mqlParams, res)
	}
	return mqlParams, nil
}

func createMqlVariables(runtime *plugin.Runtime, filePath string, vars []parsedVariable) ([]any, error) {
	var mqlVars []any
	for _, v := range vars {
		res, err := CreateResource(runtime, "bicep.variable", map[string]*llx.RawData{
			"__id":        llx.StringData("bicep.variable:" + filePath + ":" + v.name),
			"name":        llx.StringData(v.name),
			"expression":  llx.StringData(v.expression),
			"description": llx.StringData(v.description),
		})
		if err != nil {
			return nil, err
		}
		mqlVars = append(mqlVars, res)
	}
	return mqlVars, nil
}

func createMqlResources(runtime *plugin.Runtime, filePath string, resources []parsedResource) ([]any, error) {
	var mqlResources []any
	for _, r := range resources {
		dependsOn := sliceToAny(r.dependsOn)
		decorators := sliceToAny(r.decorators)

		// Parse body into a dict for properties
		var properties any
		if r.body != "" {
			dict, err := convert.JsonToDict(map[string]any{"raw": r.body})
			if err == nil {
				properties = dict
			}
		}

		res, err := CreateResource(runtime, "bicep.resource", map[string]*llx.RawData{
			"__id":         llx.StringData("bicep.resource:" + filePath + ":" + r.symbolicName),
			"symbolicName": llx.StringData(r.symbolicName),
			"type":         llx.StringData(r.typ),
			"apiVersion":   llx.StringData(r.apiVersion),
			"name":         llx.StringData(r.name),
			"location":     llx.StringData(r.location),
			"existing":     llx.BoolData(r.existing),
			"condition":    llx.StringData(r.condition),
			"parent":       llx.StringData(r.parent),
			"properties":   llx.DictData(properties),
			"dependsOn":    llx.ArrayData(dependsOn, types.String),
			"decorators":   llx.ArrayData(decorators, types.String),
		})
		if err != nil {
			return nil, err
		}
		mqlResources = append(mqlResources, res)
	}
	return mqlResources, nil
}

func createMqlModules(runtime *plugin.Runtime, filePath string, modules []parsedModule) ([]any, error) {
	var mqlModules []any
	for _, m := range modules {
		// Extract params block from body as a raw dict
		var params any
		if m.body != "" {
			if raw := extractFieldBlock(m.body, "params"); raw != "" {
				dict, err := convert.JsonToDict(map[string]any{"raw": raw})
				if err == nil {
					params = dict
				}
			}
		}

		decorators := sliceToAny(m.decorators)

		res, err := CreateResource(runtime, "bicep.module", map[string]*llx.RawData{
			"__id":           llx.StringData("bicep.module:" + filePath + ":" + m.name),
			"name":           llx.StringData(m.name),
			"source":         llx.StringData(m.source),
			"scope":          llx.StringData(m.scope),
			"params":         llx.DictData(params),
			"condition":      llx.StringData(m.condition),
			"isRegistry":     llx.BoolData(m.isRegistry),
			"isTemplateSpec": llx.BoolData(m.isTemplateSpec),
			"description":    llx.StringData(m.description),
			"decorators":     llx.ArrayData(decorators, types.String),
		})
		if err != nil {
			return nil, err
		}
		mqlModules = append(mqlModules, res)
	}
	return mqlModules, nil
}

func createMqlOutputs(runtime *plugin.Runtime, filePath string, outputs []parsedOutput) ([]any, error) {
	var mqlOutputs []any
	for _, o := range outputs {
		res, err := CreateResource(runtime, "bicep.output", map[string]*llx.RawData{
			"__id":        llx.StringData("bicep.output:" + filePath + ":" + o.name),
			"name":        llx.StringData(o.name),
			"type":        llx.StringData(o.typ),
			"expression":  llx.StringData(o.expression),
			"description": llx.StringData(o.description),
		})
		if err != nil {
			return nil, err
		}
		mqlOutputs = append(mqlOutputs, res)
	}
	return mqlOutputs, nil
}

func sliceToAny(s []string) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}

// Ensure all leaf resources satisfy the plugin interface.
var (
	_ plugin.Resource = (*mqlBicepParameter)(nil)
	_ plugin.Resource = (*mqlBicepVariable)(nil)
	_ plugin.Resource = (*mqlBicepResource)(nil)
	_ plugin.Resource = (*mqlBicepModule)(nil)
	_ plugin.Resource = (*mqlBicepOutput)(nil)
)
