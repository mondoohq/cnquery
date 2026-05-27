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

// mqlBicepVariableInternal carries the owning file's symbol resolver so the
// lazy expressionTree() accessor can resolve the root identifiers its nodes
// reference to same-file declarations.
type mqlBicepVariableInternal struct {
	resolver *symbolResolver
}

func createMqlVariables(runtime *plugin.Runtime, filePath string, vars []parsedVariable, resolver *symbolResolver) ([]any, error) {
	var mqlVars []any
	for _, v := range vars {
		args := map[string]*llx.RawData{
			"__id":        llx.StringData("bicep.variable:" + filePath + ":" + v.name),
			"name":        llx.StringData(v.name),
			"expression":  llx.StringData(v.expression),
			"description": llx.StringData(v.description),
		}
		addLoopArgs(args, v.loop)
		res, err := CreateResource(runtime, "bicep.variable", args)
		if err != nil {
			return nil, err
		}
		res.(*mqlBicepVariable).resolver = resolver
		mqlVars = append(mqlVars, res)
	}
	return mqlVars, nil
}

// mqlBicepResourceInternal caches the parsed nested child resources so the
// lazy `resources()` accessor can materialize them without re-parsing. Each
// child carries a parent-qualified `__id` (`<parentId>/<childSymbolicName>`)
// so nested resources under different parents never collide in the cache.
// The resolver carries the owning file's symbol table so the lazy
// name/location/condition expression-tree accessors can resolve referenced
// root identifiers to same-file declarations. Nested child resources inherit
// the same resolver since symbols stay file-scoped regardless of nesting.
type mqlBicepResourceInternal struct {
	nested   []parsedResource
	resolver *symbolResolver
}

func createMqlResources(runtime *plugin.Runtime, filePath string, resources []parsedResource, resolver *symbolResolver) ([]any, error) {
	var mqlResources []any
	for _, r := range resources {
		res, err := newMqlBicepResource(runtime, "bicep.resource:"+filePath+":"+r.symbolicName, r, resolver)
		if err != nil {
			return nil, err
		}
		mqlResources = append(mqlResources, res)
	}
	return mqlResources, nil
}

// newMqlBicepResource builds a single bicep.resource from a parsedResource.
// The id is supplied by the caller: top-level resources use
// `bicep.resource:<file>:<symbolicName>`, while nested resources use a
// parent-qualified `<parentId>/<childSymbolicName>`. The parsed nested
// declarations are cached on the Internal struct for the lazy `resources()`
// accessor to materialize, recursively.
func newMqlBicepResource(runtime *plugin.Runtime, id string, r parsedResource, resolver *symbolResolver) (*mqlBicepResource, error) {
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
		"__id":         llx.StringData(id),
		"symbolicName": llx.StringData(r.symbolicName),
		"type":         llx.StringData(r.typ),
		"apiVersion":   llx.StringData(r.apiVersion),
		"name":         llx.StringData(r.name),
		"location":     llx.StringData(r.location),
		"existing":     llx.BoolData(r.existing),
		"condition":    llx.StringData(r.condition),
		"parent":       llx.StringData(r.parent),
		"scope":        llx.StringData(r.scope),
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
	addLoopArgs(args, r.loop)

	res, err := CreateResource(runtime, "bicep.resource", args)
	if err != nil {
		return nil, err
	}
	mqlRes := res.(*mqlBicepResource)
	mqlRes.nested = r.nested
	mqlRes.resolver = resolver
	return mqlRes, nil
}

// resources materializes this resource's nested child resources. Each child
// gets a parent-qualified `__id` so nested resources under different parents
// don't collide; children may themselves declare further nested resources,
// resolved recursively by the same accessor.
func (r *mqlBicepResource) resources() ([]any, error) {
	out := make([]any, 0, len(r.nested))
	for _, child := range r.nested {
		childID := r.__id + "/" + child.symbolicName
		mqlChild, err := newMqlBicepResource(r.MqlRuntime, childID, child, r.resolver)
		if err != nil {
			return nil, err
		}
		out = append(out, mqlChild)
	}
	return out, nil
}

// mqlBicepModuleInternal carries the owning file's symbol resolver so the
// lazy scope/condition expression-tree accessors can resolve referenced root
// identifiers to same-file declarations.
type mqlBicepModuleInternal struct {
	resolver *symbolResolver
}

func createMqlModules(runtime *plugin.Runtime, filePath string, modules []parsedModule, resolver *symbolResolver) ([]any, error) {
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

		args := map[string]*llx.RawData{
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
		}
		addLoopArgs(args, m.loop)
		res, err := CreateResource(runtime, "bicep.module", args)
		if err != nil {
			return nil, err
		}
		res.(*mqlBicepModule).resolver = resolver
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
	// resolver is the owning file's symbol table, threaded down from the
	// declaration whose value this expression is. It lets referenceKind and
	// the referenced*() accessors resolve `target` (the root identifier of a
	// symbolicRef or propertyAccess) to the same-file declaration it names.
	// Child nodes (args, segments) inherit the same resolver so nested refs
	// resolve too.
	resolver *symbolResolver
}

