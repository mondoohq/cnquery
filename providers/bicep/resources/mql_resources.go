// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
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
			"minLength":    llx.IntDataPtr(p.minLength),
			"maxLength":    llx.IntDataPtr(p.maxLength),
			"minValue":     llx.IntDataPtr(p.minValue),
			"maxValue":     llx.IntDataPtr(p.maxValue),
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

		// Surface the resource's `properties: { ... }` sub-block as a
		// structured dict so audits can query individual keys directly
		// (e.g., `bicepResource.properties["accessTier"]`). When the
		// resource has no `properties` block the field is an empty
		// map rather than nil so the shape stays consistent across
		// resources.
		var properties any = map[string]any{}
		if r.body != "" {
			if raw := extractFieldBlock(r.body, "properties"); raw != "" {
				properties = parseBicepObject(raw)
			}
		}

		args := map[string]*llx.RawData{
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
		}
		if r.tags == nil {
			args["tags"] = llx.NilData
		} else {
			tags := make(map[string]any, len(r.tags))
			for k, v := range r.tags {
				tags[k] = v
			}
			args["tags"] = llx.MapData(tags, types.String)
		}

		res, err := CreateResource(runtime, "bicep.resource", args)
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
		// Same shape as bicep.resource.properties: parse the module's
		// `params: { ... }` block as a structured dict so audits can
		// pluck individual parameter values.
		var params any = map[string]any{}
		if m.body != "" {
			if raw := extractFieldBlock(m.body, "params"); raw != "" {
				params = parseBicepObject(raw)
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

// mqlBicepExpressionInternal caches the parsed child nodes of an expression
// so the lazy `args()` and `segments()` accessors can materialize their
// child resources without re-parsing. The synthetic `__id` of each child is
// derived from its parent's id (`<exprId>/arg[<i>]`, `<exprId>/seg[<i>]`).
type mqlBicepExpressionInternal struct {
	node *exprNode
}

// newMqlBicepExpression turns a parsed exprNode into a bicep.expression
// resource. Scalar fields are set immediately; child nodes (args, segments)
// are cached on the Internal struct and materialized lazily.
func newMqlBicepExpression(runtime *plugin.Runtime, parentID string, node *exprNode) (*mqlBicepExpression, error) {
	res, err := CreateResource(runtime, "bicep.expression", map[string]*llx.RawData{
		"__id":         llx.StringData(parentID),
		"kind":         llx.StringData(node.kind),
		"raw":          llx.StringData(node.raw),
		"functionName": llx.StringData(node.functionName),
		"target":       llx.StringData(node.target),
		"path":         llx.ArrayData(sliceToAny(node.path), types.String),
	})
	if err != nil {
		return nil, err
	}
	mqlExpr := res.(*mqlBicepExpression)
	mqlExpr.node = node
	return mqlExpr, nil
}

func (e *mqlBicepExpression) args() ([]any, error) {
	if e.node == nil {
		return []any{}, nil
	}
	out := make([]any, 0, len(e.node.args))
	for i, child := range e.node.args {
		childID := e.__id + "/arg[" + strconv.Itoa(i) + "]"
		mqlChild, err := newMqlBicepExpression(e.MqlRuntime, childID, child)
		if err != nil {
			return nil, err
		}
		out = append(out, mqlChild)
	}
	return out, nil
}

func (e *mqlBicepExpression) segments() ([]any, error) {
	if e.node == nil {
		return []any{}, nil
	}
	out := make([]any, 0, len(e.node.segments))
	for i, child := range e.node.segments {
		childID := e.__id + "/seg[" + strconv.Itoa(i) + "]"
		mqlChild, err := newMqlBicepExpression(e.MqlRuntime, childID, child)
		if err != nil {
			return nil, err
		}
		out = append(out, mqlChild)
	}
	return out, nil
}

func (v *mqlBicepVariable) expressionTree() (*mqlBicepExpression, error) {
	node := parseExpression(v.Expression.Data)
	return newMqlBicepExpression(v.MqlRuntime, v.__id+"/expr", node)
}

func (o *mqlBicepOutput) expressionTree() (*mqlBicepExpression, error) {
	node := parseExpression(o.Expression.Data)
	return newMqlBicepExpression(o.MqlRuntime, o.__id+"/expr", node)
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
	_ plugin.Resource = (*mqlBicepExpression)(nil)
)
