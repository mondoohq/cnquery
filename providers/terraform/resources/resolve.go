// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/terraform/connection"
)

// terraformFunctions is the function set used to evaluate effective values. It
// is intentionally separate from hclFunctions (which backs arguments/config)
// so expanding it never changes the shape of arguments or config. It covers
// the Terraform built-ins that map onto cty's stdlib; expressions using
// functions outside this set fall back to the reference form.
func terraformFunctions() map[string]function.Function {
	return map[string]function.Function{
		"jsonencode": stdlib.JSONEncodeFunc,
		"jsondecode": stdlib.JSONDecodeFunc,
		"csvdecode":  stdlib.CSVDecodeFunc,
		"merge":      stdlib.MergeFunc,
		"concat":     stdlib.ConcatFunc,
		"flatten":    stdlib.FlattenFunc,
		"distinct":   stdlib.DistinctFunc,
		"sort":       stdlib.SortFunc,
		"reverse":    stdlib.ReverseFunc,
		"keys":       stdlib.KeysFunc,
		"values":     stdlib.ValuesFunc,
		"lookup":     stdlib.LookupFunc,
		"element":    stdlib.ElementFunc,
		"contains":   stdlib.ContainsFunc,
		"length":     stdlib.LengthFunc,
		"coalesce":   stdlib.CoalesceFunc,
		"format":     stdlib.FormatFunc,
		"formatlist": stdlib.FormatListFunc,
		"join":       stdlib.JoinFunc,
		"split":      stdlib.SplitFunc,
		"replace":    stdlib.ReplaceFunc,
		"lower":      stdlib.LowerFunc,
		"upper":      stdlib.UpperFunc,
		"title":      stdlib.TitleFunc,
		"trimspace":  stdlib.TrimSpaceFunc,
		"regex":      stdlib.RegexFunc,
		"regexall":   stdlib.RegexAllFunc,
		"abs":        stdlib.AbsoluteFunc,
		"max":        stdlib.MaxFunc,
		"min":        stdlib.MinFunc,
	}
}

// evalContext lazily builds and caches the HCL evaluation context: var
// defaults overridden by tfvars, resolved locals, and the function set.
// evalContextFor returns the cached HCL evaluation context for a module
// (nil = root). Module contexts use that module's own variable defaults,
// overridden by the call's inputs resolved in the root context, plus the
// module's locals — so a resource inside a module resolves to the values the
// call actually passes, not whatever a same-named root variable holds.
func (t *mqlTerraform) evalContextFor(moduleCall *mqlTerraformBlock, allBlocks []any, entries []fileModuleEntry) *hcl.EvalContext {
	key := ""
	if moduleCall != nil {
		key, _ = moduleCall.id()
	}

	t.evalLock.Lock()
	defer t.evalLock.Unlock()
	if t.moduleEvalCtx == nil {
		t.moduleEvalCtx = map[string]*hcl.EvalContext{}
	}
	if ctx, ok := t.moduleEvalCtx[key]; ok {
		return ctx
	}

	// Root must exist first; module input expressions resolve against it.
	root, ok := t.moduleEvalCtx[""]
	if !ok {
		root = t.buildContext(nil, allBlocks, entries, nil)
		t.moduleEvalCtx[""] = root
	}
	if moduleCall == nil {
		return root
	}

	overrides := t.moduleInputOverrides(moduleCall, root)
	ctx := t.buildContext(moduleCall, allBlocks, entries, overrides)
	t.moduleEvalCtx[key] = ctx
	return ctx
}