// newMqlBicepExpression turns a parsed exprNode into a bicep.expression
// resource. Scalar fields are set immediately; child nodes (args, segments)
// are cached on the Internal struct and materialized lazily. The resolver is
// the owning file's symbol table, propagated to children so symbol resolution
// works at every depth.
func newMqlBicepExpression(runtime *plugin.Runtime, parentID string, node *exprNode, resolver *symbolResolver) (*mqlBicepExpression, error) {
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
	mqlExpr.resolver = resolver
	return mqlExpr, nil
}

func (e *mqlBicepExpression) args() ([]any, error) {
	if e.node == nil {
		return []any{}, nil
	}
	out := make([]any, 0, len(e.node.args))
	for i, child := range e.node.args {
		childID := e.__id + "/arg[" + strconv.Itoa(i) + "]"
		mqlChild, err := newMqlBicepExpression(e.MqlRuntime, childID, child, e.resolver)
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
		mqlChild, err := newMqlBicepExpression(e.MqlRuntime, childID, child, e.resolver)
		if err != nil {
			return nil, err
		}
		out = append(out, mqlChild)
	}
	return out, nil
}

// referenceKind reports which same-file declaration this node's `target` root
// identifier names — one of "parameter", "variable", "resource", "module", or
// "type" — or "" when `target` is empty or doesn't match a same-file
// declaration (a built-in like `resourceGroup`, or an imported symbol, which
// is resolved by a separate later layer). At most one referenced*() accessor
// is non-null, matching this kind.
func (e *mqlBicepExpression) referenceKind() (string, error) {
	target := ""
	if e.node != nil {
		target = e.node.target
	}
	return e.resolver.kind(target), nil
}

func (e *mqlBicepExpression) target_() string {
	if e.node != nil {
		return e.node.target
	}
	return e.Target.Data
}

func (e *mqlBicepExpression) referencedParameter() (*mqlBicepParameter, error) {
	p, err := e.resolver.parameter(e.MqlRuntime, e.target_())
	if err != nil {
		return nil, err
	}
	if p == nil {
		e.ReferencedParameter.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return p, nil
}

func (e *mqlBicepExpression) referencedVariable() (*mqlBicepVariable, error) {
	v, err := e.resolver.variable(e.MqlRuntime, e.target_())
	if err != nil {
		return nil, err
	}
	if v == nil {
		e.ReferencedVariable.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return v, nil
}

func (e *mqlBicepExpression) referencedResource() (*mqlBicepResource, error) {
	r, err := e.resolver.resource(e.MqlRuntime, e.target_())
	if err != nil {
		return nil, err
	}
	if r == nil {
		e.ReferencedResource.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return r, nil
}

func (e *mqlBicepExpression) referencedModule() (*mqlBicepModule, error) {
	m, err := e.resolver.module(e.MqlRuntime, e.target_())
	if err != nil {
		return nil, err
	}
	if m == nil {
		e.ReferencedModule.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return m, nil
}

func (e *mqlBicepExpression) referencedType() (*mqlBicepType, error) {
	t, err := e.resolver.typ(e.MqlRuntime, e.target_())
	if err != nil {
		return nil, err
	}
	if t == nil {
		e.ReferencedType.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return t, nil
}

// expressionTreeFor parses a raw Bicep expression string and returns it as a
// bicep.expression resource. The suffix is appended to the parent's __id so
// distinct expression fields under the same parent (e.g. nameTree vs
// locationTree on one resource) get distinct, non-colliding cache keys. An
// empty raw string parses to an `unknown` node with empty `raw`, mirroring the
// existing variable/output expressionTree() behavior.
func expressionTreeFor(runtime *plugin.Runtime, parentID, suffix, raw string, resolver *symbolResolver) (*mqlBicepExpression, error) {
	node := parseExpression(raw)
	return newMqlBicepExpression(runtime, parentID+suffix, node, resolver)
}

func (v *mqlBicepVariable) expressionTree() (*mqlBicepExpression, error) {
	return expressionTreeFor(v.MqlRuntime, v.__id, "/expr", v.Expression.Data, v.resolver)
}

func (o *mqlBicepOutput) expressionTree() (*mqlBicepExpression, error) {
	return expressionTreeFor(o.MqlRuntime, o.__id, "/expr", o.Expression.Data, o.resolver)
}

func (r *mqlBicepResource) nameTree() (*mqlBicepExpression, error) {
	return expressionTreeFor(r.MqlRuntime, r.__id, "/nameTree", r.Name.Data, r.resolver)
}

func (r *mqlBicepResource) locationTree() (*mqlBicepExpression, error) {
	return expressionTreeFor(r.MqlRuntime, r.__id, "/locationTree", r.Location.Data, r.resolver)
}

func (r *mqlBicepResource) conditionTree() (*mqlBicepExpression, error) {
	return expressionTreeFor(r.MqlRuntime, r.__id, "/conditionTree", r.Condition.Data, r.resolver)
}

func (m *mqlBicepModule) scopeTree() (*mqlBicepExpression, error) {
	return expressionTreeFor(m.MqlRuntime, m.__id, "/scopeTree", m.Scope.Data, m.resolver)
}

func (m *mqlBicepModule) conditionTree() (*mqlBicepExpression, error) {
	return expressionTreeFor(m.MqlRuntime, m.__id, "/conditionTree", m.Condition.Data, m.resolver)
}

// mqlBicepOutputInternal carries the owning file's symbol resolver so the
// lazy expressionTree() accessor can resolve referenced root identifiers to
// same-file declarations.
type mqlBicepOutputInternal struct {
	resolver *symbolResolver
}

func createMqlOutputs(runtime *plugin.Runtime, filePath string, outputs []parsedOutput, resolver *symbolResolver) ([]any, error) {
	var mqlOutputs []any
	for _, o := range outputs {
		args := map[string]*llx.RawData{
			"__id":        llx.StringData("bicep.output:" + filePath + ":" + o.name),
			"name":        llx.StringData(o.name),
			"type":        llx.StringData(o.typ),
			"expression":  llx.StringData(o.expression),
			"description": llx.StringData(o.description),
		}
		addLoopArgs(args, o.loop)
		res, err := CreateResource(runtime, "bicep.output", args)
		if err != nil {
			return nil, err
		}
		res.(*mqlBicepOutput).resolver = resolver
		mqlOutputs = append(mqlOutputs, res)
	}
	return mqlOutputs, nil
}

func createMqlTypes(runtime *plugin.Runtime, filePath string, types_ []parsedType) ([]any, error) {
	var mqlTypes []any
	for _, t := range types_ {
		decorators := sliceToAny(t.decorators)
		res, err := CreateResource(runtime, "bicep.type", map[string]*llx.RawData{
			"__id":        llx.StringData("bicep.type:" + filePath + ":" + t.name),
			"name":        llx.StringData(t.name),
			"definition":  llx.StringData(t.definition),
			"description": llx.StringData(t.description),
			"exported":    llx.BoolData(t.exported),
			"decorators":  llx.ArrayData(decorators, types.String),
		})
		if err != nil {
			return nil, err
		}
		mqlTypes = append(mqlTypes, res)
	}
	return mqlTypes, nil
}

func createMqlFunctions(runtime *plugin.Runtime, filePath string, functions []parsedFunction) ([]any, error) {
	var mqlFunctions []any
	for _, fn := range functions {
		decorators := sliceToAny(fn.decorators)

		params := make(map[string]any, len(fn.parameters))
		for k, v := range fn.parameters {
			params[k] = v
		}

		res, err := CreateResource(runtime, "bicep.function", map[string]*llx.RawData{
			"__id":        llx.StringData("bicep.function:" + filePath + ":" + fn.name),
			"name":        llx.StringData(fn.name),
			"parameters":  llx.MapData(params, types.String),
			"returnType":  llx.StringData(fn.returnType),
			"expression":  llx.StringData(fn.expression),
			"description": llx.StringData(fn.description),
			"decorators":  llx.ArrayData(decorators, types.String),
		})
		if err != nil {
			return nil, err
		}
		mqlFunctions = append(mqlFunctions, res)
	}
	return mqlFunctions, nil
}

func createMqlImports(runtime *plugin.Runtime, filePath string, imports []parsedImport) ([]any, error) {
	var mqlImports []any
	for i, imp := range imports {
		symbols := sliceToAny(imp.symbols)
		// A file can import the same source twice (e.g. a named import and a
		// wildcard import of './shared.bicep'), so the index disambiguates
		// the synthetic id.
		res, err := CreateResource(runtime, "bicep.import", map[string]*llx.RawData{
			"__id":      llx.StringData("bicep.import:" + filePath + ":" + strconv.Itoa(i) + ":" + imp.source),
			"source":    llx.StringData(imp.source),
			"symbols":   llx.ArrayData(symbols, types.String),
			"namespace": llx.StringData(imp.namespace),
			"wildcard":  llx.BoolData(imp.wildcard),
		})
		if err != nil {
			return nil, err
		}
		mqlImports = append(mqlImports, res)
	}
	return mqlImports, nil
}

// addLoopArgs sets the four flattened `for`-loop fields shared by
// bicep.resource, bicep.module, bicep.output, and bicep.variable. For a
// non-loop declaration isLoop is false and the string fields are empty.
func addLoopArgs(args map[string]*llx.RawData, loop loopInfo) {
	args["isLoop"] = llx.BoolData(loop.isLoop)
	args["loopIterator"] = llx.StringData(loop.iterator)
	args["loopIndexVar"] = llx.StringData(loop.indexVar)
	args["loopExpression"] = llx.StringData(loop.expression)
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
	_ plugin.Resource = (*mqlBicepType)(nil)
	_ plugin.Resource = (*mqlBicepFunction)(nil)
	_ plugin.Resource = (*mqlBicepImport)(nil)
	_ plugin.Resource = (*mqlBicepParamFile)(nil)
)