// buildContext assembles an evaluation context from the variable defaults and
// locals that belong to one module (nil = root), with tfvars (root) or call
// inputs (module) applied as overrides. It does not lock.
func (t *mqlTerraform) buildContext(moduleCall *mqlTerraformBlock, allBlocks []any, entries []fileModuleEntry, overrides map[string]cty.Value) *hcl.EvalContext {
	funcs := terraformFunctions()
	funcOnly := &hcl.EvalContext{Functions: funcs}

	moduleKey := ""
	if moduleCall != nil {
		moduleKey, _ = moduleCall.id()
	}

	varVals := map[string]cty.Value{}
	for i := range allBlocks {
		b := allBlocks[i].(*mqlTerraformBlock)
		if b.Type.Data != "variable" || blockModuleKey(b, entries) != moduleKey {
			continue
		}
		name := labelAt(b, 0)
		varVals[name] = cty.UnknownVal(cty.DynamicPseudoType)
		hb := blockHcl(b)
		if hb == nil {
			continue
		}
		attrs, _ := hb.Body.JustAttributes()
		if def, ok := attrs["default"]; ok {
			if val, diags := def.Expr.Value(funcOnly); !diags.HasErrors() {
				varVals[name] = val
			}
		}
	}

	if moduleCall == nil {
		if conn, ok := t.MqlRuntime.Connection.(*connection.Connection); ok {
			for name, attr := range conn.TfVars() {
				if val, diags := attr.Expr.Value(funcOnly); !diags.HasErrors() {
					varVals[name] = val
				}
			}
		}
	}
	for k, v := range overrides {
		varVals[k] = v
	}

	type localEntry struct {
		name string
		expr hcl.Expression
	}
	var locals []localEntry
	for i := range allBlocks {
		b := allBlocks[i].(*mqlTerraformBlock)
		if b.Type.Data != "locals" || blockModuleKey(b, entries) != moduleKey {
			continue
		}
		hb := blockHcl(b)
		if hb == nil {
			continue
		}
		attrs, _ := hb.Body.JustAttributes()
		for n := range attrs {
			locals = append(locals, localEntry{name: n, expr: attrs[n].Expr})
		}
	}

	localVals := map[string]cty.Value{}
	for pass := 0; pass < 8; pass++ {
		ctx := &hcl.EvalContext{
			Functions: funcs,
			Variables: map[string]cty.Value{"var": cty.ObjectVal(varVals), "local": cty.ObjectVal(localVals)},
		}
		changed := false
		for _, le := range locals {
			if _, done := localVals[le.name]; done {
				continue
			}
			if val, diags := le.expr.Value(ctx); !diags.HasErrors() && val.IsWhollyKnown() {
				localVals[le.name] = val
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	return &hcl.EvalContext{
		Functions: funcs,
		Variables: map[string]cty.Value{"var": cty.ObjectVal(varVals), "local": cty.ObjectVal(localVals)},
	}
}

// moduleInputOverrides resolves a module call's input arguments in the parent
// context, producing the variable overrides for the module's own context.
func (t *mqlTerraform) moduleInputOverrides(moduleCall *mqlTerraformBlock, parent *hcl.EvalContext) map[string]cty.Value {
	out := map[string]cty.Value{}
	hb := blockHcl(moduleCall)
	if hb == nil {
		return out
	}
	attrs, _ := hb.Body.JustAttributes()
	for name := range attrs {
		if moduleMetaArguments[name] {
			continue
		}
		if val, diags := attrs[name].Expr.Value(parent); !diags.HasErrors() && val.IsWhollyKnown() {
			out[name] = val
		}
	}
	return out
}

// blockModuleKey returns the module-call id a block belongs to ("" = root).
func blockModuleKey(b *mqlTerraformBlock, entries []fileModuleEntry) string {
	if b == nil || b.Start.State != plugin.StateIsSet || b.Start.Data == nil {
		return ""
	}
	mc := matchModuleEntry(entries, b.Start.Data.Path.Data)
	if mc == nil {
		return ""
	}
	id, _ := mc.id()
	return id
}

// resolveBlock evaluates each top-level argument to its effective value using
// the block's module context, falling back to the config-style reference form
// when a value is not wholly known (e.g. it depends on a data source/resource).
func (t *mqlTerraform) resolveBlock(b *mqlTerraformBlock) (any, error) {
	hb := blockHcl(b)
	if hb == nil {
		return map[string]any{}, nil
	}

	// Fetch shared state before building the (separately locked) context.
	blocksRaw := t.GetBlocks()
	if blocksRaw.Error != nil {
		return nil, blocksRaw.Error
	}
	entries := t.fileModuleIndex()
	moduleCall := matchModuleEntry(entries, blockFilePath(b))
	ctx := t.evalContextFor(moduleCall, blocksRaw.Data, entries)

	fallback := &hcl.EvalContext{Functions: hclFunctions()}
	attrs, _ := hb.Body.JustAttributes()
	dict := map[string]any{}
	for k := range attrs {
		expr := attrs[k].Expr
		if val, diags := expr.Value(ctx); !diags.HasErrors() && val.IsWhollyKnown() {
			dict[k] = ctyToGo(val)
			continue
		}
		v := getCtyValue(expr, fallback)
		if _, isFunc := expr.(*hclsyntax.FunctionCallExpr); isFunc {
			if lst, ok := v.([]any); ok && len(lst) == 1 {
				v = lst[0]
			}
		}
		dict[k] = v
	}
	return dict, nil
}

func (b *mqlTerraformBlock) resolved() (any, error) {
	o, err := CreateResource(b.MqlRuntime, "terraform", map[string]*llx.RawData{})
	if err != nil {
		return nil, err
	}
	tf := o.(*mqlTerraform)
	if err := tf.refreshCache(nil); err != nil {
		return nil, err
	}
	return tf.resolveBlock(b)
}

func (c *mqlTerraformResource) resolved() (any, error) {
	if c.tfBlock == nil {
		return map[string]any{}, nil
	}
	if c.tf == nil {
		return c.tfBlock.resolved()
	}
	return c.tf.resolveBlock(c.tfBlock)
}

func (c *mqlTerraformDatasource) resolved() (any, error) {
	if c.tfBlock == nil {
		return map[string]any{}, nil
	}
	if c.tf == nil {
		return c.tfBlock.resolved()
	}
	return c.tf.resolveBlock(c.tfBlock)
}

// ctyToGo converts a wholly-known cty value into the plain Go shapes MQL
// dicts use.
func ctyToGo(v cty.Value) any {
	if v.IsNull() || !v.IsKnown() {
		return nil
	}
	ty := v.Type()
	switch {
	case ty == cty.String:
		return v.AsString()
	case ty == cty.Bool:
		return v.True()
	case ty == cty.Number:
		f, _ := v.AsBigFloat().Float64()
		return f
	case ty.IsListType(), ty.IsSetType(), ty.IsTupleType():
		out := []any{}
		for it := v.ElementIterator(); it.Next(); {
			_, ev := it.Element()
			out = append(out, ctyToGo(ev))
		}
		return out
	case ty.IsMapType(), ty.IsObjectType():
		out := map[string]any{}
		for it := v.ElementIterator(); it.Next(); {
			k, ev := it.Element()
			out[k.AsString()] = ctyToGo(ev)
		}
		return out
	default:
		return nil
	}
}

// --- tree: full configuration as walkable nested data ---

func (b *mqlTerraformBlock) tree() (any, error) {
	return blockTree(blockHcl(b)), nil
}

func (c *mqlTerraformResource) tree() (any, error) {
	return blockTree(blockHcl(c.tfBlock)), nil
}

// blockTree renders a block as one document: its arguments (normalized like
// config) plus each child-block type keyed to the list of those blocks' trees.
func blockTree(hb *hcl.Block) map[string]any {
	if hb == nil {
		return map[string]any{}
	}
	res := map[string]any{}
	attrs, _ := hb.Body.JustAttributes()
	for k, v := range hclConfigAttributesToDict(attrs) {
		res[k] = v
	}
	if body, ok := hb.Body.(*hclsyntax.Body); ok {
		for _, nb := range body.Blocks {
			child := blockTree(nb.AsHCLBlock())
			lst, _ := res[nb.Type].([]any)
			res[nb.Type] = append(lst, child)
		}
	}
	return res
}
